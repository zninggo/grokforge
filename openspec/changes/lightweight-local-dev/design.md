## Context

GrokForge is a single-process control plane (API + worker + embedded UI) with PostgreSQL as system of record and Redis for queues/locks/pubsub. Product decisions lock production to separated PG + Redis and refuse new jobs when Redis is down.

Typical developer reality (generic — do not encode private host inventory in public files):

- Many operators already run a **shared or long-lived Postgres** on loopback and prefer reusing it over composing a second instance per project.
- Redis is often unused for pure management-plane work.
- Day-to-day work is native `go build` / binary run; app image build is a Phase 9 release concern.
- Optional compose data services (example ports in-repo: Postgres `25432`, Redis `26379`) remain a **sandbox**, not the only path.

**Public repo redaction (hard):** This repository is public. Committed artifacts MUST NOT contain real keys, passwords, proxy credentials, personal identifiers, or private host/topology fingerprints. Host-specific runbooks (exact ports, shared-instance admin steps, secrets file paths) belong only under gitignored `docs/` / `.env`. Public README and `.env.example` use placeholders and obvious fakes only. Align with local `docs/ops/security.md` red lines.

Today `config.Load` **requires** both `DATABASE_URL` and `REDIS_URL`. Boot always `NewRedis` + pings. Queue is a concrete `*queue.Queue` over `*redis.Client`. Health already has `ReadyRequireRedis` and nil-safe display for Redis, but config never allows a nil path.

## Goals / Non-Goals

**Goals:**

- Dev profile: boot with Postgres only; Redis optional.
- Clear operator UX: management plane works; job create fails loudly without Redis.
- Production safety: non-dev still requires Redis at config time.
- Align docs/Makefile/.env.example with reuse-shared-PG + native binary; image build is release-only.
- Minimal code surface; no storage engine swap.

**Non-Goals:**

- SQLite / dual SQL backends.
- In-memory job queue that fully replaces Redis for local job runs (optional follow-up).
- Automatic `CREATE DATABASE` on the shared instance.
- Changing GHCR/Dockerfile runtime topology.
- Multi-process Redis pub/sub fan-out redesign.

## Decisions

### D1 — Gate on `GROKFORGE_ENV=dev` (not “empty URL means optional”)

**Choice:** Redis is optional only when `GROKFORGE_ENV` is `dev` (case-insensitive). Empty Redis URL outside dev → config error (unchanged hardness).

**Why:** Empty URL alone is easy to mis-deploy. Explicit env makes intent visible in logs via `Redacted()`.

**Alternatives considered:**

| Option | Rejected because |
|--------|------------------|
| `GROKFORGE_REDIS_OPTIONAL=1` only | Easy to leave on in prod compose |
| Always optional Redis | Violates product rule “Redis down refuses jobs” and weakens prod |
| Separate binary `grokforge-dev` | Unnecessary split |

**Detail:** When `ENV=dev` and Redis URL empty → `ReadyRequireRedis=false`. When Redis URL is set in dev → connect as today; if ping fails, still fail boot (no half-connected mode).

### D2 — Nil-safe broker, not a full memory queue (this change)

**Choice:** Introduce a small broker interface (or equivalent) with:

- `Redis` implementation = current list/lock/pubsub behavior.
- `Unavailable` / nil implementation: `Ping`/`Enqueue` return `ErrRedisUnavailable`; `BRPop` not used because worker is not started.

**Why:** Meets “optional Redis” without inventing crash-recovery semantics for an in-process queue. Job create already maps `ErrRedisUnavailable` → 503.

**Follow-up (out of scope):** `MemoryBroker` with channel + mutex + direct hub broadcast for local noop/register dry-runs without Redis.

### D3 — Skip worker loop when no Redis client

**Choice:** `server.New` only starts `worker.Worker` goroutine(s) when a live Redis client exists.

**Why:** Worker’s first action is `BRPop`; nil client would panic or spin errors.

### D4 — Reuse existing Postgres; do not force a second PG container as the default path

