package services

import (
	"sort"

	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/application/dto"
	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/repositories"
	domainservices "github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/services"
)

// dbklikDisplayName is the label of our own store's column ("Harga Tayang"),
// matching KompetitorController::ourProductData.
const dbklikDisplayName = "DB Klik"

// FilterOptions returns the brand/kategori dropdown values of the Our
// Product panel — a port of KompetitorController::filterOptions.
func (s *KompetitorService) FilterOptions() (dto.FilterOptionsOutput, error) {
	brands, kategoris, err := s.ourProducts.FilterOptions()
	if err != nil {
		return dto.FilterOptionsOutput{}, err
	}
	return dto.FilterOptionsOutput{Brands: brands, Kategoris: kategoris}, nil
}

// OurProducts returns one page of the Our Product table — a port of
// KompetitorController::ourProductData.
//
// Kompetitor prices are matched on the fly (TF-IDF) instead of read from
// scraping_product_mappings, so the table always follows the current
// brand/kategori ↔ kompetitor mapping: no manual `scraping:map-products`
// re-run is needed after the mapping changes.
func (s *KompetitorService) OurProducts(in dto.OurProductInput) (dto.OurProductOutput, error) {
	length := in.Length
	if length <= 0 {
		length = 10
	}

	rows, total, err := s.ourProducts.Page(repositories.OurProductFilters{
		Search:   in.Search,
		Brand:    in.Brand,
		Kategori: in.Kategori,
		Abc:      in.Abc,
	}, in.Start, length)
	if err != nil {
		return dto.OurProductOutput{}, err
	}

	matches, batchByKompetitor, resolver, err := s.matchOurProducts(rows)
	if err != nil {
		return dto.OurProductOutput{}, err
	}

	dbklikID := s.cfg.DbklikKompetitorID
	feeCache := newFeeCache(s.fees)
	calculator := domainservices.NewMarginCalculator()

	data := make([]dto.OurProductRow, 0, len(rows))
	for _, row := range rows {
		best := matches[row.ItemID]

		buildCell := func(kompetitorID uint64, name string) dto.KompetitorCell {
			cell := dto.KompetitorCell{Kompetitor: name}

			batchInfo, hasBatch := batchByKompetitor[kompetitorID]
			cell.BelumScrape = !hasBatch || batchInfo.IsStale

			match, matched := best[kompetitorID]
			if !matched {
				return cell
			}

			productName := match.Product.Name
			price := match.Product.Price
			cell.NamaProduk = &productName
			cell.Harga = &price
			cell.Stale = hasBatch && batchInfo.IsStale

			if hasBatch {
				date := batchInfo.ExecutedAt.Format("2006-01-02")
				cell.TanggalScraping = &date
			}

			return cell
		}

		// Union of the item's brand + kategori kompetitors (unique by id),
		// resolved through the (kategori, brand) cell with universal fallback.
		refs, err := resolver.ForPair(row.KategoriID, row.BrandID)
		if err != nil {
			return dto.OurProductOutput{}, err
		}

		kompetitorHarga := make([]dto.KompetitorCell, 0, len(refs))
		for _, ref := range refs {
			// Our own store is not a competitor column — it becomes Harga Tayang.
			if dbklikID != 0 && ref.ID == dbklikID {
				continue
			}
			kompetitorHarga = append(kompetitorHarga, buildCell(ref.ID, ref.Name))
		}

		out := dto.OurProductRow{
			ID:              row.HargaID,
			SKU:             row.SKU,
			Nama:            row.NamaAccurate,
			Kategori:        orDash(row.KategoriName),
			Brand:           orDash(row.BrandName),
			HppLatest:       row.HppLatest,
			HargaShopee:     row.Shopee,
			Abc:             row.Abc,
			KompetitorHarga: kompetitorHarga,
		}

		// Harga tayang = our own scraped store price matched to this SKU.
		// Unlike `harga_shopee` (read straight from the `harga` table), this
		// is the price actually shown on the marketplace.
		if dbklikID != 0 {
			cell := buildCell(dbklikID, dbklikDisplayName)
			out.HargaTayang = &cell
		}

		// Shopee margin uses the same marketplace-fee maths as the margin
		// calculator on the harga page. Note the Laravel caller passes the
		// default free-ongkir type ("biasa"), NOT the row's own column.
		if row.Shopee != 0 {
			fee, ongkirFee, err := feeCache.lookup(row.KategoriShopeeID, row.KategoriFreeOngkirID)
			if err != nil {
				return dto.OurProductOutput{}, err
			}

			hpp := domainservices.AverageHpp(row.HppAverage)
			marginNilai := calculator.ShopeeMarginNilai(
				row.Shopee,
				hpp,
				fee,
				ongkirFee,
				domainservices.FreeOngkirCapBiasa,
				row.KategoriShopeeID != nil && *row.KategoriShopeeID != 0,
			)
			margin := calculator.MarginPersen(marginNilai, hpp)
			out.MarginShopee = &margin
		}

		data = append(data, out)
	}

	return dto.OurProductOutput{
		Draw:            in.Draw,
		RecordsTotal:    total,
		RecordsFiltered: total,
		Data:            data,
	}, nil
}

