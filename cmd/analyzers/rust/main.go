package main

import (
	"context"
	"os"

	"github.com/buffo/arch-view/internal/analysis/processplugin"
	rustanalyzer "github.com/buffo/arch-view/internal/analyzers/rust"
)

func main() {
	if err := processplugin.Run(context.Background(), rustanalyzer.New(), os.Stdin, os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}
