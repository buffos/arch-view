// Package routing contains renderer-neutral edge route primitives.
//
// The package deliberately knows nothing about SVG, HTML, ELK, or the
// canonical architecture model. Renderers and layout adapters translate their
// own output into Route values and serialize them at their boundary.
package routing

import "math"

type RouteKind string

const (
	RouteKindPolyline   RouteKind = "polyline"
	RouteKindOrthogonal RouteKind = "orthogonal"
	RouteKindSelfLoop   RouteKind = "self-loop"
)

type SegmentKind string

const (
	SegmentKindLine  SegmentKind = "line"
	SegmentKindCubic SegmentKind = "cubic"
)

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type NodeBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type RouteSegment struct {
	Kind     SegmentKind `json:"kind"`
	To       Point       `json:"to"`
	Control1 *Point      `json:"control1,omitempty"`
	Control2 *Point      `json:"control2,omitempty"`
}

type RouteSection struct {
	Start    Point          `json:"start"`
	Segments []RouteSegment `json:"segments"`
}

type Route struct {
	Kind     RouteKind      `json:"kind"`
	Sections []RouteSection `json:"sections"`
	Label    Point          `json:"label"`
}

// Router is the replaceable strategy boundary for deterministic route
// generation. Layout engines may provide their own route adapters without
// changing graph renderers or exporters.
type Router interface {
	Build(from, to NodeBox) Route
}

// Polyline creates a route from an ordered list of points. Duplicate adjacent
// points are removed so callers can safely combine layout sections and bend
// points without creating zero-length segments.
func Polyline(points []Point, label Point) Route {
	normalized := uniqueAdjacentPoints(points)
	if len(normalized) < 2 {
		return Route{Kind: RouteKindPolyline, Label: label}
	}
	segments := make([]RouteSegment, 0, len(normalized)-1)
	for _, point := range normalized[1:] {
		segments = append(segments, RouteSegment{Kind: SegmentKindLine, To: point})
	}
	return Route{
		Kind:     RouteKindPolyline,
		Sections: []RouteSection{{Start: normalized[0], Segments: segments}},
		Label:    label,
	}
}

// SelfLoop returns the established self-loop geometry used for a relationship
// whose visible source and target are the same node. General spline routing is
// intentionally not introduced here; this cubic segment only preserves the
// existing self-loop presentation.
func SelfLoop(box NodeBox) Route {
	x := box.X + box.Width/2
	start := Point{X: x, Y: box.Y}
	end := Point{X: x, Y: box.Y + box.Height}
	control1 := Point{X: x + 100, Y: box.Y - 55}
	control2 := Point{X: x + 100, Y: box.Y + box.Height + 55}
	return Route{
		Kind: RouteKindSelfLoop,
		Sections: []RouteSection{{
			Start: start,
			Segments: []RouteSegment{{
				Kind:     SegmentKindCubic,
				To:       end,
				Control1: &control1,
				Control2: &control2,
			}},
		}},
		Label: Point{X: x + 50, Y: box.Y + box.Height/2},
	}
}

// PolylinePoints returns the route's points when every segment is linear.
// A nil result means that the route needs a segment-aware path serializer.
func (route Route) PolylinePoints() []Point {
	if len(route.Sections) == 0 {
		return nil
	}
	points := make([]Point, 0)
	for _, section := range route.Sections {
		if len(section.Segments) == 0 {
			if len(points) > 0 {
				return nil
			}
			points = append(points, section.Start)
			continue
		}
		if len(points) == 0 {
			points = append(points, section.Start)
		} else if points[len(points)-1] != section.Start {
			return nil
		}
		for _, segment := range section.Segments {
			if segment.Kind != SegmentKindLine {
				return nil
			}
			points = append(points, segment.To)
		}
	}
	return uniqueAdjacentPoints(points)
}

func uniqueAdjacentPoints(points []Point) []Point {
	result := make([]Point, 0, len(points))
	for _, point := range points {
		if !finitePoint(point) {
			continue
		}
		if len(result) > 0 && result[len(result)-1] == point {
			continue
		}
		result = append(result, point)
	}
	return result
}

func finitePoint(point Point) bool {
	return math.IsNaN(point.X) == false && math.IsNaN(point.Y) == false && math.IsInf(point.X, 0) == false && math.IsInf(point.Y, 0) == false
}
