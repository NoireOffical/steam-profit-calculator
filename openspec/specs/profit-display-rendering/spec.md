## Purpose

Define terminal-oriented display rendering for profit records without changing CSV persistence behavior.

## Requirements

### Requirement: Render profit records in a pretty table view
The system SHALL provide a terminal display renderer that formats profit records as an aligned plain-text table. The table MUST include a header row and MUST preserve the canonical field order `item_name, buy_price, sell_price, quantity, net_profit, total_profit, knife_ratio`.

#### Scenario: Pretty table shows aligned rows
- **WHEN** the user displays one or more profit records in pretty mode
- **THEN** the system prints a header row
- **THEN** the system prints one row per record in the canonical field order
- **THEN** columns are aligned consistently within the rendered result set

#### Scenario: Numeric fields remain readable
- **WHEN** the system renders records in pretty mode
- **THEN** numeric fields are formatted using the same underlying values as storage
- **THEN** numeric columns are visually aligned for scanning

### Requirement: Support CSV-style display as a separate renderer
The system SHALL support a CSV-style display renderer for stdout that is independent from storage persistence. The CSV-style renderer MUST preserve the canonical field order and MUST not change on-disk CSV formatting rules.

#### Scenario: CSV display mode prints canonical rows
- **WHEN** the user selects CSV display mode
- **THEN** the system prints the canonical header row
- **THEN** the system prints one row per record in the canonical field order
- **THEN** the system does not require any storage schema change to support this mode
