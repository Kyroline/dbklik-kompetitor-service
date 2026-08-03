package unit

import (
	"testing"

	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/repositories"
	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/services"
)

func TestTokenizeStripsPunctuationAndUppercases(t *testing.T) {
	got := services.TextMatcher{}.Tokenize("Monitor Xiaomi G24i-2026 (75Hz)")

	want := []string{"MONITOR", "XIAOMI", "G24I", "2026", "75HZ"}
	if len(got) != len(want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("want %v, got %v", want, got)
		}
	}
}

func TestComputeMatchesPicksBestProductPerKompetitor(t *testing.T) {
	matcher := services.NewTextMatcher()

	items := []repositories.MappableItem{
		{ID: 100, NamaAccurate: "MONITOR XIAOMI G24I 24 INCH"},
	}
	products := []repositories.MatchableProduct{
		{ID: 1, KompetitorID: 7, Name: "Monitor Xiaomi G24i 24 inch", Price: 1_500_000},
		{ID: 2, KompetitorID: 7, Name: "Mouse Logitech G102", Price: 250_000},
		{ID: 3, KompetitorID: 8, Name: "MONITOR XIAOMI G24I 24 INCH", Price: 1_450_000},
	}

	idf := matcher.ComputeIDF(
		[]string{products[0].Name, products[1].Name, products[2].Name},
		[]string{items[0].NamaAccurate},
	)

	allowed := map[uint64]map[uint64]bool{100: {7: true, 8: true}}

	matches := matcher.ComputeMatches(items, products, 0.6, idf, allowed)

	best := matches[100]
	if len(best) != 2 {
		t.Fatalf("want a match for both kompetitors, got %d", len(best))
	}
	if best[7].Product.ID != 1 {
		t.Fatalf("kompetitor 7: want product 1, got %d", best[7].Product.ID)
	}
	if best[8].Product.ID != 3 {
		t.Fatalf("kompetitor 8: want product 3, got %d", best[8].Product.ID)
	}
	if best[8].Score > 1 {
		t.Fatalf("score must be capped at 1, got %f", best[8].Score)
	}
}

// Kompetitors outside the item's (kategori, brand) mapping must never be
// matched, even when the names are identical.
func TestComputeMatchesRespectsAllowedKompetitors(t *testing.T) {
	matcher := services.NewTextMatcher()

	items := []repositories.MappableItem{{ID: 100, NamaAccurate: "MONITOR XIAOMI G24I"}}
	products := []repositories.MatchableProduct{
		{ID: 1, KompetitorID: 9, Name: "MONITOR XIAOMI G24I"},
	}
	idf := matcher.ComputeIDF([]string{products[0].Name}, []string{items[0].NamaAccurate})

	matches := matcher.ComputeMatches(items, products, 0.6, idf, map[uint64]map[uint64]bool{
		100: {7: true},
	})

	if len(matches) != 0 {
		t.Fatalf("kompetitor 9 is not mapped to this item, want no matches, got %v", matches)
	}
}

func TestComputeMatchesDropsScoresAtOrBelowThreshold(t *testing.T) {
	matcher := services.NewTextMatcher()

	items := []repositories.MappableItem{{ID: 100, NamaAccurate: "MONITOR XIAOMI G24I 24 INCH"}}
	products := []repositories.MatchableProduct{
		{ID: 1, KompetitorID: 7, Name: "KEYBOARD LOGITECH K120 WIRED"},
	}
	idf := matcher.ComputeIDF([]string{products[0].Name}, []string{items[0].NamaAccurate})

	matches := matcher.ComputeMatches(items, products, 0.6, idf, map[uint64]map[uint64]bool{
		100: {7: true},
	})

	if len(matches) != 0 {
		t.Fatalf("unrelated names must not match, got %v", matches)
	}
}
