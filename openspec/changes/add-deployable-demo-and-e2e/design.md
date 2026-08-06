## Context

The current service embeds the browser application, provides deterministic calculation APIs, optionally calls a model through trusted tools, and stores browser-created records in SQLite. Automated tests cover packages and handlers, but not a rendered browser. Local startup also depends on a developer-selected database path and lacks a packaged health contract or aggregate request telemetry.

## Goals / Non-Goals

**Goals:**
- Run a small real-browser suite against an isolated server and database.
- Provide stable, versioned demo records through the existing CSV importer rather than a second seeding implementation.
- Package the same server binary in a non-root container with durable SQLite storage and an explicit health check.
- Attach a generated request ID to every HTTP response and write structured metadata for completed requests and server errors.
- Expose bounded process-local request and assistant counters for smoke checks and demos.

**Non-Goals:**
- Public hosting, TLS termination, horizontal scaling, PostgreSQL, authentication, or multi-user isolation.
- Persisting metrics, logs, prompts, model responses, or conversation history.
- Replacing the existing unit and race tests with browser tests.
- Exercising Steam or OpenAI over the network in the default E2E suite.
- Calculating or modifying seed values in JavaScript or model code.

## Decisions

### 1. Reuse the existing CSV import path for demo data

The repository will contain a small compatible CSV generated from deterministic calculation output. Native demo startup and Compose both pass that file through `--import-csv`; the source-aware SQLite importer makes repeated startup idempotent. E2E starts with a fresh temporary database, so expected filters and page counts are stable.

### 2. Use Playwright as an explicit browser gate

Playwright Test will launch the Go server through its `webServer` lifecycle, use Chromium in headless mode, and isolate each run in a temporary data directory. Tests will assert visible user states and API-backed persistence. The regular `make check` remains Go-first; `make e2e` is an additional gate because it installs and launches a browser runtime.

### 3. Package one server process and one durable volume

The image uses a multi-stage build, copies only the server binary and demo CSV into the runtime image, runs as an unprivileged user, and stores SQLite files under `/data`. Compose binds port 8080, attaches a named volume, enables idempotent seed import, checks `/api/health`, and gives graceful shutdown enough time to checkpoint SQLite.

### 4. Keep telemetry process-local and bounded

An observability component uses atomic counters for total requests, in-flight requests, response classes, accumulated duration, assistant runs/failures, token usage, and tool calls. `GET /api/v1/metrics` returns one aggregate snapshot and never returns request bodies, item names, prompts, tool payloads, credentials, or raw provider errors. Restarting the process resets counters.

### 5. Log request metadata, not user content

Middleware generates a request ID, returns it in `X-Request-ID`, records status and duration, and emits structured JSON logs. Successful requests are informational; 4xx responses are warnings and 5xx responses are errors. Logs include method and path but omit query strings, bodies, headers, assistant text, lookup data, and database values.

### 6. Keep readiness compatible

The existing `/api/health` response remains successful when the configured repository is usable. Repositories may implement a health-check interface; SQLite checks its live connection and CSV mode checks that its configured parent can be used. A dependency failure returns `503` without changing calculation or history contracts.

## Failure Semantics And Security Boundary

- A failed browser assertion exits the E2E command non-zero and preserves an HTML report artifact in CI.
- Browser tests use a fresh database and disabled AI provider, so no real credentials or network providers are required.
- Seed import failure prevents the demo server from listening and leaves the import transaction unchanged.
- Container health failure marks the service unhealthy but does not delete or recreate the data volume.
- Metrics are aggregate and read-only; they expose no secrets or per-user content.
- Logging never records headers, bodies, query strings, prompts, model output, market responses, or persisted record contents.
- AI numerical claims remain governed by existing tool grounding; observability only counts completion metadata returned by that workflow.

## Migration And Rollback

1. Native users may continue starting either binary exactly as before.
2. Demo users run the documented seed/import command or Compose profile; an existing database safely skips already-imported seed rows.
3. Removing the container and retaining the named volume preserves SQLite history.
4. Roll back by running the prior tagged binary against the same SQLite file or by selecting the documented CSV storage mode.
5. Disable the new browser CI job independently if a browser infrastructure outage occurs; deterministic Go checks remain intact.

## Risks / Trade-offs

- [Browser packages add CI time and download size] -> Keep one Chromium project and a small smoke suite.
- [In-memory metrics reset on restart] -> Treat them as demo and troubleshooting signals; use a managed metrics backend only with a later deployment target.
- [SQLite allows one application instance] -> Document a single replica and defer PostgreSQL until a multi-instance deployment is selected.
- [Seed timestamps are assigned at import time] -> Assert stable values and ordering, not fixed wall-clock timestamps.
- [Container validation is unavailable on machines without Docker] -> Validate the native binary and E2E locally, and run image construction in CI.
