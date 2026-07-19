## Why

Local development currently assumes a production-shaped stack: a dedicated Postgres + Redis via `docker compose`, and docs mix day-to-day `go build` with application image builds. On machines that already run a shared Postgres (and rarely need the job queue), that is heavy and misaligned with how this host actually works. We need an explicit lightweight local profile: reuse existing Postgres, run the native binary, treat Redis as optional for management-plane work, and keep image builds for release only.

## What Changes

- Add a **dev runtime profile** (`GROKFORGE_ENV=dev`): process starts with Postgres only; Redis is optional.
- When Redis is unset in dev: management APIs (setup, auth, accounts, proxies, system, health) work; creating jobs returns a clear `503 REDIS_UNAVAILABLE`; workers do not consume queues.
- When Redis is configured: behavior matches today’s production path (enqueue, locks, pub/sub).
- Production / non-dev: Redis remains **required** at config load (no silent degradation).
- `/readyz`: in dev without Redis, report `redis=not_configured` and still ready if Postgres is up.
- Document and script the default local path as: **shared/reused Postgres + native binary**; optional one-off Redis container; **not** `docker build` of the app image.
- Refresh `.env.example`, README quick start, ops/dev docs, and product decision table to record the dev contract.
- Demote full data `docker-compose.yml` to optional sandbox (or document it as non-default).

Non-goals for this change:

- SQLite or any second SQL backend.
- In-process memory job queue (follow-up if needed).
- Auto-provisioning databases on the shared Postgres instance.
- Changing production deployment topology (app image + separated PG/Redis).

## Public repository redaction (hard constraint)

This repo is **public**. Every file that may be committed MUST stay desensitized.

**MUST NOT enter git (or image layers as plaintext):**

- Real API keys (`AC-…`), proxy URIs with credentials, admin passwords, `MASTER_KEY` / JWT secrets, session tokens/cookies
- Personal emails/phones, private hostnames, non-public internal network topology, other tenants’ DB names on a shared instance
- Filled `.env`, runtime `data/`, and local full docs under `docs/` (already gitignored)

**MAY enter git only as placeholders / fakes:**

- `.env.example` empty or obviously fake values
- Compose demo user/pass like `grokforge:grokforge` on loopback example ports
- Generic `127.0.0.1` example ports (compose sandbox `25432`/`26379`), not machine-specific inventory dumps

**Split:**

| Surface | Content |
|---------|---------|
| Public (`README`, `.env.example`, code, optional `openspec/`) | Generic local-dev contract only |
| Local only (`docs/`, `.env`, `docs/private/`) | Host-specific ports, shared-instance how-to, real URLs |

Implementation and commit review for this change MUST follow the existing public-repo red lines in local `docs/ops/security.md`.

## Capabilities

### New Capabilities

- `local-dev-runtime`: Lightweight local development profile — env/config gates, optional Redis, readiness semantics, default ops path (reuse PG + native binary), and explicit separation from production image builds.

### Modified Capabilities

- (none — `openspec/specs/` has no prior capability specs)

## Impact

- **Config / boot**: `backend/internal/config`, `backend/cmd/grokforge/main.go` — Redis required only outside dev (or when URL set).
- **Queue / worker**: `backend/internal/queue`, `backend/internal/worker`, `backend/internal/server` — nil-safe broker when Redis absent; skip worker loop.
- **API behavior**: job create already maps `ErrRedisUnavailable` → 503; health/setup/system already tolerate `Redis == nil` for checks — wire consistently.
- **Docs / DX**: `README.md`, `.env.example`, `Makefile` (optional `dev` / `redis-dev` targets), `docs/ops/*`, `docs/dev/*`, `docs/product/decisions.md`.
- **Compose**: `docker-compose.yml` remains available as sandbox; not the documented default day-to-day path.
- **No schema/migration changes**; no GHCR/Dockerfile behavior changes beyond documentation boundaries.
