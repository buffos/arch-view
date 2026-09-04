package profile

import (
	"testing"

	"github.com/buffo/arch-view/internal/okf/ports"
)

type sharedSchemaRule struct {
	ports.RuleStrategy
	schema map[string]any
}

func (sharedSchemaRule) ID() string                           { return "test.shared_schema" }
func (sharedSchemaRule) Version() string                      { return "1" }
func (sharedSchemaRule) Description() string                  { return "Shared schema ownership regression" }
func (rule sharedSchemaRule) ParameterSchema() map[string]any { return rule.schema }

func TestCatalogCannotMutateProviderSchemaStringLists(t *testing.T) {
	required := []string{"field"}
	enum := []string{"visible", "hidden"}
	registry := NewRegistry()
	if err := registry.Register(sharedSchemaRule{schema: map[string]any{
		"required":   required,
		"properties": map[string]any{"state": map[string]any{"enum": enum}},
	}}); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, extension := range registry.Catalog() {
		if extension.ID != "test.shared_schema" {
			continue
		}
		found = true
		extension.ParameterSchema["required"].([]string)[0] = "mutated"
		extension.ParameterSchema["properties"].(map[string]any)["state"].(map[string]any)["enum"].([]string)[0] = "mutated"
	}
	if !found || required[0] != "field" || enum[0] != "visible" {
		t.Fatal("catalog response mutated provider-owned schema")
	}
}
