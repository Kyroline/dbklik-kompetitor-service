// Package repositories declares the kompetitor domain's persistence
// contracts. Only interfaces and their read-model row types live here;
// implementations belong in the infrastructure/repository layer.
package repositories

import (
	"time"

	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/entities"
)

// ── Read models ────────────────────────────────────────────────────────

// KompetitorRef is the slim kompetitor shape carried by mapping lookups
// (mirrors CompetitorMappingService::hydrateKompetitor()).
type KompetitorRef struct {
	ID            uint64
	Name          string
	ShopeeCode    *string
	TokopediaCode *string
}

// MappingRow is one kompetitor_mappings row joined to its kompetitor.
// KategoriID nil = the brand's universal cell.
type MappingRow struct {
	KategoriID *uint64
	BrandID    uint64
	Kompetitor KompetitorRef
}

// BatchRow is one (kompetitor, batch) pair that actually has products.
type BatchRow struct {
	KompetitorID uint64    `gorm:"column:kompetitor_id"`
	BatchID      uint64    `gorm:"column:batch_id"`
	ExecutedAt   time.Time `gorm:"column:executed_at"`
}

// BatchPair maps kompetitor_id => scraping_batch_id, the resolved
// "which batch do we read for this store" set (see PeriodResolver).
type BatchPair map[uint64]uint64

// ProductRow is one scraping product joined to its kompetitor and batch,
// used by the Riset Produk table (KompetitorController::dataNew).
type ProductRow struct {
	entities.ScrapingProduct `gorm:"embedded"`
	KompetitorName           string `gorm:"column:kompetitor_name"`
	BatchCode                string `gorm:"column:batch_code"`
}

// LegacyProductRow is the flat row shape of the older, batch-code-filtered
// product table (CompetitorService::filterCompetitorProduct).
type LegacyProductRow struct {
	Kompetitor     string   `gorm:"column:kompetitor"`
	NamaProduk     string   `gorm:"column:nama_produk"`
	Price          float64  `gorm:"column:harga_raw"`
	SoldMonthly    int64    `gorm:"column:terjual_per_bulan"`
	RevenueMonthly float64  `gorm:"column:pendapatan"`
	Rating         *float64 `gorm:"column:rating"`
	BatchCode      string   `gorm:"column:batch_id"`
}

// MatchableProduct is the minimal product shape fed to the TF-IDF matcher.
type MatchableProduct struct {
	ID           uint64  `gorm:"column:id"`
	KompetitorID uint64  `gorm:"column:kompetitor_id"`
	Name         string  `gorm:"column:name"`
	Price        float64 `gorm:"column:price"`
	SoldMonthly  int64   `gorm:"column:sold_monthly"`
}

// MappableItem is one of our items whose (kategori, brand) pair has a
// kompetitor mapping — the item side of the IDF corpus.
type MappableItem struct {
	ID           uint64  `gorm:"column:id"`
	NamaAccurate string  `gorm:"column:nama_accurate"`
	BrandID      *uint64 `gorm:"column:brand_id"`
	KategoriID   *uint64 `gorm:"column:kategori_id"`
}

// OurProductFilters mirrors the query params of
// KompetitorController::ourProductData.
type OurProductFilters struct {
	Search   string
	Brand    []string // brand names
	Kategori []string // kategori names
	Abc      []string // A..F, matched against abc_category_all w/ fallback
}

// OurProductRow is one `harga` row joined to everything the Our Product
// table displays.
type OurProductRow struct {
	HargaID              uint64
	ItemID               uint64
	SKU                  string
	NamaAccurate         string
	BrandID              *uint64
	KategoriID           *uint64
	BrandName            string
	KategoriName         string
	Shopee               float64
	KategoriShopeeID     *uint64
	KategoriFreeOngkirID *uint64
	FreeOngkirType       string
	Abc                  *string
	HppLatest            float64
	HppAverage           float64
}

// IngestProduct is one already-parsed scraping row handed over by the
// Laravel app (which still owns Excel parsing).
type IngestProduct struct {
	Name           string
	Price          float64
	SoldMonthly    int64
	RevenueMonthly float64
	SoldWeekly     int64
	RevenueWeekly  float64
	Rating         *float64
	WishlistCount  int64
	Stock          int64
}