**Choice:** Public docs present **two** Postgres options as first-class: (1) point `DATABASE_URL` at an **existing** Postgres the operator already runs; (2) optional `docker compose` sandbox. Neither path embeds real credentials. Example loopback ports in public files stay the in-repo sandbox mapping (`25432`) or a neutral placeholder — not a dump of any one machine’s layout.

**Why:** Avoids mandatory extra containers when an instance already exists; keeps public text portable and desensitized.

### D5 — Docs / DX layering

```text
Daily:    .env (PG + ENV=dev) → make backend → ./backend/bin/grokforge
Optional: make redis-dev (or docker run redis:alpine) + REDIS_URL
Sandbox:  docker compose up (full data services)
Release:  make image / compose.app / GHCR
```

**Choice:** Add or adjust Makefile targets for clarity (`dev` optional helper, `redis-dev` optional). Do not remove `make image`.

### D6 — Product decision delta

**Choice:** Amend `docs/product/decisions.md` with a “local development” row: reuse PG, native binary, Redis optional in dev; production Redis still required. Not a break of “PG+Redis separated in production.”

## Implementation sketch

```text
config.Load
  Env string  // from GROKFORGE_ENV, default ""
  IsDev() bool
  if RedisURL == "" && !IsDev() → error
  if RedisURL == "" && IsDev() → ReadyRequireRedis=false

main.run
  pool := NewPool(DATABASE_URL)  // required
  migrate
  var rdb *redis.Client
  if cfg.RedisURL != "" {
    rdb = NewRedis(...)
  } else {
    log.Warn("redis disabled (dev)")
  }
  server.New(cfg, log, pool, rdb)

queue
  type Broker interface { Ping, Enqueue, BRPop, TryLock, Unlock, Publish }
  NewRedisBroker(rdb)
  NewUnavailableBroker() // or queue.New(nil) with guards

server
  broker := ...
  if rdb != nil { start workers }
  HealthHandler{Redis: rdb, ReadyRequireRedis: cfg.ReadyRequireRedis}

JobService
  keep Ping before create → ErrRedisUnavailable
  emit: if Publish fails, ignore (already best-effort); guard nil hub/broker
```

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| `GROKFORGE_ENV=dev` left on in production | Default env not dev; image/compose examples set prod or omit; log env in startup redacted config |
| Developers assume jobs work without Redis | 503 + stable code; setup/system checks show redis not_configured; docs capability matrix |
| Nil dereference in queue/worker | Unavailable broker + don’t start workers; unit tests for config + enqueue path |
| Shared/existing PG credential mishandling | Public placeholders only; never commit real password; dedicated DB name `grokforge` |
| PG version drift (e.g. sandbox 16 vs operator 18) | SQL is generic; operator smoke-migrates on their instance |
| Docs in `/docs` are gitignored | Update them for local operators only; public README carries the desensitized quick start |
| Accidental commit of host fingerprints / secrets | Pre-commit checklist (tasks §5); no real `.env`; scrub `openspec`/README before push |
| `openspec/` if published | Keep change text free of private topology; host notes stay in gitignored `docs/private/` |

## Public vs local content map

```text
COMMIT OK (desensitized)
  README.md, .env.example, Makefile, compose examples
  Go/TS source, Dockerfile
  openspec/ changes (generic contract only)

DO NOT COMMIT
  .env, data/, credentials, keys
  /docs/** (gitignore) — may hold host-specific how-to
  docs/private/** — real ports, secret paths, operator notes
```

## Migration Plan

1. Land code + desensitized `.env.example` + public README.
2. Operators locally: create dedicated DB/user on their Postgres; copy `.env` with `GROKFORGE_ENV=dev` (**never commit `.env`**).
3. Optional: host-specific steps only in gitignored `docs/dev/` or `docs/private/`.
4. No data migration; no production config change required.
5. Rollback: revert commits; old behavior requires Redis again (safe default).
6. Before any push: confirm no secrets/internal inventory in staged files.

## Open Questions

- None blocking. Optional later: memory broker; `make dev` exact recipe (build-only vs build+run).
