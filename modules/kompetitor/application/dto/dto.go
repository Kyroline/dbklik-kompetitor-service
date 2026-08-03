// Package dto holds the application layer's input/output shapes. Field
// names mirror the JSON bodies the Laravel KompetitorController used to
// return, so this service is a drop-in replacement for those endpoints.
package dto

// ── Shared ─────────────────────────────────────────────────────────────

// NamedRow is an {id, name} reference row (brand / kategori / kompetitor).
type NamedRow struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

// ── Kompetitor CRUD (KompetitorController::manageData/store/update/destroy)

type KompetitorRow struct {
	ID            uint64 `json:"id"`
	Name          string `json:"name"`
	ShopeeCode    string `json:"shopee_code"`
	TokopediaCode string `json:"tokopedia_code"`
	MustScrape    bool   `json:"must_scrape"`
	MappingCount  int64  `json:"mapping_count"`
}

type ManageDataOutput struct {
	Data []KompetitorRow `json:"data"`
}

// SaveKompetitorInput is validated exactly like
// KompetitorController::validateKompetitor: name required, at least one of
// shopee_code/tokopedia_code, both unique across kompetitors.
type SaveKompetitorInput struct {
	ID            uint64 // 0 on create
	Name          string
	ShopeeCode    string
	TokopediaCode string
	MustScrape    bool
}

// IndexMetaOutput replaces the view data of KompetitorController::indexNew
// (the Blade view itself stays in the Laravel app).
type IndexMetaOutput struct {
	ListKompetitor     []NamedRow          `json:"list_kompetitor"`
	ListAllKompetitor  []KompetitorMetaRow `json:"list_all_kompetitor"`
	DbklikKompetitorID uint64              `json:"dbklik_kompetitor_id"`
	ListBrand          []NamedRow          `json:"list_brand"`
	ListKategori       []NamedRow          `json:"list_kategori"`
}

type KompetitorMetaRow struct {
	ID         uint64 `json:"id"`
	Name       string `json:"name"`
	MustScrape bool   `json:"must_scrape"`
}

// ── Mapping panel (matrix kategori × brand) ────────────────────────────

type MatrixInput struct {
	BrandIDs    []uint64
	KategoriIDs []uint64
}

type MatrixOutput struct {
	Brands          []NamedRow       `json:"brands"`
	Kategoris       []NamedRow       `json:"kategoris"`
	Counts          map[string]int64 `json:"counts"`
	UniversalCounts map[uint64]int64 `json:"universal_counts"`
	Filtered        bool             `json:"filtered"`
}

// CellInput addresses one cell. A nil KategoriID is the brand's universal cell.
type CellInput struct {
	KategoriID *uint64
	BrandID    uint64
}

type CellOutput struct {
	KompetitorIDs []uint64 `json:"kompetitor_ids"`
}

type CellUpdateInput struct {
	KategoriID    *uint64
	BrandID       uint64
	KompetitorIDs []uint64
}

// ── Stats & product tables ─────────────────────────────────────────────

// PeriodInput is the shared date filter; both bounds empty = "latest data".
type PeriodInput struct {
	StartDate string // YYYY-MM-DD
	EndDate   string
	// KompetitorID 0 = every kompetitor.
	KompetitorID uint64
}

type StatsMetrics struct {
	TotalOmzet       float64 `json:"total_omzet"`
	RataOmzetMinggu  float64 `json:"rata_omzet_minggu"`
	RataOmzetHari    float64 `json:"rata_omzet_hari"`
	TotalTerjual     int64   `json:"total_terjual"`
	RataTerjualMingu float64 `json:"rata_terjual_minggu"`
	RataTerjualHari  float64 `json:"rata_terjual_hari"`
	RataHarga        int64   `json:"rata_harga"`
	ProdukReady      int64   `json:"produk_ready"`
	ProdukHabis      int64   `json:"produk_habis"`
}

type StatsOutput struct {
	Current         *StatsMetrics `json:"current"`
	Previous        *StatsMetrics `json:"previous"`
	CurrentBatches  []uint64      `json:"current_batches"`
	PreviousBatches []uint64      `json:"previous_batches"`
}

type ProductsInput struct {
	PeriodInput
	Search string
	Start  int
	Length int
	Draw   int
}

