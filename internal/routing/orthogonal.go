package routing

// OrthogonalRouter preserves the deterministic obstacle-free route used for
// manual positions and the export fallback.
type OrthogonalRouter struct{}

func (OrthogonalRouter) Build(from, to NodeBox) Route {
	fromCenterX := from.X + from.Width/2
	fromCenterY := from.Y + from.Height/2
	toCenterX := to.X + to.Width/2
	toCenterY := to.Y + to.Height/2
	points := make([]Point, 0, 4)
	if abs(toCenterX-fromCenterX) >= abs(toCenterY-fromCenterY) {
		forward := toCenterX >= fromCenterX
		sourceX := from.X
		targetX := to.X + to.Width
		if forward {
			sourceX = from.X + from.Width
			targetX = to.X
		}
		middleX := (sourceX + targetX) / 2
		points = append(points,
			Point{X: sourceX, Y: fromCenterY},
			Point{X: middleX, Y: fromCenterY},
			Point{X: middleX, Y: toCenterY},
			Point{X: targetX, Y: toCenterY},
		)
	} else {
		forward := toCenterY >= fromCenterY
		sourceY := from.Y
		targetY := to.Y + to.Height
		if forward {
			sourceY = from.Y + from.Height
			targetY = to.Y
		}
		middleY := (sourceY + targetY) / 2
		points = append(points,
			Point{X: fromCenterX, Y: sourceY},
			Point{X: fromCenterX, Y: middleY},
			Point{X: toCenterX, Y: middleY},
			Point{X: toCenterX, Y: targetY},
		)
	}
	middle := points[len(points)/2]
	route := Polyline(points, Point{X: middle.X, Y: middle.Y - 7})
	route.Kind = RouteKindOrthogonal
	return route
}

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
