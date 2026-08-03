// Package container holds the framework-level dependencies that are
// shared across every module (config, logger, database), so modules
// receive them instead of constructing their own.
package container

import (
	"log/slog"

	"github.com/dbklik/dbklik-kompetitor-service/internal/config"
	"github.com/dbklik/dbklik-kompetitor-service/internal/database"
)

type Container struct {
	Config *config.Config
	Logger *slog.Logger
	DB     *database.DB // *gorm.DB, aliased so modules don't import gorm directly
}

func New(cfg *config.Config, logger *slog.Logger, db *database.DB) *Container {
	return &Container{Config: cfg, Logger: logger, DB: db}
}
