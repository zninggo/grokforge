package api

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/zninggo/grokforge/internal/buildinfo"
	"github.com/zninggo/grokforge/internal/repository"
)

// SystemHandler serves authenticated system info.
type SystemHandler struct {
	Repo  *repository.AdminRepo
	Pool  *pgxpool.Pool
	Redis *redis.Client
}

func (h *SystemHandler) Info(c echo.Context) error {
	st, err := h.Repo.GetSetupState(c.Request().Context())
	if err != nil {
		return Fail(c, http.StatusServiceUnavailable, "POSTGRES_UNAVAILABLE", "database error")
	}
	return OK(c, map[string]any{
		"name":            "grokforge",
		"version":         buildinfo.Version,
		"setup_completed": st.Completed,
		"time":            time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *SystemHandler) Health(c echo.Context) error {
	ctx := c.Request().Context()
	checks := map[string]string{}
	if h.Pool == nil {
		checks["postgres"] = "not_configured"
	} else if err := h.Pool.Ping(ctx); err != nil {
		checks["postgres"] = "down"
	} else {
		checks["postgres"] = "up"
	}
	if h.Redis == nil {
		checks["redis"] = "not_configured"
	} else if err := h.Redis.Ping(ctx).Err(); err != nil {
		checks["redis"] = "down"
	} else {
		checks["redis"] = "up"
	}
	return OK(c, map[string]any{
		"version": buildinfo.Version,
		"checks":  checks,
	})
}
