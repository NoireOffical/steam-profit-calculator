## Context

The project already has a read-only `lookup` workflow and an internal client interface, but the runtime construction path is still effectively hard-coded to the Steam Community Market public endpoint. This was acceptable for the first version of lookup, but it is now a structural limitation because the public Steam path is fragile under repeated `429` rate limiting and may need to be replaced by a proxy or commercial provider.

The current CLI contract is intentionally narrow: users express intent with `--query`, `--limit`, and `--output`, while network details are configured through environment variables. This change needs to preserve that separation. Provider switching is an operational concern and should not become part of the user-facing CLI syntax.

## Goals / Non-Goals

**Goals:**
- Add a provider selection layer so lookup can construct different provider adapters from configuration.
- Rename the current Steam-backed implementation so it is clearly one provider adapter rather than the identity of the whole lookup subsystem.
- Keep the normalized lookup output schema unchanged across providers.
- Keep provider selection in configuration rather than exposing provider-specific CLI flags.
- Preserve backward-compatible behavior by defaulting to the current public Steam implementation when no provider is configured.

**Non-Goals:**
- Implement a second real provider in this change.
- Add provider selection flags to the `lookup` command.
- Change the normalized lookup result fields or renderer behavior.
- Introduce provider fallback chains, weighted routing, or health-based failover.
- Add caching or broader `429` mitigation beyond what already exists.

## Decisions

### 1. Introduce an explicit provider selector in config

The lookup config should gain a provider selector such as `STEAM_PROFIT_LOOKUP_PROVIDER`. The selection must happen at client-construction time, not in the CLI parsing layer.

Why:
- It preserves the existing command contract.
- It treats provider selection as deployment/runtime configuration.
- It allows future providers to be added without changing `lookup --query ...`.

Alternatives considered:
- Add `--provider` to the CLI: rejected because it mixes operational routing with user intent and expands the command surface unnecessarily.
- Keep a single provider and only override `BASE_URL`: rejected because different providers may need distinct validation, auth, or payload adapters even if they expose similar endpoints.

### 2. Rename the current Steam client to `steam_public`

The current Steam-backed implementation should be renamed to reflect that it is a public Steam adapter. The selector should default to this provider when `STEAM_PROFIT_LOOKUP_PROVIDER` is unset.

Why:
- It makes the architecture honest: Steam public becomes one adapter among many.
- It gives future providers a stable naming pattern.
- It avoids breaking current users who do not set the new config key.

Alternatives considered:
- Keep the current generic naming: rejected because it implies the Steam client is the system default shape instead of a provider-specific adapter.
- Require an explicit provider value at all times: rejected because it would add migration friction for existing usage.

### 3. Keep a single normalized internal output model

Every provider adapter must continue to map its upstream response into the existing normalized item summary model before results reach rendering or CLI output.

Why:
- The renderer and CLI logic remain provider-agnostic.
- Specs stay centered on user-visible behavior instead of upstream payload shape.
- Switching providers becomes a construction/configuration change rather than a renderer rewrite.

Alternatives considered:
- Expose provider-specific fields directly in CLI output: rejected because it would fragment the lookup contract and make providers non-interchangeable.

### 4. Validate provider-specific configuration inside the factory path

The selection layer should create provider clients through a small factory or constructor switch. Each provider should validate its own required config when selected.

Why:
- It keeps provider validation local to the adapter that needs it.
- It prevents config rules for future providers from leaking into unrelated code paths.
- It provides clearer failure messages for unsupported providers or incomplete provider config.

Alternatives considered:
- Centralize every provider's config rules in one generic parser: rejected because it becomes brittle as provider count grows.

## Risks / Trade-offs

- Unsupported or misspelled provider values could break lookup startup → return an explicit configuration error naming the unsupported provider.
- Keeping shared config keys like base URL and API key may still look Steam-oriented during transition → introduce the provider selector first, then evolve naming incrementally in follow-up changes if needed.
- Only one real provider will exist after this change → acceptable because the goal is to create the stable selection seam before adding more adapters.
- Renaming the current Steam client can touch tests and constructor paths across multiple files → contain the rename within `internal/lookup` and keep public CLI behavior unchanged.

## Migration Plan

1. Add the provider selector to lookup configuration with default value `steam_public` when unset.
2. Rename the current Steam-backed client implementation to the `steam_public` naming model.
3. Introduce a provider-construction switch/factory that returns the selected provider adapter.
4. Update tests and CLI documentation to describe configuration-based provider selection.
5. Preserve current runtime behavior for users who do not set the new provider config.

Rollback is straightforward: remove the provider selector and factory path, and restore direct construction of the Steam-backed client.

## Open Questions

- Should later follow-up changes rename the remaining Steam-oriented environment variables to provider-neutral names, or is provider selection alone enough for the first migration step?
- When a `proxy` provider is introduced, should it reuse the current normalized output exactly or reserve optional provider metadata for debugging?
