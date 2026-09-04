package profile

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestRegistryComposesOrderedBasesAndHonorsExplicitRelationshipOverrides(t *testing.T) {
	registry := NewRegistry()
	custom := domain.Profile{
		ProfileID: "project:semantic-only",
		Name:      "Semantic only",
		Origin:    "project_local",
		Bases:     []string{DefaultProfileID},
		Relationships: domain.RelationshipSettings{
			ShowContainment: true,
			ShowSemantic:    false,
		},
	}
	registry.SetProjectProfiles([]domain.Profile{custom})
	effective, diagnostics := registry.ResolveProfile(custom.ProfileID)
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
	if effective.Relationships.ShowContainment != true || effective.Relationships.ShowSemantic {
		t.Fatalf("effective relationships = %#v", effective.Relationships)
	}
	if effective.Style.Tokens["state.unknown"].Fill == "" {
		t.Fatalf("built-in style tokens were lost during profile cloning: %#v", effective.Style.Tokens)
	}

	registry.SetProjectProfiles(nil)
	if _, exists := registry.Profile(custom.ProfileID); exists {
		t.Fatal("stale project profile remained in registry")
	}
}

func TestValidateProfileReportsMissingBaseAndRule(t *testing.T) {
	registry := NewRegistry()
	value, diagnostics := Validate(domain.Profile{
		ProfileID: "project:invalid",
		Origin:    "project_local",
		Bases:     []string{"project:missing"},
		Rules:     []domain.RuleInvocation{{RuleID: "okf.rule.missing", Enabled: true}},
	}, registry)
	if value.Status != domain.ProfileInvalid || len(diagnostics) < 2 {
		t.Fatalf("value=%#v diagnostics=%#v", value, diagnostics)
	}
}

func TestRuleInvocationDefaultsToEnabledOnlyWhenOmitted(t *testing.T) {
	var value struct {
		Rules []domain.RuleInvocation `json:"rules"`
	}
	if err := json.Unmarshal([]byte(`{"rules":[{"rule_id":"okf.rule.label_template","parameters":{"template":"{title}"}},{"rule_id":"okf.rule.visibility","enabled":false}]}`), &value); err != nil {
		t.Fatal(err)
	}
	if !value.Rules[0].Enabled || value.Rules[1].Enabled {
		t.Fatalf("rules = %#v", value.Rules)
	}
	result, diagnostics := NewRegistry().Evaluate(context.Background(), domain.ConceptDocument{ConceptID: "x", Title: "X"}, domain.Profile{Rules: value.Rules})
	if result.Label != "X" || len(diagnostics) != 0 {
		t.Fatalf("evaluation = %#v diagnostics=%#v", result, diagnostics)
	}
}

func TestRuleInvocationRejectsUnknownFields(t *testing.T) {
	var value domain.RuleInvocation
	if err := json.Unmarshal([]byte(`{"rule_id":"okf.rule.label_template","unexpected":true}`), &value); err == nil {
		t.Fatal("unknown rule fields should be rejected")
	}
}

func TestProfileJSONPreservesExplicitFalseRelationshipOverrides(t *testing.T) {
	var value domain.Profile
	if err := json.Unmarshal([]byte(`{"profile_id":"project:links-off","bases":["builtin:neutral"],"relationships":{"show_containment":false,"show_semantic_links":false}}`), &value); err != nil {
		t.Fatal(err)
	}
	registry := NewRegistry()
	registry.SetProjectProfiles([]domain.Profile{value})
	effective, diagnostics := registry.ResolveProfile(value.ProfileID)
	if len(diagnostics) != 0 || effective.Relationships.ShowContainment || effective.Relationships.ShowSemantic {
		t.Fatalf("explicit relationship override was lost: effective=%#v diagnostics=%#v", effective.Relationships, diagnostics)
	}
}

func TestNeutralProfileDoesNotInventAStateField(t *testing.T) {
	effective, diagnostics := NewRegistry().ResolveProfile(DefaultProfileID)
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
	if effective.State.Field != "" || len(effective.NodeFields) != 0 {
		t.Fatalf("neutral profile should remain vocabulary agnostic: %#v", effective)
	}
	if effective.Layout.Algorithm != "mrtree" {
		t.Fatalf("OKF default layout = %q, want mrtree", effective.Layout.Algorithm)
	}
}

func TestValidateNodeFieldsRejectsDuplicateAndOversizedConfiguration(t *testing.T) {
	value, diagnostics := Validate(domain.Profile{
		ProfileID: "project:fields",
		NodeFields: []domain.NodeField{
			{Source: "frontmatter.state"},
			{Source: "frontmatter.state"},
			{Source: "type"},
			{Source: "role", MaxLength: MaxNodeFieldText + 1},
		},
	}, NewRegistry())
	if value.Status != domain.ProfileInvalid || len(diagnostics) < 2 {
		t.Fatalf("value=%#v diagnostics=%#v", value, diagnostics)
	}
}

func TestEmptyNodeFieldOverlayClearsInheritedPresentationFields(t *testing.T) {
	registry := NewRegistry()
	custom := domain.Profile{ProfileID: "project:name-only", Bases: []string{FogProfileID}, NodeFields: []domain.NodeField{}}
	registry.SetProjectProfiles([]domain.Profile{custom})
	effective, diagnostics := registry.ResolveProfile(custom.ProfileID)
	if len(diagnostics) != 0 || len(effective.NodeFields) != 0 {
		t.Fatalf("node field override = %#v diagnostics=%#v", effective.NodeFields, diagnostics)
	}
}
