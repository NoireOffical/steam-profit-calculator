## ADDED Requirements

### Requirement: CLI provides a lookup command for external item information
The CLI SHALL provide a `lookup` command for querying external item metadata before users run profit calculations. The command MUST require `--query`, MUST accept optional `--limit`, MUST accept optional `--output table|csv`, and MUST not append or modify any local profit-record storage file.

Successful lookup output MUST use the normalized field order `item_id,canonical_name,app_name,sale_price_text,sell_price_text,sell_listings`.
The lookup client configuration MUST support an overridable `User-Agent` value so requests can identify the calling application without changing the CLI contract.

#### Scenario: Lookup command defaults to table output
- **WHEN** the user runs `lookup` with a valid `--query` and does not specify an output mode
- **THEN** the CLI prints normalized item summaries using the table renderer
- **THEN** the CLI does not write any profit record

#### Scenario: Lookup command supports CSV output
- **WHEN** the user runs `lookup` with a valid `--query` and `--output csv`
- **THEN** the CLI prints normalized item summaries using the CSV-style renderer
- **THEN** the CLI does not write any profit record

#### Scenario: Lookup command rejects missing query input
- **WHEN** the user runs `lookup` without providing `--query`
- **THEN** the CLI returns a usage or validation error
- **THEN** the CLI does not call the external lookup workflow

#### Scenario: Lookup command surfaces no-match failures
- **WHEN** the user runs `lookup` with a query that returns no matching items
- **THEN** the CLI returns a lookup error indicating that no items matched the query
- **THEN** the CLI does not write any profit record

#### Scenario: Lookup command surfaces provider failures
- **WHEN** the user runs `lookup` and the external provider request fails
- **THEN** the CLI returns a lookup error describing the provider failure
- **THEN** the CLI does not write any profit record
