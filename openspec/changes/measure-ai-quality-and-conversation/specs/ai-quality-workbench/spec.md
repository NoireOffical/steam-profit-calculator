## ADDED Requirements

### Requirement: Assistant context is bounded and transient

The assistant stream endpoint SHALL continue accepting the existing message-only request and MAY accept up to six prior user or assistant messages as optional transient history. The server SHALL validate roles, per-message size, combined size, and unknown fields before starting the stream. History SHALL NOT be persisted, logged, counted as current-turn evidence, or written to calculation records.

#### Scenario: User asks a bounded follow-up

- **WHEN** the browser sends a valid current message with recent page-local history
- **THEN** the model receives that history as untrusted conversation context
- **AND** any current answer containing calculated or market values still requires fresh current-turn tool evidence

#### Scenario: Client sends invalid history

- **WHEN** history exceeds a bound or contains an unsupported role
- **THEN** the endpoint returns a validation error before SSE starts
- **AND** no provider call, tool side effect, or storage write occurs

### Requirement: Provider-native streams preserve grounding safety

The OpenAI adapter SHALL request a typed Responses API stream and handle text deltas, streamed function-call arguments, completion metadata, provider failure, and error events with bounded parsing. The service SHALL reconstruct and validate the complete final answer before emitting model text to the browser. A grounded answer SHALL use the provider's original non-empty delta boundaries; an ungrounded answer SHALL expose only the safe replacement.

#### Scenario: Streamed tool call produces a grounded answer

- **WHEN** the provider streams function arguments, the trusted tool returns evidence, and the provider streams a cited answer
- **THEN** the tool executes through the existing deterministic boundary
- **AND** the browser receives evidence followed by grounded answer deltas and completion metadata indicating native streaming

#### Scenario: Streamed answer violates grounding

- **WHEN** a completed provider stream contains an unsupported number or omits required evidence citations
- **THEN** none of the provider answer deltas are exposed to the browser
- **AND** the browser receives only the server-generated grounding fallback

### Requirement: Active assistant work can be cancelled and explicitly retried

The browser SHALL provide an accessible Stop control while a request is active and a Retry control after a retryable failure or cancellation. Cancellation SHALL propagate through the HTTP context. Retry SHALL create a new explicit request using the same visible user turn and prior validated transient history; the server SHALL NOT automatically retry provider calls.

#### Scenario: User stops an active request

- **WHEN** the user activates Stop before completion
- **THEN** the browser aborts the request and restores editable controls
- **AND** the server stops provider or lookup work without appending calculation history

#### Scenario: User retries a failed request

- **WHEN** a retryable provider request fails and the user activates Retry
- **THEN** one new assistant request starts with the same visible turn context
- **AND** deterministic calculation and history APIs remain independent throughout the failure and retry

### Requirement: Model quality has versioned deterministic and live evaluation paths

The repository SHALL contain exactly 50 deterministic tool cases and a versioned semantic suite. A scripted report test SHALL run without network access. An opt-in live command SHALL record fixture case IDs, prompt and model versions, observed tools, grounding outcome, elapsed time, token usage, failures, and optional caller-priced cost, and SHALL evaluate explicit thresholds without logging credentials, arbitrary production prompts, or tool result values.

#### Scenario: Maintainer runs normal quality gates

- **WHEN** normal tests and CI run without model credentials
- **THEN** all 50 deterministic cases and scripted report behavior are verified
- **AND** no OpenAI or Steam network request is made by the evaluation gate

#### Scenario: Maintainer compares prompt variants

- **WHEN** a maintainer explicitly runs the live suite for the current and retained baseline prompts with a configured provider
- **THEN** each run emits a comparable machine-readable report
- **AND** threshold failures are attributable to case IDs and observable metadata rather than hidden provider payloads

### Requirement: Existing deterministic workflows remain isolated

Model unavailability, cancellation, invalid history, live-evaluation failure, and retry SHALL NOT break CLI commands, calculation APIs, record persistence, or SQLite health. Assistant tools SHALL remain read-only.

#### Scenario: AI dependency is unavailable

- **WHEN** the provider is disabled, missing credentials, or fails during a request
- **THEN** deterministic CLI and browser calculation workflows remain available
- **AND** the failed assistant work creates no calculation record
