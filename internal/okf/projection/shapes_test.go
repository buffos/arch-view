package projection

import (
	"context"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/profile"
)

type projectionShapeProvider struct{}

func (projectionShapeProvider) Definition() domain.ShapeDefinition {
	return domain.ShapeDefinition{ID: "test.triangle", Version: "2", Description: "Triangle", Geometry: "polygon",
		Points: []domain.ShapePoint{{X: 0.5}, {X: 1, Y: 1}, {Y: 1}}, Content: domain.ShapeBox{X: 0.4, Y: 0.5, Width: 0.2, Height: 0.3}}
}

func TestProjectionResolvesRegisteredShapeAndPreservesSnapshotIsolation(t *testing.T) {
	registry := profile.NewRegistry()
	if err := registry.RegisterShape(projectionShapeProvider{}); err != nil {
		t.Fatal(err)
	}
	index := domain.BundleIndex{BundleID: "test", ConceptOrder: []string{"root"}, Documents: map[string]domain.ConceptDocument{"root": {ConceptID: "root", SourcePath: "root.md"}}}
	effective, _ := registry.ResolveProfile(profile.DefaultProfileID)
	token := effective.Style.Tokens[effective.Style.DefaultToken]
	token.Shape = "test.triangle@2"
	effective.Style.Tokens[effective.Style.DefaultToken] = token
	snapshot, err := Build(context.Background(), index, effective, domain.NavigationState{Depth: 1}, registry)
	if err != nil || len(snapshot.Nodes) != 1 {
		t.Fatalf("projection=%+v err=%v", snapshot, err)
	}
	definition := snapshot.Nodes[0].ShapeDefinition
	if definition == nil || definition.ID != "test.triangle" || definition.Version != "2" {
		t.Fatalf("missing provider definition: %+v", definition)
	}
	cloned := domain.CloneSnapshot(snapshot)
	cloned.Nodes[0].ShapeDefinition.Points[0].X = 0
	if definition.Points[0].X != 0.5 {
		t.Fatal("snapshot clone shares shape points")
	}
	definition.Points[0].X = 0
	registered, _ := registry.ResolveShape("test.triangle@2")
	if registered.Points[0].X != 0.5 {
		t.Fatal("projection mutated registry")
	}
	token.Shape = "missing.shape@1"
	effective.Style.Tokens[effective.Style.DefaultToken] = token
	fallback, err := Build(context.Background(), index, effective, domain.NavigationState{Depth: 1}, registry)
	if err != nil || fallback.Nodes[0].Shape != "rounded_rectangle" {
		t.Fatalf("fallback=%+v err=%v", fallback, err)
	}
	found := false
	for _, diagnostic := range fallback.Diagnostics {
		found = found || diagnostic.Code == "okf_shape_unsupported" && diagnostic.ConceptID == "root"
	}
	if !found {
		t.Fatal("unknown shape silently replaced")
	}
}
