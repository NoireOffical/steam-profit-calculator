## ADDED Requirements

### Requirement: Controlled demos can require one environment-managed credential

The web server SHALL support an optional demo access credential configured outside the repository and image. When enabled, the embedded browser, metrics, and application APIs SHALL require valid HTTP Basic authentication using constant-time comparison, while the dependency health endpoint SHALL remain publicly readable. Partial credential configuration SHALL prevent startup.

#### Scenario: Reviewer opens a protected demo

- **WHEN** a reviewer requests the browser without valid demo credentials
- **THEN** the server returns `401` with an authentication challenge
- **AND** no application handler, model provider, or record repository mutation is reached

#### Scenario: Hosting platform checks dependency health

- **WHEN** the hosting platform requests `/api/health` without credentials
- **THEN** the server reports the configured repository health without exposing secrets or application data

### Requirement: Hosted API traffic and paid assistant work are bounded

The server SHALL support a configurable process-local per-client API request limit and a separate process-local assistant-run quota. Rate or quota exhaustion SHALL return `429` with a bounded retry interval before storage mutation or provider work. Limits SHALL be disabled by default for local compatibility and SHALL ignore forwarded client headers unless trusted-proxy mode is explicit.

#### Scenario: One client exceeds the API limit

- **WHEN** an authenticated client exceeds the configured request count within the active minute
- **THEN** the next application API request returns `429`
- **AND** another client retains an independent request budget

#### Scenario: Assistant quota is exhausted

- **WHEN** a valid assistant request arrives after the configured hourly run budget has been consumed
- **THEN** the server rejects it before SSE headers and before a provider call
- **AND** deterministic calculation and history APIs remain available within their request budget

### Requirement: Protection signals remain content-free and observable

The runtime metrics snapshot SHALL expose only aggregate totals for unauthorized requests, API rate-limit rejections, and assistant-quota rejections. Request logs and metrics SHALL omit credentials, client addresses, queries, bodies, prompts, answers, provider payloads, and tool or storage values.

#### Scenario: A credential and rate-limit failure are inspected

- **WHEN** maintainers inspect logs and aggregate metrics after rejected requests
- **THEN** they can distinguish authentication, API-limit, and assistant-quota outcomes
- **AND** cannot recover the submitted credential, client address, or request content from those signals

### Requirement: Paid and hosted checks are explicit and secret-safe

The repository SHALL keep normal pull-request CI independent from real OpenAI and hosted services. A live-model workflow SHALL require explicit dispatch and injected credentials, run bounded current and baseline prompt cases, and write content-free reports. A hosted synthetic workflow SHALL perform only read-only health and authenticated status checks and SHALL fail visibly when the target is unhealthy.

#### Scenario: Maintainer runs ordinary CI

- **WHEN** a pull request runs without OpenAI or hosted-demo secrets
- **THEN** deterministic tests, scripted evaluation, browser E2E, and container checks still run
- **AND** no paid model request or hosted data mutation occurs

### Requirement: Hosted storage and rollback remain reproducible

The deployment contract SHALL require one application replica, HTTPS termination, secret injection, a persistent SQLite volume, dependency health checks, graceful stop, backup and restore instructions, and rollback to an immutable prior tag or image without replacing the retained volume.

#### Scenario: Maintainer rolls back a failed hosted release

- **WHEN** a new immutable application version fails its hosted smoke
- **THEN** the previous immutable version can be started against the retained SQLite volume
- **AND** existing records remain readable without a schema migration

#### Scenario: Railway injects its runtime port

- **WHEN** the Railway deployment supplies a valid `PORT` without an explicit server address
- **THEN** the server listens on `0.0.0.0` at that port
- **AND** Railway can activate the deployment only after the public dependency health path returns `200`

#### Scenario: Railway identifies a rate-limit client

- **WHEN** trusted-proxy mode is enabled behind Railway's managed HTTPS edge
- **THEN** a valid `X-Real-IP` identifies the per-client request window
- **AND** untrusted deployments continue to ignore proxy-provided client headers by default
