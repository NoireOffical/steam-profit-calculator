package evaluation

import (
	"context"
	"errors"

	"steam-profit-calculator/internal/domain"
	"steam-profit-calculator/internal/lookup"
)

// FixtureLookup keeps live-model evaluation focused on semantic tool selection
// without depending on Steam or inventing current market values.
type FixtureLookup struct{}

func (FixtureLookup) Lookup(_ context.Context, request domain.ItemLookupRequest) ([]domain.ItemSummary, error) {
	switch request.Query {
	case "missing":
		return nil, lookup.ErrNoMatches
	case "provider-down":
		return nil, errors.New("evaluation provider unavailable")
	}
	limit := domain.DefaultLookupLimit()
	if request.Limit != nil {
		limit = *request.Limit
	}
	return []domain.ItemSummary{{
		ItemID: "evaluation-fixture", CanonicalName: request.Query + " | Evaluation Fixture", AppName: "Counter-Strike 2",
		QueryTime: "2026-08-07T00:00:00Z", SalePriceText: "$10.00", SellPriceText: "$9.90", SellListings: limit * 2,
	}}, nil
}
