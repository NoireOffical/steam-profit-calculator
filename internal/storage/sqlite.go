package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"steam-profit-calculator/internal/domain"

	_ "modernc.org/sqlite"
)

const currentSchemaVersion = 2

type schemaMigration struct {
	version    int
	statements []string
}

var schemaMigrations = []schemaMigration{
	{
		version: 1,
		statements: []string{
			`CREATE TABLE profit_records (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				item_name TEXT NOT NULL CHECK (length(trim(item_name)) > 0),
				buy_price_milli INTEGER NOT NULL,
				sell_price_milli INTEGER NOT NULL,
				quantity INTEGER NOT NULL CHECK (quantity > 0),
				net_profit_milli INTEGER NOT NULL,
				total_profit_milli INTEGER NOT NULL,
				knife_ratio_micros INTEGER NOT NULL,
				fee TEXT NOT NULL CHECK (length(trim(fee)) > 0),
				created_at_unix_micros INTEGER NOT NULL,
				source TEXT NOT NULL CHECK (source IN ('web', 'csv_import')),
				source_key TEXT UNIQUE,
				owner_id TEXT,
				session_id TEXT
			)`,
			`CREATE INDEX idx_profit_records_created_id ON profit_records (created_at_unix_micros DESC, id DESC)`,
		},
	},
	{
		version: 2,
		statements: []string{
			`CREATE INDEX idx_profit_records_item_created_id ON profit_records (item_name COLLATE NOCASE, created_at_unix_micros DESC, id DESC)`,
		},
	},
}

// SQLiteRepository persists web calculation records in a versioned SQLite database.
type SQLiteRepository struct {
	db  *sql.DB
	now func() time.Time
}

// OpenSQLiteRepository opens a database, configures it, and applies pending migrations.
func OpenSQLiteRepository(path string) (*SQLiteRepository, error) {
	return openSQLiteRepository(path, time.Now)
}

// HealthCheck verifies that the live connection can read the migration ledger.
func (r *SQLiteRepository) HealthCheck(ctx context.Context) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("SQLite repository is unavailable")
	}
	var version int
	if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&version); err != nil {
		return fmt.Errorf("read SQLite migration ledger: %w", err)
	}
	if version != currentSchemaVersion {
		return fmt.Errorf("SQLite schema version %d is not ready", version)
	}
	return nil
}

func openSQLiteRepository(path string, now func() time.Time) (*SQLiteRepository, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("database path is required")
	}
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, fmt.Errorf("create database directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open SQLite database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	repository := &SQLiteRepository{db: db, now: now}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := repository.initialize(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return repository, nil
}

func (r *SQLiteRepository) initialize(ctx context.Context) error {
	for _, statement := range []string{
		`PRAGMA foreign_keys = ON`,
		`PRAGMA busy_timeout = 5000`,
		`PRAGMA journal_mode = WAL`,
		`PRAGMA synchronous = NORMAL`,
	} {
		if _, err := r.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("configure SQLite database: %w", err)
		}
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin schema migration: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at_unix_micros INTEGER NOT NULL
	)`); err != nil {
		return fmt.Errorf("create schema migration ledger: %w", err)
	}

	var version int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&version); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	if version > currentSchemaVersion {
		return fmt.Errorf("database schema version %d is newer than supported version %d", version, currentSchemaVersion)
	}

	for _, migration := range schemaMigrations {
		if migration.version <= version {
			continue
		}
		for _, statement := range migration.statements {
			if _, err := tx.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("apply schema migration %d: %w", migration.version, err)
			}
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO schema_migrations (version, applied_at_unix_micros) VALUES (?, ?)`,
			migration.version,
			r.now().UTC().UnixMicro(),
		); err != nil {
			return fmt.Errorf("record schema migration %d: %w", migration.version, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit schema migrations: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, `PRAGMA optimize`); err != nil {
		return fmt.Errorf("optimize SQLite database: %w", err)
	}
	return nil
}

