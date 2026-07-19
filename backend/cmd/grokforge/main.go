package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zninggo/grokforge/internal/buildinfo"
	"github.com/zninggo/grokforge/internal/config"
	"github.com/zninggo/grokforge/internal/db"
	"github.com/zninggo/grokforge/internal/logger"
	"github.com/zninggo/grokforge/internal/migrate"
	"github.com/zninggo/grokforge/internal/server"
	"go.uber.org/zap"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "--version", "-v":
			fmt.Printf("grokforge %s\n", buildinfo.Version)
			return
		case "migrate":
			if err := runMigrate(); err != nil {
				fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log, err := logger.New(cfg.LogLevel)
	if err != nil {
		return err
	}
	defer func() { _ = log.Sync() }()

	log.Info("starting", zap.Any("config", cfg.Redacted()), zap.String("version", buildinfo.Version))

	ctx := context.Background()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := migrate.Up(ctx, pool); err != nil {
		return fmt.Errorf("auto-migrate: %w", err)
	}
	log.Info("migrations applied")

	rdb, err := connectRedis(ctx, cfg, log)
	if err != nil {
		return err
	}
	if rdb != nil {
		defer func() { _ = rdb.Close() }()
	}

	srv, err := server.New(cfg, log, pool, rdb)
	if err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	log.Info("ready", zap.String("hint", server.FormatListenHint(cfg.HTTPAddr)))

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Info("signal received", zap.String("signal", sig.String()))
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// connectRedis returns nil client when Redis URL is empty (dev profile).
// When URL is set, connection failure is fatal.
func connectRedis(ctx context.Context, cfg *config.Config, log *zap.Logger) (*redis.Client, error) {
	if cfg.RedisURL == "" {
		log.Warn("redis not configured; queue/workers disabled (dev profile)")
		return nil, nil
	}
	rdb, err := db.NewRedis(ctx, cfg.RedisURL)
	if err != nil {
		return nil, err
	}
	return rdb, nil
}

func runMigrate() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log, err := logger.New(cfg.LogLevel)
	if err != nil {
		return err
	}
	defer func() { _ = log.Sync() }()

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := migrate.Up(ctx, pool); err != nil {
		return err
	}
	log.Info("migrate complete")
	return nil
}
