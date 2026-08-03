// Package controllers receives HTTP requests, maps them onto the
// application layer, and returns the legacy JSON shapes. Controllers stay
// thin — no business logic lives here.
//
// This layer exists next to presentation/grpc because the Laravel app
// cannot speak gRPC (no ext-grpc); it proxies over HTTP instead, while
// service-to-service callers keep using the gRPC contract. Both entry
// points call the very same application service.
package controllers

import (
	"sort"
	"strconv"
	"strings"

	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/application/dto"
	appservices "github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/application/services"
	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/presentation/http/requests"
	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/presentation/http/responses"
	apperrors "github.com/dbklik/dbklik-kompetitor-service/pkg/errors"
	"github.com/gin-gonic/gin"
)

type KompetitorController struct {
	service *appservices.KompetitorService
}

func NewKompetitorController(service *appservices.KompetitorService) *KompetitorController {
	return &KompetitorController{service: service}
}

// ── Kompetitor CRUD ────────────────────────────────────────────────────

func (ctl *KompetitorController) Meta(c *gin.Context) {
	out, err := ctl.service.IndexMeta()
	if err != nil {
		responses.Error(c, err)
		return
	}
	responses.OK(c, out)
}

func (ctl *KompetitorController) ManageData(c *gin.Context) {
	out, err := ctl.service.ManageData()
	if err != nil {
		responses.Error(c, err)
		return
	}
	responses.OK(c, out)
}

func (ctl *KompetitorController) Store(c *gin.Context) {
	var req requests.SaveKompetitorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.Error(c, apperrors.InvalidInput(err.Error()))
		return
	}

	err := ctl.service.Create(dto.SaveKompetitorInput{
		Name:          req.Name,
		ShopeeCode:    req.ShopeeCode,
		TokopediaCode: req.TokopediaCode,
		MustScrape:    req.MustScrape,
	})
	if err != nil {
		responses.Error(c, err)
		return
	}

	responses.Created(c, "Kompetitor berhasil ditambahkan.")
}

func (ctl *KompetitorController) Update(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		responses.Error(c, err)
		return
	}

	var req requests.SaveKompetitorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.Error(c, apperrors.InvalidInput(err.Error()))
		return
	}

	err = ctl.service.Update(dto.SaveKompetitorInput{
		ID:            id,
		Name:          req.Name,
		ShopeeCode:    req.ShopeeCode,
		TokopediaCode: req.TokopediaCode,
		MustScrape:    req.MustScrape,
	})
	if err != nil {
		responses.Error(c, err)
		return
	}

	responses.Message(c, "Kompetitor berhasil diperbarui.")
}

func (ctl *KompetitorController) Destroy(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		responses.Error(c, err)
		return
	}

	if err := ctl.service.Delete(id); err != nil {
		responses.Error(c, err)
		return
	}

	responses.Message(c, "Kompetitor berhasil dihapus.")
}

// ── Mapping panel ──────────────────────────────────────────────────────

func (ctl *KompetitorController) MappingMatrix(c *gin.Context) {
	out, err := ctl.service.MappingMatrix(dto.MatrixInput{
		BrandIDs:    queryUintSlice(c, "brands"),
		KategoriIDs: queryUintSlice(c, "kategoris"),
	})
	if err != nil {
		responses.Error(c, err)
		return
	}
	responses.OK(c, out)
}

func (ctl *KompetitorController) MappingCell(c *gin.Context) {
	out, err := ctl.service.MappingCell(dto.CellInput{
		KategoriID: queryOptionalUint(c, "kategori_id"),
		BrandID:    queryUint(c, "brand_id"),
	})
	if err != nil {
		responses.Error(c, err)
		return
	}
	responses.OK(c, out)
}

func (ctl *KompetitorController) MappingCellUpdate(c *gin.Context) {
	var req requests.MappingCellUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.Error(c, apperrors.InvalidInput(err.Error()))
		return
	}

	err := ctl.service.MappingCellUpdate(dto.CellUpdateInput{
		KategoriID:    req.KategoriID,
		BrandID:       req.BrandID,
		KompetitorIDs: req.Kompetitors,
	})
	if err != nil {
		responses.Error(c, err)
		return
	}

	responses.Message(c, "Mapping kompetitor berhasil disimpan.")
}

// ── Riset Produk ───────────────────────────────────────────────────────

func (ctl *KompetitorController) Stats(c *gin.Context) {
	out, err := ctl.service.Stats(periodInput(c))
	if err != nil {
		responses.Error(c, err)
		return
	}
	responses.OK(c, out)
}

func (ctl *KompetitorController) Products(c *gin.Context) {
	out, err := ctl.service.Products(dto.ProductsInput{
		PeriodInput: periodInput(c),
		Search:      c.Query("search_global"),
		Start:       queryInt(c, "start", 0),
		Length:      queryInt(c, "length", 25),
		Draw:        queryInt(c, "draw", 0),
	})
	if err != nil {
		responses.Error(c, err)
		return
	}
	responses.OK(c, out)
}

