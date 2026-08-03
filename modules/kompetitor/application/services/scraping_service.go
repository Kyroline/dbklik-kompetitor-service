package services

import (
	"sort"
	"strconv"
	"time"

	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/application/dto"
	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/repositories"
	domainservices "github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/services"
	apperrors "github.com/dbklik/dbklik-kompetitor-service/pkg/errors"
)

// Stats returns the Riset Produk headline metrics for the requested period
// and its comparison period — a port of KompetitorController::stats. A nil
// metrics block means "no data in that period", which the panel renders as
// an empty state rather than zeroes.
func (s *KompetitorService) Stats(in dto.PeriodInput) (dto.StatsOutput, error) {
	if err := validatePeriod(in); err != nil {
		return dto.StatsOutput{}, err
	}

	resolver, err := s.periodResolver()
	if err != nil {
		return dto.StatsOutput{}, err
	}

	currentPairs := resolver.ForPeriod(in.StartDate, in.EndDate)
	previousPairs := resolver.BeforePeriod(in.StartDate, in.EndDate)

	current, err := s.metricsFor(currentPairs, in.KompetitorID)
	if err != nil {
		return dto.StatsOutput{}, err
	}

	previous, err := s.metricsFor(previousPairs, in.KompetitorID)
	if err != nil {
		return dto.StatsOutput{}, err
	}

	return dto.StatsOutput{
		Current:         current,
		Previous:        previous,
		CurrentBatches:  batchIDs(currentPairs),
		PreviousBatches: batchIDs(previousPairs),
	}, nil
}

func (s *KompetitorService) metricsFor(pairs repositories.BatchPair, kompetitorID uint64) (*dto.StatsMetrics, error) {
	if len(pairs) == 0 {
		return nil, nil
	}

	summaries, err := s.scraping.Summaries(pairs, optionalID(kompetitorID))
	if err != nil {
		return nil, err
	}
	if len(summaries) == 0 {
		return nil, nil
	}

	metrics := dto.StatsMetrics{}
	priceSum := 0.0

	for _, row := range summaries {
		metrics.TotalOmzet += row.TotalRevenueMonthly
		metrics.RataOmzetMinggu += row.AvgRevenueWeekly
		metrics.RataOmzetHari += row.AvgRevenueDaily
		metrics.TotalTerjual += row.TotalSoldMonthly
		metrics.RataTerjualMingu += row.AvgSoldWeekly
		metrics.RataTerjualHari += row.AvgSoldDaily
		metrics.ProdukReady += row.ProductsInStock
		metrics.ProdukHabis += row.ProductsOutOfStock
		priceSum += row.AvgProductPrice
	}

	// PHP casts the average to int, i.e. truncates.
	metrics.RataHarga = int64(priceSum / float64(len(summaries)))

	return &metrics, nil
}

// Products returns one page of the Riset Produk table — a port of
// KompetitorController::dataNew, including the price-change trend against
// the comparison period and our own stock for DB Klik rows.
func (s *KompetitorService) Products(in dto.ProductsInput) (dto.ProductsOutput, error) {
	if err := validatePeriod(in.PeriodInput); err != nil {
		return dto.ProductsOutput{}, err
	}

	resolver, err := s.periodResolver()
	if err != nil {
		return dto.ProductsOutput{}, err
	}

	currentPairs := resolver.ForPeriod(in.StartDate, in.EndDate)
	previousPairs := resolver.BeforePeriod(in.StartDate, in.EndDate)

	length := in.Length
	if length <= 0 {
		length = 25
	}

	rows, total, err := s.scraping.Products(currentPairs, optionalID(in.KompetitorID), in.Search, in.Start, length)
	if err != nil {
		return dto.ProductsOutput{}, err
	}

	prevByKey, err := s.previousPrices(rows, previousPairs)
	if err != nil {
		return dto.ProductsOutput{}, err
	}

	stockByProduct, err := s.dbklikStock(rows)
	if err != nil {
		return dto.ProductsOutput{}, err
	}

	dbklikID := s.cfg.DbklikKompetitorID

	data := make([]dto.ProductRow, len(rows))
	for i, row := range rows {
		out := dto.ProductRow{
			Kompetitor:      row.KompetitorName,
			NamaProduk:      row.Name,
			Harga:           formatRupiah(row.Price),
			TerjualPerBulan: row.SoldMonthly,
			Pendapatan:      formatRupiah(row.RevenueMonthly),
			RataHari:        formatRupiah(row.RevenueMonthly / 30),
			Rating:          row.Rating,
			BatchID:         row.BatchCode,
		}

		if prev, ok := prevByKey[productKey(row.KompetitorID, row.Name)]; ok && prev > 0 {
			change := roundTo((row.Price-prev)/prev*100, 2)
			out.PerubahanHarga = &change
		}

		// Stock is only shown for our own store's rows, read from `warehouses`.
		if dbklikID != 0 && row.KompetitorID == dbklikID {
			stock := stockByProduct[row.ID]
			out.Stok = &stock
		}

		data[i] = out
	}

	return dto.ProductsOutput{
		Draw:            in.Draw,
		RecordsTotal:    total,
		RecordsFiltered: total,
		Data:            data,
	}, nil
}

