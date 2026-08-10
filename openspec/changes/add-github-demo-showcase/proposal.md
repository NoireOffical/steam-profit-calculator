## Why

The repository already contains a controlled hosted demo and static screenshots, but a GitHub visitor cannot see the core calculation-to-persistence workflow without running the project or receiving reviewer credentials. A short sanitized recording makes the demonstrable path visible directly on the repository landing page without exposing secrets or coupling the README to external providers.

## What Changes

- Add a lightweight animated walkthrough to the bilingual README.
- Link the preview to a full-resolution H.264 MP4 stored as an immutable milestone Release asset.
- Record how the demo was produced, what production paths it exercises, and which data it intentionally excludes.
- Publish reusable `v0.7.0` release notes with verification, limitations, and compatibility impact.

## Capabilities

### Added Capabilities

- `github-demo-showcase`: Present a sanitized, release-backed demonstration of the browser calculation and SQLite history workflow.

## Impact

- Documentation and generated media assets only; no runtime, formula, provider, API, persistence, or CLI behavior changes.
- Adds one bounded GIF to Git history and one MP4 to the GitHub Pre-release rather than repository history.
