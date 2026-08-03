package services

import (
	"strconv"

	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/repositories"
)

// Pair is one (kategori, brand) cell. A nil KategoriID means the brand's
// universal cell.
type Pair struct {
	KategoriID *uint64
	BrandID    uint64
}

// MappingResolver is the source of truth for kompetitor mapping: one
// kompetitor list per (kategori, brand) pair — a port of
// App\Services\CompetitorMappingService's lookup half.
//
// A brand can also have a "universal" mapping (kategori_id NULL) covering
// that brand in any kategori that has no specific cell of its own.
// E.g. Xiaomi (universal) = A,B,C,D; Xiaomi × Monitor (specific) = E,F,G,H.
// Item "Monitor Xiaomi ..." → E,F,G,H (the specific cell wins outright, it
// is NOT unioned). Item "HP Xiaomi ..." (HP not mapped for Xiaomi) → A,B,C,D.
//
// Lookups are cached in the instance, so looping over items (e.g. one page
// of Our Product) does not hit the database repeatedly. Build one resolver
// per request — it is not safe for concurrent use.
type MappingResolver struct {
	repo repositories.MappingRepository

	// pairCache holds specific cells only, keyed "kategoriID|brandID".
	pairCache map[string][]repositories.KompetitorRef
	// universalCache holds universal cells (kategori_id NULL) per brand.
	universalCache map[uint64][]repositories.KompetitorRef
}

func NewMappingResolver(repo repositories.MappingRepository) *MappingResolver {
	return &MappingResolver{
		repo:           repo,
		pairCache:      map[string][]repositories.KompetitorRef{},
		universalCache: map[uint64][]repositories.KompetitorRef{},
	}
}

// ForPair returns the kompetitors of one cell. A nil kategoriID asks for
// the brand's universal mapping directly. For a real item the specific
// cell wins outright over universal when it is filled — not unioned.
func (r *MappingResolver) ForPair(kategoriID *uint64, brandID *uint64) ([]repositories.KompetitorRef, error) {
	if brandID == nil || *brandID == 0 {
		return nil, nil
	}
	if kategoriID == nil || *kategoriID == 0 {
		return r.UniversalFor(*brandID)
	}

	key := cellKey(*kategoriID, *brandID)
	if _, ok := r.pairCache[key]; !ok {
		if err := r.PreloadPairs([]Pair{{KategoriID: kategoriID, BrandID: *brandID}}); err != nil {
			return nil, err
		}
	}

	if specific := r.pairCache[key]; len(specific) > 0 {
		return specific, nil
	}
	return r.UniversalFor(*brandID)
}

// RawPair returns a specific cell's contents WITHOUT the universal
// fallback. The matrix panel uses this when opening a cell's edit modal so
// an unmapped cell shows up empty instead of echoing the universal list.
func (r *MappingResolver) RawPair(kategoriID, brandID uint64) ([]repositories.KompetitorRef, error) {
	key := cellKey(kategoriID, brandID)
	if _, ok := r.pairCache[key]; !ok {
		if err := r.PreloadPairs([]Pair{{KategoriID: &kategoriID, BrandID: brandID}}); err != nil {
			return nil, err
		}
	}
	return r.pairCache[key], nil
}

// UniversalFor returns a brand's universal mapping (kategori_id NULL) —
// the fallback used when a (kategori, brand) combination has no specific cell.
func (r *MappingResolver) UniversalFor(brandID uint64) ([]repositories.KompetitorRef, error) {
	if _, ok := r.universalCache[brandID]; !ok {
		if err := r.preloadUniversal([]uint64{brandID}); err != nil {
			return nil, err
		}
	}
	return r.universalCache[brandID], nil
}

