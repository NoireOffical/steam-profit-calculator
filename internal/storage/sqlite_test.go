package storage

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"steam-profit-calculator/internal/domain"
)

func TestSQLiteRepositoryAppendAndQuery(t *testing.T) {
	fixedTime := time.Date(2026, time.August, 6, 12, 30, 0, 123456000, time.UTC)
	repository, err := openSQLiteRepository(filepath.Join(t.TempDir(), "records.db"), func() time.Time { return fixedTime })
	if err != nil {
		t.Fatalf("openSQLiteRepository() error = %v", err)
	}
	t.Cleanup(func() { _ = repository.Close() })

	records := []domain.ProfitRecord{
		testProfitRecord("AK-47 Redline", 12.3456),
		testProfitRecord("AK-47 Asiimov", -1.2344),
		testProfitRecord("M9 Bayonet", 98.7654),
	}
	stored, err := repository.Append(context.Background(), records)
	if err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if len(stored) != 3 {
		t.Fatalf("len(stored) = %d, want 3", len(stored))
	}
	for i, value := range stored {
		if value.ID != int64(i+1) {
			t.Fatalf("stored[%d].ID = %d, want %d", i, value.ID, i+1)
		}
		if !value.CreatedAt.Equal(fixedTime) {
			t.Fatalf("stored[%d].CreatedAt = %s, want %s", i, value.CreatedAt, fixedTime)
		}
	}

	page, err := repository.Query(context.Background(), RecordQuery{
		ItemPrefix: "ak-47",
		Outcome:    RecordOutcomeLoss,
		Order:      RecordOrderNewest,
		Limit:      1,
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if page.Total != 1 || len(page.Records) != 1 {
		t.Fatalf("Query() page = %#v, want one match", page)
	}
	got := page.Records[0]
	if got.Record.ItemName != "AK-47 Asiimov" || got.Record.TotalProfit != -1.234 {
		t.Fatalf("Query() record = %#v, want canonical losing AK-47 record", got.Record)
	}

	page, err = repository.Query(context.Background(), RecordQuery{Order: RecordOrderOldest, Limit: 2, Offset: 1})
	if err != nil {
		t.Fatalf("Query(paginated) error = %v", err)
	}
	if page.Total != 3 || len(page.Records) != 2 || page.Records[0].ID != 2 || page.Records[1].ID != 3 {
		t.Fatalf("Query(paginated) page = %#v", page)
	}
}

func TestSQLiteRepositoryAppendRollsBackCompleteBatch(t *testing.T) {
	repository, err := OpenSQLiteRepository(filepath.Join(t.TempDir(), "records.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteRepository() error = %v", err)
	}
	t.Cleanup(func() { _ = repository.Close() })

	if _, err := repository.db.Exec(`CREATE TRIGGER reject_second_record BEFORE INSERT ON profit_records
		WHEN NEW.item_name = 'reject-me'
		BEGIN
			SELECT RAISE(ABORT, 'forced insert failure');
		END`); err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}

	_, err = repository.Append(context.Background(), []domain.ProfitRecord{
		testProfitRecord("first", 1),
		testProfitRecord("reject-me", 2),
	})
	if err == nil || !strings.Contains(err.Error(), "forced insert failure") {
		t.Fatalf("Append() error = %v, want forced insert failure", err)
	}
	page, queryErr := repository.Query(context.Background(), RecordQuery{})
	if queryErr != nil {
		t.Fatalf("Query() error = %v", queryErr)
	}
	if page.Total != 0 {
		t.Fatalf("Query() total = %d, want rollback to leave 0", page.Total)
	}
}

func TestSQLiteRepositoryRejectsInvalidInputWithoutSideEffects(t *testing.T) {
	repository, err := OpenSQLiteRepository(filepath.Join(t.TempDir(), "records.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteRepository() error = %v", err)
	}
	t.Cleanup(func() { _ = repository.Close() })

	invalid := testProfitRecord("", 1)
	if _, err := repository.Append(context.Background(), []domain.ProfitRecord{invalid}); err == nil {
		t.Fatal("Append() error = nil, want invalid record error")
	}
	if _, err := repository.Query(context.Background(), RecordQuery{Limit: MaxRecordLimit + 1}); err == nil {
		t.Fatal("Query() error = nil, want invalid limit error")
	}
	page, err := repository.Query(context.Background(), RecordQuery{})
	if err != nil {
		t.Fatalf("Query(default) error = %v", err)
	}
	if page.Total != 0 {
		t.Fatalf("Query(default) total = %d, want 0", page.Total)
	}
}

func TestSQLiteRepositoryMigrationsAndQueryIndexes(t *testing.T) {
	repository, err := OpenSQLiteRepository(filepath.Join(t.TempDir(), "records.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteRepository() error = %v", err)
	}
	t.Cleanup(func() { _ = repository.Close() })

	var version int
	if err := repository.db.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatalf("read schema version: %v", err)
	}
	if version != currentSchemaVersion {
		t.Fatalf("schema version = %d, want %d", version, currentSchemaVersion)
	}

	rows, err := repository.db.Query(`PRAGMA index_list('profit_records')`)
	if err != nil {
		t.Fatalf("PRAGMA index_list: %v", err)
	}
	defer rows.Close()
	indexes := map[string]bool{}
	for rows.Next() {
		var sequence, unique, partial int
		var name, origin string
		if err := rows.Scan(&sequence, &name, &unique, &origin, &partial); err != nil {
			t.Fatalf("scan index: %v", err)
		}
		indexes[name] = true
	}
	for _, name := range []string{"idx_profit_records_created_id", "idx_profit_records_item_created_id"} {
		if !indexes[name] {
			t.Fatalf("index %q missing from %#v", name, indexes)
		}
	}

	planRows, err := repository.db.Query(`EXPLAIN QUERY PLAN SELECT id FROM profit_records ORDER BY created_at_unix_micros DESC, id DESC LIMIT 20`)
	if err != nil {
		t.Fatalf("EXPLAIN QUERY PLAN: %v", err)
	}
	defer planRows.Close()
	var plan strings.Builder
	for planRows.Next() {
		var id, parent, notUsed int
		var detail string
		if err := planRows.Scan(&id, &parent, &notUsed, &detail); err != nil {
			t.Fatalf("scan query plan: %v", err)
		}
		plan.WriteString(detail)
	}
	if !strings.Contains(plan.String(), "idx_profit_records_created_id") {
		t.Fatalf("query plan = %q, want creation index", plan.String())
	}
}

