## 1. Lookup Schema

- [x] 1.1 Add `query_time` to the normalized lookup item model and field order
- [x] 1.2 Stamp one shared timestamp per lookup response and map it into all returned items

## 2. Output And Docs

- [x] 2.1 Update lookup table and CSV rendering expectations to include `query_time`
- [x] 2.2 Update `CLI_USAGE.md` and change specs to document the new lookup timestamp field

## 3. Verification

- [x] 3.1 Update lookup-focused unit and CLI tests for the new field and shared timestamp behavior
- [x] 3.2 Run `go test ./...` and validate the change with `openspec validate`
