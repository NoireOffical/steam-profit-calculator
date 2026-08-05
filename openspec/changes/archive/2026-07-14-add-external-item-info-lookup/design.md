## Context

The current CLI is a fully local workflow: users pass explicit item names and prices, the tool runs deterministic calculations, and the result is optionally persisted to CSV. This keeps the existing system easy to test and reason about, but it also means users must gather and verify item metadata outside the tool before they can calculate or store anything.

This change introduces the first network-dependent workflow in the project. That makes the design cross-cutting even though the feature itself is read-only: the CLI needs a new command, the project needs an external API client boundary, output needs a normalized item-summary model, and failures such as timeouts, provider rate limits, or upstream schema drift must be isolated from the existing calculation and storage flows.

The current proposal intentionally limits scope to a narrow lookup summary. It returns item identity plus a small set of market-summary fields, does not auto-fill calculation flags, and does not change the record schema in `steam-profit-records.csv`.

## Goals / Non-Goals

**Goals:**
- Add a read-only lookup workflow that queries an external provider by item keyword or name.
- Normalize provider responses into a project-owned item summary model so CLI output is stable even if provider payloads vary.
- Include the selected output fields `item_id`, `canonical_name`, `app_name`, `sale_price_text`, `sell_price_text`, and `sell_listings` in the normalized lookup summary.
- Keep lookup behavior isolated from calculation, reverse-calculation, import, and CSV storage workflows.
- Make the integration highly testable through an interface boundary and stubbed HTTP responses.
- Provide deterministic CLI output for successful lookups and clear user-facing errors for lookup failures.

**Non-Goals:**
- Using lookup results to compute profits automatically.
- Persisting lookup results into the existing profit-record CSV file.
- Supporting multiple providers, fallback chains, or provider comparison in the first version.
- Enriching existing stored profit records with externally fetched metadata.
- Adding interactive prompts or fuzzy selection flows beyond command-line arguments.

## Decisions

### 1. Introduce a dedicated item-info lookup boundary

The implementation should add a dedicated lookup service instead of letting the CLI call an HTTP endpoint directly. A small internal model such as `ItemInfo` or `ItemSummary` should represent normalized results, while a provider-facing client interface should hide transport details from the CLI layer.

This keeps the first external integration aligned with the current project structure, where CLI, calculation, storage, and rendering already have distinct responsibilities.

Alternatives considered:
- Calling HTTP directly inside `internal/cli/run.go`: simpler initially, but it would mix argument parsing, networking, normalization, and output handling in the least testable layer.
- Reusing `domain.ProfitRecord` for lookup results: rejected because lookup data is not a persisted profit record and should not inherit record-storage semantics.

### 2. Add a standalone `lookup` command rather than overloading existing commands

The CLI should expose a standalone `lookup` command with a required query input and optional display controls. This makes the new workflow explicit and keeps existing commands stable: `add`, `reverse`, `import`, and `list` can continue to mean “calculate or show records,” while `lookup` means “search external item metadata.”

Alternatives considered:
- Extending `add` to auto-resolve names before calculation: rejected because it couples lookup errors to the main calculation path and blurs the command contract.
- Extending `list` to query remote data: rejected because `list` currently represents local persisted data only.

### 3. Keep lookup output separate from record storage

Lookup results should be rendered for inspection only and must not be appended to `steam-profit-records.csv`. The existing storage file remains the source of truth for computed profit records, while lookup output is ephemeral CLI output.

This preserves the current machine-friendly storage schema and avoids mixing two different concepts in the same CSV file.

Alternatives considered:
- Appending lookup metadata into the record file: rejected because the current record schema models calculated profit scenarios, not catalog search results.
- Creating a second persistent catalog cache in the first version: rejected to keep the initial external integration narrow and to avoid introducing invalidation logic before it is needed.

### 4. Make provider configuration external to command flags

The specific provider endpoint, credentials, timeout settings, and `User-Agent` should live in configuration such as environment variables or a thin internal config layer, not in every command invocation. The command surface should stay focused on user intent (`lookup --query ...`) rather than transport details.

This also keeps tests straightforward because they can inject a stub client or a test endpoint without changing the user-facing CLI contract.

Alternatives considered:
- Hard-coding a provider URL in source: quick to implement, but brittle and harder to test or replace.
- Adding provider URL or token flags to the CLI: flexible, but noisy for normal use and easy to misuse.

### 5. Treat lookup failures as non-mutating command failures

Network timeouts, upstream errors, provider rate limits, invalid provider payloads, and empty-match cases should surface as explicit lookup failures. The command should exit with an error, print a clear reason, and leave the local record file untouched.

This matches the rest of the project’s preference for explicit failures over silent partial behavior. For `429 Too Many Requests`, the client should perform bounded backoff retries before returning the final failure.

Alternatives considered:
- Returning an empty successful result for all failure cases: rejected because it hides the difference between “no matching items” and “lookup system unavailable.”
- Falling back to a partial local guess: rejected because this change is specifically about verifying items against an external catalog.

### 6. Reuse explicit output modes, but with a lookup-specific renderer

The `lookup` command should follow the existing CLI convention of explicit output modes such as table and CSV. However, it should use a lookup-specific field set instead of reusing profit-record headers. That keeps output deterministic without confusing lookup summaries with persisted profit rows.

The selected v1 lookup field set should be kept intentionally small: `item_id`, `canonical_name`, `app_name`, `sale_price_text`, `sell_price_text`, and `sell_listings`. This keeps the output useful for manual review while avoiding a premature commitment to the provider's full payload surface.

Alternatives considered:
- Table-only output: simpler, but inconsistent with the existing CLI style.
- Reusing the profit display renderer: rejected because lookup summaries have a different schema and meaning.

## Risks / Trade-offs

- [External API availability can be lower than local CLI reliability] -> Use short timeouts, contextual errors, bounded retries for `429`, and a clear failure contract that does not affect local records.
- [Provider response schema may drift] -> Normalize through a narrow internal adapter and fail fast on required-field decoding errors.
- [Keyword queries may return ambiguous item lists] -> Return multiple candidate results with deterministic ordering and optional result limiting.
- [Introducing the first network dependency increases test complexity] -> Keep an interface boundary and prefer stubbed clients or `httptest` servers in tests.
- [Configuration errors can make the new command feel broken] -> Validate required provider configuration early and emit explicit startup errors for the lookup workflow only.

## Migration Plan

No data migration is required because the existing record schema and persisted files remain unchanged.

Implementation rollout can remain additive:
1. Introduce the lookup service and provider adapter behind a new command path.
2. Add CLI rendering and error handling for lookup results.
3. Leave existing calculation and storage tests unchanged, then add new lookup-focused tests.

Rollback is simple because the change does not alter on-disk formats: the lookup command and client wiring can be removed without touching existing records.

## Open Questions

- Which concrete external provider should serve as the first implementation target?
- Whether later iterations should add normalized numeric price fields in addition to the current text and listing summary fields.
