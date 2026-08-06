## ADDED Requirements

### Requirement: Server exposes a versioned JSON API

The system SHALL provide an HTTP application that exposes health, item lookup, forward calculation, reverse calculation, and persisted record history through versioned JSON routes. Existing calculation and lookup packages MUST remain the source of truth for domain behavior.

#### Scenario: Client calculates profit scenarios

- **WHEN** a client submits a valid forward-calculation request to the calculation endpoint
- **THEN** the server returns one JSON profit record per sell price
- **AND** the server appends the records through the configured record store

#### Scenario: Client submits invalid JSON input

- **WHEN** a client submits malformed, oversized, unknown, or domain-invalid input
- **THEN** the server returns a non-success status with a stable JSON error object
- **AND** no record is persisted

### Requirement: Server exposes persisted history

The system SHALL provide a JSON history endpoint that reads records from the configured server-side record store. Browser storage MUST NOT be the authoritative source for calculation history.

#### Scenario: Client reads saved records

- **WHEN** a client requests record history
- **THEN** the server returns the current persisted records in deterministic field form

### Requirement: Server exposes external item lookup

The system SHALL expose the existing normalized lookup workflow through an HTTP endpoint. The endpoint MUST validate query and limit inputs and MUST distinguish no-match, invalid-input, and provider-failure responses.

#### Scenario: Client searches for an item

- **WHEN** a client supplies a valid query and optional positive result limit
- **THEN** the server returns normalized item summaries from the configured provider
- **AND** the lookup does not modify persisted profit records

### Requirement: Browser application uses the server API

The system SHALL serve a browser application that can search items, submit forward calculations, and display persisted history. The browser MUST use the HTTP API for calculations and MUST NOT duplicate the profit formulas.

#### Scenario: User completes the first full-stack workflow

- **WHEN** a user searches for an item, selects or enters its name, submits prices, and refreshes history
- **THEN** the browser displays server-produced lookup results, calculation results, and persisted records

### Requirement: HTTP application applies baseline safeguards

The server SHALL reject unsupported methods, limit JSON request size, reject unknown JSON fields, return JSON errors for API failures, and add baseline response security headers.

#### Scenario: Client uses an unsupported method

- **WHEN** a client calls an API route with an unsupported HTTP method
- **THEN** the server returns `405 Method Not Allowed`
- **AND** advertises the allowed method
