## MODIFIED Requirements

### Requirement: Server exposes persisted history

The system SHALL provide a JSON history endpoint that reads records from the configured server-side repository. Browser storage MUST NOT be the authoritative source for calculation history. The endpoint SHALL validate item-prefix, outcome, order, limit, and offset parameters and return durable record metadata with pagination metadata.

#### Scenario: Client reads a filtered history page

- **WHEN** a client requests record history with valid filters and pagination values
- **THEN** the server returns matching records with IDs and creation timestamps in stable order
- **AND** returns total, limit, offset, and whether another page exists

#### Scenario: Client supplies an invalid history query

- **WHEN** a client supplies an unsupported filter, order, limit, or offset value
- **THEN** the server returns `400 invalid_request`
- **AND** does not query the record repository

### Requirement: Browser application uses the server API

The system SHALL serve a browser application that can search items, submit forward calculations, display persisted history, filter history by item and outcome, change stable history ordering, and navigate bounded result pages. The browser MUST use the HTTP API for calculations and history and MUST NOT duplicate profit formulas or store authoritative history locally.

#### Scenario: User explores persisted history

- **WHEN** a user changes a history filter, order, or page
- **THEN** the browser requests the corresponding server-side history page
- **AND** displays record identity, creation time, total matches, and navigation state
