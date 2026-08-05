## Context

The repository's storage implementation already persists an eight-column CSV schema that adds a `fee` field to each saved record. That field is derived from multiplier usage:

- `default` when the default multiplier is used
- a formatted multiplier value such as `0.850` when the multiplier is explicitly provided

The main `profit-record-storage` spec has not yet caught up. It still describes the older seven-column expanded schema and only mentions compatibility with the original five-column legacy format. This leaves an important gap between the code-level source of truth and the spec-level source of truth.

The current implementation also recognizes three on-disk shapes:

1. legacy five-column records
2. prior seven-column expanded records without `fee`
3. current eight-column records with `fee`

This change is therefore primarily a spec-synchronization change, but it still benefits from a design document because the compatibility story needs to be stated clearly and consistently.

## Goals / Non-Goals

**Goals:**
- Update `profit-record-storage` so it accurately documents the implemented eight-column storage header
- Document the persisted meaning of the `fee` column
- Document compatibility and append-time migration behavior for both five-column and seven-column historical files
- Keep the spec aligned with actual code paths in `internal/domain` and `internal/storage`

**Non-Goals:**
- Changing runtime storage behavior again
- Introducing a new persisted field beyond `fee`
- Changing terminal display output to include `fee`
- Reworking any calculation formulas or CLI flag semantics

## Decisions

### 1. Treat this as a modified storage capability, not a new capability

The correct spec-level framing is a modification to `profit-record-storage`, because the system is still describing the same storage responsibility, just with an evolved persisted row shape.

Alternatives considered:
- Create a new storage capability just for fee persistence: rejected because it would split one storage contract across multiple specs.

### 2. Document `fee` as a persisted storage concern, not a display concern

The spec update will state that `fee` is part of the stored CSV row. It will not imply that `fee` must appear in terminal table or CSV stdout output, because the current architecture deliberately separates storage schema from display schema.

Alternatives considered:
- Mirror `fee` into display requirements: rejected because it would inaccurately expand the scope of the current user request and the implemented behavior.

### 3. Explicitly document both historical compatibility paths

The updated spec will distinguish:

- five-column legacy compatibility
- seven-column expanded-without-fee compatibility
- append-time migration from both historical forms into the current eight-column form

This is more precise than a generic "legacy compatibility" statement and better matches what the storage layer actually does today.

Alternatives considered:
- Mention only five-column legacy files and ignore seven-column history: rejected because the repository already contains the intermediate expanded schema in its change history and code path.

### 4. Define `fee` value semantics in terms of default vs explicit multiplier persistence

The spec will state that default multiplier usage persists `fee` as `default`, while explicit multiplier usage persists the concrete multiplier value. For seven-column historical rows that lack `fee`, the compatibility requirement will state that the system derives a stored fee label during migration before rewriting into the eight-column schema.

Alternatives considered:
- Specify only the current-header shape without value semantics: rejected because it would leave the new column under-defined.

## Risks / Trade-offs

- [Spec wording may accidentally promise display behavior] -> Keep the updated requirements scoped to persisted storage and listing behavior only
- [Compatibility wording may overfit implementation details] -> Describe the supported file shapes and outcomes, not line-by-line parser internals
- [Future fee-label formatting changes could require another spec update] -> Keep the semantics focused on `default` vs explicit multiplier value, which is the stable contract users care about

## Migration Plan

1. Create a delta spec for `profit-record-storage`
2. Replace the seven-column header description with the current eight-column header
3. Extend compatibility requirements to include seven-column files without `fee`
4. Add scenarios for default fee persistence, explicit multiplier persistence, and append-time migration into the eight-column schema
5. Validate the new change so it is ready for later apply / sync work

Rollback strategy:

- Because this is a spec-sync change, rollback is simply reverting the planning artifacts if the wording proves inaccurate
- No runtime rollback is needed because no behavior change is being introduced here

## Open Questions

- No blocking open questions. The current code already establishes the intended storage contract; this change is about documenting it accurately.
