## 1. HTTP Application Boundary

- [x] 1.1 Add a testable HTTP handler with explicit JSON success and error contracts
- [x] 1.2 Expose health, lookup, forward calculation, reverse calculation, and record-history endpoints
- [x] 1.3 Persist successful calculations through the existing CSV store without changing the CLI

## 2. Browser Application

- [x] 2.1 Add an embedded responsive browser interface for lookup, calculation, and history
- [x] 2.2 Keep calculation rules on the server and render returned API records in the browser
- [x] 2.3 Provide accessible labels, loading states, empty states, and visible error feedback

## 3. Server Entrypoint And Developer Workflow

- [x] 3.1 Add a standalone server binary with configurable address and record path
- [x] 3.2 Add build and run targets without changing the existing CLI artifact
- [x] 3.3 Document local web usage, API routes, persistence limits, and the next AI milestone

## 4. Verification

- [x] 4.1 Add handler tests for methods, validation, calculations, persistence, history, lookup, and static assets
- [x] 4.2 Run formatting, static analysis, race tests, and both binary builds
- [x] 4.3 Manually verify the browser-to-server workflow locally
