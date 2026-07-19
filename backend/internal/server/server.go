package server

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"
	"github.com/zninggo/grokforge/internal/api"
	"github.com/zninggo/grokforge/internal/auth"
	"github.com/zninggo/grokforge/internal/buildinfo"
	"github.com/zninggo/grokforge/internal/config"
	"github.com/zninggo/grokforge/internal/queue"
	"github.com/zninggo/grokforge/internal/repository"
	"github.com/zninggo/grokforge/internal/service"
	"github.com/zninggo/grokforge/internal/web"
	"github.com/zninggo/grokforge/internal/worker"
	"github.com/zninggo/grokforge/internal/ws"
	"go.uber.org/zap"
)

// Server wraps the HTTP stack and background workers.
type Server struct {
	echo   *echo.Echo
	cfg    *config.Config
	log    *zap.Logger
	pool   *pgxpool.Pool
	redis  *redis.Client
	hub    *ws.Hub
	cancel context.CancelFunc
}

func New(cfg *config.Config, log *zap.Logger, pool *pgxpool.Pool, rdb *redis.Client) (*Server, error) {
	tokens, err := auth.NewTokenService(cfg.JWTSecret)
	if err != nil {
		return nil, err
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:       true,
		LogStatus:    true,
		LogMethod:    true,
		LogRequestID: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			log.Info("http",
				zap.String("id", v.RequestID),
				zap.String("method", v.Method),
				zap.String("uri", v.URI),
				zap.Int("status", v.Status),
				zap.Duration("latency", v.Latency),
			)
			return nil
		},
	}))

	adminRepo := repository.NewAdminRepo(pool)
	jobRepo := repository.NewJobRepo(pool)
	q := queue.New(rdb)
	hub := ws.NewHub(log)
	jobSvc := &service.JobService{Jobs: jobRepo, Queue: q, Hub: hub}

	h := &api.HealthHandler{
		Pool:              pool,
		Redis:             rdb,
		ReadyRequireRedis: cfg.ReadyRequireRedis,
	}
	e.GET("/healthz", h.Live)
	e.GET("/live", h.Live)
	e.GET("/readyz", h.Ready)
	e.GET("/ready", h.Ready)

	setupH := &api.SetupHandler{Repo: adminRepo, Pool: pool, Redis: rdb}
	authH := &api.AuthHandler{Repo: adminRepo, Tokens: tokens}
	sysH := &api.SystemHandler{Repo: adminRepo, Pool: pool, Redis: rdb}
	jobH := &api.JobHandler{Svc: jobSvc}
	wsH := &api.WSHandler{Hub: hub, Tokens: tokens}

	v1 := e.Group("/api/v1")
	v1.GET("/setup/status", setupH.Status)
	v1.POST("/setup/init", setupH.Init)
	v1.POST("/auth/login", authH.Login)
	v1.POST("/auth/refresh", authH.Refresh)

	// WS auth is token-based (query/header); not under setup middleware group.
	e.GET("/ws/v1", wsH.Connect)

	protected := v1.Group("", api.RequireSetup(adminRepo), api.RequireAuth(tokens))
	protected.POST("/auth/logout", authH.Logout)
	protected.GET("/auth/me", authH.Me)
	protected.POST("/auth/password", authH.ChangePassword)
	protected.GET("/system/info", sysH.Info)
	protected.GET("/system/health", sysH.Health)

	protected.POST("/jobs", jobH.Create)
	protected.GET("/jobs", jobH.List)
	protected.GET("/jobs/:id", jobH.Get)
	protected.POST("/jobs/:id/stop", jobH.Stop)
	protected.GET("/jobs/:id/events", jobH.Events)

	if err := web.Register(e); err != nil {
		return nil, err
	}

	wctx, cancel := context.WithCancel(context.Background())
	wk := &worker.Worker{Jobs: jobRepo, Queue: q, Hub: hub, Log: log.Named("worker")}
	go wk.Run(wctx)

	return &Server{echo: e, cfg: cfg, log: log, pool: pool, redis: rdb, hub: hub, cancel: cancel}, nil
}

func (s *Server) Start() error {
	s.log.Info("listening", zap.String("addr", s.cfg.HTTPAddr), zap.String("version", buildinfo.Version))
	return s.echo.Start(s.cfg.HTTPAddr)
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.cancel != nil {
		s.cancel()
	}
	return s.echo.Shutdown(ctx)
}

func (s *Server) Addr() string {
	return s.cfg.HTTPAddr
}

func GracefulShutdown(s *Server, log *zap.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		log.Error("shutdown", zap.Error(err))
	}
}

func FormatListenHint(addr string) string {
	return fmt.Sprintf("http://127.0.0.1%s", normalizePort(addr))
}

func normalizePort(addr string) string {
	if len(addr) > 0 && addr[0] == ':' {
		return addr
	}
	return ":" + addr
}
