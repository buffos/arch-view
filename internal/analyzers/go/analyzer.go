package goanalyzer

import (
	"context"
	"os"
	"path/filepath"
	"sort"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analyzers/go/observations"
	"github.com/buffo/arch-view/internal/analyzers/go/scanner"
)

type Analyzer struct{}

func New() *Analyzer {
	return &Analyzer{}
}

func (Analyzer) Manifest() analysis.Manifest {
	return analysis.Manifest{
		ID:         "org.archview.go",
		Version:    "1.0.0",
		Language:   "go",
		APIVersion: analysis.AnalyzerAPIVersion,
		DetectionMarkers: []analysis.DetectionMarker{
			{Kind: "file", Value: "go.mod", Weight: 1},
			{Kind: "file", Value: "go.work", Weight: 0.9},
		},
		Capabilities: []string{"detect", "static_dependencies", "build_constraints"},
		Options: []analysis.OptionDescriptor{
			{Name: "module", Type: "string", Default: nil, Description: "Selected module path or workspace-relative module directory."},
			{Name: "build_tags", Type: "string[]", Default: []string{}, Description: "Explicit build tags for the analysis view."},
			{Name: "include_tests", Type: "boolean", Default: false, Description: "Include test files in the source scope."},
			{Name: "include_generated", Type: "boolean", Default: false, Description: "Include files marked as generated."},
			{Name: "include_external", Type: "boolean", Default: false, Description: "Retain detail for non-local references."},
			{Name: "exclude", Type: "string[]", Default: []string{}, Description: "Additional repository-relative exclusion globs."},
			{Name: "safe_mode", Type: "boolean", Default: true, Description: "Disable target-code execution and tool-assisted execution."},
		},
	}
}

func (a Analyzer) Detect(ctx context.Context, request analysis.DetectRequest) (analysis.DetectionCandidate, error) {
	if ctx.Err() != nil {
		return analysis.DetectionCandidate{}, ctx.Err()
	}
	root := filepath.Clean(request.ProjectRoot)
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return analysis.DetectionCandidate{}, analysis.NewHostError(analysis.ErrUnreadableProject, "project root does not exist", map[string]any{"project_root": request.ProjectRoot})
		}
		return analysis.DetectionCandidate{}, analysis.WrapHostError(analysis.ErrUnreadableProject, "project root could not be read", err, map[string]any{"project_root": request.ProjectRoot})
	}
	if !info.IsDir() {
		return analysis.DetectionCandidate{}, analysis.NewHostError(analysis.ErrInvalidRequest, "project root must be a directory", map[string]any{"project_root": request.ProjectRoot})
	}
	markers := make([]string, 0, 2)
	confidence := 0.0
	boundaryHint := ""
	if existsAsFile(filepath.Join(root, "go.mod")) {
		markers = append(markers, "go.mod")
		confidence = 1
		boundaryHint = "go.mod"
	}
	if existsAsFile(filepath.Join(root, "go.work")) {
		markers = append(markers, "go.work")
		if confidence == 0 {
			confidence = 0.9
			boundaryHint = "go.work"
		}
	}
	sort.Strings(markers)
	return analysis.DetectionCandidate{
		AnalyzerID:     a.Manifest().ID,
		Confidence:     confidence,
		MatchedMarkers: markers,
		BoundaryHint:   boundaryHint,
		Reason:         "Go project marker detection",
	}, nil
}

func (a Analyzer) Analyze(ctx context.Context, request analysis.AnalyzeRequest) (analysis.AnalysisResult, error) {
	if ctx.Err() != nil {
		return analysis.AnalysisResult{}, ctx.Err()
	}
	project, err := scanner.ResolveProject(request.ProjectRoot, request.Options)
	if err != nil {
		return analysis.AnalysisResult{}, err
	}
	scan, err := scanner.ScanProject(ctx, request, project)
	if err != nil {
		return analysis.AnalysisResult{}, err
	}
	return observations.Build(scan, request, project, a.Manifest()), nil
}

func existsAsFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
