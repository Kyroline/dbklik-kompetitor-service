// Package repository implements the kompetitor domain's repository
// interfaces against the shared *gorm.DB connection (the SAME MySQL
// database as the Laravel app — no data migration, just reads/writes on
// the same tables).
package repository

import (
	"errors"
	"time"

	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/entities"
	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/repositories"
	domainservices "github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/services"
	"gorm.io/gorm"
)

// ── Kompetitor ─────────────────────────────────────────────────────────

type KompetitorRepositoryGorm struct{ db *gorm.DB }

func NewKompetitorRepositoryGorm(db *gorm.DB) *KompetitorRepositoryGorm {
	return &KompetitorRepositoryGorm{db: db}
}

func (r *KompetitorRepositoryGorm) ListAll() ([]entities.Kompetitor, error) {
	var rows []entities.Kompetitor
	err := r.db.Order("name asc").Find(&rows).Error
	return rows, err
}

func (r *KompetitorRepositoryGorm) ListAllByPriority() ([]entities.Kompetitor, error) {
	var rows []entities.Kompetitor
	err := r.db.Order("must_scrape desc").Order("name asc").Find(&rows).Error
	return rows, err
}

func (r *KompetitorRepositoryGorm) ListScraped() ([]entities.Kompetitor, error) {
	var rows []entities.Kompetitor
	err := r.db.
		Where("id IN (?)", r.db.Model(&entities.ScrapingSummary{}).Select("kompetitor_id")).
		Order("name asc").
		Find(&rows).Error
	return rows, err
}

func (r *KompetitorRepositoryGorm) FindByID(id uint64) (*entities.Kompetitor, error) {
	var row entities.Kompetitor
	err := r.db.Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *KompetitorRepositoryGorm) Create(row *entities.Kompetitor) error {
	return r.db.Create(row).Error
}

func (r *KompetitorRepositoryGorm) Update(row *entities.Kompetitor) error {
	return r.db.Save(row).Error
}

// Delete removes the kompetitor together with its mapping rows, mirroring
// KompetitorController::destroy's `$kompetitor->mappings()->delete()`.
func (r *KompetitorRepositoryGorm) Delete(id uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("kompetitor_id = ?", id).Delete(&entities.KompetitorMapping{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", id).Delete(&entities.Kompetitor{}).Error
	})
}

func (r *KompetitorRepositoryGorm) CodeTaken(column, code string, exceptID uint64) (bool, error) {
	if column != "shopee_code" && column != "tokopedia_code" {
		return false, errors.New("repository: unsupported kompetitor code column " + column)
	}

	query := r.db.Model(&entities.Kompetitor{}).Where(column+" = ?", code)
	if exceptID != 0 {
		query = query.Where("id <> ?", exceptID)
	}

	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}

func (r *KompetitorRepositoryGorm) HasScrapingData(id uint64) (bool, error) {
	for _, model := range []interface{}{
		&entities.ScrapingSummary{},
		&entities.ScrapingProduct{},
		&entities.ScrapingProductMapping{},
	} {
		var count int64
		if err := r.db.Model(model).Where("kompetitor_id = ?", id).Limit(1).Count(&count).Error; err != nil {
			return false, err
		}
		if count > 0 {
			return true, nil
		}
	}
	return false, nil
}

