package profile

import (
	"context"
	"fmt"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
)

type testPresentationProvider struct{}

func (testPresentationProvider) Metadata() ports.Extension {
	return ports.Extension{ID: "test.presentation", Version: "1", Description: "Test properties", Capabilities: []string{"label", "token", "annotations"}, ParameterSchema: map[string]any{"type": "object"}}
}
func (testPresentationProvider) ValidateParameters(parameters map[string]any) error {
	if _, ok := parameters["label"].(string); !ok {
		return fmt.Errorf("label is required")
	}
	return nil
}
func (testPresentationProvider) Properties(_ context.Context, document domain.ConceptDocument, parameters map[string]any) (ports.PresentationProperties, error) {
	document.Frontmatter["original"] = "changed"
	label := parameters["label"].(string)
	parameters["label"] = "changed"
	if label == "panic" {
		panic("provider failed")
	}
	return ports.PresentationProperties{Label: label, Token: "custom", Annotations: map[string]any{"explanation": "retained"}}, nil
}

func TestPresentationProviderUsesRuleCompositionAndIsolation(t *testing.T) {
	registry := NewRegistry()
	if err := registry.RegisterPresentationPropertyProvider(testPresentationProvider{}); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterPresentationPropertyProvider(testPresentationProvider{}); err == nil {
		t.Fatal("duplicate provider accepted")
	}
	value := domain.Profile{ProfileID: "project:properties", Rules: []domain.RuleInvocation{
		{RuleID: "test.presentation", Version: "1", Enabled: true, Priority: 2, Parameters: map[string]any{"label": "Preferred"}},
		{RuleID: "test.presentation", Version: "1", Enabled: true, Priority: 1, Parameters: map[string]any{"label": "Other"}},
	}}
	document := domain.ConceptDocument{ConceptID: "root", Frontmatter: map[string]any{"original": "retained"}}
	for _, candidate := range []*Registry{registry, registry.WithProjectProfiles(nil)} {
		resolved, diagnostics := candidate.ResolveCandidate(value)
		if resolved.Status == domain.ProfileInvalid || len(diagnostics) > 0 {
			t.Fatalf("valid provider rejected: %+v", diagnostics)
		}
		result, diagnostics := candidate.Evaluate(context.Background(), document, resolved)
		if result.Label != "Preferred" || result.Token != "custom" || result.Annotations["explanation"] != "retained" || len(diagnostics) > 0 {
			t.Fatalf("composition: %+v %+v", result, diagnostics)
		}
		if document.Frontmatter["original"] != "retained" || value.Rules[0].Parameters["label"] != "Preferred" {
			t.Fatal("provider mutated inputs")
		}
		found := false
		for _, entry := range candidate.Catalog() {
			if entry.ID == "test.presentation" {
				found = entry.Kind == "presentation_property" && entry.ParameterSchema["type"] == "object"
			}
		}
		if !found {
			t.Fatal("provider catalog metadata lost")
		}
	}
	value.Rules[1].Priority = 2
	result, diagnostics := registry.Evaluate(context.Background(), document, value)
	if result.Label != "" || len(diagnostics) == 0 {
		t.Fatal("provider bypassed equal-priority conflict handling")
	}
	value.Rules = value.Rules[:1]
	value.Rules[0].Parameters["label"] = "panic"
	result, diagnostics = registry.Evaluate(context.Background(), document, value)
	if result.Label != "" || len(diagnostics) != 1 || diagnostics[0].Code != "okf_rule_invalid" {
		t.Fatalf("provider bypassed failure isolation: %+v %+v", result, diagnostics)
	}
	value.Rules[0].Parameters = nil
	invalid, _ := registry.ResolveCandidate(value)
	if invalid.Status != domain.ProfileInvalid {
		t.Fatal("provider parameter validation bypassed")
	}
}
