## Why

The local web application still stores calculation history in a flat CSV file. That preserves CLI compatibility, but it cannot give web records durable identities and timestamps or support bounded filtering and pagination without loading the entire file. A versioned relational boundary is the next prerequisite for sessions, evaluation runs, and eventual user ownership.

## What Changes

- Add a repository interface for web calculation records with durable IDs, creation timestamps, stable ordering, filters, and pagination.
- Make a local SQLite database the default web-server persistence adapter while leaving the CLI CSV contract unchanged.
- Keep an explicit legacy CSV web mode and add an opt-in, repeatable CSV-to-SQLite import path.
- Extend the record-history API and browser interface with metadata, item/outcome filters, ordering, and page navigation.
- Version database schema migrations and reserve nullable ownership/session columns without adding authentication or conversation persistence.

## Capabilities

### New Capabilities

- `web-record-repository`: Persist and query local web calculation records through a versioned SQLite repository with a compatible CSV fallback and migration path.

### Modified Capabilities

- `web-application-interface`: Return durable record metadata and bounded history pages, and expose those controls in the browser.

## Impact

- New repository types and SQLite implementation in `internal/storage`.
- Updated `cmd/steam-profit-server` storage configuration and startup migration handling.
- Updated history API responses and embedded browser assets.
- New pure-Go SQLite dependency pinned to a Go 1.21-compatible version.
- Existing CLI flags, output, formulas, and CSV schemas remain compatible.
