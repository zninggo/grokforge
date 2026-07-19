package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// OK writes a success envelope.
func OK(c echo.Context, data any) error {
	return c.JSON(http.StatusOK, map[string]any{
		"success":    true,
		"data":       data,
		"request_id": c.Response().Header().Get(echo.HeaderXRequestID),
	})
}

// Fail writes a structured error envelope.
func Fail(c echo.Context, status int, code, message string) error {
	return c.JSON(status, map[string]any{
		"success": false,
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
		"request_id": c.Response().Header().Get(echo.HeaderXRequestID),
	})
}
