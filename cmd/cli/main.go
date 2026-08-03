// Command cli is the command-line entrypoint. Wire module CLI commands
// here as they're added under presentation/cli, via routes/cli.go.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("dbklik-kompetitor-service CLI — no commands registered yet")
	os.Exit(0)
}
