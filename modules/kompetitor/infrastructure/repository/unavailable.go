package repository

import (
	"time"

	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/entities"
	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/repositories"
	apperrors "github.com/dbklik/dbklik-kompetitor-service/pkg/errors"
)

// The Unavailable* types implement the kompetitor repository interfaces,
// returning a clear "database not configured" error at request time. Used
// when the shared container has no *gorm.DB (e.g. DB_DRIVER unset) so the
// app can still boot without this module crashing — it just can't serve
// requests until a database is configured.

func errDBUnavailable() error {
	return apperrors.Unavailable("kompetitor module: database not configured (set DB_DRIVER/DB_DSN)")
}

type UnavailableKompetitorRepo struct{}

func (UnavailableKompetitorRepo) ListAll() ([]entities.Kompetitor, error) {
	return nil, errDBUnavailable()
}
func (UnavailableKompetitorRepo) ListAllByPriority() ([]entities.Kompetitor, error) {
	return nil, errDBUnavailable()
}
func (UnavailableKompetitorRepo) ListScraped() ([]entities.Kompetitor, error) {
	return nil, errDBUnavailable()
}
func (UnavailableKompetitorRepo) FindByID(id uint64) (*entities.Kompetitor, error) {
	return nil, errDBUnavailable()
}
func (UnavailableKompetitorRepo) Create(row *entities.Kompetitor) error { return errDBUnavailable() }
func (UnavailableKompetitorRepo) Update(row *entities.Kompetitor) error { return errDBUnavailable() }
func (UnavailableKompetitorRepo) Delete(id uint64) error                { return errDBUnavailable() }
func (UnavailableKompetitorRepo) CodeTaken(column, code string, exceptID uint64) (bool, error) {
	return false, errDBUnavailable()
}
func (UnavailableKompetitorRepo) HasScrapingData(id uint64) (bool, error) {
	return false, errDBUnavailable()
}
func (UnavailableKompetitorRepo) MappingCounts() (map[uint64]int64, error) {
	return nil, errDBUnavailable()
}

type UnavailableMappingRepo struct{}

func (UnavailableMappingRepo) PairRows(kategoriIDs, brandIDs []uint64) ([]repositories.MappingRow, error) {
	return nil, errDBUnavailable()
}
func (UnavailableMappingRepo) UniversalRows(brandIDs []uint64) ([]repositories.MappingRow, error) {
	return nil, errDBUnavailable()
}
func (UnavailableMappingRepo) MatrixCounts(brandIDs, kategoriIDs []uint64) (map[string]int64, error) {
	return nil, errDBUnavailable()
}
func (UnavailableMappingRepo) UniversalCounts(brandIDs []uint64) (map[uint64]int64, error) {
	return nil, errDBUnavailable()
}
func (UnavailableMappingRepo) UsedAxes() ([]uint64, []uint64, error) {
	return nil, nil, errDBUnavailable()
}
func (UnavailableMappingRepo) SyncCell(kategoriID *uint64, brandID uint64, kompetitorIDs []uint64) error {
	return errDBUnavailable()
}
func (UnavailableMappingRepo) KompetitorIDsExist(ids []uint64) (bool, error) {
	return false, errDBUnavailable()
}

type UnavailableBrandRepo struct{}

func (UnavailableBrandRepo) ListActive(ids []uint64) ([]entities.Brand, error) {
	return nil, errDBUnavailable()
}
func (UnavailableBrandRepo) Exists(brandID uint64) (bool, error) { return false, errDBUnavailable() }

type UnavailableKategoriRepo struct{}

func (UnavailableKategoriRepo) List(ids []uint64) ([]entities.Kategori, error) {
	return nil, errDBUnavailable()
}
func (UnavailableKategoriRepo) Exists(kategoriID uint64) (bool, error) {
	return false, errDBUnavailable()
}

type UnavailableScrapingRepo struct{}

