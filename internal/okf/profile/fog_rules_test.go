package profile

import (
	"context"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
)

func TestRollupVocabularyBelongsToSelectedProfile(t *testing.T) {
	registry := NewRegistry()
	index := domain.BundleIndex{Documents: map[string]domain.ConceptDocument{
		"parent": {ConceptID: "parent", Type: "concept", Frontmatter: map[string]any{
			"state_policy": map[string]any{"mode": "rollup", "reducer": "min"},
			"workflow":     map[string]any{"category": "collection"},
		}},
	}}
	for _, profileID := range []string{DefaultProfileID, FogProfileID} {
		effective, diagnostics := registry.ResolveProfile(profileID)
		if len(diagnostics) != 0 {
			t.Fatal(diagnostics)
		}
		result, notes := registry.Evaluate(context.Background(), index.Documents["parent"], effective)
		if len(notes) != 0 {
			t.Fatal(notes)
		}
		state := ResolveConceptState("parent", index, effective, nil, map[string]ports.RuleResult{"parent": result})
		if state.IsRollup != (profileID == FogProfileID) {
			t.Fatalf("profile %s interpreted source incorrectly: %+v", profileID, state)
		}
	}
	custom, _ := registry.ResolveProfile(DefaultProfileID)
	custom.Rules = []domain.RuleInvocation{{RuleID: "okf.rule.metadata_equals", Version: "1", Enabled: true,
		Parameters: map[string]any{"field": "workflow.category", "value": "collection", "role": "rollup"}}}
	result, notes := registry.Evaluate(context.Background(), index.Documents["parent"], custom)
	if len(notes) != 0 || result.Role != "rollup" {
		t.Fatalf("custom vocabulary not mapped: %+v %+v", result, notes)
	}
}

func TestNestedMetadataKeepsLiteralKeyPrecedence(t *testing.T) {
	document := domain.ConceptDocument{Type: "concept", Frontmatter: map[string]any{
		"workflow": map[string]any{"state": "nested"}, "workflow.state": "literal",
	}}
	if value, _ := metadataField(document, "workflow.state"); value != "literal" {
		t.Fatal(value)
	}
	delete(document.Frontmatter, "workflow.state")
	if value, _ := metadataField(document, "workflow.state"); value != "nested" {
		t.Fatal(value)
	}
	if value := DeclaredState(document, "frontmatter.workflow.state"); value != "nested" {
		t.Fatal(value)
	}
	if _, exists := metadataField(document, "workflow.state.missing"); exists {
		t.Fatal("scalar treated as object")
	}
}