func TestSQLiteRepositoryMigrationFailureIsAtomic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "records.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE profit_records (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("create conflicting table: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close setup database: %v", err)
	}

	if _, err := OpenSQLiteRepository(path); err == nil {
		t.Fatal("OpenSQLiteRepository() error = nil, want migration failure")
	}
	check, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open database for verification: %v", err)
	}
	defer check.Close()
	var ledgerCount int
	if err := check.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'schema_migrations'`).Scan(&ledgerCount); err != nil {
		t.Fatalf("inspect migration ledger: %v", err)
	}
	if ledgerCount != 0 {
		t.Fatalf("migration ledger count = %d, want transaction rollback", ledgerCount)
	}
}

func TestSQLiteRepositoryDependencyFailureIsReturned(t *testing.T) {
	repository, err := OpenSQLiteRepository(filepath.Join(t.TempDir(), "records.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteRepository() error = %v", err)
	}
	if err := repository.db.Close(); err != nil {
		t.Fatalf("close database: %v", err)
	}

	if _, err := repository.Append(context.Background(), []domain.ProfitRecord{testProfitRecord("item", 1)}); err == nil {
		t.Fatal("Append() error = nil after dependency close")
	}
	if _, err := repository.Query(context.Background(), RecordQuery{}); err == nil {
		t.Fatal("Query() error = nil after dependency close")
	}
	if err := repository.HealthCheck(context.Background()); err == nil {
		t.Fatal("HealthCheck() error = nil after dependency close")
	}
}

func TestSQLiteRepositoryImportCSVIsAtomicAndIdempotent(t *testing.T) {
	directory := t.TempDir()
	csvPath := filepath.Join(directory, "records.csv")
	csvStore := NewCSVStore(csvPath)
	if err := csvStore.Append([]domain.ProfitRecord{
		testProfitRecord("Capsule", 1.234),
		testProfitRecord("Knife", -2.345),
	}); err != nil {
		t.Fatalf("prepare CSV: %v", err)
	}
	repository, err := OpenSQLiteRepository(filepath.Join(directory, "records.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteRepository() error = %v", err)
	}
	t.Cleanup(func() { _ = repository.Close() })

	first, err := repository.ImportCSV(context.Background(), csvPath)
	if err != nil {
		t.Fatalf("ImportCSV(first) error = %v", err)
	}
	if first.Inserted != 2 || first.Skipped != 0 {
		t.Fatalf("ImportCSV(first) = %#v", first)
	}
	second, err := repository.ImportCSV(context.Background(), csvPath)
	if err != nil {
		t.Fatalf("ImportCSV(second) error = %v", err)
	}
	if second.Inserted != 0 || second.Skipped != 2 {
		t.Fatalf("ImportCSV(second) = %#v", second)
	}

	if err := csvStore.Append([]domain.ProfitRecord{testProfitRecord("Sticker", 3.456)}); err != nil {
		t.Fatalf("append CSV row: %v", err)
	}
	third, err := repository.ImportCSV(context.Background(), csvPath)
	if err != nil {
		t.Fatalf("ImportCSV(third) error = %v", err)
	}
	if third.Inserted != 1 || third.Skipped != 2 {
		t.Fatalf("ImportCSV(third) = %#v", third)
	}
	page, err := repository.Query(context.Background(), RecordQuery{Limit: 10})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if page.Total != 3 {
		t.Fatalf("Query() total = %d, want 3", page.Total)
	}
}

func TestSQLiteRepositoryInvalidCSVImportHasNoSideEffect(t *testing.T) {
	directory := t.TempDir()
	csvPath := filepath.Join(directory, "invalid.csv")
	content := "item_name,buy_price,sell_price,quantity,net_profit,total_profit,knife_ratio,fee\nBroken,not-a-number,2,1,0,0,1,default\n"
	if err := os.WriteFile(csvPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write invalid CSV: %v", err)
	}
	repository, err := OpenSQLiteRepository(filepath.Join(directory, "records.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteRepository() error = %v", err)
	}
	t.Cleanup(func() { _ = repository.Close() })

	if _, err := repository.ImportCSV(context.Background(), csvPath); err == nil {
		t.Fatal("ImportCSV() error = nil, want invalid CSV error")
	}
	page, err := repository.Query(context.Background(), RecordQuery{})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if page.Total != 0 {
		t.Fatalf("Query() total = %d, want 0", page.Total)
	}
}

func TestCSVRepositorySupportsLegacyWebMode(t *testing.T) {
	repository := NewCSVRepository(filepath.Join(t.TempDir(), "records.csv"))
	stored, err := repository.Append(context.Background(), []domain.ProfitRecord{
		testProfitRecord("Alpha", 1),
		testProfitRecord("Beta", -1),
	})
	if err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if len(stored) != 2 || stored[1].ID != 2 {
		t.Fatalf("Append() stored = %#v", stored)
	}
	page, err := repository.Query(context.Background(), RecordQuery{Outcome: RecordOutcomeLoss})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if page.Total != 1 || page.Records[0].Record.ItemName != "Beta" {
		t.Fatalf("Query() page = %#v", page)
	}
	if err := repository.HealthCheck(context.Background()); err != nil {
		t.Fatalf("HealthCheck() error = %v", err)
	}
}

func TestCSVRepositoryHealthCheckRejectsInvalidExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "records.csv")
	if err := os.WriteFile(path, []byte("not,a,compatible,header\n"), 0o644); err != nil {
		t.Fatalf("write invalid CSV: %v", err)
	}
	if err := NewCSVRepository(path).HealthCheck(context.Background()); err == nil {
		t.Fatal("HealthCheck() error = nil for invalid CSV")
	}
}

func testProfitRecord(itemName string, totalProfit float64) domain.ProfitRecord {
	return domain.ProfitRecord{
		ItemName:    itemName,
		BuyPrice:    100,
		SellPrice:   120,
		Quantity:    2,
		NetProfit:   totalProfit / 2,
		TotalProfit: totalProfit,
		KnifeRatio:  0.9956194,
		Fee:         "default",
	}
}
