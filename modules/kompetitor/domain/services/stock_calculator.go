package services

import "github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/entities"

// TotalStock sums every gudang column of a warehouse row — a port of
// App\Helpers\StockCalculator::calculateTotalStock. A nil row is 0 stock.
func TotalStock(w *entities.Warehouse) int64 {
	if w == nil {
		return 0
	}

	return w.GudangAItc +
		w.GudangAtTransitItc2 +
		w.GudangBJkt +
		w.GudangBtTransitJkt +
		w.GudangCKemuning +
		w.GudangC6Lebak +
		w.GudangCtTransitPusat +
		w.GudangDSmg +
		w.GudangDtTransitSmg +
		w.GudangEJog +
		w.GudangEtTransitJog +
		w.GudangFMlg +
		w.GudangFtTransitMlg +
		w.GudangHBali +
		w.GudangHtTransitBali +
		w.GudangXSbyLama +
		w.GudangYSby +
		w.GudangY3DisplayTenggil +
		w.GudangYtTransitYSby
}