func (ctl *KompetitorController) LegacyProducts(c *gin.Context) {
	out, err := ctl.service.LegacyProducts(dto.LegacyProductsInput{
		Search:     c.Query("search"),
		Kompetitor: c.Query("kompetitor"),
		Batch:      c.Query("batch"),
		Start:      queryInt(c, "start", 0),
		Length:     queryInt(c, "length", 25),
		Draw:       queryInt(c, "draw", 0),
	})
	if err != nil {
		responses.Error(c, err)
		return
	}
	responses.OK(c, out)
}

func (ctl *KompetitorController) BatchCodes(c *gin.Context) {
	codes, err := ctl.service.BatchCodes()
	if err != nil {
		responses.Error(c, err)
		return
	}
	responses.OK(c, gin.H{"codes": codes})
}

// ── Our Product ────────────────────────────────────────────────────────

func (ctl *KompetitorController) FilterOptions(c *gin.Context) {
	out, err := ctl.service.FilterOptions()
	if err != nil {
		responses.Error(c, err)
		return
	}
	responses.OK(c, out)
}

func (ctl *KompetitorController) OurProducts(c *gin.Context) {
	out, err := ctl.service.OurProducts(dto.OurProductInput{
		Search:   c.Query("search_global"),
		Brand:    queryStringSlice(c, "brand"),
		Kategori: queryStringSlice(c, "kategori"),
		Abc:      queryStringSlice(c, "abc"),
		Start:    queryInt(c, "start", 0),
		Length:   queryInt(c, "length", 10),
		Draw:     queryInt(c, "draw", 0),
	})
	if err != nil {
		responses.Error(c, err)
		return
	}
	responses.OK(c, out)
}

// ── Ingest ─────────────────────────────────────────────────────────────

func (ctl *KompetitorController) ImportProduct(c *gin.Context) {
	var req requests.ImportProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.Error(c, apperrors.InvalidInput(err.Error()))
		return
	}

	products := make([]dto.IngestProductInput, len(req.Products))
	for i, row := range req.Products {
		products[i] = dto.IngestProductInput{
			Name:           row.Name,
			Price:          row.Price,
			SoldMonthly:    row.SoldMonthly,
			RevenueMonthly: row.RevenueMonthly,
			SoldWeekly:     row.SoldWeekly,
			RevenueWeekly:  row.RevenueWeekly,
			Rating:         row.Rating,
			WishlistCount:  row.WishlistCount,
			Stock:          row.Stock,
		}
	}

	out, err := ctl.service.ImportProducts(dto.ImportProductsInput{
		KompetitorID: req.KompetitorID,
		ExecutedAt:   req.ExecutedAt,
		BatchCode:    req.BatchCode,
		Products:     products,
	})
	if err != nil {
		responses.Error(c, err)
		return
	}

	responses.OK(c, out)
}

// ── Query helpers ──────────────────────────────────────────────────────

func periodInput(c *gin.Context) dto.PeriodInput {
	return dto.PeriodInput{
		StartDate:    c.Query("start_date"),
		EndDate:      c.Query("end_date"),
		KompetitorID: queryUint(c, "kompetitor"),
	}
}

func pathID(c *gin.Context) (uint64, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return 0, apperrors.InvalidInput("id kompetitor tidak valid.")
	}
	return id, nil
}

func queryInt(c *gin.Context, key string, fallback int) int {
	value, err := strconv.Atoi(c.Query(key))
	if err != nil {
		return fallback
	}
	return value
}

func queryUint(c *gin.Context, key string) uint64 {
	value, err := strconv.ParseUint(c.Query(key), 10, 64)
	if err != nil {
		return 0
	}
	return value
}

// queryOptionalUint distinguishes "absent/empty" (the universal cell) from
// an actual id, which a plain 0 could not express.
func queryOptionalUint(c *gin.Context, key string) *uint64 {
	raw, ok := c.GetQuery(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return nil
	}
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return nil
	}
	return &value
}

// queryStringSlice collects `key=a&key=b` as well as both PHP array styles:
// `key[]=a` (what the panels' JS sends) and `key[0]=a&key[1]=b` (what PHP's
// http_build_query produces when the portal proxies the request onwards).
func queryStringSlice(c *gin.Context, key string) []string {
	query := c.Request.URL.Query()

	keys := make([]string, 0, len(query))
	for name := range query {
		if name == key || strings.HasPrefix(name, key+"[") {
			keys = append(keys, name)
		}
	}
	sort.Strings(keys)

	out := make([]string, 0, len(keys))
	for _, name := range keys {
		for _, value := range query[name] {
			if value = strings.TrimSpace(value); value != "" {
				out = append(out, value)
			}
		}
	}
	return out
}

func queryUintSlice(c *gin.Context, key string) []uint64 {
	raw := queryStringSlice(c, key)

	out := make([]uint64, 0, len(raw))
	for _, value := range raw {
		if parsed, err := strconv.ParseUint(value, 10, 64); err == nil {
			out = append(out, parsed)
		}
	}
	return out
}
