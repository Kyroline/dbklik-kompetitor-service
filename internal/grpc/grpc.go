// Package grpc defines the contract every module's presentation layer
// implements to attach its service to the shared *grpc.Server, keeping
// internal/bootstrap decoupled from any specific module.
package grpc

import "google.golang.org/grpc"

// ModuleRegistrar is implemented by each module's infrastructure/provider,
// which knows how to build its gRPC server and register it.
type ModuleRegistrar interface {
	RegisterGRPC(s *grpc.Server)
}
