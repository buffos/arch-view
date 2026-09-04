package profile

import "testing"

type detailMetadataFixture struct {
	commonMarkRenderer
	description string
	schema      map[string]any
}

func (detailMetadataFixture) ID() string                              { return "test.detail.metadata" }
func (fixture detailMetadataFixture) Description() string             { return fixture.description }
func (fixture detailMetadataFixture) ParameterSchema() map[string]any { return fixture.schema }

func TestDetailRegistrationRequiresCompleteOwnedMetadata(t *testing.T) {
	for _, fixture := range []detailMetadataFixture{
		{schema: map[string]any{"type": "object"}},
		{description: "Missing schema"},
	} {
		registry := NewRegistry()
		if err := registry.RegisterDetailRenderer(fixture); err == nil {
			t.Fatal("incomplete descriptor accepted")
		}
		if _, exists := registry.ResolveDetailRenderer(fixture.ID(), "1"); exists {
			t.Fatal("failed registration was published")
		}
	}
	registry := NewRegistry()
	nested := map[string]string{"type": "string"}
	fixture := detailMetadataFixture{description: "Owned schema", schema: map[string]any{"type": "object", "properties": map[string]any{"label": nested}}}
	if err := registry.RegisterDetailRenderer(fixture); err != nil {
		t.Fatal(err)
	}
	nested["type"] = "mutated"
	found := false
	for _, entry := range registry.DetailRendererCatalog() {
		if entry.ID == fixture.ID() {
			found = true
			properties := entry.ParameterSchema["properties"].(map[string]any)
			if field, ok := properties["label"].(map[string]any); !ok || field["type"] != "string" {
				t.Fatalf("provider retains catalog ownership: %+v", properties)
			}
		}
	}
	if !found {
		t.Fatal("successful registration missing from catalog")
	}
}
