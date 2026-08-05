package cli_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"steam-profit-calculator/internal/cli"
	"steam-profit-calculator/internal/display"
	"steam-profit-calculator/internal/domain"
)

func TestRunAddAndListDefaultTableOutput(t *testing.T) {
	recordFile := filepath.Join(t.TempDir(), "records.csv")
	records := []domain.ProfitRecord{
		{
			ItemName:    "AK-47 Redline",
			BuyPrice:    100,
			SellPrice:   120,
			Quantity:    3,
			NetProfit:   0.44,
			TotalProfit: 1.32,
			KnifeRatio:  0.995619,
		},
		{
			ItemName:    "AK-47 Redline",
			BuyPrice:    100,
			SellPrice:   135,
			Quantity:    3,
			NetProfit:   12.995,
			TotalProfit: 38.985,
			KnifeRatio:  0.884995,
		},
	}

	var addStdout bytes.Buffer
	var addStderr bytes.Buffer
	addExitCode := cli.Run([]string{
		"add",
		"--name", "AK-47 Redline",
		"--buy", "100",
		"--quantity", "3",
		"--sell", "120",
		"--sell", "135",
		"--file", recordFile,
	}, &addStdout, &addStderr)
	if addExitCode != 0 {
		t.Fatalf("add exit code = %d, stderr = %q", addExitCode, addStderr.String())
	}
	if addStderr.Len() != 0 {
		t.Fatalf("add stderr = %q, want empty", addStderr.String())
	}

	wantAdd := renderExpected(t, records, display.ModeTable)
	addOutput := strings.TrimSpace(addStdout.String())
	if addOutput != wantAdd {
		t.Fatalf("add stdout = %q, want %q", addOutput, wantAdd)
	}

	var listStdout bytes.Buffer
	var listStderr bytes.Buffer
	listExitCode := cli.Run([]string{"list", "--file", recordFile}, &listStdout, &listStderr)
	if listExitCode != 0 {
		t.Fatalf("list exit code = %d, stderr = %q", listExitCode, listStderr.String())
	}
	if listStderr.Len() != 0 {
		t.Fatalf("list stderr = %q, want empty", listStderr.String())
	}

	listOutput := strings.TrimSpace(listStdout.String())
	if listOutput != wantAdd {
		t.Fatalf("list stdout = %q, want %q", listOutput, wantAdd)
	}
}

func TestRunAddAndListCSVOutput(t *testing.T) {
	recordFile := filepath.Join(t.TempDir(), "records.csv")
	records := []domain.ProfitRecord{
		{
			ItemName:    "AK-47 Redline",
			BuyPrice:    100,
			SellPrice:   120,
			Quantity:    3,
			NetProfit:   0.44,
			TotalProfit: 1.32,
			KnifeRatio:  0.995619,
		},
		{
			ItemName:    "AK-47 Redline",
			BuyPrice:    100,
			SellPrice:   135,
			Quantity:    3,
			NetProfit:   12.995,
			TotalProfit: 38.985,
			KnifeRatio:  0.884995,
		},
	}

	var addStdout bytes.Buffer
	var addStderr bytes.Buffer
	addExitCode := cli.Run([]string{
		"add",
		"--name", "AK-47 Redline",
		"--buy", "100",
		"--quantity", "3",
		"--sell", "120",
		"--sell", "135",
		"--output", "csv",
		"--file", recordFile,
	}, &addStdout, &addStderr)
	if addExitCode != 0 {
		t.Fatalf("add exit code = %d, stderr = %q", addExitCode, addStderr.String())
	}
	if addStderr.Len() != 0 {
		t.Fatalf("add stderr = %q, want empty", addStderr.String())
	}

	wantAdd := renderExpected(t, records, display.ModeCSV)
	addOutput := strings.TrimSpace(addStdout.String())
	if addOutput != wantAdd {
		t.Fatalf("add stdout = %q, want %q", addOutput, wantAdd)
	}

	var listStdout bytes.Buffer
	var listStderr bytes.Buffer
	listExitCode := cli.Run([]string{"list", "--output", "csv", "--file", recordFile}, &listStdout, &listStderr)
	if listExitCode != 0 {
		t.Fatalf("list exit code = %d, stderr = %q", listExitCode, listStderr.String())
	}
	if listStderr.Len() != 0 {
		t.Fatalf("list stderr = %q, want empty", listStderr.String())
	}

	listOutput := strings.TrimSpace(listStdout.String())
	if listOutput != wantAdd {
		t.Fatalf("list stdout = %q, want %q", listOutput, wantAdd)
	}
}

