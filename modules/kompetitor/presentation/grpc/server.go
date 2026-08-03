// Package grpc receives gRPC requests for the kompetitor module,
// translates them into application-layer DTOs, and maps the results back
// onto the wire — the gRPC counterpart of the Laravel controller that used
// to render these payloads as JSON.
package grpc

import (
	"context"

	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/application/dto"
	appservices "github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/application/services"
	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/presentation/grpc/pb"
	"github.com/dbklik/dbklik-kompetitor-service/pkg/grpcerrors"
)

type KompetitorGRPCServer struct {
	pb.UnimplementedKompetitorServiceServer
	service *appservices.KompetitorService
}

func NewKompetitorGRPCServer(service *appservices.KompetitorService) *KompetitorGRPCServer {
	return &KompetitorGRPCServer{service: service}
}

// ── Kompetitor CRUD ────────────────────────────────────────────────────

func (s *KompetitorGRPCServer) IndexMeta(_ context.Context, _ *pb.IndexMetaRequest) (*pb.IndexMetaResponse, error) {
	out, err := s.service.IndexMeta()
	if err != nil {
		return nil, grpcerrors.ToStatus(err)
	}

	resp := &pb.IndexMetaResponse{
		DbklikKompetitorId: out.DbklikKompetitorID,
		ListKompetitor:     namedRows(out.ListKompetitor),
		ListBrand:          namedRows(out.ListBrand),
		ListKategori:       namedRows(out.ListKategori),
		ListAllKompetitor:  make([]*pb.KompetitorMetaRow, len(out.ListAllKompetitor)),
	}
	for i, row := range out.ListAllKompetitor {
		resp.ListAllKompetitor[i] = &pb.KompetitorMetaRow{
			Id:         row.ID,
			Name:       row.Name,
			MustScrape: row.MustScrape,
		}
	}

	return resp, nil
}

func (s *KompetitorGRPCServer) ManageData(_ context.Context, _ *pb.ManageDataRequest) (*pb.ManageDataResponse, error) {
	out, err := s.service.ManageData()
	if err != nil {
		return nil, grpcerrors.ToStatus(err)
	}

	rows := make([]*pb.KompetitorRow, len(out.Data))
	for i, row := range out.Data {
		rows[i] = &pb.KompetitorRow{
			Id:            row.ID,
			Name:          row.Name,
			ShopeeCode:    row.ShopeeCode,
			TokopediaCode: row.TokopediaCode,
			MustScrape:    row.MustScrape,
			MappingCount:  row.MappingCount,
		}
	}

	return &pb.ManageDataResponse{Data: rows}, nil
}

func (s *KompetitorGRPCServer) CreateKompetitor(_ context.Context, req *pb.SaveKompetitorRequest) (*pb.MessageResponse, error) {
	if err := s.service.Create(saveInput(req)); err != nil {
		return nil, grpcerrors.ToStatus(err)
	}
	return &pb.MessageResponse{Message: "Kompetitor berhasil ditambahkan."}, nil
}

func (s *KompetitorGRPCServer) UpdateKompetitor(_ context.Context, req *pb.SaveKompetitorRequest) (*pb.MessageResponse, error) {
	if err := s.service.Update(saveInput(req)); err != nil {
		return nil, grpcerrors.ToStatus(err)
	}
	return &pb.MessageResponse{Message: "Kompetitor berhasil diperbarui."}, nil
}

func (s *KompetitorGRPCServer) DeleteKompetitor(_ context.Context, req *pb.DeleteKompetitorRequest) (*pb.MessageResponse, error) {
	if err := s.service.Delete(req.GetId()); err != nil {
		return nil, grpcerrors.ToStatus(err)
	}
	return &pb.MessageResponse{Message: "Kompetitor berhasil dihapus."}, nil
}

// ── Mapping panel ──────────────────────────────────────────────────────

func (s *KompetitorGRPCServer) MappingMatrix(_ context.Context, req *pb.MappingMatrixRequest) (*pb.MappingMatrixResponse, error) {
	out, err := s.service.MappingMatrix(dto.MatrixInput{
		BrandIDs:    req.GetBrands(),
		KategoriIDs: req.GetKategoris(),
	})
	if err != nil {
		return nil, grpcerrors.ToStatus(err)
	}

	return &pb.MappingMatrixResponse{
		Brands:          namedRows(out.Brands),
		Kategoris:       namedRows(out.Kategoris),
		Counts:          out.Counts,
		UniversalCounts: out.UniversalCounts,
		Filtered:        out.Filtered,
	}, nil
}

func (s *KompetitorGRPCServer) MappingCell(_ context.Context, req *pb.MappingCellRequest) (*pb.MappingCellResponse, error) {
	out, err := s.service.MappingCell(dto.CellInput{
		KategoriID: optionalUint64(req.KategoriId),
		BrandID:    req.GetBrandId(),
	})
	if err != nil {
		return nil, grpcerrors.ToStatus(err)
	}
	return &pb.MappingCellResponse{KompetitorIds: out.KompetitorIDs}, nil
}