// ── Contracts ──────────────────────────────────────────────────────────

// KompetitorRepository persists `kompetitors`.
type KompetitorRepository interface {
	ListAll() ([]entities.Kompetitor, error)
	// ListAllByPriority orders must_scrape stores first, then by name — the
	// order the Riset Produk page's "semua kompetitor" list expects.
	ListAllByPriority() ([]entities.Kompetitor, error)
	// ListScraped returns only kompetitors that appear in scraping_summaries,
	// ordered by name (the Riset Produk dropdown).
	ListScraped() ([]entities.Kompetitor, error)
	FindByID(id uint64) (*entities.Kompetitor, error)
	Create(row *entities.Kompetitor) error
	Update(row *entities.Kompetitor) error
	Delete(id uint64) error
	// CodeTaken reports whether another kompetitor already uses the given
	// shopee_code/tokopedia_code (column is "shopee_code" or "tokopedia_code").
	CodeTaken(column, code string, exceptID uint64) (bool, error)
	// HasScrapingData guards deletion: summaries, products or product mappings.
	HasScrapingData(id uint64) (bool, error)
	// MappingCounts returns the number of mapped cells per kompetitor_id.
	MappingCounts() (map[uint64]int64, error)
}

// MappingRepository persists `kompetitor_mappings`.
type MappingRepository interface {
	// PairRows returns specific (kategori, brand) cells for the cartesian
	// product of the given ids; callers filter down to the pairs they asked for.
	PairRows(kategoriIDs, brandIDs []uint64) ([]MappingRow, error)
	// UniversalRows returns the kategori_id IS NULL cells of the given brands.
	UniversalRows(brandIDs []uint64) ([]MappingRow, error)
	// MatrixCounts returns "kategoriID|brandID" => kompetitor count for the
	// specific cells only. Empty id slices mean "no filter".
	MatrixCounts(brandIDs, kategoriIDs []uint64) (map[string]int64, error)
	// UniversalCounts returns brandID => kompetitor count of universal cells.
	UniversalCounts(brandIDs []uint64) (map[uint64]int64, error)
	// UsedAxes returns brand ids and kategori ids that already have at least
	// one mapping — the default axes of the matrix panel.
	UsedAxes() (brandIDs []uint64, kategoriIDs []uint64, err error)
	// SyncCell replaces one cell's kompetitor list. Nil kategoriID = the
	// brand's universal cell; an empty list deletes the cell.
	SyncCell(kategoriID *uint64, brandID uint64, kompetitorIDs []uint64) error
	// KompetitorIDsExist reports whether every given id exists in `kompetitors`.
	KompetitorIDsExist(ids []uint64) (bool, error)
}

// BrandRepository reads the shared, read-only `brand` reference table.
type BrandRepository interface {
	// ListActive returns brands with status != 'suspend', ordered by name.
	// A non-empty ids slice restricts the result to those ids.
	ListActive(ids []uint64) ([]entities.Brand, error)
	Exists(brandID uint64) (bool, error)
}

// KategoriRepository reads the shared, read-only `kategori` reference table.
type KategoriRepository interface {
	// List returns kategoris ordered by name; a non-empty ids slice restricts
	// the result to those ids.
	List(ids []uint64) ([]entities.Kategori, error)
	Exists(kategoriID uint64) (bool, error)
}

