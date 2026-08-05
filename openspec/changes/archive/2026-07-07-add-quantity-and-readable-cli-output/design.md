## Context

The current Steam profit CLI calculates and stores per-item profit records, but it does not support batch quantity or total profit output. It also prints rows in compact CSV format with no spaces, which is harder to scan in terminal output. The repository already contains legacy CSV files with a five-column schema, so this change must improve the data model without making those files unreadable.

## Goals / Non-Goals

**Goals:**
- Add quantity-aware calculation with a derived total profit field.
- Keep existing CLI command shapes stable by making quantity optional with a default of `1`.
- Improve terminal readability by rendering displayed rows with comma-and-space separators.
- Expand persisted records to include quantity and total profit.
- Preserve usability of existing five-column CSV files through backward-compatible reading and append-time migration.

**Non-Goals:**
- Changing the Steam fee multiplier.
- Adding non-integer quantity support.
- Replacing CSV storage with a different format.
- Redesigning the CLI into subcommands beyond the existing `add` and `list`.

## Decisions

### 1. Quantity defaults to `1`

The CLI and calculation layer will treat omitted quantity as `1`. This keeps current command usage working while allowing users to opt into batch calculations only when needed.

Alternatives considered:
- Requiring `--quantity` for every `add`: makes the new field explicit, but unnecessarily breaks existing usage.

### 2. `net_profit` remains per-item and `total_profit` is derived

The existing `net_profit` field will continue to represent single-item profit. A new `total_profit` field will be added and always computed as `net_profit * quantity`.

Alternatives considered:
- Renaming `net_profit` to `unit_profit`: clearer in isolation, but it creates avoidable churn in code, CSV fields, and existing user expectations.

### 3. Storage schema is expanded, but legacy CSV files remain supported

New records will use the expanded header:
`item_name,buy_price,sell_price,quantity,net_profit,total_profit,knife_ratio`

When reading legacy five-column files, the system will treat missing quantity as `1` and missing total profit as the same value as `net_profit`. When appending to a legacy file, the system will rewrite it to the new schema before appending new rows.

Alternatives considered:
- Failing on legacy files: simpler implementation, but it breaks existing local data.
- Keeping mixed-width rows in one file: avoids rewrite logic, but produces an unstable file format.

### 4. CLI display format diverges from persisted CSV formatting

Stored files will remain canonical CSV with no added presentation spacing. Terminal output will render the same field order with `, ` separators for readability.

Alternatives considered:
- Adding spaces to stored CSV too: more human-readable, but it changes on-disk formatting unnecessarily.

## Risks / Trade-offs

- [Legacy migration can corrupt data if rewrite logic is wrong] -> Cover migration behavior with dedicated storage tests before changing append behavior.
- [Readable CLI formatting may drift from stored field order] -> Reuse the same header and row order from shared domain helpers, varying only the separator.
- [Default quantity may hide user mistakes] -> Keep validation strict for non-positive values while preserving omission as a deliberate compatibility path.
