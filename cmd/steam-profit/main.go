package main

import (
	"os"

	"steam-profit-calculator/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