func (s *KompetitorGRPCServer) MappingCellUpdate(_ context.Context, req *pb.MappingCellUpdateRequest) (*pb.MessageResponse, error) {
	err := s.service.MappingCellUpdate(dto.CellUpdateInput{
		KategoriID:    optionalUint64(req.KategoriId),
		BrandID:       req.GetBrandId(),
		KompetitorIDs: req.GetKompetitors(),
	})
	if err != nil {
		return nil, grpcerrors.ToStatus(err)
	}
	return &pb.MessageResponse{Message: "Mapping kompetitor berhasil disimpan."}, nil
}

// ── Riset Produk ───────────────────────────────────────────────────────

func (s *KompetitorGRPCServer) Stats(_ context.Context, req *pb.StatsRequest) (*pb.StatsResponse, error) {
	out, err := s.service.Stats(dto.PeriodInput{
		StartDate:    req.GetStartDate(),
		EndDate:      req.GetEndDate(),
		KompetitorID: req.GetKompetitorId(),
	})
	if err != nil {
		return nil, grpcerrors.ToStatus(err)
	}

	return &pb.StatsResponse{
		Current:         statsMetrics(out.Current),
		Previous:        statsMetrics(out.Previous),
		CurrentBatches:  out.CurrentBatches,
		PreviousBatches: out.PreviousBatches,
	}, nil
}

func (s *KompetitorGRPCServer) Products(_ context.Context, req *pb.ProductsRequest) (*pb.ProductsResponse, error) {
	out, err := s.service.Products(dto.ProductsInput{
		PeriodInput: dto.PeriodInput{
			StartDate:    req.GetStartDate(),
			EndDate:      req.GetEndDate(),
			KompetitorID: req.GetKompetitorId(),
		},
		Search: req.GetSearch(),
		Start:  int(req.GetStart()),
		Length: int(req.GetLength()),
		Draw:   int(req.GetDraw()),
	})
	if err != nil {
		return nil, grpcerrors.ToStatus(err)
	}

	rows := make([]*pb.ProductRow, len(out.Data))
	for i, row := range out.Data {
		rows[i] = &pb.ProductRow{
			Kompetitor:      row.Kompetitor,
			NamaProduk:      row.NamaProduk,
			Harga:           row.Harga,
			PerubahanHarga:  row.PerubahanHarga,
			TerjualPerBulan: row.TerjualPerBulan,
			Pendapatan:      row.Pendapatan,
			RataHari:        row.RataHari,
			Rating:          row.Rating,
			Stok:            row.Stok,
			BatchId:         row.BatchID,
		}
	}

	return &pb.ProductsResponse{
		Draw:            int32(out.Draw),
		RecordsTotal:    out.RecordsTotal,
		RecordsFiltered: out.RecordsFiltered,
		Data:            rows,
	}, nil
}

func (s *KompetitorGRPCServer) LegacyProducts(_ context.Context, req *pb.LegacyProductsRequest) (*pb.LegacyProductsResponse, error) {
	out, err := s.service.LegacyProducts(dto.LegacyProductsInput{
		Search:     req.GetSearch(),
		Kompetitor: req.GetKompetitor(),
		Batch:      req.GetBatch(),
		Start:      int(req.GetStart()),
		Length:     int(req.GetLength()),
		Draw:       int(req.GetDraw()),
	})
	if err != nil {
		return nil, grpcerrors.ToStatus(err)
	}

	rows := make([]*pb.LegacyProductRow, len(out.Data))
	for i, row := range out.Data {
		rows[i] = &pb.LegacyProductRow{
			Kompetitor:      row.Kompetitor,
			NamaProduk:      row.NamaProduk,
			Harga:           row.Harga,
			HargaRaw:        row.HargaRaw,
			TerjualPerBulan: row.TerjualPerBulan,
			Pendapatan:      row.Pendapatan,
			RataHari:        row.RataHari,
			Rating:          row.Rating,
			BatchId:         row.BatchID,
		}
	}

	return &pb.LegacyProductsResponse{
		Draw:            int32(out.Draw),
		RecordsTotal:    out.RecordsTotal,
		RecordsFiltered: out.RecordsFiltered,
		Data:            rows,
	}, nil
}

func (s *KompetitorGRPCServer) BatchCodes(_ context.Context, _ *pb.BatchCodesRequest) (*pb.BatchCodesResponse, error) {
	codes, err := s.service.BatchCodes()
	if err != nil {
		return nil, grpcerrors.ToStatus(err)
	}
	return &pb.BatchCodesResponse{Codes: codes}, nil
}

// ── Our Product ────────────────────────────────────────────────────────

func (s *KompetitorGRPCServer) FilterOptions(_ context.Context, _ *pb.FilterOptionsRequest) (*pb.FilterOptionsResponse, error) {
	out, err := s.service.FilterOptions()
	if err != nil {
		return nil, grpcerrors.ToStatus(err)
	}
	return &pb.FilterOptionsResponse{Brands: out.Brands, Kategoris: out.Kategoris}, nil
}

