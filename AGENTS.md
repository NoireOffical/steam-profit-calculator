# Repository guidance

## Product boundary

This repository contains a deterministic Steam market profit engine, a compatible CLI, and a local browser application. Treat calculated numbers as trusted domain output and market lookup values as provider snapshots, not financial advice.

## Architecture

- `cmd/steam-profit` owns the CLI process entrypoint.
- `cmd/steam-profit-server` owns the HTTP process entrypoint.
- `internal/calculation` is the only source of profit and reverse-calculation formulas.
- `internal/lookup` owns provider configuration, transport, retries, and normalization.
- `internal/storage` owns persisted record compatibility.
- `internal/webapi` translates HTTP requests and responses; it must not duplicate formulas.
- `openspec` records requirements and change history.

## Required workflow

1. Create or update an OpenSpec change for user-visible behavior.
2. Preserve existing CLI and CSV compatibility unless the change explicitly includes migration.
3. Use interfaces at provider, persistence, and future model boundaries.
4. Add tests for success, invalid input, dependency failure, and no-side-effect failure behavior.
5. Run `make check`, validate browser JavaScript syntax, and smoke-test affected user workflows.

## AI-specific invariants

- A model may interpret intent, choose tools, compare scenarios, and explain results.
- A model must not invent market data or calculate profit independently.
- AI responses must reference values returned by lookup or calculation tools.
- Prompts, model settings, tool schemas, evaluation cases, latency, token use, and failures must be versioned or observable.
- Model unavailability must not break deterministic CLI or calculation API workflows.
