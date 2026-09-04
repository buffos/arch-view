package presentation

import (
	"fmt"
	"math"

	"github.com/buffo/arch-view/internal/okf/domain"
)

const geometryTolerance = 1e-10

var geometryValidators = map[string]func(domain.ShapeDefinition) error{
	"rectangle": validateRectangle,
	"ellipse":   validateEllipse,
	"polygon":   validatePolygon,
}

func contentCorners(box domain.ShapeBox) []domain.ShapePoint {
	return []domain.ShapePoint{{X: box.X, Y: box.Y}, {X: box.X + box.Width, Y: box.Y},
		{X: box.X + box.Width, Y: box.Y + box.Height}, {X: box.X, Y: box.Y + box.Height}}
}

func validateRectangle(value domain.ShapeDefinition) error {
	if len(value.Points) != 0 {
		return fmt.Errorf("rectangle cannot declare polygon points")
	}
	radius := value.CornerRadius
	for _, point := range contentCorners(value.Content) {
		dx := math.Max(0, math.Abs(point.X-0.5)-0.5+radius)
		dy := math.Max(0, math.Abs(point.Y-0.5)-0.5+radius)
		if math.Hypot(dx, dy) > radius+geometryTolerance {
			return fmt.Errorf("content box escapes rounded rectangle")
		}
	}
	return nil
}

func validateEllipse(value domain.ShapeDefinition) error {
	if len(value.Points) != 0 || value.CornerRadius != 0 {
		return fmt.Errorf("ellipse cannot declare points or corner radius")
	}
	for _, point := range contentCorners(value.Content) {
		if math.Hypot((point.X-0.5)*2, (point.Y-0.5)*2) > 1+geometryTolerance {
			return fmt.Errorf("content box escapes ellipse")
		}
	}
	return nil
}

func cross(a, b, c domain.ShapePoint) float64 {
	return (b.X-a.X)*(c.Y-a.Y) - (b.Y-a.Y)*(c.X-a.X)
}

// Strict convexity excludes self-crossing and degenerate outlines and keeps
// ray-to-boundary attachment unambiguous. Either winding order is accepted.
func validatePolygon(value domain.ShapeDefinition) error {
	points := value.Points
	if len(points) < 3 || len(points) > 64 || value.CornerRadius != 0 {
		return fmt.Errorf("polygon requires 3 to 64 points and no corner radius")
	}
	winding := math.Copysign(1, cross(points[0], points[1], points[2]))
	corners := append(contentCorners(value.Content), domain.ShapePoint{X: 0.5, Y: 0.5})
	for i, a := range points {
		j := (i + 1) % len(points)
		b := points[j]
		for k, point := range points {
			if k != i && k != j && winding*cross(a, b, point) <= geometryTolerance {
				return fmt.Errorf("polygon must be nondegenerate and strictly convex")
			}
		}
		for _, point := range corners {
			if winding*cross(a, b, point) < -geometryTolerance {
				return fmt.Errorf("polygon must contain its content box and unit-box center")
			}
		}
	}
	return nil
}
