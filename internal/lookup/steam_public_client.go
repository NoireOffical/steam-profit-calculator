package lookup

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"steam-profit-calculator/internal/domain"
)

const maxRateLimitRetries = 2

var rateLimitBackoffs = []time.Duration{
	500 * time.Millisecond,
	1500 * time.Millisecond,
}

type steamCommunityClient struct {
	baseURL       *url.URL
	appID         int
	apiKey        string
	apiKeyHeader  string
	userAgent     string
	defaultClient *http.Client
}

type steamLookupResponse struct {
	Success bool                `json:"success"`
	Results []steamLookupResult `json:"results"`
}

type steamLookupResult struct {
	Name             string                     `json:"name"`
	HashName         string                     `json:"hash_name"`
	SellListings     int                        `json:"sell_listings"`
	SellPriceText    string                     `json:"sell_price_text"`
	SalePriceText    string                     `json:"sale_price_text"`
	AppName          string                     `json:"app_name"`
	AssetDescription steamLookupAssetDescriptor `json:"asset_description"`
}

type steamLookupAssetDescriptor struct {
	ClassID        string `json:"classid"`
	Name           string `json:"name"`
	MarketHashName string `json:"market_hash_name"`
}

// NewSteamCommunityClient is kept as a compatibility wrapper while the
// implementation is renamed to the steam_public provider model.
func NewSteamCommunityClient(cfg Config) (Client, error) {
	return NewSteamPublicClient(cfg)
}

// NewSteamPublicClient creates a Steam Community Market backed lookup client.
func NewSteamPublicClient(cfg Config) (Client, error) {
	baseURL, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse lookup base url %q: %w", cfg.BaseURL, err)
	}
	if cfg.AppID <= 0 {
		return nil, fmt.Errorf("lookup app id must be greater than 0")
	}
	if cfg.Timeout <= 0 {
		return nil, fmt.Errorf("lookup timeout must be greater than 0")
	}

	return &steamCommunityClient{
		baseURL:      baseURL,
		appID:        cfg.AppID,
		apiKey:       cfg.APIKey,
		apiKeyHeader: cfg.APIKeyHeader,
		userAgent:    cfg.UserAgent,
		defaultClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}, nil
}

func (c *steamCommunityClient) Lookup(ctx context.Context, req domain.ItemLookupRequest) ([]domain.ItemSummary, error) {
	endpoint := *c.baseURL
	endpoint.Path = path.Join(endpoint.Path, "/market/search/render/")

	limit := domain.DefaultLookupLimit()
	if req.Limit != nil && *req.Limit > 0 {
		limit = *req.Limit
	}

	query := endpoint.Query()
	query.Set("query", strings.TrimSpace(req.Query))
	query.Set("start", "0")
	query.Set("count", fmt.Sprintf("%d", limit))
	query.Set("search_descriptions", "0")
	query.Set("sort_column", "name")
	query.Set("sort_dir", "asc")
	query.Set("appid", fmt.Sprintf("%d", c.appID))
	query.Set("norender", "1")
	endpoint.RawQuery = query.Encode()

	resp, err := c.doLookupRequest(ctx, endpoint.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lookup provider request failed: unexpected status %s", resp.Status)
	}

	var payload steamLookupResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("lookup provider request failed: decode response: %w", err)
	}
	if !payload.Success {
		return nil, fmt.Errorf("lookup provider request failed: provider returned unsuccessful response")
	}

	queryTime := lookupQueryTime(resp)
	items := make([]domain.ItemSummary, 0, len(payload.Results))
	for i, result := range payload.Results {
		item, err := normalizeSteamLookupResult(result, queryTime)
		if err != nil {
			return nil, fmt.Errorf("result %d: %w", i+1, err)
		}
		items = append(items, item)
	}

	return items, nil
}

func (c *steamCommunityClient) doLookupRequest(ctx context.Context, endpoint string) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("build lookup request: %w", err)
		}
		httpReq.Header.Set("Accept", "application/json")
		if c.userAgent != "" {
			httpReq.Header.Set("User-Agent", c.userAgent)
		}
		if c.apiKey != "" && c.apiKeyHeader != "" {
			httpReq.Header.Set(c.apiKeyHeader, c.apiKey)
		}

		resp, err := c.defaultClient.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("lookup provider request failed: %w", err)
		}
		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		status := resp.Status
		resp.Body.Close()
		if attempt >= maxRateLimitRetries {
			return nil, fmt.Errorf("%w after %d attempts: unexpected status %s", ErrSteamOfficialRateLimit, attempt+1, status)
		}

		wait := rateLimitDelay(resp.Header.Get("Retry-After"), attempt)
		if err := waitForRetry(ctx, wait); err != nil {
			return nil, fmt.Errorf("%w and retry wait was interrupted: %v", ErrSteamOfficialRateLimit, err)
		}
	}
}

func rateLimitDelay(retryAfter string, attempt int) time.Duration {
	if parsed, ok := parseRetryAfter(retryAfter); ok {
		return parsed
	}
	if attempt >= 0 && attempt < len(rateLimitBackoffs) {
		return rateLimitBackoffs[attempt]
	}
	return rateLimitBackoffs[len(rateLimitBackoffs)-1]
}

func parseRetryAfter(value string) (time.Duration, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, false
	}

	if seconds, err := strconv.Atoi(trimmed); err == nil {
		if seconds < 0 {
			seconds = 0
		}
		return time.Duration(seconds) * time.Second, true
	}

	retryAt, err := http.ParseTime(trimmed)
	if err != nil {
		return 0, false
	}
	delay := time.Until(retryAt)
	if delay < 0 {
		return 0, true
	}
	return delay, true
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func lookupQueryTime(resp *http.Response) string {
	if resp != nil {
		if dateHeader := strings.TrimSpace(resp.Header.Get("Date")); dateHeader != "" {
			if observedAt, err := http.ParseTime(dateHeader); err == nil {
				return observedAt.UTC().Format(time.RFC3339)
			}
		}
	}
	return time.Now().UTC().Format(time.RFC3339)
}

func normalizeSteamLookupResult(result steamLookupResult, queryTime string) (domain.ItemSummary, error) {
	itemID := strings.TrimSpace(result.AssetDescription.ClassID)
	if itemID == "" {
		return domain.ItemSummary{}, fmt.Errorf("%w: missing asset_description.classid", ErrInvalidProviderPayload)
	}

	canonicalName := firstNonEmpty(
		result.AssetDescription.MarketHashName,
		result.HashName,
		result.AssetDescription.Name,
		result.Name,
	)
	if canonicalName == "" {
		return domain.ItemSummary{}, fmt.Errorf("%w: missing canonical item name", ErrInvalidProviderPayload)
	}

	appName := strings.TrimSpace(result.AppName)
	if appName == "" {
		return domain.ItemSummary{}, fmt.Errorf("%w: missing app_name", ErrInvalidProviderPayload)
	}

	return domain.ItemSummary{
		ItemID:        itemID,
		CanonicalName: canonicalName,
		AppName:       appName,
		QueryTime:     queryTime,
		SalePriceText: strings.TrimSpace(result.SalePriceText),
		SellPriceText: strings.TrimSpace(result.SellPriceText),
		SellListings:  result.SellListings,
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
