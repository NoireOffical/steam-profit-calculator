package importer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"steam-profit-calculator/internal/importer"
)

func TestParseCSVSuccess(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "import.csv")
	content := strings.Join([]string{
		"item_name,buy_price,sell_price,quantity,multiplier",
		"Capsule,0.48,0.7,5,",
		"Capsule,0.48,0.8,5,0.85",
	}, "\n")
	if err := os.WriteFile(inputPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	records, err := importer.ParseCSV(inputPath)
	if err != nil {
		t.Fatalf("ParseCSV() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("len(records) = %d, want 2", len(records))
	}

	first := strings.Join(records[0].CSVRow(), ",")
	if first != "Capsule,0.480,0.700,5,0.106,0.529,0.819252,default" {
		t.Fatalf("first record = %q", first)
	}

	second := strings.Join(records[1].CSVRow(), ",")
	if second != "Capsule,0.480,0.800,5,0.200,1.000,0.705882,0.850" {
		t.Fatalf("second record = %q", second)
	}
}

func TestParseCSVDefaultsOptionalCells(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "import.csv")
	content := strings.Join([]string{
		"item_name,buy_price,sell_price,quantity,multiplier",
		"Capsule,0.48,0.8,,",
	}, "\n")
	if err := os.WriteFile(inputPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	records, err := importer.ParseCSV(inputPath)
	if err != nil {
		t.Fatalf("ParseCSV() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("len(records) = %d, want 1", len(records))
	}

	got := strings.Join(records[0].CSVRow(), ",")
	want := "Capsule,0.480,0.800,1,0.190,0.190,0.716846,default"
	if got != want {
		t.Fatalf("record = %q, want %q", got, want)
	}
}

func TestParseCSVRejectsMalformedRows(t *testing.T) {
	tests := []struct {
		name    string
		content []string
		wantMsg string
	}{
		{
			name: "invalid sell price",
			content: []string{
				"item_name,buy_price,sell_price,quantity,multiplier",
				"Capsule,0.48,abc,5,",
			},
			wantMsg: "invalid sell price on import row 2",
		},
		{
			name: "invalid semantic quantity",
			content: []string{
				"item_name,buy_price,sell_price,quantity,multiplier",
				"Capsule,0.48,0.8,0,",
			},
			wantMsg: "invalid import row 2: quantity must be greater than 0",
		},
		{
			name: "unsupported header",
			content: []string{
				"item_name,buy_price,sell_price,quantity",
				"Capsule,0.48,0.8,1",
			},
			wantMsg: "unsupported import header",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputPath := filepath.Join(t.TempDir(), "import.csv")
			if err := os.WriteFile(inputPath, []byte(strings.Join(tt.content, "\n")), 0o644); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}

			_, err := importer.ParseCSV(inputPath)
			if err == nil {
				t.Fatal("ParseCSV() error = nil, want non-nil")
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Fatalf("ParseCSV() error = %q, want containing %q", err.Error(), tt.wantMsg)
			}
		})
	}
}
