// Package logger configures the shared pkg/logger instance from
// app-wide config, so every layer logs through the same sink.
package logger

import (
	"github.com/dbklik/dbklik-kompetitor-service/internal/config"
	"github.com/dbklik/dbklik-kompetitor-service/pkg/logger"
)

func New(cfg *config.Config) *logger.Logger {
	return logger.New(logger.Options{
		JSON:  cfg.LogJSON,
		Debug: cfg.LogDebug,
	})
}
