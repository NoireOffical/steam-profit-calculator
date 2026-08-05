package lookup

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultProvider     = ProviderSteamPublic
	defaultBaseURL      = "https://steamcommunity.com"
	defaultAppID        = 730
	defaultTimeout      = 5 * time.Second
	defaultAPIKeyHeader = "X-API-Key"
	defaultUserAgent    = "steam-profit-calculator/lookup"

	envLookupProvider     = "STEAM_PROFIT_LOOKUP_PROVIDER"
	envLookupBaseURL      = "STEAM_PROFIT_LOOKUP_BASE_URL"
	envLookupAppID        = "STEAM_PROFIT_LOOKUP_APP_ID"
	envLookupTimeoutMS    = "STEAM_PROFIT_LOOKUP_TIMEOUT_MS"
	envLookupAPIKey       = "STEAM_PROFIT_LOOKUP_API_KEY"
	envLookupAPIKeyHeader = "STEAM_PROFIT_LOOKUP_API_KEY_HEADER"
	envLookupUserAgent    = "STEAM_PROFIT_LOOKUP_USER_AGENT"
)

// Config defines how the external lookup provider is reached.
type Config struct {
	Provider     string
	BaseURL      string
	AppID        int
	Timeout      time.Duration
	APIKey       string
	APIKeyHeader string
	UserAgent    string
}

// LoadConfigFromEnvironment loads lookup provider configuration from environment variables.
func LoadConfigFromEnvironment() (Config, error) {
	cfg := Config{
		Provider:     defaultProvider,
		BaseURL:      defaultBaseURL,
		AppID:        defaultAppID,
		Timeout:      defaultTimeout,
		APIKey:       strings.TrimSpace(os.Getenv(envLookupAPIKey)),
		APIKeyHeader: defaultAPIKeyHeader,
		UserAgent:    defaultUserAgent,
	}

	if provider := strings.TrimSpace(os.Getenv(envLookupProvider)); provider != "" {
		cfg.Provider = provider
	}
	if baseURL := strings.TrimSpace(os.Getenv(envLookupBaseURL)); baseURL != "" {
		cfg.BaseURL = baseURL
	}
	if rawAppID := strings.TrimSpace(os.Getenv(envLookupAppID)); rawAppID != "" {
		appID, err := strconv.Atoi(rawAppID)
		if err != nil {
			return Config{}, fmt.Errorf("parse %s %q: %w", envLookupAppID, rawAppID, err)
		}
		if appID <= 0 {
			return Config{}, fmt.Errorf("%s must be greater than 0", envLookupAppID)
		}
		cfg.AppID = appID
	}
	if rawTimeout := strings.TrimSpace(os.Getenv(envLookupTimeoutMS)); rawTimeout != "" {
		timeoutMS, err := strconv.Atoi(rawTimeout)
		if err != nil {
			return Config{}, fmt.Errorf("parse %s %q: %w", envLookupTimeoutMS, rawTimeout, err)
		}
		if timeoutMS <= 0 {
			return Config{}, fmt.Errorf("%s must be greater than 0", envLookupTimeoutMS)
		}
		cfg.Timeout = time.Duration(timeoutMS) * time.Millisecond
	}
	if header := strings.TrimSpace(os.Getenv(envLookupAPIKeyHeader)); header != "" {
		cfg.APIKeyHeader = header
	}
	if userAgent := strings.TrimSpace(os.Getenv(envLookupUserAgent)); userAgent != "" {
		cfg.UserAgent = userAgent
	}

	return cfg, nil
}
