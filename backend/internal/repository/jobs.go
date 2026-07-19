package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zninggo/grokforge/internal/domain"
)

// JobRepo persists jobs and events.
type JobRepo struct {
	pool *pgxpool.Pool
}

func NewJobRepo(pool *pgxpool.Pool) *JobRepo {
	return &JobRepo{pool: pool}
}

type CreateJobInput struct {
	BatchID *string
	Kind    string
	Driver  string
	Options map[string]any
}

func (r *JobRepo) Create(ctx context.Context, in CreateJobInput) (*domain.Job, error) {
	opt, err := json.Marshal(in.Options)
	if err != nil {
		opt = []byte("{}")
	}
	if in.Options == nil {
		opt = []byte("{}")
	}
	var j domain.Job
	var options, result []byte
	err = r.pool.QueryRow(ctx, `
		INSERT INTO jobs (batch_id, kind, status, driver, options)
		VALUES ($1, $2, $3, $4, $5::jsonb)
		RETURNING id, batch_id, kind, status, driver, proxy_ref, email, progress, step,
		          error_code, error_message, options, result, created_at, updated_at, started_at, finished_at
	`, in.BatchID, in.Kind, domain.JobPending, in.Driver, string(opt)).Scan(
		&j.ID, &j.BatchID, &j.Kind, &j.Status, &j.Driver, &j.ProxyRef, &j.Email, &j.Progress, &j.Step,
		&j.ErrorCode, &j.ErrorMessage, &options, &result, &j.CreatedAt, &j.UpdatedAt, &j.StartedAt, &j.FinishedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert job: %w", err)
	}
	j.Options, j.Result = options, result
	return &j, nil
}

func (r *JobRepo) Get(ctx context.Context, id int64) (*domain.Job, error) {
	j, err := r.scanOne(ctx, `
		SELECT id, batch_id, kind, status, driver, proxy_ref, email, progress, step,
		       error_code, error_message, options, result, created_at, updated_at, started_at, finished_at
		FROM jobs WHERE id = $1
	`, id)
	if err != nil {
		return nil, err
	}
	return j, nil
}

func (r *JobRepo) List(ctx context.Context, kind, status string, limit, offset int) ([]domain.Job, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	var total int
	qCount := `SELECT COUNT(*) FROM jobs WHERE ($1 = '' OR kind = $1) AND ($2 = '' OR status = $2)`
	if err := r.pool.QueryRow(ctx, qCount, kind, status).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, batch_id, kind, status, driver, proxy_ref, email, progress, step,
		       error_code, error_message, options, result, created_at, updated_at, started_at, finished_at
		FROM jobs
		WHERE ($1 = '' OR kind = $1) AND ($2 = '' OR status = $2)
		ORDER BY id DESC
		LIMIT $3 OFFSET $4
	`, kind, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.Job
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *j)
	}
	return out, total, rows.Err()
}

func (r *JobRepo) MarkQueued(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE jobs SET status = $2, updated_at = now() WHERE id = $1
	`, id, domain.JobQueued)
	return err
}

func (r *JobRepo) MarkRunning(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE jobs SET status = $2, started_at = COALESCE(started_at, now()), updated_at = now(), step = 'running'
		WHERE id = $1
	`, id, domain.JobRunning)
	return err
}

func (r *JobRepo) UpdateProgress(ctx context.Context, id int64, progress int, step string) error {
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE jobs SET progress = $2, step = $3, updated_at = now() WHERE id = $1
	`, id, progress, step)
	return err
}

func (r *JobRepo) MarkSucceeded(ctx context.Context, id int64, result map[string]any) error {
	b, _ := json.Marshal(result)
	if result == nil {
		b = []byte("{}")
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE jobs
		SET status = $2, progress = 100, step = 'done', result = $3::jsonb,
		    finished_at = now(), updated_at = now(), error_code = '', error_message = ''
		WHERE id = $1
	`, id, domain.JobSucceeded, string(b))
	return err
}

func (r *JobRepo) MarkFailed(ctx context.Context, id int64, code, message string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE jobs
		SET status = $2, error_code = $3, error_message = $4, finished_at = now(), updated_at = now()
		WHERE id = $1
	`, id, domain.JobFailed, code, message)
	return err
}

func (r *JobRepo) MarkCancelled(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE jobs
		SET status = $2, finished_at = now(), updated_at = now(), step = 'cancelled'
		WHERE id = $1 AND status IN ('pending','queued','running','retrying')
	`, id, domain.JobCancelled)
	return err
}

func (r *JobRepo) AddEvent(ctx context.Context, jobID int64, level, message string, meta map[string]any) (*domain.JobEvent, error) {
	b, _ := json.Marshal(meta)
	if meta == nil {
		b = []byte("{}")
	}
	var ev domain.JobEvent
	var metaB []byte
	err := r.pool.QueryRow(ctx, `
		INSERT INTO job_events (job_id, level, message, meta)
		VALUES ($1, $2, $3, $4::jsonb)
		RETURNING id, job_id, level, message, meta, created_at
	`, jobID, level, message, string(b)).Scan(&ev.ID, &ev.JobID, &ev.Level, &ev.Message, &metaB, &ev.CreatedAt)
	if err != nil {
		return nil, err
	}
	ev.Meta = metaB
	return &ev, nil
}

func (r *JobRepo) ListEvents(ctx context.Context, jobID int64, sinceID int64, limit int) ([]domain.JobEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, job_id, level, message, meta, created_at
		FROM job_events
		WHERE job_id = $1 AND id > $2
		ORDER BY id ASC
		LIMIT $3
	`, jobID, sinceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.JobEvent
	for rows.Next() {
		var ev domain.JobEvent
		var meta []byte
		if err := rows.Scan(&ev.ID, &ev.JobID, &ev.Level, &ev.Message, &meta, &ev.CreatedAt); err != nil {
			return nil, err
		}
		ev.Meta = meta
		out = append(out, ev)
	}
	return out, rows.Err()
}

func (r *JobRepo) scanOne(ctx context.Context, q string, args ...any) (*domain.Job, error) {
	row := r.pool.QueryRow(ctx, q, args...)
	j, err := scanJob(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return j, err
}

type scannable interface {
	Scan(dest ...any) error
}

func scanJob(row scannable) (*domain.Job, error) {
	var j domain.Job
	var options, result []byte
	err := row.Scan(
		&j.ID, &j.BatchID, &j.Kind, &j.Status, &j.Driver, &j.ProxyRef, &j.Email, &j.Progress, &j.Step,
		&j.ErrorCode, &j.ErrorMessage, &options, &result, &j.CreatedAt, &j.UpdatedAt, &j.StartedAt, &j.FinishedAt,
	)
	if err != nil {
		return nil, err
	}
	j.Options, j.Result = options, result
	return &j, nil
}

// Ensure time import used if needed for future helpers.
var _ = time.Now