// PreloadPairs loads many (kategori, brand) cells in one query, plus those
// brands' universal mappings, so ForPair's fallback does not hit the
// database once per item.
func (r *MappingResolver) PreloadPairs(pairs []Pair) error {
	wanted := map[string]Pair{}
	var brandIDs []uint64

	for _, pair := range pairs {
		if pair.BrandID == 0 {
			continue
		}
		brandIDs = append(brandIDs, pair.BrandID)

		if pair.KategoriID == nil || *pair.KategoriID == 0 {
			continue
		}
		key := cellKey(*pair.KategoriID, pair.BrandID)
		if _, cached := r.pairCache[key]; !cached {
			wanted[key] = pair
		}
	}

	if len(brandIDs) > 0 {
		if err := r.preloadUniversal(brandIDs); err != nil {
			return err
		}
	}

	if len(wanted) == 0 {
		return nil
	}

	// Pairs without a mapping are still recorded as an empty list so the
	// next item does not ask the database again.
	var kategoriIDs, pairBrandIDs []uint64
	for key, pair := range wanted {
		r.pairCache[key] = []repositories.KompetitorRef{}
		kategoriIDs = append(kategoriIDs, *pair.KategoriID)
		pairBrandIDs = append(pairBrandIDs, pair.BrandID)
	}

	rows, err := r.repo.PairRows(unique(kategoriIDs), unique(pairBrandIDs))
	if err != nil {
		return err
	}

	for _, row := range rows {
		if row.KategoriID == nil {
			continue
		}
		key := cellKey(*row.KategoriID, row.BrandID)

		// The IN × IN query also drags in cross pairs nobody asked for.
		if _, asked := wanted[key]; !asked {
			continue
		}
		r.pairCache[key] = append(r.pairCache[key], row.Kompetitor)
	}

	return nil
}

// preloadUniversal loads the universal mapping (kategori_id NULL) of many
// brands at once.
func (r *MappingResolver) preloadUniversal(brandIDs []uint64) error {
	var wanted []uint64
	for _, id := range unique(brandIDs) {
		if id == 0 {
			continue
		}
		if _, cached := r.universalCache[id]; !cached {
			wanted = append(wanted, id)
		}
	}
	if len(wanted) == 0 {
		return nil
	}

	for _, id := range wanted {
		r.universalCache[id] = []repositories.KompetitorRef{}
	}

	rows, err := r.repo.UniversalRows(wanted)
	if err != nil {
		return err
	}
	for _, row := range rows {
		r.universalCache[row.BrandID] = append(r.universalCache[row.BrandID], row.Kompetitor)
	}

	return nil
}

// AllowedByItem returns, per item, the set of kompetitors that item may be
// matched against — the filter applied by the TF-IDF matcher.
func (r *MappingResolver) AllowedByItem(items []repositories.MappableItem) (map[uint64]map[uint64]bool, error) {
	pairs := make([]Pair, 0, len(items))
	for _, item := range items {
		if item.BrandID == nil {
			continue
		}
		pairs = append(pairs, Pair{KategoriID: item.KategoriID, BrandID: *item.BrandID})
	}
	if err := r.PreloadPairs(pairs); err != nil {
		return nil, err
	}

	allowed := make(map[uint64]map[uint64]bool, len(items))
	for _, item := range items {
		refs, err := r.ForPair(item.KategoriID, item.BrandID)
		if err != nil {
			return nil, err
		}
		if len(refs) == 0 {
			continue
		}
		ids := make(map[uint64]bool, len(refs))
		for _, ref := range refs {
			ids[ref.ID] = true
		}
		allowed[item.ID] = ids
	}

	return allowed, nil
}

// Forget drops a cell from the cache after it has been rewritten.
func (r *MappingResolver) Forget(kategoriID *uint64, brandID uint64) {
	if kategoriID == nil {
		delete(r.universalCache, brandID)
		return
	}
	delete(r.pairCache, cellKey(*kategoriID, brandID))
}

// CellKey is the "kategoriID|brandID" key shared by the resolver cache and
// the matrix panel's count map.
func CellKey(kategoriID, brandID uint64) string { return cellKey(kategoriID, brandID) }

func cellKey(kategoriID, brandID uint64) string {
	return strconv.FormatUint(kategoriID, 10) + "|" + strconv.FormatUint(brandID, 10)
}

func unique(ids []uint64) []uint64 {
	seen := make(map[uint64]bool, len(ids))
	out := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}
