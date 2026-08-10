## ADDED Requirements

### Requirement: The repository landing page presents a sanitized demonstrable workflow

The bilingual repository README SHALL embed a bounded animation that demonstrates multi-price calculation and persisted SQLite history using deterministic local data. The animation MUST NOT contain credentials, API keys, provider payloads, prompts, model output, account information, or real trading records.

#### Scenario: A reviewer visits the GitHub repository

- **WHEN** GitHub renders either README language
- **THEN** the reviewer can see the calculation-to-history workflow without running the service
- **AND** the preview explains that deterministic local data and production application paths were used

### Requirement: Full-resolution video is release-backed

The README preview SHALL link to a browser-compatible H.264 MP4 attached to the matching immutable milestone Pre-release. The MP4 SHALL NOT be committed to repository history, and the release notes SHALL identify the demonstrated workflow, verification, limitations, and compatibility impact.

#### Scenario: A reviewer requests the full demonstration

- **WHEN** the reviewer activates the README animation link
- **THEN** GitHub serves the full-resolution recording from the matching milestone Release
- **AND** the repository clone remains bounded to the lightweight preview asset
