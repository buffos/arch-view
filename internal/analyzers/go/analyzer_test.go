package goanalyzer

import (
	"context"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
)

func TestManifestMatchesGoContract(t *testing.T) {
	manifest := New().Manifest()
	if err := analysis.ValidateManifest(manifest); err != nil {
		t.Fatalf("validate manifest: %v", err)
	}
	if manifest.ID != "org.archview.go" || manifest.Language != "go" {
		t.Fatalf("manifest identity = %#v", manifest)
	}
	if len(manifest.Options) != 7 {
		t.Fatalf("option count = %d, want 7", len(manifest.Options))
	}
}

func TestAnalyzeRejectsAmbiguousWorkspace(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root+"/go.work", "go 1.22\nuse (\n ./one\n ./two\n)\n")
	writeFixture(t, root+"/one/go.mod", "module example.com/one\n")
	writeFixture(t, root+"/two/go.mod", "module example.com/two\n")
	_, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{
		ProjectRoot: root,
		Options:     goOptions(t, nil),
	})
	if analysis.ErrorCodeOf(err) != analysis.ErrModuleSelection {
		t.Fatalf("error code = %q, want %q", analysis.ErrorCodeOf(err), analysis.ErrModuleSelection)
	}
}
