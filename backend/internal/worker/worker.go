package worker

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zninggo/grokforge/internal/domain"
	"github.com/zninggo/grokforge/internal/queue"
	"github.com/zninggo/grokforge/internal/repository"
	"github.com/zninggo/grokforge/internal/service"
	"github.com/zninggo/grokforge/internal/ws"
	"go.uber.org/zap"
)

// Worker consumes Redis queues and executes jobs.
type Worker struct {
	Jobs     *repository.JobRepo
	Queue    *queue.Queue
	Hub      *ws.Hub
	Log      *zap.Logger
	Register *service.RegisterService
}

func (w *Worker) Run(ctx context.Context) {
	w.Log.Info("worker started")
	for {
		select {
		case <-ctx.Done():
			w.Log.Info("worker stopped")
			return
		default:
		}
		qname, jobID, err := w.Queue.BRPop(ctx, 2*time.Second)
		if err != nil {
			if errors.Is(err, redis.Nil) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				continue
			}
			w.Log.Warn("brpop", zap.Error(err))
			time.Sleep(time.Second)
			continue
		}
		_ = qname
		if err := w.handle(ctx, jobID); err != nil {
			w.Log.Error("job failed", zap.Int64("job_id", jobID), zap.Error(err))
		}
	}
}

func (w *Worker) handle(ctx context.Context, jobID int64) error {
	ok, err := w.Queue.TryLock(ctx, jobID, 15*time.Minute)
	if err != nil {
		return err
	}
	if !ok {
		w.Log.Debug("job locked elsewhere", zap.Int64("job_id", jobID))
		return nil
	}
	defer func() { _ = w.Queue.Unlock(context.Background(), jobID) }()

	job, err := w.Jobs.Get(ctx, jobID)
	if err != nil {
		return err
	}
	if job.Status == domain.JobCancelled {
		return nil
	}

	_ = w.Jobs.MarkRunning(ctx, jobID)
	w.emit("job.progress", map[string]any{"job_id": jobID, "status": domain.JobRunning, "percent": 0, "step": "running"})
	_, _ = w.Jobs.AddEvent(ctx, jobID, "info", "worker picked job", map[string]any{"kind": job.Kind})

	switch job.Kind {
	case domain.JobKindNoop:
		return w.runNoop(ctx, job)
	case domain.JobKindRegister:
		return w.runRegister(ctx, job)
	default:
		msg := "handler not implemented yet for kind=" + job.Kind
		_ = w.Jobs.MarkFailed(ctx, jobID, "NOT_IMPLEMENTED", msg)
		_, _ = w.Jobs.AddEvent(ctx, jobID, "error", msg, nil)
		w.emit("job.finished", map[string]any{"job_id": jobID, "status": domain.JobFailed, "error_code": "NOT_IMPLEMENTED"})
		return nil
	}
}

func (w *Worker) runRegister(ctx context.Context, job *domain.Job) error {
	if w.Register == nil {
		msg := "register service not configured"
		_ = w.Jobs.MarkFailed(ctx, job.ID, "REGISTER_FAILED", msg)
		w.emit("job.finished", map[string]any{"job_id": job.ID, "status": domain.JobFailed, "error_code": "REGISTER_FAILED"})
		return nil
	}
	w.Register.Progress = func(ctx context.Context, jobID int64, percent int, step, message string) {
		w.emit("job.progress", map[string]any{
			"job_id": jobID, "status": domain.JobRunning, "percent": percent, "step": step,
		})
		w.emit("job.log", map[string]any{"job_id": jobID, "level": "info", "message": message})
	}
	err := w.Register.Run(ctx, job.ID)
	if err != nil {
		code, msg := mapJobError(err)
		_ = w.Jobs.MarkFailed(ctx, job.ID, code, msg)
		_, _ = w.Jobs.AddEvent(ctx, job.ID, "error", msg, map[string]any{"code": code})
		w.emit("job.finished", map[string]any{"job_id": job.ID, "status": domain.JobFailed, "error_code": code})
		return nil
	}
	// RegisterService already MarkSucceeded
	w.emit("job.finished", map[string]any{"job_id": job.ID, "status": domain.JobSucceeded})
	return nil
}

func mapJobError(err error) (code, message string) {
	message = err.Error()
	switch {
	case strings.Contains(message, "OTP_TIMEOUT"):
		return "OTP_TIMEOUT", message
	case strings.Contains(message, "YYDS"):
		return "YYDS_ERROR", message
	case strings.Contains(message, "PROXY"):
		return "PROXY_BAD", message
	case strings.Contains(message, "BROWSER"):
		return "BROWSER_CRASH", message
	case strings.Contains(message, "REGISTER_FAILED"):
		return "REGISTER_FAILED", message
	default:
		return "REGISTER_FAILED", message
	}
}

func (w *Worker) runNoop(ctx context.Context, job *domain.Job) error {
	steps := []struct {
		p    int
		step string
		msg  string
	}{
		{20, "prepare", "noop prepare"},
		{50, "work", "noop work"},
		{80, "finalize", "noop finalize"},
		{100, "done", "noop done"},
	}
	for _, s := range steps {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
		cur, err := w.Jobs.Get(ctx, job.ID)
		if err == nil && cur.Status == domain.JobCancelled {
			_, _ = w.Jobs.AddEvent(ctx, job.ID, "warn", "cancelled during noop", nil)
			w.emit("job.finished", map[string]any{"job_id": job.ID, "status": domain.JobCancelled})
			return nil
		}
		_ = w.Jobs.UpdateProgress(ctx, job.ID, s.p, s.step)
		_, _ = w.Jobs.AddEvent(ctx, job.ID, "info", s.msg, map[string]any{"progress": s.p})
		w.emit("job.progress", map[string]any{"job_id": job.ID, "status": domain.JobRunning, "percent": s.p, "step": s.step})
		w.emit("job.log", map[string]any{"job_id": job.ID, "level": "info", "message": s.msg})
	}
	result := map[string]any{"ok": true, "kind": "noop"}
	if err := w.Jobs.MarkSucceeded(ctx, job.ID, result); err != nil {
		return err
	}
	_, _ = w.Jobs.AddEvent(ctx, job.ID, "info", "noop succeeded", result)
	w.emit("job.finished", map[string]any{"job_id": job.ID, "status": domain.JobSucceeded})
	return nil
}

func (w *Worker) emit(typ string, payload any) {
	if w.Hub != nil {
		w.Hub.Broadcast(ws.Event{Type: typ, Payload: payload})
	}
	b, _ := json.Marshal(ws.Event{Type: typ, TS: time.Now().UTC().Format(time.RFC3339), Payload: payload})
	_ = w.Queue.Publish(context.Background(), string(b))
}
