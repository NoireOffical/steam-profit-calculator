## 1. Display Architecture

- [x] 1.1 Introduce a dedicated display-rendering layer that consumes `ProfitRecord` values without changing CSV storage behavior
- [x] 1.2 Define shared field extraction helpers so all renderers preserve canonical field order

## 2. Pretty Table Rendering

- [x] 2.1 Implement a plain-text pretty table renderer with aligned columns and stable headers
- [x] 2.2 Implement a CSV-style stdout renderer as a separate display mode
- [x] 2.3 Add renderer-focused tests for alignment, headers, and field ordering

## 3. CLI Integration

- [x] 3.1 Add explicit output-mode selection to `add` and `list`
- [x] 3.2 Make pretty table output the default renderer for terminal usage
- [x] 3.3 Update CLI tests to cover default pretty mode and explicit CSV mode

## 4. Documentation And Verification

- [x] 4.1 Update local CLI documentation to explain display modes and default behavior
- [x] 4.2 Run `go test ./...` and manually verify both pretty-table and CSV display modes
