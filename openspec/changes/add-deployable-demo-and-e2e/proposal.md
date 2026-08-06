## Why

The application now has a complete local browser-to-SQLite workflow, but the project still depends on manual startup and manual browser checking. A reviewer cannot reproduce a known dataset, run a real-browser regression suite, or launch the same persisted service in a container with an explicit health signal. The next milestone should make the existing AI full-stack path repeatable and demonstrable without changing trusted calculation behavior.

## What Changes

- Add a deterministic demo dataset and a short, versioned walkthrough that exercises calculation, history, and assistant-degraded behavior.
- Add Playwright browser tests for calculation persistence, filtering and pagination, validation errors, and disabled-assistant fallback.
- Add a production-style container image and Compose configuration with a named SQLite volume, idempotent seed import, non-root execution, health check, and graceful stop behavior.
- Add request IDs, structured request/error logging, and bounded in-memory request and assistant metrics exposed through a read-only endpoint.
- Extend CI and local commands so deterministic Go checks remain independent while the browser suite runs as an explicit gate.

## Capabilities

### New Capabilities

- `deployable-demo-runtime`: Reproduce and verify the browser application with stable seed data, real-browser tests, container persistence, health checks, and operational handoff documentation.
- `runtime-observability`: Observe request outcomes and aggregate assistant usage without logging request bodies, prompts, credentials, or market payloads.

### Modified Capabilities

- `web-application-interface`: Expose readiness and read-only runtime metrics while preserving existing API and browser behavior.

## Impact

- New Playwright project files, demo assets, container manifests, and CI job.
- New `internal/observability` package and HTTP middleware.
- Updated web handler construction and assistant completion accounting.
- No calculation formula, CLI, CSV schema, SQLite schema, or model prompt changes.
- Docker is an optional delivery path; the native Go binaries remain supported.
