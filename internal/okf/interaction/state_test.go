package interaction

import (
	"context"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
	"github.com/buffo/arch-view/internal/okf/profile"
	"github.com/buffo/arch-view/internal/okf/projection"
)

type stateRule struct{ cancel context.CancelFunc }

func (stateRule) ID() string          { return "test.state" }
func (stateRule) Version() string     { return "1" }
func (stateRule) Description() string { return "State interpretation fixture" }
func (rule stateRule) Evaluate(_ context.Context, document domain.ConceptDocument, _ domain.RuleInvocation) (ports.RuleResult, error) {
	if document.ConceptID == "child" {
		if rule.cancel != nil {
			rule.cancel()
		}
		return ports.RuleResult{EffectiveState: "done"}, nil
	}
	if document.Type == "aggregate" {
		return ports.RuleResult{Role: "rollup"}, nil
	}
	return ports.RuleResult{}, nil
}

func stateFixture(t *testing.T, rule stateRule) (domain.BundleIndex, domain.Profile, *profile.Registry) {
	t.Helper()
	registry := profile.NewRegistry()
	if err := registry.Register(rule); err != nil {
		t.Fatal(err)
	}
	value, _ := registry.ResolveProfile(profile.DefaultProfileID)
	value.State.Field, value.State.RollUp = "state", true
	value.Rules = []domain.RuleInvocation{{RuleID: rule.ID(), Version: rule.Version(), Enabled: true}}
	index := domain.BundleIndex{ConceptOrder: []string{"root", "aggregate", "child"}, Documents: map[string]domain.ConceptDocument{
		"root":      {ConceptID: "root", SourcePath: "root.md", ExplicitChildren: []string{"aggregate"}},
		"aggregate": {ConceptID: "aggregate", SourcePath: "aggregate.md", Type: "aggregate", ExplicitChildren: []string{"child"}, Frontmatter: map[string]any{"state": "pending"}},
		"child":     {ConceptID: "child", SourcePath: "child.md", Frontmatter: map[string]any{"state": "pending"}},
	}}
	return index, value, registry
}

func TestDetailAndGraphShareRuleAndRollupStates(t *testing.T) {
	index, value, registry := stateFixture(t, stateRule{})
	for _, depth := range []int{1, 2} {
		snapshot, err := projection.Build(context.Background(), index, value, domain.NavigationState{Depth: depth}, registry)
		if err != nil {
			t.Fatal(err)
		}
		if len(snapshot.Nodes) != depth+1 {
			t.Fatalf("state dependencies changed visibility: %#v", snapshot.Nodes)
		}
		for _, id := range []string{"aggregate", "child"} {
			detail, err := Detail(context.Background(), index, value, id, registry)
			if err != nil {
				t.Fatal(err)
			}
			if detail.DeclaredState != "pending" || detail.EffectiveState != "done" {
				t.Fatalf("%s states: %q/%q", id, detail.DeclaredState, detail.EffectiveState)
			}
			for _, node := range snapshot.Nodes {
				if node.ConceptID == id && (node.DeclaredState != detail.DeclaredState || node.EffectiveState != detail.EffectiveState) {
					t.Fatalf("graph/detail mismatch: %#v", node)
				}
			}
		}
	}
	if index.Documents["aggregate"].Frontmatter["state"] != "pending" {
		t.Fatal("source state mutated")
	}
}

func TestCancellationDuringHiddenStateDependency(t *testing.T) {
	for _, detail := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		index, value, registry := stateFixture(t, stateRule{cancel: cancel})
		var err error
		if detail {
			_, err = Detail(ctx, index, value, "aggregate", registry)
		} else {
			_, err = projection.Build(ctx, index, value, domain.NavigationState{Depth: 1}, registry)
		}
		cancel()
		if err == nil {
			t.Fatalf("detail=%v: cancelled dependency produced successful result", detail)
		}
	}
}

func TestDetailRollupRequiresExplicitRoleAndKnownAgreeingChildren(t *testing.T) {
	for _, scenario := range []string{"disabled", "ordinary", "unknown child", "mixed children"} {
		t.Run(scenario, func(t *testing.T) {
			index, value, registry := stateFixture(t, stateRule{})
			switch scenario {
			case "disabled":
				value.State.RollUp = false
			case "ordinary":
				document := index.Documents["aggregate"]
				document.Type = "concept"
				index.Documents["aggregate"] = document
			default:
				document := index.Documents["aggregate"]
				document.ExplicitChildren = append(document.ExplicitChildren, "other")
				index.Documents["aggregate"] = document
				state := "unknown"
				if scenario == "mixed children" {
					state = "pending"
				}
				index.Documents["other"] = domain.ConceptDocument{ConceptID: "other", SourcePath: "other.md", Frontmatter: map[string]any{"state": state}}
				index.ConceptOrder = append(index.ConceptOrder, "other")
			}
			detail, err := Detail(context.Background(), index, value, "aggregate", registry)
			if err != nil {
				t.Fatal(err)
			}
			if detail.EffectiveState != "pending" {
				t.Fatalf("invalid roll-up replaced source state: %q", detail.EffectiveState)
			}
		})
	}
}
