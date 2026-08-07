package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"steam-profit-calculator/internal/domain"
	"steam-profit-calculator/internal/lookup"
)

type evaluationCase struct {
	ID        string          `json:"id"`
	Tool      string          `json:"tool"`
	Arguments json.RawMessage `json:"arguments"`
	Verified  bool            `json:"verified"`
	Contains  []string        `json:"contains"`
}

type evaluationLookup struct{}

func (evaluationLookup) Lookup(_ context.Context, request domain.ItemLookupRequest) ([]domain.ItemSummary, error) {
	switch request.Query {
	case "missing":
		return nil, lookup.ErrNoMatches
	case "provider-down":
		return nil, errors.New("provider unavailable")
	}
	limit := domain.DefaultLookupLimit()
	if request.Limit != nil {
		limit = *request.Limit
	}
	return []domain.ItemSummary{{
		ItemID: "fixture-1", CanonicalName: request.Query + " | Verified", AppName: "Counter-Strike 2",
		QueryTime: "2026-08-06T00:00:00Z", SalePriceText: "$10.00", SellPriceText: "$9.90", SellListings: limit * 2,
	}}, nil
}

func TestFixedEvaluationBaseline(t *testing.T) {
	raw, err := os.ReadFile("testdata/evaluations.json")
	if err != nil {
		t.Fatalf("read evaluation fixture: %v", err)
	}
	var cases []evaluationCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("decode evaluation fixture: %v", err)
	}
	if len(cases) != 50 {
		t.Fatalf("evaluation cases = %d, want exactly 50", len(cases))
	}

	runner, err := NewToolRunner(evaluationLookup{})
	if err != nil {
		t.Fatalf("NewToolRunner() error = %v", err)
	}
	seen := make(map[string]bool, len(cases))
	for index, evaluation := range cases {
		t.Run(evaluation.ID, func(t *testing.T) {
			if evaluation.ID == "" || seen[evaluation.ID] {
				t.Fatalf("evaluation ID %q is empty or duplicated", evaluation.ID)
			}
			seen[evaluation.ID] = true
			execution := runner.execute(context.Background(), ToolCall{
				CallID: "eval-call", Name: evaluation.Tool, Arguments: string(evaluation.Arguments),
			}, "T1")
			if execution.Evidence.Verified != evaluation.Verified {
				t.Fatalf("Verified = %v, want %v; output = %s", execution.Evidence.Verified, evaluation.Verified, execution.Output.Output)
			}
			for _, expected := range evaluation.Contains {
				if !strings.Contains(execution.Output.Output, expected) {
					t.Fatalf("output = %s, want substring %q", execution.Output.Output, expected)
				}
			}
			if execution.Evidence.ID != "T1" || execution.Output.CallID != "eval-call" {
				t.Fatalf("execution IDs = %#v", execution)
			}
		})
		_ = index
	}
}