func (r *KompetitorRepositoryGorm) MappingCounts() (map[uint64]int64, error) {
	var rows []struct {
		KompetitorID uint64 `gorm:"column:kompetitor_id"`
		Total        int64  `gorm:"column:total"`
	}

	err := r.db.Model(&entities.KompetitorMapping{}).
		Select("kompetitor_id, COUNT(*) as total").
		Group("kompetitor_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[uint64]int64, len(rows))
	for _, row := range rows {
		counts[row.KompetitorID] = row.Total
	}
	return counts, nil
}

// ── Mapping ────────────────────────────────────────────────────────────

type MappingRepositoryGorm struct{ db *gorm.DB }

func NewMappingRepositoryGorm(db *gorm.DB) *MappingRepositoryGorm {
	return &MappingRepositoryGorm{db: db}
}

// mappingJoinRow is the flat shape of a kompetitor_mappings ⋈ kompetitors row.
type mappingJoinRow struct {
	KategoriID    *uint64 `gorm:"column:kategori_id"`
	BrandID       uint64  `gorm:"column:brand_id"`
	ID            uint64  `gorm:"column:id"`
	Name          string  `gorm:"column:name"`
	ShopeeCode    *string `gorm:"column:shopee_code"`
	TokopediaCode *string `gorm:"column:tokopedia_code"`
}

func (r *MappingRepositoryGorm) PairRows(kategoriIDs, brandIDs []uint64) ([]repositories.MappingRow, error) {
	if len(kategoriIDs) == 0 || len(brandIDs) == 0 {
		return nil, nil
	}

	var rows []mappingJoinRow
	err := r.db.Table("kompetitor_mappings as km").
		Joins("JOIN kompetitors as k ON k.id = km.kompetitor_id").
		Where("km.kategori_id IS NOT NULL").
		Where("km.kategori_id IN ?", kategoriIDs).
		Where("km.brand_id IN ?", brandIDs).
		Order("k.name asc").
		Select("km.kategori_id, km.brand_id, k.id, k.name, k.shopee_code, k.tokopedia_code").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	return toMappingRows(rows), nil
}

func (r *MappingRepositoryGorm) UniversalRows(brandIDs []uint64) ([]repositories.MappingRow, error) {
	if len(brandIDs) == 0 {
		return nil, nil
	}

	var rows []mappingJoinRow
	err := r.db.Table("kompetitor_mappings as km").
		Joins("JOIN kompetitors as k ON k.id = km.kompetitor_id").
		Where("km.kategori_id IS NULL").
		Where("km.brand_id IN ?", brandIDs).
		Order("k.name asc").
		Select("km.kategori_id, km.brand_id, k.id, k.name, k.shopee_code, k.tokopedia_code").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	return toMappingRows(rows), nil
}

func toMappingRows(rows []mappingJoinRow) []repositories.MappingRow {
	out := make([]repositories.MappingRow, len(rows))
	for i, row := range rows {
		out[i] = repositories.MappingRow{
			KategoriID: row.KategoriID,
			BrandID:    row.BrandID,
			Kompetitor: repositories.KompetitorRef{
				ID:            row.ID,
				Name:          row.Name,
				ShopeeCode:    row.ShopeeCode,
				TokopediaCode: row.TokopediaCode,
			},
		}
	}
	return out
}

func (r *MappingRepositoryGorm) MatrixCounts(brandIDs, kategoriIDs []uint64) (map[string]int64, error) {
	query := r.db.Model(&entities.KompetitorMapping{}).
		Where("kategori_id IS NOT NULL").
		Select("kategori_id, brand_id, COUNT(*) as total").
		Group("kategori_id").Group("brand_id")

	if len(brandIDs) > 0 {
		query = query.Where("brand_id IN ?", brandIDs)
	}
	if len(kategoriIDs) > 0 {
		query = query.Where("kategori_id IN ?", kategoriIDs)
	}

	var rows []struct {
		KategoriID uint64 `gorm:"column:kategori_id"`
		BrandID    uint64 `gorm:"column:brand_id"`
		Total      int64  `gorm:"column:total"`
	}
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}

	counts := make(map[string]int64, len(rows))
	for _, row := range rows {
		counts[domainservices.CellKey(row.KategoriID, row.BrandID)] = row.Total
	}
	return counts, nil
}

func (r *MappingRepositoryGorm) UniversalCounts(brandIDs []uint64) (map[uint64]int64, error) {
	query := r.db.Model(&entities.KompetitorMapping{}).
		Where("kategori_id IS NULL").
		Select("brand_id, COUNT(*) as total").
		Group("brand_id")

	if len(brandIDs) > 0 {
		query = query.Where("brand_id IN ?", brandIDs)
	}

	var rows []struct {
		BrandID uint64 `gorm:"column:brand_id"`
		Total   int64  `gorm:"column:total"`
	}
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}

	counts := make(map[uint64]int64, len(rows))
	for _, row := range rows {
		counts[row.BrandID] = row.Total
	}
	return counts, nil
}

func (r *MappingRepositoryGorm) UsedAxes() ([]uint64, []uint64, error) {
	var brandIDs []uint64
	err := r.db.Model(&entities.KompetitorMapping{}).
		Distinct().
		Pluck("brand_id", &brandIDs).Error
	if err != nil {
		return nil, nil, err
	}

	var kategoriIDs []uint64
	err = r.db.Model(&entities.KompetitorMapping{}).
		Where("kategori_id IS NOT NULL").
		Distinct().
		Pluck("kategori_id", &kategoriIDs).Error
	if err != nil {
		return nil, nil, err
	}

	return brandIDs, kategoriIDs, nil
}

