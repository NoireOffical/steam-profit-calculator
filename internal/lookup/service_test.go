package lookup_test

import (
	"context"
	"errors"
	"testing"

	"steam-profit-calculator/internal/domain"
	"steam-profit-calculator/internal/lookup"
)

type stubClient struct {
	items []domain.ItemSummary
	err   error
	calls *int
}

func (s stubClient) Lookup(context.Context, domain.ItemLookupRequest) ([]domain.ItemSummary, error) {
	if s.calls != nil {
		(*s.calls)++
	}
	return s.items, s.err
}

func TestServiceLookupAppliesLimit(t *testing.T) {
	service := lookup.NewService(stubClient{
		items: []domain.ItemSummary{
			{ItemID: "1", CanonicalName: "A"},
			{ItemID: "2", CanonicalName: "B"},
			{ItemID: "3", CanonicalName: "C"},
		},
	})

	items, err := service.Lookup(context.Background(), domain.ItemLookupRequest{
		Query: "AK",
		Limit: intPtr(2),
	})
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[1].ItemID != "2" {
		t.Fatalf("items[1].ItemID = %q, want 2", items[1].ItemID)
	}
}

func TestServiceLookupReturnsNoMatchError(t *testing.T) {
	service := lookup.NewService(stubClient{})

	_, err := service.Lookup(context.Background(), domain.ItemLookupRequest{Query: "AK"})
	if !errors.Is(err, lookup.ErrNoMatches) {
		t.Fatalf("Lookup() error = %v, want ErrNoMatches", err)
	}
}

func TestServiceLookupRejectsInvalidLimit(t *testing.T) {
	service := lookup.NewService(stubClient{})

	_, err := service.Lookup(context.Background(), domain.ItemLookupRequest{
		Query: "AK",
		Limit: intPtr(0),
	})
	if err == nil {
		t.Fatal("Lookup() error = nil, want non-nil")
	}
	if err.Error() != "lookup limit must be greater than 0" {
		t.Fatalf("Lookup() error = %q, want invalid limit error", err)
	}
}

func TestServiceLookupRejectsEmptyQueryBeforeProviderWork(t *testing.T) {
	calls := 0
	service := lookup.NewService(stubClient{calls: &calls})

	_, err := service.Lookup(context.Background(), domain.ItemLookupRequest{Query: "  "})
	if err == nil || err.Error() != "lookup query is required" {
		t.Fatalf("Lookup() error = %v, want required query", err)
	}
	if calls != 0 {
		t.Fatalf("provider calls = %d, want 0", calls)
	}
}
