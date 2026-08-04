package unit

import (
	"testing"
	"time"

	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/repositories"
	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/services"
)

func day(value string) time.Time {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		panic(err)
	}
	return parsed
}

// Store 1 was scraped three times, store 2 twice — batches deliberately
// handed over out of order to prove the resolver sorts them itself.
func sampleBatches() []repositories.BatchRow {
	return []repositories.BatchRow{
		{KompetitorID: 1, BatchID: 11, ExecutedAt: day("2026-07-01")},
		{KompetitorID: 1, BatchID: 13, ExecutedAt: day("2026-07-20")},
		{KompetitorID: 1, BatchID: 12, ExecutedAt: day("2026-07-10")},
		{KompetitorID: 2, BatchID: 21, ExecutedAt: day("2026-07-05")},
		{KompetitorID: 2, BatchID: 22, ExecutedAt: day("2026-07-18")},
	}
}

func TestForPeriodPicksNewestBatchInsideRange(t *testing.T) {
	resolver := services.NewPeriodResolver(sampleBatches())

	pairs := resolver.ForPeriod("2026-07-01", "2026-07-15")

	if got := pairs[1]; got != 12 {
		t.Fatalf("store 1: want batch 12 (newest inside range), got %d", got)
	}
	if got := pairs[2]; got != 21 {
		t.Fatalf("store 2: want batch 21, got %d", got)
	}
}

func TestForPeriodWithoutBoundsUsesLatestPerStore(t *testing.T) {
	resolver := services.NewPeriodResolver(sampleBatches())

	pairs := resolver.ForPeriod("", "")

	if pairs[1] != 13 || pairs[2] != 22 {
		t.Fatalf("want latest batches 13/22, got %v", pairs)
	}
}

func TestBeforePeriodIsNewestBatchBeforeStart(t *testing.T) {
	resolver := services.NewPeriodResolver(sampleBatches())

	pairs := resolver.BeforePeriod("2026-07-15", "2026-07-31")

	// Store 1 has 13 inside the period, so its comparison is 12.
	if got := pairs[1]; got != 12 {
		t.Fatalf("store 1: want batch 12 as comparison, got %d", got)
	}
	// Store 2 has 22 inside the period, so its comparison is 21.
	if got := pairs[2]; got != 21 {
		t.Fatalf("store 2: want batch 21 as comparison, got %d", got)
	}
}

// A store with no data inside the period must be left out of the comparison
// too, so both periods' numbers come from the same set of stores.
func TestBeforePeriodSkipsStoresAbsentFromThePeriod(t *testing.T) {
	batches := []repositories.BatchRow{
		{KompetitorID: 1, BatchID: 11, ExecutedAt: day("2026-07-20")},
		{KompetitorID: 1, BatchID: 10, ExecutedAt: day("2026-07-01")},
		// Store 2 was only ever scraped before the period.
		{KompetitorID: 2, BatchID: 20, ExecutedAt: day("2026-07-02")},
	}
	resolver := services.NewPeriodResolver(batches)

	pairs := resolver.BeforePeriod("2026-07-15", "2026-07-31")

	if _, ok := pairs[2]; ok {
		t.Fatalf("store 2 has no data in the period, it must not appear in the comparison: %v", pairs)
	}
	if pairs[1] != 10 {
		t.Fatalf("store 1: want batch 10 as comparison, got %d", pairs[1])
	}
}

func TestBeforePeriodWithoutStartUsesSecondNewest(t *testing.T) {
	resolver := services.NewPeriodResolver(sampleBatches())

	pairs := resolver.BeforePeriod("", "")

	if pairs[1] != 12 || pairs[2] != 21 {
		t.Fatalf("want second-newest batches 12/21, got %v", pairs)
	}
}

func TestResolveBatchByKompetitorFlagsStaleStores(t *testing.T) {
	latestExecutedAt := day("2026-07-20")
	rows := []repositories.BatchRow{
		{KompetitorID: 1, BatchID: 13, ExecutedAt: latestExecutedAt},
		{KompetitorID: 2, BatchID: 12, ExecutedAt: day("2026-07-18")},
	}

	resolved := services.ResolveBatchByKompetitor(rows, 13)

	if resolved[1].IsStale {
		t.Fatal("store 1 was scraped in the latest batch, it must not be stale")
	}
	if !resolved[2].IsStale {
		t.Fatal("store 2 fell back to an older batch, it must be stale")
	}
}
