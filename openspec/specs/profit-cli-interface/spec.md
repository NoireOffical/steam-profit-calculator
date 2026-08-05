## Purpose

Define the CLI-specific input and output behavior for forward, reverse, and batch-import Steam profit calculations.

## Requirements

### Requirement: Add command accepts quantity-aware input
The CLI SHALL accept an optional `--quantity` flag and an optional `--multiplier` flag on the `add` command. If `--quantity` is omitted, the CLI MUST pass quantity `1` to the calculation layer. If `--multiplier` is omitted, the CLI MUST pass multiplier `0.837` to the calculation layer.

#### Scenario: User provides explicit quantity and multiplier
- **WHEN** the user runs `add` with `--name "Capsule" --buy 0.48 --quantity 5 --multiplier 0.85 --sell 0.8`
- **THEN** the CLI sends quantity `5` to the calculation workflow
- **THEN** the CLI sends multiplier `0.85` to the calculation workflow

#### Scenario: User omits quantity and multiplier
- **WHEN** the user runs `add` with `--name "Capsule" --buy 0.48 --sell 0.8`
- **THEN** the CLI sends quantity `1` to the calculation workflow
- **THEN** the CLI sends multiplier `0.837` to the calculation workflow

### Requirement: CLI displays readable record rows
The CLI SHALL support explicit output-mode selection for record display. The `add`, `reverse`, `import`, and `list` commands MUST support a display selector, and the default display mode MUST use the pretty table renderer. The CLI MUST still be able to emit CSV-style stdout output when the CSV display mode is explicitly selected.

#### Scenario: Add command defaults to pretty table output
- **WHEN** the user successfully runs `add` without specifying an output mode
- **THEN** the CLI prints records using the pretty table renderer

#### Scenario: Reverse command defaults to pretty table output
- **WHEN** the user successfully runs `reverse` without specifying an output mode
- **THEN** the CLI prints records using the pretty table renderer

#### Scenario: List command defaults to pretty table output
- **WHEN** the user successfully runs `list` without specifying an output mode
- **THEN** the CLI prints records using the pretty table renderer

#### Scenario: User selects CSV display mode
- **WHEN** the user runs `add`, `reverse`, `import`, or `list` with the CSV display mode selected
- **THEN** the CLI prints records using the CSV-style display renderer
- **THEN** the CLI does not change how records are stored on disk

### Requirement: Reverse command accepts target-driven pricing input
The CLI SHALL provide a `reverse` command for target-driven pricing analysis. The command MUST require `--solve buy|sell`, MUST require exactly one target flag from `--target-net-profit`, `--target-total-profit`, or `--target-knife-ratio`, MUST require the opposite known price (`--buy` when solving sell, `--sell` when solving buy), and MUST accept optional `--quantity`, optional `--multiplier`, and optional `--file` arguments.

#### Scenario: User solves required sell price from a total-profit target
- **WHEN** the user runs `reverse` with `--name "Capsule" --solve sell --buy 0.48 --quantity 5 --target-total-profit 1.0`
- **THEN** the CLI sends a reverse-calculation request that solves for sell price
- **THEN** the CLI persists the solved record to the configured record file

#### Scenario: User solves required sell price from a knife-ratio target
- **WHEN** the user runs `reverse` with `--name "Capsule" --solve sell --buy 0.48 --target-knife-ratio 0.7`
- **THEN** the CLI sends a reverse-calculation request that solves for sell price
- **THEN** the CLI prints the solved record using the selected output mode

#### Scenario: User solves acceptable buy price from a knife-ratio target
- **WHEN** the user runs `reverse` with `--name "Capsule" --solve buy --sell 0.8 --target-knife-ratio 0.7`
- **THEN** the CLI sends a reverse-calculation request that solves for buy price
- **THEN** the CLI prints the solved record using the selected output mode

#### Scenario: Reverse command rejects conflicting targets
- **WHEN** the user runs `reverse` with both `--target-net-profit` and `--target-knife-ratio`
- **THEN** the CLI returns a usage or validation error
- **THEN** the CLI does not write a record

### Requirement: Import command processes batch scenario files
The CLI SHALL provide an `import` command that reads forward-calculation scenarios from a CSV input file. The command MUST require `--input`, MUST accept optional `--file` and `--output` arguments, and MUST display the imported results after they are persisted.

#### Scenario: Import command processes a valid scenario file
- **WHEN** the user runs `import` with a valid input CSV file containing multiple rows
- **THEN** the CLI calculates one result per input row
- **THEN** the CLI appends the imported results to the configured record file
- **THEN** the CLI prints the imported results using the selected output mode

#### Scenario: Import command rejects an invalid scenario file
- **WHEN** the user runs `import` with an input CSV file containing an invalid row
- **THEN** the CLI returns an import error
- **THEN** the CLI does not append partial imported results to the record file

### Requirement: CLI provides a lookup command for external item information
The CLI SHALL provide a `lookup` command for querying external item metadata before users run profit calculations. The command MUST require `--query`, MUST accept optional `--limit`, MUST accept optional `--output table|csv`, and MUST not append or modify any local profit-record storage file.

Successful lookup output MUST use the normalized field order `item_id,canonical_name,app_name,query_time,sale_price_text,sell_price_text,sell_listings`.
The lookup client configuration MUST support an overridable `User-Agent` value so requests can identify the calling application without changing the CLI contract.
Lookup provider selection MUST be controlled through configuration and MUST NOT add provider-specific CLI flags to the `lookup` command.

#### Scenario: Lookup command defaults to table output
- **WHEN** the user runs `lookup` with a valid `--query` and does not specify an output mode
- **THEN** the CLI prints normalized item summaries using the table renderer
- **THEN** the CLI does not write any profit record

#### Scenario: Lookup command supports CSV output
- **WHEN** the user runs `lookup` with a valid `--query` and `--output csv`
- **THEN** the CLI prints normalized item summaries using the CSV-style renderer
- **THEN** the CLI does not write any profit record

#### Scenario: Lookup output includes the observation time
- **WHEN** the user runs `lookup` successfully
- **THEN** each lookup row includes a `query_time` column
- **THEN** the value records when the market price snapshot was observed

#### Scenario: Provider switch does not change lookup flags
- **WHEN** the runtime configuration changes the selected lookup provider
- **THEN** the user continues to run `lookup` with the same `--query`, `--limit`, and `--output` flags
- **THEN** the CLI does not require a `--provider` flag

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

#### Scenario: Lookup surfaces unsupported provider configuration
- **WHEN** the runtime configuration selects an unsupported lookup provider
- **THEN** the CLI returns a lookup configuration error
- **THEN** the CLI does not execute the external lookup workflow
