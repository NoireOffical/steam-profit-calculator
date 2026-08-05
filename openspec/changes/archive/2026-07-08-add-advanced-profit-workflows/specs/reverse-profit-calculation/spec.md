## ADDED Requirements

### Requirement: Solve target-driven buy or sell prices
The system SHALL support reverse profit calculation for one item at a time. A reverse-calculation request MUST include item name, solve direction (`buy` or `sell`), exactly one known price, exactly one target metric, an optional quantity, and an optional multiplier. If quantity is omitted, the system MUST treat it as `1`. If multiplier is omitted, the system MUST treat it as `0.837`. The system MUST return a standard calculation result containing the solved price, quantity, net profit, total profit, and knife ratio.

#### Scenario: Solve required sell price from a total-profit target
- **WHEN** the user requests item name `Capsule`, solve direction `sell`, buy price `0.48`, quantity `5`, multiplier `0.85`, and target total profit `1.0`
- **THEN** the system returns a solved sell price `0.8`
- **THEN** the system returns net profit `0.2`
- **THEN** the system returns total profit `1.0`
- **THEN** the system returns knife ratio `0.705882`

#### Scenario: Solve acceptable buy price from a knife-ratio target
- **WHEN** the user requests item name `Capsule`, solve direction `buy`, sell price `0.8`, omits quantity, omits multiplier, and target knife ratio `0.7`
- **THEN** the system returns a solved buy price `0.46872`
- **THEN** the system returns net profit `0.20088`
- **THEN** the system returns total profit `0.20088`
- **THEN** the system returns knife ratio `0.7`

#### Scenario: Solve acceptable buy price from a net-profit target
- **WHEN** the user requests item name `Capsule`, solve direction `buy`, sell price `0.8`, quantity `2`, multiplier `0.85`, and target net profit `0.15`
- **THEN** the system returns a solved buy price `0.53`
- **THEN** the system returns total profit `0.3`
- **THEN** the system returns knife ratio `0.779412`

### Requirement: Reject invalid reverse-calculation input
The system SHALL reject a reverse-calculation request when the item name is empty, the solve direction is missing or unsupported, the known price required by the solve direction is missing or less than or equal to zero, the quantity is less than or equal to zero, the multiplier is less than or equal to zero or greater than `1`, no target is provided, more than one target is provided, or the target knife ratio is less than or equal to zero.

#### Scenario: Missing known price for solve direction is rejected
- **WHEN** the user requests solve direction `sell` with a target net profit but omits buy price
- **THEN** the system returns a validation error
- **THEN** the system does not create a solved record

#### Scenario: Multiple targets are rejected
- **WHEN** the user requests solve direction `buy` with both target net profit and target knife ratio
- **THEN** the system returns a validation error
- **THEN** the system does not create a solved record

#### Scenario: Invalid multiplier is rejected
- **WHEN** the user requests reverse calculation with multiplier `1.2`
- **THEN** the system returns a validation error
- **THEN** the system does not create a solved record
