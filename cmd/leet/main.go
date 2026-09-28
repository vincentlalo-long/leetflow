package main

import (
	"os"

	"leetcli/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
