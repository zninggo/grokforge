package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/zninggo/grokforge/internal/auth"
	"github.com/zninggo/grokforge/internal/repository"
)

// AuthHandler serves login/refresh/me/password.
type AuthHandler struct {
	Repo   *repository.AdminRepo
	Tokens *auth.TokenService
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type passwordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json body")
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		return Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "username and password required")
	}

	st, err := h.Repo.GetSetupState(c.Request().Context())
	if err != nil {
		return Fail(c, http.StatusServiceUnavailable, "POSTGRES_UNAVAILABLE", "database error")
	}
	if !st.Completed {
		return Fail(c, http.StatusConflict, "SETUP_REQUIRED", "complete setup first")
	}

	user, err := h.Repo.GetByUsername(c.Request().Context(), req.Username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid credentials")
		}
		return Fail(c, http.StatusServiceUnavailable, "POSTGRES_UNAVAILABLE", "database error")
	}
	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		return Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid credentials")
	}

	pair, err := h.Tokens.Issue(user.ID, user.Username)
	if err != nil {
		return Fail(c, http.StatusInternalServerError, "INTERNAL", "token issue failed")
	}
	return OK(c, map[string]any{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
		"expires_in":    pair.ExpiresIn,
		"token_type":    pair.TokenType,
		"user": map[string]any{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

func (h *AuthHandler) Refresh(c echo.Context) error {
	var req refreshRequest
	if err := c.Bind(&req); err != nil {
		return Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json body")
	}
	if strings.TrimSpace(req.RefreshToken) == "" {
		return Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "refresh_token required")
	}
	claims, err := h.Tokens.ParseRefresh(req.RefreshToken)
	if err != nil {
		return Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid refresh token")
	}
	user, err := h.Repo.GetByID(c.Request().Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not found")
		}
		return Fail(c, http.StatusServiceUnavailable, "POSTGRES_UNAVAILABLE", "database error")
	}
	pair, err := h.Tokens.Issue(user.ID, user.Username)
	if err != nil {
		return Fail(c, http.StatusInternalServerError, "INTERNAL", "token issue failed")
	}
	return OK(c, map[string]any{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
		"expires_in":    pair.ExpiresIn,
		"token_type":    pair.TokenType,
	})
}

func (h *AuthHandler) Logout(c echo.Context) error {
	// Phase 2: client discards tokens; server-side revoke lands later if needed.
	return OK(c, map[string]any{"logged_out": true})
}

func (h *AuthHandler) Me(c echo.Context) error {
	claims, ok := c.Get("claims").(*auth.Claims)
	if !ok || claims == nil {
		return Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
	}
	user, err := h.Repo.GetByID(c.Request().Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not found")
		}
		return Fail(c, http.StatusServiceUnavailable, "POSTGRES_UNAVAILABLE", "database error")
	}
	return OK(c, map[string]any{
		"id":                  user.ID,
		"username":            user.Username,
		"must_reset_password": user.MustResetPassword,
	})
}

func (h *AuthHandler) ChangePassword(c echo.Context) error {
	claims, ok := c.Get("claims").(*auth.Claims)
	if !ok || claims == nil {
		return Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
	}
	var req passwordRequest
	if err := c.Bind(&req); err != nil {
		return Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json body")
	}
	if err := auth.ValidatePassword(req.NewPassword); err != nil {
		return Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}
	user, err := h.Repo.GetByID(c.Request().Context(), claims.UserID)
	if err != nil {
		return Fail(c, http.StatusServiceUnavailable, "POSTGRES_UNAVAILABLE", "database error")
	}
	if !auth.CheckPassword(user.PasswordHash, req.CurrentPassword) {
		return Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "current password incorrect")
	}
	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}
	if err := h.Repo.UpdatePassword(c.Request().Context(), user.ID, hash); err != nil {
		return Fail(c, http.StatusServiceUnavailable, "POSTGRES_UNAVAILABLE", "database error")
	}
	return OK(c, map[string]any{"updated": true})
}
