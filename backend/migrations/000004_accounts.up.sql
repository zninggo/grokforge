CREATE TABLE IF NOT EXISTS accounts (
    id              BIGSERIAL PRIMARY KEY,
    email           TEXT NOT NULL,
    upstream        TEXT NOT NULL DEFAULT 'chatgpt',
    status          TEXT NOT NULL DEFAULT 'active',
    health_status   TEXT NOT NULL DEFAULT 'unknown',
    health_detail   TEXT NOT NULL DEFAULT '',
    plan_snapshot   JSONB NOT NULL DEFAULT '{}'::jsonb,
    meta            JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_probed_at  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (upstream, email)
);

CREATE INDEX IF NOT EXISTS idx_accounts_health ON accounts (health_status);

CREATE TABLE IF NOT EXISTS credentials (
    id              BIGSERIAL PRIMARY KEY,
    account_id      BIGINT NOT NULL UNIQUE REFERENCES accounts(id) ON DELETE CASCADE,
    cipher_blob     TEXT NOT NULL,
    key_version     INT NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS email_leases (
    id              BIGSERIAL PRIMARY KEY,
    provider        TEXT NOT NULL DEFAULT 'yyds',
    external_id     TEXT NOT NULL DEFAULT '',
    address         TEXT NOT NULL,
    token_cipher    TEXT NOT NULL DEFAULT '',
    job_id          BIGINT,
    status          TEXT NOT NULL DEFAULT 'active',
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_email_leases_status_exp ON email_leases (status, expires_at);
