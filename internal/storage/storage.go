package storage

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"steam-profit-calculator/internal/domain"
)

type schemaKind int

const (
	schemaMissing schemaKind = iota
	schemaCurrent
	schemaExpandedWithoutFee
	schemaLegacy
)

// CSVStore persists profit records in a single CSV file.
type CSVStore struct {
	path string
}

// NewCSVStore creates a CSV-backed store for the given file path.
func NewCSVStore(path string) *CSVStore {
	return &CSVStore{path: path}
}

// Append writes one row per record, creating the file and header on first write.
func (s *CSVStore) Append(records []domain.ProfitRecord) error {
	if len(records) == 0 {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create record directory: %w", err)
	}

	schema, existing, err := s.readAll()
	if err != nil {
		return err
	}

	switch schema {
	case schemaMissing:
		return s.writeAll(records)
	case schemaLegacy:
		return s.writeAll(append(existing, records...))
	case schemaCurrent:
		file, err := os.OpenFile(s.path, os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return fmt.Errorf("open record file: %w", err)
		}
		defer file.Close()

		for _, record := range records {
			if err := writeCSVLine(file, record.CSVRow()); err != nil {
				return fmt.Errorf("write record row: %w", err)
			}
		}
		return nil
	case schemaExpandedWithoutFee:
		return s.writeAll(append(existing, records...))
	default:
		return fmt.Errorf("unsupported record schema")
	}
}

// List loads all persisted records from the CSV file.
func (s *CSVStore) List() ([]domain.ProfitRecord, error) {
	_, records, err := s.readAll()
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (s *CSVStore) readAll() (schemaKind, []domain.ProfitRecord, error) {
	file, err := os.Open(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return schemaMissing, nil, nil
		}
		return schemaMissing, nil, fmt.Errorf("open record file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	header, err := reader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return schemaMissing, nil, nil
		}
		return schemaMissing, nil, fmt.Errorf("read record header: %w", err)
	}

	schema, err := detectSchema(header)
	if err != nil {
		return schemaMissing, nil, err
	}

	var records []domain.ProfitRecord
	line := 1
	for {
		line++
		row, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return schema, records, nil
			}
			return schemaMissing, nil, fmt.Errorf("read record row %d: %w", line, err)
		}

		record, err := parseRecord(row, line, schema)
		if err != nil {
			return schemaMissing, nil, err
		}
		records = append(records, record)
	}
}

func (s *CSVStore) writeAll(records []domain.ProfitRecord) error {
	file, err := os.OpenFile(s.path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("open record file: %w", err)
	}
	defer file.Close()

	if err := writeCSVLine(file, domain.StorageRecordHeader()); err != nil {
		return fmt.Errorf("write record header: %w", err)
	}
	for _, record := range records {
		if err := writeCSVLine(file, record.CSVRow()); err != nil {
			return fmt.Errorf("write record row: %w", err)
		}
	}

	return nil
}

func writeCSVLine(w io.Writer, row []string) error {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	if err := writer.Write(row); err != nil {
		return err
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}

	line := strings.TrimSuffix(buf.String(), "\n")
	line = strings.TrimSuffix(line, "\r")
	line = addSpacesAfterDelimiters(line)
	_, err := io.WriteString(w, line+"\n")
	return err
}

func addSpacesAfterDelimiters(line string) string {
	var builder strings.Builder
	builder.Grow(len(line) + 8)

	inQuotes := false
	for i := 0; i < len(line); i++ {
		ch := line[i]
		switch ch {
		case '"':
			builder.WriteByte(ch)
			if inQuotes && i+1 < len(line) && line[i+1] == '"' {
				i++
				builder.WriteByte(line[i])
				continue
			}
			inQuotes = !inQuotes
		case ',':
			builder.WriteByte(ch)
			if !inQuotes && (i+1 >= len(line) || line[i+1] != ' ') {
				builder.WriteByte(' ')
			}
		default:
			builder.WriteByte(ch)
		}
	}

	return builder.String()
}

