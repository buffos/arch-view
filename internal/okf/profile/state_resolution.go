package profile

import (
	"context"
	"strings"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
)

// EvaluateConceptState reuses evaluated rules and reads hidden direct children
// only when they contribute to an explicitly configured structural roll-up.
func EvaluateConceptState(ctx context.Context, conceptID string, index domain.BundleIndex, effective domain.Profile, children []string, registry ports.RuleEvaluator, results map[string]ports.RuleResult) (ConceptState, []domain.Diagnostic, error) {
	var diagnostics []domain.Diagnostic
	evaluate := func(id string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, exists := results[id]; exists {
			return nil
		}
		document, exists := index.Documents[id]
		if !exists {
			return nil
		}
		result, notes := registry.Evaluate(ctx, document, effective)
		if err := ctx.Err(); err != nil {
			return err
		}
		results[id] = result
		for i := range notes {
			notes[i].BundleID = index.BundleID
			notes[i].ProfileID = effective.ProfileID
			if notes[i].ConceptID == "" {
				notes[i].ConceptID = id
			}
		}
		diagnostics = append(diagnostics, notes...)
		return nil
	}
	if err := evaluate(conceptID); err != nil {
		return ConceptState{}, diagnostics, err
	}
	state := ResolveConceptState(conceptID, index, effective, nil, results)
	if effective.State.RollUp && state.IsRollup {
		for _, childID := range children {
			if err := evaluate(childID); err != nil {
				return ConceptState{}, diagnostics, err
			}
		}
		state = ResolveConceptState(conceptID, index, effective, children, results)
	}
	return state, diagnostics, nil
}

// ConceptState is presentation interpretation, never a source-state mutation.
type ConceptState struct {
	Declared  string
	Effective string
	Role      string
	IsRollup  bool
	RolledUp  bool
}

func ResolveConceptState(conceptID string, index domain.BundleIndex, effective domain.Profile, children []string, results map[string]ports.RuleResult) ConceptState {
	document := index.Documents[conceptID]
	result := results[conceptID]
	state := ConceptState{Declared: DeclaredState(document, effective.State.Field), Role: result.Role}
	state.Effective = effectiveDocumentState(document, effective, result)
	switch strings.ToLower(strings.TrimSpace(state.Role)) {
	case "aggregate", "rollup", "roll-up":
		state.IsRollup = true
	}
	if !effective.State.RollUp || !state.IsRollup || len(children) == 0 {
		return state
	}
	rolled := ""
	for _, childID := range children {
		child, exists := index.Documents[childID]
		if !exists {
			return state
		}
		value := effectiveDocumentState(child, effective, results[childID])
		if value == "unknown" || value == "" || (rolled != "" && rolled != value) {
			return state
		}
		rolled = value
	}
	state.Effective, state.RolledUp = rolled, true
	return state
}

func effectiveDocumentState(document domain.ConceptDocument, effective domain.Profile, result ports.RuleResult) string {
	if result.EffectiveState != "" {
		return result.EffectiveState
	}
	return MappedState(DeclaredState(document, effective.State.Field), effective.State.Mapping)
}
