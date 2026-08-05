## 1. Configurable Multiplier In Core Calculation

- [x] 1.1 Extend shared domain request types and defaults so forward calculations can carry an optional multiplier while preserving the `0.837` fallback
- [x] 1.2 Update forward-calculation validation and formulas to use the effective multiplier and reject invalid multiplier values
- [x] 1.3 Add or update calculation tests for default multiplier behavior, explicit multiplier overrides, and invalid multiplier input

## 2. Reverse Calculation Workflow

- [x] 2.1 Introduce reverse-calculation request handling that solves either buy price or sell price from exactly one target metric
- [x] 2.2 Reuse the canonical `ProfitRecord` output path so reverse results can be rendered and persisted like forward results
- [x] 2.3 Add reverse-calculation tests covering target net profit, target total profit, target knife ratio, and invalid input combinations

## 3. Batch Import Parsing

- [x] 3.1 Implement a CSV scenario importer for the `item_name,buy_price,sell_price,quantity,multiplier` input format with row defaults for empty optional cells
- [x] 3.2 Make batch import atomic by validating the entire file before appending any imported results to the record store
- [x] 3.3 Add importer-focused tests for multi-row success, defaulted cells, malformed rows, and unsupported headers

## 4. CLI Integration

- [x] 4.1 Extend the `add` command with `--multiplier` while keeping existing usage backward compatible
- [x] 4.2 Add the `reverse` command and wire it into calculation, storage, and display flows
- [x] 4.3 Add the `import` command and wire it into scenario parsing, storage, and display flows
- [x] 4.4 Update CLI integration tests for multiplier-aware add, reverse command behavior, import command behavior, and no-partial-write failures

## 5. Documentation And Verification

- [x] 5.1 Update `CLI_USAGE.md` to document multiplier overrides, reverse workflow examples, import file format, and error cases
- [x] 5.2 Run `go test ./...` and perform manual smoke tests for `add`, `reverse`, `import`, and `list`
