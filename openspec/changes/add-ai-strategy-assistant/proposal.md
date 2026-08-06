## Why

The browser workbench now exposes deterministic lookup and calculation workflows, but users still need to translate a natural-language goal into exact form fields and interpret several returned scenarios themselves. The next AI full-stack milestone should add that interpretation layer without allowing a model to become a second source of market data or financial formulas.

## What Changes

- Add a replaceable model-provider boundary with an OpenAI Responses API adapter and an unavailable fallback.
- Expose item lookup, forward calculation, and reverse calculation as read-only assistant tools.
- Add a server-sent event assistant endpoint that reports model progress, trusted tool evidence, grounded answer chunks, and observable usage metadata.
- Add a browser conversation panel with example prompts, visible tool evidence, disabled-state guidance, and no durable chat storage.
- Add a numeric grounding guard and a fixed 30-case deterministic evaluation set.
- Keep the CLI, existing JSON routes, CSV schema, and calculation persistence behavior compatible.

## Capabilities

### New Capabilities

- `ai-strategy-assistant`: Interpret a natural-language market question, invoke trusted project tools, and explain only tool-produced values.

### Modified Capabilities

- `web-application-interface`: Expose assistant status and event-stream routes and render the assistant panel.

## Impact

- New `internal/assistant` package for provider, orchestration, tool execution, grounding, and evaluation fixtures.
- New optional AI environment variables; AI is disabled by default and missing credentials do not prevent server startup.
- New assistant HTTP routes and embedded browser interactions.
- No changes to CLI arguments, existing API payloads, persisted CSV schemas, or deterministic formulas.
