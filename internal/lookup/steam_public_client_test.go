package lookup_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"steam-profit-calculator/internal/domain"
	"steam-profit-calculator/internal/lookup"
)

func TestSteamPublicClientLookupNormalizesResults(t *testing.T) {
	const observedAt = "Tue, 14 Jul 2026 06:15:30 GMT"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/market/search/render" {
			t.Fatalf("path = %q, want %q", got, "/market/search/render")
		}
		if got := r.URL.Query().Get("query"); got != "AK-47" {
			t.Fatalf("query = %q, want %q", got, "AK-47")
		}
		if got := r.URL.Query().Get("count"); got != "2" {
			t.Fatalf("count = %q, want %q", got, "2")
		}
		if got := r.Header.Get("User-Agent"); got != "steam-profit-tests/lookup" {
			t.Fatalf("User-Agent = %q, want steam-profit-tests/lookup", got)
		}
		w.Header().Set("Date", observedAt)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true,
			"results": [
				{
					"name": "AK-47 | Aphrodite (Battle-Scarred)",
					"hash_name": "AK-47 | Aphrodite (Battle-Scarred)",
                                        "sell_listings": 17,
                                        "sell_price_text": "$19.12",
                                        "sale_price_text": "$18.45",
					"app_name": "Counter-Strike 2",
					"asset_description": {
						"classid": "7993054046",
                                                "market_hash_name": "AK-47 | Aphrodite (Battle-Scarred)"
					}
				},
				{
					"name": "AK-47 | Asiimov (Battle-Scarred)",
					"hash_name": "AK-47 | Asiimov (Battle-Scarred)",
                                        "sell_listings": 42,
                                        "sell_price_text": "$11.06",
                                        "sale_price_text": "$10.52",
					"app_name": "Counter-Strike 2",
					"asset_description": {
						"classid": "7151910841",
                                                "market_hash_name": "AK-47 | Asiimov (Battle-Scarred)"
					}
				}
			]
		}`))
	}))
	defer server.Close()

	client, err := lookup.NewSteamPublicClient(lookup.Config{
		BaseURL:   server.URL,
		AppID:     730,
		Timeout:   time.Second,
		UserAgent: "steam-profit-tests/lookup",
	})
	if err != nil {
		t.Fatalf("NewSteamPublicClient() error = %v", err)
	}

	items, err := client.Lookup(context.Background(), domain.ItemLookupRequest{
		Query: "AK-47",
		Limit: intPtr(2),
	})
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].ItemID != "7993054046" {
		t.Fatalf("items[0].ItemID = %q, want 7993054046", items[0].ItemID)
	}
	if items[0].CanonicalName != "AK-47 | Aphrodite (Battle-Scarred)" {
		t.Fatalf("items[0].CanonicalName = %q", items[0].CanonicalName)
	}
	if items[0].AppName != "Counter-Strike 2" {
		t.Fatalf("items[0].AppName = %q, want Counter-Strike 2", items[0].AppName)
	}
	if items[0].QueryTime != "2026-07-14T06:15:30Z" {
		t.Fatalf("items[0].QueryTime = %q, want 2026-07-14T06:15:30Z", items[0].QueryTime)
	}
	if items[0].SalePriceText != "$18.45" {
		t.Fatalf("items[0].SalePriceText = %q, want $18.45", items[0].SalePriceText)
	}
	if items[0].SellPriceText != "$19.12" {
		t.Fatalf("items[0].SellPriceText = %q, want $19.12", items[0].SellPriceText)
	}
	if items[0].SellListings != 17 {
		t.Fatalf("items[0].SellListings = %d, want 17", items[0].SellListings)
	}
	if items[1].QueryTime != items[0].QueryTime {
		t.Fatalf("items[1].QueryTime = %q, want %q", items[1].QueryTime, items[0].QueryTime)
	}
}

func TestSteamPublicClientLookupProviderFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
	}))
	defer server.Close()

	client, err := lookup.NewSteamPublicClient(lookup.Config{
		BaseURL: server.URL,
		AppID:   730,
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewSteamPublicClient() error = %v", err)
	}

	_, err = client.Lookup(context.Background(), domain.ItemLookupRequest{Query: "AK-47"})
	if err == nil {
		t.Fatal("Lookup() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "unexpected status 502 Bad Gateway") {
		t.Fatalf("Lookup() error = %q, want provider failure", err)
	}
}

func TestSteamPublicClientLookupRetriesRateLimit(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.Header().Set("Retry-After", "0")
			http.Error(w, "rate limited", http.StatusTooManyRequests)
			return
		}

		w.Header().Set("Date", "Tue, 14 Jul 2026 06:17:00 GMT")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true,
			"results": [
				{
					"name": "AK-47 | Head Shot (Field-Tested)",
					"hash_name": "AK-47 | Head Shot (Field-Tested)",
					"sell_listings": 254,
					"sell_price_text": "$1.30",
					"sale_price_text": "$1.25",
					"app_name": "Counter-Strike 2",
					"asset_description": {
						"classid": "7246719818",
						"market_hash_name": "AK-47 | Head Shot (Field-Tested)"
					}
				}
			]
		}`))
	}))
	defer server.Close()

	client, err := lookup.NewSteamPublicClient(lookup.Config{
		BaseURL: server.URL,
		AppID:   730,
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewSteamPublicClient() error = %v", err)
	}

	items, err := client.Lookup(context.Background(), domain.ItemLookupRequest{Query: "AK-47"})
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].QueryTime != "2026-07-14T06:17:00Z" {
		t.Fatalf("items[0].QueryTime = %q, want 2026-07-14T06:17:00Z", items[0].QueryTime)
	}
}

