## ADDED Requirements

### Requirement: Web records have durable persistence metadata

The system SHALL persist successful web calculation records with a durable positive ID and a UTC creation timestamp. A batch created by one calculation request MUST be committed atomically, and persistence MUST NOT recalculate domain values.

#### Scenario: Calculation batch is stored successfully

- **WHEN** the web application appends a valid batch of calculated records
- **THEN** every returned record has a positive durable ID and UTC creation timestamp
- **AND** the stored calculation values equal the values supplied by the deterministic calculation package

#### Scenario: One insert in a batch fails

- **WHEN** the repository cannot persist the complete batch
- **THEN** none of the batch records remain committed
- **AND** the caller receives a persistence error

### Requirement: SQLite schema changes are versioned

The default web repository SHALL apply ordered schema migrations transactionally and record each applied version. Existing migration versions MUST NOT be edited after release.

#### Scenario: New database opens for the first time

- **WHEN** the SQLite repository opens an empty database
- **THEN** it creates the migration ledger, profit record schema, and query indexes
- **AND** records the applied migration version before serving requests

#### Scenario: Migration fails

- **WHEN** an unapplied database migration cannot complete
- **THEN** that migration leaves no partial schema changes
- **AND** server startup fails before accepting requests

### Requirement: History queries are bounded and stable

The repository SHALL return filtered record pages with a total match count, a bounded page size, an offset, and stable ordering by creation time plus ID.

#### Scenario: History is filtered and paginated

- **WHEN** a caller supplies a valid item prefix, outcome, order, limit, and offset
- **THEN** the repository returns only matching records in the requested stable order
- **AND** returns the total match count independently of the page size

### Requirement: Historical CSV data can be imported safely

The system SHALL provide an explicit CSV-to-SQLite import that accepts all currently supported CSV schemas, does not modify the source file, and skips rows already imported from the same source identity.

#### Scenario: Same CSV is imported repeatedly

- **WHEN** an unchanged CSV file is imported more than once
- **THEN** the first import inserts its records
- **AND** later imports report those rows as skipped without creating duplicates

#### Scenario: CSV input is invalid

- **WHEN** the source CSV cannot be completely parsed using the compatible CSV reader
- **THEN** the import returns an error
- **AND** inserts no records

### Requirement: CSV compatibility remains available

The existing CLI CSV read/write behavior SHALL remain unchanged, and the web server SHALL provide an explicit CSV repository mode for operational rollback.

#### Scenario: Operator selects CSV rollback mode

- **WHEN** the server starts with CSV storage selected and a compatible record path
- **THEN** successful web calculations append to that CSV
- **AND** history remains available through the current record API contract
