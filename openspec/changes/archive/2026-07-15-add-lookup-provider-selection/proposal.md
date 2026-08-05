## Why

The current `lookup` workflow already uses an internal client interface, but the runtime wiring is still effectively hard-coded to the Steam Community Market public endpoint. Since that endpoint is now operationally fragile under repeated `429` rate limiting, the project needs a provider selection layer so lookup can switch data sources without rewriting the CLI workflow.

The key design principle for this change is that provider switching must happen through configuration rather than command-line flags. The `lookup` command should continue to express user intent such as query terms, output mode, and limits, while provider choice remains an operational concern handled by config.

## What Changes

- Add a lookup provider selection layer so the runtime can choose a provider implementation from configuration instead of always constructing the current Steam public client.
- Rename the current Steam-backed client to make its role explicit as a public Steam provider implementation rather than the default shape of the whole lookup system.
- Introduce provider-oriented configuration naming and selection rules that can support future sources such as a proxy service or a paid third-party provider.
- Keep the normalized lookup output schema and CLI contract stable while moving provider-specific wiring behind the selection layer.
- Preserve the current `lookup` command flags and avoid adding provider-specific CLI parameters.

## Design Details

### Provider selection belongs to configuration

The lookup workflow should select its provider from configuration, not from CLI flags. This keeps the command surface stable:

- `lookup --query ... --limit ... --output ...` continues to describe user intent
- provider choice such as `steam_public`, `proxy`, or a future vendor remains part of runtime configuration

This avoids turning the CLI into a deployment control surface and keeps provider replacement low-risk.

### The current Steam implementation becomes one provider among many

The existing Steam-backed client should be renamed to reflect what it actually is: a public Steam provider adapter. After this change, the system should conceptually look like:

`lookup command -> lookup service -> provider factory -> provider adapter -> normalized item summary`

The Steam public implementation should become a single adapter behind this selection layer rather than the implicit default identity of the entire lookup subsystem.

### Normalized output stays stable

The user-facing lookup output schema should remain stable across providers. Provider-specific response formats, authentication, retry behavior, and payload mapping should stay inside each adapter. This lets the project change data sources without redesigning the renderer or the CLI query flow.

### Backward-compatible operational migration

This change should favor a low-friction migration path:

- introduce a provider selector such as `STEAM_PROFIT_LOOKUP_PROVIDER`
- default to `steam_public` when no provider is specified
- continue to support the current common config fields such as base URL, timeout, API key, and user agent
- allow future provider-specific validation without changing the core `lookup` command syntax

## Configuration Examples

These examples show the intended operational model: provider selection is done through config, while the CLI command itself stays unchanged.

### Example 1: Public Steam provider

```bash
export STEAM_PROFIT_LOOKUP_PROVIDER="steam_public"
export STEAM_PROFIT_LOOKUP_BASE_URL="https://steamcommunity.com"
export STEAM_PROFIT_LOOKUP_APP_ID="730"
export STEAM_PROFIT_LOOKUP_TIMEOUT_MS="5000"
export STEAM_PROFIT_LOOKUP_USER_AGENT="steam-profit-calculator/lookup"

CGO_ENABLED=0 go run ./cmd/steam-profit lookup --query "2023" --limit 5 --output table
```

### Example 2: Proxy provider

```bash
export STEAM_PROFIT_LOOKUP_PROVIDER="proxy"
export STEAM_PROFIT_LOOKUP_BASE_URL="https://lookup-proxy.example.com"
export STEAM_PROFIT_LOOKUP_TIMEOUT_MS="5000"
export STEAM_PROFIT_LOOKUP_API_KEY="Bearer your-token"
export STEAM_PROFIT_LOOKUP_API_KEY_HEADER="Authorization"
export STEAM_PROFIT_LOOKUP_USER_AGENT="steam-profit-calculator/lookup"

CGO_ENABLED=0 go run ./cmd/steam-profit lookup --query "2023" --limit 5 --output csv
```

### Example 3: Provider switch without CLI changes

```bash
# only configuration changes
export STEAM_PROFIT_LOOKUP_PROVIDER="proxy"

# command stays the same
CGO_ENABLED=0 go run ./cmd/steam-profit lookup --query "2023" --limit 1
```

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `external-item-info-lookup`: The lookup workflow will support selecting among interchangeable provider implementations while preserving normalized output behavior.
- `profit-cli-interface`: The `lookup` command configuration contract will expand to include provider selection without changing the user-facing query/output flags.

## Impact

- Affected code: `internal/lookup/config.go`, `internal/lookup/lookup.go`, `internal/lookup/steam_client.go` or its renamed equivalent, lookup client construction, provider-specific tests, and `CLI_USAGE.md`
- Affected specs: `openspec/specs/external-item-info-lookup/spec.md`, `openspec/specs/profit-cli-interface/spec.md`
- Operational impact: future providers such as `proxy` or commercial data sources can be added behind the same lookup command without redesigning the CLI flow
- Config impact: provider replacement becomes a configuration change instead of a CLI contract change
