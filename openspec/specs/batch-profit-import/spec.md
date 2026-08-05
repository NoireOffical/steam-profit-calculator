## Purpose

Define the expected behavior for importing forward-calculation scenarios from a structured CSV file.

## Requirements

### Requirement: Import forward-calculation scenarios from CSV
The system SHALL support batch import of forward-calculation scenarios from a dedicated CSV input file. The input file MUST use the header `item_name,buy_price,sell_price,quantity,multiplier`. Each data row MUST describe exactly one price scenario and MUST produce exactly one calculated result. Empty `quantity` cells MUST default to `1`. Empty `multiplier` cells MUST default to `0.837`.

#### Scenario: Multiple valid rows produce multiple imported results
- **WHEN** the user imports a CSV file with rows `Capsule,0.48,0.7,5,` and `Capsule,0.48,0.8,5,0.85`
- **THEN** the system returns two calculation results
- **THEN** the first result uses quantity `5` and multiplier `0.837`
- **THEN** the second result uses quantity `5` and multiplier `0.85`
- **THEN** the system can persist both imported results through the standard record store

#### Scenario: Empty optional cells use default values
- **WHEN** the user imports a CSV row `Capsule,0.48,0.8,,`
- **THEN** the system treats quantity as `1`
- **THEN** the system treats multiplier as `0.837`
- **THEN** the imported result uses the same formulas as a normal forward calculation request

### Requirement: Reject malformed import files atomically
The system SHALL reject an import file when the header is missing or unsupported, when a row cannot be parsed into a valid scenario, or when any row violates forward-calculation validation rules. If any row is invalid, the system MUST fail the import without appending partial imported results to the record store.

#### Scenario: Invalid numeric value causes the whole import to fail
- **WHEN** the user imports a CSV file where one row contains sell price `abc`
- **THEN** the system returns an import error
- **THEN** the system does not append any imported records from that file

#### Scenario: Invalid semantic value causes the whole import to fail
- **WHEN** the user imports a CSV file where one row contains quantity `0`
- **THEN** the system returns an import error
- **THEN** the system does not append any imported records from that file

#### Scenario: Unsupported header is rejected
- **WHEN** the user imports a CSV file whose header does not match `item_name,buy_price,sell_price,quantity,multiplier`
- **THEN** the system returns an import error
- **THEN** the system does not append any imported records from that file