// SyncCell diffs the cell against what is stored and applies only the
// difference, so untouched rows keep their timestamps.
func (r *MappingRepositoryGorm) SyncCell(kategoriID *uint64, brandID uint64, kompetitorIDs []uint64) error {
	wanted := map[uint64]bool{}
	var ordered []uint64
	for _, id := range kompetitorIDs {
		if wanted[id] {
			continue
		}
		wanted[id] = true
		ordered = append(ordered, id)
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		scope := func() *gorm.DB {
			q := tx.Model(&entities.KompetitorMapping{}).Where("brand_id = ?", brandID)
			if kategoriID == nil {
				return q.Where("kategori_id IS NULL")
			}
			return q.Where("kategori_id = ?", *kategoriID)
		}

		var existing []uint64
		if err := scope().Pluck("kompetitor_id", &existing).Error; err != nil {
			return err
		}

		have := make(map[uint64]bool, len(existing))
		var toDelete []uint64
		for _, id := range existing {
			have[id] = true
			if !wanted[id] {
				toDelete = append(toDelete, id)
			}
		}

		if len(toDelete) > 0 {
			if err := scope().Where("kompetitor_id IN ?", toDelete).Delete(&entities.KompetitorMapping{}).Error; err != nil {
				return err
			}
		}

		now := time.Now()
		var toInsert []entities.KompetitorMapping
		for _, id := range ordered {
			if have[id] {
				continue
			}
			toInsert = append(toInsert, entities.KompetitorMapping{
				KategoriID:   kategoriID,
				BrandID:      brandID,
				KompetitorID: id,
				CreatedAt:    now,
				UpdatedAt:    now,
			})
		}

		if len(toInsert) == 0 {
			return nil
		}
		return tx.Create(&toInsert).Error
	})
}

func (r *MappingRepositoryGorm) KompetitorIDsExist(ids []uint64) (bool, error) {
	if len(ids) == 0 {
		return true, nil
	}

	unique := map[uint64]bool{}
	for _, id := range ids {
		unique[id] = true
	}

	var count int64
	err := r.db.Model(&entities.Kompetitor{}).Where("id IN ?", ids).Distinct("id").Count(&count).Error
	if err != nil {
		return false, err
	}

	return count == int64(len(unique)), nil
}

// ── Brand & Kategori (read-only reference tables) ──────────────────────

type BrandRepositoryGorm struct{ db *gorm.DB }

func NewBrandRepositoryGorm(db *gorm.DB) *BrandRepositoryGorm { return &BrandRepositoryGorm{db: db} }

func (r *BrandRepositoryGorm) ListActive(ids []uint64) ([]entities.Brand, error) {
	query := r.db.Where("status <> ?", "suspend").Order("nama_brand asc")
	if len(ids) > 0 {
		query = query.Where("id IN ?", ids)
	}

	var rows []entities.Brand
	err := query.Find(&rows).Error
	return rows, err
}

func (r *BrandRepositoryGorm) Exists(brandID uint64) (bool, error) {
	var count int64
	err := r.db.Model(&entities.Brand{}).Where("id = ?", brandID).Count(&count).Error
	return count > 0, err
}

type KategoriRepositoryGorm struct{ db *gorm.DB }

func NewKategoriRepositoryGorm(db *gorm.DB) *KategoriRepositoryGorm {
	return &KategoriRepositoryGorm{db: db}
}

func (r *KategoriRepositoryGorm) List(ids []uint64) ([]entities.Kategori, error) {
	query := r.db.Order("nama_kategori asc")
	if len(ids) > 0 {
		query = query.Where("id IN ?", ids)
	}

	var rows []entities.Kategori
	err := query.Find(&rows).Error
	return rows, err
}

func (r *KategoriRepositoryGorm) Exists(kategoriID uint64) (bool, error) {
	var count int64
	err := r.db.Model(&entities.Kategori{}).Where("id = ?", kategoriID).Count(&count).Error
	return count > 0, err
}

// ── Scraping ───────────────────────────────────────────────────────────

type ScrapingRepositoryGorm struct{ db *gorm.DB }

func NewScrapingRepositoryGorm(db *gorm.DB) *ScrapingRepositoryGorm {
	return &ScrapingRepositoryGorm{db: db}
}

