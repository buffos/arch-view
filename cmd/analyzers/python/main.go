package main

import (
	"context"
	"os"

	"github.com/buffo/arch-view/internal/analysis/processplugin"
	pyanalyzer "github.com/buffo/arch-view/internal/analyzers/python"
)

func main() {
	if err := processplugin.Run(context.Background(), pyanalyzer.New(), os.Stdin, os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}
