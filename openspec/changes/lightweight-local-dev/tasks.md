## 1. Config and boot

- [x] 1.1 Add `GROKFORGE_ENV` to config (`Env` field, `IsDev()`), bind env, include in `Redacted()`
- [x] 1.2 Allow empty Redis URL when `IsDev()`; keep required otherwise; set `ReadyRequireRedis=false` when dev and Redis unset
- [x] 1.3 Update `config` unit tests for dev-without-redis success and non-dev-without-redis failure
- [x] 1.4 Change `cmd/grokforge` to skip `NewRedis` when URL empty; pass nil client; log warn in dev

## 2. Queue and worker safety

- [x] 2.1 Make queue nil-safe: Unavailable broker or guarded `Queue` so `Ping`/`Enqueue` return `ErrUnavailable` without panic
- [x] 2.2 Guard `Publish`/`TryLock`/`Unlock`/`BRPop` on missing client
- [x] 2.3 Wire `JobService` to the safe broker (keep 503 mapping for create)
- [x] 2.4 Start workers in `server.New` only when Redis client is non-nil

## 3. Readiness and status surfaces

- [x] 3.1 Ensure `/readyz` with dev + no Redis returns ready when Postgres is up and `checks.redis=not_configured`
- [x] 3.2 Confirm setup/system dependency checks show redis not_configured without failing setup readiness incorrectly

## 4. Developer experience docs and scripts

- [x] 4.1 Update `.env.example`: `GROKFORGE_ENV=dev`, placeholder `DATABASE_URL` only (no real passwords/keys), Redis commented optional
- [x] 4.2 Rewrite public `README.md` quick start: native binary + reused/existing PG first; image build under release; optional Redis — **desensitized only**
- [x] 4.3 Annotate `docker-compose.yml` as optional sandbox (not default daily path)
- [x] 4.4 Optional Makefile helpers: `redis-dev` (and/or document-only `dev`); keep `image` as release
- [x] 4.5 Update **gitignored** local docs (`docs/ops/deploy.md`, `docs/ops/config-reference.md`, `docs/product/decisions.md`, `docs/dev/status.md`) with dev contract; host-specific ports/secret steps only under `docs/` or `docs/private/`, never public README
- [x] 4.6 If committing `openspec/`, ensure change text has no real secrets or private host inventory (generic wording only)

## 5. Verification and public-repo redaction

- [x] 5.1 `go test ./...` (config + any new queue tests)
- [x] 5.2 Manual smoke checklist: boot with `ENV=dev` without Redis → healthz/readyz/setup path; job create → 503; with Redis URL → enqueue works
- [x] 5.3 Operator-only: migrate smoke against their existing Postgres (not documented with private details in public files)
- [x] 5.4 Pre-commit / pre-push checklist for this change:
  - no `.env` staged; only `.env.example` with empty/fake values
  - no real `AC-` keys, proxy passwords, `MASTER_KEY`, JWT secrets
  - no personal emails/phones; no private hostnames or non-public topology dumps in tracked files
  - `docs/` remains gitignored; do not force-add it
  - `git status` / diff review before any commit to the public remote
