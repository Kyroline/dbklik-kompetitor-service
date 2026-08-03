// Package server wraps net/http.Server around the gin.Engine, adding
// graceful shutdown on SIGINT/SIGTERM.
package server

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

type Server struct {
	http   *http.Server
	logger *slog.Logger
}

func New(engine *gin.Engine, port string, logger *slog.Logger) *Server {
	return &Server{
		http: &http.Server{
			Addr:    ":" + port,
			Handler: engine,
		},
		logger: logger,
	}
}

// Run starts the server and blocks until it is shut down gracefully via
// SIGINT/SIGTERM, allowing in-flight requests up to 10s to complete.
func (s *Server) Run() error {
	go func() {
		s.logger.Info("server: listening", "addr", s.http.Addr)
		if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("server: failed to serve", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Info("server: shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return s.http.Shutdown(ctx)
}
