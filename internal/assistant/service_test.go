package assistant

import (
	"context"
	"errors"
	"strings"
	"testing"

	"steam-profit-calculator/internal/domain"
)

type scriptedModel struct {
	status    Status
	responses []ModelResponse
	errors    []error
	requests  []ModelRequest
}

func (m *scriptedModel) Status() Status { return m.status }

func (m *scriptedModel) Respond(_ context.Context, request ModelRequest) (ModelResponse, error) {
	m.requests = append(m.requests, request)
	index := len(m.requests) - 1
	if index < len(m.errors) && m.errors[index] != nil {
		return ModelResponse{}, m.errors[index]
	}
	if index >= len(m.responses) {
		return ModelResponse{}, errors.New("unexpected model turn")
	}
	return m.responses[index], nil
}

type assistantLookupStub struct {
	items []domain.ItemSummary
	err   error
}

func (s *assistantLookupStub) Lookup(context.Context, domain.ItemLookupRequest) ([]domain.ItemSummary, error) {
	return s.items, s.err
}

func TestServiceRunsToolLoopAndEmitsGroundedEvidence(t *testing.T) {
	model := &scriptedModel{
		status: Status{Available: true, Provider: "fake", Model: "fake-1", PromptVersion: PromptVersion},
		responses: []ModelResponse{
			{State: "turn-1", ToolCalls: []ToolCall{{
				CallID: "call-1", Name: toolCalculateProfit,
				Arguments: `{"item_name":"Capsule","buy_price":0.48,"quantity":5,"multiplier":0.837,"sell_prices":[0.8]}`,
			}}, Usage: Usage{InputTokens: 10, OutputTokens: 3, TotalTokens: 13}},
			{State: "turn-2", Text: "Capsule 数量 5 的总收益为 0.948 [T1]。仅供研究，不构成投资或交易建议。", RequestID: "req-2", Usage: Usage{InputTokens: 20, OutputTokens: 8, TotalTokens: 28}},
		},
	}
	service := newAssistantService(t, model)
	var events []Event
	if err := service.Run(context.Background(), "帮我算收益", func(event Event) error {
		events = append(events, event)
		return nil
	}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(model.requests) != 2 {
		t.Fatalf("model turns = %d, want 2", len(model.requests))
	}
	if got := model.requests[1].ToolOutputs; len(got) != 1 || !strings.Contains(got[0].Output, `"total_profit":0.948`) {
		t.Fatalf("second-turn tool outputs = %#v", got)
	}
	if !hasEvent(events, "evidence") || !hasEvent(events, "delta") || !hasEvent(events, "done") {
		t.Fatalf("events = %#v, want evidence, delta, and done", events)
	}
	completion := events[len(events)-1].Completion
	if completion == nil || !completion.Grounded || completion.ToolCount != 1 || completion.Usage.TotalTokens != 41 {
		t.Fatalf("completion = %#v", completion)
	}
}

func TestServiceReplacesUngroundedModelText(t *testing.T) {
	model := &scriptedModel{
		status: Status{Available: true, Provider: "fake", Model: "fake-1", PromptVersion: PromptVersion},
		responses: []ModelResponse{
			{State: "turn-1", ToolCalls: []ToolCall{{CallID: "call-1", Name: toolCalculateProfit, Arguments: `{"item_name":"Capsule","buy_price":0.48,"quantity":1,"multiplier":0.837,"sell_prices":[0.8]}`}}},
			{Text: "总收益一定是 99.9 [T1]。"},
		},
	}
	service := newAssistantService(t, model)
	var answer strings.Builder
	var completion *Completion
	if err := service.Run(context.Background(), "算一下", func(event Event) error {
		if event.Type == "delta" {
			answer.WriteString(event.Text)
		}
		if event.Completion != nil {
			completion = event.Completion
		}
		return nil
	}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if strings.Contains(answer.String(), "99.9") || !strings.Contains(answer.String(), "未通过数值溯源") {
		t.Fatalf("answer = %q, want safe fallback", answer.String())
	}
	if completion == nil || completion.Grounded {
		t.Fatalf("completion = %#v, want ungrounded", completion)
	}
}

func TestServiceReturnsProviderFailure(t *testing.T) {
	model := &scriptedModel{
		status: Status{Available: true, Provider: "fake", PromptVersion: PromptVersion},
		errors: []error{errors.New("provider down")},
	}
	service := newAssistantService(t, model)
	err := service.Run(context.Background(), "解释收益", nil)
	if err == nil || !strings.Contains(err.Error(), "provider down") {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestServicePreservesNativeDeltasAfterGrounding(t *testing.T) {
	model := &scriptedModel{
		status: Status{Available: true, Provider: "fake", Model: "fake-1", PromptVersion: PromptVersion, NativeStreaming: true},
		responses: []ModelResponse{
			{State: "turn-1", ToolCalls: []ToolCall{{CallID: "call-1", Name: toolCalculateProfit, Arguments: `{"item_name":"Capsule","buy_price":0.48,"quantity":1,"multiplier":0.837,"sell_prices":[0.8]}`}}},
			{Text: "结果见 [T1]。", TextDeltas: []string{"结果", "见 ", "[T1]。"}, NativeStreaming: true},
		},
	}
	service := newAssistantService(t, model)
	var deltas []string
	var completion *Completion
	if err := service.Run(context.Background(), "计算", func(event Event) error {
		if event.Type == "delta" {
			deltas = append(deltas, event.Text)
		}
		if event.Completion != nil {
			completion = event.Completion
		}
		return nil
	}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if strings.Join(deltas, "|") != "结果|见 |[T1]。" || completion == nil || !completion.NativeStreaming {
		t.Fatalf("deltas/completion = %#v / %#v", deltas, completion)
	}
}

func TestServiceDoesNotExposeUngroundedNativeDeltas(t *testing.T) {
	model := &scriptedModel{
		status: Status{Available: true, Provider: "fake", PromptVersion: PromptVersion, NativeStreaming: true},
		responses: []ModelResponse{
			{State: "turn-1", ToolCalls: []ToolCall{{CallID: "call-1", Name: toolCalculateProfit, Arguments: `{"item_name":"Capsule","buy_price":0.48,"quantity":1,"multiplier":0.837,"sell_prices":[0.8]}`}}},
			{Text: "凭空声称 99.9 [T1]。", TextDeltas: []string{"凭空声称 ", "99.9 ", "[T1]。"}, NativeStreaming: true},
		},
	}
	service := newAssistantService(t, model)
	var answer strings.Builder
	if err := service.Run(context.Background(), "计算", func(event Event) error {
		if event.Type == "delta" {
			answer.WriteString(event.Text)
		}
		return nil
	}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if strings.Contains(answer.String(), "99.9") || !strings.Contains(answer.String(), "未通过数值溯源") {
		t.Fatalf("answer = %q", answer.String())
	}
}

func TestServiceValidatesAndPassesTransientHistory(t *testing.T) {
	model := &scriptedModel{
		status:    Status{Available: true, Provider: "fake", PromptVersion: PromptVersion},
		responses: []ModelResponse{{Text: "可以继续，但不使用历史数字。"}},
	}
	service := newAssistantService(t, model)
	history := []ConversationMessage{{Role: " user ", Content: " previous question "}, {Role: "assistant", Content: "previous answer"}}
	if err := service.RunConversation(context.Background(), "follow up", history, nil); err != nil {
		t.Fatalf("RunConversation() error = %v", err)
	}
	if len(model.requests) != 1 || len(model.requests[0].History) != 2 || model.requests[0].History[0].Role != "user" || model.requests[0].History[0].Content != "previous question" {
		t.Fatalf("model history = %#v", model.requests)
	}

	tooMany := make([]ConversationMessage, maxHistoryMessages+1)
	for index := range tooMany {
		tooMany[index] = ConversationMessage{Role: "user", Content: "message"}
	}
	if err := service.RunConversation(context.Background(), "follow up", tooMany, nil); !errors.Is(err, ErrInvalidHistory) {
		t.Fatalf("invalid history error = %v", err)
	}
	if len(model.requests) != 1 {
		t.Fatalf("model requests after invalid history = %d, want 1", len(model.requests))
	}
}

type cancellingModel struct{}

func (cancellingModel) Status() Status {
	return Status{Available: true, Provider: "blocking", PromptVersion: PromptVersion}
}

func (cancellingModel) Respond(ctx context.Context, _ ModelRequest) (ModelResponse, error) {
	<-ctx.Done()
	return ModelResponse{}, ctx.Err()
}

func TestServicePropagatesCancellation(t *testing.T) {
	service := newAssistantService(t, cancellingModel{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := service.Run(ctx, "cancel this", nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want context cancellation", err)
	}
}

func TestServiceRejectsUnavailableAndInvalidMessages(t *testing.T) {
	service := newAssistantService(t, &scriptedModel{status: Status{Available: false, Provider: "disabled"}})
	if err := service.Run(context.Background(), "", nil); !errors.Is(err, ErrInvalidMessage) {
		t.Fatalf("empty message error = %v", err)
	}
	if err := service.Run(context.Background(), "hello", nil); !errors.Is(err, ErrModelUnavailable) {
		t.Fatalf("unavailable error = %v", err)
	}
}

func newAssistantService(t *testing.T, model Model) *Service {
	t.Helper()
	runner, err := NewToolRunner(&assistantLookupStub{})
	if err != nil {
		t.Fatalf("NewToolRunner() error = %v", err)
	}
	service, err := NewService(model, runner)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

func hasEvent(events []Event, eventType string) bool {
	for _, event := range events {
		if event.Type == eventType {
			return true
		}
	}
	return false
}
