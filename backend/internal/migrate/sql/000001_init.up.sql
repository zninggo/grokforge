-- Phase 1 baseline schema (auth tables filled in Phase 2; present early for migrate smoke).

CREATE TABLE IF NOT EXISTS schema_migrations_meta (
    id          BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id),
    applied_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    note        TEXT NOT NULL DEFAULT 'grokforge'
);

CREATE TABLE IF NOT EXISTS setup_state (
    id              BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id),
    completed       BOOLEAN NOT NULL DEFAULT FALSE,
    completed_at    TIMESTAMPTZ
);

INSERT INTO setup_state (id, completed)
VALUES (TRUE, FALSE)
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS admin_users (
    id              BIGSERIAL PRIMARY KEY,
    username        TEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL DEFAULT '',
    must_reset_password BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS app_settings (
    key         TEXT PRIMARY KEY,
    value       JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
