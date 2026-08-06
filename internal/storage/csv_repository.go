package storage

import (
	"context"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"steam-profit-calculator/internal/domain"
)

// CSVRepository adapts the compatibility CSV store to the web repository contract.
// Metadata is derived; SQLite should be used when durable metadata is required.
type CSVRepository struct {
	path  string
	store *CSVStore
	mu    sync.Mutex
	now   func() time.Time
}

// NewCSVRepository creates an explicit legacy web repository.
func NewCSVRepository(path string) *CSVRepository {
	return &CSVRepository{path: path, store: NewCSVStore(path), now: time.Now}
}

// HealthCheck verifies that an existing compatibility file remains readable.
// A missing file is healthy because the first successful append creates it.
func (r *CSVRepository) HealthCheck(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, err := r.store.List(); err != nil {
		return fmt.Errorf("read CSV repository: %w", err)
	}
	return nil
}

// Append writes through the existing compatible CSV store.
func (r *CSVRepository) Append(ctx context.Context, records []domain.ProfitRecord) ([]StoredProfitRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return []StoredProfitRecord{}, nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, err := r.store.List()
	if err != nil {
		return nil, fmt.Errorf("read CSV records before append: %w", err)
	}
	if err := r.store.Append(records); err != nil {
		return nil, err
	}
	timestamp := r.now().UTC().Truncate(time.Microsecond)
	stored := make([]StoredProfitRecord, 0, len(records))
	for i, record := range records {
		stored = append(stored, StoredProfitRecord{
			ID:        int64(len(existing) + i + 1),
			CreatedAt: timestamp,
			Record:    record,
		})
	}
	return stored, nil
}

// Query reads and filters the compatibility CSV file in memory.
func (r *CSVRepository) Query(ctx context.Context, query RecordQuery) (RecordPage, error) {
	query, err := NormalizeRecordQuery(query)
	if err != nil {
		return RecordPage{}, err
	}
	if err := ctx.Err(); err != nil {
		return RecordPage{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	records, err := r.store.List()
	if err != nil {
		return RecordPage{}, err
	}

	timestamp := time.Unix(0, 0).UTC()
	if info, statErr := os.Stat(r.path); statErr == nil {
		timestamp = info.ModTime().UTC().Truncate(time.Microsecond)
	}
	matched := make([]StoredProfitRecord, 0, len(records))
	for i, record := range records {
		if !recordMatches(record, query) {
			continue
		}
		matched = append(matched, StoredProfitRecord{
			ID:        int64(i + 1),
			CreatedAt: timestamp,
			Record:    record,
		})
	}
	sort.SliceStable(matched, func(i, j int) bool {
		if query.Order == RecordOrderOldest {
			return matched[i].ID < matched[j].ID
		}
		return matched[i].ID > matched[j].ID
	})

	page := RecordPage{Total: len(matched), Limit: query.Limit, Offset: query.Offset, Records: []StoredProfitRecord{}}
	if query.Offset >= len(matched) {
		return page, nil
	}
	end := query.Offset + query.Limit
	if end > len(matched) {
		end = len(matched)
	}
	page.Records = append(page.Records, matched[query.Offset:end]...)
	return page, nil
}
