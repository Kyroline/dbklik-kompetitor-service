// Package config holds kompetitor-module-specific configuration, separate
// from the app-wide internal/config. Values mirror the Laravel app's
// config/scraping.php so both sides agree on the same numbers.
package config

import (
	"strconv"

	domainservices "github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/services"
	"github.com/dbklik/dbklik-kompetitor-service/pkg/config"
)

type Config struct {
	// DbklikKompetitorID is our own store's row in `kompetitors`. Its
	// scraped prices are the "harga tayang" column, not a competitor price.
	// 0 disables the column (config/scraping.php: dbklik_kompetitor_id).
	DbklikKompetitorID uint64
	// MatchThreshold is the TF-IDF cosine cutoff for accepting a match
	// (KompetitorController::ourProductData passes 0.6).
	MatchThreshold float64
	// MaxStaleDays is how old a kompetitor's batch may be and still be used
	// as a fallback (ProductMappingService::MAX_STALE_DAYS).
	MaxStaleDays int
}

func Load() *Config {
	return &Config{
		DbklikKompetitorID: parseUint(config.GetString("DBKLIK_KOMPETITOR_ID", "0")),
		MatchThreshold:     parseFloat(config.GetString("KOMPETITOR_MATCH_THRESHOLD", "0.6"), 0.6),
		MaxStaleDays:       config.GetInt("KOMPETITOR_MAX_STALE_DAYS", domainservices.MaxStaleDays),
	}
}

func parseUint(raw string) uint64 {
	v, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

func parseFloat(raw string, fallback float64) float64 {
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return v
}
