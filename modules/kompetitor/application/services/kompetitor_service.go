// Package services exposes the module's use cases as a single facade
// consumed by the presentation layer, so the gRPC server depends on one
// cohesive API instead of wiring repositories itself. The facade's methods
// are split across files by concern: CRUD here, mapping panel in
// mapping_service.go, scraping tables in scraping_service.go, the Our
// Product table in our_product_service.go, ingest in ingest_service.go.
package services

import (
	"log/slog"
	"strings"

	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/application/dto"
	modconfig "github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/config"
	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/entities"
	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/repositories"
	apperrors "github.com/dbklik/dbklik-kompetitor-service/pkg/errors"
)

type KompetitorService struct {
	kompetitors repositories.KompetitorRepository
	mappings    repositories.MappingRepository
	brands      repositories.BrandRepository
	kategoris   repositories.KategoriRepository
	scraping    repositories.ScrapingRepository
	ingest      repositories.IngestRepository
	items       repositories.ItemRepository
	ourProducts repositories.OurProductRepository
	warehouses  repositories.WarehouseRepository
	fees        repositories.MarketplaceFeeRepository

	cfg    *modconfig.Config
	idf    *IdfCache
	logger *slog.Logger
}

func NewKompetitorService(
	kompetitors repositories.KompetitorRepository,
	mappings repositories.MappingRepository,
	brands repositories.BrandRepository,
	kategoris repositories.KategoriRepository,
	scraping repositories.ScrapingRepository,
	ingest repositories.IngestRepository,
	items repositories.ItemRepository,
	ourProducts repositories.OurProductRepository,
	warehouses repositories.WarehouseRepository,
	fees repositories.MarketplaceFeeRepository,
	cfg *modconfig.Config,
	logger *slog.Logger,
) *KompetitorService {
	return &KompetitorService{
		kompetitors: kompetitors,
		mappings:    mappings,
		brands:      brands,
		kategoris:   kategoris,
		scraping:    scraping,
		ingest:      ingest,
		items:       items,
		ourProducts: ourProducts,
		warehouses:  warehouses,
		fees:        fees,
		cfg:         cfg,
		idf:         NewIdfCache(),
		logger:      logger,
	}
}

// ManageData lists every kompetitor with the number of (kategori × brand)
// cells it is monitored in — a port of KompetitorController::manageData.
// The mappings themselves are managed in the Mapping tab, not here.
func (s *KompetitorService) ManageData() (dto.ManageDataOutput, error) {
	rows, err := s.kompetitors.ListAll()
	if err != nil {
		return dto.ManageDataOutput{}, err
	}

	counts, err := s.kompetitors.MappingCounts()
	if err != nil {
		return dto.ManageDataOutput{}, err
	}

	data := make([]dto.KompetitorRow, len(rows))
	for i, row := range rows {
		data[i] = dto.KompetitorRow{
			ID:            row.ID,
			Name:          row.Name,
			ShopeeCode:    derefString(row.ShopeeCode),
			TokopediaCode: derefString(row.TokopediaCode),
			MustScrape:    row.MustScrape,
			MappingCount:  counts[row.ID],
		}
	}

	return dto.ManageDataOutput{Data: data}, nil
}

