package scene

import (
	"sort"

	"github.com/buffo/arch-view/internal/analysis"
)

type relationshipAccumulator struct {
	Type               string
	FromVisibleID      string
	ToVisibleID        string
	TargetScope        string
	RelationshipIDs    []string
	SourceReferenceIDs []string
}

type relationshipProjection struct {
	visible                       []VisibleRelationship
	nodeRelationshipCounts        map[string]int
	internalRelationshipIDsByNode map[string][]string
	nodeEvidenceIDs               map[string][]string
}

func (state *sceneBuildState) buildVisibleRelationships() relationshipProjection {
	result := relationshipProjection{
		visible:                       []VisibleRelationship{},
		nodeRelationshipCounts:        make(map[string]int, len(state.visibleProjectionNodes)),
		internalRelationshipIDsByNode: make(map[string][]string, len(state.visibleProjectionNodes)),
		nodeEvidenceIDs:               make(map[string][]string, len(state.visibleProjectionNodes)),
	}
	accumulators := make(map[string]*relationshipAccumulator)
	for _, projected := range state.projection.Relationships {
		if _, exists := state.visibleNodeIDs[projected.FromNodeID]; !exists {
			continue
		}
		toVisibleID := projected.ToNodeID
		targetScope := ""
		if reference, isReference := state.referencesByID[projected.ToNodeID]; isReference {
			if !scopeAllowed(reference.Scope, state.allowedScopes) || state.options.ReferenceVisibility == ReferenceVisibilityHidden {
				continue
			}
			targetScope = reference.Scope
			if state.options.ReferenceVisibility == ReferenceVisibilityAggregated {
				toVisibleID = referenceBoundaryID(reference.Scope)
			}
		}
		if _, exists := state.visibleNodeIDs[toVisibleID]; !exists {
			continue
		}
		key := projected.Type + "\x00" + projected.FromNodeID + "\x00" + toVisibleID
		accumulator := accumulators[key]
		if accumulator == nil {
			accumulator = &relationshipAccumulator{
				Type:          projected.Type,
				FromVisibleID: projected.FromNodeID,
				ToVisibleID:   toVisibleID,
				TargetScope:   targetScope,
			}
			accumulators[key] = accumulator
		}
		for _, relationshipID := range projected.RelationshipIDs {
			accumulator.RelationshipIDs = appendUniqueString(accumulator.RelationshipIDs, relationshipID)
		}
		for _, sourceID := range projected.SourceReferenceIDs {
			accumulator.SourceReferenceIDs = appendUniqueString(accumulator.SourceReferenceIDs, sourceID)
		}
	}

	relationshipKeys := make([]string, 0, len(accumulators))
	for key := range accumulators {
		relationshipKeys = append(relationshipKeys, key)
	}
	sort.Strings(relationshipKeys)
	result.visible = make([]VisibleRelationship, 0, len(relationshipKeys))
	for _, key := range relationshipKeys {
		accumulator := accumulators[key]
		contributors := make([]analysis.RelationshipObservation, 0, len(accumulator.RelationshipIDs))
		for _, relationshipID := range accumulator.RelationshipIDs {
			if relationship, ok := state.relationshipsByID[relationshipID]; ok {
				contributors = append(contributors, relationship)
			}
		}
		confidenceState, confidenceBasis, confidenceScore := aggregateConfidence(contributors)
		cycleState := relationshipCycleState(accumulator.RelationshipIDs, state.cycleRelationships, state.feedbackRelationships)
		projectedNode, hasProjectedNode := state.visibleProjectionByID[accumulator.FromVisibleID]
		if hasProjectedNode && shouldSummarizeInternalRelationship(projectedNode, accumulator.FromVisibleID, accumulator.ToVisibleID, cycleState, contributors) {
			for _, relationshipID := range accumulator.RelationshipIDs {
				result.internalRelationshipIDsByNode[accumulator.FromVisibleID] = appendUniqueString(result.internalRelationshipIDsByNode[accumulator.FromVisibleID], relationshipID)
			}
			for _, sourceID := range accumulator.SourceReferenceIDs {
				result.nodeEvidenceIDs[accumulator.FromVisibleID] = appendUniqueString(result.nodeEvidenceIDs[accumulator.FromVisibleID], sourceID)
			}
			continue
		}
		visible := VisibleRelationship{
			ID:                 stableProjectionRelationshipID(accumulator.Type, accumulator.FromVisibleID, accumulator.ToVisibleID),
			Type:               accumulator.Type,
			FromVisibleID:      accumulator.FromVisibleID,
			ToVisibleID:        accumulator.ToVisibleID,
			RelationshipIDs:    append([]string{}, accumulator.RelationshipIDs...),
			SourceReferenceIDs: append([]string{}, accumulator.SourceReferenceIDs...),
			Count:              len(accumulator.RelationshipIDs),
			Directed:           true,
			CycleState:         cycleState,
			TargetScope:        accumulator.TargetScope,
			ConfidenceState:    confidenceState,
			ConfidenceBasis:    confidenceBasis,
			ConfidenceScore:    confidenceScore,
		}
		visible.AccessibleLabel = relationshipAccessibleLabel(visible, projectionNodeLabel(state.visibleProjectionNodes, visible.FromVisibleID), projectionNodeLabel(state.visibleProjectionNodes, visible.ToVisibleID))
		result.visible = append(result.visible, visible)
		result.nodeRelationshipCounts[accumulator.FromVisibleID]++
		if accumulator.ToVisibleID != accumulator.FromVisibleID {
			result.nodeRelationshipCounts[accumulator.ToVisibleID]++
		}
		for _, sourceID := range accumulator.SourceReferenceIDs {
			result.nodeEvidenceIDs[accumulator.FromVisibleID] = appendUniqueString(result.nodeEvidenceIDs[accumulator.FromVisibleID], sourceID)
			result.nodeEvidenceIDs[accumulator.ToVisibleID] = appendUniqueString(result.nodeEvidenceIDs[accumulator.ToVisibleID], sourceID)
		}
	}
	for nodeID := range result.nodeEvidenceIDs {
		sort.Strings(result.nodeEvidenceIDs[nodeID])
	}
	for nodeID := range result.internalRelationshipIDsByNode {
		sort.Strings(result.internalRelationshipIDsByNode[nodeID])
	}
	return result
}
