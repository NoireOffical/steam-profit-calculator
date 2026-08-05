package importer

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"steam-profit-calculator/internal/calculation"
	"steam-profit-calculator/internal/domain"
)

var expectedHeader = []string{
	"item_name",
	"buy_price",
	"sell_price",
	"quantity",
	"multiplier",
}

// ParseCSV reads one forward-calculation scenario per CSV row and returns calculated records.
func ParseCSV(path string) ([]domain.ProfitRecord, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open import file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("read import header: empty file")
		}
		return nil, fmt.Errorf("read import header: %w", err)
	}
	if !sameHeader(header, expectedHeader) {
		return nil, fmt.Errorf("unsupported import header: %q", header)
	}

	records := make([]domain.ProfitRecord, 0)
	line := 1
	for {
		line++
		row, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return records, nil
			}
			return nil, fmt.Errorf("read import row %d: %w", line, err)
		}

		record, err := parseRow(row, line)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
}

func parseRow(row []string, line int) (domain.ProfitRecord, error) {
	if len(row) != len(expectedHeader) {
		return domain.ProfitRecord{}, fmt.Errorf("invalid import row %d: got %d columns, want %d", line, len(row), len(expectedHeader))
	}

	buyPrice, err := strconv.ParseFloat(row[1], 64)
	if err != nil {
		return domain.ProfitRecord{}, fmt.Errorf("invalid buy price on import row %d: %w", line, err)
	}
	sellPrice, err := strconv.ParseFloat(row[2], 64)
	if err != nil {
		return domain.ProfitRecord{}, fmt.Errorf("invalid sell price on import row %d: %w", line, err)
	}

	var quantity *int
	if row[3] != "" {
		parsedQuantity, parseErr := strconv.Atoi(row[3])
		if parseErr != nil {
			return domain.ProfitRecord{}, fmt.Errorf("invalid quantity on import row %d: %w", line, parseErr)
		}
		quantity = &parsedQuantity
	}

	var multiplier *float64
	if row[4] != "" {
		parsedMultiplier, parseErr := strconv.ParseFloat(row[4], 64)
		if parseErr != nil {
			return domain.ProfitRecord{}, fmt.Errorf("invalid multiplier on import row %d: %w", line, parseErr)
		}
		multiplier = &parsedMultiplier
	}

	records, err := calculation.Calculate(domain.CalculationRequest{
		ItemName:   row[0],
		BuyPrice:   buyPrice,
		Quantity:   quantity,
		Multiplier: multiplier,
		SellPrices: []float64{sellPrice},
	})
	if err != nil {
		return domain.ProfitRecord{}, fmt.Errorf("invalid import row %d: %w", line, err)
	}

	return records[0], nil
}

func sameHeader(got, expected []string) bool {
	if len(got) != len(expected) {
		return false
	}
	for i := range expected {
		if got[i] != expected[i] {
			return false
		}
	}
	return true
}
