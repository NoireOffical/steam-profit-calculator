## MODIFIED Requirements

### Requirement: System queries an external provider for normalized item summaries
The system SHALL support querying an external item-information provider by user-supplied item keyword or item name. For each matched item, the system MUST return a normalized item summary that includes `item_id`, `canonical_name`, `app_name`, `query_time`, `sale_price_text`, `sell_price_text`, and `sell_listings`.

The selected lookup provider MUST be chosen from configuration rather than from CLI query flags. If no explicit provider is configured, the system MUST default to the public Steam provider.

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
