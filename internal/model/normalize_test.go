package model_test

import (
	"encoding/json"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	modelpkg "github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/model/canonical"
	"github.com/buffo/arch-view/internal/quality"
)

func TestNormalizeProducesDeterministicModelAndCycleProjections(t *testing.T) {
	first := fixtureAnalysisResult()
	second := fixtureAnalysisResult()
	second.Modules[0], second.Modules[1] = second.Modules[1], second.Modules[0]
	second.Relationships[0], second.Relationships[1] = second.Relationships[1], second.Relationships[0]

	left, err := canonical.Normalize(first)
	if err != nil {
		t.Fatalf("normalize first result: %v", err)
	}
	right, err := canonical.Normalize(second)
	if err != nil {
		t.Fatalf("normalize second result: %v", err)
	}
	leftJSON, err := json.Marshal(left)
	if err != nil {
		t.Fatalf("marshal first model: %v", err)
	}
	rightJSON, err := json.Marshal(right)
	if err != nil {
		t.Fatalf("marshal second model: %v", err)
	}
	if string(leftJSON) != string(rightJSON) || left.ModelID != right.ModelID {
		t.Fatalf("models are not deterministic:\n%s\n%s", leftJSON, rightJSON)
	}
	if len(left.Derived.Cycles) != 1 || len(left.Derived.Cycles[0].ModuleIDs) != 2 {
		t.Fatalf("cycles = %#v", left.Derived.Cycles)
	}
	if len(left.Derived.FeedbackRelationshipIDs) == 0 {
		t.Fatal("cycle has no deterministic feedback relationship")
	}
	if len(left.Derived.Layers) == 0 {
		t.Fatal("derived layers are empty")
	}

	projection, err := modelpkg.BuildHierarchyProjection(left, []string{"internal"})
	if err != nil {
		t.Fatalf("build projection: %v", err)
	}
	if len(projection.Nodes) != 2 || len(projection.Relationships) != 2 {
		t.Fatalf("projection = %#v", projection)
	}
	if projection.Nodes[0].Kind != "group" || projection.Nodes[1].Kind != "group" {
		t.Fatalf("projection nodes = %#v", projection.Nodes)
	}
}

func TestQualityReportIsOptionalValidatedModelSibling(t *testing.T) {
	model, err := canonical.Normalize(fixtureAnalysisResult())
	if err != nil {
		t.Fatalf("normalize fixture: %v", err)
	}
	legacyJSON, err := json.Marshal(model)
	if err != nil {
		t.Fatalf("marshal legacy model: %v", err)
	}
	profile := quality.QualityProfile{SchemaVersion: quality.SchemaVersion, ProfileID: "profile:test", ProfileVersion: "1.0.0", SeverityPolicy: quality.TypedConfigBlock{Namespace: "severity:default", SchemaVersion: "1.0.0", Payload: map[string]any{}}, EnabledRules: []quality.RuleBinding{}, Constraints: []quality.ArchitectureConstraint{}, Extensions: []quality.ExtensionBlock{}}
	report, err := quality.EvaluateQualityProfile(profile, quality.EvaluationInput{}, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("evaluate empty quality profile: %v", err)
	}
	withReportResult := fixtureAnalysisResult()
	withReportResult.QualityReport = &report
	modelWithReport, err := canonical.Normalize(withReportResult)
	if err != nil {
		t.Fatalf("normalize model with quality report: %v", err)
	}
	if err := canonical.Validate(modelWithReport); err != nil {
		t.Fatalf("validate model with quality report: %v", err)
	}
	withReportJSON, err := json.Marshal(modelWithReport)
	if err != nil {
		t.Fatalf("marshal model with quality report: %v", err)
	}
	if string(withReportJSON) == string(legacyJSON) {
		t.Fatal("quality report attachment did not change the serialized model")
	}
	withoutReportJSON, err := json.Marshal(model)
	if err != nil {
		t.Fatalf("marshal model after removing quality report: %v", err)
	}
	if string(withoutReportJSON) != string(legacyJSON) {
		t.Fatal("omitted quality report changed the legacy model representation")
	}

	foreignReport, err := quality.EvaluateQualityProfile(profile, quality.EvaluationInput{SourceSnapshots: []quality.SourceSnapshot{{SnapshotID: "foreign-snapshot", ScopeID: "foreign-scope"}}}, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("evaluate foreign quality report: %v", err)
	}
	model.QualityReport = &foreignReport
	if err := canonical.Validate(model); analysis.ErrorCodeOf(err) != analysis.ErrInvalidModel {
		t.Fatalf("foreign quality report error code = %q, want %q", analysis.ErrorCodeOf(err), analysis.ErrInvalidModel)
	}
}