// ScrapingRepository reads the scraping tables.
type ScrapingRepository interface {
	// BatchesByKompetitor returns every (kompetitor, completed batch) pair
	// that has products, newest batch first.
	BatchesByKompetitor() ([]BatchRow, error)
	// Summaries returns scraping_summaries scoped to the resolved batch pairs,
	// optionally narrowed to one kompetitor.
	Summaries(pairs BatchPair, kompetitorID *uint64) ([]entities.ScrapingSummary, error)
	// Products returns one page of scraping products scoped to the resolved
	// batch pairs, ordered by revenue_monthly desc, plus the unpaged total.
	Products(pairs BatchPair, kompetitorID *uint64, search string, offset, limit int) ([]ProductRow, int64, error)
	// ProductsByNames returns products with the given names, scoped to the
	// resolved batch pairs — the previous-period price comparison.
	ProductsByNames(pairs BatchPair, names []string) ([]MatchableProduct, error)
	// ProductsForPairs returns every product of the resolved batch pairs,
	// used as the matching corpus of the Our Product panel.
	ProductsForPairs(pairs BatchPair) ([]MatchableProduct, error)
	// ProductsForBatch returns every product of one batch, the corpus of the
	// persisted mapping pass that runs after ingest.
	ProductsForBatch(batchID uint64) ([]MatchableProduct, error)
	// ProductNamesForBatches returns every product name of the given batches
	// (the product side of the IDF corpus).
	ProductNamesForBatches(batchIDs []uint64) ([]string, error)
	// LatestCompletedBatch returns the newest completed batch, or nil.
	LatestCompletedBatch() (*entities.ScrapingBatch, error)
	// BatchRowsBetween returns (kompetitor, batch) pairs of completed batches
	// executed within [from, to], oldest first.
	BatchRowsBetween(from, to time.Time) ([]BatchRow, error)
	// FilterProducts is the legacy batch-code/kompetitor-name filtered table.
	FilterProducts(search, kompetitorName, batchCode string, offset, limit int) ([]LegacyProductRow, int64, error)
	// CompletedBatchCodes returns completed batch codes, newest first.
	CompletedBatchCodes() ([]string, error)
	// ItemIDsByProductIDs returns scraping_product_id => item_id from the
	// persisted mappings (used to show our own stock in the Riset Produk table).
	ItemIDsByProductIDs(productIDs []uint64) (map[uint64]uint64, error)
}

// IngestRepository writes scraping data parsed elsewhere (the Laravel app
// still owns Excel parsing and hands over plain JSON rows).
type IngestRepository interface {
	// FirstOrCreateBatch returns the batch with this code, creating it with
	// status "running" when absent.
	FirstOrCreateBatch(code string, executedAt time.Time) (*entities.ScrapingBatch, error)
	// ReplaceProducts deletes this kompetitor's rows in the batch and inserts
	// the given ones, in a single transaction.
	ReplaceProducts(batchID, kompetitorID uint64, products []IngestProduct) error
	// ComputeSummaries recomputes scraping_summaries for every kompetitor in
	// the batch from the raw products (ScrapingBatch::computeSummaries).
	ComputeSummaries(batchID uint64) error
	// MarkStatus sets the batch status ("completed"/"failed").
	MarkStatus(batchID uint64, status string) error
	// ReplaceProductMappings swaps the batch's persisted TF-IDF matches.
	ReplaceProductMappings(batchID uint64, rows []entities.ScrapingProductMapping) error
}

// ItemRepository reads our own catalogue.
type ItemRepository interface {
	// Mappable returns items whose (kategori_id, brand_id) pair has at least
	// one specific kompetitor mapping — the item side of the IDF corpus.
	Mappable() ([]MappableItem, error)
}

// OurProductRepository reads the `harga`-driven Our Product table.
type OurProductRepository interface {
	Page(filters OurProductFilters, offset, limit int) ([]OurProductRow, int64, error)
	// FilterOptions returns active brand names and every kategori name.
	FilterOptions() (brands []string, kategoris []string, err error)
}

// WarehouseRepository reads our own stock.
type WarehouseRepository interface {
	// TotalStockByItemIDs returns item_id => summed stock across all gudang columns.
	TotalStockByItemIDs(itemIDs []uint64) (map[uint64]int64, error)
}

// MarketplaceFeeRepository reads the Shopee fee reference tables used by
// the margin calculation.
type MarketplaceFeeRepository interface {
	// ShopeeFee returns marketplace_fee_shopee.biaya_layanan (0 when absent).
	ShopeeFee(id uint64) (float64, error)
	// FreeOngkirFee returns marketplace_fee_gratis_ongkirs.variabel (0 when absent).
	FreeOngkirFee(id uint64) (float64, error)
}
