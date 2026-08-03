package services

import (
	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/application/dto"
	domainservices "github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/services"
	apperrors "github.com/dbklik/dbklik-kompetitor-service/pkg/errors"
)

// MappingMatrix returns the matrix panel data: kategori on the X axis,
// brand on the Y axis, each cell holding the number of mapped kompetitors
// — a port of KompetitorController::mappingMatrix. Without a filter only
// axes that already have mappings are shown, otherwise the matrix would
// explode into thousands of columns.
func (s *KompetitorService) MappingMatrix(in dto.MatrixInput) (dto.MatrixOutput, error) {
	usedBrandIDs, usedKategoriIDs, err := s.mappings.UsedAxes()
	if err != nil {
		return dto.MatrixOutput{}, err
	}

	brandIDs := in.BrandIDs
	if len(brandIDs) == 0 {
		brandIDs = usedBrandIDs
	}
	kategoriIDs := in.KategoriIDs
	if len(kategoriIDs) == 0 {
		kategoriIDs = usedKategoriIDs
	}

	// An empty axis after the fallback means "nothing mapped yet" — passing
	// nil on to the repository would wrongly mean "no filter", so bail out
	// with empty axes instead.
	brands := []dto.NamedRow{}
	if len(brandIDs) > 0 {
		rows, err := s.brands.ListActive(brandIDs)
		if err != nil {
			return dto.MatrixOutput{}, err
		}
		brands = make([]dto.NamedRow, len(rows))
		for i, row := range rows {
			brands[i] = dto.NamedRow{ID: row.ID, Name: row.NamaBrand}
		}
	}

	kategoris := []dto.NamedRow{}
	if len(kategoriIDs) > 0 {
		rows, err := s.kategoris.List(kategoriIDs)
		if err != nil {
			return dto.MatrixOutput{}, err
		}
		kategoris = make([]dto.NamedRow, len(rows))
		for i, row := range rows {
			kategoris[i] = dto.NamedRow{ID: row.ID, Name: row.NamaKategori}
		}
	}

	shownBrandIDs := idsOf(brands)
	shownKategoriIDs := idsOf(kategoris)

	counts, err := s.mappings.MatrixCounts(shownBrandIDs, shownKategoriIDs)
	if err != nil {
		return dto.MatrixOutput{}, err
	}

	universal, err := s.mappings.UniversalCounts(shownBrandIDs)
	if err != nil {
		return dto.MatrixOutput{}, err
	}

	return dto.MatrixOutput{
		Brands:          brands,
		Kategoris:       kategoris,
		Counts:          counts,
		UniversalCounts: universal,
		Filtered:        len(in.BrandIDs) > 0 || len(in.KategoriIDs) > 0,
	}, nil
}

// MappingCell returns one cell's kompetitor list for the (kategori, brand)
// pair. A nil kategori_id asks for the brand's universal cell. Specific
// cells are returned raw — without the universal fallback — so an unmapped
// cell shows up empty in the edit modal.
func (s *KompetitorService) MappingCell(in dto.CellInput) (dto.CellOutput, error) {
	if err := s.assertCellExists(in.KategoriID, in.BrandID); err != nil {
		return dto.CellOutput{}, err
	}

	resolver := domainservices.NewMappingResolver(s.mappings)

	if in.KategoriID == nil {
		universal, err := resolver.UniversalFor(in.BrandID)
		if err != nil {
			return dto.CellOutput{}, err
		}
		return dto.CellOutput{KompetitorIDs: refIDs(universal)}, nil
	}

	specific, err := resolver.RawPair(*in.KategoriID, in.BrandID)
	if err != nil {
		return dto.CellOutput{}, err
	}

	return dto.CellOutput{KompetitorIDs: refIDs(specific)}, nil
}

// MappingCellUpdate replaces one cell's kompetitor list. An empty list
// deletes the cell; a nil kategori_id addresses the brand's universal cell.
func (s *KompetitorService) MappingCellUpdate(in dto.CellUpdateInput) error {
	if err := s.assertCellExists(in.KategoriID, in.BrandID); err != nil {
		return err
	}

	if len(in.KompetitorIDs) > 0 {
		ok, err := s.mappings.KompetitorIDsExist(in.KompetitorIDs)
		if err != nil {
			return err
		}
		if !ok {
			return apperrors.InvalidInput("kompetitor yang dipilih tidak valid.")
		}
	}

	return s.mappings.SyncCell(in.KategoriID, in.BrandID, in.KompetitorIDs)
}

// assertCellExists mirrors the controller's exists:kategori,id /
// exists:brand,id validation rules.
func (s *KompetitorService) assertCellExists(kategoriID *uint64, brandID uint64) error {
	if brandID == 0 {
		return apperrors.InvalidInput("brand wajib diisi.")
	}

	exists, err := s.brands.Exists(brandID)
	if err != nil {
		return err
	}
	if !exists {
		return apperrors.InvalidInput("brand tidak ditemukan.")
	}

	if kategoriID == nil {
		return nil
	}

	exists, err = s.kategoris.Exists(*kategoriID)
	if err != nil {
		return err
	}
	if !exists {
		return apperrors.InvalidInput("kategori tidak ditemukan.")
	}

	return nil
}

func idsOf(rows []dto.NamedRow) []uint64 {
	ids := make([]uint64, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	return ids
}
