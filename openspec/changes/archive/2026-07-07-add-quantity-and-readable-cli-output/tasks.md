## 1. Domain And Formatting

- [x] 1.1 Extend shared request and record types with quantity and total profit fields
- [x] 1.2 Add display-format helpers that print record rows with comma-and-space separators while keeping storage field order stable

## 2. Calculation Logic

- [x] 2.1 Update calculation validation to support quantity with default `1` and reject non-positive values
- [x] 2.2 Update result generation so total profit is derived from per-item net profit and quantity
- [x] 2.3 Add or update calculation tests for quantity-aware results and validation behavior

## 3. Storage Compatibility

- [x] 3.1 Expand the CSV schema to persist quantity and total profit in new records
- [x] 3.2 Add backward-compatible reads and append-time migration for legacy five-column CSV files
- [x] 3.3 Add or update storage tests for expanded rows, legacy readback, and legacy migration

## 4. CLI Behavior

- [x] 4.1 Add the `--quantity` flag to the `add` command and pass it into the calculation layer
- [x] 4.2 Update `add` and `list` output to include quantity and total profit with readable comma-and-space separators
- [x] 4.3 Add or update CLI tests for quantity input, readable output, and legacy-file behavior

## 5. Verification And Docs

- [x] 5.1 Update the local CLI usage document to describe quantity and the new output columns
- [x] 5.2 Run `go test ./...` and execute a smoke test against both new and legacy CSV files
