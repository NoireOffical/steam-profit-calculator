## Why

The AI assistant can already choose trusted tools and produce a grounded single-turn answer, but its quality is not yet measurable against a live provider and the browser cannot continue, cancel, or retry a conversation. The provider also returns a complete response before the application emits synthetic chunks. The next portfolio milestone should make model behavior comparable and the interaction resilient without weakening the deterministic calculation or evidence boundaries.

## What Changes

- Expand the deterministic tool baseline from 30 to 50 cases and add a versioned semantic evaluation suite for tool selection, grounding, latency, tokens, and optional caller-supplied cost rates.
- Add an opt-in evaluation command that can run a live configured model, compare the current and baseline prompts, emit a machine-readable report, and fail on explicit thresholds without exposing credentials or arbitrary prompt content.
- Extend the provider-neutral model boundary and OpenAI adapter to consume typed Responses API streaming events, including text and streamed function-call arguments.
- Add bounded transient conversation history while preserving the existing `message` request field and keeping all chat content out of SQLite, CSV, logs, and metrics.
- Add browser stop and retry controls. Cancellation propagates through the HTTP request context, and retry replays only the visible transient turn through a new request.
- Preserve the grounding guard by buffering provider text until the complete answer passes evidence and numeric validation, then emitting the provider's original text-delta boundaries.

## Capabilities

### New Capabilities

- `ai-quality-workbench`: Measure model tool selection and grounded-answer behavior, compare prompt versions, and provide a bounded multi-turn browser workflow with cancellation, retry, and safe provider-native streaming.

### Modified Capabilities

- `ai-strategy-assistant`: Accept bounded transient history and expose safe streamed completion metadata while preserving read-only tools and deterministic grounding.

## Impact

- New evaluation runner, fixtures, report schema, command, Make targets, and CI-safe deterministic tests.
- Updated provider interface, OpenAI Responses transport, assistant orchestration, HTTP request schema, and browser assistant controls.
- The existing `message`-only assistant request remains valid; new history is optional and transient.
- CLI calculation commands, deterministic formulas, market normalization, CSV schemas, SQLite schema, and calculation-history behavior do not change.
- Live evaluation remains explicitly opt-in and requires a replaceable environment key; normal checks and deterministic workflows make no model or market network calls.
