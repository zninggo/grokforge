package api

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/zninggo/grokforge/internal/auth"
	"github.com/zninggo/grokforge/internal/repository"
)

// RequireAuth validates Bearer access JWT and stores claims in context.
func RequireAuth(tokens *auth.TokenService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Request().Header.Get("Authorization")
			if h == "" {
				return Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing authorization")
			}
			const prefix = "Bearer "
			if !strings.HasPrefix(h, prefix) {
				return Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid authorization scheme")
			}
			raw := strings.TrimSpace(strings.TrimPrefix(h, prefix))
			if raw == "" {
				return Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing token")
			}
			claims, err := tokens.ParseAccess(raw)
			if err != nil {
				return Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
			}
			c.Set("claims", claims)
			return next(c)
		}
	}
}

// RequireSetup blocks business routes until installation is complete.
func RequireSetup(repo *repository.AdminRepo) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			st, err := repo.GetSetupState(c.Request().Context())
			if err != nil {
				return Fail(c, http.StatusServiceUnavailable, "POSTGRES_UNAVAILABLE", "database error")
			}
			if !st.Completed {
				return Fail(c, http.StatusConflict, "SETUP_REQUIRED", "complete setup first")
			}
			return next(c)
		}
	}
}
