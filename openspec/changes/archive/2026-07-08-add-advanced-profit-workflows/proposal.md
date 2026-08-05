## Why

The current tool is good at forward calculation from explicit buy and sell inputs, but it still assumes a fixed market multiplier and mostly supports one-off interactive-style usage. The next evolution should make it useful for decision-making and batch processing by adding target-driven reverse calculation, configurable fee handling, and bulk import workflows.

## What Changes

- Add reverse calculation workflows that solve for target sell price or acceptable buy price from user-provided profit or knife-ratio goals.
- Add configurable fee or multiplier input so the calculation model is no longer fixed to a single hard-coded market assumption.
- Add batch import support so multiple items and price scenarios can be processed from an input file instead of only through repeated command flags.
- Keep the existing forward calculation flow available, while extending the CLI into multiple analysis workflows.
- Preserve existing CSV record storage as the output record system, while allowing imported input data to come from a separate structured file.

## Capabilities

### New Capabilities
- `reverse-profit-calculation`: Solve target-driven pricing questions such as required sell price, acceptable buy price, or target ratio thresholds.
- `batch-profit-import`: Read multiple item scenarios from an import file and process them in one execution.

### Modified Capabilities
- `steam-profit-calculation`: Profit calculation must support configurable fee or multiplier input in addition to the current default model.
- `profit-cli-interface`: The CLI must expose reverse-calculation and batch-import workflows, not just direct `add` and `list` usage.

## Impact

- Changes to the calculation layer to parameterize fee assumptions and add reverse formulas.
- Changes to the CLI to support new workflow entrypoints and arguments.
- New import parsing logic and validation for batch input files.
- Potential updates to output rendering so imported and reverse-calculated results are easy to inspect.
- Expanded tests across calculation, CLI, and import handling.