// scopeToPairs limits a query to the resolved (kompetitor, batch) pairs.
// An empty set becomes 1 = 0 so an empty period never turns into "all data",
// mirroring ResolvesScrapingPeriods::scopeToBatchPairs.
func scopeToPairs(db *gorm.DB, query *gorm.DB, pairs repositories.BatchPair, table string) *gorm.DB {
	if len(pairs) == 0 {
		return query.Where("1 = 0")
	}

	condition := db.Session(&gorm.Session{NewDB: true})
	for kompetitorID, batchID := range pairs {
		condition = condition.Or(
			db.Session(&gorm.Session{NewDB: true}).
				Where(table+".kompetitor_id = ?", kompetitorID).
				Where(table+".scraping_batch_id = ?", batchID),
		)
	}

	return query.Where(condition)
}

func (r *ScrapingRepositoryGorm) BatchesByKompetitor() ([]repositories.BatchRow, error) {
	var rows []repositories.BatchRow
	err := r.db.Table("scraping_products as sp").
		Joins("JOIN scraping_batches as b ON b.id = sp.scraping_batch_id").
		Where("b.status = ?", "completed").
		Select("sp.kompetitor_id, b.id as batch_id, b.executed_at").
		Group("sp.kompetitor_id").Group("b.id").Group("b.executed_at").
		Order("b.executed_at desc").
		Scan(&rows).Error
	return rows, err
}

func (r *ScrapingRepositoryGorm) Summaries(pairs repositories.BatchPair, kompetitorID *uint64) ([]entities.ScrapingSummary, error) {
	query := scopeToPairs(r.db, r.db.Model(&entities.ScrapingSummary{}), pairs, "scraping_summaries")
	if kompetitorID != nil {
		query = query.Where("kompetitor_id = ?", *kompetitorID)
	}

	var rows []entities.ScrapingSummary
	err := query.Find(&rows).Error
	return rows, err
}

