// Package entities defines the kompetitor module's GORM-mapped structs.
// These mirror the Laravel migrations 1:1 (same table/column names) since
// this module reads/writes the SAME shared MySQL database as the Laravel
// app portal_produk_dbklik — no data migration, no shadow tables.
package entities

import "time"

// ── Kompetitor & mapping (owned by this module) ────────────────────────

// Kompetitor mirrors table `kompetitors`.
type Kompetitor struct {
	ID            uint64    `gorm:"column:id;primaryKey"`
	Name          string    `gorm:"column:name"`
	ShopeeCode    *string   `gorm:"column:shopee_code"`
	TokopediaCode *string   `gorm:"column:tokopedia_code"`
	MustScrape    bool      `gorm:"column:must_scrape;default:true"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

func (Kompetitor) TableName() string { return "kompetitors" }

// KompetitorMapping mirrors table `kompetitor_mappings`: one row = one
// kompetitor monitored for the (kategori, brand) pair. A NULL kategori_id
// marks the brand's "universal" cell, used as fallback for any kategori
// that has no specific cell of its own.
type KompetitorMapping struct {
	ID           uint64    `gorm:"column:id;primaryKey"`
	KategoriID   *uint64   `gorm:"column:kategori_id"`
	BrandID      uint64    `gorm:"column:brand_id"`
	KompetitorID uint64    `gorm:"column:kompetitor_id"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (KompetitorMapping) TableName() string { return "kompetitor_mappings" }

// ── Scraping (owned by this module) ────────────────────────────────────

// ScrapingBatch mirrors table `scraping_batches`.
type ScrapingBatch struct {
	ID         uint64    `gorm:"column:id;primaryKey"`
	Code       string    `gorm:"column:code"`
	ExecutedAt time.Time `gorm:"column:executed_at"`
	Status     string    `gorm:"column:status;default:running"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

func (ScrapingBatch) TableName() string { return "scraping_batches" }

// ScrapingSummary mirrors table `scraping_summaries` — one aggregate row
// per (batch, kompetitor).
type ScrapingSummary struct {
	ID                  uint64    `gorm:"column:id;primaryKey"`
	ScrapingBatchID     uint64    `gorm:"column:scraping_batch_id"`
	KompetitorID        uint64    `gorm:"column:kompetitor_id"`
	TotalRevenueMonthly float64   `gorm:"column:total_revenue_monthly"`
	AvgRevenueWeekly    float64   `gorm:"column:avg_revenue_weekly"`
	AvgRevenueDaily     float64   `gorm:"column:avg_revenue_daily"`
	TotalSoldMonthly    int64     `gorm:"column:total_sold_monthly"`
	AvgSoldWeekly       float64   `gorm:"column:avg_sold_weekly"`
	AvgSoldDaily        float64   `gorm:"column:avg_sold_daily"`
	AvgProductPrice     float64   `gorm:"column:avg_product_price"`
	ProductsInStock     int64     `gorm:"column:products_in_stock"`
	ProductsOutOfStock  int64     `gorm:"column:products_out_of_stock"`
	CreatedAt           time.Time `gorm:"column:created_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at"`
}

func (ScrapingSummary) TableName() string { return "scraping_summaries" }

// ScrapingProduct mirrors table `scraping_products`.
type ScrapingProduct struct {
	ID              uint64    `gorm:"column:id;primaryKey"`
	ScrapingBatchID uint64    `gorm:"column:scraping_batch_id"`
	KompetitorID    uint64    `gorm:"column:kompetitor_id"`
	Name            string    `gorm:"column:name"`
	Price           float64   `gorm:"column:price"`
	SoldMonthly     int64     `gorm:"column:sold_monthly"`
	RevenueMonthly  float64   `gorm:"column:revenue_monthly"`
	SoldWeekly      int64     `gorm:"column:sold_weekly"`
	RevenueWeekly   float64   `gorm:"column:revenue_weekly"`
	Rating          *float64  `gorm:"column:rating"`
	WishlistCount   int64     `gorm:"column:wishlist_count"`
	Stock           int64     `gorm:"column:stock"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at"`
}

func (ScrapingProduct) TableName() string { return "scraping_products" }

// ScrapingProductMapping mirrors table `scraping_product_mappings` — the
// persisted best match per (batch × our item × kompetitor).
type ScrapingProductMapping struct {
	ID                uint64    `gorm:"column:id;primaryKey"`
	ScrapingBatchID   uint64    `gorm:"column:scraping_batch_id"`
	ScrapingProductID uint64    `gorm:"column:scraping_product_id"`
	KompetitorID      uint64    `gorm:"column:kompetitor_id"`
	ItemID            uint64    `gorm:"column:item_id"`
	Score             float64   `gorm:"column:score"`
	CreatedAt         time.Time `gorm:"column:created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at"`
}

func (ScrapingProductMapping) TableName() string { return "scraping_product_mappings" }

// ── Read-only reference tables (owned by the Laravel app) ──────────────

// Brand mirrors the reference table `brand`. `status` drives the Active
// scope (Brand::scopeActive: status != 'suspend').
type Brand struct {
	ID        uint64 `gorm:"column:id;primaryKey"`
	NamaBrand string `gorm:"column:nama_brand"`
	Status    string `gorm:"column:status"`
}

func (Brand) TableName() string { return "brand" }

// Kategori mirrors the reference table `kategori`.
type Kategori struct {
	ID           uint64 `gorm:"column:id;primaryKey"`
	NamaKategori string `gorm:"column:nama_kategori"`
}

func (Kategori) TableName() string { return "kategori" }

// Item mirrors the reference table `item` (only the columns this module
// reads). Soft-deleted rows and bundling items are excluded by the
// repository, mirroring Item's SoftDeletes + NonBundlingScope.
type Item struct {
	ID           uint64  `gorm:"column:id;primaryKey"`
	SKU          string  `gorm:"column:sku"`
	NamaAccurate string  `gorm:"column:nama_accurate"`
	BrandID      *uint64 `gorm:"column:brand_id"`
	KategoriID   *uint64 `gorm:"column:kategori_id"`
}

func (Item) TableName() string { return "item" }

// Harga mirrors the reference table `harga` (only the columns the Our
// Product panel reads).
type Harga struct {
	ID                   uint64  `gorm:"column:id;primaryKey"`
	ItemID               uint64  `gorm:"column:item_id"`
	Shopee               float64 `gorm:"column:shopee"`
	KategoriShopeeID     *uint64 `gorm:"column:kategori_shopee_id"`
	KategoriFreeOngkirID *uint64 `gorm:"column:kategori_free_ongkir_id"`
	FreeOngkirType       *string `gorm:"column:free_ongkir_type"`
}

func (Harga) TableName() string { return "harga" }

// Hpp mirrors the reference table `hpp`. The Our Product panel reads the
// average hpp (margin base) and the latest row's hpp (displayed value).
type Hpp struct {
	ID        uint64    `gorm:"column:id;primaryKey"`
	HargaID   uint64    `gorm:"column:harga_id"`
	Hpp       float64   `gorm:"column:hpp"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (Hpp) TableName() string { return "hpp" }

// AbcAnalysis mirrors the reference table `abc_analysis`.
type AbcAnalysis struct {
	ID             uint64  `gorm:"column:id;primaryKey"`
	ItemID         uint64  `gorm:"column:item_id"`
	AbcCategory    *string `gorm:"column:abc_category"`
	AbcCategoryAll *string `gorm:"column:abc_category_all"`
}

func (AbcAnalysis) TableName() string { return "abc_analysis" }

// MarketplaceFeeShopee mirrors the reference table `marketplace_fee_shopee`.
type MarketplaceFeeShopee struct {
	ID           uint64  `gorm:"column:id;primaryKey"`
	BiayaLayanan float64 `gorm:"column:biaya_layanan"`
}

func (MarketplaceFeeShopee) TableName() string { return "marketplace_fee_shopee" }

// MarketplaceFeeGratisOngkir mirrors `marketplace_fee_gratis_ongkirs`.
type MarketplaceFeeGratisOngkir struct {
	ID       uint64  `gorm:"column:id;primaryKey"`
	Variabel float64 `gorm:"column:variabel"`
}

func (MarketplaceFeeGratisOngkir) TableName() string { return "marketplace_fee_gratis_ongkirs" }

// Warehouse mirrors the reference table `warehouses`. Stock shown for our
// own products is the sum of every gudang_* column (see
// domain/services.TotalStock, a port of App\Helpers\StockCalculator).
type Warehouse struct {
	ID                     uint64 `gorm:"column:id;primaryKey"`
	ItemID                 uint64 `gorm:"column:item_id"`
	GudangAItc             int64  `gorm:"column:gudang_a_itc"`
	GudangAtTransitItc2    int64  `gorm:"column:gudang_at_transit_itc2"`
	GudangBJkt             int64  `gorm:"column:gudang_b_jkt"`
	GudangBtTransitJkt     int64  `gorm:"column:gudang_bt_transit_jkt"`
	GudangCKemuning        int64  `gorm:"column:gudang_c_kemuning"`
	GudangC6Lebak          int64  `gorm:"column:gudang_c6_lebak"`
	GudangCtTransitPusat   int64  `gorm:"column:gudang_ct_transit_pusat"`
	GudangDSmg             int64  `gorm:"column:gudang_d_smg"`
	GudangDtTransitSmg     int64  `gorm:"column:gudang_dt_transit_smg"`
	GudangEJog             int64  `gorm:"column:gudang_e_jog"`
	GudangEtTransitJog     int64  `gorm:"column:gudang_et_transit_jog"`
	GudangFMlg             int64  `gorm:"column:gudang_f_mlg"`
	GudangFtTransitMlg     int64  `gorm:"column:gudang_ft_transit_mlg"`
	GudangHBali            int64  `gorm:"column:gudang_h_bali"`
	GudangHtTransitBali    int64  `gorm:"column:gudang_ht_transit_bali"`
	GudangXSbyLama         int64  `gorm:"column:gudang_x_sby_lama"`
	GudangYSby             int64  `gorm:"column:gudang_y_sby"`
	GudangY3DisplayTenggil int64  `gorm:"column:gudang_y3_display_y_tenggilis"`
	GudangYtTransitYSby    int64  `gorm:"column:gudang_yt_transit_y_sby"`
}

func (Warehouse) TableName() string { return "warehouses" }
