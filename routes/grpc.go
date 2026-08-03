// Package routes is the root-level aggregator: it lists every module
// that should be mounted onto the gRPC server, decoupling internal/bootstrap
// from knowing which modules exist.
package routes

import (
	"github.com/dbklik/dbklik-kompetitor-service/internal/container"
	internalgrpc "github.com/dbklik/dbklik-kompetitor-service/internal/grpc"
	kompetitorprovider "github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/infrastructure/provider"
)

// GRPCModules returns every module's gRPC service registrar. Add new
// modules here as they're built — this is the single place that lists them.
func GRPCModules(c *container.Container) []internalgrpc.ModuleRegistrar {
	return []internalgrpc.ModuleRegistrar{
		kompetitorprovider.New(c),
	}
}