func (r *ScrapingRepositoryGorm) Products(pairs repositories.BatchPair, kompetitorID *uint64, search string, offset, limit int) ([]repositories.ProductRow, int64, error) {
	base := func() *gorm.DB {
		query := scopeToPairs(r.db, r.db.Model(&entities.ScrapingProduct{}), pairs, "scraping_products")
		if kompetitorID != nil {
			query = query.Where("scraping_products.kompetitor_id = ?", *kompetitorID)
		}
		if search != "" {
			query = query.Where("scraping_products.name LIKE ?", "%"+search+"%")
		}
		return query
	}

	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []repositories.ProductRow
	err := base().
		Joins("JOIN kompetitors ON kompetitors.id = scraping_products.kompetitor_id").
		Joins("JOIN scraping_batches ON scraping_batches.id = scraping_products.scraping_batch_id").
		Select("scraping_products.*, kompetitors.name as kompetitor_name, scraping_batches.code as batch_code").
		Order("scraping_products.revenue_monthly desc").
		Offset(offset).
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

func (r *ScrapingRepositoryGorm) ProductsByNames(pairs repositories.BatchPair, names []string) ([]repositories.MatchableProduct, error) {
	if len(names) == 0 {
		return nil, nil
	}

	query := scopeToPairs(r.db, r.db.Model(&entities.ScrapingProduct{}), pairs, "scraping_products").
		Where("scraping_products.name IN ?", names)

	var rows []repositories.MatchableProduct
	err := query.Select("id, kompetitor_id, name, price, sold_monthly").Scan(&rows).Error
	return rows, err
}

func (r *ScrapingRepositoryGorm) ProductsForPairs(pairs repositories.BatchPair) ([]repositories.MatchableProduct, error) {
	query := scopeToPairs(r.db, r.db.Model(&entities.ScrapingProduct{}), pairs, "scraping_products")

	var rows []repositories.MatchableProduct
	err := query.Select("id, kompetitor_id, name, price, sold_monthly").Scan(&rows).Error
	return rows, err
}

func (r *ScrapingRepositoryGorm) ProductsForBatch(batchID uint64) ([]repositories.MatchableProduct, error) {
	var rows []repositories.MatchableProduct
	err := r.db.Model(&entities.ScrapingProduct{}).
		Where("scraping_batch_id = ?", batchID).
		Select("id, kompetitor_id, name, price, sold_monthly").
		Scan(&rows).Error
	return rows, err
}

func (r *ScrapingRepositoryGorm) ProductNamesForBatches(batchIDs []uint64) ([]string, error) {
	if len(batchIDs) == 0 {
		return nil, nil
	}

	var names []string
	err := r.db.Model(&entities.ScrapingProduct{}).
		Where("scraping_batch_id IN ?", batchIDs).
		Pluck("name", &names).Error
	return names, err
}

func (r *ScrapingRepositoryGorm) LatestCompletedBatch() (*entities.ScrapingBatch, error) {
	var row entities.ScrapingBatch
	err := r.db.Where("status = ?", "completed").Order("executed_at desc").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *ScrapingRepositoryGorm) BatchRowsUpTo(to time.Time) ([]repositories.BatchRow, error) {
	var rows []repositories.BatchRow
	err := r.db.Table("scraping_products as sp").
		Joins("JOIN scraping_batches as sb ON sb.id = sp.scraping_batch_id").
		Where("sb.status = ?", "completed").
		Where("sb.executed_at <= ?", to).
		Distinct().
		Select("sp.kompetitor_id, sp.scraping_batch_id as batch_id, sb.executed_at").
		Order("sb.executed_at asc").
		Scan(&rows).Error
	return rows, err
}

func (r *ScrapingRepositoryGorm) FilterProducts(search, kompetitorName, batchCode string, offset, limit int) ([]repositories.LegacyProductRow, int64, error) {
	base := func() *gorm.DB {
		query := r.db.Model(&entities.ScrapingProduct{}).
			Joins("JOIN kompetitors ON kompetitors.id = scraping_products.kompetitor_id").
			Joins("JOIN scraping_batches ON scraping_batches.id = scraping_products.scraping_batch_id")

		if kompetitorName != "" {
			query = query.Where("kompetitors.name = ?", kompetitorName)
		}
		if batchCode != "" {
			query = query.Where("scraping_batches.code = ?", batchCode)
		}
		if search != "" {
			query = query.Where("scraping_products.name LIKE ?", "%"+search+"%")
		}
		return query
	}

	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []repositories.LegacyProductRow
	err := base().
		Select(`kompetitors.name as kompetitor,
			scraping_products.name as nama_produk,
			scraping_products.price as harga_raw,
			scraping_products.sold_monthly as terjual_per_bulan,
			scraping_products.revenue_monthly as pendapatan,
			scraping_products.rating,
			scraping_batches.code as batch_id`).
		Offset(offset).
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

func (r *ScrapingRepositoryGorm) CompletedBatchCodes() ([]string, error) {
	var codes []string
	err := r.db.Model(&entities.ScrapingBatch{}).
		Where("status = ?", "completed").
		Order("executed_at desc").
		Pluck("code", &codes).Error
	return codes, err
}

func (r *ScrapingRepositoryGorm) ItemIDsByProductIDs(productIDs []uint64) (map[uint64]uint64, error) {
	if len(productIDs) == 0 {
		return nil, nil
	}

	var rows []struct {
		ScrapingProductID uint64 `gorm:"column:scraping_product_id"`
		ItemID            uint64 `gorm:"column:item_id"`
	}
	err := r.db.Model(&entities.ScrapingProductMapping{}).
		Where("scraping_product_id IN ?", productIDs).
		Select("scraping_product_id, item_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[uint64]uint64, len(rows))
	for _, row := range rows {
		result[row.ScrapingProductID] = row.ItemID
	}
	return result, nil
}

// ── Ingest ─────────────────────────────────────────────────────────────

type IngestRepositoryGorm struct{ db *gorm.DB }

func NewIngestRepositoryGorm(db *gorm.DB) *IngestRepositoryGorm {
	return &IngestRepositoryGorm{db: db}
}

func (r *IngestRepositoryGorm) FirstOrCreateBatch(code string, executedAt time.Time) (*entities.ScrapingBatch, error) {
	var row entities.ScrapingBatch
	err := r.db.
		Where(entities.ScrapingBatch{Code: code}).
		Attrs(entities.ScrapingBatch{ExecutedAt: executedAt, Status: "running"}).
		FirstOrCreate(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *IngestRepositoryGorm) ReplaceProducts(batchID, kompetitorID uint64, products []repositories.IngestProduct) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		err := tx.Where("scraping_batch_id = ?", batchID).
			Where("kompetitor_id = ?", kompetitorID).
			Delete(&entities.ScrapingProduct{}).Error
		if err != nil {
			return err
		}

		now := time.Now()
		rows := make([]entities.ScrapingProduct, len(products))
		for i, product := range products {
			rows[i] = entities.ScrapingProduct{
				ScrapingBatchID: batchID,
				KompetitorID:    kompetitorID,
				Name:            product.Name,
				Price:           product.Price,
				SoldMonthly:     product.SoldMonthly,
				RevenueMonthly:  product.RevenueMonthly,
				SoldWeekly:      product.SoldWeekly,
				RevenueWeekly:   product.RevenueWeekly,
				Rating:          product.Rating,
				WishlistCount:   product.WishlistCount,
				Stock:           product.Stock,
				CreatedAt:       now,
				UpdatedAt:       now,
			}
		}

		return tx.CreateInBatches(&rows, 500).Error
	})
}

// ComputeSummaries rebuilds scraping_summaries for the batch from the raw
// products — a port of ScrapingBatch::computeSummaries().
func (r *IngestRepositoryGorm) ComputeSummaries(batchID uint64) error {
	var aggregates []struct {
		KompetitorID        uint64  `gorm:"column:kompetitor_id"`
		TotalRevenueMonthly float64 `gorm:"column:total_revenue_monthly"`
		TotalSoldMonthly    int64   `gorm:"column:total_sold_monthly"`
		AvgProductPrice     float64 `gorm:"column:avg_product_price"`
		ProductsInStock     int64   `gorm:"column:products_in_stock"`
		ProductsOutOfStock  int64   `gorm:"column:products_out_of_stock"`
	}

	err := r.db.Model(&entities.ScrapingProduct{}).
		Where("scraping_batch_id = ?", batchID).
		Select(`kompetitor_id,
			SUM(revenue_monthly) as total_revenue_monthly,
			SUM(sold_monthly) as total_sold_monthly,
			AVG(price) as avg_product_price,
			SUM(stock > 0) as products_in_stock,
			SUM(stock = 0) as products_out_of_stock`).
		Group("kompetitor_id").
		Scan(&aggregates).Error
	if err != nil {
		return err
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, agg := range aggregates {
			var existing entities.ScrapingSummary
			err := tx.Where("scraping_batch_id = ?", batchID).
				Where("kompetitor_id = ?", agg.KompetitorID).
				First(&existing).Error

			isNew := errors.Is(err, gorm.ErrRecordNotFound)
			if err != nil && !isNew {
				return err
			}

			existing.ScrapingBatchID = batchID
			existing.KompetitorID = agg.KompetitorID
			existing.TotalRevenueMonthly = agg.TotalRevenueMonthly
			existing.AvgRevenueWeekly = agg.TotalRevenueMonthly / 4
			existing.AvgRevenueDaily = agg.TotalRevenueMonthly / 30
			existing.TotalSoldMonthly = agg.TotalSoldMonthly
			existing.AvgSoldWeekly = float64(agg.TotalSoldMonthly) / 4
			existing.AvgSoldDaily = float64(agg.TotalSoldMonthly) / 30
			existing.AvgProductPrice = agg.AvgProductPrice
			existing.ProductsInStock = agg.ProductsInStock
			existing.ProductsOutOfStock = agg.ProductsOutOfStock

			now := time.Now()
			existing.UpdatedAt = now
			if isNew {
				existing.CreatedAt = now
				if err := tx.Create(&existing).Error; err != nil {
					return err
				}
				continue
			}

			if err := tx.Save(&existing).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *IngestRepositoryGorm) MarkStatus(batchID uint64, status string) error {
	return r.db.Model(&entities.ScrapingBatch{}).
		Where("id = ?", batchID).
		Updates(map[string]interface{}{"status": status, "updated_at": time.Now()}).Error
}

func (r *IngestRepositoryGorm) ReplaceProductMappings(batchID uint64, rows []entities.ScrapingProductMapping) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		err := tx.Where("scraping_batch_id = ?", batchID).
			Delete(&entities.ScrapingProductMapping{}).Error
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}

		now := time.Now()
		for i := range rows {
			rows[i].CreatedAt = now
			rows[i].UpdatedAt = now
		}

		return tx.CreateInBatches(&rows, 500).Error
	})
}

