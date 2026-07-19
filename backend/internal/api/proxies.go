package api

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/zninggo/grokforge/internal/plugin/proxy"
	"github.com/zninggo/grokforge/internal/repository"
)

// ProxyHandler manages the static proxy pool.
type ProxyHandler struct {
	Repo *repository.ProxyRepo
}

type proxyImportBody struct {
	Text string `json:"text"`
}

func (h *ProxyHandler) List(c echo.Context) error {
	rows, err := h.Repo.List(c.Request().Context())
	if err != nil {
		return Fail(c, http.StatusServiceUnavailable, "POSTGRES_UNAVAILABLE", "database error")
	}
	items := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		items = append(items, map[string]any{
			"id":         r.ID,
			"scheme":     r.Scheme,
			"host":       r.Host,
			"port":       r.Port,
			"username":   r.Username,
			"display":    r.Display,
			"enabled":    r.Enabled,
			"fail_count": r.FailCount,
			"last_error": r.LastError,
			"created_at": r.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return OK(c, map[string]any{"items": items, "total": len(items)})
}

func (h *ProxyHandler) Replace(c echo.Context) error {
	var body proxyImportBody
	if err := c.Bind(&body); err != nil {
		return Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json body")
	}
	entries, errs := proxy.ParseLines(body.Text)
	if len(errs) > 0 && len(entries) == 0 {
		return Fail(c, http.StatusBadRequest, "PROXY_BAD", "no valid proxies: "+errs[0])
	}
	n, err := h.Repo.ReplaceAll(c.Request().Context(), entries)
	if err != nil {
		return Fail(c, http.StatusServiceUnavailable, "POSTGRES_UNAVAILABLE", "database error")
	}
	return OK(c, map[string]any{
		"imported": n,
		"errors":   errs,
	})
}

func (h *ProxyHandler) Validate(c echo.Context) error {
	var body proxyImportBody
	if err := c.Bind(&body); err != nil {
		return Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json body")
	}
	entries, errs := proxy.ParseLines(body.Text)
	out := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		out = append(out, map[string]any{
			"display":  e.Display(),
			"scheme":   e.Scheme,
			"host":     e.Host,
			"port":     e.Port,
			"username": e.Username,
		})
	}
	return OK(c, map[string]any{
		"valid":  out,
		"errors": errs,
		"count":  len(out),
	})
}
