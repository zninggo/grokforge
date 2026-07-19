package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/zninggo/grokforge/internal/repository"
)

// AuditHandler lists audit logs.
type AuditHandler struct {
	Repo *repository.AuditRepo
}

func (h *AuditHandler) List(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	items, total, err := h.Repo.List(c.Request().Context(), pageSize, (page-1)*pageSize)
	if err != nil {
		return Fail(c, http.StatusServiceUnavailable, "POSTGRES_UNAVAILABLE", "database error")
	}
	out := make([]map[string]any, 0, len(items))
	for _, a := range items {
		var detail any
		_ = json.Unmarshal(a.Detail, &detail)
		out = append(out, map[string]any{
			"id":          a.ID,
			"actor":       a.Actor,
			"action":      a.Action,
			"target_type": a.TargetType,
			"target_id":   a.TargetID,
			"detail":      detail,
			"created_at":  a.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return OK(c, map[string]any{"items": out, "page": page, "page_size": pageSize, "total": total})
}
