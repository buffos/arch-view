package scene

import (
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

func (state *sceneBuildState) buildVisibleNodes(relationships relationshipProjection, referenceDetails []ReferenceDetail) []VisibleNode {
	referenceDetailsByID := make(map[string]ReferenceDetail, len(referenceDetails))
	for _, detail := range referenceDetails {
		referenceDetailsByID[detail.ID] = detail
	}

	visibleNodes := make([]VisibleNode, 0, len(state.visibleProjectionNodes))
	for _, projected := range state.visibleProjectionNodes {
		tags := []string{}
		sourceIDs := append([]string{}, relationships.nodeEvidenceIDs[projected.ID]...)
		for _, moduleID := range projected.ModuleIDs {
			module, ok := state.modulesByID[moduleID]
			if !ok {
				continue
			}
			for _, tag := range module.Tags {
				tags = appendUniqueString(tags, tag)
			}
			for _, sourceID := range module.SourceReferenceIDs {
				sourceIDs = appendUniqueString(sourceIDs, sourceID)
			}
		}
		sort.Strings(tags)
		sort.Strings(sourceIDs)

		cycleState := nodeCycleState(projected.ModuleIDs, state.cycleModules, state.feedbackRelationships, state.value.Relationships)
		diagnosticIDs := diagnosticsForVisibleNode(projected.ID, state.diagnosticNodeIDs)
		diagnosticState := diagnosticsState(diagnosticIDs, state.diagnosticsByID)
		internalRelationshipIDs := append([]string{}, relationships.internalRelationshipIDsByNode[projected.ID]...)
		layerValues := nodeLayers(projected.ModuleIDs, state.layersByModule)
		var layer *int
		if len(layerValues) == 1 {
			layerValue := layerValues[0]
			layer = &layerValue
		}

		confidenceState := "not_applicable"
		referenceScope := state.referenceNodeScopes[projected.ID]
		if referenceScope != "" {
			contributors := []analysis.RelationshipObservation{}
			if strings.HasPrefix(projected.ID, "reference-boundary:") {
				for _, detail := range referenceDetails {
					if detail.Scope != referenceScope {
						continue
					}
					for _, relationshipID := range detail.RelationshipIDs {
						if relationship, ok := state.relationshipsByID[relationshipID]; ok {
							contributors = append(contributors, relationship)
						}
					}
				}
			} else if detail, ok := referenceDetailsByID[projected.ID]; ok {
				for _, relationshipID := range detail.RelationshipIDs {
					if relationship, ok := state.relationshipsByID[relationshipID]; ok {
						contributors = append(contributors, relationship)
					}
				}
			}
			confidenceState, _, _ = aggregateConfidence(contributors)
		}

		visible := VisibleNode{
			ID:                      projected.ID,
			Kind:                    projected.Kind,
			Label:                   projected.Label,
			ModuleIDs:               append([]string{}, projected.ModuleIDs...),
			HierarchyPath:           append([]string{}, projected.Hierarchy...),
			ReferenceScope:          referenceScope,
			Tags:                    tags,
			CycleState:              cycleState,
			DiagnosticState:         diagnosticState,
			IdentityState:           "stable",
			Layer:                   layer,
			Layers:                  append([]int{}, layerValues...),
			ConfidenceState:         confidenceState,
			InternalRelationshipIDs: internalRelationshipIDs,
			EvidenceIDs:             append([]string{}, sourceIDs...),
			Counts: NodeCounts{
				ModuleCount:               len(projected.ModuleIDs),
				RelationshipCount:         relationships.nodeRelationshipCounts[projected.ID],
				InternalRelationshipCount: len(internalRelationshipIDs),
				EvidenceCount:             len(sourceIDs),
				DiagnosticCount:           len(diagnosticIDs),
			},
		}
		visible.AccessibleLabel = nodeAccessibleLabel(visible)
		visibleNodes = append(visibleNodes, visible)
	}
	return visibleNodes
}
