package assistant

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestOpenAIModelFollowsResponsesFunctionCallingContract(t *testing.T) {
	requests := 0
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		if r.URL.Path != "/responses" || r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("request path/auth = %q/%q", r.URL.Path, r.Header.Get("Authorization"))
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if payload["store"] != false || payload["parallel_tool_calls"] != false {
			t.Errorf("request storage/parallel = %#v/%#v", payload["store"], payload["parallel_tool_calls"])
		}
		if requests == 1 {
			return jsonHTTPResponse(http.StatusOK, `{"id":"resp-1","model":"gpt-test","output":[{"type":"reasoning","id":"reason-1"},{"type":"function_call","call_id":"call-1","name":"calculate_profit","arguments":"{\"item_name\":\"Capsule\",\"buy_price\":0.48,\"quantity\":1,\"multiplier\":0.837,\"sell_prices\":[0.8]}"}],"usage":{"input_tokens":10,"output_tokens":4,"total_tokens":14}}`), nil
		}
		input := payload["input"].([]any)
		encoded, _ := json.Marshal(input)
		if !strings.Contains(string(encoded), "function_call_output") || !strings.Contains(string(encoded), "call-1") || !strings.Contains(string(encoded), "reason-1") {
			t.Errorf("second request input = %s", encoded)
		}
		return jsonHTTPResponse(http.StatusOK, `{"id":"resp-2","model":"gpt-test","output":[{"type":"message","content":[{"type":"output_text","text":"结果见 [T1]。"}]}],"usage":{"input_tokens":20,"output_tokens":5,"total_tokens":25}}`), nil
	})}

	model, err := NewOpenAIModel(OpenAIConfig{
		APIKey: "test-key", BaseURL: "https://api.example.test", Model: "gpt-test", ReasoningEffort: "low", Verbosity: "low",
		Timeout: 2 * time.Second, HTTPClient: client,
	})
	if err != nil {
		t.Fatalf("NewOpenAIModel() error = %v", err)
	}
	first, err := model.Respond(context.Background(), ModelRequest{Instructions: "test", Message: "calculate", Tools: ToolDefinitions()})
	if err != nil {
		t.Fatalf("first Respond() error = %v", err)
	}
	if len(first.ToolCalls) != 1 || first.ToolCalls[0].Name != toolCalculateProfit {
		t.Fatalf("first response = %#v", first)
	}
	second, err := model.Respond(context.Background(), ModelRequest{
		Instructions: "test", Message: "calculate", Tools: ToolDefinitions(), State: first.State,
		ToolOutputs: []ToolOutput{{CallID: "call-1", Output: `{"evidence_id":"T1"}`}},
	})
	if err != nil {
		t.Fatalf("second Respond() error = %v", err)
	}
	if second.Text != "结果见 [T1]。" || second.RequestID != "resp-2" || requests != 2 {
		t.Fatalf("second response = %#v, requests = %d", second, requests)
	}
}

func TestNewModelFromEnvironmentDegradesSafely(t *testing.T) {
	t.Setenv("STEAM_PROFIT_AI_PROVIDER", "openai")
	t.Setenv("STEAM_PROFIT_AI_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	model := NewModelFromEnvironment()
	if model.Status().Available || !strings.Contains(model.Status().Reason, "API_KEY") {
		t.Fatalf("status = %#v, want unavailable missing-key status", model.Status())
	}
}

func TestOpenAIProviderErrorDoesNotEchoRawBody(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonHTTPResponse(http.StatusBadGateway, "raw proxy body with internal details"), nil
	})}
	model, _ := NewOpenAIModel(OpenAIConfig{APIKey: "test-key", BaseURL: "https://api.example.test", Model: "gpt-test", ReasoningEffort: "low", Verbosity: "low", HTTPClient: client})
	_, err := model.Respond(context.Background(), ModelRequest{Message: "hello", Tools: ToolDefinitions()})
	if err == nil || strings.Contains(err.Error(), "internal details") {
		t.Fatalf("Respond() error = %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func jsonHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
