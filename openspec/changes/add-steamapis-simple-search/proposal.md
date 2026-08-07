## Why

The existing `steam_public` lookup adapter now receives repeated `429 Too Many Requests` responses during real CLI and Railway use even though the Steam Community Market host itself remains reachable. The application therefore needs one authenticated provider path that can complete a simple keyword search without scraping the Steam market page endpoint.

## What Changes

- Add a `steamapis` lookup provider backed by SteamApis v2 `GET /v2/items/search`.
- Send the configured API key in the provider header and fail startup when `steamapis` is selected without a key.
- Map item name, game, marketplace lowest USD offer, offer count, and observation time into the existing normalized lookup summary.
- Include the source marketplace in the returned price text so third-party prices are not presented as official Steam Community Market prices.
- Keep direct search deliberately small: no full-catalog synchronization, local search index, detail request, fallback chain, or persistence.
- Preserve the current default `steam_public` provider and all existing CLI, HTTP, CSV, and SQLite contracts.
- Document the lookup and model integrations as optional, replaceable adapters reused through browser, API, CLI, and assistant entry points rather than prerequisites for deterministic website workflows.

## Capabilities

### Modified Capabilities

- `external-item-info-lookup`: Support an authenticated SteamApis keyword-search adapter while retaining the normalized read-only lookup contract.

## Impact

- New provider adapter and provider-specific default base URL in `internal/lookup`.
- New environment configuration documentation for selecting `steamapis` and injecting its API key.
- Browser copy changes from “Steam public market” to the configured market-data provider boundary.
- Clear degraded-state copy: a provider lookup failure does not imply that calculation, target solving, or history is unavailable.
- No calculation formula, CLI flag, HTTP schema, CSV schema, SQLite schema, assistant tool schema, or persisted data changes.
