package main

import (
	"os"

	"github.com/indexer-benchmark/cmd/indexer/app"
)

func main() {
	if err := app.Run(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
