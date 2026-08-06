## 1. Repository Boundary And SQLite

- [x] 1.1 Add context-aware stored-record, query, page, and repository types without changing the domain calculation model
- [x] 1.2 Add versioned SQLite initialization, canonical value encoding, atomic append, filtered query, and lifecycle optimization
- [x] 1.3 Add indexes for observed ordering and item-prefix query patterns and verify query plans in tests

## 2. Compatibility And Migration

- [x] 2.1 Keep all existing CLI CSV behavior and tests unchanged
- [x] 2.2 Add a legacy CSV repository adapter for explicit web rollback mode
- [x] 2.3 Add an explicit atomic, idempotent CSV-to-SQLite importer with inserted/skipped reporting

## 3. HTTP And Browser Experience

- [x] 3.1 Return stored IDs/timestamps for calculations and add validated record filters, stable order, and pagination metadata
- [x] 3.2 Add history controls, record metadata, total/result state, and previous/next page navigation to the embedded browser
- [x] 3.3 Configure SQLite as the server default while documenting database, CSV rollback, and import options

## 4. Verification And Handoff

- [x] 4.1 Cover repository success, invalid query, dependency failure, transaction rollback, migration repeatability, and no-side-effect failures
- [x] 4.2 Cover HTTP contracts and browser JavaScript syntax; run `make check` and both binary builds
- [x] 4.3 Smoke-test SQLite creation, calculation persistence, filtered/paginated history, repeated CSV import, and CSV rollback mode
- [x] 4.4 Update the AI full-stack roadmap and prepare immutable `v0.3.0` pre-release notes for the completed milestone