func TestSteamPublicClientLookupFailsAfterRateLimitRetries(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Retry-After", "0")
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer server.Close()

	client, err := lookup.NewSteamPublicClient(lookup.Config{
		BaseURL: server.URL,
		AppID:   730,
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewSteamPublicClient() error = %v", err)
	}

	_, err = client.Lookup(context.Background(), domain.ItemLookupRequest{Query: "AK-47"})
	if err == nil {
		t.Fatal("Lookup() error = nil, want non-nil")
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
	if !strings.Contains(err.Error(), "rate limited after 3 attempts") {
		t.Fatalf("Lookup() error = %q, want rate limit retry message", err)
	}
	if !errors.Is(err, lookup.ErrSteamOfficialRateLimit) {
		t.Fatalf("Lookup() error = %q, want ErrSteamOfficialRateLimit", err)
	}
}

func TestSteamPublicClientLookupRejectsInvalidPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true,
			"results": [
				{
					"name": "AK-47 | Aphrodite (Battle-Scarred)",
					"hash_name": "AK-47 | Aphrodite (Battle-Scarred)",
                                        "sell_listings": 17,
                                        "sell_price_text": "$19.12",
                                        "sale_price_text": "$18.45",
					"asset_description": {
                                                "classid": "7993054046",
                                                "market_hash_name": "AK-47 | Aphrodite (Battle-Scarred)"
					}
				}
			]
		}`))
	}))
	defer server.Close()

	client, err := lookup.NewSteamPublicClient(lookup.Config{
		BaseURL: server.URL,
		AppID:   730,
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewSteamPublicClient() error = %v", err)
	}

	_, err = client.Lookup(context.Background(), domain.ItemLookupRequest{Query: "AK-47"})
	if err == nil {
		t.Fatal("Lookup() error = nil, want non-nil")
	}
	if !errors.Is(err, lookup.ErrInvalidProviderPayload) {
		t.Fatalf("Lookup() error = %v, want invalid payload sentinel", err)
	}
}

func TestLoadConfigFromEnvironmentLoadsOverrides(t *testing.T) {
	t.Setenv("STEAM_PROFIT_LOOKUP_PROVIDER", "steam_public")
	t.Setenv("STEAM_PROFIT_LOOKUP_BASE_URL", "https://example.com")
	t.Setenv("STEAM_PROFIT_LOOKUP_APP_ID", "570")
	t.Setenv("STEAM_PROFIT_LOOKUP_TIMEOUT_MS", "2500")
	t.Setenv("STEAM_PROFIT_LOOKUP_API_KEY", "secret")
	t.Setenv("STEAM_PROFIT_LOOKUP_API_KEY_HEADER", "Authorization")
	t.Setenv("STEAM_PROFIT_LOOKUP_USER_AGENT", "steam-profit-tests/custom")

	cfg, err := lookup.LoadConfigFromEnvironment()
	if err != nil {
		t.Fatalf("LoadConfigFromEnvironment() error = %v", err)
	}
	if cfg.Provider != "steam_public" {
		t.Fatalf("Provider = %q, want steam_public", cfg.Provider)
	}
	if cfg.BaseURL != "https://example.com" {
		t.Fatalf("BaseURL = %q, want https://example.com", cfg.BaseURL)
	}
	if cfg.AppID != 570 {
		t.Fatalf("AppID = %d, want 570", cfg.AppID)
	}
	if cfg.Timeout != 2500*time.Millisecond {
		t.Fatalf("Timeout = %v, want 2500ms", cfg.Timeout)
	}
	if cfg.APIKey != "secret" {
		t.Fatalf("APIKey = %q, want secret", cfg.APIKey)
	}
	if cfg.APIKeyHeader != "Authorization" {
		t.Fatalf("APIKeyHeader = %q, want Authorization", cfg.APIKeyHeader)
	}
	if cfg.UserAgent != "steam-profit-tests/custom" {
		t.Fatalf("UserAgent = %q, want steam-profit-tests/custom", cfg.UserAgent)
	}
}

func TestLoadConfigFromEnvironmentDefaultsProvider(t *testing.T) {
	cfg, err := lookup.LoadConfigFromEnvironment()
	if err != nil {
		t.Fatalf("LoadConfigFromEnvironment() error = %v", err)
	}
	if cfg.Provider != lookup.ProviderSteamPublic {
		t.Fatalf("Provider = %q, want %q", cfg.Provider, lookup.ProviderSteamPublic)
	}
	if cfg.BaseURL != "https://steamcommunity.com" {
		t.Fatalf("BaseURL = %q, want Steam public default", cfg.BaseURL)
	}
}

func TestLoadConfigFromEnvironmentUsesSteamAPIsDefaultBaseURL(t *testing.T) {
	t.Setenv("STEAM_PROFIT_LOOKUP_PROVIDER", "steamapis")
	t.Setenv("STEAM_PROFIT_LOOKUP_BASE_URL", "")

	cfg, err := lookup.LoadConfigFromEnvironment()
	if err != nil {
		t.Fatalf("LoadConfigFromEnvironment() error = %v", err)
	}
	if cfg.BaseURL != "https://marketplaceapi.steamapis.com" {
		t.Fatalf("BaseURL = %q, want SteamApis default", cfg.BaseURL)
	}
}

func TestNewClientUsesSupportedProviderSelection(t *testing.T) {
	client, err := lookup.NewClient(lookup.Config{
		Provider: lookup.ProviderSteamPublic,
		BaseURL:  "https://steamcommunity.com",
		AppID:    730,
		Timeout:  time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}
}

func TestNewClientUsesSteamAPIsProviderSelection(t *testing.T) {
	client, err := lookup.NewClient(lookup.Config{
		Provider:     lookup.ProviderSteamAPIs,
		BaseURL:      "https://marketplaceapi.steamapis.com",
		AppID:        730,
		Timeout:      time.Second,
		APIKey:       "test-key",
		APIKeyHeader: "X-API-Key",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}
}

func TestNewClientRejectsUnsupportedProvider(t *testing.T) {
	_, err := lookup.NewClient(lookup.Config{
		Provider: "proxy",
		BaseURL:  "https://example.com",
		AppID:    730,
		Timeout:  time.Second,
	})
	if err == nil {
		t.Fatal("NewClient() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), `unsupported lookup provider "proxy"`) {
		t.Fatalf("NewClient() error = %q, want unsupported provider message", err)
	}
}

func intPtr(value int) *int {
	return &value
}
