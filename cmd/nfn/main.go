package main

import (
	"os"

	"github.com/nordic-financial-news/nfn-cli/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