// IndexMeta returns the reference data the Riset Produk page used to get
// from KompetitorController::indexNew's view payload.
func (s *KompetitorService) IndexMeta() (dto.IndexMetaOutput, error) {
	scraped, err := s.kompetitors.ListScraped()
	if err != nil {
		return dto.IndexMetaOutput{}, err
	}

	all, err := s.kompetitors.ListAllByPriority()
	if err != nil {
		return dto.IndexMetaOutput{}, err
	}

	brands, err := s.brands.ListActive(nil)
	if err != nil {
		return dto.IndexMetaOutput{}, err
	}

	kategoris, err := s.kategoris.List(nil)
	if err != nil {
		return dto.IndexMetaOutput{}, err
	}

	out := dto.IndexMetaOutput{
		DbklikKompetitorID: s.cfg.DbklikKompetitorID,
		ListKompetitor:     make([]dto.NamedRow, len(scraped)),
		ListAllKompetitor:  make([]dto.KompetitorMetaRow, len(all)),
		ListBrand:          make([]dto.NamedRow, len(brands)),
		ListKategori:       make([]dto.NamedRow, len(kategoris)),
	}

	for i, row := range scraped {
		out.ListKompetitor[i] = dto.NamedRow{ID: row.ID, Name: row.Name}
	}
	for i, row := range all {
		out.ListAllKompetitor[i] = dto.KompetitorMetaRow{ID: row.ID, Name: row.Name, MustScrape: row.MustScrape}
	}
	for i, row := range brands {
		out.ListBrand[i] = dto.NamedRow{ID: row.ID, Name: row.NamaBrand}
	}
	for i, row := range kategoris {
		out.ListKategori[i] = dto.NamedRow{ID: row.ID, Name: row.NamaKategori}
	}

	return out, nil
}

// Create stores a new kompetitor after the same validation the Laravel
// controller applied.
func (s *KompetitorService) Create(in dto.SaveKompetitorInput) error {
	if err := s.validateKompetitor(in); err != nil {
		return err
	}

	row := &entities.Kompetitor{
		Name:          in.Name,
		ShopeeCode:    nullableString(in.ShopeeCode),
		TokopediaCode: nullableString(in.TokopediaCode),
		MustScrape:    in.MustScrape,
	}

	return s.kompetitors.Create(row)
}

// Update rewrites an existing kompetitor.
func (s *KompetitorService) Update(in dto.SaveKompetitorInput) error {
	row, err := s.kompetitors.FindByID(in.ID)
	if err != nil {
		return err
	}
	if row == nil {
		return apperrors.NotFound("Kompetitor tidak ditemukan.")
	}

	if err := s.validateKompetitor(in); err != nil {
		return err
	}

	row.Name = in.Name
	row.ShopeeCode = nullableString(in.ShopeeCode)
	row.TokopediaCode = nullableString(in.TokopediaCode)
	row.MustScrape = in.MustScrape

	return s.kompetitors.Update(row)
}

// Delete removes a kompetitor and its mappings. A kompetitor that already
// has scraping data cannot be deleted.
func (s *KompetitorService) Delete(id uint64) error {
	row, err := s.kompetitors.FindByID(id)
	if err != nil {
		return err
	}
	if row == nil {
		return apperrors.NotFound("Kompetitor tidak ditemukan.")
	}

	hasData, err := s.kompetitors.HasScrapingData(id)
	if err != nil {
		return err
	}
	if hasData {
		return apperrors.InvalidInput("Kompetitor tidak bisa dihapus karena sudah memiliki data scraping.")
	}

	return s.kompetitors.Delete(id)
}

// validateKompetitor ports KompetitorController::validateKompetitor: name
// required, at least one of the two codes, both unique across kompetitors.
func (s *KompetitorService) validateKompetitor(in dto.SaveKompetitorInput) error {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return apperrors.InvalidInput("nama toko wajib diisi.")
	}
	if len(name) > 255 {
		return apperrors.InvalidInput("nama toko maksimal 255 karakter.")
	}

	if in.ShopeeCode == "" && in.TokopediaCode == "" {
		return apperrors.InvalidInput("Isi minimal salah satu: code Shopee atau code Tokopedia.")
	}

	for _, check := range []struct {
		column string
		value  string
		label  string
	}{
		{"shopee_code", in.ShopeeCode, "code Shopee"},
		{"tokopedia_code", in.TokopediaCode, "code Tokopedia"},
	} {
		if check.value == "" {
			continue
		}
		if len(check.value) > 255 {
			return apperrors.InvalidInput(check.label + " maksimal 255 karakter.")
		}

		taken, err := s.kompetitors.CodeTaken(check.column, check.value, in.ID)
		if err != nil {
			return err
		}
		if taken {
			return apperrors.InvalidInput(check.label + " sudah dipakai kompetitor lain.")
		}
	}

	return nil
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func nullableString(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
