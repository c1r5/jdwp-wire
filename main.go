package main

import (
	"os"

	"github.com/c1r5/jdwp-wire/cmd"
)

func main() {
	os.Exit(cmd.Run(os.Args[1:]))
}
