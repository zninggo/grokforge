package api

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/zninggo/grokforge/internal/buildinfo"
)

// HealthHandler serves liveness and readiness probes.
type HealthHandler struct {
	Pool              *pgxpool.Pool
	Redis             *redis.Client
	ReadyRequireRedis bool
}

func (h *HealthHandler) Live(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"status":  "ok",
		"version": buildinfo.Version,
	})
}

func (h *HealthHandler) Ready(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 3*time.Second)
	defer cancel()

	checks := map[string]string{}
	ok := true

	if h.Pool == nil {
		checks["postgres"] = "not_configured"
		ok = false
	} else if err := h.Pool.Ping(ctx); err != nil {
		checks["postgres"] = "down"
		ok = false
	} else {
		checks["postgres"] = "up"
	}

	if h.Redis == nil {
		checks["redis"] = "not_configured"
		if h.ReadyRequireRedis {
			ok = false
		}
	} else if err := h.Redis.Ping(ctx).Err(); err != nil {
		checks["redis"] = "down"
		if h.ReadyRequireRedis {
			ok = false
		}
	} else {
		checks["redis"] = "up"
	}

	status := http.StatusOK
	state := "ready"
	if !ok {
		status = http.StatusServiceUnavailable
		state = "not_ready"
	}
	return c.JSON(status, map[string]any{
		"status":  state,
		"version": buildinfo.Version,
		"checks":  checks,
	})
}
