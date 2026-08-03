package services

import (
	"strings"
	"time"

	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/application/dto"
	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/entities"
	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/repositories"
	domainservices "github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/services"
	apperrors "github.com/dbklik/dbklik-kompetitor-service/pkg/errors"
)

// ImportProducts persists one store's already-parsed scraping rows.
//
// Excel parsing stays in the Laravel app (it owns the upload form, the
// shops.json filename matching and the header rules); this service only
// receives plain rows, so it is the Go half of
// App\Services\ScrapingProductImportService::import plus
// ScrapingBatch::markCompleted.
func (s *KompetitorService) ImportProducts(in dto.ImportProductsInput) (dto.ImportProductsOutput, error) {
	kompetitor, err := s.kompetitors.FindByID(in.KompetitorID)
	if err != nil {
		return dto.ImportProductsOutput{}, err
	}
	if kompetitor == nil {
		return dto.ImportProductsOutput{}, apperrors.NotFound("Kompetitor tidak ditemukan.")
	}

	executedAt, err := time.Parse("2006-01-02", in.ExecutedAt)
	if err != nil {
		return dto.ImportProductsOutput{}, apperrors.InvalidInput("tanggal scraping harus berformat YYYY-MM-DD.")
	}

	products := normalizeIngestProducts(in.Products)
	if len(products) == 0 {
		return dto.ImportProductsOutput{}, apperrors.InvalidInput("Tidak ada baris data yang bisa dibaca dari file.")
	}

	batchCode := strings.TrimSpace(in.BatchCode)
	if batchCode == "" {
		batchCode = "import-" + executedAt.Format("20060102")
	}

	batch, err := s.ingest.FirstOrCreateBatch(batchCode, executedAt)
	if err != nil {
		return dto.ImportProductsOutput{}, err
	}

	if err := s.ingest.ReplaceProducts(batch.ID, kompetitor.ID, products); err != nil {
		return dto.ImportProductsOutput{}, err
	}

	if err := s.ingest.ComputeSummaries(batch.ID); err != nil {
		return dto.ImportProductsOutput{}, err
	}

	if err := s.ingest.MarkStatus(batch.ID, "completed"); err != nil {
		return dto.ImportProductsOutput{}, err
	}

	// Matching our items against the batch must not fail the import — it can
	// always be redone. Mirrors markCompleted()'s try/catch around mapBatch().
	if err := s.mapBatch(batch.ID); err != nil {
		s.logger.Error("scraping product mapping failed", "batch", batch.Code, "error", err)
	}

	return dto.ImportProductsOutput{
		BatchCode:     batch.Code,
		ExecutedAt:    executedAt.Format("2006-01-02"),
		TotalProducts: len(products),
		Kompetitor:    kompetitor.Name,
	}, nil
}

// mapBatch recomputes and stores the persisted TF-IDF matches of a batch —
// a port of ProductMappingService::mapBatch. The Our Product panel matches
// on the fly, but the stored rows still back the Riset Produk stock column
// and the `scraping:map-products` reporting.
func (s *KompetitorService) mapBatch(batchID uint64) error {
	products, err := s.scraping.ProductsForBatch(batchID)
	if err != nil {
		return err
	}
	if len(products) == 0 {
		return nil
	}

	items, err := s.items.Mappable()
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}

	// Recompute the batch's IDF from scratch so the on-the-fly matching in
	// OurProducts uses exactly the same token weights.
	s.idf.Forget([]uint64{batchID})
	idf, err := s.idfForBatches([]uint64{batchID})
	if err != nil {
		return err
	}

	resolver := domainservices.NewMappingResolver(s.mappings)
	allowedByItem, err := resolver.AllowedByItem(items)
	if err != nil {
		return err
	}

	matches := domainservices.NewTextMatcher().
		ComputeMatches(items, products, s.cfg.MatchThreshold, idf, allowedByItem)

	var rows []entities.ScrapingProductMapping
	for itemID, perKompetitor := range matches {
		for kompetitorID, match := range perKompetitor {
			rows = append(rows, entities.ScrapingProductMapping{
				ScrapingBatchID:   batchID,
				ScrapingProductID: match.Product.ID,
				KompetitorID:      kompetitorID,
				ItemID:            itemID,
				Score:             roundTo(match.Score, 4),
			})
		}
	}

	return s.ingest.ReplaceProductMappings(batchID, rows)
}

// normalizeIngestProducts applies the same row cleaning the Excel importer
// did: skip blank/formula names, cap names at 255 characters, drop ratings
// outside 0–5.
func normalizeIngestProducts(in []dto.IngestProductInput) []repositories.IngestProduct {
	out := make([]repositories.IngestProduct, 0, len(in))

	for _, row := range in {
		name := strings.TrimSpace(row.Name)
		if name == "" || strings.HasPrefix(name, "=") {
			continue
		}
		if len([]rune(name)) > 255 {
			name = string([]rune(name)[:255])
		}

		rating := row.Rating
		if rating != nil && (*rating < 0 || *rating > 5) {
			rating = nil
		} else if rating != nil {
			rounded := roundTo(*rating, 1)
			rating = &rounded
		}

		out = append(out, repositories.IngestProduct{
			Name:           name,
			Price:          row.Price,
			SoldMonthly:    row.SoldMonthly,
			RevenueMonthly: row.RevenueMonthly,
			SoldWeekly:     row.SoldWeekly,
			RevenueWeekly:  row.RevenueWeekly,
			Rating:         rating,
			WishlistCount:  row.WishlistCount,
			Stock:          row.Stock,
		})
	}

	return out
}
