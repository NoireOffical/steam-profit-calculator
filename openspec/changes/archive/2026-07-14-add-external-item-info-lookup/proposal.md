## Why

The current tool requires users to already know the exact item name and to collect item metadata manually before running calculations. As the record set grows, manual searching and copy-pasting make it harder to confirm canonical item names and basic item details, while still offering no in-tool way to verify that the target item actually exists in an external catalog.

## What Changes

- Add a new external item-info lookup workflow that queries a third-party API for item identity plus selected market summary fields by keyword or item name.
- Add a CLI entrypoint for item lookup so users can search for candidate items before running `add`, `reverse`, or `import`.
- Normalize provider responses into a project-owned item info model so the CLI output does not depend on raw upstream response shapes.
- Keep the current profit calculation, reverse calculation, batch import, and CSV record storage workflows unchanged.
- Return explicit lookup errors for network failures, upstream errors, and empty-result cases without writing any calculation records.

## Capabilities

### New Capabilities
- `external-item-info-lookup`: Query an external provider for normalized item identity and selected market summary fields, then present item search results to the user.

### Modified Capabilities
- `profit-cli-interface`: The CLI requirements must expand to cover an item-info lookup command and its success and failure behaviors.

## Impact

- New external API client logic, response normalization, and error handling.
- CLI changes to expose a lookup workflow alongside the existing calculation commands.
- New tests for provider integration boundaries using mocks or stubbed responses.
- A new external dependency on a third-party item information API, including timeout and availability considerations.
