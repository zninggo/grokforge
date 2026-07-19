package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/zninggo/grokforge/internal/auth"
	"github.com/zninggo/grokforge/internal/repository"
)

// SetupHandler serves installation wizard endpoints.
type SetupHandler struct {
	Repo  *repository.AdminRepo
	Pool  *pgxpool.Pool
	Redis *redis.Client
}

type setupInitRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *SetupHandler) Status(c echo.Context) error {
	ctx := c.Request().Context()
	st, err := h.Repo.GetSetupState(ctx)
	if err != nil {
		return Fail(c, http.StatusServiceUnavailable, "POSTGRES_UNAVAILABLE", "database error")
	}

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

	var completedAt any
	if st.CompletedAt != nil {
		completedAt = st.CompletedAt.UTC().Format(time.RFC3339)
	}

	return OK(c, map[string]any{
		"completed":    st.Completed,
		"completed_at": completedAt,
		"checks":       checks,
	})
}

func (h *SetupHandler) Init(c echo.Context) error {
	var req setupInitRequest
	if err := c.Bind(&req); err != nil {
		return Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json body")
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		req.Username = "admin"
	}
	if len(req.Username) < 3 || len(req.Username) > 64 {
		return Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "username must be 3-64 characters")
	}
	if err := auth.ValidatePassword(req.Password); err != nil {
		return Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	user, err := h.Repo.CompleteSetup(c.Request().Context(), req.Username, hash)
	if err != nil {
		if err == repository.ErrSetupAlreadyDone {
			return Fail(c, http.StatusConflict, "CONFLICT", "setup already completed")
		}
		return Fail(c, http.StatusServiceUnavailable, "POSTGRES_UNAVAILABLE", "database error")
	}

	return OK(c, map[string]any{
		"completed": true,
		"username":  user.Username,
		"user_id":   user.ID,
	})
}
