package presentation

import (
	"math"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

type shapeProvider struct{ value domain.ShapeDefinition }

func (provider shapeProvider) Definition() domain.ShapeDefinition { return provider.value }

type panicProvider struct{}

func (panicProvider) Definition() domain.ShapeDefinition { panic("bad provider") }

func testShape() domain.ShapeDefinition {
	return domain.ShapeDefinition{ID: "test.triangle", Version: "1", Description: "Triangle",
		Geometry: "polygon", Points: []domain.ShapePoint{{X: 0.5}, {X: 1, Y: 1}, {Y: 1}},
		Content: domain.ShapeBox{X: 0.4, Y: 0.5, Width: 0.2, Height: 0.3}}
}

func TestShapeRegistrationSnapshotsDefinition(t *testing.T) {
	registry := NewShapeRegistry()
	value := testShape()
	if err := registry.Register(shapeProvider{value}); err != nil {
		t.Fatal(err)
	}
	value.Points[0].X = 0
	actual, ok := registry.Resolve(value.ID, value.Version)
	if !ok || actual.Points[0].X != 0.5 {
		t.Fatalf("provider mutated registered definition: %+v", actual)
	}
	actual.Points[0].X = 0
	catalog := registry.Catalog()
	if len(catalog) != 1 || catalog[0].Points[0].X != 0.5 {
		t.Fatal(catalog)
	}
	catalog[0].Points[0].X = 0
	actual, _ = registry.Resolve(value.ID, value.Version)
	if actual.Points[0].X != 0.5 {
		t.Fatal("catalog mutation leaked")
	}
	if err := registry.Register(shapeProvider{testShape()}); err == nil {
		t.Fatal("duplicate registration accepted")
	}
	value = testShape()
	value.Version = "2"
	if err := registry.Register(shapeProvider{value}); err != nil {
		t.Fatal(err)
	}
	if len(registry.Catalog()) != 2 {
		t.Fatal("versioned registration lost")
	}
}

func TestShapeRegistrationRejectsMalformedAndPanickingProviders(t *testing.T) {
	registry := NewShapeRegistry()
	if err := registry.Register(panicProvider{}); err == nil {
		t.Fatal("panic escaped boundary")
	}
	for _, mutate := range []func(*domain.ShapeDefinition){
		func(v *domain.ShapeDefinition) { v.Geometry = "<script>" },
		func(v *domain.ShapeDefinition) { v.Content.Width = 0 },
		func(v *domain.ShapeDefinition) { v.Content.Width = math.SmallestNonzeroFloat64 },
		func(v *domain.ShapeDefinition) { v.Content.Height = math.SmallestNonzeroFloat64 },
		func(v *domain.ShapeDefinition) { v.Points[0].X = math.NaN() },
		func(v *domain.ShapeDefinition) { v.Points = nil },
		func(v *domain.ShapeDefinition) { v.CornerRadius = math.Inf(1) },
	} {
		value := testShape()
		mutate(&value)
		if err := registry.Register(shapeProvider{value}); err == nil {
			t.Fatalf("invalid shape accepted: %+v", value)
		}
	}
	if len(registry.Catalog()) != 0 {
		t.Fatal("failed registration published data")
	}
}

func TestShapeRegistrationRejectsAmbiguousReferences(t *testing.T) {
	for _, identity := range [][2]string{
		{"test.triangle@2", "1"}, {"test.triangle", "2@1"},
		{".triangle", "1"}, {"test.", "1"}, {"test..triangle", "1"},
		{"test. triangle", "1"}, {"test.triangle", " 1"},
		{"test.triangle\x00", "1"}, {"test.triangle", "1\n"},
	} {
		t.Run(identity[0]+"@"+identity[1], func(t *testing.T) {
			registry := NewShapeRegistry()
			value := testShape()
			value.ID, value.Version = identity[0], identity[1]
			if err := registry.Register(shapeProvider{value}); err == nil {
				t.Fatal("ambiguous or malformed identity accepted")
			}
			if len(registry.Catalog()) != 0 {
				t.Fatal("invalid registration changed catalog")
			}
		})
	}
	registry := NewShapeRegistry()
	value := testShape()
	value.ID, value.Version = "vendor.custom-shape", "2.1-beta+build"
	if err := registry.Register(shapeProvider{value}); err != nil {
		t.Fatal(err)
	}
	if _, ok := registry.Resolve(value.ID, value.Version); !ok {
		t.Fatal("valid versioned identity did not round trip")
	}
}

func TestShapeContentPrecisionBoundary(t *testing.T) {
	for _, extent := range []float64{math.Nextafter(0x1p-52, 0), 0x1p-52} {
		value := testShape()
		value.Geometry, value.Points = "rectangle", nil
		value.Content = domain.ShapeBox{Width: extent, Height: extent}
		err := NewShapeRegistry().Register(shapeProvider{value})
		if (err == nil) != (extent == 0x1p-52) {
			t.Fatalf("unexpected registration at precision boundary %g: %v", extent, err)
		}
	}
}
