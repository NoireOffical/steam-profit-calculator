package assistant

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	maxAssistantMessageRunes = 4000
	maxModelRounds           = 4
	answerChunkRunes         = 28
)

// ErrInvalidMessage indicates that a browser prompt is missing or too large.
var ErrInvalidMessage = errors.New("assistant message must contain between 1 and 4000 characters")

// Service coordinates one bounded model/tool request without persisting conversation state.
type Service struct {
	model Model
	tools *ToolRunner
}

// NewService creates an assistant service from injectable model and lookup boundaries.
func NewService(model Model, tools *ToolRunner) (*Service, error) {
	if model == nil {
		return nil, fmt.Errorf("model is required")
	}
	if tools == nil {
		return nil, fmt.Errorf("tool runner is required")
	}
	return &Service{model: model, tools: tools}, nil
}

// Status returns safe public model configuration details.
func (s *Service) Status() Status {
	return s.model.Status()
}

// Run executes one stateless assistant request and emits progressive lifecycle events.
func (s *Service) Run(ctx context.Context, message string, emit EmitFunc) error {
	message = strings.TrimSpace(message)
	if err := ValidateMessage(message); err != nil {
		return ErrInvalidMessage
	}
	if !s.model.Status().Available {
		return ErrModelUnavailable
	}
	if emit == nil {
		emit = func(Event) error { return nil }
	}

	started := time.Now()
	if err := emit(Event{Type: "status", Message: "正在理解目标并选择可信工具…"}); err != nil {
		return err
	}

	var state any
	var outputs []ToolOutput
	var evidence []Evidence
	var usage Usage
	var lastRequestID string
	modelName := s.model.Status().Model

	for round := 0; round < maxModelRounds; round++ {
		response, err := s.model.Respond(ctx, ModelRequest{
			Instructions: Instructions(), Message: message, Tools: ToolDefinitions(),
			State: state, ToolOutputs: outputs,
		})
		if err != nil {
			return fmt.Errorf("model response: %w", err)
		}
		state = response.State
		outputs = nil
		usage.InputTokens += response.Usage.InputTokens
		usage.OutputTokens += response.Usage.OutputTokens
		usage.TotalTokens += response.Usage.TotalTokens
		if response.RequestID != "" {
			lastRequestID = response.RequestID
		}
		if response.Model != "" {
			modelName = response.Model
		}

		if len(response.ToolCalls) > 0 {
			for _, call := range response.ToolCalls {
				if call.CallID == "" {
					return fmt.Errorf("model returned a tool call without a call ID")
				}
				if err := emit(Event{Type: "tool_start", Tool: call.Name, Message: "正在执行可信工具"}); err != nil {
					return err
				}
				evidenceID := fmt.Sprintf("T%d", len(evidence)+1)
				execution := s.tools.execute(ctx, call, evidenceID)
				evidence = append(evidence, execution.Evidence)
				outputs = append(outputs, execution.Output)
				if err := emit(Event{Type: "evidence", Tool: call.Name, Evidence: &execution.Evidence}); err != nil {
					return err
				}
			}
			continue
		}

		if strings.TrimSpace(response.Text) == "" {
			return fmt.Errorf("model returned neither tool calls nor answer text")
		}

		grounded := enforceGrounding(response.Text, evidence)
		for _, chunk := range chunkText(grounded.Text, answerChunkRunes) {
			if err := emit(Event{Type: "delta", Text: chunk}); err != nil {
				return err
			}
		}
		status := s.model.Status()
		completion := Completion{
			Provider: status.Provider, Model: modelName, PromptVersion: PromptVersion,
			ReasoningEffort: status.ReasoningEffort, Verbosity: status.Verbosity, MaxOutputTokens: status.MaxOutputTokens,
			RequestID: lastRequestID, Grounded: grounded.Grounded, GroundingNote: grounded.Note,
			ToolCount: len(evidence), ElapsedMS: time.Since(started).Milliseconds(), Usage: usage,
		}
		return emit(Event{Type: "done", Completion: &completion})
	}

	return fmt.Errorf("assistant exceeded the maximum of %d model rounds", maxModelRounds)
}

// ValidateMessage checks the public assistant prompt boundary before streaming starts.
func ValidateMessage(message string) error {
	message = strings.TrimSpace(message)
	if message == "" || len([]rune(message)) > maxAssistantMessageRunes {
		return ErrInvalidMessage
	}
	return nil
}

func chunkText(text string, maximum int) []string {
	characters := []rune(text)
	if len(characters) == 0 {
		return nil
	}
	chunks := make([]string, 0, (len(characters)+maximum-1)/maximum)
	for start := 0; start < len(characters); start += maximum {
		end := start + maximum
		if end > len(characters) {
			end = len(characters)
		}
		chunks = append(chunks, string(characters[start:end]))
	}
	return chunks
}