func TestNormalizeMergesEvidenceAndPreservesPartialDiagnostics(t *testing.T) {
	result := fixtureAnalysisResult()
	result.Status = analysis.StatusPartial
	result.Modules = append(result.Modules, analysis.ModuleObservation{
		ID:                 "go:example.com/app/internal/a",
		Language:           "go",
		Kind:               "package",
		Name:               "a",
		DisplayName:        "example.com/app/internal/a",
		Hierarchy:          []string{"internal", "a"},
		SourceReferenceIDs: []string{"src:a-extra"},
		Tags:               []string{"test"},
	})
	result.SourceReferences = append(result.SourceReferences, analysis.SourceReference{ID: "src:a-extra", Path: "internal/a/a_test.go", Kind: "file"})
	result.Diagnostics = []analysis.Diagnostic{{
		Code:        "go_unresolved_import",
		Severity:    "warning",
		Message:     "unresolved import",
		Path:        "internal/a/a.go",
		Location:    &analysis.Position{Line: 4, Column: 2},
		Recoverable: true,
	}}

	model, err := canonical.Normalize(result)
	if err != nil {
		t.Fatalf("normalize partial result: %v", err)
	}
	if model.Status != modelpkg.StatusPartial {
		t.Fatalf("status = %q, want partial", model.Status)
	}
	if len(model.Modules) != 2 || len(model.Modules[0].SourceReferenceIDs) != 2 {
		t.Fatalf("merged modules = %#v", model.Modules)
	}
	if len(model.Diagnostics) != 1 || len(model.Diagnostics[0].SourceReferenceIDs) == 0 {
		t.Fatalf("diagnostics = %#v", model.Diagnostics)
	}
}

func TestNormalizeReportsConflictingDuplicateObservationsAsPartial(t *testing.T) {
	result := fixtureAnalysisResult()
	conflict := result.Modules[0]
	conflict.Name = "different"
	result.Modules = append(result.Modules, conflict)

	normalized, err := canonical.Normalize(result)
	if err != nil {
		t.Fatalf("normalize conflicting result: %v", err)
	}
	if normalized.Status != modelpkg.StatusPartial {
		t.Fatalf("status = %q, want partial", normalized.Status)
	}
	if len(normalized.Diagnostics) != 1 || normalized.Diagnostics[0].Code != "model_conflicting_module" || !normalized.Diagnostics[0].Recoverable {
		t.Fatalf("diagnostics = %#v", normalized.Diagnostics)
	}
	if _, ok := normalized.Diagnostics[0].Metadata["first_observation"]; !ok {
		t.Fatalf("conflict diagnostic did not retain first observation: %#v", normalized.Diagnostics[0].Metadata)
	}
	if _, ok := normalized.Diagnostics[0].Metadata["conflicting_observation"]; !ok {
		t.Fatalf("conflict diagnostic did not retain conflicting observation: %#v", normalized.Diagnostics[0].Metadata)
	}
}

func TestNormalizeSortsDiagnosticsAddedDuringNormalization(t *testing.T) {
	result := fixtureAnalysisResult()
	result.Diagnostics = []analysis.Diagnostic{{
		Code:        "raw_warning",
		Severity:    "warning",
		Message:     "raw warning",
		Recoverable: true,
	}}
	conflict := result.Modules[0]
	conflict.Name = "different"
	result.Modules = append(result.Modules, conflict)

	normalized, err := canonical.Normalize(result)
	if err != nil {
		t.Fatalf("normalize result: %v", err)
	}
	if len(normalized.Diagnostics) != 2 || normalized.Diagnostics[0].ID > normalized.Diagnostics[1].ID {
		t.Fatalf("diagnostics are not sorted: %#v", normalized.Diagnostics)
	}
}

