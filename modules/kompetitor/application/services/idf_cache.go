package services

import (
	"sort"
	"strconv"
	"strings"
	"time"

	pkgcache "github.com/dbklik/dbklik-kompetitor-service/pkg/cache"
)

// idfTTL mirrors ProductMappingService's Cache::remember(now()->addHours(12)).
const idfTTL = 12 * time.Hour

// IdfCache caches TF-IDF token weights per batch set, the equivalent of
// ProductMappingService::idfForBatches()'s Laravel cache entry.
//
// IDF must come from the FULL batch corpus, not just the items on the
// current page: otherwise token weights follow whichever items happen to
// share a page and borderline matches (score ≈ threshold) flicker. Completed
// batches are immutable, so caching them is safe; ingest busts the entry for
// the batch it just rewrote.
type IdfCache struct {
	store *pkgcache.MemoryCache
}

func NewIdfCache() *IdfCache {
	return &IdfCache{store: pkgcache.NewMemoryCache()}
}

// Remember returns the cached IDF for this batch set, computing it via
// build on a miss.
func (c *IdfCache) Remember(batchIDs []uint64, build func() (map[string]float64, error)) (map[string]float64, error) {
	key := idfCacheKey(batchIDs)

	if cached, ok := c.store.Get(key); ok {
		if idf, ok := cached.(map[string]float64); ok {
			return idf, nil
		}
	}

	idf, err := build()
	if err != nil {
		return nil, err
	}

	c.store.Set(key, idf, idfTTL)
	return idf, nil
}

// Forget drops one batch set's entry — called after a batch is re-ingested.
func (c *IdfCache) Forget(batchIDs []uint64) {
	c.store.Delete(idfCacheKey(batchIDs))
}

func idfCacheKey(batchIDs []uint64) string {
	ids := make([]uint64, len(batchIDs))
	copy(ids, batchIDs)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatUint(id, 10)
	}

	return "scraping_batch_idf:" + strings.Join(parts, "-")
}
