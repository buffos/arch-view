package main

import (
	"context"
	"os"

	"github.com/buffo/arch-view/internal/analysis/processplugin"
	clojureanalyzer "github.com/buffo/arch-view/internal/analyzers/clojure"
)

func main() {
	if err := processplugin.Run(context.Background(), clojureanalyzer.New(), os.Stdin, os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}
