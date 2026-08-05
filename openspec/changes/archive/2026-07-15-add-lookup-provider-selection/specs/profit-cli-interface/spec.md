## MODIFIED Requirements

### Requirement: CLI provides a lookup command for external item information
The CLI SHALL provide a `lookup` command for querying external item metadata before users run profit calculations. The command MUST require `--query`, MUST accept optional `--limit`, MUST accept optional `--output table|csv`, and MUST not append or modify any local profit-record storage file.

Successful lookup output MUST use the normalized field order `item_id,canonical_name,app_name,query_time,sale_price_text,sell_price_text,sell_listings`.
Lookup provider selection MUST be controlled through configuration and MUST NOT add provider-specific CLI flags to the `lookup` command.

#### Scenario: Provider switch does not change lookup flags
- **WHEN** the runtime configuration changes the selected lookup provider
- **THEN** the user continues to run `lookup` with the same `--query`, `--limit`, and `--output` flags
- **THEN** the CLI does not require a `--provider` flag

#### Scenario: Lookup surfaces unsupported provider configuration
- **WHEN** the runtime configuration selects an unsupported lookup provider
- **THEN** the CLI returns a lookup configuration error
- **THEN** the CLI does not execute the external lookup workflow
