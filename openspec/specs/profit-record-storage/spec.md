## Purpose

Define how Steam profit calculation results are persisted to CSV, migrated from legacy files, and read back for listing.

## Requirements

### Requirement: Persist one CSV record per calculation result
The system SHALL persist each calculation result as a separate CSV record so that multiple items and multiple sell-price scenarios can be saved over time. Each record MUST include the item name, buy price, sell price, quantity, net profit, total profit, knife ratio, and fee.

#### Scenario: Multiple sell prices are saved as multiple rows
- **WHEN** the user adds one item with buy price `100`, quantity `3`, and sell prices `120` and `135`
- **THEN** the system appends two CSV records
- **THEN** each record contains the same item name, buy price, and quantity
- **THEN** each record contains its own sell price, net profit, total profit, knife ratio, and fee

#### Scenario: Default multiplier persists fee as default
- **WHEN** the user saves a calculation result that uses the default multiplier
- **THEN** the persisted CSV row stores `fee` as `default`

#### Scenario: Explicit multiplier persists fee as the concrete multiplier
- **WHEN** the user saves a calculation result with explicit multiplier `0.85`
- **THEN** the persisted CSV row stores `fee` as `0.850`

#### Scenario: Saved records remain after later writes
- **WHEN** the system already contains saved records for one item and the user adds another item
- **THEN** the system appends new CSV records without removing existing records
- **THEN** previously saved records remain available for listing

### Requirement: Initialize and read the CSV record store
The system SHALL create a CSV file with a stable header when the record store does not yet exist, and it SHALL read saved records back for listing. The current storage header MUST be `item_name,buy_price,sell_price,quantity,net_profit,total_profit,knife_ratio,fee`.

#### Scenario: First write creates the current CSV file
- **WHEN** the user saves calculation results and the CSV file does not exist yet
- **THEN** the system creates the CSV file
- **THEN** the system writes the current eight-column CSV header before appending data rows

#### Scenario: List returns current saved records
- **WHEN** the user requests the saved record list after one or more records have been stored in the current schema
- **THEN** the system returns all saved records from the CSV file
- **THEN** each returned record includes the persisted item name, buy price, sell price, quantity, net profit, total profit, knife ratio, and fee

### Requirement: Reject malformed stored data
The system SHALL fail with a clear read error when the CSV file contains rows that cannot be parsed into valid saved records and do not match a supported schema.

#### Scenario: Malformed CSV row causes read failure
- **WHEN** the user requests the saved record list and the CSV file contains an invalid row shape or an unparsable numeric value that is not part of a supported schema
- **THEN** the system returns a read error
- **THEN** the system does not silently skip the malformed row

### Requirement: Preserve compatibility with historical CSV files
The system SHALL continue to read legacy five-column record stores with the header `item_name,buy_price,sell_price,net_profit,knife_ratio`. The system SHALL also continue to read the intermediate seven-column expanded schema with the header `item_name,buy_price,sell_price,quantity,net_profit,total_profit,knife_ratio`. When reading a legacy five-column row, the system MUST treat quantity as `1`, total profit as the same value as net profit, and fee as `default`. When reading a seven-column row without fee, the system MUST derive the missing fee label for in-memory results. When appending new records to a five-column or seven-column file, the system MUST migrate the file to the current eight-column schema before writing new rows.

#### Scenario: Listing a legacy five-column file derives missing fields
- **WHEN** the user requests the saved record list for a legacy five-column CSV file
- **THEN** the system returns records with quantity `1`
- **THEN** the system returns total profit equal to net profit for those legacy rows
- **THEN** the system returns fee as `default` for those legacy rows

#### Scenario: Listing a seven-column file derives the missing fee label
- **WHEN** the user requests the saved record list for a seven-column expanded CSV file without `fee`
- **THEN** the system returns records using the stored item name, buy price, sell price, quantity, net profit, total profit, and knife ratio
- **THEN** the system derives and returns a fee label for each record

#### Scenario: Appending to a historical file upgrades the header
- **WHEN** the user appends new records to a five-column or seven-column CSV file
- **THEN** the system rewrites existing rows into the current eight-column schema
- **THEN** the system writes the current eight-column header before appending the new records
