## 1. Model And Tool Boundaries

- [x] 1.1 Add an injectable model interface, unavailable fallback, and optional OpenAI Responses API adapter
- [x] 1.2 Define strict schemas for lookup, forward calculation, and reverse calculation tools
- [x] 1.3 Execute tools through existing lookup and calculation packages without persistence side effects
- [x] 1.4 Version the assistant prompt and expose model settings and usage metadata

## 2. Grounded Assistant Workflow

- [x] 2.1 Add a bounded model/tool loop with structured evidence IDs
- [x] 2.2 Reject uncited or unsupported numeric model prose with a safe fallback
- [x] 2.3 Add status and SSE assistant routes with stable lifecycle and error events
- [x] 2.4 Keep disabled, misconfigured, and failed model states isolated from existing workflows

## 3. Browser Application

- [x] 3.1 Add a responsive assistant conversation panel and example prompts
- [x] 3.2 Render streaming status, answer chunks, trusted evidence cards, and completion metadata
- [x] 3.3 Provide accessible disabled, loading, empty, and error states without browser persistence

## 4. Evaluation And Documentation

- [x] 4.1 Add the first 30 fixed tool-evaluation cases and a deterministic runner
- [x] 4.2 Add provider, orchestration, grounding, tool, HTTP, and no-side-effect failure tests
- [x] 4.3 Document AI configuration, privacy, limitations, API events, and model choice
- [x] 4.4 Update the AI full-stack roadmap and workload estimate

## 5. Verification

- [x] 5.1 Run formatting, JavaScript syntax validation, static analysis, race tests, and binary builds
- [x] 5.2 Smoke-test deterministic workflows with AI disabled
- [x] 5.3 Smoke-test the assistant stream with an injected fake provider
