package profile

import (
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
)

func TestRuleSelectedRoleOverridesSourceRollupPolicy(t *testing.T) {
	index := domain.BundleIndex{Documents: map[string]domain.ConceptDocument{
		"parent": {ConceptID: "parent", Type: "aggregate", Frontmatter: map[string]any{
			"state": "pending", "role": "aggregate", "state_policy": map[string]any{"mode": "rollup"},
		}},
		"child": {ConceptID: "child", Frontmatter: map[string]any{"state": "done"}},
	}}
	effective := domain.Profile{State: domain.StateSettings{Field: "state", RollUp: true}}
	for _, test := range []struct {
		role   string
		rollup bool
		state  string
	}{
		{"independent", false, "pending"},
		{"rollup", true, "done"},
		{"", false, "pending"},
	} {
		result := ResolveConceptState("parent", index, effective, []string{"child"}, map[string]ports.RuleResult{
			"parent": {Role: test.role},
		})
		if result.IsRollup != test.rollup || result.RolledUp != test.rollup || result.Effective != test.state || result.Declared != "pending" {
			t.Fatalf("role=%q state=%+v", test.role, result)
		}
	}
	if index.Documents["parent"].Frontmatter["state"] != "pending" {
		t.Fatal("role interpretation changed source state")
	}
}