type ProductRow struct {
	Kompetitor      string   `json:"kompetitor"`
	NamaProduk      string   `json:"nama_produk"`
	Harga           string   `json:"harga"`
	PerubahanHarga  *float64 `json:"perubahan_harga"`
	TerjualPerBulan int64    `json:"terjual_per_bulan"`
	Pendapatan      string   `json:"pendapatan"`
	RataHari        string   `json:"rata_hari"`
	Rating          *float64 `json:"rating"`
	Stok            *int64   `json:"stok"`
	BatchID         string   `json:"batch_id"`
}

type ProductsOutput struct {
	Draw            int          `json:"draw"`
	RecordsTotal    int64        `json:"recordsTotal"`
	RecordsFiltered int64        `json:"recordsFiltered"`
	Data            []ProductRow `json:"data"`
}

// LegacyProductsInput drives the older batch-code filtered table
// (KompetitorController::data → CompetitorService::filterCompetitorProduct).
type LegacyProductsInput struct {
	Search     string
	Kompetitor string // kompetitor NAME, not id
	Batch      string // batch CODE, not id
	Start      int
	Length     int
	Draw       int
}

type LegacyProductRow struct {
	Kompetitor      string   `json:"kompetitor"`
	NamaProduk      string   `json:"nama_produk"`
	Harga           string   `json:"harga"`
	HargaRaw        float64  `json:"harga_raw"`
	TerjualPerBulan int64    `json:"terjual_per_bulan"`
	Pendapatan      string   `json:"pendapatan"`
	RataHari        string   `json:"rata_hari"`
	Rating          *float64 `json:"rating"`
	BatchID         string   `json:"batch_id"`
}

type LegacyProductsOutput struct {
	Draw            int                `json:"draw"`
	RecordsTotal    int64              `json:"recordsTotal"`
	RecordsFiltered int64              `json:"recordsFiltered"`
	Data            []LegacyProductRow `json:"data"`
}

// ── Our Product ────────────────────────────────────────────────────────

type FilterOptionsOutput struct {
	Brands    []string `json:"brands"`
	Kategoris []string `json:"kategoris"`
}

type OurProductInput struct {
	Search   string
	Brand    []string
	Kategori []string
	Abc      []string
	Start    int
	Length   int
	Draw     int
}

// KompetitorCell is one kompetitor column of the Our Product table:
// store name + matched product price + the batch date that price came
// from. Stale = the price comes from an older batch because the store had
// not been scraped in the latest one; BelumScrape marks a store absent
// from the latest batch (with or without a fallback), so an empty cell can
// be told apart from "checked, nothing found".
type KompetitorCell struct {
	Kompetitor      string   `json:"kompetitor"`
	NamaProduk      *string  `json:"nama_produk"`
	Harga           *float64 `json:"harga"`
	TanggalScraping *string  `json:"tanggal_scraping"`
	Stale           bool     `json:"stale"`
	BelumScrape     bool     `json:"belum_scrape"`
}

type OurProductRow struct {
	ID              uint64           `json:"id"`
	SKU             string           `json:"sku"`
	Nama            string           `json:"nama"`
	Kategori        string           `json:"kategori"`
	Brand           string           `json:"brand"`
	HppLatest       float64          `json:"hpp_latest"`
	HargaShopee     float64          `json:"harga_shopee"`
	HargaTayang     *KompetitorCell  `json:"harga_tayang"`
	MarginShopee    *float64         `json:"margin_shopee"`
	Abc             *string          `json:"abc"`
	KompetitorHarga []KompetitorCell `json:"kompetitor_harga"`
}

type OurProductOutput struct {
	Draw            int             `json:"draw"`
	RecordsTotal    int64           `json:"recordsTotal"`
	RecordsFiltered int64           `json:"recordsFiltered"`
	Data            []OurProductRow `json:"data"`
}

// ── Ingest (Laravel parses the Excel, this service persists the rows) ───

type IngestProductInput struct {
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

type ImportProductsInput struct {
	KompetitorID uint64
	ExecutedAt   string // YYYY-MM-DD
	BatchCode    string // empty = "import-<Ymd>"
	Products     []IngestProductInput
}

type ImportProductsOutput struct {
	BatchCode     string `json:"batch_code"`
	ExecutedAt    string `json:"executed_at"`
	TotalProducts int    `json:"total_products"`
	Kompetitor    string `json:"kompetitor"`
}
