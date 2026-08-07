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

func TestSteamAPIsClientLookupNormalizesDirectSearch(t *testing.T) {
	const observedAt = "Fri, 07 Aug 2026 08:40:00 GMT"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/v2/items/search" {
			t.Fatalf("path = %q, want /v2/items/search", got)
		}
		if got := r.URL.Query().Get("game"); got != "CS2" {
			t.Fatalf("game = %q, want CS2", got)
		}
		if got := r.URL.Query().Get("query"); got != "AK-47 Redline" {
			t.Fatalf("query = %q, want AK-47 Redline", got)
		}
		if got := r.URL.Query().Get("limit"); got != "2" {
			t.Fatalf("limit = %q, want 2", got)
		}
		if got := r.Header.Get("X-API-Key"); got != "test-key" {
			t.Fatalf("X-API-Key = %q, want test-key", got)
		}
		if got := r.Header.Get("User-Agent"); got != "steam-profit-tests/steamapis" {
			t.Fatalf("User-Agent = %q", got)
		}

		w.Header().Set("Date", observedAt)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{
				"name":"AK-47 | Redline (Field-Tested)",
				"marketplace":"Buff163",
				"game":"CS2",
				"priceUSD":45.99,
				"offerCount":128,
				"thumbnail":"https://example.com/redline.png",
				"updatedAt":1711152000
			},
			{
				"name":"AK-47 | Redline (Minimal Wear)",
				"marketplace":"SkinPort",
				"game":"CS2",
				"priceUSD":null,
				"offerCount":0,
				"updatedAt":1711151000
			}
		]`))
	}))
	defer server.Close()

	client := newSteamAPIsTestClient(t, server.URL)
	items, err := client.Lookup(context.Background(), domain.ItemLookupRequest{
		Query: "AK-47 Redline",
		Limit: intPtr(2),
	})
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if !strings.HasPrefix(items[0].ItemID, "steamapis:") {
		t.Fatalf("ItemID = %q, want steamapis prefix", items[0].ItemID)
	}
	if items[0].CanonicalName != "AK-47 | Redline (Field-Tested)" {
		t.Fatalf("CanonicalName = %q", items[0].CanonicalName)
	}
	if items[0].AppName != "Counter-Strike 2" {
		t.Fatalf("AppName = %q, want Counter-Strike 2", items[0].AppName)
	}
	if items[0].QueryTime != "2026-08-07T08:40:00Z" || items[1].QueryTime != items[0].QueryTime {
		t.Fatalf("QueryTime values = %q/%q", items[0].QueryTime, items[1].QueryTime)
	}
	if items[0].SalePriceText != "" {
		t.Fatalf("SalePriceText = %q, want empty", items[0].SalePriceText)
	}
	if items[0].SellPriceText != "$45.99 (Buff163)" {
		t.Fatalf("SellPriceText = %q", items[0].SellPriceText)
	}
	if items[0].SellListings != 128 {
		t.Fatalf("SellListings = %d, want 128", items[0].SellListings)
	}
	if items[1].SellPriceText != "" {
		t.Fatalf("null price SellPriceText = %q, want empty", items[1].SellPriceText)
	}
}

func TestSteamAPIsClientEmptySearchBecomesNoMatches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	service := lookup.NewService(newSteamAPIsTestClient(t, server.URL))
	_, err := service.Lookup(context.Background(), domain.ItemLookupRequest{Query: "missing"})
	if !errors.Is(err, lookup.ErrNoMatches) {
		t.Fatalf("Lookup() error = %v, want ErrNoMatches", err)
	}
}

func TestSteamAPIsClientRejectsIncompleteConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		config  lookup.Config
		wantErr string
	}{
		{
			name: "missing key",
			config: lookup.Config{
				BaseURL:      "https://marketplaceapi.steamapis.com",
				AppID:        730,
				Timeout:      time.Second,
				APIKeyHeader: "X-API-Key",
			},
			wantErr: "requires STEAM_PROFIT_LOOKUP_API_KEY",
		},
		{
			name: "unsupported app",
			config: lookup.Config{
				BaseURL:      "https://marketplaceapi.steamapis.com",
				AppID:        999,
				Timeout:      time.Second,
				APIKey:       "test-key",
				APIKeyHeader: "X-API-Key",
			},
			wantErr: "does not support app id 999",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := lookup.NewSteamAPIsClient(test.config)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("NewSteamAPIsClient() error = %v, want %q", err, test.wantErr)
			}
		})
	}
}

func TestSteamAPIsClientReturnsDependencyFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "secret upstream detail", http.StatusBadGateway)
	}))
	defer server.Close()

	client := newSteamAPIsTestClient(t, server.URL)
	_, err := client.Lookup(context.Background(), domain.ItemLookupRequest{Query: "AK-47"})
	if err == nil || !strings.Contains(err.Error(), "unexpected status 502 Bad Gateway") {
		t.Fatalf("Lookup() error = %v, want dependency status", err)
	}
	if strings.Contains(err.Error(), "secret upstream detail") {
		t.Fatalf("Lookup() error leaked response body: %v", err)
	}
}

func TestSteamAPIsClientRetriesBoundedRateLimit(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Retry-After", "0")
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := newSteamAPIsTestClient(t, server.URL)
	_, err := client.Lookup(context.Background(), domain.ItemLookupRequest{Query: "AK-47"})
	if err == nil || !strings.Contains(err.Error(), "rate limited after 3 attempts") {
		t.Fatalf("Lookup() error = %v, want bounded rate-limit failure", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

func TestSteamAPIsClientRejectsInvalidPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{
			"name":"AK-47 | Redline (Field-Tested)",
			"marketplace":"",
			"game":"CS2",
			"priceUSD":45.99,
			"offerCount":128
		}]`))
	}))
	defer server.Close()

	client := newSteamAPIsTestClient(t, server.URL)
	_, err := client.Lookup(context.Background(), domain.ItemLookupRequest{Query: "AK-47"})
	if !errors.Is(err, lookup.ErrInvalidProviderPayload) {
		t.Fatalf("Lookup() error = %v, want ErrInvalidProviderPayload", err)
	}
}

func newSteamAPIsTestClient(t *testing.T, baseURL string) lookup.Client {
	t.Helper()
	client, err := lookup.NewSteamAPIsClient(lookup.Config{
		BaseURL:      baseURL,
		AppID:        730,
		Timeout:      time.Second,
		APIKey:       "test-key",
		APIKeyHeader: "X-API-Key",
		UserAgent:    "steam-profit-tests/steamapis",
	})
	if err != nil {
		t.Fatalf("NewSteamAPIsClient() error = %v", err)
	}
	return client
}
