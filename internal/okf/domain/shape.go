package domain

// ShapeDefinition uses unit-box geometry, never markup or executable code.
type ShapeDefinition struct {
	ID           string       `json:"id"`
	Version      string       `json:"version"`
	Description  string       `json:"description"`
	Geometry     string       `json:"geometry"`
	Points       []ShapePoint `json:"points,omitempty"`
	CornerRadius float64      `json:"corner_radius,omitempty"`
	Content      ShapeBox     `json:"content"`
}

type ShapePoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}
type ShapeBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}
