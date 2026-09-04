package presentation

// ShapeDefinitionSchema describes the declarative wire format. Registration
// additionally checks convexity and content containment across coordinates.
// Each call owns its maps so catalog consumers cannot mutate future responses.
func ShapeDefinitionSchema() map[string]any {
	unit := func() map[string]any {
		return map[string]any{"type": "number", "minimum": 0, "maximum": 1}
	}
	positiveUnit := func() map[string]any {
		return map[string]any{"type": "number", "minimum": minimumContentExtent, "maximum": 1}
	}
	object := func(properties map[string]any, required ...string) map[string]any {
		return map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
	}
	point := object(map[string]any{"x": unit(), "y": unit()}, "x", "y")
	content := object(map[string]any{"x": unit(), "y": unit(), "width": positiveUnit(), "height": positiveUnit()}, "x", "y", "width", "height")
	content["description"] = "Must lie within the unit box and rendered outline; checked during registration."
	schema := object(map[string]any{
		"id":            map[string]any{"type": "string", "minLength": 3, "description": "Namespaced ID with nonempty dot-separated segments; no whitespace, control characters, or @."},
		"version":       map[string]any{"type": "string", "minLength": 1, "description": "Reference version; no whitespace, control characters, or @."},
		"description":   map[string]any{"type": "string", "pattern": `\S`},
		"geometry":      map[string]any{"type": "string", "enum": []string{"rectangle", "ellipse", "polygon"}},
		"points":        map[string]any{"type": "array", "minItems": 3, "maxItems": 64, "items": point, "description": "Polygon only. Must be strictly convex and contain the unit-box center and content box."},
		"corner_radius": map[string]any{"type": "number", "minimum": 0, "maximum": 0.5, "description": "Rectangle only; omitted or zero for other geometries."},
		"content":       content,
	}, "id", "version", "description", "geometry", "content")
	schema["$schema"] = "https://json-schema.org/draft/2020-12/schema"
	schema["$id"] = "urn:arch-view:okf:shape-definition:1"
	schema["oneOf"] = []any{
		map[string]any{"properties": map[string]any{"geometry": map[string]any{"const": "rectangle"}}, "not": map[string]any{"required": []string{"points"}}},
		map[string]any{"properties": map[string]any{"geometry": map[string]any{"const": "ellipse"}, "corner_radius": map[string]any{"const": 0}}, "not": map[string]any{"required": []string{"points"}}},
		map[string]any{"properties": map[string]any{"geometry": map[string]any{"const": "polygon"}, "corner_radius": map[string]any{"const": 0}}, "required": []string{"points"}},
	}
	return schema
}
