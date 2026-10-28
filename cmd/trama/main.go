package main

import (
	"os"

	"github.com/lukascivil/trama/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
