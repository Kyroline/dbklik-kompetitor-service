// Package router assembles the gin.Engine: global middleware, health check,
// and the versioned API group that module routes attach to.
package router

import (
	"net/http"

	"github.com/dbklik/dbklik-kompetitor-service/internal/container"
	internalhttp "github.com/dbklik/dbklik-kompetitor-service/internal/http"
	"github.com/dbklik/dbklik-kompetitor-service/internal/middleware"
	"github.com/gin-gonic/gin"
)

func New(c *container.Container, modules ...internalhttp.ModuleRouter) *gin.Engine {
	if c.Config.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(middleware.Recovery(c.Logger), middleware.RequestLogger(c.Logger), middleware.CORS())

	engine.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := engine.Group("/api/v1")
	for _, m := range modules {
		m.RegisterRoutes(api)
	}

	return engine
}
