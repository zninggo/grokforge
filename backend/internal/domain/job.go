package domain

import "time"

// Job kinds.
const (
	JobKindNoop     = "noop"
	JobKindRegister = "register"
	JobKindProbe    = "probe"
	JobKindPipeline = "pipeline"
)

// Job statuses.
const (
	JobPending    = "pending"
	JobQueued     = "queued"
	JobRunning    = "running"
	JobSucceeded  = "succeeded"
	JobFailed     = "failed"
	JobCancelled  = "cancelled"
	JobTimedOut   = "timed_out"
	JobRetrying   = "retrying"
)

// Job is the durable task record.
type Job struct {
	ID           int64
	BatchID      *string
	Kind         string
	Status       string
	Driver       string
	ProxyRef     string
	Email        string
	Progress     int
	Step         string
	ErrorCode    string
	ErrorMessage string
	Options      []byte
	Result       []byte
	CreatedAt    time.Time
	UpdatedAt    time.Time
	StartedAt    *time.Time
	FinishedAt   *time.Time
}

// JobEvent is an append-only log line for a job.
type JobEvent struct {
	ID        int64
	JobID     int64
	Level     string
	Message   string
	Meta      []byte
	CreatedAt time.Time
}
