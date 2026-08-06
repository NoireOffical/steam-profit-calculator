## 1. Demo And Browser Gate

- [x] 1.1 Add deterministic seed records that are imported through the existing idempotent CSV path
- [x] 1.2 Add Playwright configuration and isolated server lifecycle without external provider credentials
- [x] 1.3 Cover calculation persistence, history filtering/pagination, invalid input, and assistant-disabled fallback
- [x] 1.4 Add a concise three-to-five-minute demo walkthrough and local commands

## 2. Deployable Runtime

- [x] 2.1 Add a multi-stage, non-root container image containing only the server runtime and demo seed
- [x] 2.2 Add Compose startup with a named SQLite volume, idempotent seed import, health check, and graceful stop
- [x] 2.3 Document native/container startup, persistence, backup, restore, and rollback boundaries

## 3. Observability

- [x] 3.1 Add request IDs, status-aware structured logs, and aggregate request metrics without user content
- [x] 3.2 Add aggregate assistant completion, failure, latency, token, and tool counters
- [x] 3.3 Make health dependency-aware and expose a read-only metrics snapshot
- [x] 3.4 Test success, invalid request, dependency failure, no-secret logging, and concurrent metric access

## 4. Verification And Handoff

- [x] 4.1 Extend Make and CI with separate browser and container gates while preserving deterministic checks
- [x] 4.2 Run OpenSpec validation, `make check`, browser E2E, JavaScript syntax validation, and native smoke tests
- [x] 4.3 Build and health-check the container where Docker is available; document any local runtime limitation
- [x] 4.4 Update the roadmap, changelog, README files, and prepare immutable `v0.4.0` Pre-release notes
