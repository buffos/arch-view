package profile

import (
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestValidateBuiltinRuleParameters(t *testing.T) {
	for _, tc := range []struct {
		id             string
		valid, invalid map[string]any
	}{
		{"metadata_equals", map[string]any{"field": "state", "value": "ready"}, map[string]any{"field": "state"}},
		{"metadata_contains", map[string]any{"field": "tags", "value": "ready"}, map[string]any{"field": "tags", "value": nil}},
		{"state_mapping", map[string]any{"mapping": map[string]any{"ready": "done"}}, map[string]any{"mapping": map[string]any{"ready": 3}}},
		{"label_template", map[string]any{"template": "{title}"}, map[string]any{"template": false}},
		{"visibility", map[string]any{"field": "hidden", "value": true, "visible": false}, map[string]any{"field": "hidden", "value": true, "visible": "false"}},
	} {
		t.Run(tc.id, func(t *testing.T) {
			registry := NewRegistry()
			for _, valid := range []bool{false, true} {
				parameters := tc.invalid
				if valid {
					parameters = tc.valid
				}
				value := domain.Profile{ProfileID: "project:test", Rules: []domain.RuleInvocation{{RuleID: "okf.rule." + tc.id, Enabled: true, Parameters: parameters}}}
				checked, diagnostics := Validate(value, registry)
				if valid && (checked.Status != domain.ProfileValid || len(diagnostics) != 0) {
					t.Fatalf("valid rejected: %#v", diagnostics)
				}
				if !valid && (checked.Status != domain.ProfileInvalid || len(diagnostics) != 1 || diagnostics[0].Code != "okf_rule_invalid") {
					t.Fatalf("invalid accepted: %#v", diagnostics)
				}
			}
		})
	}
}

type panickingValidator struct{ panickingRule }

func (panickingValidator) ValidateParameters(parameters map[string]any) error {
	parameters["nested"].(map[string]any)["value"] = "changed"
	panic("invalid fixture")
}

func TestRuleValidationIsolatedFromProfileAndHost(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(panickingValidator{}); err != nil {
		t.Fatal(err)
	}
	parameters := map[string]any{"nested": map[string]any{"value": "original"}}
	_, diagnostics := Validate(domain.Profile{ProfileID: "project:test", Rules: []domain.RuleInvocation{{RuleID: "test.panic", Enabled: true, Parameters: parameters}}}, registry)
	if len(diagnostics) != 1 || diagnostics[0].Code != "okf_rule_invalid" {
		t.Fatalf("panic not isolated: %#v", diagnostics)
	}
	if parameters["nested"].(map[string]any)["value"] != "original" {
		t.Fatal("validation mutated profile")
	}
}