// ── Item ───────────────────────────────────────────────────────────────

type ItemRepositoryGorm struct{ db *gorm.DB }

func NewItemRepositoryGorm(db *gorm.DB) *ItemRepositoryGorm { return &ItemRepositoryGorm{db: db} }

// Mappable returns items whose (kategori_id, brand_id) pair has at least
// one SPECIFIC kompetitor mapping. Universal-only brands are excluded
// exactly as in PHP, where the NULL kategori_id never matches the
// whereColumn comparison.
func (r *ItemRepositoryGorm) Mappable() ([]repositories.MappableItem, error) {
	var rows []repositories.MappableItem
	err := r.db.Table("item").
		Where("item.deleted_at IS NULL").
		Where("item.is_bundling = ?", 0).
		Where(`EXISTS (
			SELECT 1 FROM kompetitor_mappings km
			WHERE km.kategori_id = item.kategori_id
			  AND km.brand_id = item.brand_id
		)`).
		Select("item.id, item.nama_accurate, item.brand_id, item.kategori_id").
		Scan(&rows).Error
	return rows, err
}

// ── Our Product (harga-driven table) ───────────────────────────────────

type OurProductRepositoryGorm struct{ db *gorm.DB }

func NewOurProductRepositoryGorm(db *gorm.DB) *OurProductRepositoryGorm {
	return &OurProductRepositoryGorm{db: db}
}

