package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zninggo/grokforge/internal/domain"
	"github.com/zninggo/grokforge/internal/queue"
	"github.com/zninggo/grokforge/internal/repository"
	"github.com/zninggo/grokforge/internal/ws"
)

// JobService orchestrates job create/enqueue and event emission.
type JobService struct {
	Jobs  *repository.JobRepo
	Queue *queue.Queue
	Hub   *ws.Hub
}

type CreateJobsRequest struct {
	Kind    string         `json:"kind"`
	Count   int            `json:"count"`
	Driver  string         `json:"driver"`
	Options map[string]any `json:"options"`
}

func (s *JobService) Create(ctx context.Context, req CreateJobsRequest) ([]domain.Job, error) {
	if req.Kind == "" {
		req.Kind = domain.JobKindNoop
	}
	switch req.Kind {
	case domain.JobKindNoop, domain.JobKindRegister, domain.JobKindProbe, domain.JobKindPipeline:
	default:
		return nil, fmt.Errorf("unsupported kind %q", req.Kind)
	}
	if req.Count <= 0 {
		req.Count = 1
	}
	if req.Count > 50 {
		return nil, fmt.Errorf("count must be <= 50")
	}
	if req.Driver == "" {
		req.Driver = "noop"
	}

	// Redis must be up before accepting new jobs (nil/missing broker → unavailable).
	if s.Queue == nil || !s.Queue.Available() {
		return nil, ErrRedisUnavailable
	}
	if err := s.Queue.Ping(ctx); err != nil {
		return nil, ErrRedisUnavailable
	}

	batchID := fmt.Sprintf("b%d", time.Now().UnixNano())
	var created []domain.Job
	for i := 0; i < req.Count; i++ {
		bid := batchID
		j, err := s.Jobs.Create(ctx, repository.CreateJobInput{
			BatchID: &bid,
			Kind:    req.Kind,
			Driver:  req.Driver,
			Options: req.Options,
		})
		if err != nil {
			return created, err
		}
		if err := s.Queue.Enqueue(ctx, req.Kind, j.ID); err != nil {
			_ = s.Jobs.MarkFailed(ctx, j.ID, "REDIS_UNAVAILABLE", err.Error())
			return created, ErrRedisUnavailable
		}
		_ = s.Jobs.MarkQueued(ctx, j.ID)
		j.Status = domain.JobQueued
		created = append(created, *j)
		s.emit("job.created", map[string]any{
			"job_id":   j.ID,
			"batch_id": batchID,
			"kind":     j.Kind,
		})
		_, _ = s.Jobs.AddEvent(ctx, j.ID, "info", "job queued", map[string]any{"batch_id": batchID})
	}
	return created, nil
}

func (s *JobService) Get(ctx context.Context, id int64) (*domain.Job, error) {
	return s.Jobs.Get(ctx, id)
}

func (s *JobService) List(ctx context.Context, kind, status string, page, pageSize int) ([]domain.Job, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	return s.Jobs.List(ctx, kind, status, pageSize, offset)
}

func (s *JobService) Stop(ctx context.Context, id int64) error {
	j, err := s.Jobs.Get(ctx, id)
	if err != nil {
		return err
	}
	switch j.Status {
	case domain.JobSucceeded, domain.JobFailed, domain.JobCancelled, domain.JobTimedOut:
		return nil
	}
	if err := s.Jobs.MarkCancelled(ctx, id); err != nil {
		return err
	}
	_, _ = s.Jobs.AddEvent(ctx, id, "warn", "job cancelled by user", nil)
	s.emit("job.finished", map[string]any{"job_id": id, "status": domain.JobCancelled})
	return nil
}

func (s *JobService) Events(ctx context.Context, jobID, sinceID int64) ([]domain.JobEvent, error) {
	return s.Jobs.ListEvents(ctx, jobID, sinceID, 100)
}

func (s *JobService) emit(typ string, payload any) {
	if s.Hub != nil {
		s.Hub.Broadcast(ws.Event{Type: typ, Payload: payload})
	}
	// best-effort redis pubsub for multi-process later
	if s.Queue == nil || !s.Queue.Available() {
		return
	}
	b, _ := json.Marshal(ws.Event{Type: typ, TS: time.Now().UTC().Format(time.RFC3339), Payload: payload})
	_ = s.Queue.Publish(context.Background(), string(b))
}

// Sentinel.
var ErrRedisUnavailable = fmt.Errorf("redis unavailable")
