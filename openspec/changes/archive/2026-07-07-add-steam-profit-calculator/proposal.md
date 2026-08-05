## Why

This project needs a small but complete OpenSpec experiment that is easy to specify, implement, and test with very high coverage. A Steam knife-arbitrage calculator fits that goal because its core formulas are fixed, the input/output behavior is explicit, and the persistence requirement is small enough to keep the first change focused.

## What Changes

- Add a Go CLI for recording Steam item profit calculations.
- Allow users to provide one item name, one buy price, and multiple sell prices in a single command.
- Calculate net profit for each sell price using `sell_price * 0.837 - buy_price`.
- Calculate knife ratio for each sell price using `buy_price / (sell_price * 0.837)`.
- Save each calculated result as a separate CSV row so that multiple items and multiple sell-price scenarios can be stored over time.
- Add a command to list previously saved calculation records.
- Validate required inputs and reject invalid prices or empty item names with clear errors.

## Capabilities

### New Capabilities
- `steam-profit-calculation`: Calculate net profit and knife ratio for one item buy price against one or more sell prices.
- `profit-record-storage`: Persist and list Steam profit calculation records in CSV format.

### Modified Capabilities

None.

## Impact

- New Go module and CLI entrypoint.
- New calculation package for formula validation and result shaping.
- New CSV-backed storage package for appending and listing saved records.
- New unit tests covering calculation rules, validation, CSV encoding/decoding, and CLI behavior.
