package adapter_test

import (
	"context"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/sourceindex"
	gosyntax "github.com/buffo/arch-view/internal/analysis/syntax/go"
	"github.com/buffo/arch-view/internal/analyzers/go/sourcefacts"
	"github.com/buffo/arch-view/internal/quality/adapter"
)

func TestEvaluationInputUsesAggregateProjectionIdentity(t *testing.T) {
	index, _, err := sourceindex.BuildSourceIndex(context.Background(), sourceindex.BuildInput{
		Scope:                 analysis.ScopeContext{ScopeID: "scope-a", ProjectRoot: ".", Mode: analysis.SourceIndexScopeMode},
		Producer:              analysis.ProducerContext{AnalyzerID: "org.archview.go", AnalyzerVersion: "1.0.0"},
		Files:                 []sourceindex.SourceFileInput{{Path: "main.go", Content: []byte("package main\n\nfunc main() {}\n"), Language: analysis.LanguageRef{ID: "language:go"}, Roles: []string{"role:source"}, ModuleID: "example.com/app"}},
		RequestedCapabilities: []string{sourceindex.CapabilityFiles, sourceindex.CapabilitySize},
		SyntaxProvider:        gosyntax.NewProvider(),
		Extractors:            sourcefacts.NewRegistry(),
	})
	if err != nil {
		t.Fatalf("build source index: %v", err)
	}
	projection, err := sourceindex.BuildCombinedProjection(index.Snapshots)
	if err != nil {
		t.Fatalf("build combined projection: %v", err)
	}
	index.Projection = projection
	input, err := adapter.EvaluationInputFromSourceIndex(index)
	if err != nil {
		t.Fatalf("adapt aggregate source index: %v", err)
	}
	if len(input.SourceSnapshots) != 1 || input.SourceSnapshots[0].SnapshotID != projection.SnapshotID {
		t.Fatalf("quality snapshots = %#v, want only projection %q", input.SourceSnapshots, projection.SnapshotID)
	}
	if len(input.SourceSnapshots[0].Files) != 1 || input.SourceSnapshots[0].Files[0].ID != projection.Files[0].ID {
		t.Fatalf("quality file identity = %#v, want projection file %q", input.SourceSnapshots[0].Files, projection.Files[0].ID)
	}
}
