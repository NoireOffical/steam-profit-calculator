## Context

The lookup subsystem already exposes a provider-neutral `lookup.Client` interface and selects an adapter from environment configuration. Its only real implementation calls Steam Community Market's public page JSON endpoint, which is operationally fragile under automated requests. SteamApis v2 exposes a direct authenticated search endpoint whose lightweight response is sufficient for the existing lookup table.

## Goals / Non-Goals

**Goals:**
- Add the smallest real SteamApis keyword-search path.
- Preserve the normalized item-summary and existing CLI/API contracts.
- Keep API keys server-side and out of logs, errors, browser responses, and repository files.
- Make third-party marketplace price provenance visible in existing text fields.
- Keep invalid input and provider failure free of storage side effects.
- Make the adapter advantage explicit: one normalized boundary supports browser, JSON API, CLI, and assistant-tool entry points without coupling external credentials to calculation history.

**Non-Goals:**
- Download or cache a full Steam item catalog.
- Request SteamApis item details, order histograms, or price history.
- Add provider fallback, automatic retries beyond bounded `429` handling, or background refresh.
- Claim that a third-party marketplace offer is an official Steam Community Market price.
- Change calculation, record persistence, or assistant grounding behavior.

## Decisions

### 1. Add an explicit `steamapis` adapter

`STEAM_PROFIT_LOOKUP_PROVIDER=steamapis` constructs a dedicated adapter. The existing `steam_public` default remains unchanged for backward compatibility, so deployment activation is an explicit environment change.

The provider defaults to `https://marketplaceapi.steamapis.com` only when `steamapis` is selected and `STEAM_PROFIT_LOOKUP_BASE_URL` is unset. Test and private proxy URLs can still override the base URL.

### 2. Use one direct search request

Each lookup calls `GET /v2/items/search` with `game`, `query`, and `limit`. App IDs supported by the SteamApis marketplace search are translated to its documented game codes. Unsupported app IDs fail configuration instead of silently querying the wrong game.

No catalog cache or follow-up details request is introduced. This keeps the milestone demonstrable and bounds provider cost, at the expense of returning only the lightweight search fields.

### 3. Preserve the normalized response contract

The adapter maps:

- `name` to `canonical_name`;
- the provider game code to the existing human-readable `app_name`;
- `priceUSD` to `sell_price_text` formatted as USD and suffixed with the marketplace name;
- `offerCount` to `sell_listings`;
- one response observation time to `query_time` for all returned rows.

`sale_price_text` remains empty because the search endpoint does not return a distinct recent-sale value. A stable opaque `item_id` is derived from the game, marketplace, and name because the search response has no provider item ID.

### 4. Fail safely and keep lookup read-only

Selecting `steamapis` requires a non-empty API key and API-key header. The key is sent only to the configured provider host and never included in errors or logs. Non-2xx responses, invalid JSON, missing required fields, negative prices/counts, timeouts, and exhausted `429` retries return provider errors through the existing HTTP boundary.

Lookup performs no repository call. Invalid input is rejected by the service before any provider request, and provider failure cannot create calculation records.

## Risks / Trade-offs

- Search results can contain the same item from multiple marketplaces → retain marketplace provenance in the displayed price text rather than deduplicating ambiguous offers.
- Direct provider calls consume one request per search → keep the existing UI submit action and bounded result limit; caching is intentionally deferred.
- SteamApis v2 response or plan behavior can change → isolate it behind the existing client interface and validate payloads strictly.
- A real API key is required for final smoke testing → deterministic tests use a local HTTP server; release readiness still requires one bounded live query.

## Migration And Rollback

Activation requires setting `STEAM_PROFIT_LOOKUP_PROVIDER=steamapis` and `STEAM_PROFIT_LOOKUP_API_KEY` in the runtime environment. Rollback changes the provider back to `steam_public` or deploys the prior immutable tag. No data migration or persisted rollback is required.

Provider selection remains explicit rather than an automatic fallback chain. The default `steam_public` path requires no API key. A valid key must be present before selecting `steamapis`; after successful startup, a provider request failure affects lookup calls only and cannot mutate or disable deterministic calculation and history workflows. The optional OpenAI adapter follows the same product boundary at runtime: disabling or losing model access disables the assistant while leaving deterministic workflows available.