func (s *KompetitorGRPCServer) OurProducts(_ context.Context, req *pb.OurProductsRequest) (*pb.OurProductsResponse, error) {
	out, err := s.service.OurProducts(dto.OurProductInput{
		Search:   req.GetSearch(),
		Brand:    req.GetBrand(),
		Kategori: req.GetKategori(),
		Abc:      req.GetAbc(),
		Start:    int(req.GetStart()),
		Length:   int(req.GetLength()),
		Draw:     int(req.GetDraw()),
	})
	if err != nil {
		return nil, grpcerrors.ToStatus(err)
	}

	rows := make([]*pb.OurProductRow, len(out.Data))
	for i, row := range out.Data {
		cells := make([]*pb.KompetitorCell, len(row.KompetitorHarga))
		for j := range row.KompetitorHarga {
			cells[j] = kompetitorCell(&row.KompetitorHarga[j])
		}

		rows[i] = &pb.OurProductRow{
			Id:              row.ID,
			Sku:             row.SKU,
			Nama:            row.Nama,
			Kategori:        row.Kategori,
			Brand:           row.Brand,
			HppLatest:       row.HppLatest,
			HargaShopee:     row.HargaShopee,
			HargaTayang:     kompetitorCell(row.HargaTayang),
			MarginShopee:    row.MarginShopee,
			Abc:             row.Abc,
			KompetitorHarga: cells,
		}
	}

	return &pb.OurProductsResponse{
		Draw:            int32(out.Draw),
		RecordsTotal:    out.RecordsTotal,
		RecordsFiltered: out.RecordsFiltered,
		Data:            rows,
	}, nil
}

// ── Ingest ─────────────────────────────────────────────────────────────

func (s *KompetitorGRPCServer) ImportProducts(_ context.Context, req *pb.ImportProductsRequest) (*pb.ImportProductsResponse, error) {
	products := make([]dto.IngestProductInput, len(req.GetProducts()))
	for i, row := range req.GetProducts() {
		products[i] = dto.IngestProductInput{
			Name:           row.GetName(),
			Price:          row.GetPrice(),
			SoldMonthly:    row.GetSoldMonthly(),
			RevenueMonthly: row.GetRevenueMonthly(),
			SoldWeekly:     row.GetSoldWeekly(),
			RevenueWeekly:  row.GetRevenueWeekly(),
			Rating:         row.Rating,
			WishlistCount:  row.GetWishlistCount(),
			Stock:          row.GetStock(),
		}
	}

	out, err := s.service.ImportProducts(dto.ImportProductsInput{
		KompetitorID: req.GetKompetitorId(),
		ExecutedAt:   req.GetExecutedAt(),
		BatchCode:    req.GetBatchCode(),
		Products:     products,
	})
	if err != nil {
		return nil, grpcerrors.ToStatus(err)
	}

	return &pb.ImportProductsResponse{
		BatchCode:     out.BatchCode,
		ExecutedAt:    out.ExecutedAt,
		TotalProducts: int32(out.TotalProducts),
		Kompetitor:    out.Kompetitor,
	}, nil
}

// ── Mapping helpers ────────────────────────────────────────────────────

func saveInput(req *pb.SaveKompetitorRequest) dto.SaveKompetitorInput {
	return dto.SaveKompetitorInput{
		ID:            req.GetId(),
		Name:          req.GetName(),
		ShopeeCode:    req.GetShopeeCode(),
		TokopediaCode: req.GetTokopediaCode(),
		MustScrape:    req.GetMustScrape(),
	}
}

func namedRows(rows []dto.NamedRow) []*pb.NamedRow {
	out := make([]*pb.NamedRow, len(rows))
	for i, row := range rows {
		out[i] = &pb.NamedRow{Id: row.ID, Name: row.Name}
	}
	return out
}

func statsMetrics(in *dto.StatsMetrics) *pb.StatsMetrics {
	if in == nil {
		return nil
	}
	return &pb.StatsMetrics{
		TotalOmzet:        in.TotalOmzet,
		RataOmzetMinggu:   in.RataOmzetMinggu,
		RataOmzetHari:     in.RataOmzetHari,
		TotalTerjual:      in.TotalTerjual,
		RataTerjualMinggu: in.RataTerjualMingu,
		RataTerjualHari:   in.RataTerjualHari,
		RataHarga:         in.RataHarga,
		ProdukReady:       in.ProdukReady,
		ProdukHabis:       in.ProdukHabis,
	}
}

func kompetitorCell(in *dto.KompetitorCell) *pb.KompetitorCell {
	if in == nil {
		return nil
	}
	return &pb.KompetitorCell{
		Kompetitor:      in.Kompetitor,
		NamaProduk:      in.NamaProduk,
		Harga:           in.Harga,
		TanggalScraping: in.TanggalScraping,
		Stale:           in.Stale,
		BelumScrape:     in.BelumScrape,
	}
}

// optionalUint64 turns proto3 field presence into the nil-able kategori id
// the application layer expects (unset = the brand's universal cell).
func optionalUint64(v *uint64) *uint64 {
	if v == nil {
		return nil
	}
	value := *v
	return &value
}
