package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const maxProviderErrorBytes = 64 << 10

type openAIModel struct {
	config OpenAIConfig
}

type openAIState struct {
	Input []json.RawMessage
}

type openAIRequest struct {
	Model             string            `json:"model"`
	Instructions      string            `json:"instructions"`
	Input             []json.RawMessage `json:"input"`
	Tools             []ToolDefinition  `json:"tools"`
	ToolChoice        string            `json:"tool_choice"`
	ParallelToolCalls bool              `json:"parallel_tool_calls"`
	Reasoning         struct {
		Effort string `json:"effort"`
	} `json:"reasoning"`
	Text struct {
		Verbosity string `json:"verbosity"`
	} `json:"text"`
	Store           bool `json:"store"`
	MaxOutputTokens int  `json:"max_output_tokens"`
}

type openAIResponse struct {
	ID     string            `json:"id"`
	Model  string            `json:"model"`
	Output []json.RawMessage `json:"output"`
	Usage  struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type openAIOutputItem struct {
	Type      string `json:"type"`
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	Content   []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

func (m *openAIModel) Status() Status {
	return Status{
		Available: true, Provider: "openai", Model: m.config.Model, PromptVersion: PromptVersion,
		ReasoningEffort: m.config.ReasoningEffort, Verbosity: m.config.Verbosity, MaxOutputTokens: m.config.MaxOutputTokens,
	}
}

func (m *openAIModel) Respond(ctx context.Context, request ModelRequest) (ModelResponse, error) {
	state, err := m.prepareState(request)
	if err != nil {
		return ModelResponse{}, err
	}

	payload := openAIRequest{
		Model: m.config.Model, Instructions: request.Instructions, Input: state.Input, Tools: request.Tools,
		ToolChoice: "auto", ParallelToolCalls: false, Store: false, MaxOutputTokens: m.config.MaxOutputTokens,
	}
	payload.Reasoning.Effort = m.config.ReasoningEffort
	payload.Text.Verbosity = m.config.Verbosity
	body, err := json.Marshal(payload)
	if err != nil {
		return ModelResponse{}, fmt.Errorf("encode Responses API request: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, m.config.BaseURL+"/responses", bytes.NewReader(body))
	if err != nil {
		return ModelResponse{}, fmt.Errorf("create Responses API request: %w", err)
	}
	httpRequest.Header.Set("Authorization", "Bearer "+m.config.APIKey)
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")

	httpResponse, err := m.config.HTTPClient.Do(httpRequest)
	if err != nil {
		return ModelResponse{}, fmt.Errorf("call Responses API: %w", err)
	}
	defer httpResponse.Body.Close()
	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(httpResponse.Body, maxProviderErrorBytes))
		return ModelResponse{}, fmt.Errorf("Responses API returned %d: %s", httpResponse.StatusCode, safeProviderError(detail))
	}

	var response openAIResponse
	decoder := json.NewDecoder(io.LimitReader(httpResponse.Body, 4<<20))
	if err := decoder.Decode(&response); err != nil {
		return ModelResponse{}, fmt.Errorf("decode Responses API response: %w", err)
	}
	if response.Error != nil {
		return ModelResponse{}, fmt.Errorf("Responses API error: %s", response.Error.Message)
	}

	modelResponse := ModelResponse{
		RequestID: response.ID, Model: response.Model,
		Usage: Usage{InputTokens: response.Usage.InputTokens, OutputTokens: response.Usage.OutputTokens, TotalTokens: response.Usage.TotalTokens},
	}
	for _, raw := range response.Output {
		var item openAIOutputItem
		if err := json.Unmarshal(raw, &item); err != nil {
			return ModelResponse{}, fmt.Errorf("decode Responses API output item: %w", err)
		}
		switch item.Type {
		case "function_call":
			modelResponse.ToolCalls = append(modelResponse.ToolCalls, ToolCall{CallID: item.CallID, Name: item.Name, Arguments: item.Arguments})
		case "message":
			for _, content := range item.Content {
				if content.Type == "output_text" {
					modelResponse.Text += content.Text
				}
			}
		}
	}

	state.Input = append(state.Input, response.Output...)
	modelResponse.State = state
	return modelResponse, nil
}

func (m *openAIModel) prepareState(request ModelRequest) (*openAIState, error) {
	var state *openAIState
	if request.State == nil {
		message, err := json.Marshal(map[string]any{"role": "user", "content": request.Message})
		if err != nil {
			return nil, fmt.Errorf("encode user message: %w", err)
		}
		state = &openAIState{Input: []json.RawMessage{message}}
	} else {
		provided, ok := request.State.(*openAIState)
		if !ok {
			return nil, fmt.Errorf("invalid OpenAI conversation state")
		}
		state = &openAIState{Input: append([]json.RawMessage(nil), provided.Input...)}
	}

	for _, output := range request.ToolOutputs {
		item, err := json.Marshal(map[string]any{
			"type": "function_call_output", "call_id": output.CallID, "output": output.Output,
		})
		if err != nil {
			return nil, fmt.Errorf("encode function output: %w", err)
		}
		state.Input = append(state.Input, item)
	}
	return state, nil
}

func safeProviderError(raw []byte) string {
	var payload struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &payload) == nil && strings.TrimSpace(payload.Error.Message) != "" {
		return payload.Error.Message
	}
	return "provider request failed"
}
