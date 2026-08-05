## MODIFIED Requirements

### Requirement: Calculate profit metrics for each sell price
The system SHALL accept one item name, one buy price, an optional quantity, an optional multiplier, and one or more sell prices, and it SHALL generate one calculation result for each sell price. If quantity is omitted, the system MUST treat it as `1`. If multiplier is omitted, the system MUST treat it as `0.837`. Each result MUST include quantity, compute per-item net profit as `sell_price * multiplier - buy_price`, compute total profit as `net_profit * quantity`, and compute knife ratio as `buy_price / (sell_price * multiplier)`.

#### Scenario: Multiple sell prices produce multiple quantity-aware results
- **WHEN** the user submits item name `AK-47 Redline`, buy price `100`, quantity `3`, omits multiplier, and sell prices `120` and `135`
- **THEN** the system returns two calculation results for the same item
- **THEN** the first result uses sell price `120`, quantity `3`, net profit `0.44`, total profit `1.32`, and knife ratio `0.995619`
- **THEN** the second result uses sell price `135`, quantity `3`, net profit `12.995`, total profit `38.985`, and knife ratio `0.884995`

#### Scenario: Omitted quantity defaults to one
- **WHEN** the user submits item name `Capsule`, buy price `0.48`, omits quantity, omits multiplier, and provides sell price `0.8`
- **THEN** the system treats quantity as `1`
- **THEN** the returned result uses total profit equal to the per-item net profit

#### Scenario: Omitted multiplier defaults to Steam net multiplier
- **WHEN** the user submits item name `Capsule`, buy price `0.48`, quantity `2`, omits multiplier, and provides sell price `0.8`
- **THEN** the system treats multiplier as `0.837`
- **THEN** the returned result uses net profit `0.19`
- **THEN** the returned result uses total profit `0.379`
- **THEN** the returned result uses knife ratio `0.716846`

#### Scenario: Explicit multiplier changes the calculation model
- **WHEN** the user submits item name `Capsule`, buy price `0.48`, quantity `2`, multiplier `0.85`, and sell price `0.8`
- **THEN** the system returns a calculation result with net profit `0.2`
- **THEN** the system returns total profit `0.4`
- **THEN** the system returns knife ratio `0.705882`

#### Scenario: Negative profit is still a valid result
- **WHEN** the user submits buy price `100`, quantity `2`, omits multiplier, and sell price `110`
- **THEN** the system returns a calculation result with net profit `-7.93`
- **THEN** the system returns total profit `-15.86`
- **THEN** the system does not reject the result only because profit is negative

### Requirement: Reject invalid calculation input
The system SHALL reject a calculation request when the item name is empty, the buy price is less than or equal to zero, the quantity is less than or equal to zero, the multiplier is less than or equal to zero or greater than `1`, no sell prices are provided, or any sell price is less than or equal to zero.

#### Scenario: Empty item name is rejected
- **WHEN** the user submits an empty item name with a valid buy price, quantity, multiplier, and sell prices
- **THEN** the system returns a validation error
- **THEN** the system does not create calculation results

#### Scenario: Non-positive buy price is rejected
- **WHEN** the user submits buy price `0` with a valid item name, quantity, multiplier, and sell prices
- **THEN** the system returns a validation error
- **THEN** the system does not create calculation results

#### Scenario: Non-positive quantity is rejected
- **WHEN** the user submits quantity `0` with a valid item name, buy price, multiplier, and sell prices
- **THEN** the system returns a validation error
- **THEN** the system does not create calculation results

#### Scenario: Invalid multiplier is rejected
- **WHEN** the user submits multiplier `1.2` with a valid item name, buy price, quantity, and sell prices
- **THEN** the system returns a validation error
- **THEN** the system does not create calculation results

#### Scenario: Missing sell prices are rejected
- **WHEN** the user submits a valid item name, buy price, quantity, and multiplier but no sell prices
- **THEN** the system returns a validation error
- **THEN** the system does not create calculation results

#### Scenario: Non-positive sell price is rejected
- **WHEN** the user submits sell prices `120` and `0`
- **THEN** the system returns a validation error
- **THEN** the system does not create partial calculation results
