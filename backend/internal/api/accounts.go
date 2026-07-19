package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/zninggo/grokforge/internal/auth"
	"github.com/zninggo/grokforge/internal/crypto"
	"github.com/zninggo/grokforge/internal/repository"
	"github.com/zninggo/grokforge/internal/service"
)

// AccountHandler manages account assets.
type AccountHandler struct {
	Accounts *repository.AccountRepo
	Audit    *repository.AuditRepo
	Box      *crypto.Box
	Probe    *service.ProbeService
	Jobs     *service.JobService
}

func (h *AccountHandler) List(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	items, total, err := h.Accounts.List(c.Request().Context(), c.QueryParam("health"), pageSize, (page-1)*pageSize)
	if err != nil {
		return Fail(c, http.StatusServiceUnavailable, "POSTGRES_UNAVAILABLE", "database error")
	}
	out := make([]map[string]any, 0, len(items))
	for _, a := range items {
		out = append(out, accountDTO(a, false, nil))
	}
	return OK(c, map[string]any{"items": out, "page": page, "page_size": pageSize, "total": total})
}

func (h *AccountHandler) Get(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid id")
	}
	a, err := h.Accounts.Get(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return Fail(c, http.StatusNotFound, "NOT_FOUND", "account not found")
		}
		return Fail(c, http.StatusServiceUnavailable, "POSTGRES_UNAVAILABLE", "database error")
	}
	return OK(c, accountDTO(*a, false, nil))
}

func (h *AccountHandler) Delete(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid id")
	}
	if err := h.Accounts.Delete(c.Request().Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return Fail(c, http.StatusNotFound, "NOT_FOUND", "account not found")
		}
		return Fail(c, http.StatusServiceUnavailable, "POSTGRES_UNAVAILABLE", "database error")
	}
	_ = h.Audit.Add(c.Request().Context(), actorName(c), "account.delete", "account", strconv.FormatInt(id, 10), nil)
	return OK(c, map[string]any{"deleted": true, "id": id})
}

type deleteBatchBody struct {
	IDs []int64 `json:"ids"`
}

func (h *AccountHandler) DeleteBatch(c echo.Context) error {
	var body deleteBatchBody
	if err := c.Bind(&body); err != nil || len(body.IDs) == 0 {
		return Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "ids required")
	}
	n, err := h.Accounts.DeleteBatch(c.Request().Context(), body.IDs)
	if err != nil {
		return Fail(c, http.StatusServiceUnavailable, "POSTGRES_UNAVAILABLE", "database error")
	}
	_ = h.Audit.Add(c.Request().Context(), actorName(c), "account.delete_batch", "account", "", map[string]any{
		"ids": body.IDs, "deleted": n,
	})
	return OK(c, map[string]any{"deleted": n})
}

type exportBody struct {
	IDs           []int64 `json:"ids"`
	IncludeSecret bool    `json:"include_secret"`
}

func (h *AccountHandler) Export(c echo.Context) error {
	var body exportBody
	_ = c.Bind(&body)
	if h.Box == nil && body.IncludeSecret {
		return Fail(c, http.StatusForbidden, "FORBIDDEN", "master key not configured; cannot decrypt credentials")
	}

	// If no ids, export all (capped).
	var accounts []repository.Account
	if len(body.IDs) == 0 {
		items, _, err := h.Accounts.List(c.Request().Context(), "", 100, 0)
		if err != nil {
			return Fail(c, http.StatusServiceUnavailable, "POSTGRES_UNAVAILABLE", "database error")
		}
		accounts = items
	} else {
		for _, id := range body.IDs {
			a, err := h.Accounts.Get(c.Request().Context(), id)
			if err != nil {
				continue
			}
			accounts = append(accounts, *a)
		}
	}

	items := make([]map[string]any, 0, len(accounts))
	for _, a := range accounts {
		var secrets map[string]any
		if body.IncludeSecret {
			if h.Box == nil {
				return Fail(c, http.StatusForbidden, "FORBIDDEN", "master key not configured")
			}
			cipher, _, err := h.Accounts.GetCredentialCipher(c.Request().Context(), a.ID)
			if err == nil {
				pt, err := h.Box.Open(cipher)
				if err != nil {
					return Fail(c, http.StatusInternalServerError, "INTERNAL", "decrypt failed")
				}
				_ = json.Unmarshal([]byte(pt), &secrets)
			}
		}
		items = append(items, accountDTO(a, body.IncludeSecret, secrets))
	}

	_ = h.Audit.Add(c.Request().Context(), actorName(c), "account.export", "account", "", map[string]any{
		"count":          len(items),
		"include_secret": body.IncludeSecret,
		// never log secrets
	})

	return OK(c, map[string]any{
		"items":          items,
		"count":          len(items),
		"include_secret": body.IncludeSecret,
		"exported_at":    time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *AccountHandler) ProbeOne(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid id")
	}
	if h.Probe == nil {
		return Fail(c, http.StatusServiceUnavailable, "INTERNAL", "probe service unavailable")
	}
	res, err := h.Probe.ProbeAccount(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return Fail(c, http.StatusNotFound, "NOT_FOUND", "account not found")
		}
		if errors.Is(err, service.ErrNoMasterKey) {
			return Fail(c, http.StatusForbidden, "FORBIDDEN", "master key required to probe credentials")
		}
		return Fail(c, http.StatusUnprocessableEntity, "PROBE_FAILED", err.Error())
	}
	return OK(c, res)
}

