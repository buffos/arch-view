package viewer

import (
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
)

func fixtureModel(t *testing.T) model.Model {
	t.Helper()
	position := func(line, column int) *analysis.Position {
		return &analysis.Position{Line: line, Column: column}
	}
	sources := []analysis.SourceReference{
		{ID: "src-api", Path: "api/api.go", Kind: "file"},
		{ID: "src-api-import", Path: "api/api.go", Start: position(4, 8), End: position(4, 28), Symbol: "example.com/app/core", Kind: "import"},
		{ID: "src-core", Path: "core/core.go", Kind: "file"},
		{ID: "src-core-import", Path: "core/core.go", Start: position(4, 8), End: position(4, 35), Symbol: "example.com/app/core/internal", Kind: "import"},
		{ID: "src-internal", Path: "core/internal/internal.go", Kind: "file"},
		{ID: "src-internal-import", Path: "core/internal/internal.go", Start: position(4, 8), End: position(4, 28), Symbol: "example.com/app/core", Kind: "import"},
		{ID: "src-api-external", Path: "api/api.go", Start: position(5, 8), End: position(5, 18), Symbol: "example.com/missing", Kind: "import"},
	}
	result := analysis.AnalysisResult{
		Status: analysis.StatusPartial,
		Analyzer: analysis.AnalyzerInfo{
			ID:         "org.archview.go",
			Version:    "1.0.0",
			Language:   "go",
			APIVersion: analysis.AnalyzerAPIVersion,
		},
		Project: analysis.ProjectInfo{RootLabel: "fixture", Boundary: "go.mod"},
		Modules: []analysis.ModuleObservation{
			{ID: "go:example.com/app/api", Language: "go", Kind: "package", Name: "api", DisplayName: "example.com/app/api", Hierarchy: []string{"api"}, SourceReferenceIDs: []string{"src-api"}},
			{ID: "go:example.com/app/core", Language: "go", Kind: "package", Name: "core", DisplayName: "example.com/app/core", Hierarchy: []string{"core"}, SourceReferenceIDs: []string{"src-core"}},
			{ID: "go:example.com/app/core/internal", Language: "go", Kind: "package", Name: "internal", DisplayName: "example.com/app/core/internal", Hierarchy: []string{"core", "internal"}, SourceReferenceIDs: []string{"src-internal"}},
		},
		References: []analysis.Reference{
			{ID: "ref-missing", Name: "example.com/missing", Scope: "unresolved", Language: "go"},
		},
		SourceReferences: sources,
		Relationships: []analysis.RelationshipObservation{
			{ID: "rel-api-core", Type: "depends_on", FromModuleID: "go:example.com/app/api", ToModuleID: "go:example.com/app/core", SourceReferenceIDs: []string{"src-api-import"}, Confidence: &analysis.Confidence{Basis: "resolved", Score: 1}},
			{ID: "rel-core-internal", Type: "depends_on", FromModuleID: "go:example.com/app/core", ToModuleID: "go:example.com/app/core/internal", SourceReferenceIDs: []string{"src-core-import"}, Confidence: &analysis.Confidence{Basis: "resolved", Score: 1}},
			{ID: "rel-internal-core", Type: "depends_on", FromModuleID: "go:example.com/app/core/internal", ToModuleID: "go:example.com/app/core", SourceReferenceIDs: []string{"src-internal-import"}, Confidence: &analysis.Confidence{Basis: "resolved", Score: 1}},
			{ID: "rel-api-missing", Type: "depends_on", FromModuleID: "go:example.com/app/api", ToReferenceID: "ref-missing", SourceReferenceIDs: []string{"src-api-external"}, Confidence: &analysis.Confidence{Basis: "unresolved", Score: 0.2}},
		},
		Diagnostics: []analysis.Diagnostic{{Code: "go_unresolved_import", Severity: "warning", Message: "unresolved import", Path: "api/api.go", Location: position(5, 8), Recoverable: true}},
	}
	value, err := model.Normalize(result)
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	return value
}
