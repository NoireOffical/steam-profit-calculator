## Why

The main `profit-record-storage` spec still documents a seven-column CSV schema, but the implemented storage layer has already moved to an eight-column format with a persisted `fee` field. This mismatch weakens the repo's source of truth and makes future maintenance harder because the spec no longer accurately describes what is actually stored on disk.

## What Changes

- Update the `profit-record-storage` capability to document the current eight-column CSV header: `item_name,buy_price,sell_price,quantity,net_profit,total_profit,knife_ratio,fee`
- Specify that each persisted record includes a `fee` field in addition to the existing profit metrics
- Specify that records created with the default multiplier persist `fee` as `default`
- Specify that records created with an explicit multiplier persist the concrete multiplier value in the `fee` column
- Specify compatibility rules for existing stored files:
  - legacy five-column files remain readable
  - existing seven-column expanded files remain readable
  - append-time migration upgrades five-column and seven-column files to the eight-column schema before writing new rows

## Capabilities

### New Capabilities

### Modified Capabilities
- `profit-record-storage`: update the persisted record schema, fee-field semantics, and compatibility requirements to match the implemented eight-column storage behavior

## Impact

- Main spec content in `openspec/specs/profit-record-storage/spec.md`
- Change delta spec under `openspec/changes/sync-profit-record-storage-fee-column/specs/profit-record-storage/spec.md`
- Documentation and validation workflow for CSV storage behavior
- No intended runtime behavior change; this change synchronizes the spec with already-implemented storage logic
