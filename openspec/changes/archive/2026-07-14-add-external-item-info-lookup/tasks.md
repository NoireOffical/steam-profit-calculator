## 1. Lookup Domain And Provider Boundary

- [x] 1.1 Introduce a normalized item-summary model and lookup request/result types that are separate from `domain.ProfitRecord`
- [x] 1.2 Define a provider-facing lookup client interface plus configuration loading for endpoint, credentials, and timeout settings
- [x] 1.3 Implement a first provider-backed lookup client that maps upstream responses into normalized item summaries and rejects invalid payloads

## 2. Lookup Workflow And Rendering

- [x] 2.1 Add a lookup service or workflow layer that executes provider queries, applies optional result limits, and returns explicit no-match or provider errors
- [x] 2.2 Implement lookup-specific table and CSV renderers with deterministic field order for normalized item summaries
- [x] 2.3 Keep lookup output ephemeral by ensuring the workflow never appends to or mutates the existing profit-record CSV store

## 3. CLI Integration

- [x] 3.1 Add a `lookup` command that requires `--query` and accepts optional `--limit` and `--output table|csv`
- [x] 3.2 Wire the `lookup` command into the new lookup workflow and render successful results with the selected output mode
- [x] 3.3 Surface clear CLI errors for missing query input, no matching items, provider failures, and invalid provider responses

## 4. Test Coverage

- [x] 4.1 Add unit tests for provider response normalization, result limiting, no-match handling, provider failures, and invalid payload failures
- [x] 4.2 Add renderer-focused tests for lookup table output, CSV output, and stable field ordering
- [x] 4.3 Add CLI integration tests covering successful lookup output, missing `--query`, no-match failures, and provider failure behavior without record-file writes

## 5. Documentation And Verification

- [x] 5.1 Update `CLI_USAGE.md` to document the `lookup` command, supported flags, output modes, configuration expectations, and representative failure cases
- [x] 5.2 Run `go test ./...` and perform manual smoke tests for the `lookup` command in both table and CSV output modes
