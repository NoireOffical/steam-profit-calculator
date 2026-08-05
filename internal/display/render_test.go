package display_test

import (
	"bytes"
	"strings"
	"testing"

	"steam-profit-calculator/internal/display"
	"steam-profit-calculator/internal/domain"
)

func TestWriteTable(t *testing.T) {
	records := []domain.ProfitRecord{
		{
			ItemName:    "Capsule",
			BuyPrice:    0.48,
			SellPrice:   0.8,
			Quantity:    5,
			NetProfit:   0.19,
			TotalProfit: 0.948,
			KnifeRatio:  0.716846,
		},
	}

	var buf bytes.Buffer
	if err := display.Write(&buf, records, display.ModeTable); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	got := strings.TrimSpace(buf.String())
	want := strings.Join([]string{
		"item_name  buy_price  sell_price  quantity  net_profit  total_profit  knife_ratio",
		"---------  ---------  ----------  --------  ----------  ------------  -----------",
		"Capsule        0.480       0.800         5       0.190         0.948     0.716846",
	}, "\n")
	if got != want {
		t.Fatalf("table output = %q, want %q", got, want)
	}
}

func TestWriteCSV(t *testing.T) {
	records := []domain.ProfitRecord{
		{
			ItemName:    "Capsule",
			BuyPrice:    0.48,
			SellPrice:   0.8,
			Quantity:    5,
			NetProfit:   0.19,
			TotalProfit: 0.948,
			KnifeRatio:  0.716846,
		},
	}

	var buf bytes.Buffer
	if err := display.Write(&buf, records, display.ModeCSV); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	got := strings.TrimSpace(buf.String())
	want := strings.Join([]string{
		"item_name,buy_price,sell_price,quantity,net_profit,total_profit,knife_ratio",
		"Capsule,0.480,0.800,5,0.190,0.948,0.716846",
	}, "\n")
	if got != want {
		t.Fatalf("csv output = %q, want %q", got, want)
	}
}

func TestParseMode(t *testing.T) {
	tests := []struct {
		input string
		want  display.Mode
		ok    bool
	}{
		{input: "", want: display.ModeTable, ok: true},
		{input: "table", want: display.ModeTable, ok: true},
		{input: "csv", want: display.ModeCSV, ok: true},
		{input: "TABLE", want: display.ModeTable, ok: true},
		{input: "unknown", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := display.ParseMode(tt.input)
			if tt.ok {
				if err != nil {
					t.Fatalf("ParseMode() error = %v", err)
				}
				if got != tt.want {
					t.Fatalf("ParseMode() = %q, want %q", got, tt.want)
				}
				return
			}
			if err == nil {
				t.Fatal("ParseMode() error = nil, want non-nil")
			}
		})
	}
}

func TestWriteLookupTable(t *testing.T) {
	items := []domain.ItemSummary{
		{
			ItemID:        "7993054046",
			CanonicalName: "AK-47 | Aphrodite (Battle-Scarred)",
			AppName:       "Counter-Strike 2",
			QueryTime:     "2026-07-14T06:15:30Z",
			SalePriceText: "$18.45",
			SellPriceText: "$19.12",
			SellListings:  17,
		},
	}

	var buf bytes.Buffer
	if err := display.WriteLookup(&buf, items, display.ModeTable); err != nil {
		t.Fatalf("WriteLookup() error = %v", err)
	}

	got := strings.TrimSpace(buf.String())
	want := strings.Join([]string{
		"item_id     canonical_name                      app_name          query_time            sale_price_text  sell_price_text  sell_listings",
		"----------  ----------------------------------  ----------------  --------------------  ---------------  ---------------  -------------",
			"7993054046  AK-47 | Aphrodite (Battle-Scarred)  Counter-Strike 2  2026-07-14T06:15:30Z           $18.45           $19.12             17",
	}, "\n")
	if got != want {
		t.Fatalf("lookup table output = %q, want %q", got, want)
	}
}

func TestWriteLookupCSV(t *testing.T) {
	items := []domain.ItemSummary{
		{
			ItemID:        "7993054046",
			CanonicalName: "AK-47 | Aphrodite (Battle-Scarred)",
			AppName:       "Counter-Strike 2",
			QueryTime:     "2026-07-14T06:15:30Z",
			SalePriceText: "$18.45",
			SellPriceText: "$19.12",
			SellListings:  17,
		},
	}

	var buf bytes.Buffer
	if err := display.WriteLookup(&buf, items, display.ModeCSV); err != nil {
		t.Fatalf("WriteLookup() error = %v", err)
	}

	got := strings.TrimSpace(buf.String())
	want := strings.Join([]string{
		"item_id,canonical_name,app_name,query_time,sale_price_text,sell_price_text,sell_listings",
		"7993054046,AK-47 | Aphrodite (Battle-Scarred),Counter-Strike 2,2026-07-14T06:15:30Z,$18.45,$19.12,17",
	}, "\n")
	if got != want {
		t.Fatalf("lookup csv output = %q, want %q", got, want)
	}
}
