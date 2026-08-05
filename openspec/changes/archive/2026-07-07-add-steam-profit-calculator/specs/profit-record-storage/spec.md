## ADDED Requirements

### Requirement: Persist one CSV record per calculation result
The system SHALL persist each calculation result as a separate CSV record so that multiple items and multiple sell-price scenarios can be saved over time. Each record MUST include the item name, buy price, sell price, net profit, and knife ratio.

#### Scenario: Multiple sell prices are saved as multiple rows
- **WHEN** the user adds one item with buy price `100` and sell prices `120` and `135`
- **THEN** the system appends two CSV records
- **THEN** each record contains the same item name and buy price
- **THEN** each record contains the sell price and calculated metrics for its own scenario

#### Scenario: Saved records remain after later writes
- **WHEN** the system already contains saved records for one item and the user adds another item
- **THEN** the system appends new CSV records without removing existing records
- **THEN** previously saved records remain available for listing

### Requirement: Initialize and read the CSV record store
The system SHALL create a CSV file with a stable header when the record store does not yet exist, and it SHALL read saved records back for listing.

#### Scenario: First write creates the CSV file
- **WHEN** the user saves calculation results and the CSV file does not exist yet
- **THEN** the system creates the CSV file
- **THEN** the system writes the CSV header before appending data rows

#### Scenario: List returns saved records
- **WHEN** the user requests the saved record list after one or more records have been stored
- **THEN** the system returns all saved records from the CSV file
- **THEN** each returned record includes the persisted item name, buy price, sell price, net profit, and knife ratio

### Requirement: Reject malformed stored data
The system SHALL fail with a clear read error when the CSV file contains rows that cannot be parsed into valid saved records.

#### Scenario: Malformed CSV row causes read failure
- **WHEN** the user requests the saved record list and the CSV file contains an invalid row shape or an unparsable numeric value
- **THEN** the system returns a read error
- **THEN** the system does not silently skip the malformed row
