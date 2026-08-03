package services

import (
	"math"
	"strconv"
	"strings"
)

// formatRupiah renders an amount the way the Laravel controllers did:
// 'Rp'.number_format($value, 0, ',', '.') — no decimals, dot thousand
// separators, no space after "Rp".
func formatRupiah(value float64) string {
	rounded := int64(math.Round(value))

	sign := ""
	if rounded < 0 {
		sign = "-"
		rounded = -rounded
	}

	digits := strconv.FormatInt(rounded, 10)

	var b strings.Builder
	b.WriteString("Rp")
	b.WriteString(sign)
	for i, r := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}

	return b.String()
}

// roundTo rounds to the given number of decimals, matching PHP's round().
func roundTo(value float64, decimals int) float64 {
	factor := math.Pow(10, float64(decimals))
	return math.Round(value*factor) / factor
}
