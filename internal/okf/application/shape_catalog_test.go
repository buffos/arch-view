package application

import (
	"context"
	"testing"

	"github.com/buffo/arch-view/internal/okf/contract"
	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
	"github.com/buffo/arch-view/internal/okf/profile"
)

type catalogShape struct{ definition domain.ShapeDefinition }

func (value catalogShape) Definition() domain.ShapeDefinition { return value.definition }

func TestShapeCatalogExposesRegisteredGeometryAndChangesRevision(t *testing.T) {
	registry := profile.NewRegistry()
	service := NewWithDependencies(t.TempDir(), nil, nil, nil, registry)
	before, err := service.Extensions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	beforeRevision, err := contract.ExtensionCatalogRevision(before)
	if err != nil {
		t.Fatal(err)
	}
	definition := domain.ShapeDefinition{ID: "test.catalog", Version: "2", Description: "Catalog triangle", Geometry: "polygon",
		Points: []domain.ShapePoint{{X: 0.5}, {X: 1, Y: 1}, {Y: 1}}, Content: domain.ShapeBox{X: 0.4, Y: 0.5, Width: 0.2, Height: 0.3}}
	if err := registry.RegisterShape(catalogShape{definition}); err != nil {
		t.Fatal(err)
	}
	after, err := service.Extensions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	afterRevision, err := contract.ExtensionCatalogRevision(after)
	if err != nil || beforeRevision == afterRevision {
		t.Fatalf("registry change absent from revision: %v", err)
	}
	var custom *domain.ShapeDefinition
	for _, value := range after {
		extension := value.(ports.Extension)
		if extension.ID == definition.ID {
			custom = extension.ShapeDefinition
		}
	}
	if custom == nil || custom.Version != "2" || len(custom.Points) != 3 {
		t.Fatalf("custom shape not cataloged: %+v", after)
	}
	custom.Points[0].X = 0
	registered, _ := registry.ResolveShape("test.catalog@2")
	if registered.Points[0].X != 0.5 {
		t.Fatal("catalog caller mutated active geometry")
	}
}
