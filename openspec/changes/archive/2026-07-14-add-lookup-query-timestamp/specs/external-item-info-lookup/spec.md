## MODIFIED Requirements

### Requirement: System queries an external provider for normalized item summaries
The system SHALL support querying an external item-information provider by user-supplied item keyword or item name. For each matched item, the system MUST return a normalized item summary that includes `item_id`, `canonical_name`, `app_name`, `query_time`, `sale_price_text`, `sell_price_text`, and `sell_listings`.

#### Scenario: Query returns multiple matching items
- **WHEN** the user submits a lookup query that matches multiple items in the external provider
- **THEN** the system returns one normalized item summary per matched item
- **THEN** each returned summary includes `item_id`, `canonical_name`, `app_name`, `query_time`, `sale_price_text`, `sell_price_text`, and `sell_listings`

#### Scenario: Query stamps one shared observation time
- **WHEN** the system returns multiple items from a single lookup request
- **THEN** all returned item summaries share the same `query_time`
- **THEN** `query_time` is formatted as a UTC RFC3339 timestamp
