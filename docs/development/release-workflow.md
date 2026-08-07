# Milestone release workflow

The project publishes every demonstrable milestone as an immutable Git tag and a GitHub Pre-release. This preserves reproducible portfolio checkpoints while the application is still local-first and not ready for public operation.

## Release gate

A milestone can be released only when all of the following are true:

1. The milestone has a runnable demonstration path.
2. Its OpenSpec change is valid and all planned tasks are complete.
3. `make check` passes, including JavaScript syntax, static analysis, race-enabled tests, and both binary builds.
4. A smoke test covers the affected user workflow and its safe failure or disabled state.
5. Release notes describe capabilities, verification, compatibility, and known limitations.

## Version and tag policy

- Use annotated Semantic Version tags: `vMAJOR.MINOR.PATCH`.
- A published tag identifies exactly one commit and is permanent.
- Never force-push, move, delete, or reuse a published milestone tag.
- If release notes or code need correction after publication, create a new patch version.
- Use a minor version for each new demonstrable capability milestone while the project remains pre-1.0.

## GitHub release policy

Every milestone tag must have a corresponding GitHub Release with the **Pre-release** flag enabled. Use `docs/releases/<tag>.md` as the release body and include:

- what can now be demonstrated;
- important architecture or API additions;
- verification performed;
- compatibility or migration impact;
- known limitations and the next production gap.

Pre-release status is required until the public-operation checklist covers durable database storage, user/data isolation, rate limiting, secret handling, production observability, backups, and rollback.

## Release sequence

1. Complete and validate the OpenSpec change.
2. Run `make check` and the milestone smoke tests.
3. Update `CHANGELOG.md` and `docs/releases/<tag>.md`.
4. Commit and push the milestone to `main`.
5. Create an annotated tag on the exact milestone commit and push the tag.
6. Create a GitHub Release from that tag and enable **Pre-release**.
7. Verify the remote branch, tag target, release URL, and Pre-release state.

## Current milestone map

| Tag | Commit scope | GitHub release type |
| --- | --- | --- |
| `v0.1.0` | Web application foundation | Pre-release |
| `v0.2.0` | AI strategy assistant MVP | Pre-release |
| `v0.3.0` | Versioned SQLite data layer and history explorer | Pre-release |
| `v0.4.0` | Deployable demo, browser E2E, and safe runtime observability | Pre-release |
