package lookup

import (
	"context"
	"errors"

	"steam-profit-calculator/internal/domain"
)

var (
	ErrNoMatches              = errors.New("no matching items found")
	ErrInvalidProviderPayload = errors.New("provider response is invalid")
	ErrSteamOfficialRateLimit = errors.New("Steam official market API is rate limited")
)

// Client fetches normalized item summaries from an external provider.
type Client interface {
	Lookup(ctx context.Context, req domain.ItemLookupRequest) ([]domain.ItemSummary, error)
}
