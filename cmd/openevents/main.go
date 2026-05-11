package main

import (
	"os"

	"github.com/sentiolabs/open-events/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
