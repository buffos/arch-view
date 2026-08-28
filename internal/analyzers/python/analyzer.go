// Package pyanalyzer provides the in-process Python analyzer.
//
// This package performs read-only project/module discovery and conservative
// static import analysis behind the common analyzer contract. It never
// imports, executes, installs, or introspects the target Python project.
package pyanalyzer

import (
	"context"
	"os"
	"path/filepath"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/syntax"
	pysyntax "github.com/buffo/arch-view/internal/analysis/syntax/python"
)

// Analyzer implements the language-neutral analyzer contract for Python.
type Analyzer struct {
	syntaxProvider syntax.Provider
}

// New returns the built-in Python analyzer.
func New() *Analyzer {
	return NewWithSyntaxProvider(pysyntax.NewProvider())
}

// NewWithSyntaxProvider returns a Python analyzer backed directly by the
// supplied syntax provider.
func NewWithSyntaxProvider(provider syntax.Provider) *Analyzer {
	return &Analyzer{syntaxProvider: provider}
}

// Manifest describes the stable public Python analyzer contract.
func (Analyzer) Manifest() analysis.Manifest {
	return analysis.Manifest{
		ID:         "org.archview.python",
		Version:    "1.0.0",
		Language:   "python",
		APIVersion: analysis.AnalyzerAPIVersion,
		DetectionMarkers: []analysis.DetectionMarker{
			{Kind: "file", Value: "pyproject.toml", Weight: 1},
			{Kind: "file", Value: "setup.cfg", Weight: 0.9},
			{Kind: "file", Value: "setup.py", Weight: 0.8},
		},
		Capabilities: []string{"detect", "static_dependencies", "dynamic_diagnostics"},
		Options: []analysis.OptionDescriptor{
			{Name: "source_roots", Type: "string[]", Default: []string{}, Description: "Explicit repository-relative Python source roots."},
			{Name: "python_version", Type: "string", Default: nil, Description: "Optional Python major/minor version used for static interpretation."},
			{Name: "include_stubs", Type: "boolean", Default: false, Description: "Include .pyi stub files as module evidence."},
			{Name: "include_tests", Type: "boolean", Default: false, Description: "Include Python test files and test directories."},
			{Name: "exclude", Type: "string[]", Default: []string{}, Description: "Additional repository-relative exclusion globs."},
		},
	}
}

// Detect identifies common Python project markers without reading or running
// Python code. Marker precedence is the declaration order in Manifest.
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
		Reason:         "Python project marker detection",
	}, nil
}

// Analyze resolves the Python boundary and emits module/package observations,
// static dependency observations, references, evidence, and diagnostics. The
// host supplies selection provenance, run identity, and the effective option
// fingerprint after this method returns.
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
	discovery, err := DiscoverWithSyntaxProvider(ctx, project, request.Options, a.syntaxProvider)
	if err != nil {
		return analysis.AnalysisResult{}, err
	}
	return BuildResult(project, discovery, request, a.Manifest()), nil
}

func existsAsFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