func TestRunAddValidationFailure(t *testing.T) {
	recordFile := filepath.Join(t.TempDir(), "records.csv")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := cli.Run([]string{
		"add",
		"--name", "AK-47 Redline",
		"--buy", "100",
		"--file", recordFile,
	}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "at least one sell price is required") {
		t.Fatalf("stderr = %q, want validation error", stderr.String())
	}
}

func TestRunAddDefaultsQuantityToOne(t *testing.T) {
	recordFile := filepath.Join(t.TempDir(), "records.csv")
	records := []domain.ProfitRecord{
		{
			ItemName:    "Capsule",
			BuyPrice:    0.48,
			SellPrice:   0.8,
			Quantity:    1,
			NetProfit:   0.19,
			TotalProfit: 0.19,
			KnifeRatio:  0.716846,
		},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := cli.Run([]string{
		"add",
		"--name", "Capsule",
		"--buy", "0.48",
		"--sell", "0.8",
		"--file", recordFile,
	}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %q", exitCode, stderr.String())
	}
	want := renderExpected(t, records, display.ModeTable)
	if strings.TrimSpace(stdout.String()) != want {
		t.Fatalf("stdout = %q, want %q", strings.TrimSpace(stdout.String()), want)
	}
}

func TestRunAddSupportsMultiplier(t *testing.T) {
	recordFile := filepath.Join(t.TempDir(), "records.csv")
	records := []domain.ProfitRecord{
		{
			ItemName:    "Capsule",
			BuyPrice:    0.48,
			SellPrice:   0.8,
			Quantity:    2,
			NetProfit:   0.2,
			TotalProfit: 0.4,
			KnifeRatio:  0.705882,
		},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := cli.Run([]string{
		"add",
		"--name", "Capsule",
		"--buy", "0.48",
		"--quantity", "2",
		"--multiplier", "0.85",
		"--sell", "0.8",
		"--file", recordFile,
	}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %q", exitCode, stderr.String())
	}
	want := renderExpected(t, records, display.ModeTable)
	if strings.TrimSpace(stdout.String()) != want {
		t.Fatalf("stdout = %q, want %q", strings.TrimSpace(stdout.String()), want)
	}
}

func TestRunReverseAndListDefaultTableOutput(t *testing.T) {
	recordFile := filepath.Join(t.TempDir(), "records.csv")
	records := []domain.ProfitRecord{
		{
			ItemName:    "Capsule",
			BuyPrice:    0.48,
			SellPrice:   0.8,
			Quantity:    5,
			NetProfit:   0.2,
			TotalProfit: 1.0,
			KnifeRatio:  0.705882,
		},
	}

	var reverseStdout bytes.Buffer
	var reverseStderr bytes.Buffer
	reverseExitCode := cli.Run([]string{
		"reverse",
		"--name", "Capsule",
		"--solve", "sell",
		"--buy", "0.48",
		"--quantity", "5",
		"--multiplier", "0.85",
		"--target-total-profit", "1.0",
		"--file", recordFile,
	}, &reverseStdout, &reverseStderr)
	if reverseExitCode != 0 {
		t.Fatalf("reverse exit code = %d, stderr = %q", reverseExitCode, reverseStderr.String())
	}
	if reverseStderr.Len() != 0 {
		t.Fatalf("reverse stderr = %q, want empty", reverseStderr.String())
	}

	want := renderExpected(t, records, display.ModeTable)
	if strings.TrimSpace(reverseStdout.String()) != want {
		t.Fatalf("reverse stdout = %q, want %q", strings.TrimSpace(reverseStdout.String()), want)
	}

	var listStdout bytes.Buffer
	var listStderr bytes.Buffer
	listExitCode := cli.Run([]string{"list", "--file", recordFile}, &listStdout, &listStderr)
	if listExitCode != 0 {
		t.Fatalf("list exit code = %d, stderr = %q", listExitCode, listStderr.String())
	}
	if listStderr.Len() != 0 {
		t.Fatalf("list stderr = %q, want empty", listStderr.String())
	}
	if strings.TrimSpace(listStdout.String()) != want {
		t.Fatalf("list stdout = %q, want %q", strings.TrimSpace(listStdout.String()), want)
	}
}

func TestRunReverseSupportsSellPriceFromKnifeRatio(t *testing.T) {
	recordFile := filepath.Join(t.TempDir(), "records.csv")
	records := []domain.ProfitRecord{
		{
			ItemName:    "Capsule",
			BuyPrice:    0.48,
			SellPrice:   0.819252432155658,
			Quantity:    1,
			NetProfit:   0.20571428571428574,
			TotalProfit: 0.20571428571428574,
			KnifeRatio:  0.7,
		},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := cli.Run([]string{
		"reverse",
		"--name", "Capsule",
		"--solve", "sell",
		"--buy", "0.48",
		"--target-knife-ratio", "0.7",
		"--file", recordFile,
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %q", exitCode, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	want := renderExpected(t, records, display.ModeTable)
	if strings.TrimSpace(stdout.String()) != want {
		t.Fatalf("stdout = %q, want %q", strings.TrimSpace(stdout.String()), want)
	}
}

func TestRunReverseValidationFailure(t *testing.T) {
	recordFile := filepath.Join(t.TempDir(), "records.csv")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := cli.Run([]string{
		"reverse",
		"--name", "Capsule",
		"--solve", "buy",
		"--sell", "0.8",
		"--target-net-profit", "0.19",
		"--target-knife-ratio", "0.7",
		"--file", recordFile,
	}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "exactly one reverse target is required") {
		t.Fatalf("stderr = %q, want reverse validation error", stderr.String())
	}
}

func TestRunImportAndListCSVOutput(t *testing.T) {
	tempDir := t.TempDir()
	recordFile := filepath.Join(tempDir, "records.csv")
	inputFile := filepath.Join(tempDir, "import.csv")
	inputContent := strings.Join([]string{
		"item_name,buy_price,sell_price,quantity,multiplier",
		"Capsule,0.48,0.7,5,",
		"Capsule,0.48,0.8,5,0.85",
	}, "\n")
	if err := os.WriteFile(inputFile, []byte(inputContent), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	records := []domain.ProfitRecord{
		{
			ItemName:    "Capsule",
			BuyPrice:    0.48,
			SellPrice:   0.7,
			Quantity:    5,
			NetProfit:   0.106,
			TotalProfit: 0.529,
			KnifeRatio:  0.819252,
		},
		{
			ItemName:    "Capsule",
			BuyPrice:    0.48,
			SellPrice:   0.8,
			Quantity:    5,
			NetProfit:   0.2,
			TotalProfit: 1.0,
			KnifeRatio:  0.705882,
		},
	}

	var importStdout bytes.Buffer
	var importStderr bytes.Buffer
	importExitCode := cli.Run([]string{
		"import",
		"--input", inputFile,
		"--output", "csv",
		"--file", recordFile,
	}, &importStdout, &importStderr)
	if importExitCode != 0 {
		t.Fatalf("import exit code = %d, stderr = %q", importExitCode, importStderr.String())
	}
	if importStderr.Len() != 0 {
		t.Fatalf("import stderr = %q, want empty", importStderr.String())
	}

	want := renderExpected(t, records, display.ModeCSV)
	if strings.TrimSpace(importStdout.String()) != want {
		t.Fatalf("import stdout = %q, want %q", strings.TrimSpace(importStdout.String()), want)
	}

	var listStdout bytes.Buffer
	var listStderr bytes.Buffer
	listExitCode := cli.Run([]string{"list", "--output", "csv", "--file", recordFile}, &listStdout, &listStderr)
	if listExitCode != 0 {
		t.Fatalf("list exit code = %d, stderr = %q", listExitCode, listStderr.String())
	}
	if listStderr.Len() != 0 {
		t.Fatalf("list stderr = %q, want empty", listStderr.String())
	}
	if strings.TrimSpace(listStdout.String()) != want {
		t.Fatalf("list stdout = %q, want %q", strings.TrimSpace(listStdout.String()), want)
	}
}

func TestRunImportDoesNotPartiallyWriteOnFailure(t *testing.T) {
	tempDir := t.TempDir()
	recordFile := filepath.Join(tempDir, "records.csv")
	inputFile := filepath.Join(tempDir, "import.csv")
	inputContent := strings.Join([]string{
		"item_name,buy_price,sell_price,quantity,multiplier",
		"Capsule,0.48,0.8,1,",
		"Capsule,0.48,abc,5,",
	}, "\n")
	if err := os.WriteFile(inputFile, []byte(inputContent), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := cli.Run([]string{
		"import",
		"--input", inputFile,
		"--file", recordFile,
	}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "invalid sell price on import row 3") {
		t.Fatalf("stderr = %q, want import error", stderr.String())
	}

	_, err := os.Stat(recordFile)
	if !os.IsNotExist(err) {
		t.Fatalf("record file err = %v, want not-exist", err)
	}
}

func TestRunListMalformedStorageFailure(t *testing.T) {
	recordFile := filepath.Join(t.TempDir(), "records.csv")

	var addStdout bytes.Buffer
	var addStderr bytes.Buffer
	if exitCode := cli.Run([]string{
		"add",
		"--name", "AK-47 Redline",
		"--buy", "100",
		"--quantity", "3",
		"--sell", "120",
		"--file", recordFile,
	}, &addStdout, &addStderr); exitCode != 0 {
		t.Fatalf("seed add exit code = %d, stderr = %q", exitCode, addStderr.String())
	}

	if err := os.WriteFile(recordFile, []byte("item_name,buy_price,sell_price,quantity,net_profit,total_profit,knife_ratio\nAK-47 Redline,nope,120.000,3,0.440,1.320,0.995619\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := cli.Run([]string{"list", "--file", recordFile}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "invalid buy price on row 2") {
		t.Fatalf("stderr = %q, want malformed-row error", stderr.String())
	}
}

func TestRunListLegacyStorageFile(t *testing.T) {
	recordFile := filepath.Join(t.TempDir(), "records.csv")
	legacyContent := "item_name,buy_price,sell_price,net_profit,knife_ratio\nCapsule,0.480,0.800,0.190,0.716846\n"
	if err := os.WriteFile(recordFile, []byte(legacyContent), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := cli.Run([]string{"list", "--file", recordFile}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %q", exitCode, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	want := renderExpected(t, []domain.ProfitRecord{
		{
			ItemName:    "Capsule",
			BuyPrice:    0.48,
			SellPrice:   0.8,
			Quantity:    1,
			NetProfit:   0.19,
			TotalProfit: 0.19,
			KnifeRatio:  0.716846,
		},
	}, display.ModeTable)
	if strings.TrimSpace(stdout.String()) != want {
		t.Fatalf("stdout = %q, want %q", strings.TrimSpace(stdout.String()), want)
	}
}

func TestRunLookupDefaultTableOutput(t *testing.T) {
	server := newLookupTestServer(t, http.StatusOK, `{
                "success": true,
                "results": [
                        {
                                "name": "AK-47 | Aphrodite (Battle-Scarred)",
                                "hash_name": "AK-47 | Aphrodite (Battle-Scarred)",
                                "sell_listings": 17,
                                "sell_price_text": "$19.12",
                                "sale_price_text": "$18.45",
                                "app_name": "Counter-Strike 2",
                                "asset_description": {
                                        "classid": "7993054046",
                                        "market_hash_name": "AK-47 | Aphrodite (Battle-Scarred)"
                                }
                        },
                        {
                                "name": "AK-47 | Asiimov (Battle-Scarred)",
                                "hash_name": "AK-47 | Asiimov (Battle-Scarred)",
                                "sell_listings": 42,
                                "sell_price_text": "$11.06",
                                "sale_price_text": "$10.52",
                                "app_name": "Counter-Strike 2",
                                "asset_description": {
                                        "classid": "7151910841",
                                        "market_hash_name": "AK-47 | Asiimov (Battle-Scarred)"
                                }
                        }
                ]
        }`)
	defer server.Close()

	tempDir := t.TempDir()
	withWorkingDirectory(t, tempDir)
	configureLookupEnvironment(t, server)

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
		{
			ItemID:        "7151910841",
			CanonicalName: "AK-47 | Asiimov (Battle-Scarred)",
			AppName:       "Counter-Strike 2",
			QueryTime:     "2026-07-14T06:15:30Z",
			SalePriceText: "$10.52",
			SellPriceText: "$11.06",
			SellListings:  42,
		},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := cli.Run([]string{"lookup", "--query", "AK-47"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %q", exitCode, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	want := renderExpectedLookup(t, items, display.ModeTable)
	if strings.TrimSpace(stdout.String()) != want {
		t.Fatalf("stdout = %q, want %q", strings.TrimSpace(stdout.String()), want)
	}

	if _, err := os.Stat(filepath.Join(tempDir, "steam-profit-records.csv")); !os.IsNotExist(err) {
		t.Fatalf("record file err = %v, want not-exist", err)
	}
}

func TestRunLookupCSVOutputWithLimit(t *testing.T) {
	server := newLookupTestServer(t, http.StatusOK, `{
                "success": true,
                "results": [
                        {
                                "name": "AK-47 | Aphrodite (Battle-Scarred)",
                                "hash_name": "AK-47 | Aphrodite (Battle-Scarred)",
                                "sell_listings": 17,
                                "sell_price_text": "$19.12",
                                "sale_price_text": "$18.45",
                                "app_name": "Counter-Strike 2",
                                "asset_description": {
                                        "classid": "7993054046",
                                        "market_hash_name": "AK-47 | Aphrodite (Battle-Scarred)"
                                }
                        },
                        {
                                "name": "AK-47 | Asiimov (Battle-Scarred)",
                                "hash_name": "AK-47 | Asiimov (Battle-Scarred)",
                                "sell_listings": 42,
                                "sell_price_text": "$11.06",
                                "sale_price_text": "$10.52",
                                "app_name": "Counter-Strike 2",
                                "asset_description": {
                                        "classid": "7151910841",
                                        "market_hash_name": "AK-47 | Asiimov (Battle-Scarred)"
                                }
                        }
                ]
        }`)
	defer server.Close()

	configureLookupEnvironment(t, server)

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

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := cli.Run([]string{
		"lookup",
		"--query", "AK-47",
		"--limit", "1",
		"--output", "csv",
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %q", exitCode, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	want := renderExpectedLookup(t, items, display.ModeCSV)
	if strings.TrimSpace(stdout.String()) != want {
		t.Fatalf("stdout = %q, want %q", strings.TrimSpace(stdout.String()), want)
	}
}

func TestRunLookupRejectsMissingQuery(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := cli.Run([]string{"lookup"}, &stdout, &stderr)

	if exitCode != 2 {
		t.Fatalf("exit code = %d, want 2", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "--query is required") {
		t.Fatalf("stderr = %q, want missing query error", stderr.String())
	}
}

func TestRunLookupNoMatchFailure(t *testing.T) {
	server := newLookupTestServer(t, http.StatusOK, `{"success": true, "results": []}`)
	defer server.Close()

	tempDir := t.TempDir()
	withWorkingDirectory(t, tempDir)
	configureLookupEnvironment(t, server)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := cli.Run([]string{"lookup", "--query", "missing"}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "no matching items found") {
		t.Fatalf("stderr = %q, want no-match error", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(tempDir, "steam-profit-records.csv")); !os.IsNotExist(err) {
		t.Fatalf("record file err = %v, want not-exist", err)
	}
}

func TestRunLookupProviderFailure(t *testing.T) {
	server := newLookupTestServer(t, http.StatusBadGateway, `upstream unavailable`)
	defer server.Close()

	configureLookupEnvironment(t, server)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := cli.Run([]string{"lookup", "--query", "AK-47"}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "lookup provider request failed") {
		t.Fatalf("stderr = %q, want provider failure", stderr.String())
	}
}

func TestRunLookupInvalidProviderResponse(t *testing.T) {
	server := newLookupTestServer(t, http.StatusOK, `{
                "success": true,
                "results": [
                        {
                                "name": "AK-47 | Aphrodite (Battle-Scarred)",
                                "hash_name": "AK-47 | Aphrodite (Battle-Scarred)",
                                "app_name": "Counter-Strike 2",
                                "asset_description": {
                                        "appid": 730,
                                        "market_hash_name": "AK-47 | Aphrodite (Battle-Scarred)",
                                        "type": "Covert Rifle"
                                }
                        }
                ]
        }`)
	defer server.Close()

	configureLookupEnvironment(t, server)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := cli.Run([]string{"lookup", "--query", "AK-47"}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "provider response is invalid") {
		t.Fatalf("stderr = %q, want invalid response error", stderr.String())
	}
}

func TestRunLookupUnsupportedProviderConfiguration(t *testing.T) {
	t.Setenv("STEAM_PROFIT_LOOKUP_PROVIDER", "proxy")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := cli.Run([]string{"lookup", "--query", "AK-47"}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), `unsupported lookup provider "proxy"`) {
		t.Fatalf("stderr = %q, want unsupported provider error", stderr.String())
	}
}

func renderExpected(t *testing.T, records []domain.ProfitRecord, mode display.Mode) string {
	t.Helper()

	var buf bytes.Buffer
	if err := display.Write(&buf, records, mode); err != nil {
		t.Fatalf("display.Write() error = %v", err)
	}
	return strings.TrimSpace(buf.String())
}

func renderExpectedLookup(t *testing.T, items []domain.ItemSummary, mode display.Mode) string {
	t.Helper()

	var buf bytes.Buffer
	if err := display.WriteLookup(&buf, items, mode); err != nil {
		t.Fatalf("display.WriteLookup() error = %v", err)
	}
	return strings.TrimSpace(buf.String())
}

func configureLookupEnvironment(t *testing.T, server *httptest.Server) {
	t.Helper()
	t.Setenv("STEAM_PROFIT_LOOKUP_BASE_URL", server.URL)
	t.Setenv("STEAM_PROFIT_LOOKUP_APP_ID", "730")
	t.Setenv("STEAM_PROFIT_LOOKUP_TIMEOUT_MS", "1000")
}

func newLookupTestServer(t *testing.T, statusCode int, body string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Date", "Tue, 14 Jul 2026 06:15:30 GMT")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(body))
	}))
}

func withWorkingDirectory(t *testing.T, dir string) {
	t.Helper()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore Chdir() error = %v", err)
		}
	})
}
