// Package logger is a thin, reusable wrapper around log/slog so that
// application code depends on this package instead of the standard
// library directly, keeping the logging backend swappable.
package logger

import (
	"os"

	"log/slog"
)

type Logger = slog.Logger

type Options struct {
	JSON  bool
	Debug bool
}

func New(opts Options) *Logger {
	level := slog.LevelInfo
	if opts.Debug {
		level = slog.LevelDebug
	}

	handlerOpts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if opts.JSON {
		handler = slog.NewJSONHandler(os.Stdout, handlerOpts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, handlerOpts)
	}

	return slog.New(handler)
}
