package storage_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"steam-profit-calculator/internal/domain"
	"steam-profit-calculator/internal/storage"
)

func TestCSVStoreAppendCreatesFileWithHeader(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "records.csv")
	store := storage.NewCSVStore(storePath)

	err := store.Append([]domain.ProfitRecord{
		{
			ItemName:    "AK-47 Redline",
			BuyPrice:    100,
			SellPrice:   120,
			Quantity:    3,
			NetProfit:   0.44,
			TotalProfit: 1.32,
			KnifeRatio:  0.995619,
                        Fee:         "default",
		},
	})
	if err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	content, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	got := strings.TrimSpace(string(content))
	want := strings.Join([]string{
                "item_name, buy_price, sell_price, quantity, net_profit, total_profit, knife_ratio, fee",
                "AK-47 Redline, 100.000, 120.000, 3, 0.440, 1.320, 0.995619, default",
	}, "\n")
	if got != want {
		t.Fatalf("file content = %q, want %q", got, want)
	}
}

func TestCSVStoreAppendAndList(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "records.csv")
	store := storage.NewCSVStore(storePath)

	firstBatch := []domain.ProfitRecord{
		{
			ItemName:    "AK-47 Redline",
			BuyPrice:    100,
			SellPrice:   120,
			Quantity:    3,
			NetProfit:   0.44,
			TotalProfit: 1.32,
			KnifeRatio:  0.995619,
                        Fee:         "default",
		},
	}
	secondBatch := []domain.ProfitRecord{
		{
			ItemName:    "M9 Bayonet",
			BuyPrice:    200,
			SellPrice:   260,
			Quantity:    2,
			NetProfit:   17.62,
			TotalProfit: 35.24,
			KnifeRatio:  0.920471,
                        Fee:         "default",
		},
	}

	if err := store.Append(firstBatch); err != nil {
		t.Fatalf("Append(firstBatch) error = %v", err)
	}
	if err := store.Append(secondBatch); err != nil {
		t.Fatalf("Append(secondBatch) error = %v", err)
	}

	records, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("len(records) = %d, want 2", len(records))
	}

	first := strings.Join(records[0].CSVRow(), ",")
        if first != "AK-47 Redline,100.000,120.000,3,0.440,1.320,0.995619,default" {
		t.Fatalf("first record = %q", first)
	}

	second := strings.Join(records[1].CSVRow(), ",")
        if second != "M9 Bayonet,200.000,260.000,2,17.620,35.240,0.920471,default" {
		t.Fatalf("second record = %q", second)
	}
}

func TestCSVStoreListLegacyData(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "records.csv")
	content := strings.Join([]string{
		"item_name,buy_price,sell_price,net_profit,knife_ratio",
		"Capsule,0.480,0.800,0.190,0.716846",
	}, "\n")
	if err := os.WriteFile(storePath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store := storage.NewCSVStore(storePath)
	records, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("len(records) = %d, want 1", len(records))
	}

	got := strings.Join(records[0].CSVRow(), ",")
        want := "Capsule,0.480,0.800,1,0.190,0.190,0.716846,default"
	if got != want {
		t.Fatalf("legacy record = %q, want %q", got, want)
	}
}

func TestCSVStoreAppendMigratesLegacyData(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "records.csv")
	content := strings.Join([]string{
		"item_name,buy_price,sell_price,net_profit,knife_ratio",
		"Capsule,0.480,0.800,0.190,0.716846",
	}, "\n")
	if err := os.WriteFile(storePath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store := storage.NewCSVStore(storePath)
	err := store.Append([]domain.ProfitRecord{
		{
			ItemName:    "Capsule",
			BuyPrice:    0.48,
			SellPrice:   0.9,
			Quantity:    4,
			NetProfit:   0.273,
			TotalProfit: 1.092,
			KnifeRatio:  0.637197,
                        Fee:         "default",
		},
	})
	if err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	contentBytes, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	got := strings.TrimSpace(string(contentBytes))
	want := strings.Join([]string{
                "item_name, buy_price, sell_price, quantity, net_profit, total_profit, knife_ratio, fee",
                "Capsule, 0.480, 0.800, 1, 0.190, 0.190, 0.716846, default",
                "Capsule, 0.480, 0.900, 4, 0.273, 1.092, 0.637197, default",
	}, "\n")
	if got != want {
		t.Fatalf("migrated file = %q, want %q", got, want)
	}
}

func TestCSVStoreListMalformedData(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "records.csv")
	content := strings.Join([]string{
                "item_name,buy_price,sell_price,quantity,net_profit,total_profit,knife_ratio,fee",
                "AK-47 Redline,not-a-number,120.000,3,0.440,1.320,0.995619,default",
	}, "\n")
	if err := os.WriteFile(storePath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store := storage.NewCSVStore(storePath)
	_, err := store.List()
	if err == nil {
		t.Fatal("List() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "invalid buy price on row 2") {
		t.Fatalf("List() error = %q, want message containing %q", err.Error(), "invalid buy price on row 2")
	}
}

func TestCSVStoreListExpandedDataWithoutFeeDefaultsLabel(t *testing.T) {
        storePath := filepath.Join(t.TempDir(), "records.csv")
        content := strings.Join([]string{
                "item_name,buy_price,sell_price,quantity,net_profit,total_profit,knife_ratio",
                "Capsule,0.480,0.800,2,0.190,0.380,0.716846",
        }, "\n")
        if err := os.WriteFile(storePath, []byte(content), 0o644); err != nil {
                t.Fatalf("WriteFile() error = %v", err)
        }

        store := storage.NewCSVStore(storePath)
        records, err := store.List()
        if err != nil {
                t.Fatalf("List() error = %v", err)
        }
        if len(records) != 1 {
                t.Fatalf("len(records) = %d, want 1", len(records))
        }

        got := strings.Join(records[0].CSVRow(), ",")
        want := "Capsule,0.480,0.800,2,0.190,0.380,0.716846,default"
        if got != want {
                t.Fatalf("expanded record = %q, want %q", got, want)
        }
}

func TestCSVStoreListExpandedDataWithoutFeeDerivesExplicitMultiplier(t *testing.T) {
        storePath := filepath.Join(t.TempDir(), "records.csv")
        content := strings.Join([]string{
                "item_name,buy_price,sell_price,quantity,net_profit,total_profit,knife_ratio",
                "Capsule,0.480,0.800,2,0.200,0.400,0.705882",
        }, "\n")
        if err := os.WriteFile(storePath, []byte(content), 0o644); err != nil {
                t.Fatalf("WriteFile() error = %v", err)
        }

        store := storage.NewCSVStore(storePath)
        records, err := store.List()
        if err != nil {
                t.Fatalf("List() error = %v", err)
        }
        if len(records) != 1 {
                t.Fatalf("len(records) = %d, want 1", len(records))
        }

        got := strings.Join(records[0].CSVRow(), ",")
        want := "Capsule,0.480,0.800,2,0.200,0.400,0.705882,0.850"
        if got != want {
                t.Fatalf("expanded record = %q, want %q", got, want)
        }
}
