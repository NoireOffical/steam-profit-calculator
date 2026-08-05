## 1. Module Setup

- [x] 1.1 Initialize the Go module, project layout, and main entrypoint for the Steam profit CLI
- [x] 1.2 Define shared domain types, constants, and deterministic numeric formatting helpers for calculation results

## 2. Calculation Logic

- [x] 2.1 Implement validation for item name, buy price, and repeated sell prices
- [x] 2.2 Implement result generation for one buy price against one or more sell prices using the required profit and knife-ratio formulas
- [x] 2.3 Add table-driven unit tests for successful calculations and validation failures

## 3. CSV Record Storage

- [x] 3.1 Implement CSV store initialization with a stable header and append-only writes
- [x] 3.2 Implement CSV record loading for the list operation and explicit errors for malformed rows
- [x] 3.3 Add storage unit tests covering first-write creation, append behavior, readback, and malformed data

## 4. CLI Commands

- [x] 4.1 Implement the `add` command to parse flags, invoke calculation logic, and persist one CSV row per sell price
- [x] 4.2 Implement the `list` command to read saved records and print them in a stable, testable format
- [x] 4.3 Add CLI tests covering successful add/list flows and invalid input failures

## 5. Verification

- [x] 5.1 Run `go test ./...` and fix any remaining failures
- [x] 5.2 Manually smoke-test the CLI against a temporary CSV file to confirm end-to-end behavior
