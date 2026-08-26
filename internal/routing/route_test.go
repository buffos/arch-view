package routing

import (
	"math"
	"testing"
)

func TestOrthogonalRouterPreservesHorizontalRoute(t *testing.T) {
	from := NodeBox{X: 40, Y: 42, Width: 190, Height: 82}
	to := NodeBox{X: 314, Y: 42, Width: 190, Height: 82}
	got := (OrthogonalRouter{}).Build(from, to)
	want := []Point{{X: 230, Y: 83}, {X: 272, Y: 83}, {X: 314, Y: 83}}
	assertPoints(t, got.PolylinePoints(), want)
	if got.Kind != RouteKindOrthogonal {
		t.Fatalf("route kind = %q, want %q", got.Kind, RouteKindOrthogonal)
	}
}

func TestOrthogonalRouterPreservesVerticalRoute(t *testing.T) {
	from := NodeBox{X: 40, Y: 42, Width: 190, Height: 82}
	to := NodeBox{X: 40, Y: 240, Width: 190, Height: 82}
	got := (OrthogonalRouter{}).Build(from, to)
	want := []Point{{X: 135, Y: 124}, {X: 135, Y: 182}, {X: 135, Y: 240}}
	assertPoints(t, got.PolylinePoints(), want)
}

func TestPolylineRemovesInvalidAndAdjacentDuplicatePoints(t *testing.T) {
	route := Polyline([]Point{{X: 1, Y: 2}, {X: 1, Y: 2}, {X: 3, Y: 4}, {X: math.NaN(), Y: 1}}, Point{})
	assertPoints(t, route.PolylinePoints(), []Point{{X: 1, Y: 2}, {X: 3, Y: 4}})
}

func TestSelfLoopUsesExistingCubicGeometry(t *testing.T) {
	route := SelfLoop(NodeBox{X: 10, Y: 20, Width: 190, Height: 82})
	if route.Kind != RouteKindSelfLoop || len(route.Sections) != 1 || len(route.Sections[0].Segments) != 1 {
		t.Fatalf("unexpected self-loop route: %#v", route)
	}
	segment := route.Sections[0].Segments[0]
	if segment.Kind != SegmentKindCubic || segment.Control1 == nil || segment.Control2 == nil {
		t.Fatalf("self-loop segment = %#v", segment)
	}
	if route.PolylinePoints() != nil {
		t.Fatal("cubic self-loop should not be treated as a polyline")
	}
}

func assertPoints(t *testing.T, got, want []Point) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("points = %#v, want %#v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("point[%d] = %#v, want %#v", index, got[index], want[index])
		}
	}
}
