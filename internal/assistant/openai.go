package assistant

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

const (
	maxProviderErrorBytes  = 64 << 10
	maxProviderEventBytes  = 512 << 10
	maxProviderStreamBytes = 8 << 20
)

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
	Stream          bool `json:"stream"`
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

type openAIStreamEvent struct {
	Type        string             `json:"type"`
	Delta       string             `json:"delta"`
	Arguments   string             `json:"arguments"`
	OutputIndex int                `json:"output_index"`
	Item        json.RawMessage    `json:"item"`
	Response    *openAIResponse    `json:"response"`
	Error       *openAIStreamError `json:"error,omitempty"`
	Message     string             `json:"message,omitempty"`
}

type openAIStreamError struct {
	Message string `json:"message"`
}

func (m *openAIModel) Status() Status {
	return Status{
		Available: true, Provider: "openai", Model: m.config.Model, PromptVersion: PromptVersion,
		NativeStreaming: true, MaxHistory: maxHistoryMessages,
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
		ToolChoice: "auto", ParallelToolCalls: false, Store: false, Stream: true, MaxOutputTokens: m.config.MaxOutputTokens,
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
	httpRequest.Header.Set("Accept", "text/event-stream")

	httpResponse, err := m.config.HTTPClient.Do(httpRequest)
	if err != nil {
		return ModelResponse{}, fmt.Errorf("call Responses API: %w", err)
	}
	defer httpResponse.Body.Close()
	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(httpResponse.Body, maxProviderErrorBytes))
		return ModelResponse{}, fmt.Errorf("Responses API returned %d: %s", httpResponse.StatusCode, safeProviderError(detail))
	}

	if !strings.HasPrefix(strings.ToLower(httpResponse.Header.Get("Content-Type")), "text/event-stream") {
		return m.decodeJSONResponse(httpResponse.Body, state)
	}
	return m.decodeStream(httpResponse.Body, state)
}

func (m *openAIModel) decodeJSONResponse(reader io.Reader, state *openAIState) (ModelResponse, error) {
	var response openAIResponse
	decoder := json.NewDecoder(io.LimitReader(reader, 4<<20))
	if err := decoder.Decode(&response); err != nil {
		return ModelResponse{}, fmt.Errorf("decode Responses API response: %w", err)
	}
	return modelResponseFromOpenAI(&response, state, nil, false)
}

func (m *openAIModel) decodeStream(reader io.Reader, state *openAIState) (ModelResponse, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64<<10), maxProviderEventBytes)
	var dataLines []string
	eventBytes := 0
	totalBytes := 0
	textDeltas := make([]string, 0)
	functionCalls := make(map[int]openAIOutputItem)
	completedItems := make(map[int]json.RawMessage)
	var completed *openAIResponse

	processEvent := func() error {
		if len(dataLines) == 0 {
			return nil
		}
		data := strings.Join(dataLines, "\n")
		dataLines = dataLines[:0]
		eventBytes = 0
		if data == "[DONE]" {
			return nil
		}
		var event openAIStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return fmt.Errorf("decode Responses API stream event: %w", err)
		}
		switch event.Type {
		case "response.output_text.delta":
			if event.Delta != "" {
				textDeltas = append(textDeltas, event.Delta)
			}
		case "response.output_item.added":
			var item openAIOutputItem
			if len(event.Item) > 0 && json.Unmarshal(event.Item, &item) == nil && item.Type == "function_call" {
				functionCalls[event.OutputIndex] = item
			}
		case "response.function_call_arguments.delta":
			item, ok := functionCalls[event.OutputIndex]
			if !ok {
				return fmt.Errorf("Responses API streamed function arguments before a function call item")
			}
			item.Arguments += event.Delta
			functionCalls[event.OutputIndex] = item
		case "response.function_call_arguments.done":
			item, ok := functionCalls[event.OutputIndex]
			if ok && event.Arguments != "" {
				item.Arguments = event.Arguments
				functionCalls[event.OutputIndex] = item
			}
		case "response.output_item.done":
			if len(event.Item) > 0 {
				completedItems[event.OutputIndex] = append(json.RawMessage(nil), event.Item...)
			}
		case "response.completed":
			if event.Response == nil {
				return fmt.Errorf("Responses API completed without response metadata")
			}
			completed = event.Response
		case "response.failed", "response.incomplete", "error":
			return fmt.Errorf("Responses API stream failed")
		}
		return nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		totalBytes += len(line) + 1
		if totalBytes > maxProviderStreamBytes {
			return ModelResponse{}, fmt.Errorf("Responses API stream exceeded %d bytes", maxProviderStreamBytes)
		}
		if line == "" {
			if err := processEvent(); err != nil {
				return ModelResponse{}, err
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLine := strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " ")
			eventBytes += len(dataLine)
			if eventBytes > maxProviderEventBytes {
				return ModelResponse{}, fmt.Errorf("Responses API stream event exceeded %d bytes", maxProviderEventBytes)
			}
			dataLines = append(dataLines, dataLine)
		}
	}
	if err := scanner.Err(); err != nil {
		return ModelResponse{}, fmt.Errorf("read Responses API stream: %w", err)
	}
	if err := processEvent(); err != nil {
		return ModelResponse{}, err
	}
	if completed == nil {
		return ModelResponse{}, fmt.Errorf("Responses API stream ended before completion")
	}
	if len(completed.Output) == 0 && len(completedItems) > 0 {
		indexes := make([]int, 0, len(completedItems))
		for index := range completedItems {
			indexes = append(indexes, index)
		}
		sort.Ints(indexes)
		for _, index := range indexes {
			completed.Output = append(completed.Output, completedItems[index])
		}
	}
	modelResponse, err := modelResponseFromOpenAI(completed, state, textDeltas, true)
	if err != nil {
		return ModelResponse{}, err
	}
	if len(modelResponse.ToolCalls) == 0 && len(functionCalls) > 0 {
		indexes := make([]int, 0, len(functionCalls))
		for index := range functionCalls {
			indexes = append(indexes, index)
		}
		sort.Ints(indexes)
		for _, index := range indexes {
			item := functionCalls[index]
			modelResponse.ToolCalls = append(modelResponse.ToolCalls, ToolCall{CallID: item.CallID, Name: item.Name, Arguments: item.Arguments})
		}
	}
	return modelResponse, nil
}

func modelResponseFromOpenAI(response *openAIResponse, state *openAIState, textDeltas []string, nativeStreaming bool) (ModelResponse, error) {
	if response.Error != nil {
		return ModelResponse{}, fmt.Errorf("Responses API error: %s", response.Error.Message)
	}
	modelResponse := ModelResponse{
		RequestID: response.ID, Model: response.Model, TextDeltas: append([]string(nil), textDeltas...), NativeStreaming: nativeStreaming,
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
	if len(textDeltas) > 0 {
		modelResponse.Text = strings.Join(textDeltas, "")
	}

	state.Input = append(state.Input, response.Output...)
	modelResponse.State = state
	return modelResponse, nil
}

func (m *openAIModel) prepareState(request ModelRequest) (*openAIState, error) {
	var state *openAIState
	if request.State == nil {
		state = &openAIState{}
		for _, previous := range request.History {
			message, err := json.Marshal(map[string]any{"role": previous.Role, "content": previous.Content})
			if err != nil {
				return nil, fmt.Errorf("encode conversation history: %w", err)
			}
			state.Input = append(state.Input, message)
		}
		message, err := json.Marshal(map[string]any{"role": "user", "content": request.Message})
		if err != nil {
			return nil, fmt.Errorf("encode user message: %w", err)
		}
		state.Input = append(state.Input, message)
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
