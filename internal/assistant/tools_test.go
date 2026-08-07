package assistant

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"steam-profit-calculator/internal/domain"
	"steam-profit-calculator/internal/lookup"
)

func TestToolDefinitionsAreStrictClosedSchemas(t *testing.T) {
	definitions := ToolDefinitions()
	if len(definitions) != 3 {
		t.Fatalf("definitions = %d, want 3", len(definitions))
	}
	for _, definition := range definitions {
		if definition.Type != "function" || !definition.Strict {
			t.Fatalf("definition %q is not a strict function", definition.Name)
		}
		if closed, ok := definition.Parameters["additionalProperties"].(bool); !ok || closed {
			t.Fatalf("definition %q additionalProperties = %#v, want false", definition.Name, definition.Parameters["additionalProperties"])
		}
		properties := definition.Parameters["properties"].(map[string]any)
		required := definition.Parameters["required"].([]string)
		if len(properties) != len(required) {
			t.Fatalf("definition %q properties = %d, required = %d", definition.Name, len(properties), len(required))
		}
	}
}

func TestToolRunnerExecutesReadOnlyCalculation(t *testing.T) {
	runner, _ := NewToolRunner(&assistantLookupStub{})
	execution := runner.execute(context.Background(), ToolCall{
		CallID: "call-1", Name: toolCalculateProfit,
		Arguments: `{"item_name":"Capsule","buy_price":0.48,"quantity":5,"multiplier":0.837,"sell_prices":[0.7,0.8]}`,
	}, "T1")
	if !execution.Evidence.Verified || !strings.Contains(execution.Output.Output, `"total_profit":0.948`) {
		t.Fatalf("execution = %#v", execution)
	}
}

func TestToolRunnerNormalizesLookupFailures(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code string
	}{
		{name: "no matches", err: lookup.ErrNoMatches, code: "no_matches"},
		{name: "official rate limit", err: fmt.Errorf("retry exhausted: %w", lookup.ErrSteamOfficialRateLimit), code: "official_provider_rate_limited"},
		{name: "provider", err: errors.New("secret upstream detail"), code: "provider_error"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runner, _ := NewToolRunner(&assistantLookupStub{err: test.err})
			execution := runner.execute(context.Background(), ToolCall{CallID: "call-1", Name: toolLookupItems, Arguments: `{"query":"AK","limit":3}`}, "T1")
			if execution.Evidence.Verified || !strings.Contains(execution.Output.Output, `"code":"`+test.code+`"`) {
				t.Fatalf("execution = %#v", execution)
			}
			if strings.Contains(execution.Output.Output, "secret upstream detail") {
				t.Fatalf("provider detail leaked in %q", execution.Output.Output)
			}
		})
	}
}

func TestToolRunnerRejectsUnknownFieldsAndUnknownTools(t *testing.T) {
	runner, _ := NewToolRunner(&assistantLookupStub{items: []domain.ItemSummary{{CanonicalName: "AK"}}})
	invalid := runner.execute(context.Background(), ToolCall{CallID: "call-1", Name: toolLookupItems, Arguments: `{"query":"AK","limit":3,"extra":true}`}, "T1")
	if invalid.Evidence.Verified || !strings.Contains(invalid.Output.Output, "invalid_arguments") {
		t.Fatalf("invalid execution = %#v", invalid)
	}
	unknown := runner.execute(context.Background(), ToolCall{CallID: "call-2", Name: "run_shell", Arguments: `{}`}, "T2")
	if unknown.Evidence.Verified || !strings.Contains(unknown.Output.Output, "unknown_tool") {
		t.Fatalf("unknown execution = %#v", unknown)
	}
}