// previousPrices matches this page's products by name (per kompetitor) in
// the comparison period, to compute the price-change trend.
func (s *KompetitorService) previousPrices(rows []repositories.ProductRow, previousPairs repositories.BatchPair) (map[string]float64, error) {
	if len(rows) == 0 || len(previousPairs) == 0 {
		return nil, nil
	}

	// Only stores present on this page take part in the comparison.
	comparable := repositories.BatchPair{}
	names := make([]string, 0, len(rows))
	seenName := map[string]bool{}

	for _, row := range rows {
		if batchID, ok := previousPairs[row.KompetitorID]; ok {
			comparable[row.KompetitorID] = batchID
		}
		if !seenName[row.Name] {
			seenName[row.Name] = true
			names = append(names, row.Name)
		}
	}

	if len(comparable) == 0 {
		return nil, nil
	}

	products, err := s.scraping.ProductsByNames(comparable, names)
	if err != nil {
		return nil, err
	}

	prices := make(map[string]float64, len(products))
	for _, product := range products {
		prices[productKey(product.KompetitorID, product.Name)] = product.Price
	}

	return prices, nil
}

// dbklikStock resolves our own stock for the DB Klik rows on this page,
// via scraping_product_mappings → item → warehouses.
func (s *KompetitorService) dbklikStock(rows []repositories.ProductRow) (map[uint64]int64, error) {
	dbklikID := s.cfg.DbklikKompetitorID
	if dbklikID == 0 {
		return nil, nil
	}

	var productIDs []uint64
	for _, row := range rows {
		if row.KompetitorID == dbklikID {
			productIDs = append(productIDs, row.ID)
		}
	}
	if len(productIDs) == 0 {
		return nil, nil
	}

	itemByProduct, err := s.scraping.ItemIDsByProductIDs(productIDs)
	if err != nil {
		return nil, err
	}
	if len(itemByProduct) == 0 {
		return nil, nil
	}

	itemIDs := make([]uint64, 0, len(itemByProduct))
	for _, itemID := range itemByProduct {
		itemIDs = append(itemIDs, itemID)
	}

	stockByItem, err := s.warehouses.TotalStockByItemIDs(itemIDs)
	if err != nil {
		return nil, err
	}

	stockByProduct := make(map[uint64]int64, len(itemByProduct))
	for productID, itemID := range itemByProduct {
		stockByProduct[productID] = stockByItem[itemID]
	}

	return stockByProduct, nil
}

// LegacyProducts is the older product table filtered by kompetitor name
// and batch code — a port of KompetitorController::data →
// CompetitorService::filterCompetitorProduct.
func (s *KompetitorService) LegacyProducts(in dto.LegacyProductsInput) (dto.LegacyProductsOutput, error) {
	length := in.Length
	if length <= 0 {
		length = 25
	}

	rows, total, err := s.scraping.FilterProducts(in.Search, in.Kompetitor, in.Batch, in.Start, length)
	if err != nil {
		return dto.LegacyProductsOutput{}, err
	}

	data := make([]dto.LegacyProductRow, len(rows))
	for i, row := range rows {
		data[i] = dto.LegacyProductRow{
			Kompetitor:      row.Kompetitor,
			NamaProduk:      row.NamaProduk,
			Harga:           formatRupiah(row.Price),
			HargaRaw:        row.Price,
			TerjualPerBulan: row.SoldMonthly,
			Pendapatan:      formatRupiah(row.RevenueMonthly),
			RataHari:        formatRupiah(row.RevenueMonthly / 30),
			Rating:          row.Rating,
			BatchID:         row.BatchCode,
		}
	}

	return dto.LegacyProductsOutput{
		Draw:            in.Draw,
		RecordsTotal:    total,
		RecordsFiltered: total,
		Data:            data,
	}, nil
}

// BatchCodes lists completed batch codes, newest first — the batch filter
// of the legacy table (CompetitorService::getUniqueCompetitorAndBatch).
func (s *KompetitorService) BatchCodes() ([]string, error) {
	return s.scraping.CompletedBatchCodes()
}

// validatePeriod mirrors KompetitorController::periodOf's rules:
// both dates optional, format Y-m-d, end_date not before start_date.
func validatePeriod(in dto.PeriodInput) error {
	const layout = "2006-01-02"

	if in.StartDate != "" {
		if _, err := time.Parse(layout, in.StartDate); err != nil {
			return apperrors.InvalidInput("start_date harus berformat YYYY-MM-DD.")
		}
	}
	if in.EndDate != "" {
		if _, err := time.Parse(layout, in.EndDate); err != nil {
			return apperrors.InvalidInput("end_date harus berformat YYYY-MM-DD.")
		}
	}
	if in.StartDate != "" && in.EndDate != "" && in.EndDate < in.StartDate {
		return apperrors.InvalidInput("end_date harus setelah atau sama dengan start_date.")
	}

	return nil
}

// periodResolver builds a fresh resolver from the (kompetitor, batch) index.
func (s *KompetitorService) periodResolver() (*domainservices.PeriodResolver, error) {
	rows, err := s.scraping.BatchesByKompetitor()
	if err != nil {
		return nil, err
	}
	return domainservices.NewPeriodResolver(rows), nil
}

// batchIDs flattens a resolved pair set into a batch-id list, ordered by
// kompetitor id so the response is stable across calls.
func batchIDs(pairs repositories.BatchPair) []uint64 {
	kompetitorIDs := make([]uint64, 0, len(pairs))
	for id := range pairs {
		kompetitorIDs = append(kompetitorIDs, id)
	}
	sort.Slice(kompetitorIDs, func(i, j int) bool { return kompetitorIDs[i] < kompetitorIDs[j] })

	ids := make([]uint64, 0, len(pairs))
	for _, id := range kompetitorIDs {
		ids = append(ids, pairs[id])
	}
	return ids
}

func optionalID(id uint64) *uint64 {
	if id == 0 {
		return nil
	}
	return &id
}

func productKey(kompetitorID uint64, name string) string {
	return strconv.FormatUint(kompetitorID, 10) + "|" + name
}
