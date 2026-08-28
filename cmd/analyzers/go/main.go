package main

import (
	"context"
	"os"

	"github.com/buffo/arch-view/internal/analysis/processplugin"
	goanalyzer "github.com/buffo/arch-view/internal/analyzers/go"
)

func main() {
	if err := processplugin.Run(context.Background(), goanalyzer.New(), os.Stdin, os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}
