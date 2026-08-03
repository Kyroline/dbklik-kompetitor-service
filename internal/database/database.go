// Package database owns the shared database connection. Modules never
// open their own connections — they receive *gorm.DB through the
// container and build their infrastructure/persistence repositories on it.
package database

import (
	"log/slog"

	"github.com/dbklik/dbklik-kompetitor-service/internal/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// DB is the shared connection handle used by every module's
// infrastructure/persistence layer.
type DB = gorm.DB

// Connect opens the shared connection described by cfg. When no driver is
// configured, it returns nil so the app can still run modules that only
// need an in-memory repository (as the sample inventory module does).
func Connect(cfg *config.Config) *DB {
	if cfg.DBDriver == "" {
		slog.Info("database: no driver configured, skipping connection")
		return nil
	}

	slog.Info("database: connecting", "driver", cfg.DBDriver)

	switch cfg.DBDriver {
	case "mysql":
		db, err := gorm.Open(mysql.Open(cfg.DBDSN), &gorm.Config{})
		if err != nil {
			slog.Error("database: failed to connect", "error", err)
			return nil
		}
		return db
	default:
		slog.Warn("database: unsupported driver, skipping connection", "driver", cfg.DBDriver)
		return nil
	}
}
