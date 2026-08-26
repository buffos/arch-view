package scene

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
)

func TestBuildScenePreservesTopLevelSemanticsAndAggregation(t *testing.T) {
	value := fixtureModel(t)
	first, err := BuildScene(value, nil, "overview")
	if err != nil {
		t.Fatalf("BuildScene() error = %v", err)
	}
	second, err := BuildScene(value, nil, "overview")
	if err != nil {
		t.Fatalf("second BuildScene() error = %v", err)
	}
	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal first scene: %v", err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("marshal second scene: %v", err)
	}
	if string(firstJSON) != string(secondJSON) {
		t.Fatalf("scene serialization is not deterministic\nfirst:  %s\nsecond: %s", firstJSON, secondJSON)
	}
	if first.SchemaVersion != SceneSchemaVersion || first.ModelID != value.ModelID || first.ModelRevision != value.ModelID {
		t.Fatalf("scene identity = %#v", first)
	}
	if len(first.VisibleNodes) != 2 {
		t.Fatalf("visible nodes = %#v, want local groups only in the default overview", first.VisibleNodes)
	}
	if first.Summary.VisibleRelationshipCount != 2 || first.Summary.CycleCount != 1 || first.Summary.DiagnosticCount != 1 {
		t.Fatalf("scene summary = %#v", first.Summary)
	}
	if first.ReferenceVisibility != ReferenceVisibilityHidden || first.ReferenceSummary.Total != 1 || first.ReferenceSummary.HiddenCount != 1 || len(first.ReferenceDetails) != 1 {
		t.Fatalf("reference boundary summary = %#v details=%#v", first.ReferenceSummary, first.ReferenceDetails)
	}
	if first.ReferenceDetails[0].Scope != "unresolved" || first.ReferenceDetails[0].ConfidenceState != "low" || len(first.ReferenceDetails[0].SourceReferenceIDs) != 1 {
		t.Fatalf("reference detail = %#v", first.ReferenceDetails[0])
	}
	core, ok := findNode(first.VisibleNodes, "group:core")
	if !ok {
		t.Fatalf("core group missing from %#v", first.VisibleNodes)
	}
	if core.Kind != "group" || len(core.ModuleIDs) != 2 || core.CycleState != "cycle" || len(core.Layers) != 2 {
		t.Fatalf("core group semantics = %#v", core)
	}
	if core.Counts.EvidenceCount == 0 || core.Counts.RelationshipCount == 0 {
		t.Fatalf("core group counts = %#v", core.Counts)
	}
	cycleRelation, ok := findRelationship(first.VisibleRelationships, "group:core", "group:core")
	if !ok || cycleRelation.CycleState != "cycle" || !cycleRelation.Directed || cycleRelation.Count != 2 {
		t.Fatalf("aggregated cycle relation = %#v", cycleRelation)
	}
	if len(cycleRelation.RelationshipIDs) != 2 || len(cycleRelation.SourceReferenceIDs) == 0 {
		t.Fatalf("cycle contributors/evidence = %#v", cycleRelation)
	}
	if len(first.CycleIndicators) != 1 || len(first.DiagnosticIndicators) != 1 || len(first.LayerLabels) == 0 {
		t.Fatalf("scene indicators = cycles %#v diagnostics %#v layers %#v", first.CycleIndicators, first.DiagnosticIndicators, first.LayerLabels)
	}
	if len(first.EvidenceLinks) < 3 || len(first.Accessibility.ReadingOrder) == 0 {
		t.Fatalf("scene accessibility/evidence = %#v %#v", first.Accessibility, first.EvidenceLinks)
	}
	for _, evidence := range first.EvidenceLinks {
		if !evidence.ReadOnly {
			t.Fatalf("evidence link is not read-only: %#v", evidence)
		}
	}
}

func TestBuildSceneReferenceVisibilityModes(t *testing.T) {
	value := fixtureModel(t)

	aggregated, err := BuildSceneWithOptions(value, nil, "overview", SceneOptions{ReferenceVisibility: ReferenceVisibilityAggregated})
	if err != nil {
		t.Fatalf("aggregated scene: %v", err)
	}
	if len(aggregated.VisibleNodes) != 3 || aggregated.ReferenceSummary.AggregatedCount != 1 {
		t.Fatalf("aggregated scene = %#v summary=%#v", aggregated.VisibleNodes, aggregated.ReferenceSummary)
	}
	boundary, ok := findNode(aggregated.VisibleNodes, "reference-boundary:unresolved")
	if !ok || boundary.ReferenceScope != "unresolved" || boundary.ConfidenceState != "low" {
		t.Fatalf("aggregated reference boundary = %#v", boundary)
	}
	aggregatedRelation, ok := findRelationship(aggregated.VisibleRelationships, "group:api", "reference-boundary:unresolved")
	if !ok || aggregatedRelation.TargetScope != "unresolved" || aggregatedRelation.ConfidenceState != "low" {
		t.Fatalf("aggregated reference relation = %#v", aggregatedRelation)
	}

	expanded, err := BuildSceneWithOptions(value, nil, "list", SceneOptions{ReferenceVisibility: ReferenceVisibilityExpanded})
	if err != nil {
		t.Fatalf("expanded scene: %v", err)
	}
	if len(expanded.VisibleNodes) != 3 || expanded.ReferenceSummary.ExpandedCount != 1 {
		t.Fatalf("expanded scene = %#v summary=%#v", expanded.VisibleNodes, expanded.ReferenceSummary)
	}
	reference, ok := findNode(expanded.VisibleNodes, "ref-missing")
	if !ok || reference.ReferenceScope != "unresolved" || reference.ConfidenceState != "low" || strings.Contains(reference.AccessibleLabel, "unknown confidence") {
		t.Fatalf("expanded reference = %#v", reference)
	}

	filtered, err := BuildSceneWithOptions(value, nil, "overview", SceneOptions{ReferenceVisibility: ReferenceVisibilityExpanded, ReferenceScopes: []string{"standard_library"}})
	if err != nil {
		t.Fatalf("filtered scene: %v", err)
	}
	if len(filtered.ReferenceDetails) != 0 || filtered.ReferenceSummary.Total != 0 || len(filtered.VisibleNodes) != 2 {
		t.Fatalf("filtered reference scene = %#v summary=%#v", filtered.ReferenceDetails, filtered.ReferenceSummary)
	}
}

