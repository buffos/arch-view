package scene

import (
	"fmt"
	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
	"sort"
)

func sortedReferenceIDs(values map[string]*referenceAccumulator) []string {
	result := make([]string, 0, len(values))
	for referenceID := range values {
		result = append(result, referenceID)
	}
	sort.Strings(result)
	return result
}

func referenceBoundaryID(scope string) string {
	return "reference-boundary:" + scope
}

func referenceBoundaryLabel(scope string) string {
	switch scope {
	case "standard_library":
		return "Standard library"
	case "external":
		return "External references"
	case "unresolved":
		return "Unresolved references"
	case "dynamic":
		return "Dynamic references"
	default:
		return scope + " references"
	}
}

func stableProjectionRelationshipID(kind, from, to string) string {
	return "projection:" + kind + ":" + from + ":" + to
}

func shouldSummarizeInternalRelationship(node model.ProjectionNode, fromID, toID, cycleState string, contributors []analysis.RelationshipObservation) bool {
	if node.Kind != "group" || fromID != toID || cycleState != "none" || len(contributors) == 0 {
		return false
	}
	for _, relationship := range contributors {
		if relationship.ToModuleID == "" || relationship.FromModuleID == relationship.ToModuleID {
			return false
		}
	}
	return true
}

func buildReferenceDetails(values map[string]*referenceAccumulator, relationships map[string]analysis.RelationshipObservation) []ReferenceDetail {
	result := make([]ReferenceDetail, 0, len(values))
	for _, referenceID := range sortedReferenceIDs(values) {
		accumulator := values[referenceID]
		relationshipIDs := append([]string{}, accumulator.RelationshipIDs...)
		sourceReferenceIDs := append([]string{}, accumulator.SourceReferenceIDs...)
		fromVisibleIDs := append([]string{}, accumulator.FromVisibleIDs...)
		sort.Strings(relationshipIDs)
		sort.Strings(sourceReferenceIDs)
		sort.Strings(fromVisibleIDs)
		contributors := make([]analysis.RelationshipObservation, 0, len(relationshipIDs))
		for _, relationshipID := range relationshipIDs {
			if relationship, ok := relationships[relationshipID]; ok {
				contributors = append(contributors, relationship)
			}
		}
		confidenceState, confidenceBasis, confidenceScore := aggregateConfidence(contributors)
		label := fmt.Sprintf("%s reference %s; %d import(s); %s confidence", accumulator.Reference.Scope, accumulator.Reference.Name, len(relationshipIDs), confidenceState)
		result = append(result, ReferenceDetail{
			ID:                 accumulator.Reference.ID,
			Name:               accumulator.Reference.Name,
			Scope:              accumulator.Reference.Scope,
			FromVisibleIDs:     fromVisibleIDs,
			RelationshipIDs:    relationshipIDs,
			SourceReferenceIDs: sourceReferenceIDs,
			Count:              len(relationshipIDs),
			ConfidenceState:    confidenceState,
			ConfidenceBasis:    confidenceBasis,
			ConfidenceScore:    confidenceScore,
			AccessibleLabel:    label,
		})
	}
	return result
}

func buildReferenceSummary(values map[string]*referenceAccumulator, visibility string) ReferenceSummary {
	byScope := make(map[string]*ReferenceScopeSummary)
	result := ReferenceSummary{ByScope: []ReferenceScopeSummary{}}
	for _, referenceID := range sortedReferenceIDs(values) {
		accumulator := values[referenceID]
		result.Total++
		result.RelationshipCount += len(accumulator.RelationshipIDs)
		summary := byScope[accumulator.Reference.Scope]
		if summary == nil {
			summary = &ReferenceScopeSummary{Scope: accumulator.Reference.Scope}
			byScope[accumulator.Reference.Scope] = summary
		}
		summary.ReferenceCount++
		summary.RelationshipCount += len(accumulator.RelationshipIDs)
	}
	for _, scope := range referenceScopeOrder {
		summary, ok := byScope[scope]
		if !ok {
			continue
		}
		switch visibility {
		case ReferenceVisibilityHidden:
			result.HiddenCount += summary.ReferenceCount
		case ReferenceVisibilityAggregated:
			result.AggregatedCount++
			summary.VisibleNodeCount = 1
		case ReferenceVisibilityExpanded:
			result.ExpandedCount += summary.ReferenceCount
			summary.VisibleNodeCount = summary.ReferenceCount
		}
		result.VisibleCount += summary.VisibleNodeCount
		result.ByScope = append(result.ByScope, *summary)
	}
	return result
}

func aggregateConfidence(values []analysis.RelationshipObservation) (string, string, *float64) {
	if len(values) == 0 {
		return "unknown", "", nil
	}
	var minimum float64 = 1
	hasConfidence := false
	bases := map[string]struct{}{}
	for _, value := range values {
		if value.Confidence == nil {
			continue
		}
		hasConfidence = true
		if value.Confidence.Score < minimum {
			minimum = value.Confidence.Score
		}
		bases[value.Confidence.Basis] = struct{}{}
	}
	if !hasConfidence {
		return "unknown", "", nil
	}
	basis := ""
	if len(bases) == 1 {
		for value := range bases {
			basis = value
		}
	} else {
		basis = "mixed"
	}
	score := minimum
	return confidenceState(score), basis, &score
}
