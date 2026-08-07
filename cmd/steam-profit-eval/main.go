package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"steam-profit-calculator/internal/assistant"
	"steam-profit-calculator/internal/evaluation"
)

func main() {
	os.Exit(run())
}

func run() int {
	flags := flag.NewFlagSet("steam-profit-eval", flag.ContinueOnError)
	live := flags.Bool("live", false, "explicitly allow configured live model calls")
	prompt := flags.String("prompt-version", "current", "current, baseline, or an immutable prompt version")
	maxCases := flags.Int("max-cases", 3, "stable suite prefix to run; 0 runs all cases")
	output := flags.String("output", ".build/evaluations/report.json", "JSON report path; empty writes stdout")
	timeout := flags.Duration("timeout", 3*time.Minute, "whole evaluation timeout")
	minPassRate := flags.Float64("min-pass-rate", 1, "minimum aggregate pass rate")
	maxP95MS := flags.Int64("max-p95-ms", 25000, "maximum aggregate p95 latency in milliseconds; 0 disables")
	maxAverageTokens := flags.Float64("max-average-tokens", 2500, "maximum average total tokens; 0 disables")
	inputRate := flags.Float64("input-usd-per-million", 0, "caller-supplied input-token rate")
	outputRate := flags.Float64("output-usd-per-million", 0, "caller-supplied output-token rate")
	maxCost := flags.Float64("max-cost-usd", 0, "maximum estimated suite cost; 0 disables")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return 2
	}
	if !*live {
		fmt.Fprintln(os.Stderr, "live evaluation requires the explicit --live flag")
		return 2
	}
	if *maxCases < 0 || *minPassRate < 0 || *minPassRate > 1 || *inputRate < 0 || *outputRate < 0 || *maxCost < 0 {
		fmt.Fprintln(os.Stderr, "evaluation limits and pricing must be non-negative; min-pass-rate must be between 0 and 1")
		return 2
	}

	promptVersion, err := resolvePromptVersion(*prompt)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	model := assistant.NewModelFromEnvironment()
	if !model.Status().Available {
		fmt.Fprintf(os.Stderr, "model unavailable: %s\n", model.Status().Reason)
		return 2
	}
	runner, err := assistant.NewToolRunner(evaluation.FixtureLookup{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "create evaluation tools failed")
		return 2
	}
	service, err := assistant.NewServiceForPrompt(model, runner, promptVersion)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	cases, err := evaluation.LoadSuite(*maxCases)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	report, runErr := evaluation.Run(ctx, service, cases, evaluation.Options{
		Thresholds: evaluation.Thresholds{
			MinPassRate: *minPassRate, MaxP95ElapsedMS: *maxP95MS,
			MaxAverageTokens: *maxAverageTokens, MaxEstimatedCostUSD: *maxCost,
		},
		Pricing: evaluation.Pricing{InputUSDPerMillion: *inputRate, OutputUSDPerMillion: *outputRate},
	})
	if err := writeReport(*output, report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if runErr != nil {
		fmt.Fprintln(os.Stderr, "evaluation did not complete")
		return 1
	}
	if !report.Passed {
		fmt.Fprintln(os.Stderr, "evaluation thresholds failed; inspect the JSON report")
		return 1
	}
	fmt.Fprintf(os.Stdout, "evaluation passed: %d/%d cases, prompt %s\n", report.Summary.Passed, report.Summary.Total, report.PromptVersion)
	return 0
}

func resolvePromptVersion(value string) (string, error) {
	switch value {
	case "current", assistant.PromptVersion:
		return assistant.PromptVersion, nil
	case "baseline", assistant.BaselinePromptVersion:
		return assistant.BaselinePromptVersion, nil
	default:
		return "", fmt.Errorf("unsupported prompt version %q", value)
	}
}

func writeReport(path string, report evaluation.Report) error {
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("encode evaluation report: %w", err)
	}
	encoded = append(encoded, '\n')
	if path == "" {
		_, err = os.Stdout.Write(encoded)
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create report directory: %w", err)
	}
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		return fmt.Errorf("write evaluation report: %w", err)
	}
	return nil
}
