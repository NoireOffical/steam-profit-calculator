## MODIFIED Requirements

### Requirement: Server exposes bounded JSON APIs

The system SHALL provide versioned JSON endpoints for item lookup, forward calculation, reverse calculation, persisted history, assistant status and streaming, aggregate runtime metrics, and dependency-aware health. Successful calculation responses SHALL use values produced by the deterministic calculation package and persisted by the configured server-side repository. Invalid input, dependency failures, unsupported methods, and unknown API routes SHALL return stable error envelopes without partial writes.

#### Scenario: Operator checks runtime state

- **WHEN** an operator requests health and metrics
- **THEN** health reflects required persistence availability
- **AND** metrics return aggregate request and assistant counters without user content

#### Scenario: Required persistence is unavailable

- **WHEN** the configured record repository fails its health check
- **THEN** health returns `503` with a stable error envelope
- **AND** internal dependency details are not exposed
