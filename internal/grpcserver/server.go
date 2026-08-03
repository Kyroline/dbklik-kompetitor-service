// Package grpcserver wraps *grpc.Server with a net.Listener, adding
// graceful shutdown on SIGINT/SIGTERM — the gRPC counterpart of
// internal/server.
package grpcserver

import (
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
)

type Server struct {
	grpc   *grpc.Server
	addr   string
	logger *slog.Logger
}

func New(server *grpc.Server, port string, logger *slog.Logger) *Server {
	return &Server{
		grpc:   server,
		addr:   ":" + port,
		logger: logger,
	}
}

// Run starts the server and blocks until it is shut down gracefully via
// SIGINT/SIGTERM, allowing in-flight RPCs to complete.
func (s *Server) Run() error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	go func() {
		s.logger.Info("grpcserver: listening", "addr", s.addr)
		if err := s.grpc.Serve(lis); err != nil {
			s.logger.Error("grpcserver: failed to serve", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Info("grpcserver: shutting down")
	s.grpc.GracefulStop()

	return nil
}
