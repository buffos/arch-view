package goanalyzer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
)

func goOptions(t *testing.T, values map[string]any) analysis.EffectiveOptions {
	t.Helper()
	options, err := analysis.ResolveOptions(New().Manifest(), nil, values)
	if err != nil {
		t.Fatalf("resolve Go options: %v", err)
	}
	return options
}

func writeFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}

func TestResolveProjectFromGoMod(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, filepath.Join(root, "go.mod"), "module example.com/service\n\ngo 1.22\n")
	project, err := ResolveProject(root, goOptions(t, nil))
	if err != nil {
		t.Fatalf("resolve project: %v", err)
	}
	if project.ModulePath != "example.com/service" {
		t.Fatalf("module path = %q, want example.com/service", project.ModulePath)
	}
	if project.RelativeModuleRoot != "." {
		t.Fatalf("relative module root = %q, want .", project.RelativeModuleRoot)
	}
	if project.Boundary != "go.mod" {
		t.Fatalf("boundary = %q, want go.mod", project.Boundary)
	}
}

func TestResolveProjectRequiresExplicitModuleForMultiModuleWorkspace(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, filepath.Join(root, "go.work"), "go 1.22\nuse (\n  ./service\n  ./shared\n)\n")
	writeFixture(t, filepath.Join(root, "service", "go.mod"), "module example.com/service\n")
	writeFixture(t, filepath.Join(root, "shared", "go.mod"), "module example.com/shared\n")

	_, err := ResolveProject(root, goOptions(t, nil))
	if analysis.ErrorCodeOf(err) != analysis.ErrModuleSelection {
		t.Fatalf("error code = %q, want %q", analysis.ErrorCodeOf(err), analysis.ErrModuleSelection)
	}
	project, err := ResolveProject(root, goOptions(t, map[string]any{"module": "./service"}))
	if err != nil {
		t.Fatalf("resolve explicit module: %v", err)
	}
	if project.ModulePath != "example.com/service" {
		t.Fatalf("module path = %q, want example.com/service", project.ModulePath)
	}
	if project.RelativeModuleRoot != "service" {
		t.Fatalf("relative module root = %q, want service", project.RelativeModuleRoot)
	}
	if project.RelativeWorkspacePath != "go.work" {
		t.Fatalf("relative workspace path = %q, want go.work", project.RelativeWorkspacePath)
	}
}

func TestResolveProjectAcceptsModulePathSelector(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, filepath.Join(root, "go.work"), "go 1.22\nuse (\n  ./service\n  ./shared\n)\n")
	writeFixture(t, filepath.Join(root, "service", "go.mod"), "module example.com/service\n")
	writeFixture(t, filepath.Join(root, "shared", "go.mod"), "module example.com/shared\n")

	project, err := ResolveProject(root, goOptions(t, map[string]any{"module": "example.com/shared"}))
	if err != nil {
		t.Fatalf("resolve module path selector: %v", err)
	}
	if project.ModulePath != "example.com/shared" {
		t.Fatalf("module path = %q, want example.com/shared", project.ModulePath)
	}
}

func TestResolveProjectParsesTabSeparatedUseDirective(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, filepath.Join(root, "go.work"), "go 1.22\nuse\t./service\n")
	writeFixture(t, filepath.Join(root, "service", "go.mod"), "module example.com/service\n")

	project, err := ResolveProject(root, goOptions(t, nil))
	if err != nil {
		t.Fatalf("resolve tab-separated use directive: %v", err)
	}
	if project.ModulePath != "example.com/service" {
		t.Fatalf("module path = %q, want example.com/service", project.ModulePath)
	}
}

func TestResolveProjectRejectsWorkspaceModuleOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(root, "..", "outside-module")
	writeFixture(t, filepath.Join(root, "go.work"), "go 1.22\nuse ../outside-module\n")
	writeFixture(t, filepath.Join(outside, "go.mod"), "module example.com/outside\n")
	t.Cleanup(func() {
		_ = os.RemoveAll(outside)
	})

	_, err := ResolveProject(root, goOptions(t, map[string]any{"module": "./outside-module"}))
	if analysis.ErrorCodeOf(err) != analysis.ErrUnsupportedProject {
		t.Fatalf("error code = %q, want %q", analysis.ErrorCodeOf(err), analysis.ErrUnsupportedProject)
	}
}

func TestGoAnalyzerDetectsMarkersWithoutExecutingCode(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, filepath.Join(root, "go.mod"), "module example.com/service\n")
	analyzer := New()
	candidate, err := analyzer.Detect(context.Background(), analysis.DetectRequest{ProjectRoot: root})
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if candidate.AnalyzerID != analyzer.Manifest().ID || candidate.Confidence != 1 {
		t.Fatalf("candidate = %#v", candidate)
	}
	if len(candidate.MatchedMarkers) != 1 || candidate.MatchedMarkers[0] != "go.mod" {
		t.Fatalf("matched markers = %#v", candidate.MatchedMarkers)
	}
}

func TestGoAnalyzerReturnsPartialForModuleWithoutEligiblePackages(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, filepath.Join(root, "go.mod"), "module example.com/service\n")
	analyzer := New()
	options := goOptions(t, nil)
	result, err := analyzer.Analyze(context.Background(), analysis.AnalyzeRequest{
		ProjectRoot: root,
		Options:     options,
	})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if result.Status != analysis.StatusPartial {
		t.Fatalf("status = %q, want partial", result.Status)
	}
	if result.Project.ModulePath != "example.com/service" {
		t.Fatalf("module path = %q, want example.com/service", result.Project.ModulePath)
	}
	if len(result.Diagnostics) != 1 || !result.Diagnostics[0].Recoverable {
		t.Fatalf("diagnostics = %#v", result.Diagnostics)
	}
}

func TestGoAnalyzerRejectsUnsupportedRoot(t *testing.T) {
	root := t.TempDir()
	candidate, err := New().Detect(context.Background(), analysis.DetectRequest{ProjectRoot: root})
	if err != nil {
		t.Fatalf("detect unsupported root returned error: %v", err)
	}
	if candidate.Confidence != 0 {
		t.Fatalf("confidence = %v, want zero", candidate.Confidence)
	}
	_, err = ResolveProject(root, goOptions(t, nil))
	if analysis.ErrorCodeOf(err) != analysis.ErrUnsupportedProject {
		t.Fatalf("resolve error code = %q, want %q", analysis.ErrorCodeOf(err), analysis.ErrUnsupportedProject)
	}
}
