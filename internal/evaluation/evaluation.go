// Package evaluation measures assistant behavior without making model output authoritative.
package evaluation

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"steam-profit-calculator/internal/assistant"
)

const (
	ReportSchemaVersion = "steam-profit-ai-eval-report-v1"
	SuiteVersion        = "steam-profit-semantic-eval-v1"
)

//go:embed testdata/semantic.json
var suiteFiles embed.FS

// Case is one versioned semantic model scenario. Prompt text stays in the
// fixture and is intentionally omitted from reports.
type Case struct {
	ID              string                          `json:"id"`
	Message         string                          `json:"message"`
	History         []assistant.ConversationMessage `json:"history,omitempty"`
	ExpectedTools   []string                        `json:"expected_tools"`
	RequireGrounded bool                            `json:"require_grounded"`
	MaxElapsedMS    int64                           `json:"max_elapsed_ms"`
	MaxTotalTokens  int                             `json:"max_total_tokens"`
}

// Pricing contains optional caller-supplied rates rather than a hard-coded price table.
type Pricing struct {
	InputUSDPerMillion  float64 `json:"input_usd_per_million,omitempty"`
	OutputUSDPerMillion float64 `json:"output_usd_per_million,omitempty"`
}

// Thresholds are aggregate report gates.
type Thresholds struct {
	MinPassRate         float64 `json:"min_pass_rate"`
	MaxP95ElapsedMS     int64   `json:"max_p95_elapsed_ms,omitempty"`
	MaxAverageTokens    float64 `json:"max_average_tokens,omitempty"`
	MaxEstimatedCostUSD float64 `json:"max_estimated_cost_usd,omitempty"`
}

// Options configures one report run.
type Options struct {
	Thresholds Thresholds
	Pricing    Pricing
}

// CaseResult contains observable metadata only, not prompts or tool values.
type CaseResult struct {
	ID               string          `json:"id"`
	Passed           bool            `json:"passed"`
	ObservedTools    []string        `json:"observed_tools"`
	Grounded         bool            `json:"grounded"`
	NativeStreaming  bool            `json:"native_streaming"`
	ElapsedMS        int64           `json:"elapsed_ms"`
	Usage            assistant.Usage `json:"usage"`
	EstimatedCostUSD float64         `json:"estimated_cost_usd,omitempty"`
	FailureCodes     []string        `json:"failure_codes,omitempty"`
}

// Summary aggregates one versioned evaluation run.
type Summary struct {
	Total             int      `json:"total"`
	Passed            int      `json:"passed"`
	PassRate          float64  `json:"pass_rate"`
	P95ElapsedMS      int64    `json:"p95_elapsed_ms"`
	AverageTokens     float64  `json:"average_tokens"`
	TotalInputTokens  int      `json:"total_input_tokens"`
	TotalOutputTokens int      `json:"total_output_tokens"`
	EstimatedCostUSD  float64  `json:"estimated_cost_usd,omitempty"`
	ThresholdFailures []string `json:"threshold_failures,omitempty"`
}

// Report is the machine-readable, content-free evaluation artifact.
type Report struct {
	SchemaVersion string       `json:"schema_version"`
	SuiteVersion  string       `json:"suite_version"`
	GeneratedAt   string       `json:"generated_at"`
	Provider      string       `json:"provider"`
	Model         string       `json:"model,omitempty"`
	PromptVersion string       `json:"prompt_version"`
	Thresholds    Thresholds   `json:"thresholds"`
	Pricing       Pricing      `json:"pricing"`
	Cases         []CaseResult `json:"cases"`
	Summary       Summary      `json:"summary"`
	Passed        bool         `json:"passed"`
}

// LoadSuite reads and validates the embedded semantic suite. A positive limit
// selects a stable prefix for a low-cost smoke run.
func LoadSuite(limit int) ([]Case, error) {
	raw, err := suiteFiles.ReadFile("testdata/semantic.json")
	if err != nil {
		return nil, fmt.Errorf("read semantic evaluation suite: %w", err)
	}
	var cases []Case
	if err := json.Unmarshal(raw, &cases); err != nil {
		return nil, fmt.Errorf("decode semantic evaluation suite: %w", err)
	}
	if len(cases) == 0 {
		return nil, fmt.Errorf("semantic evaluation suite is empty")
	}
	seen := make(map[string]bool, len(cases))
	for _, evaluation := range cases {
		if evaluation.ID == "" || seen[evaluation.ID] {
			return nil, fmt.Errorf("semantic evaluation ID %q is empty or duplicated", evaluation.ID)
		}
		seen[evaluation.ID] = true
		if err := assistant.ValidateMessage(evaluation.Message); err != nil {
			return nil, fmt.Errorf("semantic evaluation %s has invalid message", evaluation.ID)
		}
		if _, err := assistant.ValidateHistory(evaluation.History); err != nil {
			return nil, fmt.Errorf("semantic evaluation %s has invalid history", evaluation.ID)
		}
	}
	if limit > 0 && limit < len(cases) {
		cases = cases[:limit]
	}
	return cases, nil
}

