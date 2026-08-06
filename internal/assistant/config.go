package assistant

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAIBaseURL         = "https://api.openai.com/v1"
	defaultAIModel           = "gpt-5.6-terra"
	defaultAITimeout         = 25 * time.Second
	defaultReasoningEffort   = "low"
	defaultResponseVerbosity = "low"
	defaultMaxOutputTokens   = 1200
)

// OpenAIConfig configures the optional Responses API adapter.
type OpenAIConfig struct {
	APIKey          string
	BaseURL         string
	Model           string
	ReasoningEffort string
	Verbosity       string
	MaxOutputTokens int
	Timeout         time.Duration
	HTTPClient      *http.Client
}

// NewModelFromEnvironment builds an optional model without making server startup depend on it.
func NewModelFromEnvironment() Model {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("STEAM_PROFIT_AI_PROVIDER")))
	if provider == "" || provider == "disabled" || provider == "off" {
		return NewUnavailableModel("disabled", "", "AI 策略助手未启用")
	}
	if provider != "openai" {
		return NewUnavailableModel(provider, "", "不支持的 AI Provider")
	}

	apiKey := strings.TrimSpace(os.Getenv("STEAM_PROFIT_AI_API_KEY"))
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	}
	model := environmentOrDefault("STEAM_PROFIT_AI_MODEL", defaultAIModel)
	if apiKey == "" {
		return NewUnavailableModel("openai", model, "缺少 STEAM_PROFIT_AI_API_KEY 或 OPENAI_API_KEY")
	}

	timeout, err := environmentDurationMS("STEAM_PROFIT_AI_TIMEOUT_MS", defaultAITimeout)
	if err != nil {
		return NewUnavailableModel("openai", model, err.Error())
	}
	maxOutputTokens, err := environmentInteger("STEAM_PROFIT_AI_MAX_OUTPUT_TOKENS", defaultMaxOutputTokens, 128, 4096)
	if err != nil {
		return NewUnavailableModel("openai", model, err.Error())
	}
	configured, err := NewOpenAIModel(OpenAIConfig{
		APIKey: apiKey, BaseURL: environmentOrDefault("STEAM_PROFIT_AI_BASE_URL", defaultAIBaseURL),
		Model: model, ReasoningEffort: environmentOrDefault("STEAM_PROFIT_AI_REASONING_EFFORT", defaultReasoningEffort),
		Verbosity:       environmentOrDefault("STEAM_PROFIT_AI_VERBOSITY", defaultResponseVerbosity),
		MaxOutputTokens: maxOutputTokens, Timeout: timeout,
	})
	if err != nil {
		return NewUnavailableModel("openai", model, err.Error())
	}
	return configured
}

// NewOpenAIModel validates configuration and creates an OpenAI Responses API model.
func NewOpenAIModel(config OpenAIConfig) (Model, error) {
	config.APIKey = strings.TrimSpace(config.APIKey)
	config.BaseURL = strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	config.Model = strings.TrimSpace(config.Model)
	config.ReasoningEffort = strings.TrimSpace(config.ReasoningEffort)
	config.Verbosity = strings.TrimSpace(config.Verbosity)
	if config.APIKey == "" {
		return nil, fmt.Errorf("AI API key is required")
	}
	if config.BaseURL == "" {
		config.BaseURL = defaultAIBaseURL
	}
	parsed, err := url.Parse(config.BaseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return nil, fmt.Errorf("AI base URL must be an absolute HTTP(S) URL")
	}
	if config.Model == "" {
		config.Model = defaultAIModel
	}
	if config.ReasoningEffort == "" {
		config.ReasoningEffort = defaultReasoningEffort
	}
	if !oneOf(config.ReasoningEffort, "none", "low", "medium", "high", "xhigh", "max") {
		return nil, fmt.Errorf("AI reasoning effort must be none, low, medium, high, xhigh, or max")
	}
	if config.Verbosity == "" {
		config.Verbosity = defaultResponseVerbosity
	}
	if !oneOf(config.Verbosity, "low", "medium", "high") {
		return nil, fmt.Errorf("AI verbosity must be low, medium, or high")
	}
	if config.MaxOutputTokens == 0 {
		config.MaxOutputTokens = defaultMaxOutputTokens
	}
	if config.MaxOutputTokens < 128 || config.MaxOutputTokens > 4096 {
		return nil, fmt.Errorf("AI max output tokens must be between 128 and 4096")
	}
	if config.Timeout <= 0 {
		config.Timeout = defaultAITimeout
	}
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: config.Timeout}
	}
	return &openAIModel{config: config}, nil
}

func environmentOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func environmentDurationMS(name string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	milliseconds, err := strconv.Atoi(raw)
	if err != nil || milliseconds < 1000 || milliseconds > 120000 {
		return 0, fmt.Errorf("%s must be an integer between 1000 and 120000", name)
	}
	return time.Duration(milliseconds) * time.Millisecond, nil
}

func environmentInteger(name string, fallback, minimum, maximum int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || value > maximum {
		return 0, fmt.Errorf("%s must be an integer between %d and %d", name, minimum, maximum)
	}
	return value, nil
}

func oneOf(value string, options ...string) bool {
	for _, option := range options {
		if value == option {
			return true
		}
	}
	return false
}
