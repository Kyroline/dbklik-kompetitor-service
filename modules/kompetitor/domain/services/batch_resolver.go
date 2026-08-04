package services

import (
	"sort"
	"time"

	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/repositories"
)

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
// latest batch fall back to the last completed batch that does have their
// data, no matter how old — there is no staleness cutoff anymore.
//
// rows must already be limited to completed batches up to the latest one;
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
