## Why

The application now has a measurable local AI workflow, browser E2E coverage, a non-root container, durable single-instance SQLite storage, and content-free runtime metrics. It still cannot be safely shared as an online portfolio demo because the browser and paid assistant endpoint have no access gate, request budget, hosted smoke workflow, or documented secret and rollback contract.

## What Changes

- Add optional demo-wide HTTP Basic authentication that protects the browser and application APIs while leaving the dependency health endpoint available to hosting platforms.
- Add bounded process-local API rate limiting and a separate assistant-run quota with safe `429` responses and content-free rejection counters.
- Add explicit manual live-model evaluation and hosted synthetic-smoke workflows that consume repository or environment secrets without placing paid calls in normal pull-request CI.
- Document a single-replica HTTPS deployment contract with secret injection, persistent SQLite storage, health checks, backup, restore, and immutable-tag rollback.
- Preserve local development defaults: when the new controls are not configured, the current native, CLI, CSV, SQLite, and browser workflows remain compatible.

## Capabilities

### New Capabilities

- `controlled-hosted-demo`: Share one single-replica demo through HTTPS with optional access authentication, bounded request/model usage, secret-safe automation, synthetic health checks, and an explicit rollback path.

### Modified Capabilities

- `runtime-observability`: Expose aggregate authentication, rate-limit, and assistant-quota rejection counters without logging credentials, client addresses, prompts, or request content.

## Impact

- New access-control configuration and HTTP middleware in `internal/webapi`.
- New content-free security counters in `internal/observability`.
- New environment variables, GitHub Actions workflows, deployment documentation, and smoke scripts.
- A Railway config-as-code manifest and platform `PORT` fallback for the selected hosted target.
- Existing local startup remains unprotected unless credentials or limits are explicitly configured.
- No formula, CLI, CSV, SQLite schema, prompt, tool schema, or assistant grounding changes.
