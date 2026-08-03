package services

import (
	"sort"
	"time"

	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/repositories"
)

// MaxStaleDays is how old a kompetitor's data may be before it stops being
// an acceptable fallback (ProductMappingService::MAX_STALE_DAYS).
const MaxStaleDays = 3

// EffectiveBatch is the batch a kompetitor's prices are read from on the
// Our Product panel, plus whether that batch is older than the latest one.
type EffectiveBatch struct {
	BatchID    uint64
	ExecutedAt time.Time
	IsStale    bool
}

// ResolveBatchByKompetitor picks the effective batch per kompetitor — a
// port of ProductMappingService::resolveBatchByKompetitor.
//
// Kompetitors present in the latest batch use that batch: if a product is
// missing there, that is a valid signal (delisted / no price), so the cell
// is deliberately left empty. Kompetitors that were NOT scraped yet in the
// latest batch fall back to the last batch that does have their data, as
// long as it is within MaxStaleDays of the latest batch — older than that
// is considered too stale to base a pricing decision on.
//
// rows must already be limited to completed batches inside that window;
// they are sorted ascending here so newer batches overwrite older ones.
func ResolveBatchByKompetitor(rows []repositories.BatchRow, latestBatchID uint64) map[uint64]EffectiveBatch {
	sorted := make([]repositories.BatchRow, len(rows))
	copy(sorted, rows)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].ExecutedAt.Before(sorted[j].ExecutedAt)
	})

	resolved := make(map[uint64]EffectiveBatch, len(sorted))
	for _, row := range sorted {
		resolved[row.KompetitorID] = EffectiveBatch{
			BatchID:    row.BatchID,
			ExecutedAt: row.ExecutedAt,
			IsStale:    row.BatchID != latestBatchID,
		}
	}

	return resolved
}

// StaleCutoff is the oldest execution time still accepted as a fallback.
func StaleCutoff(latestExecutedAt time.Time, maxStaleDays int) time.Time {
	return latestExecutedAt.AddDate(0, 0, -maxStaleDays)
}
