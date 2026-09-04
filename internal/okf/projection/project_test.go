package projection

import (
	"context"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/profile"
)

func TestBuildSeparatesContainmentAndSemanticRelationshipsAndHonorsDepth(t *testing.T) {
	index := domain.BundleIndex{BundleID: "bundle/.okf", SourceRevision: "source:1", ConceptOrder: []string{"root", "root/child", "root/child/grand"}, Documents: map[string]domain.ConceptDocument{
		"root":             {ConceptID: "root", SourcePath: "root.md", Type: "root", ExplicitChildren: []string{"root/child"}},
		"root/child":       {ConceptID: "root/child", SourcePath: "root/child.md", Type: "child", ExplicitParents: []string{"root"}},
		"root/child/grand": {ConceptID: "root/child/grand", SourcePath: "root/child/grand.md", Type: "grand", ExplicitParents: []string{"root/child"}},
	}, Relationships: []domain.Relationship{{RelationshipID: "semantic:root:root/child:0", Kind: domain.RelationshipSemantic, From: "root", To: "root/child"}}}
	registry := profile.NewRegistry()
	effective, diagnostics := registry.ResolveProfile(profile.DefaultProfileID)
	if len(diagnostics) != 0 {
		t.Fatalf("profile diagnostics = %#v", diagnostics)
	}
	snapshot, err := Build(context.Background(), index, effective, domain.NavigationState{Depth: 1}, registry)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if snapshot.Status != domain.ProjectionReady || len(snapshot.Nodes) != 2 {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	if len(snapshot.Relationships) != 2 {
		t.Fatalf("relationships = %#v", snapshot.Relationships)
	}
	if snapshot.Profile.Layout.Algorithm != "mrtree" {
		t.Fatalf("effective layout = %#v", snapshot.Profile.Layout)
	}
	if snapshot.Relationships[0].Kind == snapshot.Relationships[1].Kind {
		t.Fatalf("relationship kinds were collapsed: %#v", snapshot.Relationships)
	}
	if snapshot.Counts.HiddenNodes != 1 {
		t.Fatalf("hidden nodes = %#v", snapshot.Counts)
	}
}

func TestBuildReturnsTruncationAndCancellationWithoutMutatingIndex(t *testing.T) {
	index := domain.BundleIndex{BundleID: "bundle/.okf", SourceRevision: "source:1", ConceptOrder: []string{"a", "b"}, Documents: map[string]domain.ConceptDocument{"a": {ConceptID: "a", SourcePath: "a.md", Type: "a"}, "b": {ConceptID: "b", SourcePath: "b.md", Type: "b"}}}
	registry := profile.NewRegistry()
	effective, _ := registry.ResolveProfile(profile.DefaultProfileID)
	effective.Navigation.MaxNodes = 1
	snapshot, err := Build(context.Background(), index, effective, domain.NavigationState{Depth: 1}, registry)
	if err != nil || snapshot.Status != domain.ProjectionTruncated || snapshot.Counts.HiddenNodes != 1 {
		t.Fatalf("snapshot = %#v err=%v", snapshot, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Build(ctx, index, effective, domain.NavigationState{Depth: 1}, registry); err == nil {
		t.Fatal("cancelled projection unexpectedly succeeded")
	}
}

func TestBuildAppliesExplicitAggregateRollupWithoutInferringRoleFromChildren(t *testing.T) {
	index := domain.BundleIndex{BundleID: "bundle/.okf", SourceRevision: "source:1", ConceptOrder: []string{"root", "root/child"}, Documents: map[string]domain.ConceptDocument{
		"root":       {ConceptID: "root", SourcePath: "root.md", Type: "area", Frontmatter: map[string]any{"role": "aggregate"}, ExplicitChildren: []string{"root/child"}},
		"root/child": {ConceptID: "root/child", SourcePath: "root/child.md", Type: "concept", Frontmatter: map[string]any{"state": "implemented"}, ExplicitParents: []string{"root"}},
	}}
	registry := profile.NewRegistry()
	effective, diagnostics := registry.ResolveProfile(profile.FogProfileID)
	if len(diagnostics) != 0 {
		t.Fatalf("profile diagnostics = %#v", diagnostics)
	}
	snapshot, err := Build(context.Background(), index, effective, domain.NavigationState{Depth: 2}, registry)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(snapshot.Nodes) != 2 || snapshot.Nodes[0].EffectiveState != "implemented" || snapshot.Nodes[0].Annotations["state_rollup"] != "structural_children" {
		t.Fatalf("rollup nodes = %#v", snapshot.Nodes)
	}
	if snapshot.Nodes[0].PresentationToken != "state.implemented" || snapshot.Nodes[0].Annotations["source_path"] != "root.md" {
		t.Fatalf("rollup presentation = %#v", snapshot.Nodes[0])
	}
}

func TestBuildRecognizesStatePolicyRollupForGenericTypes(t *testing.T) {
	index := domain.BundleIndex{BundleID: "bundle/.okf", SourceRevision: "source:1", ConceptOrder: []string{"root", "root/child"}, Documents: map[string]domain.ConceptDocument{
		"root":       {ConceptID: "root", SourcePath: "root.md", Type: "capability", Frontmatter: map[string]any{"state_policy": map[string]any{"mode": "rollup"}}, ExplicitChildren: []string{"root/child"}},
		"root/child": {ConceptID: "root/child", SourcePath: "child.md", Type: "item", Frontmatter: map[string]any{"state": "specified"}, ExplicitParents: []string{"root"}},
	}}
	registry := profile.NewRegistry()
	effective, _ := registry.ResolveProfile(profile.FogProfileID)
	snapshot, err := Build(context.Background(), index, effective, domain.NavigationState{Depth: 2}, registry)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Nodes) != 2 || !snapshot.Nodes[0].IsRollup || snapshot.Nodes[0].PresentationStyle.Stroke != "#f472b6" {
		t.Fatalf("generic roll-up facts = %#v", snapshot.Nodes)
	}
}

func TestBuildClampsInvalidProfileLimitsToApplicationCaps(t *testing.T) {
	index := domain.BundleIndex{BundleID: "bundle/.okf", SourceRevision: "source:1", ConceptOrder: []string{"root"}, Documents: map[string]domain.ConceptDocument{
		"root": {ConceptID: "root", SourcePath: "root.md", Type: "root"},
	}}
	registry := profile.NewRegistry()
	effective, _ := registry.ResolveProfile(profile.DefaultProfileID)
	effective.Navigation.MaxNodes = profile.HardMaxNodes + 1
	effective.Navigation.MaxRelationships = profile.HardMaxRelations + 1
	snapshot, err := Build(context.Background(), index, effective, domain.NavigationState{Depth: 1}, registry)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Nodes) != 1 || snapshot.Status != domain.ProjectionReady {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	count := 0
	for _, diagnostic := range snapshot.Diagnostics {
		if diagnostic.Code == "okf_limit_invalid" {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("limit diagnostics = %#v", snapshot.Diagnostics)
	}
}

func TestBuildUsesOnlyProfileConfiguredNodeFields(t *testing.T) {
	index := domain.BundleIndex{BundleID: "bundle/.okf", SourceRevision: "source:1", ConceptOrder: []string{"root"}, Documents: map[string]domain.ConceptDocument{
		"root": {ConceptID: "root", SourcePath: "root.md", Title: "Root", Type: "capability", Frontmatter: map[string]any{"state": "implemented", "owner": "platform"}},
	}}
	registry := profile.NewRegistry()
	neutral, _ := registry.ResolveProfile(profile.DefaultProfileID)
	neutralSnapshot, err := Build(context.Background(), index, neutral, domain.NavigationState{Depth: 1}, registry)
	if err != nil {
		t.Fatal(err)
	}
	if len(neutralSnapshot.Nodes) != 1 || len(neutralSnapshot.Nodes[0].PresentationFields) != 0 {
		t.Fatalf("neutral presentation fields = %#v", neutralSnapshot.Nodes[0].PresentationFields)
	}

	custom := neutral
	custom.ProfileID = "project:state-view"
	custom.NodeFields = []domain.NodeField{{Source: "frontmatter.state", Label: "state", MaxLength: 20}, {Source: "frontmatter.owner"}}
	snapshot, err := Build(context.Background(), index, custom, domain.NavigationState{Depth: 1}, registry)
	if err != nil {
		t.Fatal(err)
	}
	fields := snapshot.Nodes[0].PresentationFields
	if len(fields) != 2 || fields[0].Label != "state" || fields[0].Value != "implemented" || fields[1].Label != "Owner" || fields[1].Value != "platform" {
		t.Fatalf("configured presentation fields = %#v", fields)
	}
}

func TestBuildAppliesProfileStructuralDecorationsWithoutReplacingStateStyle(t *testing.T) {
	index := domain.BundleIndex{BundleID: "bundle/.okf", SourceRevision: "source:1", ConceptOrder: []string{"root", "root/child"}, Documents: map[string]domain.ConceptDocument{
		"root":       {ConceptID: "root", SourcePath: "root.md", Type: "area", Frontmatter: map[string]any{"role": "aggregate"}, ExplicitChildren: []string{"root/child"}},
		"root/child": {ConceptID: "root/child", SourcePath: "child.md", Type: "item", Frontmatter: map[string]any{"state": "implemented"}, ExplicitParents: []string{"root"}},
	}}
	registry := profile.NewRegistry()
	effective, _ := registry.ResolveProfile(profile.FogProfileID)
	snapshot, err := Build(context.Background(), index, effective, domain.NavigationState{Depth: 2}, registry)
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.Nodes[0].IsRoot || !snapshot.Nodes[0].IsRollup {
		t.Fatalf("structural facts = %#v", snapshot.Nodes[0])
	}
	if snapshot.Nodes[0].PresentationStyle.Fill != "#4338ca" || snapshot.Nodes[0].PresentationStyle.Text != "#ffffff" || snapshot.Nodes[0].PresentationStyle.Stroke != "#f472b6" || snapshot.Nodes[0].PresentationStyle.StrokeWidth != 3 {
		t.Fatalf("root/roll-up style precedence = %#v", snapshot.Nodes[0].PresentationStyle)
	}
	if snapshot.Nodes[1].PresentationStyle.Fill != "#d9f4df" || snapshot.Nodes[1].IsRoot {
		t.Fatalf("state style or root fact = %#v", snapshot.Nodes[1])
	}
	if len(snapshot.Legend) != 1 || snapshot.Legend[0].Fill != "#d9f4df" {
		t.Fatalf("root decoration contaminated state legend: %#v", snapshot.Legend)
	}
	focused, err := Build(context.Background(), index, effective, domain.NavigationState{FocusRoot: "root/child", Depth: 1}, registry)
	if err != nil {
		t.Fatal(err)
	}
	if len(focused.Legend) != 1 || focused.Legend[0] != snapshot.Legend[0] {
		t.Fatalf("focus changed token legend: %#v", focused.Legend)
	}
}
