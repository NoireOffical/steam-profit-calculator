package assistant

import (
	"context"
	"encoding/json"
)

// Status describes whether the optional model layer can accept assistant work.
type Status struct {
	Available       bool   `json:"available"`
	Provider        string `json:"provider"`
	Model           string `json:"model,omitempty"`
	PromptVersion   string `json:"prompt_version"`
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
	Verbosity       string `json:"verbosity,omitempty"`
	MaxOutputTokens int    `json:"max_output_tokens,omitempty"`
	Reason          string `json:"reason,omitempty"`
}

// Usage records model usage returned by a provider.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// ToolDefinition is the provider-neutral function tool contract.
type ToolDefinition struct {
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
	Strict      bool           `json:"strict"`
}

// ToolCall is one allow-listed function request produced by a model.
type ToolCall struct {
	CallID    string
	Name      string
	Arguments string
}

// ToolOutput is returned to the model using the original provider call ID.
type ToolOutput struct {
	CallID string
	Output string
}

// ModelRequest contains one turn in a bounded model/tool loop.
type ModelRequest struct {
	Instructions string
	Message      string
	Tools        []ToolDefinition
	State        any
	ToolOutputs  []ToolOutput
}

// ModelResponse is the provider-neutral result of one model turn.
type ModelResponse struct {
	State     any
	Text      string
	ToolCalls []ToolCall
	RequestID string
	Model     string
	Usage     Usage
}

// Model is the replaceable model-provider boundary used by the assistant.
type Model interface {
	Status() Status
	Respond(ctx context.Context, request ModelRequest) (ModelResponse, error)
}

// Evidence is authoritative structured output produced by a project tool.
type Evidence struct {
	ID       string          `json:"id"`
	Tool     string          `json:"tool"`
	Status   string          `json:"status"`
	Data     json.RawMessage `json:"data"`
	Verified bool            `json:"verified"`
}

// Completion summarizes observable assistant execution metadata.
type Completion struct {
	Provider        string `json:"provider"`
	Model           string `json:"model,omitempty"`
	PromptVersion   string `json:"prompt_version"`
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
	Verbosity       string `json:"verbosity,omitempty"`
	MaxOutputTokens int    `json:"max_output_tokens,omitempty"`
	RequestID       string `json:"request_id,omitempty"`
	Grounded        bool   `json:"grounded"`
	GroundingNote   string `json:"grounding_note,omitempty"`
	ToolCount       int    `json:"tool_count"`
	ElapsedMS       int64  `json:"elapsed_ms"`
	Usage           Usage  `json:"usage"`
}

// Event is emitted progressively by Service.Run.
type Event struct {
	Type       string      `json:"-"`
	Message    string      `json:"message,omitempty"`
	Tool       string      `json:"tool,omitempty"`
	Text       string      `json:"text,omitempty"`
	Evidence   *Evidence   `json:"evidence,omitempty"`
	Completion *Completion `json:"completion,omitempty"`
}

// EmitFunc receives one assistant lifecycle event.
type EmitFunc func(Event) error
