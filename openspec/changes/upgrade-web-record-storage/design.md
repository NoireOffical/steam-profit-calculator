## Context

The deterministic calculation engine and CLI already treat CSV as a compatibility format. The web application needs a stronger persistence boundary: each successful calculation must become an addressable record, history queries must be bounded, and later AI/session features need stable foreign-key targets. The application remains local-only and single-process in this change.

## Goals / Non-Goals

**Goals:**
- Store web calculation records in a versioned SQLite schema by default.
- Return durable record IDs and UTC creation timestamps from successful writes.
- Query history with validated filters, stable sorting, totals, and bounded pages.
- Preserve the current CLI and all historical CSV reading/writing behavior.
- Provide an explicit CSV web mode and a repeatable, no-duplicate CSV import command path.
- Keep persistence failures atomic and observable at the HTTP boundary.

**Non-Goals:**
- Adding login, authorization, multi-user isolation, or cloud database hosting.
- Persisting AI conversations, prompts, or model responses in this milestone.
- Changing calculation formulas, rounding, lookup providers, or CLI output.
- Automatically deleting, renaming, or rewriting a source CSV after import.
- Deploying the application publicly.

## Decisions

### 1. Put the repository contract in the storage layer

The web API will depend on a context-aware record repository that appends a batch atomically and returns stored records, then queries a page using an explicit filter object. Stored metadata wraps the existing `domain.ProfitRecord`; calculation values remain owned by the calculation package.

### 2. Use SQLite as the default local web adapter

The server will default to a database file configured by `--db` or `STEAM_PROFIT_DATABASE_FILE`. It uses a CGo-free driver compatible with the repository's Go 1.21 declaration. Connections enable foreign keys, a bounded busy timeout, WAL journaling, and normal synchronous mode. A single open connection keeps connection-scoped settings predictable for the local process.

### 3. Version schema changes in transactions

A `schema_migrations` table records applied integer versions. Startup applies each missing migration transactionally before serving traffic. The initial `profit_records` table stores canonical scaled integers for monetary and ratio values, a UTC timestamp, a source label, and an optional unique source key. Nullable `owner_id` and `session_id` columns reserve additive ownership/session boundaries without implying current authorization semantics.

Indexes match current query patterns: stable creation-time ordering and case-insensitive item-name prefix filtering. Startup verifies the schema and runs `PRAGMA optimize` after migrations.

### 4. Keep web writes atomic and deterministic

Forward calculations can produce multiple scenarios. The repository inserts the full batch in one transaction and returns IDs/timestamps only after every insert succeeds. A failed write rolls back the batch and the API returns `storage_error`; the deterministic calculation API and CLI remain usable.

Domain decimal values are converted to scaled integers using their existing formatted precision before storage and converted back when read. The repository does not recalculate profit.

### 5. Make CSV migration explicit and idempotent

`--import-csv` (or its environment variable) imports a selected historical CSV before the HTTP listener starts. Each source row receives a stable key derived from the cleaned absolute source path, row position, and canonical row content. A uniqueness constraint makes repeating the same import safe and allows appending new rows to the same file without re-importing old rows.

The source CSV is read completely before the database transaction begins. Invalid input produces no database changes. Successful or failed imports never modify the CSV file.

### 6. Retain a legacy CSV server mode

`--storage=csv --file=...` preserves the previous web storage option for rollback and compatibility. Its repository adapter exposes the new query contract by reading the file and applying the same filters in memory. SQLite remains the default because durable IDs and timestamps cannot be represented in the historical CSV schema.

### 7. Bound history queries at the API

`GET /api/v1/records` accepts an item-name prefix, outcome (`all`, `profit`, or `loss`), order (`newest` or `oldest`), limit, and offset. Limit defaults to 20 and is capped at 100. Responses include records and pagination metadata. Sorting always includes ID as a tie-breaker.

## Failure Semantics And Security Boundary

- Invalid filter, order, limit, or offset values return `400 invalid_request` without querying persistence.
- Schema or database-open failures stop server startup before accepting requests.
- Append and import operations are transactionally all-or-nothing.
- SQLite files and CSV files stay server-side; the browser receives only API response fields.
- User-provided filters are bound SQL parameters, never interpolated into statements.
- No AI model is permitted to write, alter, or invent persisted calculation values.

## Migration And Rollback

1. Start the new server with its default SQLite database to create the schema.
2. If historical web CSV data is desired, pass `--import-csv <path>` once; repeated startup with the same option is safe.
3. Verify counts and sample records through the history interface.
4. Keep the original CSV as the rollback artifact.
5. Roll back operationally with `--storage=csv --file <path>`; no reverse schema conversion is required.

Database migrations and released tags are immutable. Corrections are added as new migration versions and a new release.

## Risks / Trade-offs

- [SQLite is still single-host persistence] -> Keep the server local-only and retain a repository interface for a future managed database adapter.
- [The Go 1.21-compatible driver is not the newest driver release] -> Pin the newest driver version whose module supports Go 1.21; treat a Go toolchain upgrade as a separate change.
- [Offset pagination becomes slower for very large histories] -> Bound page size now; introduce cursor pagination only when observed dataset size warrants it.
- [CSV fallback metadata is derived rather than stored] -> Document SQLite as the durable default and keep CSV mode for compatibility/rollback only.
