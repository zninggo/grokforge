package main

import (
	"context"
	"fmt"
	"os"

	"github.com/zninggo/grokforge/internal/buildinfo"
	"github.com/zninggo/grokforge/internal/config"
	"github.com/zninggo/grokforge/internal/db"
	"github.com/zninggo/grokforge/internal/logger"
	"github.com/zninggo/grokforge/internal/migrate"
	"go.uber.org/zap"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "--version", "-v":
			fmt.Printf("grokforge-migrate %s\n", buildinfo.Version)
			return
		}
	}

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
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

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := migrate.Up(ctx, pool); err != nil {
		return err
	}
	log.Info("migrate complete", zap.String("version", buildinfo.Version))
	return nil
}