// Run executes cases sequentially so usage and latency remain attributable.
func Run(ctx context.Context, service *assistant.Service, cases []Case, options Options) (Report, error) {
	if service == nil {
		return Report{}, fmt.Errorf("assistant service is required")
	}
	if len(cases) == 0 {
		return Report{}, fmt.Errorf("at least one evaluation case is required")
	}
	status := service.Status()
	report := Report{
		SchemaVersion: ReportSchemaVersion,
		SuiteVersion:  SuiteVersion,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Provider:      status.Provider,
		Model:         status.Model,
		PromptVersion: status.PromptVersion,
		Thresholds:    options.Thresholds,
		Pricing:       options.Pricing,
		Cases:         make([]CaseResult, 0, len(cases)),
	}

	for _, evaluation := range cases {
		result := runCase(ctx, service, evaluation, options.Pricing)
		report.Cases = append(report.Cases, result)
		if ctx.Err() != nil {
			return report, ctx.Err()
		}
	}
	report.Summary = summarize(report.Cases, options.Thresholds)
	report.Passed = len(report.Summary.ThresholdFailures) == 0
	return report, nil
}

func runCase(ctx context.Context, service *assistant.Service, evaluation Case, pricing Pricing) CaseResult {
	result := CaseResult{ID: evaluation.ID, ObservedTools: []string{}}
	started := time.Now()
	err := service.RunConversation(ctx, evaluation.Message, evaluation.History, func(event assistant.Event) error {
		if event.Type == "tool_start" {
			result.ObservedTools = append(result.ObservedTools, event.Tool)
		}
		if event.Completion != nil {
			result.Grounded = event.Completion.Grounded
			result.NativeStreaming = event.Completion.NativeStreaming
			result.ElapsedMS = event.Completion.ElapsedMS
			result.Usage = event.Completion.Usage
		}
		return nil
	})
	if result.ElapsedMS == 0 {
		result.ElapsedMS = time.Since(started).Milliseconds()
	}
	if err != nil {
		result.FailureCodes = append(result.FailureCodes, classifyError(err))
	}
	if !equalStrings(result.ObservedTools, evaluation.ExpectedTools) {
		result.FailureCodes = append(result.FailureCodes, "tool_mismatch")
	}
	if evaluation.RequireGrounded && !result.Grounded {
		result.FailureCodes = append(result.FailureCodes, "ungrounded")
	}
	if evaluation.MaxElapsedMS > 0 && result.ElapsedMS > evaluation.MaxElapsedMS {
		result.FailureCodes = append(result.FailureCodes, "latency_threshold")
	}
	if evaluation.MaxTotalTokens > 0 && result.Usage.TotalTokens > evaluation.MaxTotalTokens {
		result.FailureCodes = append(result.FailureCodes, "token_threshold")
	}
	result.EstimatedCostUSD = estimateCost(result.Usage, pricing)
	result.Passed = len(result.FailureCodes) == 0
	return result
}

func summarize(results []CaseResult, thresholds Thresholds) Summary {
	summary := Summary{Total: len(results)}
	elapsed := make([]int64, 0, len(results))
	for _, result := range results {
		if result.Passed {
			summary.Passed++
		}
		elapsed = append(elapsed, result.ElapsedMS)
		summary.TotalInputTokens += result.Usage.InputTokens
		summary.TotalOutputTokens += result.Usage.OutputTokens
		summary.EstimatedCostUSD += result.EstimatedCostUSD
	}
	summary.PassRate = float64(summary.Passed) / float64(summary.Total)
	summary.AverageTokens = float64(summary.TotalInputTokens+summary.TotalOutputTokens) / float64(summary.Total)
	sort.Slice(elapsed, func(i, j int) bool { return elapsed[i] < elapsed[j] })
	index := int(math.Ceil(float64(len(elapsed))*0.95)) - 1
	if index < 0 {
		index = 0
	}
	summary.P95ElapsedMS = elapsed[index]
	if summary.PassRate < thresholds.MinPassRate {
		summary.ThresholdFailures = append(summary.ThresholdFailures, "minimum_pass_rate")
	}
	if thresholds.MaxP95ElapsedMS > 0 && summary.P95ElapsedMS > thresholds.MaxP95ElapsedMS {
		summary.ThresholdFailures = append(summary.ThresholdFailures, "p95_latency")
	}
	if thresholds.MaxAverageTokens > 0 && summary.AverageTokens > thresholds.MaxAverageTokens {
		summary.ThresholdFailures = append(summary.ThresholdFailures, "average_tokens")
	}
	if thresholds.MaxEstimatedCostUSD > 0 && summary.EstimatedCostUSD > thresholds.MaxEstimatedCostUSD {
		summary.ThresholdFailures = append(summary.ThresholdFailures, "estimated_cost")
	}
	return summary
}

func classifyError(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "cancelled"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, assistant.ErrModelUnavailable):
		return "model_unavailable"
	default:
		return "assistant_error"
	}
}

func estimateCost(usage assistant.Usage, pricing Pricing) float64 {
	return float64(usage.InputTokens)*pricing.InputUSDPerMillion/1_000_000 +
		float64(usage.OutputTokens)*pricing.OutputUSDPerMillion/1_000_000
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