// Close optimizes and closes the database connection.
func (r *SQLiteRepository) Close() error {
	if r == nil || r.db == nil {
		return nil
	}
	_, optimizeErr := r.db.Exec(`PRAGMA optimize`)
	closeErr := r.db.Close()
	if optimizeErr != nil {
		return fmt.Errorf("optimize SQLite database: %w", optimizeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close SQLite database: %w", closeErr)
	}
	return nil
}

// Append stores a complete web calculation batch in one transaction.
func (r *SQLiteRepository) Append(ctx context.Context, records []domain.ProfitRecord) ([]StoredProfitRecord, error) {
	if len(records) == 0 {
		return []StoredProfitRecord{}, nil
	}
	encoded := make([]encodedRecord, len(records))
	for i, record := range records {
		value, err := encodeRecord(record)
		if err != nil {
			return nil, fmt.Errorf("encode record %d: %w", i+1, err)
		}
		encoded[i] = value
	}

	timestamp := r.now().UTC().Truncate(time.Microsecond)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin record append: %w", err)
	}
	defer tx.Rollback()

	stored := make([]StoredProfitRecord, 0, len(records))
	for i, value := range encoded {
		result, err := tx.ExecContext(ctx, insertRecordSQL,
			value.itemName,
			value.buyPriceMilli,
			value.sellPriceMilli,
			value.quantity,
			value.netProfitMilli,
			value.totalProfitMilli,
			value.knifeRatioMicros,
			value.fee,
			timestamp.UnixMicro(),
			"web",
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("insert record %d: %w", i+1, err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("read inserted record %d identity: %w", i+1, err)
		}
		stored = append(stored, StoredProfitRecord{ID: id, CreatedAt: timestamp, Record: records[i]})
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit record append: %w", err)
	}
	return stored, nil
}

const insertRecordSQL = `INSERT INTO profit_records (
	item_name,
	buy_price_milli,
	sell_price_milli,
	quantity,
	net_profit_milli,
	total_profit_milli,
	knife_ratio_micros,
	fee,
	created_at_unix_micros,
	source,
	source_key
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

// Query returns a validated, stable, bounded page of stored records.
func (r *SQLiteRepository) Query(ctx context.Context, query RecordQuery) (RecordPage, error) {
	query, err := NormalizeRecordQuery(query)
	if err != nil {
		return RecordPage{}, err
	}

	where, arguments := recordWhereClause(query)
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM profit_records`+where, arguments...).Scan(&total); err != nil {
		return RecordPage{}, fmt.Errorf("count profit records: %w", err)
	}

	order := ` ORDER BY created_at_unix_micros DESC, id DESC`
	if query.Order == RecordOrderOldest {
		order = ` ORDER BY created_at_unix_micros ASC, id ASC`
	}
	pageArguments := append(append([]any{}, arguments...), query.Limit, query.Offset)
	rows, err := r.db.QueryContext(ctx, `SELECT
		id,
		item_name,
		buy_price_milli,
		sell_price_milli,
		quantity,
		net_profit_milli,
		total_profit_milli,
		knife_ratio_micros,
		fee,
		created_at_unix_micros
	FROM profit_records`+where+order+` LIMIT ? OFFSET ?`, pageArguments...)
	if err != nil {
		return RecordPage{}, fmt.Errorf("query profit records: %w", err)
	}
	defer rows.Close()

	records := make([]StoredProfitRecord, 0, query.Limit)
	for rows.Next() {
		stored, err := scanStoredRecord(rows)
		if err != nil {
			return RecordPage{}, err
		}
		records = append(records, stored)
	}
	if err := rows.Err(); err != nil {
		return RecordPage{}, fmt.Errorf("iterate profit records: %w", err)
	}

	return RecordPage{Records: records, Total: total, Limit: query.Limit, Offset: query.Offset}, nil
}

func recordWhereClause(query RecordQuery) (string, []any) {
	clauses := make([]string, 0, 2)
	arguments := make([]any, 0, 2)
	if query.ItemPrefix != "" {
		clauses = append(clauses, `item_name LIKE ? ESCAPE '\' COLLATE NOCASE`)
		arguments = append(arguments, escapeLikePrefix(query.ItemPrefix)+"%")
	}
	switch query.Outcome {
	case RecordOutcomeProfit:
		clauses = append(clauses, `total_profit_milli >= 0`)
	case RecordOutcomeLoss:
		clauses = append(clauses, `total_profit_milli < 0`)
	}
	if len(clauses) == 0 {
		return "", arguments
	}
	return " WHERE " + strings.Join(clauses, " AND "), arguments
}

func escapeLikePrefix(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	return strings.ReplaceAll(value, `_`, `\_`)
}

type rowScanner interface {
	Scan(...any) error
}