func (r *OurProductRepositoryGorm) Page(filters repositories.OurProductFilters, offset, limit int) ([]repositories.OurProductRow, int64, error) {
	base := func() *gorm.DB {
		query := r.db.Table("harga").
			Joins("JOIN item ON item.id = harga.item_id AND item.deleted_at IS NULL AND item.is_bundling = 0")

		if filters.Search != "" {
			like := "%" + filters.Search + "%"
			query = query.Where("(item.nama_accurate LIKE ? OR item.sku LIKE ?)", like, like)
		}
		if len(filters.Brand) > 0 {
			query = query.Where("item.brand_id IN (?)",
				r.db.Table("brand").Select("id").Where("nama_brand IN ?", filters.Brand))
		}
		if len(filters.Kategori) > 0 {
			query = query.Where("item.kategori_id IN (?)",
				r.db.Table("kategori").Select("id").Where("nama_kategori IN ?", filters.Kategori))
		}
		if len(filters.Abc) > 0 {
			// Match the ABC value actually displayed: abc_category_all, falling
			// back to abc_category when abc_category_all is NULL.
			query = query.Where(`EXISTS (
				SELECT 1 FROM abc_analysis a
				WHERE a.item_id = item.id
				  AND (a.abc_category_all IN (?)
				       OR (a.abc_category_all IS NULL AND a.abc_category IN (?)))
			)`, filters.Abc, filters.Abc)
		}

		return query
	}

	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []struct {
		HargaID              uint64  `gorm:"column:harga_id"`
		ItemID               uint64  `gorm:"column:item_id"`
		SKU                  string  `gorm:"column:sku"`
		NamaAccurate         string  `gorm:"column:nama_accurate"`
		BrandID              *uint64 `gorm:"column:brand_id"`
		KategoriID           *uint64 `gorm:"column:kategori_id"`
		BrandName            *string `gorm:"column:brand_name"`
		KategoriName         *string `gorm:"column:kategori_name"`
		Shopee               float64 `gorm:"column:shopee"`
		KategoriShopeeID     *uint64 `gorm:"column:kategori_shopee_id"`
		KategoriFreeOngkirID *uint64 `gorm:"column:kategori_free_ongkir_id"`
		FreeOngkirType       *string `gorm:"column:free_ongkir_type"`
		AbcCategory          *string `gorm:"column:abc_category"`
		AbcCategoryAll       *string `gorm:"column:abc_category_all"`
	}

	err := base().
		Joins("LEFT JOIN brand ON brand.id = item.brand_id").
		Joins("LEFT JOIN kategori ON kategori.id = item.kategori_id").
		Joins("LEFT JOIN abc_analysis ON abc_analysis.item_id = item.id").
		Select(`harga.id as harga_id, harga.item_id, harga.shopee,
			harga.kategori_shopee_id, harga.kategori_free_ongkir_id, harga.free_ongkir_type,
			item.sku, item.nama_accurate, item.brand_id, item.kategori_id,
			brand.nama_brand as brand_name, kategori.nama_kategori as kategori_name,
			abc_analysis.abc_category, abc_analysis.abc_category_all`).
		Order("harga.id asc").
		Offset(offset).
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	hargaIDs := make([]uint64, len(rows))
	for i, row := range rows {
		hargaIDs[i] = row.HargaID
	}

	averages, latest, err := r.hppByHargaID(hargaIDs)
	if err != nil {
		return nil, 0, err
	}

	out := make([]repositories.OurProductRow, len(rows))
	for i, row := range rows {
		abc := row.AbcCategoryAll
		if abc == nil {
			abc = row.AbcCategory
		}

		out[i] = repositories.OurProductRow{
			HargaID:              row.HargaID,
			ItemID:               row.ItemID,
			SKU:                  row.SKU,
			NamaAccurate:         row.NamaAccurate,
			BrandID:              row.BrandID,
			KategoriID:           row.KategoriID,
			BrandName:            deref(row.BrandName),
			KategoriName:         deref(row.KategoriName),
			Shopee:               row.Shopee,
			KategoriShopeeID:     row.KategoriShopeeID,
			KategoriFreeOngkirID: row.KategoriFreeOngkirID,
			FreeOngkirType:       deref(row.FreeOngkirType),
			Abc:                  abc,
			HppLatest:            latest[row.HargaID],
			HppAverage:           averages[row.HargaID],
		}
	}

	return out, total, nil
}

