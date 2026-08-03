// Command grpc is the gRPC API entrypoint.
package main

import (
	"log"

	"github.com/dbklik/dbklik-kompetitor-service/internal/bootstrap"
	"github.com/dbklik/dbklik-kompetitor-service/internal/grpcserver"
)

func main() {
	server, c := bootstrap.BootGRPC()
	if err := grpcserver.New(server, c.Config.GRPCPort, c.Logger).Run(); err != nil {
		log.Fatalf("grpc server exited with error: %v", err)
	}
}
