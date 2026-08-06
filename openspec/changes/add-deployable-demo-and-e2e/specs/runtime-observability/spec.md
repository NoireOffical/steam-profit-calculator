## ADDED Requirements

### Requirement: HTTP requests are correlated and logged safely

The server SHALL generate a request ID for each HTTP request, return it in `X-Request-ID`, and emit a structured completion log containing method, path, status, duration, and request ID. Logs MUST NOT include query strings, headers, bodies, prompts, model output, market results, persisted record values, or credentials.

#### Scenario: API request fails validation

- **WHEN** an API request returns a client error
- **THEN** the server writes a structured warning with correlation metadata
- **AND** no submitted field value or secret is present in the log

### Requirement: Runtime exposes aggregate request metrics

The server SHALL expose a read-only aggregate snapshot containing process start time, total and in-flight requests, response-class counts, and accumulated request duration. Concurrent requests MUST be recorded without races.

#### Scenario: Operator reads metrics

- **WHEN** an operator requests the metrics endpoint
- **THEN** the server returns process-local aggregate counters
- **AND** the response contains no per-request content or credentials

### Requirement: Runtime exposes aggregate assistant metrics

The aggregate snapshot SHALL include assistant run and failure counts, accumulated elapsed time, input/output/total tokens, and tool calls from assistant completion metadata. Model unavailability MUST NOT affect deterministic endpoints.

#### Scenario: Assistant completes a grounded run

- **WHEN** the assistant emits trusted completion metadata
- **THEN** the runtime increments assistant usage counters from that metadata
- **AND** does not persist or expose the prompt, answer, or tool payload

### Requirement: Health reflects required persistence

The health endpoint SHALL confirm that the configured record repository is usable before returning success. A failed repository health check SHALL return `503` without exposing internal dependency details.

#### Scenario: SQLite dependency is unavailable

- **WHEN** the health endpoint cannot use the configured SQLite repository
- **THEN** it returns a generic unavailable response with HTTP `503`
- **AND** does not attempt a calculation or record write
