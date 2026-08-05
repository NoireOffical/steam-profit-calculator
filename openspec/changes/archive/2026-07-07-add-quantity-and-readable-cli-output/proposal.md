## Why

The current CLI only supports per-item profit output and prints rows in compact CSV form, which makes multi-item analysis harder to read and prevents users from directly estimating batch收益. The next change needs to add quantity-aware profit calculation while improving CLI readability without breaking existing saved data.

## What Changes

- Add a quantity input to the `add` command and use it when generating calculation results.
- Add a total profit output field computed as per-item net profit multiplied by quantity.
- Make CLI output rows easier to scan by printing fields with a comma-and-space separator.
- Expand persisted records so quantity and total profit are saved and later displayed.
- Preserve compatibility with existing CSV files by supporting legacy records that do not yet contain quantity and total profit.

## Capabilities

### New Capabilities
- `profit-cli-interface`: Define CLI-specific input and output behavior for readable record display and quantity-aware command usage.

### Modified Capabilities
- `steam-profit-calculation`: Calculation results now include quantity-aware total profit and quantity validation/defaulting behavior.
- `profit-record-storage`: Stored records now include quantity and total profit, and existing legacy CSV files must remain readable and upgradable.

## Impact

- Changes to shared domain types and output formatting helpers.
- Changes to calculation validation and result shaping.
- Changes to CSV read/write logic, including legacy schema compatibility.
- Changes to CLI flags, displayed headers/rows, and usage documentation.
- Test updates across calculation, storage, and CLI packages.
