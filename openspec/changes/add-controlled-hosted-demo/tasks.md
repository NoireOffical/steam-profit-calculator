## 1. Contract And Configuration

- [x] 1.1 Add and validate optional demo authentication, API rate-limit, assistant quota, and trusted-proxy environment configuration
- [x] 1.2 Protect the browser and application APIs while keeping dependency health public and existing local defaults compatible
- [x] 1.3 Add aggregate unauthorized, rate-limited, and assistant-quota metrics without sensitive labels or log fields

## 2. Enforcement And Tests

- [x] 2.1 Add constant-time Basic authentication tests for disabled, success, missing, wrong, and partial-configuration paths
- [x] 2.2 Add bounded fixed-window API-limit tests for client isolation, retry timing, trusted-proxy behavior, eviction, and no-side-effect rejection
- [x] 2.3 Add assistant quota tests proving invalid/unavailable requests do not consume budget and started/cancelled/failed provider work does
- [x] 2.4 Extend browser E2E to verify the protected demo path without calling Steam or OpenAI

## 3. Hosted Operations

- [x] 3.1 Add an explicit manual current/baseline live-model workflow with bounded cases, secret checks, content-free reports, and artifact retention
- [x] 3.2 Add a hosted synthetic smoke script and scheduled/manual workflow that performs read-only health and authenticated status checks
- [x] 3.3 Document the vendor-neutral one-replica HTTPS, secret injection, persistent volume, backup, restore, and immutable rollback contract
- [x] 3.4 Select Railway after user approval and add its deployment manifest without creating an unapproved paid persistent-storage resource

## 4. Verification And Release

- [x] 4.1 Run formatting, JavaScript syntax validation, vet, race tests, all binaries, deterministic evaluation, and Chromium E2E
- [ ] 4.2 Run remote container validation for protected startup, health, authentication, rate limiting, quota, restart persistence, and graceful stop
- [ ] 4.3 Run a manual hosted smoke when a target exists; do not claim a live deployment or real-model baseline without credentials and evidence
- [ ] 4.4 Update README, roadmap, demo runbook, changelog, and prepare immutable `v0.6.0` Pre-release notes after the demonstrable milestone is complete