// hppByHargaID returns the average hpp (margin base, Harga::getAverageHpp)
// and the latest hpp row's value (the displayed hpp_latest, Harga::hppLatest
// = latestOfMany() on the primary key).
func (r *OurProductRepositoryGorm) hppByHargaID(hargaIDs []uint64) (map[uint64]float64, map[uint64]float64, error) {
	averages := map[uint64]float64{}
	latest := map[uint64]float64{}

	if len(hargaIDs) == 0 {
		return averages, latest, nil
	}

	var avgRows []struct {
		HargaID uint64  `gorm:"column:harga_id"`
		Average float64 `gorm:"column:average"`
	}
	err := r.db.Model(&entities.Hpp{}).
		Where("harga_id IN ?", hargaIDs).
		Select("harga_id, AVG(hpp) as average").
		Group("harga_id").
		Scan(&avgRows).Error
	if err != nil {
		return nil, nil, err
	}
	for _, row := range avgRows {
		averages[row.HargaID] = row.Average
	}

	var latestRows []struct {
		HargaID uint64  `gorm:"column:harga_id"`
		Hpp     float64 `gorm:"column:hpp"`
	}
	err = r.db.Table("hpp").
		Joins(`JOIN (
			SELECT harga_id, MAX(id) as latest_id
			FROM hpp WHERE harga_id IN ? GROUP BY harga_id
		) newest ON newest.latest_id = hpp.id`, hargaIDs).
		Select("hpp.harga_id, hpp.hpp").
		Scan(&latestRows).Error
	if err != nil {
		return nil, nil, err
	}
	for _, row := range latestRows {
		latest[row.HargaID] = row.Hpp
	}

	return averages, latest, nil
}

func (r *OurProductRepositoryGorm) FilterOptions() ([]string, []string, error) {
	var brands []string
	err := r.db.Model(&entities.Brand{}).
		Where("status <> ?", "suspend").
		Order("nama_brand asc").
		Pluck("nama_brand", &brands).Error
	if err != nil {
		return nil, nil, err
	}

	var kategoris []string
	err = r.db.Model(&entities.Kategori{}).
		Order("nama_kategori asc").
		Pluck("nama_kategori", &kategoris).Error
	if err != nil {
		return nil, nil, err
	}

	return brands, kategoris, nil
}

// ── Warehouse & marketplace fees ───────────────────────────────────────

type WarehouseRepositoryGorm struct{ db *gorm.DB }

func NewWarehouseRepositoryGorm(db *gorm.DB) *WarehouseRepositoryGorm {
	return &WarehouseRepositoryGorm{db: db}
}

func (r *WarehouseRepositoryGorm) TotalStockByItemIDs(itemIDs []uint64) (map[uint64]int64, error) {
	if len(itemIDs) == 0 {
		return nil, nil
	}

	var rows []entities.Warehouse
	if err := r.db.Where("item_id IN ?", itemIDs).Find(&rows).Error; err != nil {
		return nil, err
	}

	stock := make(map[uint64]int64, len(rows))
	for i := range rows {
		stock[rows[i].ItemID] = domainservices.TotalStock(&rows[i])
	}
	return stock, nil
}

type MarketplaceFeeRepositoryGorm struct{ db *gorm.DB }

func NewMarketplaceFeeRepositoryGorm(db *gorm.DB) *MarketplaceFeeRepositoryGorm {
	return &MarketplaceFeeRepositoryGorm{db: db}
}

func (r *MarketplaceFeeRepositoryGorm) ShopeeFee(id uint64) (float64, error) {
	var row entities.MarketplaceFeeShopee
	err := r.db.Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return row.BiayaLayanan, nil
}

func (r *MarketplaceFeeRepositoryGorm) FreeOngkirFee(id uint64) (float64, error) {
	var row entities.MarketplaceFeeGratisOngkir
	err := r.db.Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return row.Variabel, nil
}

func deref(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
