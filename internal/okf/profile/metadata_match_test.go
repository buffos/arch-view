package profile

import (
	"context"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestExactRulesPreserveMetadataTypesAndPresence(t *testing.T) {
	for _, tc := range []struct {
		name    string
		source  map[string]any
		want    any
		matched bool
	}{
		{"missing", map[string]any{}, nil, false},
		{"explicit null", map[string]any{"field": nil}, nil, true},
		{"boolean string", map[string]any{"field": "true"}, true, false},
		{"number string", map[string]any{"field": "1"}, float64(1), false},
		{"numeric decoders", map[string]any{"field": 1}, float64(1), true},
		{"array string", map[string]any{"field": []any{"a", "b"}}, "[a b]", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, id := range []string{"okf.rule.metadata_equals", "okf.rule.visibility"} {
				registry := NewRegistry()
				value := domain.Profile{Rules: []domain.RuleInvocation{{RuleID: id, Enabled: true, Parameters: map[string]any{"field": "field", "value": tc.want, "label": "matched", "visible": false}}}}
				result, diagnostics := registry.Evaluate(context.Background(), domain.ConceptDocument{Frontmatter: tc.source}, value)
				matched := result.Label == "matched" || result.Visible != nil
				if len(diagnostics) != 0 || matched != tc.matched {
					t.Fatalf("%s: %#v %#v", id, result, diagnostics)
				}
			}
		})
	}
}

func TestContainsUsesMembershipForCollections(t *testing.T) {
	for _, tc := range []struct {
		source, needle any
		matched        bool
	}{
		{[]any{"unimplemented"}, "implemented", false},
		{[]any{"implemented"}, "implemented", true},
		{[]string{"implemented"}, "implemented", true},
		{[]any{true}, "true", false},
		{[]any{1}, float64(1), true},
		{"Ready for review", "READY", true},
		{map[string]any{"ready": true}, "ready", false},
		{nil, "nil", false},
	} {
		value := domain.Profile{Rules: []domain.RuleInvocation{{RuleID: "okf.rule.metadata_contains", Enabled: true, Parameters: map[string]any{"field": "field", "value": tc.needle, "label": "matched"}}}}
		result, diagnostics := NewRegistry().Evaluate(context.Background(), domain.ConceptDocument{Frontmatter: map[string]any{"field": tc.source}}, value)
		if len(diagnostics) != 0 || (result.Label == "matched") != tc.matched {
			t.Fatalf("source=%#v needle=%#v: %#v %#v", tc.source, tc.needle, result, diagnostics)
		}
	}
}
