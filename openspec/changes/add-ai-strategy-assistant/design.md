## Context

The repository already has trusted calculation and lookup boundaries and a same-origin HTTP application. Model integration must sit above those boundaries: it may interpret intent and choose a tool, but it cannot reproduce formulas, invent current market values, or silently persist calculations.

The OpenAI Responses API function-calling contract represents tools with JSON Schema, returns `function_call` output items, and accepts matching `function_call_output` items. The provider adapter keeps that vendor-specific protocol behind a project-owned interface.

## Goals / Non-Goals

**Goals:**
- Provide a complete natural-language-to-tool-to-grounded-answer vertical slice.
- Make provider behavior injectable and testable without network access.
- Stream lifecycle events and answer chunks to the browser over a same-origin endpoint.
- Show exact structured tool evidence next to model prose.
- Make prompt version, model, latency, token usage, tool calls, and failures observable.
- Degrade to an explicit unavailable state without affecting deterministic workflows.

**Non-Goals:**
- Persisting chat sessions or model responses.
- Letting assistant tools append calculation history.
- Authentication, user isolation, public rate limiting, or public deployment.
- Replacing CSV persistence or the existing browser calculation forms.
- Guaranteeing financial outcomes or automating trades.

## Decisions

### 1. Keep the model behind a semantic provider interface

The orchestrator depends on a `Model` interface that accepts instructions, strict tool definitions, optional provider state, and tool outputs. The OpenAI adapter owns Responses API JSON and HTTP details. Tests inject a scripted model.

AI configuration is optional. `STEAM_PROFIT_AI_PROVIDER=disabled` is the default. Selecting `openai` without a usable key produces an unavailable model status instead of failing application startup.

### 2. Use three read-only trusted tools

The assistant can call `lookup_items`, `calculate_profit`, and `solve_target`. Lookup delegates to the normalized lookup service. Both numerical tools call the existing `internal/calculation` package. They never append records, so asking a conversational question has no persistence side effect.

Tool schemas use strict JSON Schema with all properties required and `additionalProperties: false`. Nullable fields represent mutually exclusive reverse-calculation inputs.

### 3. Carry provider output within one request

Each assistant HTTP request is stateless. During its bounded tool loop, the OpenAI adapter replays returned response output items and matching tool outputs in memory with storage disabled. No model conversation ID, prompt, response, or tool result is written to CSV or browser storage.

### 4. Treat tool evidence as the authoritative answer surface

Each successful tool result receives an evidence ID such as `T1`. The system prompt requires citations like `[T1]`. Before emitting the final model text, a grounding guard verifies that every evidence ID is referenced and that every numeric token in the prose is present in trusted tool output. A failed check replaces the prose with a neutral server-generated fallback and still exposes the exact evidence cards.

This guard is deliberately conservative. It can suppress a useful explanation, but it cannot turn an unverified number into a trusted result.

### 5. Stream application events with SSE

`POST /api/v1/assistant/stream` returns server-sent events for status, tool start, tool evidence, answer deltas, completion metadata, and errors. The first provider adapter completes each model turn before the orchestrator emits bounded answer chunks; tool progress and evidence are still delivered as soon as they become available. A future provider can add native token deltas without changing the browser event contract.

### 6. Make model behavior observable but keep secrets server-side

Completion metadata includes provider/model identity, prompt version, grounding status, tool count, elapsed time, and aggregate token usage. API keys remain only in process environment and are never included in status, event, error, or log payloads.

## Failure Semantics

- Disabled or misconfigured AI returns an unavailable status and `503` for new assistant requests.
- Invalid client input returns a stable JSON error before an event stream starts.
- Provider or orchestration failures after streaming starts produce an SSE `error` event.
- Tool validation and upstream lookup failures are returned to the model as structured tool errors and shown to the user; they do not mutate storage.
- The tool loop stops after a fixed number of rounds to bound latency and cost.
- Existing deterministic API and CLI workflows do not depend on model availability.

## Security Boundaries

- Same-origin policy and existing response security headers remain in force.
- Assistant request bodies and message length are bounded and unknown JSON fields are rejected.
- Tool names are allow-listed; arguments are decoded with unknown-field rejection and domain validation.
- Assistant tools are read-only and cannot execute shell commands, external URLs, trades, or arbitrary persistence operations.
- Upstream error bodies are size-limited and never expose the API key.

## Rollback

Set `STEAM_PROFIT_AI_PROVIDER=disabled` to remove all model calls while retaining the assistant unavailable state and every deterministic workflow. The feature can also be reverted without migrating CSV data because it introduces no persisted schema.
