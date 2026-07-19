CREATE TABLE IF NOT EXISTS proxy_pool (
    id              BIGSERIAL PRIMARY KEY,
    line_raw        TEXT NOT NULL,
    scheme          TEXT NOT NULL DEFAULT 'http',
    host            TEXT NOT NULL,
    port            INT NOT NULL,
    username        TEXT NOT NULL DEFAULT '',
    password_cipher TEXT NOT NULL DEFAULT '',
    display         TEXT NOT NULL DEFAULT '',
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    fail_count      INT NOT NULL DEFAULT 0,
    last_error      TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_proxy_pool_enabled ON proxy_pool (enabled);
