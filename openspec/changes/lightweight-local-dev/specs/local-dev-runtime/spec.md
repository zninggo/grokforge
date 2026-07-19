## ADDED Requirements

### Requirement: Dev profile allows boot without Redis

The system SHALL allow process startup when `GROKFORGE_ENV` is `dev` (case-insensitive) and `REDIS_URL` / `GROKFORGE_REDIS_URL` is empty, provided `DATABASE_URL` is valid and JWT secret rules are satisfied.

#### Scenario: Dev boot with Postgres only
- **WHEN** `GROKFORGE_ENV=dev`, a reachable `DATABASE_URL` is set, Redis URL is unset, and a valid master/JWT secret is set
- **THEN** the process starts successfully and serves HTTP on the configured address

#### Scenario: Non-dev still requires Redis
- **WHEN** `GROKFORGE_ENV` is not `dev` (or is unset/prod) and Redis URL is empty
- **THEN** configuration loading fails with an error that Redis URL is required

### Requirement: Postgres remains mandatory

The system MUST require a non-empty `DATABASE_URL` (or `GROKFORGE_DATABASE_URL`) in all environments, including dev.

#### Scenario: Missing database URL
- **WHEN** configuration is loaded without a database URL
- **THEN** loading fails regardless of `GROKFORGE_ENV`

### Requirement: Management plane works without Redis in dev

When running in dev without Redis, the system SHALL serve management-plane functionality that depends only on Postgres (and optional external APIs), including setup, authentication, accounts, proxies, audit listing, and system status reads.

#### Scenario: Account list without Redis
- **WHEN** the server is running in dev without Redis and the admin is authenticated
- **THEN** account list/read APIs succeed if Postgres is healthy

#### Scenario: Job create without Redis
- **WHEN** a client creates jobs while Redis is not configured
- **THEN** the API responds with HTTP 503 and error code `REDIS_UNAVAILABLE` (or equivalent stable code) and does not leave jobs stuck in a queued-without-broker state without a failed mark

### Requirement: Workers do not consume queues without Redis

When Redis is not configured, the system MUST NOT start Redis-backed queue consumers that would nil-dereference or busy-fail; job execution via Redis queues SHALL be inactive until Redis is configured and the process is restarted (or reconnected per implementation).

#### Scenario: No worker consume without broker
- **WHEN** the server boots in dev without Redis
- **THEN** no Redis `BRPOP` consumer loop is active against a nil client

### Requirement: Readyz treats missing Redis as non-fatal in dev

In dev, when Redis is not configured, `/readyz` MUST report Postgres health as the readiness gate and MUST NOT mark the service not-ready solely because Redis is absent. The checks object SHOULD include `redis` as `not_configured`.

#### Scenario: Ready without Redis in dev
- **WHEN** Postgres is up, env is dev, and Redis is not configured
- **THEN** `/readyz` returns HTTP 200 with overall ready status and `checks.redis` indicating not configured

#### Scenario: Ready still fails if Postgres is down
- **WHEN** Postgres is unreachable
- **THEN** `/readyz` returns not ready regardless of Redis configuration

### Requirement: Configured Redis uses existing production path

When a Redis URL is set and the client connects successfully, enqueue, job locks, and event publish SHALL behave as in the current Redis-backed implementation.

#### Scenario: Job create with Redis
- **WHEN** Redis is configured and healthy
- **THEN** creating a job enqueues it and marks it queued as today

### Requirement: Documented local path reuses Postgres and native binary

Project documentation and env examples MUST present the default local development path as: reuse an existing/shared PostgreSQL instance, run the native Go binary (and optional frontend embed build), with Redis optional; application image build MUST be documented under production/release, not as the default day-to-day dev step.

#### Scenario: README quick start order
- **WHEN** a developer follows the primary quick-start for local development
- **THEN** steps use database URL against an existing Postgres and a local binary, and do not require `docker build` of the app image

#### Scenario: Env example reflects optional Redis
- **WHEN** a developer copies `.env.example`
- **THEN** Redis is shown as optional/commented for local dev and Postgres URL is the required data dependency

### Requirement: Public git surface stays desensitized

Files intended for the public repository MUST NOT contain real secrets, credentialed proxy URIs, personal identifiers, or private host-specific inventory. Committed env samples MUST use empty or obviously fake placeholders. Host-specific operator notes, if any, MUST live only in paths that are gitignored (for example local `docs/`).

#### Scenario: Env example has no live secrets
- **WHEN** `.env.example` is inspected as committed
- **THEN** `MASTER_KEY`, API keys, and passwords are empty or clearly fake placeholders and not production values

#### Scenario: Public quick start has no private topology dump
- **WHEN** the public README local-dev section is read
- **THEN** it describes reusable Postgres and optional Redis in generic terms without embedding private host credentials or non-public network details
