// Package services holds the kompetitor module's pure domain logic —
// period resolution, TF-IDF matching, margin and stock math. These carry
// business rules only: no HTTP, no gRPC, and (except where a repository
// interface is injected) no persistence.
package services

import (
	"sort"

	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/repositories"
)

// PeriodResolver filters scraping data by date period rather than by a
// single batch — a port of App\Services\Concerns\ResolvesScrapingPeriods.
//
// Each scraping batch only covers SOME of the stores, so one period can
// span many batches and a store can have more than one batch inside it.
// The batch used per store is the NEWEST one inside the period: summing
// every batch would double-count a store that happened to be scraped twice.
//
// The comparison period ("periode lalu") is each store's newest batch that
// falls before the period starts.
type PeriodResolver struct {
	// byKompetitor holds each kompetitor's batches, newest first.
	byKompetitor map[uint64][]repositories.BatchRow
}

// NewPeriodResolver groups raw (kompetitor, batch) rows by kompetitor and
// sorts each group newest-first, so the caller can hand rows over in any order.
func NewPeriodResolver(rows []repositories.BatchRow) *PeriodResolver {
	grouped := make(map[uint64][]repositories.BatchRow)
	for _, row := range rows {
		grouped[row.KompetitorID] = append(grouped[row.KompetitorID], row)
	}
	for id := range grouped {
		batches := grouped[id]
		sort.SliceStable(batches, func(i, j int) bool {
			return batches[i].ExecutedAt.After(batches[j].ExecutedAt)
		})
	}
	return &PeriodResolver{byKompetitor: grouped}
}

// ForPeriod returns the active batch per kompetitor for the requested
// period. Both bounds empty means "latest data": each kompetitor's newest
// batch, with no date limit. Dates are "YYYY-MM-DD".
func (r *PeriodResolver) ForPeriod(startDate, endDate string) repositories.BatchPair {
	pairs := repositories.BatchPair{}
	for id, batches := range r.byKompetitor {
		if batch := firstWithin(batches, startDate, endDate); batch != nil {
			pairs[id] = batch.BatchID
		}
	}
	return pairs
}

// BeforePeriod returns the comparison batch per kompetitor: the newest
// batch before the period starts. With no start date, the comparison is
// the batch just before each store's newest one.
func (r *PeriodResolver) BeforePeriod(startDate, endDate string) repositories.BatchPair {
	pairs := repositories.BatchPair{}

	for id, batches := range r.byKompetitor {
		if startDate == "" {
			if len(batches) > 1 {
				pairs[id] = batches[1].BatchID
			}
			continue
		}

		// Stores with no data in this period are left out of the comparison
		// too, so both periods' numbers come from the same set of stores.
		if firstWithin(batches, startDate, endDate) == nil {
			continue
		}

		for _, batch := range batches {
			if executedDate(batch) < startDate {
				pairs[id] = batch.BatchID
				break
			}
		}
	}

	return pairs
}

// firstWithin returns the newest batch inside the date range; empty bounds
// are unbounded.
func firstWithin(batches []repositories.BatchRow, startDate, endDate string) *repositories.BatchRow {
	for i := range batches {
		date := executedDate(batches[i])

		if startDate != "" && date < startDate {
			continue
		}
		if endDate != "" && date > endDate {
			continue
		}
		return &batches[i]
	}
	return nil
}

// executedDate formats a batch's execution timestamp as YYYY-MM-DD for
// string comparison, mirroring the PHP trait's substr(..., 0, 10).
func executedDate(batch repositories.BatchRow) string {
	return batch.ExecutedAt.Format("2006-01-02")
}
