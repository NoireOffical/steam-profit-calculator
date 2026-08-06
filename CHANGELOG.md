# Changelog

All notable demonstrable milestones are recorded here. Until the public-operation requirements are complete, each version is published as a GitHub Pre-release with an immutable annotated tag.

## [v0.3.0] - Unreleased

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

[v0.2.0]: https://github.com/NoireOffical/steam-profit-calculator/releases/tag/v0.2.0
[v0.1.0]: https://github.com/NoireOffical/steam-profit-calculator/releases/tag/v0.1.0
