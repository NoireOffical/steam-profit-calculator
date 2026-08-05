## MODIFIED Requirements

### Requirement: CLI provides a lookup command for external item information
Successful lookup output MUST use the normalized field order `item_id,canonical_name,app_name,query_time,sale_price_text,sell_price_text,sell_listings`.

#### Scenario: Lookup output includes the observation time
- **WHEN** the user runs `lookup` successfully
- **THEN** each lookup row includes a `query_time` column
- **THEN** the value records when the market price snapshot was observed
