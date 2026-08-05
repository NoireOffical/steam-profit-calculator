## Context

The current implementation uses the same record field order for storage and CLI output, but it still ties terminal presentation closely to CSV-shaped strings. This creates friction for future improvements such as aligned columns, mode switching, summaries, or richer renderers, because every display change risks leaking into persistence expectations.

The user has explicitly chosen to keep CSV as the storage format and to move beautification into the presentation layer only. This change therefore needs to create a clear boundary between record persistence and record rendering.

## Goals / Non-Goals

**Goals:**
- Keep on-disk CSV persistence unchanged.
- Introduce a dedicated display-rendering path for terminal output.
- Make the CLI support explicit output modes, with a pretty table as the default.
- Ensure all display modes reuse the same canonical record ordering and values.

**Non-Goals:**
- Changing CSV headers, row shapes, or migration behavior.
- Adding non-terminal outputs such as HTML or web UI.
- Adding business logic changes to profit calculation.
- Adding ANSI color or complex terminal dependencies in the first iteration.

## Decisions

### 1. Separate storage serialization from display rendering

CSV persistence will remain the responsibility of the storage/domain formatting path. Terminal display will move to a separate renderer that consumes `ProfitRecord` values and produces strings for stdout.

Alternatives considered:
- Continue evolving CSV-like output in place: less initial code, but it keeps storage and presentation entangled.

### 2. Add explicit output modes

The CLI will expose an output selector, such as `--output table` and `--output csv`, for both `add` and `list`. The default mode will be `table` for interactive readability.

Alternatives considered:
- Hard-switch all terminal output to table only: simpler UX, but it removes an easy machine-friendly stdout mode.

### 3. Use a plain-text aligned table as the default pretty renderer

The first renderer will use fixed-width column alignment with headers and rows derived from the existing field order. Numeric columns should be right-aligned for scanability.

Alternatives considered:
- Box-drawing tables: visually richer, but harder to keep compact and portable.
- Markdown tables: useful for documents, but less natural as a terminal default.

### 4. Keep renderer outputs deterministic

Renderer output should be deterministic and testable, with stable column order and predictable spacing. Width calculation should depend only on headers and record values in the current result set.

Alternatives considered:
- Adaptive wrapping and truncation heuristics: nicer for large data, but harder to specify and test in a first pass.

## Risks / Trade-offs

- [Table formatting adds layout logic] -> Keep the first renderer minimal: plain-text table, fixed order, no color, no wrapping.
- [Multiple output modes can drift] -> Reuse shared record field extraction so table and CSV display read from the same canonical source.
- [Wide item names can stretch output] -> Accept wide rows in v1 and defer truncation/width limits to a later change.
