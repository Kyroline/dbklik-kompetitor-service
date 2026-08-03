// Package requests holds the HTTP-bound structs of the kompetitor module.
// Field names deliberately mirror the query/body keys the Laravel panels
// already send, so the proxy can forward them unchanged.
package requests

// SaveKompetitorRequest is the body of store()/update().
type SaveKompetitorRequest struct {
	Name          string `json:"name"`
	ShopeeCode    string `json:"shopee_code"`
	TokopediaCode string `json:"tokopedia_code"`
	MustScrape    bool   `json:"must_scrape"`
}

// MappingCellUpdateRequest is the body of mappingCellUpdate(). A null
// kategori_id addresses the brand's universal cell; an empty kompetitors
// list deletes the cell.
type MappingCellUpdateRequest struct {
	KategoriID  *uint64  `json:"kategori_id"`
	BrandID     uint64   `json:"brand_id"`
	Kompetitors []uint64 `json:"kompetitors"`
}

// ImportProductRequest carries rows the Laravel app already parsed out of
// the uploaded Excel file (it keeps owning the upload and the header rules).
type ImportProductRequest struct {
	KompetitorID uint64             `json:"kompetitor_id"`
	ExecutedAt   string             `json:"executed_at"`
	BatchCode    string             `json:"batch_code"`
	Products     []ImportProductRow `json:"products"`
}

type ImportProductRow struct {
	Name           string   `json:"name"`
	Price          float64  `json:"price"`
	SoldMonthly    int64    `json:"sold_monthly"`
	RevenueMonthly float64  `json:"revenue_monthly"`
	SoldWeekly     int64    `json:"sold_weekly"`
	RevenueWeekly  float64  `json:"revenue_weekly"`
	Rating         *float64 `json:"rating"`
	WishlistCount  int64    `json:"wishlist_count"`
	Stock          int64    `json:"stock"`
}