func TestBuildSceneSummarizesNonCycleInternalGroupRelationship(t *testing.T) {
	result := analysis.AnalysisResult{
		Status: analysis.StatusComplete,
		Analyzer: analysis.AnalyzerInfo{
			ID:         "org.archview.go",
			Version:    "1.0.0",
			Language:   "go",
			APIVersion: analysis.AnalyzerAPIVersion,
		},
		Project: analysis.ProjectInfo{RootLabel: "fixture", Boundary: "go.mod"},
		Modules: []analysis.ModuleObservation{
			{ID: "go:example.com/app/cmd", Language: "go", Kind: "package", Name: "cmd", DisplayName: "example.com/app/cmd", Hierarchy: []string{"cmd"}, SourceReferenceIDs: []string{"src-cmd"}},
			{ID: "go:example.com/app/internal/a", Language: "go", Kind: "package", Name: "a", DisplayName: "example.com/app/internal/a", Hierarchy: []string{"internal", "a"}, SourceReferenceIDs: []string{"src-a"}},
			{ID: "go:example.com/app/internal/b", Language: "go", Kind: "package", Name: "b", DisplayName: "example.com/app/internal/b", Hierarchy: []string{"internal", "b"}, SourceReferenceIDs: []string{"src-b"}},
		},
		SourceReferences: []analysis.SourceReference{
			{ID: "src-cmd", Path: "cmd/cmd.go", Kind: "file"},
			{ID: "src-a", Path: "internal/a/a.go", Kind: "file"},
			{ID: "src-b", Path: "internal/b/b.go", Kind: "file"},
			{ID: "src-cmd-import", Path: "cmd/cmd.go", Start: &analysis.Position{Line: 4, Column: 2}, Kind: "import"},
			{ID: "src-b-import", Path: "internal/b/b.go", Start: &analysis.Position{Line: 4, Column: 2}, Kind: "import"},
		},
		Relationships: []analysis.RelationshipObservation{
			{ID: "rel-cmd-a", Type: "depends_on", FromModuleID: "go:example.com/app/cmd", ToModuleID: "go:example.com/app/internal/a", SourceReferenceIDs: []string{"src-cmd-import"}, Confidence: &analysis.Confidence{Basis: "resolved", Score: 1}},
			{ID: "rel-b-a", Type: "depends_on", FromModuleID: "go:example.com/app/internal/b", ToModuleID: "go:example.com/app/internal/a", SourceReferenceIDs: []string{"src-b-import"}, Confidence: &analysis.Confidence{Basis: "resolved", Score: 1}},
		},
	}
	value, err := model.Normalize(result)
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	scene, err := BuildScene(value, nil, "overview")
	if err != nil {
		t.Fatalf("BuildScene() error = %v", err)
	}
	internal, ok := findNode(scene.VisibleNodes, "group:internal")
	if !ok {
		t.Fatalf("internal group missing from %#v", scene.VisibleNodes)
	}
	if internal.IdentityState != "stable" || internal.ConfidenceState != "not_applicable" {
		t.Fatalf("internal identity/confidence = %#v", internal)
	}
	if internal.Counts.RelationshipCount != 1 || internal.Counts.InternalRelationshipCount != 1 {
		t.Fatalf("internal relationship counts = %#v", internal.Counts)
	}
	if len(internal.InternalRelationshipIDs) != 1 || internal.InternalRelationshipIDs[0] != "rel-b-a" {
		t.Fatalf("internal relationship IDs = %#v", internal.InternalRelationshipIDs)
	}
	if _, ok := findRelationship(scene.VisibleRelationships, "group:internal", "group:internal"); ok {
		t.Fatal("non-cycle internal relationship was rendered as a self-loop")
	}
	if !strings.Contains(internal.AccessibleLabel, "1 internal relationship(s)") {
		t.Fatalf("internal accessible label = %q", internal.AccessibleLabel)
	}
}

func TestBuildSceneSupportsHierarchyPathAndRejectsUnknownPath(t *testing.T) {
	value := fixtureModel(t)
	scene, err := BuildScene(value, []string{"core"}, "list")
	if err != nil {
		t.Fatalf("BuildScene(core) error = %v", err)
	}
	if scene.DisplayMode != "list" || len(scene.HierarchyPath) != 1 || scene.HierarchyPath[0] != "core" {
		t.Fatalf("selected scene state = %#v", scene)
	}
	if len(scene.VisibleNodes) != 2 {
		t.Fatalf("core projection nodes = %#v", scene.VisibleNodes)
	}
	if _, err := BuildScene(value, []string{"missing"}, "overview"); err == nil {
		t.Fatal("BuildScene(missing) error = nil")
	}
}

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

func findNode(nodes []VisibleNode, id string) (VisibleNode, bool) {
	for _, node := range nodes {
		if node.ID == id {
			return node, true
		}
	}
	return VisibleNode{}, false
}

func findRelationship(relationships []VisibleRelationship, from, to string) (VisibleRelationship, bool) {
	for _, relationship := range relationships {
		if relationship.FromVisibleID == from && relationship.ToVisibleID == to {
			return relationship, true
		}
	}
	return VisibleRelationship{}, false
}
