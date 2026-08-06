## Context

The repository already separates deterministic calculation, external lookup, display, and storage concerns. The next step toward an AI full-stack application is to expose those capabilities through a network boundary while preserving their current behavior and testability.

This change intentionally stops before model integration, user authentication, or database migration. It establishes the smallest complete browser-to-server slice that future AI tools can call without teaching a model to reproduce financial formulas.

## Goals / Non-Goals

**Goals:**
- Provide a runnable browser application backed by a Go JSON API.
- Keep all numerical results owned by the existing calculation package.
- Keep item lookup behind the existing provider interface.
- Keep persisted history on the server and outside browser storage.
- Make API behavior testable without listening on a real network port.
- Embed browser assets into the server binary for a single-artifact local demo.

**Non-Goals:**
- Integrating an LLM or choosing a model vendor.
- Adding authentication, multi-user ownership, or authorization.
- Replacing CSV persistence with a relational database.
- Deploying the application to a public environment.
- Turning the browser into a second source of business rules.

## Decisions

### 1. Add a separate server binary

The existing `steam-profit` CLI remains unchanged. A new `steam-profit-server` binary owns HTTP lifecycle concerns and constructs the same lookup and storage adapters used by the CLI.

### 2. Use a versioned same-origin JSON API

The first API surface uses `/api/v1` routes and explicit JSON request/response types. The browser is served by the same process, so the initial application needs no cross-origin policy or separate frontend deployment.

Endpoints:
- `GET /api/health`
- `GET /api/v1/items?query=...&limit=...`
- `POST /api/v1/calculations`
- `POST /api/v1/reverse-calculations`
- `GET /api/v1/records`

### 3. Reuse deterministic domain services

HTTP handlers translate JSON into existing domain request values, call the current calculation or lookup package, and translate results back to JSON. They must not reimplement profit formulas or provider normalization.

### 4. Keep persistence server-side behind a narrow interface

Handlers depend on an append/list record-store interface implemented by the existing CSV store. This supports isolated tests today and a later relational implementation without changing API handlers.

### 5. Embed a dependency-free browser interface

The initial frontend uses semantic HTML, CSS, and browser JavaScript embedded with the Go server. This keeps the first vertical slice small, removes a second dependency toolchain, and provides a concrete UI that can later be replaced without changing the API.

### 6. Reserve AI as the next bounded layer

No endpoint will fabricate recommendations in this change. The versioned calculation and lookup contracts become the trusted tools for a later assistant, and AI responses will be required to reference tool-produced values.

## Risks / Trade-offs

- [CSV is not safe for multi-instance or multi-user deployment] -> Keep the first server local-only and plan a database migration before public hosting.
- [A dependency-free frontend has less component structure than a framework application] -> Keep state and interactions small; preserve the API contract so a framework migration is additive.
- [The Steam public provider can be rate limited] -> Preserve existing timeout and bounded retry behavior and surface provider failures explicitly.
- [New HTTP input expands the attack surface] -> Limit request bodies, reject unknown JSON fields, set response security headers, and avoid cross-origin access in the first version.

## Migration Plan

The change is additive. Existing CLI commands and CSV files remain valid. The server defaults to the same record filename and can be removed without migrating persisted data.
