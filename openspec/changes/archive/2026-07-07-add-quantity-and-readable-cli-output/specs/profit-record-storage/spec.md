## MODIFIED Requirements

### Requirement: Persist one CSV record per calculation result
The system SHALL persist each calculation result as a separate CSV record so that multiple items and multiple sell-price scenarios can be saved over time. Each record MUST include the item name, buy price, sell price, quantity, net profit, total profit, and knife ratio.

#### Scenario: Multiple sell prices are saved as multiple rows
- **WHEN** the user adds one item with buy price `100`, quantity `3`, and sell prices `120` and `135`
- **THEN** the system appends two CSV records
- **THEN** each record contains the same item name, buy price, and quantity
- **THEN** each record contains its own sell price, net profit, total profit, and knife ratio

#### Scenario: Saved records remain after later writes
- **WHEN** the system already contains saved records for one item and the user adds another item
- **THEN** the system appends new CSV records without removing existing records
- **THEN** previously saved records remain available for listing

### Requirement: Initialize and read the CSV record store
The system SHALL create a CSV file with a stable header when the record store does not yet exist, and it SHALL read saved records back for listing. The expanded header MUST be `item_name,buy_price,sell_price,quantity,net_profit,total_profit,knife_ratio`.

#### Scenario: First write creates the expanded CSV file
- **WHEN** the user saves calculation results and the CSV file does not exist yet
- **THEN** the system creates the CSV file
- **THEN** the system writes the expanded CSV header before appending data rows

#### Scenario: List returns expanded saved records
- **WHEN** the user requests the saved record list after one or more records have been stored in the expanded schema
- **THEN** the system returns all saved records from the CSV file
- **THEN** each returned record includes the persisted item name, buy price, sell price, quantity, net profit, total profit, and knife ratio

### Requirement: Reject malformed stored data
The system SHALL fail with a clear read error when the CSV file contains rows that cannot be parsed into valid saved records and do not match a supported schema.

#### Scenario: Malformed CSV row causes read failure
- **WHEN** the user requests the saved record list and the CSV file contains an invalid row shape or an unparsable numeric value that is not part of a supported schema
- **THEN** the system returns a read error
- **THEN** the system does not silently skip the malformed row

## ADDED Requirements

### Requirement: Preserve compatibility with legacy CSV files
The system SHALL continue to read legacy five-column record stores with the header `item_name,buy_price,sell_price,net_profit,knife_ratio`. When reading a legacy row, the system MUST treat quantity as `1` and total profit as the same value as net profit. When appending new records to a legacy file, the system MUST migrate the file to the expanded schema before writing new rows.

#### Scenario: Listing a legacy file derives missing fields
- **WHEN** the user requests the saved record list for a legacy CSV file
- **THEN** the system returns records with quantity `1`
- **THEN** the system returns total profit equal to net profit for those legacy rows

#### Scenario: Appending to a legacy file upgrades the header
- **WHEN** the user appends new records to a legacy CSV file
- **THEN** the system rewrites existing rows into the expanded schema
- **THEN** the system writes the expanded header before appending the new records
