package scene

import (
	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
	"sort"
)

func confidenceState(score float64) string {
	switch {
	case score >= 0.9:
		return "high"
	case score >= 0.6:
		return "medium"
	default:
		return "low"
	}
}

func relationshipCycleState(relationshipIDs []string, cycleRelationships, feedbackRelationships map[string]struct{}) string {
	for _, relationshipID := range relationshipIDs {
		if _, ok := cycleRelationships[relationshipID]; ok {
			return "cycle"
		}
	}
	for _, relationshipID := range relationshipIDs {
		if _, ok := feedbackRelationships[relationshipID]; ok {
			return "feedback"
		}
	}
	return "none"
}

func nodeCycleState(moduleIDs []string, cycleModules, feedbackRelationships map[string]struct{}, relationships []analysis.RelationshipObservation) string {
	for _, moduleID := range moduleIDs {
		if _, ok := cycleModules[moduleID]; ok {
			return "cycle"
		}
	}
	for _, relationship := range relationships {
		if _, ok := feedbackRelationships[relationship.ID]; !ok {
			continue
		}
		for _, moduleID := range moduleIDs {
			if relationship.FromModuleID == moduleID || relationship.ToModuleID == moduleID {
				return "feedback"
			}
		}
	}
	return "none"
}

func nodeLayers(moduleIDs []string, layersByModule map[string]int) []int {
	values := []int{}
	seen := map[int]struct{}{}
	for _, moduleID := range moduleIDs {
		layer, ok := layersByModule[moduleID]
		if !ok {
			continue
		}
		if _, exists := seen[layer]; exists {
			continue
		}
		seen[layer] = struct{}{}
		values = append(values, layer)
	}
	sort.Ints(values)
	return values
}

func diagnosticsForVisibleNode(nodeID string, diagnosticNodeIDs map[string][]string) []string {
	values := []string{}
	for diagnosticID, nodeIDs := range diagnosticNodeIDs {
		for _, candidate := range nodeIDs {
			if candidate == nodeID {
				values = append(values, diagnosticID)
				break
			}
		}
	}
	sort.Strings(values)
	return values
}

func diagnosticsState(diagnosticIDs []string, diagnostics map[string]model.Diagnostic) string {
	state := "none"
	for _, diagnosticID := range diagnosticIDs {
		diagnostic, ok := diagnostics[diagnosticID]
		if !ok {
			continue
		}
		if diagnostic.Severity == "error" {
			return "error"
		}
		if diagnostic.Severity == "warning" {
			state = "warning"
		} else if state == "none" {
			state = "info"
		}
	}
	return state
}

func sourceModuleIDsForDiagnostic(diagnostic model.Diagnostic, sourceModuleIDs map[string][]string) []string {
	values := []string{}
	for _, sourceID := range diagnostic.SourceReferenceIDs {
		for _, moduleID := range sourceModuleIDs[sourceID] {
			values = appendUniqueString(values, moduleID)
		}
	}
	sort.Strings(values)
	return values
}
