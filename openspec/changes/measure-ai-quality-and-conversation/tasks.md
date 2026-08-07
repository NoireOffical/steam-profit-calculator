## 1. Evaluation Foundation

- [x] 1.1 Expand the deterministic tool fixture from 30 to exactly 50 success, invalid-input, dependency-failure, and boundary cases
- [x] 1.2 Add a versioned semantic suite and a report engine for expected tools, grounding, latency, token use, and optional caller-priced cost
- [x] 1.3 Add an opt-in live evaluation command with current/baseline prompt selection, bounded case count, JSON output, and threshold exit behavior
- [x] 1.4 Add CI-safe scripted tests proving report success, threshold failure, provider failure, and content-free output

## 2. Provider Streaming And Conversation Contract

- [x] 2.1 Add bounded provider-neutral history messages and preserve the existing message-only request contract
- [x] 2.2 Parse OpenAI typed streaming text, function-call argument, completed, failed, and error events with bounded input
- [x] 2.3 Preserve provider text-delta boundaries only after the complete answer passes current-turn grounding
- [x] 2.4 Propagate cancellation and classify safe retryable failures without automatic paid retries

## 3. Browser Experience

- [x] 3.1 Send at most six transient prior messages and keep chat state page-local
- [x] 3.2 Add accessible Stop and explicit Retry controls with clear cancelled, failed, and active states
- [x] 3.3 Display prompt version, native-stream metadata, grounding, latency, tool count, and token use without exposing sensitive content

## 4. Verification And Documentation

- [x] 4.1 Add service, OpenAI adapter, HTTP, browser, and no-side-effect tests for success, invalid history, cancellation, dependency failure, and retry
- [x] 4.2 Run formatting, JavaScript syntax validation, vet, race tests, all three binary builds, deterministic evaluation, and Chromium E2E
- [x] 4.3 Run a manual disabled-provider smoke and document the opt-in live smoke path; no temporary key was available, so no paid result is claimed
- [x] 4.4 Update README, roadmap, demo runbook, changelog, and prepare immutable `v0.5.0` Pre-release notes
