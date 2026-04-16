package main

import (
	"os"

	"github.com/skalluru/velocix/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
