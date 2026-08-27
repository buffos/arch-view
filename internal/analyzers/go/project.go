package goanalyzer

import (
	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analyzers/go/scanner"
)

// Project remains available from the Go analyzer package for callers that
// need the resolved module boundary. Resolution itself belongs to scanner.
type Project = scanner.Project

func ResolveProject(root string, options analysis.EffectiveOptions) (Project, error) {
	return scanner.ResolveProject(root, options)
}
