## ADDED Requirements

### Requirement: Assistant uses replaceable model providers

The system SHALL place model access behind an injectable provider interface. Model configuration, prompt version, model identity, latency, token usage, and failure outcomes MUST be observable without exposing credentials.

#### Scenario: Model provider is not configured

- **WHEN** the server starts without an enabled model provider or usable credential
- **THEN** the assistant reports an unavailable state
- **AND** the server, existing JSON API, browser calculation workbench, and CLI remain usable

### Requirement: Assistant invokes trusted project tools

The assistant SHALL expose item lookup, forward calculation, and reverse calculation as allow-listed tools with strict argument schemas. Numerical tools MUST call the deterministic calculation package and MUST NOT reproduce formulas in model prompts, provider code, or browser code.

#### Scenario: User asks for a profit comparison

- **WHEN** the model selects the forward-calculation tool with valid arguments
- **THEN** the orchestrator returns calculation-package records as structured tool evidence
- **AND** the conversational workflow does not append those records to persisted history

#### Scenario: Tool arguments are invalid

- **WHEN** a model requests an unknown tool or supplies invalid arguments
- **THEN** the orchestrator produces a bounded structured tool error
- **AND** no record or partial result is persisted

### Requirement: Assistant answers are grounded in tool evidence

Every successful assistant tool result SHALL receive an evidence ID. Model prose containing tool-derived numerical claims MUST cite all applicable evidence IDs, and numerical tokens MUST occur in trusted tool output. The system MUST replace prose that fails this check with a neutral evidence-directed fallback.

#### Scenario: Model invents a numerical value

- **WHEN** final model prose contains a number absent from all successful tool outputs
- **THEN** the user does not receive that prose as a grounded answer
- **AND** the user can still inspect the exact structured tool evidence

### Requirement: Assistant streams an observable event contract

The web application SHALL expose a same-origin SSE endpoint that emits status, tool progress, structured evidence, answer delta, completion, and error events. Completion metadata MUST identify the prompt version, provider/model, grounding outcome, tool count, elapsed time, and available token usage.

#### Scenario: User submits an assistant prompt

- **WHEN** the assistant is available and receives a valid bounded message
- **THEN** the browser progressively renders lifecycle events and the grounded result
- **AND** API keys and raw authorization values are absent from every event

### Requirement: Browser provides an assistant conversation surface

The browser SHALL provide accessible prompt input, examples, model availability, progress, answer, evidence, completion metadata, and error states. Chat content MUST remain transient in this milestone.

#### Scenario: AI is disabled

- **WHEN** the browser reads an unavailable assistant status
- **THEN** the assistant input is disabled with configuration guidance
- **AND** lookup, calculation, reverse API, and history interfaces remain available

### Requirement: Assistant behavior has a fixed evaluation baseline

The repository SHALL contain at least 30 versioned deterministic evaluation cases covering lookup, forward calculation, reverse calculation, invalid input, and grounding behavior. Evaluation execution MUST NOT require a live model credential.

#### Scenario: Maintainer runs the quality gate

- **WHEN** automated tests execute without network credentials
- **THEN** all evaluation fixtures are validated against injected providers and deterministic project tools
