// Package provider wires the kompetitor module's dependencies together
// (repositories -> application service -> gRPC server) and exposes
// RegisterGRPC so internal/bootstrap can attach the module without knowing
// its internals.
package provider

import (
	"github.com/dbklik/dbklik-kompetitor-service/internal/container"
	appservices "github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/application/services"
	modconfig "github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/config"
	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/repositories"
	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/infrastructure/repository"
	grpcpresentation "github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/presentation/grpc"
	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/presentation/grpc/pb"
	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/presentation/http/controllers"
	modroutes "github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/routes"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

// KompetitorProvider exposes the module over two transports backed by the
// same application service: gRPC for service-to-service callers, and HTTP
// for the Laravel portal, which cannot speak gRPC (no ext-grpc) and proxies
// these endpoints instead.
type KompetitorProvider struct {
	grpcServer *grpcpresentation.KompetitorGRPCServer
	controller *controllers.KompetitorController
}

// New builds the module's dependency graph. This module has no meaningful
// in-memory fallback — it reads/writes the SAME shared MySQL database as
// the Laravel portal_produk_dbklik app. When the container has no *gorm.DB
// (DB_DRIVER unset), we still construct the provider so the app doesn't
// crash at boot, but every repository call returns a clear "database not
// configured" error at request time.
func New(c *container.Container) *KompetitorProvider {
	var (
		kompetitorRepo repositories.KompetitorRepository
		mappingRepo    repositories.MappingRepository
		brandRepo      repositories.BrandRepository
		kategoriRepo   repositories.KategoriRepository
		scrapingRepo   repositories.ScrapingRepository
		ingestRepo     repositories.IngestRepository
		itemRepo       repositories.ItemRepository
		ourProductRepo repositories.OurProductRepository
		warehouseRepo  repositories.WarehouseRepository
		feeRepo        repositories.MarketplaceFeeRepository
	)

	if c.DB == nil {
		c.Logger.Warn("kompetitor module: no database configured, RPCs will return UNAVAILABLE until DB_DRIVER/DB_DSN is set")
		kompetitorRepo = repository.UnavailableKompetitorRepo{}
		mappingRepo = repository.UnavailableMappingRepo{}
		brandRepo = repository.UnavailableBrandRepo{}
		kategoriRepo = repository.UnavailableKategoriRepo{}
		scrapingRepo = repository.UnavailableScrapingRepo{}
		ingestRepo = repository.UnavailableIngestRepo{}
		itemRepo = repository.UnavailableItemRepo{}
		ourProductRepo = repository.UnavailableOurProductRepo{}
		warehouseRepo = repository.UnavailableWarehouseRepo{}
		feeRepo = repository.UnavailableMarketplaceFeeRepo{}
	} else {
		kompetitorRepo = repository.NewKompetitorRepositoryGorm(c.DB)
		mappingRepo = repository.NewMappingRepositoryGorm(c.DB)
		brandRepo = repository.NewBrandRepositoryGorm(c.DB)
		kategoriRepo = repository.NewKategoriRepositoryGorm(c.DB)
		scrapingRepo = repository.NewScrapingRepositoryGorm(c.DB)
		ingestRepo = repository.NewIngestRepositoryGorm(c.DB)
		itemRepo = repository.NewItemRepositoryGorm(c.DB)
		ourProductRepo = repository.NewOurProductRepositoryGorm(c.DB)
		warehouseRepo = repository.NewWarehouseRepositoryGorm(c.DB)
		feeRepo = repository.NewMarketplaceFeeRepositoryGorm(c.DB)
	}

	service := appservices.NewKompetitorService(
		kompetitorRepo,
		mappingRepo,
		brandRepo,
		kategoriRepo,
		scrapingRepo,
		ingestRepo,
		itemRepo,
		ourProductRepo,
		warehouseRepo,
		feeRepo,
		modconfig.Load(),
		c.Logger,
	)

	return &KompetitorProvider{
		grpcServer: grpcpresentation.NewKompetitorGRPCServer(service),
		controller: controllers.NewKompetitorController(service),
	}
}

func (p *KompetitorProvider) RegisterGRPC(s *grpc.Server) {
	pb.RegisterKompetitorServiceServer(s, p.grpcServer)
}

func (p *KompetitorProvider) RegisterRoutes(api *gin.RouterGroup) {
	modroutes.RegisterAPI(api, p.controller)
}
