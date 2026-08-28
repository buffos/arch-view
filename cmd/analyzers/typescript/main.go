package main

import (
	"context"
	"os"

	"github.com/buffo/arch-view/internal/analysis/processplugin"
	tsanalyzer "github.com/buffo/arch-view/internal/analyzers/typescript"
)

func main() {
	if err := processplugin.Run(context.Background(), tsanalyzer.New(), os.Stdin, os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}
