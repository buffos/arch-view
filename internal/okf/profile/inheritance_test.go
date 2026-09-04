package profile

import (
	"encoding/json"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestMultipleBasesApplyDefaultsAfterComposition(t *testing.T) {
	var values []domain.Profile
	if err := json.Unmarshal([]byte(`[
	 {"profile_id":"project:first","navigation":{"default_depth":7,"max_nodes":33},"layout":{"algorithm":"layered","options":{"elk.direction":"DOWN"}},"hierarchy":{"use_explicit":false},"style":{"default_token":"custom","tokens":{"custom":{"fill":"red"}}}},
	 {"profile_id":"project:second","node_fields":[{"source":"title"}]},
	 {"profile_id":"project:composed","bases":["project:first","project:second"]}
	]`), &values); err != nil {
		t.Fatal(err)
	}
	registry := NewRegistry()
	registry.SetProjectProfiles(values)
	effective, diagnostics := registry.ResolveProfile("project:composed")
	if len(diagnostics) != 0 {
		t.Fatal(diagnostics)
	}
	if effective.Navigation.DefaultDepth != 7 || effective.Navigation.MaxNodes != 33 || effective.Layout.Algorithm != "layered" || effective.Layout.Options["elk.direction"] != "DOWN" || effective.Hierarchy.UseExplicit || effective.Style.DefaultToken != "custom" || effective.Style.Tokens["custom"].Fill != "red" {
		t.Fatalf("omitted values in later base overwrote explicit earlier values: %#v", effective)
	}
	if len(effective.NodeFields) != 1 || effective.NodeFields[0].Source != "title" {
		t.Fatal("later declaration lost")
	}
	values[1].Navigation.DefaultDepth = 4
	registry.SetProjectProfiles(values)
	effective, _ = registry.ResolveProfile("project:composed")
	if effective.Navigation.DefaultDepth != 4 {
		t.Fatal("explicit later override ignored")
	}
}

func TestEmptyStyleMappingsSurviveProfileRoundTrip(t *testing.T) {
	var values []domain.Profile
	if err := json.Unmarshal([]byte(`[{"profile_id":"project:base","style":{"tokens":{"custom":{"fill":"red"}},"state_tokens":{"ready":"custom"}}},{"profile_id":"project:child","bases":["project:base"],"style":{"tokens":{},"state_tokens":{}}}]`), &values); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, &values); err != nil {
		t.Fatal(err)
	}
	registry := NewRegistry()
	registry.SetProjectProfiles(values)
	effective, diagnostics := registry.ResolveProfile("project:child")
	if len(diagnostics) != 0 {
		t.Fatal(diagnostics)
	}
	if effective.Style.Tokens == nil || len(effective.Style.Tokens) != 0 || effective.Style.StateTokens == nil || len(effective.Style.StateTokens) != 0 {
		t.Fatalf("empty override was lost: %#v", effective.Style)
	}
}

func TestPartialRootProfileDefaultsOnlyOmittedBooleans(t *testing.T) {
	var value domain.Profile
	if err := json.Unmarshal([]byte(`{"profile_id":"project:partial","hierarchy":{"use_explicit":false},"relationships":{"show_semantic_links":false}}`), &value); err != nil {
		t.Fatal(err)
	}
	registry := NewRegistry()
	registry.SetProjectProfiles([]domain.Profile{value})
	effective, diagnostics := registry.ResolveProfile(value.ProfileID)
	if len(diagnostics) != 0 {
		t.Fatal(diagnostics)
	}
	if effective.Hierarchy.UseExplicit || !effective.Hierarchy.UseFilesystem || !effective.Relationships.ShowContainment || effective.Relationships.ShowSemantic {
		t.Fatalf("partial defaults erased omission or false: %#v %#v", effective.Hierarchy, effective.Relationships)
	}
}

func TestExplicitZeroNavigationLimitsAreNotDefaulted(t *testing.T) {
	for _, field := range []string{"default_depth", "max_nodes", "max_relationships"} {
		t.Run(field, func(t *testing.T) {
			var value domain.Profile
			if err := json.Unmarshal([]byte(`{"profile_id":"project:invalid","bases":["builtin:neutral"],"navigation":{"`+field+`":0}}`), &value); err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(domain.CloneProfile(value))
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(encoded, &value); err != nil {
				t.Fatal(err)
			}
			registry := NewRegistry()
			if _, diagnostics := Validate(value, registry); len(diagnostics) == 0 {
				t.Fatal("explicit zero was silently defaulted")
			}
			registry.SetProjectProfiles([]domain.Profile{value})
			effective, _ := registry.ResolveProfile(value.ProfileID)
			if effective.Status != domain.ProfileInvalid {
				t.Fatalf("invalid persisted limit accepted: %#v", effective.Navigation)
			}
		})
	}
}

func TestSparseProfileInheritanceSurvivesValidationAndPersistence(t *testing.T) {
	var values []domain.Profile
	data := `[{
	 "profile_id":"project:base", "layout":{"algorithm":"layered","options":{"elk.direction":"DOWN"}},
	 "navigation":{"default_depth":4}, "hierarchy":{"use_explicit":true,"use_filesystem_fallback":false},
	 "relationships":{"show_containment":true,"show_semantic_links":false},
	 "details":{"show_raw_markdown":true,"show_unknown_frontmatter":true},
	 "state":{"field":"status","roll_up":true}, "node_fields":[{"source":"type"}],
	 "rules":[{"rule_id":"okf.rule.label_template","parameters":{"template":"{title}"}}]
	},{"profile_id":"project:child","bases":["project:base"],"relationships":{"show_semantic_links":true},
	 "state":{"roll_up":false},"details":{"show_raw_markdown":false},"node_fields":[]}]`
	if err := json.Unmarshal([]byte(data), &values); err != nil {
		t.Fatal(err)
	}
	registry := NewRegistry()
	registry.SetProjectProfiles(values)
	validated, diagnostics := Validate(values[1], registry)
	if len(diagnostics) != 0 {
		t.Fatal(diagnostics)
	}
	values[1] = domain.CloneProfile(validated)
	encoded, err := json.Marshal(values)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, &values); err != nil {
		t.Fatal(err)
	}
	registry.SetProjectProfiles(values)
	effective, diagnostics := registry.ResolveProfile("project:child")
	if len(diagnostics) != 0 {
		t.Fatal(diagnostics)
	}
	if effective.Layout.Algorithm != "layered" || effective.Layout.Options["elk.direction"] != "DOWN" || effective.Navigation.DefaultDepth != 4 {
		t.Fatalf("inherited layout/navigation lost: %#v", effective)
	}
	if !effective.Hierarchy.UseExplicit || effective.Hierarchy.UseFilesystem || !effective.Relationships.ShowContainment || !effective.Relationships.ShowSemantic {
		t.Fatalf("partial hierarchy/relationship overrides lost: %#v", effective)
	}
	if effective.State.Field != "status" || effective.State.RollUp || effective.Details.ShowRawMarkdown || !effective.Details.ShowUnknown || len(effective.NodeFields) != 0 || len(effective.Rules) != 1 {
		t.Fatalf("inherited state/details/rules lost: %#v", effective)
	}
	before := effective.Revision
	values[0].Layout.Algorithm = "mrtree"
	registry.SetProjectProfiles(values)
	effective, _ = registry.ResolveProfile("project:child")
	if before == effective.Revision {
		t.Fatal("inherited change did not advance effective profile revision")
	}
}
