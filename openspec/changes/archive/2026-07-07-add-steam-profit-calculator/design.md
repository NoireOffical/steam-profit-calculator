## Context

This repository is currently an empty OpenSpec experiment area with no implementation code. The first change needs to demonstrate a full spec-driven workflow on a problem that is small, deterministic, and easy to verify with near-complete automated test coverage.

The target tool is a Go CLI that records Steam knife-arbitrage calculations. A user provides an item name, one buy price, and one or more sell prices. For each sell price, the system calculates net profit using `sell_price * 0.837 - buy_price` and knife ratio using `buy_price / (sell_price * 0.837)`, then persists the result in CSV so multiple items can be stored and reviewed later.

## Goals / Non-Goals

**Goals:**
- Provide a minimal but complete CLI workflow for adding and listing Steam profit records.
- Keep core calculation logic separate from argument parsing and file persistence so tests can cover each layer directly.
- Store each computed sell-price scenario as a separate CSV row to keep persistence flat and easy to inspect.
- Make output deterministic enough for stable unit tests and repeatable spec-driven implementation.

**Non-Goals:**
- Calling Steam APIs or fetching live market prices.
- Supporting interactive prompts in the first version.
- Making the fee multiplier configurable in the first version.
- Editing, deleting, or migrating saved records.
- Building a GUI or web service.

## Decisions

### 1. Use a Go CLI with explicit `add` and `list` commands

The first version will expose two user operations:
- `add`: accept `--name`, `--buy`, and repeated `--sell` flags, calculate one result per sell price, and persist all results.
- `list`: read and print previously saved records.

This keeps the external interface small while still covering input validation, computation, persistence, and retrieval.

Alternatives considered:
- Interactive terminal prompts: easier for manual use, but harder to test thoroughly.
- Library only: simpler internally, but it does not satisfy the requirement to save and review multiple products as a standalone program.

### 2. Split implementation into calculation, storage, and CLI layers

The code will be organized so that:
- a calculation package owns formulas and validation for buy/sell inputs,
- a storage package owns CSV header management, append, and readback,
- the CLI layer maps command flags to those packages.

This matches the main behavioral boundaries in the spec and keeps tests focused.

Alternatives considered:
- Single-file CLI implementation: faster to start, but it would mix parsing, business rules, and file I/O, making spec-to-test traceability worse.

### 3. Store one computed result per CSV row

Each sell price produces one persisted record. The CSV file will use a fixed header and append-only writes. One item entered with three sell prices will therefore create three rows.

This flat representation is a better fit for CSV than storing a nested list of sell prices in one row, and it makes listing behavior straightforward.

Alternatives considered:
- One row per item with embedded sell-price arrays: poor fit for CSV and harder to parse.
- JSON persistence: structurally easier for nested data, but the user explicitly asked for CSV.

### 4. Treat the Steam fee multiplier as a fixed constant in the first version

The multiplier will be fixed at `0.837` and applied consistently everywhere. This locks the first change to the exact formula requested and avoids widening the configuration surface before the core workflow is proven.

Alternatives considered:
- Configurable multiplier flag: more flexible, but not required for the initial experiment and expands the validation matrix.

### 5. Normalize numeric output for persistence and display

The implementation should avoid surprising output drift by formatting persisted and displayed numeric values consistently. Internal representation can remain simple, but storage and CLI output should use a stable fixed precision so test expectations remain deterministic.

Alternatives considered:
- Raw floating-point string output: simpler, but can produce noisy representations that make CSV fixtures and CLI assertions brittle.

## Risks / Trade-offs

- [Floating-point precision noise] -> Use consistent parsing and fixed output formatting at the storage and CLI boundaries.
- [CSV is append-only and schema-light] -> Keep the first file format narrow, document the fixed header, and defer record mutation features.
- [No interactive mode in v1] -> Keep the core logic reusable so an interactive frontend can be added later without changing formulas or storage behavior.
- [Single-file local storage can be corrupted by manual edits] -> Fail fast on malformed CSV rows and surface clear read errors.
