package main

import (
	"os"

	"storage-optimizer/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}