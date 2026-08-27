// Package clojureanalyzer provides a read-only, in-process Clojure-family
// analyzer for the common Arch View analyzer contract.
//
// The adapter parses source and project configuration as data. It never
// evaluates forms, loads namespaces, starts a Clojure runtime, or consults a
// target project's classpath.
package clojureanalyzer

import (
	"context"
	"os"
	"path/filepath"

	"github.com/buffo/arch-view/internal/analysis"
)

// Analyzer implements the language-neutral analyzer contract for Clojure.
type Analyzer struct{}

// New returns the built-in Clojure-family analyzer.
func New() *Analyzer {
	return &Analyzer{}
}

// Manifest describes the stable public Clojure analyzer contract.
func (Analyzer) Manifest() analysis.Manifest {
	return analysis.Manifest{
		ID:         "org.archview.clojure",
		Version:    "1.0.0",
		Language:   "clojure",
		APIVersion: analysis.AnalyzerAPIVersion,
		DetectionMarkers: []analysis.DetectionMarker{
			{Kind: "file", Value: "deps.edn", Weight: 1},
			{Kind: "file", Value: "project.clj", Weight: 0.9},
			{Kind: "file", Value: "shadow-cljs.edn", Weight: 0.8},
		},
		Capabilities: []string{"detect", "static_dependencies", "polymorphic_metadata"},
		Options: []analysis.OptionDescriptor{
			{Name: "source_roots", Type: "string[]", Default: []string{}, Description: "Explicit repository-relative Clojure source roots."},
			{Name: "platform", Type: "string", Default: "both", AllowedValues: []string{"clj", "cljs", "both"}, Description: "Reader-conditional platform view."},
			{Name: "include_tests", Type: "boolean", Default: false, Description: "Include Clojure test files and test directories."},
			{Name: "exclude", Type: "string[]", Default: []string{}, Description: "Additional repository-relative exclusion globs."},
		},
	}
}

// Detect identifies common Clojure project markers without reading or
// executing any Clojure source.
func (a Analyzer) Detect(ctx context.Context, request analysis.DetectRequest) (analysis.DetectionCandidate, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return analysis.DetectionCandidate{}, err
	}
	root := filepath.Clean(request.ProjectRoot)
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return analysis.DetectionCandidate{}, analysis.NewHostError(analysis.ErrUnreadableProject, "Clojure project root does not exist", map[string]any{"project_root": request.ProjectRoot})
		}
		return analysis.DetectionCandidate{}, analysis.WrapHostError(analysis.ErrUnreadableProject, "Clojure project root could not be read", err, map[string]any{"project_root": request.ProjectRoot})
	}
	if !info.IsDir() {
		return analysis.DetectionCandidate{}, analysis.NewHostError(analysis.ErrInvalidRequest, "Clojure project root must be a directory", map[string]any{"project_root": request.ProjectRoot})
	}

	markers := a.Manifest().DetectionMarkers
	matched := make([]string, 0, len(markers))
	confidence := 0.0
	boundary := ""
	for _, marker := range markers {
		if !safeProjectFile(root, marker.Value) {
			continue
		}
		matched = append(matched, marker.Value)
		if boundary == "" {
			confidence = marker.Weight
			boundary = marker.Value
		}
	}
	return analysis.DetectionCandidate{
		AnalyzerID:     a.Manifest().ID,
		Confidence:     confidence,
		MatchedMarkers: matched,
		BoundaryHint:   boundary,
		Reason:         "Clojure-family project marker detection",
	}, nil
}

// Analyze resolves the Clojure project boundary, discovers namespace modules,
// and emits static observations, source evidence, diagnostics, and
// language-specific metadata. The host supplies selection provenance, run
// identity, and the effective option fingerprint.
func (a Analyzer) Analyze(ctx context.Context, request analysis.AnalyzeRequest) (analysis.AnalysisResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return analysis.AnalysisResult{}, err
	}
	project, err := ResolveProject(request.ProjectRoot, request.Options)
	if err != nil {
		return analysis.AnalysisResult{}, err
	}
	discovery, err := Discover(ctx, project, request.Options)
	if err != nil {
		return analysis.AnalysisResult{}, err
	}
	return BuildResult(project, discovery, a.Manifest()), nil
}

func existsAsFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
