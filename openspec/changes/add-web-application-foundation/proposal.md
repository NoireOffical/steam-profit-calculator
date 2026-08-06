## Why

The project currently exposes its calculation, lookup, and record workflows only through a local CLI. That is a useful deterministic core, but it does not provide the browser-to-server boundary needed for a full-stack application or a stable tool surface for a later AI assistant.

## What Changes

- Add a standalone HTTP server binary without changing the existing CLI contract.
- Expose health, item lookup, forward calculation, reverse calculation, and record-history workflows through versioned JSON endpoints.
- Add a same-origin browser interface for searching items, calculating profit scenarios, and reviewing persisted history.
- Reuse the existing calculation, lookup, domain, and CSV storage behavior rather than duplicating business rules.
- Define stable JSON success and error envelopes that can later be consumed by an AI tool-calling layer.

## Capabilities

### New Capabilities

- `web-application-interface`: Serve a browser application and versioned HTTP API over the existing deterministic domain workflows.

### Modified Capabilities

- None. Existing CLI and storage contracts remain compatible.

## Impact

- New `cmd/steam-profit-server` entrypoint.
- New `internal/webapi` package and embedded browser assets.
- New handler-level tests for JSON contracts, persistence behavior, and static application delivery.
- New development commands and documentation for running the web application.
