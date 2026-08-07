## ADDED Requirements

### Requirement: SteamApis can provide a direct authenticated keyword search

The system SHALL support selecting a `steamapis` lookup provider that sends one authenticated SteamApis v2 keyword-search request and normalizes its lightweight results without downloading a full catalog or issuing a follow-up item-details request. The provider API key MUST remain server-side and MUST NOT appear in logs, browser responses, or normalized item summaries.

#### Scenario: SteamApis returns matching marketplace offers

- **WHEN** runtime configuration selects `steamapis` with a valid API key and the user submits a supported-game keyword query
- **THEN** the system sends the game code, query, and bounded result limit to the SteamApis search endpoint
- **AND** returns normalized item summaries containing the item name, application name, lowest USD offer, marketplace provenance, offer count, and one shared UTC observation time
- **AND** does not write calculation or lookup records

#### Scenario: SteamApis returns no matches

- **WHEN** SteamApis returns an empty successful search response
- **THEN** the existing lookup service returns the normalized no-matches error
- **AND** no follow-up request or storage mutation occurs

#### Scenario: SteamApis configuration is incomplete

- **WHEN** runtime configuration selects `steamapis` without an API key or with an unsupported Steam App ID
- **THEN** provider construction fails with a configuration error
- **AND** no external request is attempted

#### Scenario: SteamApis dependency fails

- **WHEN** SteamApis times out, returns a non-success response, exhausts bounded rate-limit retries, or returns an invalid payload
- **THEN** lookup fails through the existing safe provider-error boundary
- **AND** the API key and upstream response body are not exposed
- **AND** no calculation record is created

#### Scenario: Marketplace provenance remains visible

- **WHEN** a SteamApis result contains a price from a third-party marketplace
- **THEN** the normalized price text identifies that marketplace
- **AND** the browser does not describe the value as an official Steam Community Market price

### Requirement: Provider adapters remain isolated from deterministic workflows

The system SHALL expose normalized market lookup through the browser, JSON API, CLI, and assistant tool without coupling provider credentials or runtime provider availability to calculation records. Provider selection and API-key rotation SHALL NOT require a UI, command, HTTP-schema, CSV-schema, or SQLite migration.

#### Scenario: A configured provider is unavailable at request time

- **WHEN** the selected market-data provider fails during a lookup request
- **THEN** the lookup entry point reports a bounded provider failure
- **AND** browser calculation, target solving, and history workflows remain available
- **AND** no calculation record is created or changed by the failed lookup

#### Scenario: Steam official lookup remains rate limited after retries

- **WHEN** the default Steam official market endpoint returns `429` through all bounded retry attempts
- **THEN** the HTTP API and assistant tool return the dedicated `official_provider_rate_limited` failure classification
- **AND** the browser explains that the Steam official market API is rate limited
- **AND** recommends configuring a supported third-party Steam Market Provider for effective virtual-item lookup
- **AND** does not describe the deterministic website as unavailable

#### Scenario: A provider or key is changed

- **WHEN** an operator changes the explicit provider or rotates its server-side API key and restarts the service with valid configuration
- **THEN** the existing browser, JSON API, CLI, and assistant lookup entry points use the newly selected adapter
- **AND** existing calculation history remains compatible and unchanged
