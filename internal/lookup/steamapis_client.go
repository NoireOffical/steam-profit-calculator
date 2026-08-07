package lookup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"steam-profit-calculator/internal/domain"
)

const maxSteamAPIsResponseBytes = 2 << 20

var steamAPIsGames = map[int]struct {
	code string
	name string
}{
	730:    {code: "CS2", name: "Counter-Strike 2"},
	570:    {code: "Dota2", name: "Dota 2"},
	440:    {code: "TF2", name: "Team Fortress 2"},
	252490: {code: "Rust", name: "Rust"},
}

type steamAPIsClient struct {
	baseURL       *url.URL
	gameCode      string
	appName       string
	apiKey        string
	apiKeyHeader  string
	userAgent     string
	defaultClient *http.Client
}

type steamAPIsSearchResult struct {
	Name        string   `json:"name"`
	Marketplace string   `json:"marketplace"`
	Game        string   `json:"game"`
	PriceUSD    *float64 `json:"priceUSD"`
	OfferCount  int      `json:"offerCount"`
}

// NewSteamAPIsClient creates a direct SteamApis v2 keyword-search client.
func NewSteamAPIsClient(cfg Config) (Client, error) {
	baseURL, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse lookup base url %q: %w", cfg.BaseURL, err)
	}
	if baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, fmt.Errorf("lookup base url must be absolute")
	}
	if cfg.Timeout <= 0 {
		return nil, fmt.Errorf("lookup timeout must be greater than 0")
	}
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("steamapis lookup provider requires STEAM_PROFIT_LOOKUP_API_KEY")
	}
	apiKeyHeader := strings.TrimSpace(cfg.APIKeyHeader)
	if apiKeyHeader == "" {
		return nil, fmt.Errorf("steamapis lookup provider requires STEAM_PROFIT_LOOKUP_API_KEY_HEADER")
	}
	game, ok := steamAPIsGames[cfg.AppID]
	if !ok {
		return nil, fmt.Errorf("steamapis lookup provider does not support app id %d", cfg.AppID)
	}

	return &steamAPIsClient{
		baseURL:      baseURL,
		gameCode:     game.code,
		appName:      game.name,
		apiKey:       apiKey,
		apiKeyHeader: apiKeyHeader,
		userAgent:    cfg.UserAgent,
		defaultClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}, nil
}

func (c *steamAPIsClient) Lookup(ctx context.Context, req domain.ItemLookupRequest) ([]domain.ItemSummary, error) {
	endpoint := *c.baseURL
	endpoint.Path = path.Join(endpoint.Path, "/v2/items/search")

	limit := domain.DefaultLookupLimit()
	if req.Limit != nil && *req.Limit > 0 {
		limit = *req.Limit
	}

	query := endpoint.Query()
	query.Set("game", c.gameCode)
	query.Set("query", strings.TrimSpace(req.Query))
	query.Set("limit", fmt.Sprintf("%d", limit))
	endpoint.RawQuery = query.Encode()

	resp, err := c.doLookupRequest(ctx, endpoint.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steamapis lookup request failed: unexpected status %s", resp.Status)
	}

	var payload []steamAPIsSearchResult
	decoder := json.NewDecoder(io.LimitReader(resp.Body, maxSteamAPIsResponseBytes))
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("steamapis lookup request failed: decode response: %w", err)
	}

	queryTime := lookupQueryTime(resp)
	items := make([]domain.ItemSummary, 0, len(payload))
	for i, result := range payload {
		item, err := c.normalizeResult(result, queryTime)
		if err != nil {
			return nil, fmt.Errorf("result %d: %w", i+1, err)
		}
		items = append(items, item)
	}
	return items, nil
}

func (c *steamAPIsClient) doLookupRequest(ctx context.Context, endpoint string) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("build steamapis lookup request: %w", err)
		}
		httpReq.Header.Set("Accept", "application/json")
		httpReq.Header.Set(c.apiKeyHeader, c.apiKey)
		if c.userAgent != "" {
			httpReq.Header.Set("User-Agent", c.userAgent)
		}

		resp, err := c.defaultClient.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("steamapis lookup request failed: %w", err)
		}
		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		status := resp.Status
		retryAfter := resp.Header.Get("Retry-After")
		resp.Body.Close()
		if attempt >= maxRateLimitRetries {
			return nil, fmt.Errorf("steamapis lookup request failed: rate limited after %d attempts: unexpected status %s", attempt+1, status)
		}
		if err := waitForRetry(ctx, rateLimitDelay(retryAfter, attempt)); err != nil {
			return nil, fmt.Errorf("steamapis lookup request failed: rate limited and retry wait interrupted: %w", err)
		}
	}
}

func (c *steamAPIsClient) normalizeResult(result steamAPIsSearchResult, queryTime string) (domain.ItemSummary, error) {
	name := strings.TrimSpace(result.Name)
	marketplace := strings.TrimSpace(result.Marketplace)
	game := strings.TrimSpace(result.Game)
	if name == "" {
		return domain.ItemSummary{}, fmt.Errorf("%w: missing name", ErrInvalidProviderPayload)
	}
	if marketplace == "" {
		return domain.ItemSummary{}, fmt.Errorf("%w: missing marketplace", ErrInvalidProviderPayload)
	}
	if !strings.EqualFold(game, c.gameCode) {
		return domain.ItemSummary{}, fmt.Errorf("%w: unexpected game %q", ErrInvalidProviderPayload, game)
	}
	if result.PriceUSD != nil && *result.PriceUSD < 0 {
		return domain.ItemSummary{}, fmt.Errorf("%w: negative priceUSD", ErrInvalidProviderPayload)
	}
	if result.OfferCount < 0 {
		return domain.ItemSummary{}, fmt.Errorf("%w: negative offerCount", ErrInvalidProviderPayload)
	}

	return domain.ItemSummary{
		ItemID:        steamAPIsItemID(game, marketplace, name),
		CanonicalName: name,
		AppName:       c.appName,
		QueryTime:     queryTime,
		SellPriceText: steamAPIsPriceText(result.PriceUSD, marketplace),
		SellListings:  result.OfferCount,
	}, nil
}

func steamAPIsItemID(game string, marketplace string, name string) string {
	sum := sha256.Sum256([]byte(game + "\x00" + marketplace + "\x00" + name))
	return "steamapis:" + hex.EncodeToString(sum[:8])
}

func steamAPIsPriceText(priceUSD *float64, marketplace string) string {
	if priceUSD == nil {
		return ""
	}
	return fmt.Sprintf("$%.2f (%s)", *priceUSD, marketplace)
}
