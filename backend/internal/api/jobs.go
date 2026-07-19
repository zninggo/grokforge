package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/zninggo/grokforge/internal/domain"
	"github.com/zninggo/grokforge/internal/repository"
	"github.com/zninggo/grokforge/internal/service"
)

// JobHandler exposes job management endpoints.
type JobHandler struct {
	Svc *service.JobService
}

type createJobsBody struct {
	Kind    string         `json:"kind"`
	Count   int            `json:"count"`
	Driver  string         `json:"driver"`
	Options map[string]any `json:"options"`
}

func (h *JobHandler) Create(c echo.Context) error {
	var body createJobsBody
	if err := c.Bind(&body); err != nil {
		return Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json body")
	}
	jobs, err := h.Svc.Create(c.Request().Context(), service.CreateJobsRequest{
		Kind:    body.Kind,
		Count:   body.Count,
		Driver:  body.Driver,
		Options: body.Options,
	})
	if err != nil {
		if errors.Is(err, service.ErrRedisUnavailable) {
			return Fail(c, http.StatusServiceUnavailable, "REDIS_UNAVAILABLE", "redis unavailable; refuse new jobs")
		}
		return Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}
	items := make([]map[string]any, 0, len(jobs))
	for _, j := range jobs {
		items = append(items, jobDTO(j))
	}
	return OK(c, map[string]any{"items": items, "count": len(items)})
}

func (h *JobHandler) List(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	items, total, err := h.Svc.List(c.Request().Context(), c.QueryParam("kind"), c.QueryParam("status"), page, pageSize)
	if err != nil {
		return Fail(c, http.StatusServiceUnavailable, "POSTGRES_UNAVAILABLE", "database error")
	}
	out := make([]map[string]any, 0, len(items))
	for _, j := range items {
		out = append(out, jobDTO(j))
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	return OK(c, map[string]any{
		"items":     out,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	})
}

func (h *JobHandler) Get(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid id")
	}
	j, err := h.Svc.Get(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return Fail(c, http.StatusNotFound, "NOT_FOUND", "job not found")
		}
		return Fail(c, http.StatusServiceUnavailable, "POSTGRES_UNAVAILABLE", "database error")
	}
	return OK(c, jobDTO(*j))
}

func (h *JobHandler) Stop(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid id")
	}
	if err := h.Svc.Stop(c.Request().Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return Fail(c, http.StatusNotFound, "NOT_FOUND", "job not found")
		}
		return Fail(c, http.StatusServiceUnavailable, "POSTGRES_UNAVAILABLE", "database error")
	}
	j, _ := h.Svc.Get(c.Request().Context(), id)
	if j == nil {
		return OK(c, map[string]any{"stopped": true})
	}
	return OK(c, jobDTO(*j))
}

func (h *JobHandler) Events(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid id")
	}
	since, _ := strconv.ParseInt(c.QueryParam("since_id"), 10, 64)
	evs, err := h.Svc.Events(c.Request().Context(), id, since)
	if err != nil {
		return Fail(c, http.StatusServiceUnavailable, "POSTGRES_UNAVAILABLE", "database error")
	}
	out := make([]map[string]any, 0, len(evs))
	for _, e := range evs {
		var meta any
		_ = json.Unmarshal(e.Meta, &meta)
		out = append(out, map[string]any{
			"id":         e.ID,
			"job_id":     e.JobID,
			"level":      e.Level,
			"message":    e.Message,
			"meta":       meta,
			"created_at": e.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return OK(c, map[string]any{"items": out})
}

func jobDTO(j domain.Job) map[string]any {
	var options any
	var result any
	_ = json.Unmarshal(j.Options, &options)
	_ = json.Unmarshal(j.Result, &result)
	var batch any
	if j.BatchID != nil {
		batch = *j.BatchID
	}
	dto := map[string]any{
		"id":            j.ID,
		"batch_id":      batch,
		"kind":          j.Kind,
		"status":        j.Status,
		"driver":        j.Driver,
		"proxy_ref":     j.ProxyRef,
		"email":         j.Email,
		"progress":      j.Progress,
		"step":          j.Step,
		"error_code":    j.ErrorCode,
		"error_message": j.ErrorMessage,
		"options":       options,
		"result":        result,
		"created_at":    j.CreatedAt.UTC().Format(time.RFC3339),
		"updated_at":    j.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if j.StartedAt != nil {
		dto["started_at"] = j.StartedAt.UTC().Format(time.RFC3339)
	}
	if j.FinishedAt != nil {
		dto["finished_at"] = j.FinishedAt.UTC().Format(time.RFC3339)
	}
	return dto
}
