package services

import "math"

// Free-ongkir caps, mirroring Harga::freeOngkirTypeMapping().
const (
	FreeOngkirCapBiasa  = 40000.0
	FreeOngkirCapKhusus = 60000.0
)

// FreeOngkirCap maps a harga row's free_ongkir_type to its cap. Anything
// other than "khusus" falls back to the "biasa" cap, matching the Laravel
// column default.
func FreeOngkirCap(freeOngkirType string) float64 {
	if freeOngkirType == "khusus" {
		return FreeOngkirCapKhusus
	}
	return FreeOngkirCapBiasa
}

// MarginCalculator ports the Shopee branch of Harga::getMarginNilai() plus
// Harga::getMarginPersen(). Only Shopee is ported: the Our Product panel is
// the sole caller and it asks for Shopee margin only.
type MarginCalculator struct{}

func NewMarginCalculator() *MarginCalculator { return &MarginCalculator{} }

// ShopeeMarginNilai returns the rupiah margin of a Shopee price.
//
//	value   = harga - hpp - harga*fee
//	          - min(freeOngkirCap, harga*freeOngkirFee)
//	          - harga*0.005 - min(50000, harga*0.018) - min(60000, harga*0.045)
//	margin  = value - harga*0.001
//
// It returns 0 when the row has no Shopee fee category or no price, exactly
// like the PHP `if ($katId && $harga)` guard. Deliberately not rounded —
// the PHP comment marks that as intentional for Shopee.
func (MarginCalculator) ShopeeMarginNilai(harga, hpp, fee, freeOngkirFee, freeOngkirCap float64, hasFeeKategori bool) float64 {
	if !hasFeeKategori || harga == 0 {
		return 0
	}

	value := harga - hpp -
		(harga * fee) -
		math.Min(freeOngkirCap, harga*freeOngkirFee) -
		(harga * 0.005) -
		math.Min(50000, harga*0.018) -
		math.Min(60000, harga*0.045)

	return value - (harga * 0.001)
}

// MarginPersen expresses a rupiah margin as a percentage of hpp, rounded to
// one decimal. A zero hpp is treated as 1 to avoid dividing by zero, exactly
// as the PHP does.
func (MarginCalculator) MarginPersen(marginNilai, hpp float64) float64 {
	if hpp == 0 {
		hpp = 1
	}
	return math.Round(marginNilai/hpp*100*10) / 10
}

// AverageHpp mirrors Harga::getAverageHpp()'s `(int)` cast of the average
// hpp — the margin base is truncated, not rounded.
func AverageHpp(avg float64) float64 { return math.Trunc(avg) }
