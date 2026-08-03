// Package app is the top-level application object that cmd/api's main.go
// creates and runs, hiding the bootstrap/server wiring behind two calls.
package app

import (
	"github.com/dbklik/dbklik-kompetitor-service/internal/bootstrap"
	"github.com/dbklik/dbklik-kompetitor-service/internal/container"
	"github.com/dbklik/dbklik-kompetitor-service/internal/server"
	"github.com/gin-gonic/gin"
)

type Application struct {
	Engine    *gin.Engine
	Container *container.Container
}

func New() *Application {
	engine, c := bootstrap.Boot()
	return &Application{Engine: engine, Container: c}
}

func (a *Application) Run() error {
	return server.New(a.Engine, a.Container.Config.AppPort, a.Container.Logger).Run()
}
