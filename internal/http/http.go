// Package http defines the contract every module's presentation layer
// implements to attach its routes to the shared gin engine, keeping
// internal/router decoupled from any specific module.
package http

import "github.com/gin-gonic/gin"

// ModuleRouter is implemented by each module's infrastructure/provider,
// which knows how to build its controllers and register its own routes.
type ModuleRouter interface {
	RegisterRoutes(api *gin.RouterGroup)
}
