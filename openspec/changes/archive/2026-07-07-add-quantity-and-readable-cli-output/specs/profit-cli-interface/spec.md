## ADDED Requirements

### Requirement: Add command accepts quantity-aware input
The CLI SHALL accept an optional `--quantity` flag on the `add` command. If the flag is omitted, the CLI MUST pass quantity `1` to the calculation layer.

#### Scenario: User provides explicit quantity
- **WHEN** the user runs `add` with `--name "Capsule" --buy 0.48 --quantity 5 --sell 0.8`
- **THEN** the CLI sends quantity `5` to the calculation workflow

#### Scenario: User omits quantity
- **WHEN** the user runs `add` with `--name "Capsule" --buy 0.48 --sell 0.8`
- **THEN** the CLI sends quantity `1` to the calculation workflow

### Requirement: CLI displays readable record rows
The CLI SHALL display record output using the canonical field order `item_name, buy_price, sell_price, quantity, net_profit, total_profit, knife_ratio` with a comma-and-space separator between displayed fields.

#### Scenario: Add command prints readable rows
- **WHEN** the user successfully runs `add`
- **THEN** the CLI prints a header row with comma-and-space separators
- **THEN** the CLI prints each calculated result row with the same separator style

#### Scenario: List command prints readable rows
- **WHEN** the user successfully runs `list`
- **THEN** the CLI prints a header row with comma-and-space separators
- **THEN** the CLI prints each loaded result row with the same separator style
