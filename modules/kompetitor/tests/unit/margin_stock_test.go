package unit

import (
	"math"
	"testing"

	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/entities"
	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/services"
)

// Reference values computed straight from the PHP formula in
// Harga::getMarginNilai()'s shopee branch.
func TestShopeeMarginNilaiMatchesThePhpFormula(t *testing.T) {
	calculator := services.NewMarginCalculator()

	harga := 1_000_000.0
	hpp := 800_000.0
	fee := 0.08
	ongkirFee := 0.04

	got := calculator.ShopeeMarginNilai(harga, hpp, fee, ongkirFee, services.FreeOngkirCapBiasa, true)

	want := harga - hpp -
		harga*fee -
		math.Min(services.FreeOngkirCapBiasa, harga*ongkirFee) -
		harga*0.005 -
		math.Min(50000, harga*0.018) -
		math.Min(60000, harga*0.045) -
		harga*0.001

	if math.Abs(got-want) > 0.0001 {
		t.Fatalf("want %f, got %f", want, got)
	}
}

func TestShopeeMarginNilaiIsZeroWithoutFeeKategoriOrPrice(t *testing.T) {
	calculator := services.NewMarginCalculator()

	if got := calculator.ShopeeMarginNilai(1_000_000, 800_000, 0.08, 0.04, services.FreeOngkirCapBiasa, false); got != 0 {
		t.Fatalf("no fee kategori: want 0, got %f", got)
	}
	if got := calculator.ShopeeMarginNilai(0, 800_000, 0.08, 0.04, services.FreeOngkirCapBiasa, true); got != 0 {
		t.Fatalf("no price: want 0, got %f", got)
	}
}

func TestMarginPersenGuardsAgainstZeroHpp(t *testing.T) {
	calculator := services.NewMarginCalculator()

	if got := calculator.MarginPersen(150, 0); got != 15000 {
		t.Fatalf("zero hpp is treated as 1: want 15000, got %f", got)
	}
	if got := calculator.MarginPersen(25_000, 100_000); got != 25 {
		t.Fatalf("want 25, got %f", got)
	}
}

func TestAverageHppTruncates(t *testing.T) {
	if got := services.AverageHpp(1999.99); got != 1999 {
		t.Fatalf("want 1999, got %f", got)
	}
}

func TestFreeOngkirCap(t *testing.T) {
	if services.FreeOngkirCap("khusus") != services.FreeOngkirCapKhusus {
		t.Fatal("khusus must use the 60k cap")
	}
	if services.FreeOngkirCap("") != services.FreeOngkirCapBiasa {
		t.Fatal("an empty type falls back to the 40k cap")
	}
}

func TestTotalStockSumsEveryGudangColumn(t *testing.T) {
	if got := services.TotalStock(nil); got != 0 {
		t.Fatalf("a missing warehouse row is 0 stock, got %d", got)
	}

	warehouse := &entities.Warehouse{
		GudangAItc: 3,
		GudangBJkt: 4,
		GudangYSby: 5,
	}
	if got := services.TotalStock(warehouse); got != 12 {
		t.Fatalf("want 12, got %d", got)
	}
}
