// Package tsanalyzer provides the in-process TypeScript analyzer.
//
// The analyzer reads TypeScript project configuration and source syntax as
// data. It does not invoke tsc, a bundler, package scripts, or target code.
package tsanalyzer

import (
	"context"
	"os"
	"path/filepath"

	"github.com/buffo/arch-view/internal/analysis"
)

// Analyzer implements the language-neutral analyzer contract for TypeScript.
type Analyzer struct{}

// New returns the built-in TypeScript analyzer.
func New() *Analyzer {
	return &Analyzer{}
}

// Manifest describes the stable public TypeScript analyzer contract.
func (Analyzer) Manifest() analysis.Manifest {
	return analysis.Manifest{
		ID:         "org.archview.typescript",
		Version:    "1.0.0",
		Language:   "typescript",
		APIVersion: analysis.AnalyzerAPIVersion,
		DetectionMarkers: []analysis.DetectionMarker{
			{Kind: "file", Value: "tsconfig.json", Weight: 1},
			{Kind: "file", Value: "package.json", Weight: 0.6},
		},
		Capabilities: []string{"detect", "static_dependencies", "aliases", "exports"},
		Options: []analysis.OptionDescriptor{
			{Name: "config", Type: "string", Default: nil, Description: "Repository-relative tsconfig path when more than one project config exists."},
			{Name: "include_js", Type: "boolean", Default: false, Description: "Include JavaScript and JSX files in addition to TypeScript files."},
			{Name: "include_tests", Type: "boolean", Default: false, Description: "Include test and spec files and directories."},
			{Name: "runtime", Type: "string", Default: "auto", AllowedValues: []string{"auto", "esm", "cjs"}, Description: "Static package-export condition preference."},
			{Name: "exclude", Type: "string[]", Default: []string{}, Description: "Additional repository-relative exclusion globs."},
		},
	}
}

// Detect identifies TypeScript project markers without reading or executing
// project source or package scripts. tsconfig.json is the stronger marker.
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
			return analysis.DetectionCandidate{}, analysis.NewHostError(analysis.ErrUnreadableProject, "project root does not exist", map[string]any{"project_root": request.ProjectRoot})
		}
		return analysis.DetectionCandidate{}, analysis.WrapHostError(analysis.ErrUnreadableProject, "project root could not be read", err, map[string]any{"project_root": request.ProjectRoot})
	}
	if !info.IsDir() {
		return analysis.DetectionCandidate{}, analysis.NewHostError(analysis.ErrInvalidRequest, "project root must be a directory", map[string]any{"project_root": request.ProjectRoot})
	}

	manifest := a.Manifest()
	matched := make([]string, 0, len(manifest.DetectionMarkers))
	confidence := 0.0
	boundary := ""
	for _, marker := range manifest.DetectionMarkers {
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
		AnalyzerID:     manifest.ID,
		Confidence:     confidence,
		MatchedMarkers: matched,
		BoundaryHint:   boundary,
		Reason:         "TypeScript project marker detection",
	}, nil
}

// Analyze resolves one TypeScript project and emits common observations. The
// host supplies selection provenance, run identity, and option fingerprint.
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
	return BuildResult(project, discovery, request, a.Manifest()), nil
}

func existsAsFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
