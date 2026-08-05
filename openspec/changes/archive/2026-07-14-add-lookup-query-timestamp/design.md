## Context

`lookup` is a read-only workflow that surfaces live Steam market summary fields such as `sale_price_text` and `sell_price_text`. Those values are snapshots rather than durable facts, so the current output is missing critical context when users compare results captured at different times.

The existing lookup flow already normalizes provider responses into project-owned fields before rendering table or CSV output. That makes this a small cross-cutting schema change: the item summary model, provider adapter, renderers, tests, and documentation all need to agree on the new output column.

## Goals / Non-Goals

**Goals:**
- Add a deterministic `query_time` field to lookup output.
- Ensure every item produced by one lookup request shares the same timestamp.
- Keep the timestamp machine-friendly and easy to compare across runs.

**Non-Goals:**
- Persist lookup results to the profit record CSV.
- Introduce historical storage, caching, or time-series analysis.
- Add provider-specific raw timestamp fields to the CLI output.

## Decisions

### 1. Add `query_time` to the normalized lookup schema

The normalized lookup row should include a `query_time` column rather than printing the time outside the table/CSV payload. This keeps the timestamp attached to each row and preserves usefulness when users redirect lookup output to files.

Alternatives considered:
- Print a separate header line above the table: rejected because it would not fit CSV mode cleanly.
- Leave timing implicit in shell history: rejected because copied output would lose the observation time.

### 2. Use one shared timestamp per lookup request

All rows returned by one provider response should receive the same query timestamp. This matches the user's mental model of one market snapshot and avoids meaningless per-row drift caused by iterating through results.

Alternatives considered:
- Stamp each row independently with `time.Now()`: rejected because rows from one response would differ by tiny, unhelpful deltas.

### 3. Format the timestamp as UTC RFC3339

The output should use UTC `RFC3339` formatting, such as `2026-07-14T06:15:30Z`. That keeps it human-readable while remaining easy to compare and parse programmatically.

Alternatives considered:
- Local time: rejected because it varies across environments.
- Raw provider `Date` header text: rejected because RFC1123 text is less convenient for downstream comparisons.

### 4. Prefer the HTTP response time when available

If the provider response includes a valid `Date` header, the client should use it as the observation time. Otherwise it should fall back to the local current UTC time. This keeps tests deterministic and aligns the snapshot with the actual response boundary.

Alternatives considered:
- Always use local `time.Now()`: simpler, but harder to test deterministically without extra injection seams.

## Risks / Trade-offs

- [Provider omits or mangles the `Date` header] -> Fall back to local UTC time.
- [Lookup output schema changes] -> Update specs, docs, and renderer tests together.
- [Users may misread `query_time` as persisted history] -> Keep lookup explicitly non-mutating in docs and specs.
