package evaluation

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"steam-profit-calculator/internal/assistant"
)

type scriptedEvaluationModel struct {
	fail bool
}

func (m *scriptedEvaluationModel) Status() assistant.Status {
	return assistant.Status{Available: true, Provider: "scripted", Model: "scripted-eval", PromptVersion: assistant.PromptVersion}
}

func (m *scriptedEvaluationModel) Respond(_ context.Context, request assistant.ModelRequest) (assistant.ModelResponse, error) {
	if m.fail {
		return assistant.ModelResponse{}, errors.New("SECRET provider payload")
	}
	if len(request.ToolOutputs) == 0 {
		return assistant.ModelResponse{
			State: "tool-turn",
			ToolCalls: []assistant.ToolCall{{
				CallID: "eval-call", Name: "calculate_profit",
				Arguments: `{"item_name":"Evaluation","buy_price":0.48,"quantity":1,"multiplier":0.837,"sell_prices":[0.8]}`,
			}},
			Usage: assistant.Usage{InputTokens: 10, OutputTokens: 2, TotalTokens: 12},
		}, nil
	}
	return assistant.ModelResponse{
		Text: "结果见 [T1]。", TextDeltas: []string{"结果见 ", "[T1]。"}, NativeStreaming: true,
		RequestID: "eval-response", Usage: assistant.Usage{InputTokens: 12, OutputTokens: 3, TotalTokens: 15},
	}, nil
}

func TestLoadSuiteReturnsVersionedCases(t *testing.T) {
	cases, err := LoadSuite(0)
	if err != nil {
		t.Fatalf("LoadSuite() error = %v", err)
	}
	if len(cases) != 12 {
		t.Fatalf("semantic cases = %d, want 12", len(cases))
	}
	limited, err := LoadSuite(3)
	if err != nil || len(limited) != 3 || limited[0].ID != cases[0].ID {
		t.Fatalf("limited suite = %#v, error = %v", limited, err)
	}
}

func TestReportPassesAndOmitsPromptAndToolContent(t *testing.T) {
	service := evaluationService(t, &scriptedEvaluationModel{})
	cases := []Case{{
		ID: "content-free", Message: "SECRET evaluation prompt", ExpectedTools: []string{"calculate_profit"},
		RequireGrounded: true, MaxElapsedMS: 1000, MaxTotalTokens: 100,
	}}
	report, err := Run(context.Background(), service, cases, Options{
		Thresholds: Thresholds{MinPassRate: 1, MaxP95ElapsedMS: 1000, MaxAverageTokens: 100, MaxEstimatedCostUSD: 1},
		Pricing:    Pricing{InputUSDPerMillion: 1, OutputUSDPerMillion: 2},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !report.Passed || report.Summary.Passed != 1 || !report.Cases[0].NativeStreaming {
		t.Fatalf("report = %#v", report)
	}
	encoded, _ := json.Marshal(report)
	for _, forbidden := range []string{"SECRET", "Evaluation", "0.48", "total_profit"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("report leaked %q: %s", forbidden, encoded)
		}
	}
}

func TestReportAttributesToolAndAggregateThresholdFailures(t *testing.T) {
	service := evaluationService(t, &scriptedEvaluationModel{})
	report, err := Run(context.Background(), service, []Case{{
		ID: "wrong-tool", Message: "run", ExpectedTools: []string{"solve_target"}, RequireGrounded: true,
	}}, Options{Thresholds: Thresholds{MinPassRate: 1}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Passed || report.Cases[0].Passed || !contains(report.Cases[0].FailureCodes, "tool_mismatch") || !contains(report.Summary.ThresholdFailures, "minimum_pass_rate") {
		t.Fatalf("report = %#v", report)
	}
}

func TestReportNormalizesProviderFailure(t *testing.T) {
	service := evaluationService(t, &scriptedEvaluationModel{fail: true})
	report, err := Run(context.Background(), service, []Case{{ID: "provider-failure", Message: "run"}}, Options{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	encoded, _ := json.Marshal(report)
	if !contains(report.Cases[0].FailureCodes, "assistant_error") || strings.Contains(string(encoded), "SECRET") {
		t.Fatalf("report = %s", encoded)
	}
}

func evaluationService(t *testing.T, model assistant.Model) *assistant.Service {
	t.Helper()
	runner, err := assistant.NewToolRunner(FixtureLookup{})
	if err != nil {
		t.Fatalf("NewToolRunner() error = %v", err)
	}
	service, err := assistant.NewService(model, runner)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
