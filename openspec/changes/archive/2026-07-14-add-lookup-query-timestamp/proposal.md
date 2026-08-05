## Why

The `lookup` command currently returns live market price text without any timestamp, which makes cross-run price comparisons ambiguous. A lookup result should carry the time at which the price snapshot was observed so users can compare historical CLI outputs meaningfully.

## What Changes

- Add a `query_time` field to normalized lookup output.
- Stamp all items returned by one lookup request with the same observation time.
- Document the timestamp format and update CLI/spec expectations for lookup output.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `external-item-info-lookup`: Lookup results now include the time of the observed market snapshot.
- `profit-cli-interface`: The `lookup` command output schema now includes a `query_time` column.

## Impact

- Affected code: `internal/domain/item_lookup.go`, `internal/lookup/steam_client.go`, `internal/display/render.go`, lookup-related tests, and `CLI_USAGE.md`
- Affected specs: `openspec/specs/external-item-info-lookup/spec.md`, `openspec/specs/profit-cli-interface/spec.md`
