## Context

The assistant currently executes a bounded provider/tool loop, exposes trusted evidence, and validates the final prose before sending it to the browser. The OpenAI adapter uses non-streaming Responses API calls and keeps provider state only for the duration of one HTTP request. The browser shows a transient transcript but sends only the newest message. A 30-case fixture validates tool execution, not semantic model selection.

OpenAI Responses streaming uses typed SSE events. Text arrives through `response.output_text.delta`; function calls start with `response.output_item.added` and accumulate `response.function_call_arguments.delta` until the completed output item is available. Provider-specific parsing remains behind the project-owned model interface.

## Goals / Non-Goals

**Goals:**
- Preserve strict, read-only tools and the deterministic source of all calculated numbers.
- Consume provider-native text and function-call streaming without exposing ungrounded partial prose.
- Support a small, validated, page-local conversation context that is never persisted.
- Let the browser cancel an active request and explicitly retry a failed or cancelled turn.
- Produce versioned JSON evaluation reports for deterministic and optional live-model quality.
- Compare prompt variants using the same cases and explicit latency, token, grounding, and optional cost thresholds.

**Non-Goals:**
- Persisting chats, provider response IDs, tool evidence, prompts, or model output.
- Adding authentication, cross-device sessions, public quotas, or multi-user isolation.
- Automatically retrying paid provider requests behind the user's back.
- Showing unverified model tokens before grounding completes.
- Shipping a hard-coded model price table or making live evaluation a required CI dependency.
- Changing calculation formulas, storage schemas, or CLI/CSV behavior.

## Decisions

### 1. Add history without creating server sessions

The existing assistant request keeps its required `message` field and gains an optional `history` array containing only `user` and `assistant` text. The server accepts at most six prior messages, limits every message and the combined rune count, rejects unknown fields and invalid roles, and passes a defensive copy to the model boundary. The browser keeps only the most recent accepted turns in memory and clears them on reload.

History is untrusted context, not evidence. The versioned prompt requires a fresh trusted tool call whenever a follow-up needs market or calculated values. Grounding applies only to evidence generated during the current request.

### 2. Keep streaming provider-specific and grounding provider-neutral

The model interface gains an optional streaming response method. The OpenAI adapter sends `stream: true`, parses typed SSE data with bounded event and response sizes, aggregates function-call arguments by output index, captures completed response metadata, and returns original text-delta boundaries with the provider-neutral response.

The service never forwards provider deltas immediately. It first reconstructs and validates the complete answer against current evidence. If validation passes, it emits the original non-empty deltas; if validation fails, it emits only the safe server-generated replacement. Scripted and non-streaming models keep the existing bounded chunk fallback. This trades first-token latency for the stronger no-ungrounded-number invariant while still exercising and preserving the native provider stream.

### 3. Make cancellation and retry explicit

The browser owns one `AbortController` per active request. A visible Stop control aborts the fetch, returns the form to an editable state, and marks the interrupted assistant turn without creating history or calculation records. Go request cancellation propagates to the provider and lookup calls.

Provider and transport failures expose a safe retryable error classification after streaming begins. The browser shows a Retry button that starts a new request with the same user message and the same validated prior history. The server does not automatically retry, preventing hidden duplicated model cost. Assistant tools remain read-only, so cancellation or retry cannot append calculation history.

### 4. Separate deterministic gates from opt-in live evaluation

The deterministic fixture grows to exactly 50 tool cases and stays in normal tests. A separate versioned semantic suite declares a case ID, prompt, optional history, expected tool sequence, grounding expectation, and per-case token/latency budgets. Tests exercise the report engine with a scripted model; they do not use a network.

The `steam-profit-eval` command is explicitly opt-in. It uses the same configured model boundary, can select the current or retained baseline prompt, limits the number of cases, and writes a JSON report containing case IDs, prompt/model versions, observed tools, grounding, elapsed time, usage, threshold failures, and an optional cost estimate calculated only from caller-supplied rates. Credentials, raw authorization data, provider payloads, arbitrary user traffic, and tool result values are omitted.

### 5. Version prompt behavior and observable capabilities

The production prompt advances to a new immutable version and explicitly describes follow-up context as untrusted. The previous prompt remains available only as an evaluation baseline. Public status and completion metadata report the active prompt version and whether native provider streaming was used; they do not expose configuration secrets.

## Failure Semantics

- Invalid history is rejected with `422` before SSE headers, provider calls, or storage changes.
- Provider stream parse errors, oversized events, failed responses, and timeouts become safe SSE errors without raw provider bodies.
- Client cancellation terminates work through context cancellation and does not emit a synthetic provider-failure event after the connection is gone.
- A failed live evaluation case is recorded in the report; threshold mode returns a non-zero process status only after writing the report.
- Missing model credentials make the live evaluation command fail before any case runs and do not affect normal tests, CLI commands, or the web server.

## Security Boundaries

- Conversation history is length-bounded, role-allow-listed, copied before use, and never trusted as tool evidence.
- The streaming parser caps individual events and total provider bytes and ignores unsupported event types.
- Provider errors are normalized; API keys, headers, raw bodies, prompts, answers, and evidence values stay out of logs and aggregate metrics.
- Reports contain versioned fixture IDs and aggregate metadata rather than credentials or arbitrary production conversations.
- Model tools remain allow-listed, strict-schema, read-only calls into lookup and deterministic calculation packages.

## Rollback

Set `STEAM_PROFIT_AI_PROVIDER=disabled` to stop all model and evaluation calls while retaining deterministic workflows. The optional `history` field and new status metadata are additive, so older clients continue to work. The provider can fall back to the non-streaming `Respond` interface in tests or alternative adapters. Reverting this change requires no data migration because it adds no persisted schema or chat storage.
