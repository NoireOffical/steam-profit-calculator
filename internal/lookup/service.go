package lookup

import (
	"context"
	"fmt"
	"strings"

	"steam-profit-calculator/internal/domain"
)

// Service coordinates external lookup requests and applies command-level behavior.
type Service struct {
	client Client
}

// NewService creates a lookup service from an external provider client.
func NewService(client Client) *Service {
	return &Service{client: client}
}

// Lookup validates the request, queries the provider, and applies the effective limit.
func (s *Service) Lookup(ctx context.Context, req domain.ItemLookupRequest) ([]domain.ItemSummary, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("lookup service is not configured")
	}

	query := strings.TrimSpace(req.Query)
	if query == "" {
		return nil, fmt.Errorf("lookup query is required")
	}

	normalized := domain.ItemLookupRequest{
		Query: query,
		Limit: req.Limit,
	}
	if req.Limit != nil {
		if *req.Limit <= 0 {
			return nil, fmt.Errorf("lookup limit must be greater than 0")
		}
	}

	items, err := s.client.Lookup(ctx, normalized)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrNoMatches
	}
	if normalized.Limit != nil && len(items) > *normalized.Limit {
		items = items[:*normalized.Limit]
	}
	return items, nil
}
