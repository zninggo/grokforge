package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"
	"github.com/zninggo/grokforge/internal/api"
	"github.com/zninggo/grokforge/internal/buildinfo"
	"github.com/zninggo/grokforge/internal/config"
	"go.uber.org/zap"
)

// Server wraps the HTTP stack.
type Server struct {
	echo   *echo.Echo
	cfg    *config.Config
	log    *zap.Logger
	pool   *pgxpool.Pool
	redis  *redis.Client
}

func New(cfg *config.Config, log *zap.Logger, pool *pgxpool.Pool, rdb *redis.Client) *Server {
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

	h := &api.HealthHandler{
		Pool:              pool,
		Redis:             rdb,
		ReadyRequireRedis: cfg.ReadyRequireRedis,
	}
	e.GET("/healthz", h.Live)
	e.GET("/live", h.Live)
	e.GET("/readyz", h.Ready)
	e.GET("/ready", h.Ready)
	e.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]any{
			"name":    "grokforge",
			"version": buildinfo.Version,
			"port":    cfg.HTTPAddr,
			"message": "API up; admin UI embeds in Phase 3",
		})
	})

	return &Server{echo: e, cfg: cfg, log: log, pool: pool, redis: rdb}
}

func (s *Server) Start() error {
	s.log.Info("listening", zap.String("addr", s.cfg.HTTPAddr), zap.String("version", buildinfo.Version))
	return s.echo.Start(s.cfg.HTTPAddr)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.echo.Shutdown(ctx)
}

// Addr helps tests.
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
