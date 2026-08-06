package storage

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"steam-profit-calculator/internal/domain"
)

const (
	// DefaultRecordLimit is the default bounded history page size.
	DefaultRecordLimit = 20
	// MaxRecordLimit is the largest history page accepted by a repository.
	MaxRecordLimit     = 100
	maxItemPrefixRunes = 100
)

// RecordOutcome limits history to non-losing or losing records.
type RecordOutcome string

const (
	RecordOutcomeAll    RecordOutcome = "all"
	RecordOutcomeProfit RecordOutcome = "profit"
	RecordOutcomeLoss   RecordOutcome = "loss"
)

// RecordOrder controls the stable creation order of history results.
type RecordOrder string

const (
	RecordOrderNewest RecordOrder = "newest"
	RecordOrderOldest RecordOrder = "oldest"
)

// StoredProfitRecord adds persistence metadata without changing domain output.
type StoredProfitRecord struct {
	ID        int64
	CreatedAt time.Time
	Record    domain.ProfitRecord
}

// RecordQuery describes one bounded history request.
type RecordQuery struct {
	ItemPrefix string
	Outcome    RecordOutcome
	Order      RecordOrder
	Limit      int
	Offset     int
}

// RecordPage contains one history page and the total number of matches.
type RecordPage struct {
	Records []StoredProfitRecord
	Total   int
	Limit   int
	Offset  int
}

// RecordRepository is the persistence boundary used by the web application.
type RecordRepository interface {
	Append(context.Context, []domain.ProfitRecord) ([]StoredProfitRecord, error)
	Query(context.Context, RecordQuery) (RecordPage, error)
}

// HealthChecker is implemented by repositories that can verify their required dependency.
type HealthChecker interface {
	HealthCheck(context.Context) error
}

// NormalizeRecordQuery applies defaults and validates one history query.
func NormalizeRecordQuery(query RecordQuery) (RecordQuery, error) {
	query.ItemPrefix = strings.TrimSpace(query.ItemPrefix)
	if utf8.RuneCountInString(query.ItemPrefix) > maxItemPrefixRunes {
		return RecordQuery{}, fmt.Errorf("item prefix must not exceed %d characters", maxItemPrefixRunes)
	}

	if query.Outcome == "" {
		query.Outcome = RecordOutcomeAll
	}
	switch query.Outcome {
	case RecordOutcomeAll, RecordOutcomeProfit, RecordOutcomeLoss:
	default:
		return RecordQuery{}, fmt.Errorf("outcome must be all, profit, or loss")
	}

	if query.Order == "" {
		query.Order = RecordOrderNewest
	}
	switch query.Order {
	case RecordOrderNewest, RecordOrderOldest:
	default:
		return RecordQuery{}, fmt.Errorf("order must be newest or oldest")
	}

	if query.Limit == 0 {
		query.Limit = DefaultRecordLimit
	}
	if query.Limit < 1 || query.Limit > MaxRecordLimit {
		return RecordQuery{}, fmt.Errorf("limit must be an integer between 1 and %d", MaxRecordLimit)
	}
	if query.Offset < 0 {
		return RecordQuery{}, fmt.Errorf("offset must be a non-negative integer")
	}

	return query, nil
}

func recordMatches(record domain.ProfitRecord, query RecordQuery) bool {
	if query.ItemPrefix != "" && !strings.HasPrefix(strings.ToLower(record.ItemName), strings.ToLower(query.ItemPrefix)) {
		return false
	}
	switch query.Outcome {
	case RecordOutcomeProfit:
		return record.TotalProfit >= 0
	case RecordOutcomeLoss:
		return record.TotalProfit < 0
	default:
		return true
	}
}