// matchOurProducts runs the on-the-fly TF-IDF matching for one page of
// items and returns the matches, the effective batch per kompetitor, and
// the mapping resolver (reused to build each row's kompetitor columns).
func (s *KompetitorService) matchOurProducts(rows []repositories.OurProductRow) (
	map[uint64]map[uint64]domainservices.Match,
	map[uint64]domainservices.EffectiveBatch,
	*domainservices.MappingResolver,
	error,
) {
	resolver := domainservices.NewMappingResolver(s.mappings)
	empty := map[uint64]map[uint64]domainservices.Match{}

	items := make([]repositories.MappableItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, repositories.MappableItem{
			ID:           row.ItemID,
			NamaAccurate: row.NamaAccurate,
			BrandID:      row.BrandID,
			KategoriID:   row.KategoriID,
		})
	}

	latestBatch, err := s.scraping.LatestCompletedBatch()
	if err != nil {
		return nil, nil, nil, err
	}
	if latestBatch == nil || len(items) == 0 {
		// Still preload the cells so the caller can render empty columns.
		if _, err := resolver.AllowedByItem(items); err != nil {
			return nil, nil, nil, err
		}
		return empty, nil, resolver, nil
	}

	// Batches are read per kompetitor, not one global batch: a store that
	// has not been scraped in the latest batch keeps its last known price
	// (flagged stale in the UI), while a store that WAS scraped but whose
	// product was not found is deliberately left blank.
	batchRows, err := s.scraping.BatchRowsUpTo(latestBatch.ExecutedAt)
	if err != nil {
		return nil, nil, nil, err
	}
	batchByKompetitor := domainservices.ResolveBatchByKompetitor(batchRows, latestBatch.ID)

	allowedByItem, err := resolver.AllowedByItem(items)
	if err != nil {
		return nil, nil, nil, err
	}

	// Our own store must be matchable against every item regardless of the
	// (kategori, brand) mapping: harga tayang is our price, not a mapping.
	if dbklikID := s.cfg.DbklikKompetitorID; dbklikID != 0 {
		for _, item := range items {
			if allowedByItem[item.ID] == nil {
				allowedByItem[item.ID] = map[uint64]bool{}
			}
			allowedByItem[item.ID][dbklikID] = true
		}
	}

	pairs := repositories.BatchPair{}
	for _, allowed := range allowedByItem {
		for kompetitorID := range allowed {
			if batch, ok := batchByKompetitor[kompetitorID]; ok {
				pairs[kompetitorID] = batch.BatchID
			}
		}
	}
	if len(pairs) == 0 {
		return empty, batchByKompetitor, resolver, nil
	}

	products, err := s.scraping.ProductsForPairs(pairs)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(products) == 0 {
		return empty, batchByKompetitor, resolver, nil
	}

	idf, err := s.idfForBatches(batchIDsOf(pairs))
	if err != nil {
		return nil, nil, nil, err
	}

	matcher := domainservices.NewTextMatcher()
	matches := matcher.ComputeMatches(items, products, s.cfg.MatchThreshold, idf, allowedByItem)

	return matches, batchByKompetitor, resolver, nil
}

// idfForBatches builds (or reuses) the token weights of the full corpus:
// every product name in the batches actually read, plus every mappable item
// name. Tokens outside that corpus weigh 0, so a fallback batch must be
// part of it.
func (s *KompetitorService) idfForBatches(batchIDs []uint64) (map[string]float64, error) {
	return s.idf.Remember(batchIDs, func() (map[string]float64, error) {
		productNames, err := s.scraping.ProductNamesForBatches(batchIDs)
		if err != nil {
			return nil, err
		}

		mappable, err := s.items.Mappable()
		if err != nil {
			return nil, err
		}
		itemNames := make([]string, len(mappable))
		for i, item := range mappable {
			itemNames[i] = item.NamaAccurate
		}

		return domainservices.NewTextMatcher().ComputeIDF(productNames, itemNames), nil
	})
}

// feeCache memoizes the marketplace fee lookups for one request.
type feeCache struct {
	repo   repositories.MarketplaceFeeRepository
	shopee map[uint64]float64
	ongkir map[uint64]float64
}

func newFeeCache(repo repositories.MarketplaceFeeRepository) *feeCache {
	return &feeCache{
		repo:   repo,
		shopee: map[uint64]float64{},
		ongkir: map[uint64]float64{},
	}
}

func (c *feeCache) lookup(shopeeKategoriID, freeOngkirKategoriID *uint64) (float64, float64, error) {
	fee := 0.0
	if shopeeKategoriID != nil && *shopeeKategoriID != 0 {
		cached, ok := c.shopee[*shopeeKategoriID]
		if !ok {
			value, err := c.repo.ShopeeFee(*shopeeKategoriID)
			if err != nil {
				return 0, 0, err
			}
			c.shopee[*shopeeKategoriID] = value
			cached = value
		}
		fee = cached
	}

	ongkir := 0.0
	if freeOngkirKategoriID != nil && *freeOngkirKategoriID != 0 {
		cached, ok := c.ongkir[*freeOngkirKategoriID]
		if !ok {
			value, err := c.repo.FreeOngkirFee(*freeOngkirKategoriID)
			if err != nil {
				return 0, 0, err
			}
			c.ongkir[*freeOngkirKategoriID] = value
			cached = value
		}
		ongkir = cached
	}

	return fee, ongkir, nil
}

func batchIDsOf(pairs repositories.BatchPair) []uint64 {
	seen := map[uint64]bool{}
	ids := make([]uint64, 0, len(pairs))
	for _, batchID := range pairs {
		if seen[batchID] {
			continue
		}
		seen[batchID] = true
		ids = append(ids, batchID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func refIDs(refs []repositories.KompetitorRef) []uint64 {
	ids := make([]uint64, len(refs))
	for i, ref := range refs {
		ids[i] = ref.ID
	}
	return ids
}

func orDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
