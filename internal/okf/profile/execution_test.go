package profile

import (
	"context"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
)

type panickingRule struct{}

func (panickingRule) ID() string          { return "test.panic" }
func (panickingRule) Version() string     { return "1" }
func (panickingRule) Description() string { return "Mutation and panic fixture" }
func (panickingRule) Evaluate(_ context.Context, document domain.ConceptDocument, invocation domain.RuleInvocation) (ports.RuleResult, error) {
	document.Frontmatter["nested"].(map[string]any)["value"] = "changed"
	invocation.Parameters["value"] = "changed"
	panic("fixture failure")
}

func TestRuleFailureIsIsolatedFromSourceAndProfile(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(panickingRule{}); err != nil {
		t.Fatal(err)
	}
	document := domain.ConceptDocument{Frontmatter: map[string]any{"nested": map[string]any{"value": "original"}}}
	value := domain.Profile{Rules: []domain.RuleInvocation{{RuleID: "test.panic", Enabled: true, Parameters: map[string]any{"value": "original"}}}}
	result, diagnostics := registry.Evaluate(context.Background(), document, value)
	if len(diagnostics) != 1 || diagnostics[0].Code != "okf_rule_invalid" || result.Visible != nil {
		t.Fatalf("failure not isolated: %#v %#v", result, diagnostics)
	}
	if document.Frontmatter["nested"].(map[string]any)["value"] != "original" || value.Rules[0].Parameters["value"] != "original" {
		t.Fatal("rule mutated shared inputs")
	}
}

func TestAnnotationPriorityAndConflicts(t *testing.T) {
	result := ports.RuleResult{Annotations: map[string]any{}}
	priorities := map[string]int{}
	diagnostics := []domain.Diagnostic{}
	applyResult(&result, priorities, 10, ports.RuleResult{Annotations: map[string]any{"note": "high"}}, &diagnostics, "concept")
	applyResult(&result, priorities, 1, ports.RuleResult{Annotations: map[string]any{"note": "low", "other": "kept"}}, &diagnostics, "concept")
	if result.Annotations["note"] != "high" || result.Annotations["other"] != "kept" {
		t.Fatalf("priority ignored: %#v", result.Annotations)
	}
	applyResult(&result, priorities, 10, ports.RuleResult{Annotations: map[string]any{"note": "conflict"}}, &diagnostics, "concept")
	applyResult(&result, priorities, 10, ports.RuleResult{Annotations: map[string]any{"note": "high"}}, &diagnostics, "concept")
	if _, exists := result.Annotations["note"]; exists {
		t.Fatal("conflict was resurrected")
	}
	if len(diagnostics) == 0 || diagnostics[0].Code != "okf_rule_conflict" {
		t.Fatal("missing conflict diagnostic")
	}
}

func TestAnnotationConflictsHaveStableOrder(t *testing.T) {
	for attempt := 0; attempt < 100; attempt++ {
		result := ports.RuleResult{Annotations: map[string]any{}}
		priorities := map[string]int{}
		var diagnostics []domain.Diagnostic
		applyResult(&result, priorities, 1, ports.RuleResult{Annotations: map[string]any{"z": "first", "a": "first"}}, &diagnostics, "concept")
		applyResult(&result, priorities, 1, ports.RuleResult{Annotations: map[string]any{"z": "second", "a": "second"}}, &diagnostics, "concept")
		if len(diagnostics) != 2 || diagnostics[0].Details["field"] != "a" || diagnostics[1].Details["field"] != "z" {
			t.Fatalf("unstable annotation diagnostics: %#v", diagnostics)
		}
	}
}

func TestRuleOutputCaptureIsAtomicAndOwnsVisibility(t *testing.T) {
	visible := true
	rule := compositionRule{id: "test.capture", result: ports.RuleResult{Visible: &visible}}
	result, err := evaluateRule(context.Background(), rule, domain.ConceptDocument{}, domain.RuleInvocation{})
	if err != nil || result.Visible == nil {
		t.Fatalf("capture failed: %+v %v", result, err)
	}
	visible = false
	if !*result.Visible {
		t.Fatal("rule retained ownership of visibility")
	}
	rule.result.Annotations = map[string]any{"invalid": make(chan int)}
	result, err = evaluateRule(context.Background(), rule, domain.ConceptDocument{}, domain.RuleInvocation{})
	if err == nil || result.Visible != nil || result.Annotations != nil {
		t.Fatalf("partial invalid result escaped: %+v %v", result, err)
	}
}