func scanStoredRecord(scanner rowScanner) (StoredProfitRecord, error) {
	var (
		stored              StoredProfitRecord
		itemName            string
		buyPriceMilli       int64
		sellPriceMilli      int64
		quantity            int
		netProfitMilli      int64
		totalProfitMilli    int64
		knifeRatioMicros    int64
		fee                 string
		createdAtUnixMicros int64
	)
	if err := scanner.Scan(
		&stored.ID,
		&itemName,
		&buyPriceMilli,
		&sellPriceMilli,
		&quantity,
		&netProfitMilli,
		&totalProfitMilli,
		&knifeRatioMicros,
		&fee,
		&createdAtUnixMicros,
	); err != nil {
		return StoredProfitRecord{}, fmt.Errorf("scan profit record: %w", err)
	}
	stored.CreatedAt = time.UnixMicro(createdAtUnixMicros).UTC()
	stored.Record = domain.ProfitRecord{
		ItemName:    itemName,
		BuyPrice:    float64(buyPriceMilli) / 1_000,
		SellPrice:   float64(sellPriceMilli) / 1_000,
		Quantity:    quantity,
		NetProfit:   float64(netProfitMilli) / 1_000,
		TotalProfit: float64(totalProfitMilli) / 1_000,
		KnifeRatio:  float64(knifeRatioMicros) / 1_000_000,
		Fee:         fee,
	}
	return stored, nil
}

type encodedRecord struct {
	itemName         string
	buyPriceMilli    int64
	sellPriceMilli   int64
	quantity         int
	netProfitMilli   int64
	totalProfitMilli int64
	knifeRatioMicros int64
	fee              string
}

func encodeRecord(record domain.ProfitRecord) (encodedRecord, error) {
	if strings.TrimSpace(record.ItemName) == "" {
		return encodedRecord{}, fmt.Errorf("item name must not be empty")
	}
	if record.Quantity <= 0 {
		return encodedRecord{}, fmt.Errorf("quantity must be greater than 0")
	}
	if strings.TrimSpace(record.Fee) == "" {
		return encodedRecord{}, fmt.Errorf("fee must not be empty")
	}
	for name, value := range map[string]float64{
		"buy price": record.BuyPrice, "sell price": record.SellPrice, "net profit": record.NetProfit,
		"total profit": record.TotalProfit, "knife ratio": record.KnifeRatio,
	} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return encodedRecord{}, fmt.Errorf("%s must be finite", name)
		}
	}

	buyPrice, err := parseScaled(domain.FormatPrice(record.BuyPrice), 3)
	if err != nil {
		return encodedRecord{}, fmt.Errorf("encode buy price: %w", err)
	}
	sellPrice, err := parseScaled(domain.FormatPrice(record.SellPrice), 3)
	if err != nil {
		return encodedRecord{}, fmt.Errorf("encode sell price: %w", err)
	}
	netProfit, err := parseScaled(domain.FormatPrice(record.NetProfit), 3)
	if err != nil {
		return encodedRecord{}, fmt.Errorf("encode net profit: %w", err)
	}
	totalProfit, err := parseScaled(domain.FormatPrice(record.TotalProfit), 3)
	if err != nil {
		return encodedRecord{}, fmt.Errorf("encode total profit: %w", err)
	}
	knifeRatio, err := parseScaled(domain.FormatRatio(record.KnifeRatio), 6)
	if err != nil {
		return encodedRecord{}, fmt.Errorf("encode knife ratio: %w", err)
	}

	return encodedRecord{
		itemName:         record.ItemName,
		buyPriceMilli:    buyPrice,
		sellPriceMilli:   sellPrice,
		quantity:         record.Quantity,
		netProfitMilli:   netProfit,
		totalProfitMilli: totalProfit,
		knifeRatioMicros: knifeRatio,
		fee:              record.Fee,
	}, nil
}

func parseScaled(formatted string, precision int) (int64, error) {
	negative := strings.HasPrefix(formatted, "-")
	if negative {
		formatted = strings.TrimPrefix(formatted, "-")
	}
	parts := strings.Split(formatted, ".")
	if len(parts) != 2 || len(parts[1]) != precision {
		return 0, fmt.Errorf("invalid fixed precision value %q", formatted)
	}
	value, err := strconv.ParseInt(parts[0]+parts[1], 10, 64)
	if err != nil {
		return 0, err
	}
	if negative {
		if value == math.MinInt64 {
			return 0, errors.New("scaled value overflows int64")
		}
		value = -value
	}
	return value, nil
}
