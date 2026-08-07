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
	maxHistoryMessages       = 6
	maxHistoryRunes          = 12000
	maxModelRounds           = 4
	answerChunkRunes         = 28
)

// ErrInvalidMessage indicates that a browser prompt is missing or too large.
var ErrInvalidMessage = errors.New("assistant message must contain between 1 and 4000 characters")

// ErrInvalidHistory indicates that transient browser context violated a public bound.
var ErrInvalidHistory = errors.New("assistant history must contain at most 6 user or assistant messages and 12000 characters")

// Service coordinates one bounded model/tool request without persisting conversation state.
type Service struct {
	model         Model
	tools         *ToolRunner
	instructions  string
	promptVersion string
}

// NewService creates an assistant service from injectable model and lookup boundaries.
func NewService(model Model, tools *ToolRunner) (*Service, error) {
	return NewServiceForPrompt(model, tools, PromptVersion)
}

// NewServiceForPrompt creates a service for an immutable production or evaluation prompt version.
func NewServiceForPrompt(model Model, tools *ToolRunner, promptVersion string) (*Service, error) {
	if model == nil {
		return nil, fmt.Errorf("model is required")
	}
	if tools == nil {
		return nil, fmt.Errorf("tool runner is required")
	}
	instructions, err := PromptInstructions(promptVersion)
	if err != nil {
		return nil, err
	}
	if promptVersion == "" {
		promptVersion = PromptVersion
	}
	return &Service{model: model, tools: tools, instructions: instructions, promptVersion: promptVersion}, nil
}

// Status returns safe public model configuration details.
func (s *Service) Status() Status {
	status := s.model.Status()
	status.PromptVersion = s.promptVersion
	status.RetrySupported = true
	status.MaxHistory = maxHistoryMessages
	return status
}

// Run executes one stateless assistant request and emits progressive lifecycle events.
func (s *Service) Run(ctx context.Context, message string, emit EmitFunc) error {
	return s.RunConversation(ctx, message, nil, emit)
}

// RunConversation executes one request with bounded, transient prior context.
func (s *Service) RunConversation(ctx context.Context, message string, history []ConversationMessage, emit EmitFunc) error {
	message = strings.TrimSpace(message)
	if err := ValidateMessage(message); err != nil {
		return ErrInvalidMessage
	}
	normalizedHistory, err := ValidateHistory(history)
	if err != nil {
		return err
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
	nativeStreaming := false

	for round := 0; round < maxModelRounds; round++ {
		response, err := s.model.Respond(ctx, ModelRequest{
			Instructions: s.instructions, Message: message, Tools: ToolDefinitions(),
			History: normalizedHistory, State: state, ToolOutputs: outputs,
		})
		if err != nil {
			return fmt.Errorf("model response: %w", err)
		}
		state = response.State
		nativeStreaming = nativeStreaming || response.NativeStreaming
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
		answerChunks := chunkText(grounded.Text, answerChunkRunes)
		if grounded.Grounded && response.NativeStreaming && strings.Join(response.TextDeltas, "") == response.Text {
			answerChunks = nonEmptyChunks(response.TextDeltas)
		}
		for _, chunk := range answerChunks {
			if err := emit(Event{Type: "delta", Text: chunk}); err != nil {
				return err
			}
		}
		status := s.model.Status()
		completion := Completion{
			Provider: status.Provider, Model: modelName, PromptVersion: s.promptVersion,
			ReasoningEffort: status.ReasoningEffort, Verbosity: status.Verbosity, MaxOutputTokens: status.MaxOutputTokens,
			RequestID: lastRequestID, Grounded: grounded.Grounded, GroundingNote: grounded.Note,
			NativeStreaming: nativeStreaming, ToolCount: len(evidence), ElapsedMS: time.Since(started).Milliseconds(), Usage: usage,
		}
		return emit(Event{Type: "done", Completion: &completion})
	}

	return fmt.Errorf("assistant exceeded the maximum of %d model rounds", maxModelRounds)
}

// ValidateHistory normalizes and defensively copies transient conversation context.
func ValidateHistory(history []ConversationMessage) ([]ConversationMessage, error) {
	if len(history) > maxHistoryMessages {
		return nil, ErrInvalidHistory
	}
	normalized := make([]ConversationMessage, 0, len(history))
	totalRunes := 0
	for _, message := range history {
		role := strings.TrimSpace(message.Role)
		content := strings.TrimSpace(message.Content)
		if (role != "user" && role != "assistant") || content == "" || len([]rune(content)) > maxAssistantMessageRunes {
			return nil, ErrInvalidHistory
		}
		totalRunes += len([]rune(content))
		if totalRunes > maxHistoryRunes {
			return nil, ErrInvalidHistory
		}
		normalized = append(normalized, ConversationMessage{Role: role, Content: content})
	}
	return normalized, nil
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

func nonEmptyChunks(chunks []string) []string {
	result := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk != "" {
			result = append(result, chunk)
		}
	}
	return result
}
