package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// CSVImportResult reports the effect of one idempotent import.
type CSVImportResult struct {
	Inserted int
	Skipped  int
}

// ImportCSV imports a compatible CSV into SQLite without modifying the source file.
func (r *SQLiteRepository) ImportCSV(ctx context.Context, path string) (CSVImportResult, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return CSVImportResult{}, fmt.Errorf("resolve CSV import path: %w", err)
	}
	info, err := os.Stat(absolutePath)
	if err != nil {
		return CSVImportResult{}, fmt.Errorf("inspect CSV import file: %w", err)
	}
	if info.IsDir() {
		return CSVImportResult{}, fmt.Errorf("CSV import path must be a file")
	}

	records, err := NewCSVStore(absolutePath).List()
	if err != nil {
		return CSVImportResult{}, fmt.Errorf("read CSV import file: %w", err)
	}
	encoded := make([]encodedRecord, len(records))
	keys := make([]string, len(records))
	for i, record := range records {
		value, err := encodeRecord(record)
		if err != nil {
			return CSVImportResult{}, fmt.Errorf("encode CSV row %d: %w", i+2, err)
		}
		encoded[i] = value
		keys[i], err = csvSourceKey(absolutePath, i+2, record.CSVRow())
		if err != nil {
			return CSVImportResult{}, fmt.Errorf("identify CSV row %d: %w", i+2, err)
		}
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return CSVImportResult{}, fmt.Errorf("begin CSV import: %w", err)
	}
	defer tx.Rollback()

	baseTimestamp := info.ModTime().UTC().Truncate(time.Microsecond)
	result := CSVImportResult{}
	for i, value := range encoded {
		timestamp := baseTimestamp.Add(time.Duration(i) * time.Microsecond)
		execution, err := tx.ExecContext(ctx, insertRecordSQL+` ON CONFLICT(source_key) DO NOTHING`,
			value.itemName,
			value.buyPriceMilli,
			value.sellPriceMilli,
			value.quantity,
			value.netProfitMilli,
			value.totalProfitMilli,
			value.knifeRatioMicros,
			value.fee,
			timestamp.UnixMicro(),
			"csv_import",
			keys[i],
		)
		if err != nil {
			return CSVImportResult{}, fmt.Errorf("import CSV row %d: %w", i+2, err)
		}
		rowsAffected, err := execution.RowsAffected()
		if err != nil {
			return CSVImportResult{}, fmt.Errorf("read CSV row %d import result: %w", i+2, err)
		}
		if rowsAffected == 0 {
			result.Skipped++
		} else {
			result.Inserted++
		}
	}

	if err := tx.Commit(); err != nil {
		return CSVImportResult{}, fmt.Errorf("commit CSV import: %w", err)
	}
	return result, nil
}

func csvSourceKey(path string, line int, row []string) (string, error) {
	canonicalRow, err := json.Marshal(row)
	if err != nil {
		return "", err
	}
	digest := sha256.New()
	digest.Write([]byte("csv:v1\x00"))
	digest.Write([]byte(filepath.Clean(path)))
	digest.Write([]byte("\x00"))
	digest.Write([]byte(strconv.Itoa(line)))
	digest.Write([]byte("\x00"))
	digest.Write(canonicalRow)
	return hex.EncodeToString(digest.Sum(nil)), nil
}
