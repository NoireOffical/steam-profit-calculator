## Context

The current CLI supports forward profit calculation through `add` and `list`, persists normalized result rows to CSV, and renders results through a dedicated display layer. That foundation is already good for one-off analysis, but three practical gaps remain:

1. the calculation model still assumes a fixed Steam net multiplier of `0.837`
2. users cannot ask target-driven questions such as "what sell price reaches this profit?"
3. users must enter scenarios manually through repeated flags instead of importing a batch file

This change intentionally bundles those gaps into one higher-level workflow upgrade. The design therefore needs to extend the calculation model and CLI surface without undoing the current separation between calculation, storage, and terminal rendering.

Key constraints:

- Existing `add` and `list` usage must remain valid when no new flags are provided.
- Persisted output records should stay in the current expanded CSV schema so old data and current listing behavior remain stable.
- Batch import input should be separate from persisted output records; input files describe scenarios, while the record store continues to hold computed results.

## Goals / Non-Goals

**Goals:**
- Make the net multiplier configurable per request while keeping `0.837` as the default.
- Add a reverse-calculation workflow that can solve either required sell price or acceptable buy price.
- Add batch import for forward-calculation scenarios from a structured CSV input file.
- Reuse the existing `ProfitRecord`, storage, and display flows so all workflows produce the same output shape.
- Keep validation deterministic and easy to cover with table-driven tests.

**Non-Goals:**
- Changing the persisted record header or introducing a second result-storage format.
- Supporting JSON, Excel, or interactive prompt based batch imports in this iteration.
- Supporting multiple reverse targets in one command invocation.
- Adding partial-success batch imports that write valid rows while skipping invalid ones.
- Reworking the display renderer beyond wiring new workflows into the existing output modes.

## Decisions

### 1. Use net multiplier as the single canonical fee parameter

The system will treat the effective market multiplier as the canonical configuration value. Forward calculation, reverse calculation, and batch import rows will all normalize onto a `multiplier` field, with `0.837` used when no override is provided.

This keeps the formulas aligned with the current code and avoids ambiguity between "fee rate" and "net multiplier" representations. Documentation can still explain that Steam's default assumption corresponds to a fee rate of `16.3%`, but the implementation path will store and compute only the multiplier.

Alternatives considered:
- Accept both `--fee-rate` and `--multiplier` in v1: more flexible, but introduces precedence rules and more validation branches in the first bundled change.
- Replace the existing multiplier mental model with fee-rate-only inputs: friendlier for some users, but causes larger churn in formulas, naming, and tests.

### 2. Add a dedicated `reverse` CLI workflow instead of overloading `add`

Reverse calculation will be exposed as a new `reverse` subcommand. The command will require:

- `--solve buy|sell` to select which price is being solved
- exactly one known price input:
  - `--buy` when solving for sell price
  - `--sell` when solving for buy price
- exactly one target:
  - `--target-net-profit`
  - `--target-total-profit`
  - `--target-knife-ratio`
- optional `--quantity`, defaulting to `1`
- optional `--multiplier`, defaulting to `0.837`

The reverse workflow will compute a standard `ProfitRecord`, so the solved result can be displayed and persisted through the same existing layers as forward calculation.

Alternatives considered:
- Add reverse flags onto `add`: smaller command count, but it makes one command serve incompatible input modes and creates harder-to-read validation rules.
- Split reverse into multiple subcommands such as `reverse-sell` and `reverse-buy`: explicit, but grows the CLI surface more than necessary for the first iteration.

### 3. Normalize reverse targets onto the existing per-record model

Reverse formulas will operate on the same semantics already used by stored records:

- `net_profit` remains per-item profit
- `total_profit` remains `net_profit * quantity`
- `knife_ratio` remains `buy_price / (sell_price * multiplier)`

When the user supplies `--target-total-profit`, the system will derive the equivalent per-item net profit by dividing by quantity. This preserves one canonical record shape and lets reverse results reuse the current renderer and CSV store without introducing a special reverse-only schema.

Alternatives considered:
- Introduce a separate reverse result type with target metadata: potentially richer, but it would force branching in rendering, storage, and tests.
- Persist the target value instead of derived record values: useful for auditability, but outside the current record schema and not required for the analysis goal.

### 4. Batch import uses a dedicated scenario CSV with one price scenario per row

Batch import will be exposed as an `import` subcommand that reads a separate input CSV file. The input format will be row-oriented and forward-calculation only for v1, with header:

`item_name,buy_price,sell_price,quantity,multiplier`

Each row describes one scenario and produces one output record. `quantity` and `multiplier` may be omitted as empty cells, in which case the existing defaults apply (`1` and `0.837`). Users who want multiple sell prices for the same item can repeat the item across multiple rows.

This format deliberately avoids list parsing inside a CSV cell and keeps import behavior easy to validate and reason about.

Alternatives considered:
- Allow semicolon-delimited sell-price lists in one cell: more compact files, but adds another parsing layer and more edge cases.
- Make import support both forward and reverse rows from day one: more powerful, but materially increases the first implementation bundle.

### 5. Batch import is atomic at the command level

The import workflow will fully read and validate the input file before appending anything to the result store. If any row is malformed or semantically invalid, the command will fail and write no imported records.

This avoids partially imported result files, which would be hard to reason about during analysis and awkward to clean up manually.

Alternatives considered:
- Best-effort import with per-row skips: convenient for noisy files, but makes result persistence less predictable and complicates error reporting.

### 6. Keep record persistence unchanged and extend workflows around it

Neither reverse calculation nor batch import requires a new persisted record schema. Both workflows will emit the current `ProfitRecord` fields and append them to the existing CSV store. The display layer will continue to own human-readable output, while storage remains machine-friendly CSV persistence.

This keeps the implementation incremental and aligns with the repository's current architecture decisions.

Alternatives considered:
- Add an input-history or workflow-type column to persisted records: potentially useful later, but unnecessary for the current value proposition and would create an avoidable schema migration.

## Risks / Trade-offs

- [Multiplier-only configuration is less intuitive for users who think in fee rate] -> Document the default mapping and keep room for a future `fee-rate` alias if usage shows demand.
- [Reverse command can become ambiguous when users mix incompatible flags] -> Enforce strict mutual exclusion and produce targeted validation errors for missing or conflicting inputs.
- [Strict import schema may reject loosely formatted files] -> Keep the first import header small and stable, and provide explicit row-level error messages with row numbers.
- [Atomic import requires buffering all parsed rows before writing] -> Accept the memory cost because this is a local CLI workflow and expected batch sizes are modest.

## Migration Plan

1. Extend the calculation/domain layer with request types that can carry an optional multiplier and reverse-calculation inputs.
2. Add new CLI entrypoints for `reverse` and `import`, while keeping `add` and `list` backward compatible.
3. Implement the batch import parser and validation flow, then wire imported records into the existing CSV store.
4. Update local CLI documentation and add regression tests around default multiplier compatibility.
5. Run `go test ./...` and manual CLI smoke tests for forward, reverse, and import workflows.

Rollback strategy:

- The change is local-code only and does not migrate existing persisted records, so rollback is simply reverting the new code paths and docs.
- Imported result rows written after rollout remain valid current-schema records and do not require special cleanup logic.

## Open Questions

- No blocking open questions are required for this planning stage. The design intentionally chooses multiplier as the only fee configuration surface and limits batch import to forward scenarios so implementation can stay readable within one bundled change.
