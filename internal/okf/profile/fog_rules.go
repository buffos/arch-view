package profile

import "github.com/buffo/arch-view/internal/okf/domain"

// Planning vocabulary belongs to this profile, not the shared state resolver.
func fogRollupRules() []domain.RuleInvocation {
	var rules []domain.RuleInvocation
	add := func(field, value string) {
		rules = append(rules, domain.RuleInvocation{
			RuleID: "okf.rule.metadata_equals", Version: "1", Enabled: true, Priority: -1,
			Parameters: map[string]any{"field": field, "value": value, "role": "rollup"},
		})
	}
	for _, field := range []string{"role", "type"} {
		for _, value := range []string{"aggregate", "rollup", "roll-up"} {
			add(field, value)
		}
	}
	add("state_policy.mode", "rollup")
	return rules
}