type probeBatchBody struct {
	IDs []int64 `json:"ids"`
}

func (h *AccountHandler) ProbeBatch(c echo.Context) error {
	var body probeBatchBody
	if err := c.Bind(&body); err != nil || len(body.IDs) == 0 {
		return Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "ids required")
	}
	if h.Jobs == nil {
		return Fail(c, http.StatusServiceUnavailable, "INTERNAL", "job service unavailable")
	}
	if len(body.IDs) > 50 {
		return Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "ids must be <= 50")
	}
	var items []map[string]any
	for _, id := range body.IDs {
		js, err := h.Jobs.Create(c.Request().Context(), service.CreateJobsRequest{
			Kind:    "probe",
			Count:   1,
			Driver:  "session",
			Options: map[string]any{"account_id": id},
		})
		if err != nil {
			if errors.Is(err, service.ErrRedisUnavailable) {
				return Fail(c, http.StatusServiceUnavailable, "REDIS_UNAVAILABLE", "redis unavailable")
			}
			return Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		}
		if len(js) > 0 {
			items = append(items, map[string]any{"job_id": js[0].ID, "account_id": id})
		}
	}
	return OK(c, map[string]any{"items": items, "count": len(items)})
}

func (h *AccountHandler) ProbeAll(c echo.Context) error {
	ids, err := h.Accounts.ListIDs(c.Request().Context(), false)
	if err != nil {
		return Fail(c, http.StatusServiceUnavailable, "POSTGRES_UNAVAILABLE", "database error")
	}
	if len(ids) == 0 {
		return OK(c, map[string]any{"items": []any{}, "count": 0})
	}
	// cap
	if len(ids) > 50 {
		ids = ids[:50]
	}
	var items []map[string]any
	for _, id := range ids {
		js, err := h.Jobs.Create(c.Request().Context(), service.CreateJobsRequest{
			Kind:    "probe",
			Count:   1,
			Driver:  "session",
			Options: map[string]any{"account_id": id},
		})
		if err != nil {
			if errors.Is(err, service.ErrRedisUnavailable) {
				return Fail(c, http.StatusServiceUnavailable, "REDIS_UNAVAILABLE", "redis unavailable")
			}
			return Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		}
		if len(js) > 0 {
			items = append(items, map[string]any{"job_id": js[0].ID, "account_id": id})
		}
	}
	return OK(c, map[string]any{"items": items, "count": len(items)})
}

func accountDTO(a repository.Account, withSecret bool, secrets map[string]any) map[string]any {
	var meta any
	var plan any
	_ = json.Unmarshal(a.Meta, &meta)
	_ = json.Unmarshal(a.PlanSnapshot, &plan)
	dto := map[string]any{
		"id":             a.ID,
		"email":          a.Email,
		"upstream":       a.Upstream,
		"status":         a.Status,
		"health_status":  a.HealthStatus,
		"health_detail":  a.HealthDetail,
		"plan_snapshot":  plan,
		"meta":           meta,
		"has_credential": a.HasCredential,
		"created_at":     a.CreatedAt.UTC().Format(time.RFC3339),
		"updated_at":     a.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if a.LastProbedAt != nil {
		dto["last_probed_at"] = a.LastProbedAt.UTC().Format(time.RFC3339)
	}
	if withSecret && secrets != nil {
		// still avoid dumping huge raw cookies in list UIs; export includes them intentionally
		dto["credentials"] = secrets
	}
	return dto
}

func actorName(c echo.Context) string {
	if claims, ok := c.Get("claims").(*auth.Claims); ok && claims != nil {
		return claims.Username
	}
	return "admin"
}
