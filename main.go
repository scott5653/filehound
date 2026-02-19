package main

import (
	"os"

	"github.com/ripkitten-co/filehound/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