func detectSchema(header []string) (schemaKind, error) {
	switch {
	case sameHeader(header, domain.StorageRecordHeader()):
		return schemaCurrent, nil
	case sameHeader(header, domain.RecordHeader()):
		return schemaExpandedWithoutFee, nil
	case sameHeader(header, domain.LegacyRecordHeader()):
		return schemaLegacy, nil
	default:
		return schemaMissing, fmt.Errorf("invalid record header: %q", header)
	}
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

func parseRecord(row []string, line int, schema schemaKind) (domain.ProfitRecord, error) {
	switch schema {
	case schemaCurrent:
		return parseCurrentRecord(row, line)
	case schemaExpandedWithoutFee:
		return parseExpandedRecordWithoutFee(row, line)
	case schemaLegacy:
		return parseLegacyRecord(row, line)
	default:
		return domain.ProfitRecord{}, fmt.Errorf("unsupported record schema on row %d", line)
	}
}

func parseCurrentRecord(row []string, line int) (domain.ProfitRecord, error) {
	if len(row) != len(domain.StorageRecordHeader()) {
		return domain.ProfitRecord{}, fmt.Errorf("invalid record row %d: got %d columns, want %d", line, len(row), len(domain.StorageRecordHeader()))
	}

	buyPrice, err := strconv.ParseFloat(row[1], 64)
	if err != nil {
		return domain.ProfitRecord{}, fmt.Errorf("invalid buy price on row %d: %w", line, err)
	}
	sellPrice, err := strconv.ParseFloat(row[2], 64)
	if err != nil {
		return domain.ProfitRecord{}, fmt.Errorf("invalid sell price on row %d: %w", line, err)
	}
	quantity, err := strconv.Atoi(row[3])
	if err != nil {
		return domain.ProfitRecord{}, fmt.Errorf("invalid quantity on row %d: %w", line, err)
	}
	if quantity <= 0 {
		return domain.ProfitRecord{}, fmt.Errorf("invalid quantity on row %d: must be greater than 0", line)
	}

	netProfit, err := strconv.ParseFloat(row[4], 64)
	if err != nil {
		return domain.ProfitRecord{}, fmt.Errorf("invalid net profit on row %d: %w", line, err)
	}
	totalProfit, err := strconv.ParseFloat(row[5], 64)
	if err != nil {
		return domain.ProfitRecord{}, fmt.Errorf("invalid total profit on row %d: %w", line, err)
	}
	knifeRatio, err := strconv.ParseFloat(row[6], 64)
	if err != nil {
		return domain.ProfitRecord{}, fmt.Errorf("invalid knife ratio on row %d: %w", line, err)
	}
	if row[7] == "" {
		return domain.ProfitRecord{}, fmt.Errorf("invalid fee on row %d: must not be empty", line)
	}

	if row[0] == "" {
		return domain.ProfitRecord{}, fmt.Errorf("invalid item name on row %d: must not be empty", line)
	}

	return domain.ProfitRecord{
		ItemName:    row[0],
		BuyPrice:    buyPrice,
		SellPrice:   sellPrice,
		Quantity:    quantity,
		NetProfit:   netProfit,
		TotalProfit: totalProfit,
		KnifeRatio:  knifeRatio,
		Fee:         row[7],
	}, nil
}

func parseExpandedRecordWithoutFee(row []string, line int) (domain.ProfitRecord, error) {
	if len(row) != len(domain.RecordHeader()) {
		return domain.ProfitRecord{}, fmt.Errorf("invalid record row %d: got %d columns, want %d", line, len(row), len(domain.RecordHeader()))
	}

	buyPrice, err := strconv.ParseFloat(row[1], 64)
	if err != nil {
		return domain.ProfitRecord{}, fmt.Errorf("invalid buy price on row %d: %w", line, err)
	}
	sellPrice, err := strconv.ParseFloat(row[2], 64)
	if err != nil {
		return domain.ProfitRecord{}, fmt.Errorf("invalid sell price on row %d: %w", line, err)
	}
	quantity, err := strconv.Atoi(row[3])
	if err != nil {
		return domain.ProfitRecord{}, fmt.Errorf("invalid quantity on row %d: %w", line, err)
	}
	if quantity <= 0 {
		return domain.ProfitRecord{}, fmt.Errorf("invalid quantity on row %d: must be greater than 0", line)
	}
	netProfit, err := strconv.ParseFloat(row[4], 64)
	if err != nil {
		return domain.ProfitRecord{}, fmt.Errorf("invalid net profit on row %d: %w", line, err)
	}
	totalProfit, err := strconv.ParseFloat(row[5], 64)
	if err != nil {
		return domain.ProfitRecord{}, fmt.Errorf("invalid total profit on row %d: %w", line, err)
	}
	knifeRatio, err := strconv.ParseFloat(row[6], 64)
	if err != nil {
		return domain.ProfitRecord{}, fmt.Errorf("invalid knife ratio on row %d: %w", line, err)
	}
	if row[0] == "" {
		return domain.ProfitRecord{}, fmt.Errorf("invalid item name on row %d: must not be empty", line)
	}

	feeLabel := domain.DefaultFeeLabel()
	if sellPrice > 0 && knifeRatio > 0 {
		multiplier := buyPrice / (sellPrice * knifeRatio)
		feeLabel = formatFeeLabel(multiplier)
	} else if sellPrice > 0 {
		multiplier := (netProfit + buyPrice) / sellPrice
		feeLabel = formatFeeLabel(multiplier)
	}

	return domain.ProfitRecord{
		ItemName:    row[0],
		BuyPrice:    buyPrice,
		SellPrice:   sellPrice,
		Quantity:    quantity,
		NetProfit:   netProfit,
		TotalProfit: totalProfit,
		KnifeRatio:  knifeRatio,
		Fee:         feeLabel,
	}, nil
}

func parseLegacyRecord(row []string, line int) (domain.ProfitRecord, error) {
	if len(row) != len(domain.LegacyRecordHeader()) {
		return domain.ProfitRecord{}, fmt.Errorf("invalid record row %d: got %d columns, want %d", line, len(row), len(domain.LegacyRecordHeader()))
	}

	buyPrice, err := strconv.ParseFloat(row[1], 64)
	if err != nil {
		return domain.ProfitRecord{}, fmt.Errorf("invalid buy price on row %d: %w", line, err)
	}
	sellPrice, err := strconv.ParseFloat(row[2], 64)
	if err != nil {
		return domain.ProfitRecord{}, fmt.Errorf("invalid sell price on row %d: %w", line, err)
	}
	netProfit, err := strconv.ParseFloat(row[3], 64)
	if err != nil {
		return domain.ProfitRecord{}, fmt.Errorf("invalid net profit on row %d: %w", line, err)
	}
	knifeRatio, err := strconv.ParseFloat(row[4], 64)
	if err != nil {
		return domain.ProfitRecord{}, fmt.Errorf("invalid knife ratio on row %d: %w", line, err)
	}
	if row[0] == "" {
		return domain.ProfitRecord{}, fmt.Errorf("invalid item name on row %d: must not be empty", line)
	}

	return domain.ProfitRecord{
		ItemName:    row[0],
		BuyPrice:    buyPrice,
		SellPrice:   sellPrice,
		Quantity:    domain.DefaultQuantity(),
		NetProfit:   netProfit,
		TotalProfit: netProfit,
		KnifeRatio:  knifeRatio,
		Fee:         domain.DefaultFeeLabel(),
	}, nil
}

func formatFeeLabel(multiplier float64) string {
	if math.Abs(multiplier-domain.DefaultMultiplier()) <= 0.0005 {
		return domain.DefaultFeeLabel()
	}

	return domain.FormatMultiplier(multiplier)
}
