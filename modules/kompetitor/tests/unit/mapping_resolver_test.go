package unit

import (
	"testing"

	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/repositories"
	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/services"
)

// fakeMappingRepo is an in-memory MappingRepository: only the two lookup
// methods the resolver uses are implemented meaningfully.
type fakeMappingRepo struct {
	rows      []repositories.MappingRow
	pairCalls int
}

func (r *fakeMappingRepo) PairRows(kategoriIDs, brandIDs []uint64) ([]repositories.MappingRow, error) {
	r.pairCalls++

	var out []repositories.MappingRow
	for _, row := range r.rows {
		if row.KategoriID == nil {
			continue
		}
		if contains(kategoriIDs, *row.KategoriID) && contains(brandIDs, row.BrandID) {
			out = append(out, row)
		}
	}
	return out, nil
}

func (r *fakeMappingRepo) UniversalRows(brandIDs []uint64) ([]repositories.MappingRow, error) {
	var out []repositories.MappingRow
	for _, row := range r.rows {
		if row.KategoriID == nil && contains(brandIDs, row.BrandID) {
			out = append(out, row)
		}
	}
	return out, nil
}

func (r *fakeMappingRepo) MatrixCounts([]uint64, []uint64) (map[string]int64, error) { return nil, nil }
func (r *fakeMappingRepo) UniversalCounts([]uint64) (map[uint64]int64, error)        { return nil, nil }
func (r *fakeMappingRepo) UsedAxes() ([]uint64, []uint64, error)                     { return nil, nil, nil }
func (r *fakeMappingRepo) SyncCell(*uint64, uint64, []uint64) error                  { return nil }
func (r *fakeMappingRepo) KompetitorIDsExist([]uint64) (bool, error)                 { return true, nil }

func contains(ids []uint64, id uint64) bool {
	for _, candidate := range ids {
		if candidate == id {
			return true
		}
	}
	return false
}

func ref(id uint64, name string) repositories.KompetitorRef {
	return repositories.KompetitorRef{ID: id, Name: name}
}

func ptr(v uint64) *uint64 { return &v }

// Xiaomi (universal) = A,B; Xiaomi × Monitor (specific) = C.
func xiaomiRepo() *fakeMappingRepo {
	return &fakeMappingRepo{rows: []repositories.MappingRow{
		{KategoriID: nil, BrandID: 5, Kompetitor: ref(1, "Toko A")},
		{KategoriID: nil, BrandID: 5, Kompetitor: ref(2, "Toko B")},
		{KategoriID: ptr(9), BrandID: 5, Kompetitor: ref(3, "Toko C")},
	}}
}

// The specific cell wins outright over universal — it is not unioned.
func TestForPairPrefersSpecificCellOverUniversal(t *testing.T) {
	resolver := services.NewMappingResolver(xiaomiRepo())

	refs, err := resolver.ForPair(ptr(9), ptr(5))
	if err != nil {
		t.Fatal(err)
	}

	if len(refs) != 1 || refs[0].ID != 3 {
		t.Fatalf("want only the specific cell (Toko C), got %v", refs)
	}
}

// A kategori without its own cell falls back to the brand's universal list.
func TestForPairFallsBackToUniversal(t *testing.T) {
	resolver := services.NewMappingResolver(xiaomiRepo())

	refs, err := resolver.ForPair(ptr(77), ptr(5))
	if err != nil {
		t.Fatal(err)
	}

	if len(refs) != 2 {
		t.Fatalf("want the universal list (2 stores), got %v", refs)
	}
}

// RawPair must not fall back, so an unmapped cell opens empty in the UI.
func TestRawPairDoesNotFallBackToUniversal(t *testing.T) {
	resolver := services.NewMappingResolver(xiaomiRepo())

	refs, err := resolver.RawPair(77, 5)
	if err != nil {
		t.Fatal(err)
	}

	if len(refs) != 0 {
		t.Fatalf("want an empty cell, got %v", refs)
	}
}

// Repeated lookups of the same cell must be served from the cache.
func TestForPairCachesLookups(t *testing.T) {
	repo := xiaomiRepo()
	resolver := services.NewMappingResolver(repo)

	for i := 0; i < 3; i++ {
		if _, err := resolver.ForPair(ptr(9), ptr(5)); err != nil {
			t.Fatal(err)
		}
	}

	if repo.pairCalls != 1 {
		t.Fatalf("want a single repository call, got %d", repo.pairCalls)
	}
}

func TestAllowedByItemSkipsItemsWithoutMapping(t *testing.T) {
	resolver := services.NewMappingResolver(xiaomiRepo())

	allowed, err := resolver.AllowedByItem([]repositories.MappableItem{
		{ID: 100, NamaAccurate: "MONITOR XIAOMI", BrandID: ptr(5), KategoriID: ptr(9)},
		{ID: 200, NamaAccurate: "TANPA BRAND", BrandID: nil, KategoriID: ptr(9)},
	})
	if err != nil {
		t.Fatal(err)
	}

	if !allowed[100][3] {
		t.Fatalf("item 100 should be matchable against kompetitor 3, got %v", allowed[100])
	}
	if _, ok := allowed[200]; ok {
		t.Fatal("an item without a brand has no mapping and must be absent")
	}
}
