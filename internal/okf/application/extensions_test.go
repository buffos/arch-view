package application

import (
	"context"
	"errors"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
	"github.com/buffo/arch-view/internal/okf/profile"
)

type describedRelationshipAdapter struct{ relationshipTestAdapter }

func (describedRelationshipAdapter) Description() string { return "Test semantic links" }
func (describedRelationshipAdapter) ParameterSchema() map[string]any {
	return map[string]any{"type": "object"}
}

type brokenCatalogRule struct {
	ports.RuleStrategy
	failSchema bool
}

func (brokenCatalogRule) ID() string      { return "test.broken_catalog" }
func (brokenCatalogRule) Version() string { return "1" }
func (rule brokenCatalogRule) Description() string {
	if !rule.failSchema {
		panic("description failure")
	}
	return "Broken schema"
}
func (brokenCatalogRule) ParameterSchema() map[string]any { panic("schema failure") }

func TestExtensionCatalogMetadataPanicsReturnStructuredFailure(t *testing.T) {
	for _, failSchema := range []bool{false, true} {
		registry := profile.NewRegistry()
		if err := registry.Register(brokenCatalogRule{failSchema: failSchema}); err != nil {
			t.Fatal(err)
		}
		service := NewWithDependencies(t.TempDir(), nil, nil, nil, registry)
		values, err := service.Extensions(context.Background())
		var failure *domain.Error
		if values != nil || !errors.As(err, &failure) || failure.Code != "okf_extension_catalog_failed" || failure.Status != 500 {
			t.Fatalf("catalog failure=%+v values=%+v", err, values)
		}
		if _, ok := registry.Resolve("okf.rule.metadata_equals", "1"); !ok {
			t.Fatal("catalog failure corrupted rule registry")
		}
	}
}

func TestExtensionCatalogIncludesInjectedRelationshipAdapters(t *testing.T) {
	service := NewWithDependencies(t.TempDir(), nil, nil, nil, nil, nil, describedRelationshipAdapter{})
	values, err := service.Extensions(context.Background())
	if err != nil || len(values) != 13 {
		t.Fatalf("catalog=%+v err=%v", values, err)
	}
	var adapter ports.Extension
	for _, value := range values {
		if entry := value.(ports.Extension); entry.Kind == "relationship_adapter" {
			adapter = entry
		}
	}
	if adapter.Kind != "relationship_adapter" || adapter.ID != "test" || adapter.Version != "1" || adapter.Description != "Test semantic links" || adapter.ParameterSchema["type"] != "object" {
		t.Fatalf("adapter descriptor=%+v", adapter)
	}
	legacy, err := relationshipExtension(relationshipTestAdapter{})
	if err != nil || legacy.ParameterSchema != nil || legacy.Description != "" {
		t.Fatalf("fabricated legacy metadata: %+v %v", legacy, err)
	}
}
