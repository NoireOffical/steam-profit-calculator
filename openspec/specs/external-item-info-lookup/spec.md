## Purpose

Define the read-only external item lookup behavior used to search Steam item metadata and market summary fields before running profit calculations.

## Requirements

### Requirement: System queries an external provider for normalized item summaries
The system SHALL support querying an external item-information provider by user-supplied item keyword or item name. For each matched item, the system MUST return a normalized item summary that includes `item_id`, `canonical_name`, `app_name`, `query_time`, `sale_price_text`, `sell_price_text`, and `sell_listings`.
The selected lookup provider MUST be chosen from configuration rather than from CLI query flags. If no explicit provider is configured, the system MUST default to the public Steam provider.

#### Scenario: Query returns multiple matching items
- **WHEN** the user submits a lookup query that matches multiple items in the external provider
- **THEN** the system returns one normalized item summary per matched item
- **THEN** each returned summary includes `item_id`, `canonical_name`, `app_name`, `query_time`, `sale_price_text`, `sell_price_text`, and `sell_listings`

#### Scenario: Query applies an explicit result limit
- **WHEN** the user submits a lookup query with an explicit result limit smaller than the total number of matched items
- **THEN** the system returns no more than the requested number of normalized item summaries

#### Scenario: Query stamps one shared observation time
- **WHEN** the system returns multiple items from a single lookup request
- **THEN** all returned item summaries share the same `query_time`
- **THEN** `query_time` is formatted as a UTC RFC3339 timestamp

#### Scenario: Provider defaults to the public Steam adapter
- **WHEN** the lookup workflow is constructed without an explicit provider selection
- **THEN** the system uses the `steam_public` lookup provider by default
- **THEN** the lookup output still follows the normalized item-summary contract

#### Scenario: Provider is selected from configuration
- **WHEN** runtime configuration explicitly selects a supported lookup provider
- **THEN** the system constructs that provider adapter for the lookup workflow
- **THEN** the lookup output still follows the normalized item-summary contract

#### Scenario: Unsupported provider selection fails explicitly
- **WHEN** runtime configuration selects a lookup provider value that is not supported
- **THEN** the system returns a configuration error identifying the unsupported provider
- **THEN** the lookup workflow is not executed with an implicit fallback to another provider

### Requirement: System fails lookup explicitly when no usable external result is available
The system SHALL treat lookup as a non-mutating read operation with explicit failure semantics. If the provider returns no matching items, is unreachable, or returns an invalid response that cannot be normalized, the system MUST fail the lookup request with a user-facing error instead of returning partial or ambiguous success.

#### Scenario: Query matches no items
- **WHEN** the user submits a lookup query and the external provider returns no matching items
- **THEN** the system returns a lookup error indicating that no items matched the query

#### Scenario: Provider request fails
- **WHEN** the system cannot complete the external provider request because of a timeout, network error, or upstream error response
- **THEN** the system returns a lookup error describing the provider failure

#### Scenario: Provider rate-limits the request
- **WHEN** the external provider responds with `429 Too Many Requests`
- **THEN** the system retries the lookup request with backoff, honoring `Retry-After` when it is present
- **THEN** if the provider still rate-limits the request after the bounded retries, the system returns a lookup error describing the rate-limit failure

#### Scenario: Provider payload cannot be normalized
- **WHEN** the external provider returns a response that does not contain the required fields for normalized item summaries
- **THEN** the system returns a lookup error indicating that the provider response is invalid
