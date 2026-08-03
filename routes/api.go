// Package routes is the root-level aggregator: it lists every module
// that should be mounted onto the API, decoupling internal/bootstrap
// from knowing which modules exist.
package routes

import (
	"github.com/dbklik/dbklik-kompetitor-service/internal/container"
	internalhttp "github.com/dbklik/dbklik-kompetitor-service/internal/http"
	kompetitorprovider "github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/infrastructure/provider"
)

// Modules returns every module's HTTP route registrar. Add new modules here
// as they're built — this is the single place that lists them. The
// kompetitor module is mounted on both transports: HTTP here (what the
// Laravel portal proxies) and gRPC in routes/grpc.go.
func Modules(c *container.Container) []internalhttp.ModuleRouter {
	return []internalhttp.ModuleRouter{
		kompetitorprovider.New(c),
	}
}
