// Package rustanalyzer provides a read-only, in-process Rust/Cargo analyzer.
//
// The adapter deliberately reads Cargo manifests and Rust source as data. It
// never invokes Cargo, rustc, build scripts, proc-macro hosts, or the target
// application.
package rustanalyzer

import (
	"context"
	"os"
	"path/filepath"

	"github.com/buffo/arch-view/internal/analysis"
)

// Analyzer implements the language-neutral analyzer contract for Rust.
type Analyzer struct{}

// New returns the built-in Rust analyzer.
func New() *Analyzer {
	return &Analyzer{}
}

// Manifest describes the stable public Rust analyzer contract.
func (Analyzer) Manifest() analysis.Manifest {
	return analysis.Manifest{
		ID:         "org.archview.rust",
		Version:    "1.0.0",
		Language:   "rust",
		APIVersion: analysis.AnalyzerAPIVersion,
		DetectionMarkers: []analysis.DetectionMarker{
			{Kind: "file", Value: "Cargo.toml", Weight: 1},
		},
		Capabilities: []string{"detect", "static_dependencies", "cfg_metadata"},
		Options: []analysis.OptionDescriptor{
			{Name: "crate", Type: "string", Default: nil, Description: "Selected Cargo package name or workspace-relative crate directory."},
			{Name: "features", Type: "string[]", Default: []string{}, Description: "Explicit Cargo features recorded for the static analysis view."},
			{Name: "target", Type: "string", Default: nil, Description: "Explicit Rust target triple or target selector recorded for the view."},
			{Name: "include_tests", Type: "boolean", Default: false, Description: "Include Rust test targets and cfg(test) modules."},
			{Name: "include_examples", Type: "boolean", Default: false, Description: "Include Rust examples and benches."},
			{Name: "exclude", Type: "string[]", Default: []string{}, Description: "Additional repository-relative exclusion globs."},
		},
	}
}

// Detect identifies a Cargo project without reading or executing Rust code.
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
	if !safeProjectFile(root, "Cargo.toml") {
		return analysis.DetectionCandidate{
			AnalyzerID: a.Manifest().ID,
			Reason:     "Rust Cargo project marker detection",
		}, nil
	}
	return analysis.DetectionCandidate{
		AnalyzerID:     a.Manifest().ID,
		Confidence:     1,
		MatchedMarkers: []string{"Cargo.toml"},
		BoundaryHint:   "Cargo.toml",
		Reason:         "Rust Cargo project marker detection",
	}, nil
}

// Analyze resolves one Cargo crate and emits common module, relationship,
// reference, evidence, and diagnostic observations.
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

func safeProjectFile(root, relative string) bool {
	filePath := filepath.Join(root, filepath.FromSlash(relative))
	info, err := os.Stat(filePath)
	return err == nil && !info.IsDir() && resolvedPathWithin(root, filePath)
}

func resolvedPathWithin(root, candidate string) bool {
	resolvedRoot, rootErr := filepath.EvalSymlinks(root)
	resolvedCandidate, candidateErr := filepath.EvalSymlinks(candidate)
	if rootErr != nil || candidateErr != nil {
		return false
	}
	return pathWithin(filepath.Clean(resolvedRoot), filepath.Clean(resolvedCandidate))
}

func pathWithin(root, candidate string) bool {
	relative, err := filepath.Rel(filepath.Clean(root), filepath.Clean(candidate))
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !isParentPath(relative))
}

func isParentPath(relative string) bool {
	return len(relative) >= 2 && relative[:2] == ".." && (len(relative) == 2 || relative[2] == filepath.Separator)
}
