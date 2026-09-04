package presentation

import (
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestShapeGeometryRejectsUnusableOutlinesAndEscapingText(t *testing.T) {
	for name, change := range map[string]func(*domain.ShapeDefinition){
		"collinear": func(v *domain.ShapeDefinition) { v.Points = []domain.ShapePoint{{}, {X: 0.5, Y: 0.5}, {X: 1, Y: 1}} },
		"crossing":  func(v *domain.ShapeDefinition) { v.Points = []domain.ShapePoint{{}, {X: 1, Y: 1}, {X: 1}, {Y: 1}} },
		"duplicate": func(v *domain.ShapeDefinition) { v.Points[1] = v.Points[0] },
		"concave": func(v *domain.ShapeDefinition) {
			v.Points = []domain.ShapePoint{{}, {X: 1}, {X: 0.4, Y: 0.4}, {X: 1, Y: 1}, {Y: 1}}
		},
		"polygon text": func(v *domain.ShapeDefinition) { v.Content = domain.ShapeBox{Width: 0.5, Height: 0.5} },
		"ellipse text": func(v *domain.ShapeDefinition) {
			v.Geometry = "ellipse"
			v.Points = nil
			v.Content = domain.ShapeBox{Width: 1, Height: 1}
		},
		"rounded text": func(v *domain.ShapeDefinition) {
			v.Geometry = "rectangle"
			v.Points = nil
			v.CornerRadius = 0.5
			v.Content = domain.ShapeBox{Width: 1, Height: 1}
		},
		"irrelevant points": func(v *domain.ShapeDefinition) { v.Geometry = "ellipse" },
	} {
		t.Run(name, func(t *testing.T) {
			value := testShape()
			change(&value)
			registry := NewShapeRegistry()
			if err := registry.Register(shapeProvider{value}); err == nil {
				t.Fatalf("accepted unusable shape: %+v", value)
			}
			if len(registry.Catalog()) != 0 {
				t.Fatal("invalid geometry was published")
			}
		})
	}
}

func TestShapeGeometryAcceptsBothWindingsAndInscribedText(t *testing.T) {
	value := testShape()
	for range 2 {
		if err := validateShape(value); err != nil {
			t.Fatal(err)
		}
		value.Points[0], value.Points[2] = value.Points[2], value.Points[0]
	}
	for _, geometry := range []string{"rectangle", "ellipse"} {
		value.Geometry, value.Points = geometry, nil
		value.Content = domain.ShapeBox{X: 0.25, Y: 0.25, Width: 0.5, Height: 0.5}
		if err := validateShape(value); err != nil {
			t.Fatal(err)
		}
	}
}
