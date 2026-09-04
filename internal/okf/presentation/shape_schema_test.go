package presentation

import (
	"encoding/json"
	"testing"
)

func TestShapeSchemaOwnsNestedMapsAndHasGeometryVariants(t *testing.T) {
	first := ShapeDefinitionSchema()
	original, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	properties := first["properties"].(map[string]any)
	properties["content"].(map[string]any)["properties"].(map[string]any)["width"].(map[string]any)["maximum"] = 99
	variants := first["oneOf"].([]any)
	if len(variants) != 3 {
		t.Fatal("missing geometry variants")
	}
	variants[0].(map[string]any)["properties"].(map[string]any)["geometry"].(map[string]any)["const"] = "invalid"
	second, err := json.Marshal(ShapeDefinitionSchema())
	if err != nil || string(original) != string(second) {
		t.Fatalf("caller changed future schema: %v", err)
	}
}
