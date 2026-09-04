package profile

import (
	"context"
	"reflect"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
)

type compositionRule struct {
	id     string
	result ports.RuleResult
}

func (rule compositionRule) ID() string     { return rule.id }
func (compositionRule) Version() string     { return "1" }
func (compositionRule) Description() string { return "Composition acceptance fixture" }
func (rule compositionRule) Evaluate(context.Context, domain.ConceptDocument, domain.RuleInvocation) (ports.RuleResult, error) {
	return rule.result, nil
}

// SC-013: observe composition through registry validation/evaluation, not the
// private field-merging helpers. Registration order must not affect results.
func TestSC013RegisteredRuleCompositionIsOrderIndependent(t *testing.T) {
	visible, hidden := true, false
	rules := []compositionRule{
		{"test.compose.a", ports.RuleResult{Label: "High priority", Role: "first", Visible: &visible, Annotations: map[string]any{"shared": "same", "a": "retained", "conflict": "first"}}},
		{"test.compose.b", ports.RuleResult{Role: "second", Visible: &hidden, Annotations: map[string]any{"shared": "same", "b": "retained", "conflict": "second"}}},
		{"test.compose.c", ports.RuleResult{Role: "first", Visible: &visible, Annotations: map[string]any{"conflict": "first"}}},
		{"test.compose.low", ports.RuleResult{Label: "Low priority", Role: "lower", Annotations: map[string]any{"conflict": "lower"}}},
	}
	value := domain.Profile{ProfileID: "project:composition"}
	for i, rule := range rules {
		priority := 10
		if i == 3 {
			priority = 1
		}
		value.Rules = append(value.Rules, domain.RuleInvocation{RuleID: rule.ID(), Version: "1", Enabled: true, Priority: priority})
	}
	var baseline ports.RuleResult
	var baselineDiagnostics []domain.Diagnostic
	for _, order := range [][]int{{0, 1, 2, 3}, {3, 2, 1, 0}, {1, 3, 0, 2}} {
		registry := NewRegistry()
		for _, index := range order {
			if err := registry.Register(rules[index]); err != nil {
				t.Fatal(err)
			}
		}
		validated, diagnostics := Validate(value, registry)
		if validated.Status != domain.ProfileValid || len(diagnostics) != 0 {
			t.Fatalf("valid composition rejected: %+v", diagnostics)
		}
		result, diagnostics := registry.Evaluate(context.Background(), domain.ConceptDocument{ConceptID: "subject"}, validated)
		if result.Label != "High priority" || result.Role != "" || result.Visible != nil {
			t.Fatalf("scalar precedence/conflict: %+v", result)
		}
		if !reflect.DeepEqual(result.Annotations, map[string]any{"shared": "same", "a": "retained", "b": "retained"}) {
			t.Fatalf("annotation composition: %+v", result.Annotations)
		}
		if len(diagnostics) == 0 {
			t.Fatal("conflicts lack diagnostics")
		}
		for _, diagnostic := range diagnostics {
			if diagnostic.Code != "okf_rule_conflict" || diagnostic.ConceptID != "subject" {
				t.Fatalf("unscoped conflict: %+v", diagnostic)
			}
		}
		if baselineDiagnostics == nil {
			baseline, baselineDiagnostics = result, diagnostics
		} else if !reflect.DeepEqual(baseline, result) || !reflect.DeepEqual(baselineDiagnostics, diagnostics) {
			t.Fatal("registration order changed composition")
		}
	}
}
