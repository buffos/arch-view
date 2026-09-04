package contract

import (
	"testing"

	"github.com/buffo/arch-view/internal/okf/ports"
)

func TestExtensionRevisionIncludesDescriptorAndSchema(t *testing.T) {
	value := ports.Extension{ID: "test.rule", Version: "1", Kind: "rule", ParameterSchema: map[string]any{"type": "object", "required": []string{"field"}}}
	first, err := ExtensionCatalogRevision([]any{value})
	if err != nil || first == "" {
		t.Fatalf("revision=%q error=%v", first, err)
	}
	value.ParameterSchema = map[string]any{"required": []string{"field"}, "type": "object"}
	same, err := ExtensionCatalogRevision([]any{value})
	if err != nil || same != first {
		t.Fatalf("object key order changed revision: %q %v", same, err)
	}
	value.ParameterSchema["required"] = []string{"other"}
	changed, err := ExtensionCatalogRevision([]any{value})
	if err != nil || changed == first {
		t.Fatalf("schema change not reflected: %q %v", changed, err)
	}
	value.Version = "2"
	next, err := ExtensionCatalogRevision([]any{value})
	if err != nil || next == changed {
		t.Fatalf("provider version change not reflected: %q %v", next, err)
	}
	value.ParameterSchema["invalid"] = make(chan int)
	if revision, err := ExtensionCatalogRevision([]any{value}); err == nil || revision != "" {
		t.Fatalf("invalid schema produced revision=%q error=%v", revision, err)
	}
}
