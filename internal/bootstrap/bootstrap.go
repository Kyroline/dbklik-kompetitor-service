// Package bootstrap composes config, logger, database, container and
// router into a ready-to-serve gin.Engine. cmd/* entrypoints call this
// instead of wiring the framework themselves.
package bootstrap

import (
	"github.com/dbklik/dbklik-kompetitor-service/internal/config"
	"github.com/dbklik/dbklik-kompetitor-service/internal/container"
	"github.com/dbklik/dbklik-kompetitor-service/internal/database"
	"github.com/dbklik/dbklik-kompetitor-service/internal/logger"
	"github.com/dbklik/dbklik-kompetitor-service/internal/middleware"
	"github.com/dbklik/dbklik-kompetitor-service/internal/router"
	"github.com/dbklik/dbklik-kompetitor-service/routes"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

func Boot() (*gin.Engine, *container.Container) {
	cfg := config.Load()
	log := logger.New(cfg)
	db := database.Connect(cfg)
	c := container.New(cfg, log, db)

	engine := router.New(c, routes.Modules(c)...)

	return engine, c
}

// BootGRPC composes config, logger, database, container and a *grpc.Server
// with the shared recovery/logging interceptors, ready to serve. cmd/grpc
// calls this instead of wiring the framework itself.
func BootGRPC() (*grpc.Server, *container.Container) {
	cfg := config.Load()
	log := logger.New(cfg)
	db := database.Connect(cfg)
	c := container.New(cfg, log, db)

	server := grpc.NewServer(grpc.ChainUnaryInterceptor(
		middleware.RecoveryInterceptor(log),
		middleware.LoggingInterceptor(log),
	))
	for _, m := range routes.GRPCModules(c) {
		m.RegisterGRPC(server)
	}

	return server, c
}
