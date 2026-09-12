package main

import (
	"os"

	"github.com/c1r5/jdwp-wire/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