func TestValidateRejectsBrokenModelEndpoint(t *testing.T) {
	model, err := canonical.Normalize(fixtureAnalysisResult())
	if err != nil {
		t.Fatalf("normalize fixture: %v", err)
	}
	model.Relationships[0].ToModuleID = "missing"
	if err := canonical.Validate(model); analysis.ErrorCodeOf(err) != analysis.ErrInvalidModel {
		t.Fatalf("error code = %q, want %q", analysis.ErrorCodeOf(err), analysis.ErrInvalidModel)
	}
}

func TestValidateRejectsStaleModelIDAndUnknownReferenceScope(t *testing.T) {
	model, err := canonical.Normalize(fixtureAnalysisResult())
	if err != nil {
		t.Fatalf("normalize fixture: %v", err)
	}
	model.ModelID = "model-stale"
	if err := canonical.Validate(model); analysis.ErrorCodeOf(err) != analysis.ErrInvalidModel {
		t.Fatalf("stale model id error code = %q, want %q", analysis.ErrorCodeOf(err), analysis.ErrInvalidModel)
	}

	model, err = canonical.Normalize(fixtureAnalysisResult())
	if err != nil {
		t.Fatalf("normalize fixture again: %v", err)
	}
	model.References = append(model.References, analysis.Reference{ID: "ref-invalid", Name: "invalid", Scope: "unknown", Language: "go"})
	if err := canonical.Validate(model); analysis.ErrorCodeOf(err) != analysis.ErrInvalidModel {
		t.Fatalf("unknown reference scope error code = %q, want %q", analysis.ErrorCodeOf(err), analysis.ErrInvalidModel)
	}
}

func fixtureAnalysisResult() analysis.AnalysisResult {
	return analysis.AnalysisResult{
		RunID:  "run-fixture",
		Status: analysis.StatusComplete,
		Analyzer: analysis.AnalyzerInfo{
			ID:         "org.archview.go",
			Version:    "1.0.0",
			Language:   "go",
			APIVersion: analysis.AnalyzerAPIVersion,
		},
		Project: analysis.ProjectInfo{RootLabel: "fixture", Boundary: "go.mod"},
		Modules: []analysis.ModuleObservation{
			{
				ID:                 "go:example.com/app/internal/a",
				Language:           "go",
				Kind:               "package",
				Name:               "a",
				DisplayName:        "example.com/app/internal/a",
				Hierarchy:          []string{"internal", "a"},
				SourceReferenceIDs: []string{"src:a"},
			},
			{
				ID:                 "go:example.com/app/internal/b",
				Language:           "go",
				Kind:               "package",
				Name:               "b",
				DisplayName:        "example.com/app/internal/b",
				Hierarchy:          []string{"internal", "b"},
				SourceReferenceIDs: []string{"src:b"},
			},
		},
		SourceReferences: []analysis.SourceReference{
			{ID: "src:a", Path: "internal/a/a.go", Kind: "file"},
			{ID: "src:b", Path: "internal/b/b.go", Kind: "file"},
			{ID: "src:ab", Path: "internal/a/a.go", Start: &analysis.Position{Line: 4, Column: 2}, End: &analysis.Position{Line: 4, Column: 12}, Kind: "import"},
			{ID: "src:ba", Path: "internal/b/b.go", Start: &analysis.Position{Line: 4, Column: 2}, End: &analysis.Position{Line: 4, Column: 12}, Kind: "import"},
		},
		Relationships: []analysis.RelationshipObservation{
			{ID: "rel-a-b", Type: "depends_on", FromModuleID: "go:example.com/app/internal/a", ToModuleID: "go:example.com/app/internal/b", SourceReferenceIDs: []string{"src:ab"}, Confidence: &analysis.Confidence{Basis: "resolved", Score: 1}},
			{ID: "rel-b-a", Type: "depends_on", FromModuleID: "go:example.com/app/internal/b", ToModuleID: "go:example.com/app/internal/a", SourceReferenceIDs: []string{"src:ba"}, Confidence: &analysis.Confidence{Basis: "resolved", Score: 1}},
		},
	}
}
