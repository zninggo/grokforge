CREATE TABLE IF NOT EXISTS jobs (
    id              BIGSERIAL PRIMARY KEY,
    batch_id        TEXT,
    kind            TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    driver          TEXT NOT NULL DEFAULT '',
    proxy_ref       TEXT NOT NULL DEFAULT '',
    email           TEXT NOT NULL DEFAULT '',
    progress        INT NOT NULL DEFAULT 0,
    step            TEXT NOT NULL DEFAULT '',
    error_code      TEXT NOT NULL DEFAULT '',
    error_message   TEXT NOT NULL DEFAULT '',
    options         JSONB NOT NULL DEFAULT '{}'::jsonb,
    result          JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_jobs_status_kind_created
    ON jobs (status, kind, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_jobs_batch_id
    ON jobs (batch_id);

CREATE TABLE IF NOT EXISTS job_events (
    id          BIGSERIAL PRIMARY KEY,
    job_id      BIGINT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    level       TEXT NOT NULL DEFAULT 'info',
    message     TEXT NOT NULL,
    meta        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_job_events_job_id_id
    ON job_events (job_id, id);
