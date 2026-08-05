package domain

import "strconv"

const defaultLookupLimit = 10

var itemSummaryHeader = []string{
	"item_id",
	"canonical_name",
	"app_name",
	"query_time",
	"sale_price_text",
	"sell_price_text",
	"sell_listings",
}

// ItemLookupRequest is the normalized input for one external item lookup command.
type ItemLookupRequest struct {
	Query string
	Limit *int
}

// ItemSummary represents one normalized external item lookup result.
type ItemSummary struct {
	ItemID        string
	CanonicalName string
	AppName       string
	QueryTime     string
	SalePriceText string
	SellPriceText string
	SellListings  int
}

// ItemSummaryHeader returns the canonical field order used by item lookup display.
func ItemSummaryHeader() []string {
	header := make([]string, len(itemSummaryHeader))
	copy(header, itemSummaryHeader)
	return header
}

// DefaultLookupLimit returns the default result limit for lookup flows.
func DefaultLookupLimit() int {
	return defaultLookupLimit
}

// Fields returns the canonical display order used by item lookup renderers.
func (s ItemSummary) Fields() []string {
	return []string{
		s.ItemID,
		s.CanonicalName,
		s.AppName,
		s.QueryTime,
		s.SalePriceText,
		s.SellPriceText,
		strconv.Itoa(s.SellListings),
	}
}
