package presentation

import "github.com/buffo/arch-view/internal/okf/domain"

type fixedShape struct{ value domain.ShapeDefinition }

func (shape fixedShape) Definition() domain.ShapeDefinition { return shape.value }

func NewDefaultShapeRegistry() *ShapeRegistry {
	registry := NewShapeRegistry()
	for _, value := range []domain.ShapeDefinition{
		{ID: "okf.shape.rectangle", Geometry: "rectangle", Content: domain.ShapeBox{Width: 1, Height: 1}},
		{ID: "okf.shape.rounded_rectangle", Geometry: "rectangle", CornerRadius: 0.1, Content: domain.ShapeBox{X: 0.05, Y: 0.05, Width: 0.9, Height: 0.9}},
		{ID: "okf.shape.pill", Geometry: "rectangle", CornerRadius: 0.5, Content: domain.ShapeBox{X: 0.25, Y: 0.25, Width: 0.5, Height: 0.5}},
		{ID: "okf.shape.ellipse", Geometry: "ellipse", Content: domain.ShapeBox{X: 0.25, Y: 0.25, Width: 0.5, Height: 0.5}},
		{ID: "okf.shape.diamond", Geometry: "polygon", Points: []domain.ShapePoint{{X: 0.5}, {X: 1, Y: 0.5}, {X: 0.5, Y: 1}, {Y: 0.5}}, Content: domain.ShapeBox{X: 0.25, Y: 0.25, Width: 0.5, Height: 0.5}},
		{ID: "okf.shape.hexagon", Geometry: "polygon", Points: []domain.ShapePoint{{X: 0.25}, {X: 0.75}, {X: 1, Y: 0.5}, {X: 0.75, Y: 1}, {X: 0.25, Y: 1}, {Y: 0.5}}, Content: domain.ShapeBox{X: 0.25, Width: 0.5, Height: 1}},
	} {
		value.Version, value.Description = "1", value.ID
		if err := registry.Register(fixedShape{value}); err != nil {
			panic(err)
		}
	}
	return registry
}