func (UnavailableScrapingRepo) BatchesByKompetitor() ([]repositories.BatchRow, error) {
	return nil, errDBUnavailable()
}
func (UnavailableScrapingRepo) Summaries(pairs repositories.BatchPair, kompetitorID *uint64) ([]entities.ScrapingSummary, error) {
	return nil, errDBUnavailable()
}
func (UnavailableScrapingRepo) Products(pairs repositories.BatchPair, kompetitorID *uint64, search string, offset, limit int) ([]repositories.ProductRow, int64, error) {
	return nil, 0, errDBUnavailable()
}
func (UnavailableScrapingRepo) ProductsByNames(pairs repositories.BatchPair, names []string) ([]repositories.MatchableProduct, error) {
	return nil, errDBUnavailable()
}
func (UnavailableScrapingRepo) ProductsForPairs(pairs repositories.BatchPair) ([]repositories.MatchableProduct, error) {
	return nil, errDBUnavailable()
}
func (UnavailableScrapingRepo) ProductsForBatch(batchID uint64) ([]repositories.MatchableProduct, error) {
	return nil, errDBUnavailable()
}
func (UnavailableScrapingRepo) ProductNamesForBatches(batchIDs []uint64) ([]string, error) {
	return nil, errDBUnavailable()
}
func (UnavailableScrapingRepo) LatestCompletedBatch() (*entities.ScrapingBatch, error) {
	return nil, errDBUnavailable()
}
func (UnavailableScrapingRepo) BatchRowsUpTo(to time.Time) ([]repositories.BatchRow, error) {
	return nil, errDBUnavailable()
}
func (UnavailableScrapingRepo) FilterProducts(search, kompetitorName, batchCode string, offset, limit int) ([]repositories.LegacyProductRow, int64, error) {
	return nil, 0, errDBUnavailable()
}
func (UnavailableScrapingRepo) CompletedBatchCodes() ([]string, error) {
	return nil, errDBUnavailable()
}
func (UnavailableScrapingRepo) ItemIDsByProductIDs(productIDs []uint64) (map[uint64]uint64, error) {
	return nil, errDBUnavailable()
}

type UnavailableIngestRepo struct{}

func (UnavailableIngestRepo) FirstOrCreateBatch(code string, executedAt time.Time) (*entities.ScrapingBatch, error) {
	return nil, errDBUnavailable()
}
func (UnavailableIngestRepo) ReplaceProducts(batchID, kompetitorID uint64, products []repositories.IngestProduct) error {
	return errDBUnavailable()
}
func (UnavailableIngestRepo) ComputeSummaries(batchID uint64) error { return errDBUnavailable() }
func (UnavailableIngestRepo) MarkStatus(batchID uint64, status string) error {
	return errDBUnavailable()
}
func (UnavailableIngestRepo) ReplaceProductMappings(batchID uint64, rows []entities.ScrapingProductMapping) error {
	return errDBUnavailable()
}

type UnavailableItemRepo struct{}

func (UnavailableItemRepo) Mappable() ([]repositories.MappableItem, error) {
	return nil, errDBUnavailable()
}

type UnavailableOurProductRepo struct{}

func (UnavailableOurProductRepo) Page(filters repositories.OurProductFilters, offset, limit int) ([]repositories.OurProductRow, int64, error) {
	return nil, 0, errDBUnavailable()
}
func (UnavailableOurProductRepo) FilterOptions() ([]string, []string, error) {
	return nil, nil, errDBUnavailable()
}

type UnavailableWarehouseRepo struct{}

func (UnavailableWarehouseRepo) TotalStockByItemIDs(itemIDs []uint64) (map[uint64]int64, error) {
	return nil, errDBUnavailable()
}

type UnavailableMarketplaceFeeRepo struct{}

func (UnavailableMarketplaceFeeRepo) ShopeeFee(id uint64) (float64, error) {
	return 0, errDBUnavailable()
}
func (UnavailableMarketplaceFeeRepo) FreeOngkirFee(id uint64) (float64, error) {
	return 0, errDBUnavailable()
}
