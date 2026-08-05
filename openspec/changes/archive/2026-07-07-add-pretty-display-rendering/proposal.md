## Why

The current CLI mixes human-readable display concerns with CSV-oriented record formatting, which makes future output beautification harder to evolve without disturbing storage behavior. The next change should explicitly separate display rendering from CSV persistence so terminal output can improve while on-disk records remain stable.

## What Changes

- Add a dedicated display-rendering capability for terminal output.
- Keep CSV storage unchanged as the canonical persistence format.
- Introduce explicit CLI output modes so human-facing display and CSV-style display are separate concerns.
- Define a default pretty table view for terminal usage.
- Keep existing record data and field order intact while allowing alternate renderers to reuse the same record model.

## Capabilities

### New Capabilities
- `profit-display-rendering`: Render profit records in terminal-friendly display modes that are independent from CSV persistence.

### Modified Capabilities
- `profit-cli-interface`: Add output-mode selection and make the default CLI presentation use the display-rendering capability instead of raw CSV-style rows.

## Impact

- New rendering layer or package for terminal formatting.
- CLI command changes for selecting display modes.
- No storage schema change and no migration requirement for CSV files.
- Updated CLI tests and usage documentation.
