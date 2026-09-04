package profile

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/buffo/arch-view/internal/okf/ports"
)

func TestBuiltInCatalogPublishesIndependentParameterSchemas(t *testing.T) {
	registry := NewRegistry()
	catalog := registry.Catalog()
	if len(catalog) != 5 {
		t.Fatalf("builtins=%d", len(catalog))
	}
	for _, extension := range catalog {
		if extension.ParameterSchema["type"] != "object" || extension.Version == "" {
			t.Fatalf("missing versioned parameter schema: %+v", extension)
		}
		if _, err := json.Marshal(extension); err != nil {
			t.Fatal(err)
		}
		properties := extension.ParameterSchema["properties"].(map[string]any)
		if len(properties) == 0 {
			t.Fatalf("empty properties for %s", extension.ID)
		}
		properties["injected"] = true
	}
	for _, extension := range registry.Catalog() {
		if extension.ParameterSchema["properties"].(map[string]any)["injected"] != nil {
			t.Fatal("catalog caller changed provider schema")
		}
	}
}

type reentrantCatalogStrategy struct {
	ports.RuleStrategy
	registry *Registry
}

func (reentrantCatalogStrategy) ID() string { return "test.reentrant_catalog" }
func (strategy reentrantCatalogStrategy) Description() string {
	// Register takes the write lock. Extension callbacks must run unlocked.
	_ = strategy.registry.Register(metadataEqualsStrategy{})
	return "Reentrant metadata"
}

func TestCatalogDoesNotHoldRegistryLockDuringExtensionMetadata(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(reentrantCatalogStrategy{RuleStrategy: metadataEqualsStrategy{}, registry: registry}); err != nil {
		t.Fatal(err)
	}
	done := make(chan []ports.Extension, 1)
	go func() { done <- registry.Catalog() }()
	select {
	case catalog := <-done:
		if len(catalog) != 6 || catalog[5].Description != "Reentrant metadata" {
			t.Fatalf("catalog=%+v", catalog)
		}
	case <-time.After(time.Second):
		t.Fatal("extension metadata deadlocked on registry lock")
	}
}
