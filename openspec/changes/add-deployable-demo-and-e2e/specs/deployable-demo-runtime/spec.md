## ADDED Requirements

### Requirement: Demo data is deterministic and repeatable

The repository SHALL contain a versioned compatible seed dataset whose numerical values were produced by the deterministic calculation engine. Demo startup SHALL import the dataset through the existing transactional SQLite CSV importer, and repeating startup against the same database SHALL NOT duplicate seed records.

#### Scenario: Reviewer starts the demo twice

- **WHEN** a reviewer starts the demo twice against the same SQLite volume
- **THEN** the second import reports all existing seed rows as skipped
- **AND** the persisted history count does not increase

### Requirement: Browser workflows have an isolated E2E gate

The project SHALL provide a headless real-browser suite that starts the application with a fresh temporary SQLite database and no external provider credentials. The suite SHALL cover calculation persistence, history filtering and pagination, client validation errors, and disabled-assistant fallback.

#### Scenario: Maintainer runs browser E2E

- **WHEN** the E2E command runs on a supported Chromium environment
- **THEN** the browser completes the deterministic workflows against the real HTTP server
- **AND** the command returns non-zero if a visible state or persistence contract regresses

### Requirement: Container startup preserves local data

The project SHALL provide a single-process container configuration that runs as a non-root user, listens on all container interfaces, stores SQLite files in a named volume, imports demo data idempotently, exposes a health check, and allows graceful shutdown.

#### Scenario: Container is recreated

- **WHEN** the application container is stopped and recreated while retaining its named data volume
- **THEN** previously saved history remains available
- **AND** startup does not duplicate bundled seed rows

### Requirement: Demo workflow is documented

The project SHALL provide a concise walkthrough that identifies startup, expected visible states, health and metric checks, persistence verification, backup, restore, and rollback boundaries.

#### Scenario: Reviewer follows the walkthrough

- **WHEN** a reviewer follows the documented native or container path
- **THEN** the reviewer can demonstrate the browser-to-calculation-to-SQLite workflow in three to five minutes
