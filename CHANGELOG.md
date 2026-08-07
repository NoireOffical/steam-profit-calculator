# Changelog

All notable demonstrable milestones are recorded here. Until the public-operation requirements are complete, each version is published as a GitHub Pre-release with an immutable annotated tag.

## [v0.6.0] - Unreleased

### Added

- Optional shared Basic credentials for the browser, static assets, and application APIs while keeping dependency health public.
- Bounded per-client API request limiting and a process-wide hourly quota for provider-started assistant runs.
- Content-free aggregate counters for unauthorized, rate-limited, and assistant-quota rejections.
- A seventh Chromium workflow covering public health, rejected credentials, and authenticated browser access.
- Manual-only current/baseline live-model evaluation with bounded cases and short-lived sanitized report artifacts.
- Scheduled/manual read-only hosted smoke checks plus a Railway manifest and HTTPS, secret, persistent-volume, backup, restore, and rollback runbook.
- Railway `PORT` startup compatibility and trusted `X-Real-IP` client identification without changing explicit local address behavior.

### Compatibility

- All access controls default to disabled, so existing local server, CLI, calculation, SQLite, and CSV workflows remain unchanged.
- Health checks remain unauthenticated; trusted forwarded client addresses require an explicit opt-in.
- No live Railway deployment or paid model baseline is claimed until persistent storage, hosted smoke, restore evidence, and temporary model credentials are supplied and verified.

## [v0.5.0] - 2026-08-07

### Added

- Typed OpenAI Responses streaming for text and function-call arguments with bounded parsing.
- Up to six validated page-local prior messages while keeping conversations out of persistence and logs.
- Accessible Stop and explicit Retry controls with HTTP-context cancellation and no automatic paid retry.
- Fifty deterministic tool cases plus a 12-case semantic suite and content-free JSON report engine.
- Opt-in current/baseline prompt evaluation with latency, token, grounding, tool-selection, and caller-priced cost thresholds.
- Two Chromium workflows for transient context/retry and cancellation without calculation-history side effects.

### Changed

- Prompt `steam-profit-assistant-v1.1.0` treats prior conversation as untrusted context and requires fresh current-turn evidence for all market or calculated values.
- Grounded provider text is emitted with its original native delta boundaries only after whole-answer validation; unsafe partial model text is never exposed.
- `make build` now also produces the opt-in `steam-profit-eval` binary.

### Compatibility

- Existing `message`-only assistant requests remain valid; the optional `history` array is additive and transient.
- CLI commands, calculation APIs and formulas, CSV schemas, SQLite schema, and deterministic provider-disabled workflows remain unchanged.
- Live semantic evaluation is never part of normal CI and requires an explicit `--live` flag plus replaceable environment credentials.

## [v0.4.0] - 2026-08-06

### Added

- Four isolated Playwright Chromium workflows for persisted calculation, history pagination, invalid input, and AI-disabled fallback.
- Deterministic 12-row demo seed imported through the existing idempotent CSV path.
- Non-root multi-stage container image and Compose runtime with a named SQLite volume and health check.
- Request correlation IDs, content-free structured logs, and aggregate request/assistant metrics.
- Dependency-aware health checks for SQLite and legacy CSV repositories.
- CI browser and container persistence gates plus a demo, backup, restore, and rollback runbook.

### Compatibility

- CLI commands, calculation formulas, CSV schemas, SQLite schema, prompt version, and existing JSON workflows remain unchanged.
- Metrics are process-local and reset on restart; the container remains a single SQLite-backed application instance.

## [v0.3.0] - 2026-08-06

### Added

- Versioned CGo-free SQLite repository for web calculation history.
- Durable record IDs, UTC timestamps, source metadata, and reserved ownership/session fields.
- Atomic web batches and explicit idempotent CSV-to-SQLite import.
- Item/outcome filters, stable order, bounded pagination, and matching browser controls.
- Explicit legacy CSV web mode for compatibility and operational rollback.

### Compatibility

- Existing CLI commands and all supported CSV schemas remain unchanged.
- SQLite is now the default web storage; `--storage=csv --file ...` restores the previous mode.
- Source CSV files are never modified by SQLite import.

## [v0.2.0] - 2026-08-06

### Added

- Optional OpenAI Responses API model provider with a disabled-by-default fallback.
- Read-only `lookup_items`, `calculate_profit`, and `solve_target` assistant tools.
- SSE assistant status, progress, evidence, answer, usage, and error events.
- Responsive browser conversation panel with structured trusted-evidence cards.
- Numeric grounding guard and 30 deterministic evaluation fixtures.
- Observable prompt version, model settings, latency, token usage, and tool calls.

### Compatibility

- Existing CLI commands, deterministic JSON routes, and CSV schemas remain compatible.
- Missing or failed model configuration does not prevent deterministic workflows.

## [v0.1.0] - 2026-08-06

### Added

- Standalone Go web-server binary and embedded same-origin browser workbench.
- Versioned JSON routes for health, market lookup, calculations, reverse calculations, and history.
- Server-side CSV record persistence with existing schema compatibility.
- API validation, stable error envelopes, request limits, method checks, and security headers.
- Responsive lookup, scenario calculation, result, and history interfaces.

### Compatibility

- The existing CLI and historical CSV files remain compatible.
- The milestone is local, single-user, and does not require a model provider.

[v0.3.0]: https://github.com/NoireOffical/steam-profit-calculator/releases/tag/v0.3.0
[v0.4.0]: https://github.com/NoireOffical/steam-profit-calculator/releases/tag/v0.4.0
[v0.5.0]: https://github.com/NoireOffical/steam-profit-calculator/releases/tag/v0.5.0
[v0.6.0]: https://github.com/NoireOffical/steam-profit-calculator/compare/v0.5.0...main
[v0.2.0]: https://github.com/NoireOffical/steam-profit-calculator/releases/tag/v0.2.0
[v0.1.0]: https://github.com/NoireOffical/steam-profit-calculator/releases/tag/v0.1.0
