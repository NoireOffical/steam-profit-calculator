## Context

The current Go server embeds the browser application and exposes deterministic calculation, lookup, history, metrics, health, and optional paid assistant endpoints. The container already runs as a non-root user with a read-only root filesystem and a named SQLite volume, but Compose binds only to localhost and intentionally disables the model. Public hosting requires a small protection layer before a reviewer-facing URL or provider key can be added.

The intended milestone is a controlled portfolio demonstration, not a multi-user production service. A single replica and one SQLite volume remain acceptable as long as the deployment prevents horizontal scaling, protects paid calls, keeps secrets outside the image and repository, and has a rehearsed rollback path.

## Goals / Non-Goals

**Goals:**
- Require one environment-supplied demo credential for the browser and application APIs when protection is enabled.
- Keep `/api/health` unauthenticated so hosting platforms and synthetic monitors can check the live repository dependency.
- Bound API traffic by client identity and bound paid assistant runs separately.
- Reject unauthenticated, rate-limited, and quota-exhausted requests before body decoding, provider calls, or storage changes.
- Record only aggregate rejection counts and safe request metadata.
- Provide opt-in live evaluation and hosted smoke workflows whose secrets are injected by GitHub or the hosting platform.
- Document one-replica persistent storage, TLS termination, backup, restore, and immutable-tag rollback requirements.

**Non-Goals:**
- User registration, password reset, roles, tenants, durable sessions, or cross-device chat history.
- Persisted/distributed rate limits or quotas across multiple replicas.
- PostgreSQL, horizontal scaling, managed telemetry, or a vendor-specific paid resource created without user approval.
- Automatic paid evaluation in pull requests or automatic retries of failed model calls.
- Logging credentials, client IP addresses, request bodies, prompts, answers, tool values, or provider payloads.

## Decisions

### 1. Use optional HTTP Basic authentication for the controlled demo

The server accepts a username and password only from environment configuration. When both are empty, local behavior is unchanged. When either is set, both are required and startup fails on partial or weak configuration. The middleware uses constant-time credential comparison, challenges unauthorized requests, and protects the embedded browser, metrics, and all versioned APIs. `/api/health` remains public and content-free.

Basic authentication is intentionally a portfolio-demo boundary rather than an account system. The deployment contract requires HTTPS termination before enabling it. Credentials never appear in HTML, JavaScript, image layers, command-line arguments, logs, or metric labels.

### 2. Use bounded process-local fixed-window controls

The server applies a configurable per-client API request limit over a one-minute window and a separate global assistant-run limit over a one-hour window. Defaults are disabled for local compatibility. State is held in a bounded map, old client windows are evicted, and the deployment remains one replica so limits cannot be bypassed by load balancing.

Client identity uses the remote peer address by default. A forwarded client header is trusted only when an explicit proxy setting is enabled for a known hosting proxy. Rejections return `429` and `Retry-After` before handlers decode JSON or call dependencies.

The assistant quota is consumed only after request/history validation and immediately before a provider run begins. Invalid input and unavailable-provider responses do not consume paid-run budget. Cancellation or provider failure does consume one attempt because an upstream call may already have occurred.

### 3. Keep security telemetry aggregate and content-free

The existing process-local metrics snapshot gains totals for unauthorized requests, API rate-limit rejections, and assistant-quota rejections. Counters contain no client key, address, credential, route parameters, prompt, answer, or model payload. Existing request logging continues to include only request ID, method, path, status, and duration.

### 4. Separate deterministic CI, paid evaluation, and hosted smoke

Normal push and pull-request CI remains deterministic and credential-free. A manual live-model workflow requires an explicit dispatch and a repository secret, runs a bounded current and baseline prompt prefix, writes content-free JSON reports, and uploads them with short retention. Missing credentials fail before cases run.

A separate synthetic workflow accepts a configured hosted base URL and optional demo credentials, checks public dependency health, then authenticates and checks assistant status without submitting a paid assistant prompt. Its failure is the initial alert signal through GitHub Actions status and notifications.

### 5. Use Railway only after explicit target approval

The repository documents a vendor-neutral contract first: one container replica, HTTPS, secret injection, a persistent `/data` volume, `/api/health`, graceful stop, and an immutable image/tag rollback. The user selected Railway, so the repository adds `railway.toml` for the existing Dockerfile, deployment health check, bounded restart, and drain behavior. Service startup follows Railway's injected `PORT` unless the explicit server address takes precedence. The `/data` Volume, public domain, variables, and single replica remain service settings that must be verified during deployment.

Railway documents `X-Real-IP` as its client-address request header. Trusted-proxy mode therefore accepts a valid `X-Real-IP` before the compatible `X-Forwarded-For` fallback. The Railway service enables this option only when traffic arrives directly through Railway's managed HTTPS edge.

## Failure Semantics

- Partial access credentials or negative limits prevent server startup with a safe configuration error.
- Unauthenticated protected requests return `401`, include a Basic challenge, and never reach application handlers.
- API requests over the configured rate return `429` with a bounded retry interval and have no storage or provider side effect.
- Valid assistant requests over the hourly quota return `429` before SSE headers and before a provider call.
- Missing workflow secrets prevent paid evaluation from starting; no fake report or successful live claim is produced.
- Hosted health failure makes the synthetic workflow fail without attempting data mutation or an assistant prompt.

## Security Boundaries

- Access credentials and OpenAI keys are environment or secret-manager inputs only.
- HTTPS is mandatory whenever Basic authentication is enabled outside loopback development.
- Proxy headers are ignored unless an explicit trusted-proxy option is enabled.
- Rate-limit state and metrics are process-local, bounded, content-free, and reset on restart.
- Assistant tools remain strict-schema and read-only; deterministic calculation APIs keep their existing validation and transaction behavior.

## Rollback

Disable access and limits by removing the new environment variables in local development. Disable paid model calls with `STEAM_PROFIT_AI_PROVIDER=disabled`. A hosted rollback selects the previous immutable application tag or image while retaining the same single SQLite volume. No database migration is introduced, so `v0.5.0` can read the existing data unchanged.

## Risks / Trade-offs

- [Basic authentication is not an account system] -> Label it as a controlled-demo gate and require HTTPS; defer durable identity to a later user-isolation milestone.
- [In-memory limits reset on restart] -> Run one replica and treat the controls as cost protection, not billing enforcement.
- [Client address behind a proxy may collapse into one bucket] -> Trust Railway's documented `X-Real-IP` only under explicit deployment configuration, with a compatible forwarded-for fallback for other approved proxies.
- [Scheduled live evaluation can create recurring cost] -> Ship manual dispatch first; document scheduling as an explicit later opt-in.
- [SQLite volume ties the service to one instance] -> Keep one replica and defer PostgreSQL until multi-instance or user isolation is required.
