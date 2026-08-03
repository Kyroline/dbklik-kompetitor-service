// Command api is the REST API entrypoint.
package main

import (
	"log"

	"github.com/dbklik/dbklik-kompetitor-service/internal/app"
)

func main() {
	a := app.New()
	if err := a.Run(); err != nil {
		log.Fatalf("server exited with error: %v", err)
	}
}
