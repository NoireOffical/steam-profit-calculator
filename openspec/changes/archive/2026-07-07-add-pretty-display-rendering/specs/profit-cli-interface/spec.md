## MODIFIED Requirements

### Requirement: CLI displays readable record rows
The CLI SHALL support explicit output-mode selection for record display. Both `add` and `list` commands MUST support a display selector, and the default display mode MUST use the pretty table renderer. The CLI MUST still be able to emit CSV-style stdout output when the CSV display mode is explicitly selected.

#### Scenario: Add command defaults to pretty table output
- **WHEN** the user successfully runs `add` without specifying an output mode
- **THEN** the CLI prints records using the pretty table renderer

#### Scenario: List command defaults to pretty table output
- **WHEN** the user successfully runs `list` without specifying an output mode
- **THEN** the CLI prints records using the pretty table renderer

#### Scenario: User selects CSV display mode
- **WHEN** the user runs `add` or `list` with the CSV display mode selected
- **THEN** the CLI prints records using the CSV-style display renderer
- **THEN** the CLI does not change how records are stored on disk
