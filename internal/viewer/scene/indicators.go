package scene

import (
	"fmt"
	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
	"sort"
	"strings"
)

func buildCycleIndicators(cycles []model.CycleGroup, moduleNodeIDs map[string]string, visibleNodeIDs map[string]struct{}) []CycleIndicator {
	result := make([]CycleIndicator, 0, len(cycles))
	for _, cycle := range cycles {
		visible := []string{}
		for _, moduleID := range cycle.ModuleIDs {
			if nodeID, ok := moduleNodeIDs[moduleID]; ok {
				if _, exists := visibleNodeIDs[nodeID]; exists {
					visible = appendUniqueString(visible, nodeID)
				}
			}
		}
		if len(visible) == 0 {
			continue
		}
		sort.Strings(visible)
		moduleIDs := append([]string{}, cycle.ModuleIDs...)
		relationshipIDs := append([]string{}, cycle.RelationshipIDs...)
		sort.Strings(moduleIDs)
		sort.Strings(relationshipIDs)
		result = append(result, CycleIndicator{
			ID:              cycle.ID,
			ModuleIDs:       moduleIDs,
			VisibleNodeIDs:  visible,
			RelationshipIDs: relationshipIDs,
			State:           "cycle",
			Label:           fmt.Sprintf("Cycle involving %d module(s) and %d relationship(s).", len(moduleIDs), len(relationshipIDs)),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func buildDiagnosticIndicators(diagnostics []model.Diagnostic, diagnosticNodeIDs map[string][]string, visibleNodeIDs map[string]struct{}) []DiagnosticIndicator {
	result := make([]DiagnosticIndicator, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		nodeIDs := []string{}
		for _, nodeID := range diagnosticNodeIDs[diagnostic.ID] {
			if _, ok := visibleNodeIDs[nodeID]; ok {
				nodeIDs = appendUniqueString(nodeIDs, nodeID)
			}
		}
		sort.Strings(nodeIDs)
		sourceIDs := append([]string{}, diagnostic.SourceReferenceIDs...)
		sort.Strings(sourceIDs)
		label := fmt.Sprintf("%s: %s", strings.ToUpper(diagnostic.Severity), diagnostic.Message)
		result = append(result, DiagnosticIndicator{
			ID:                 diagnostic.ID,
			Code:               diagnostic.Code,
			Severity:           diagnostic.Severity,
			Message:            diagnostic.Message,
			Subject:            diagnostic.Subject,
			Path:               diagnostic.Path,
			VisibleNodeIDs:     nodeIDs,
			SourceReferenceIDs: sourceIDs,
			Recoverable:        diagnostic.Recoverable,
			Label:              label,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func buildLayerLabels(layers []model.Layer, moduleNodeIDs map[string]string, visibleNodeIDs map[string]struct{}) []LayerLabel {
	result := make([]LayerLabel, 0, len(layers))
	for _, layer := range layers {
		visible := []string{}
		for _, moduleID := range layer.ModuleIDs {
			if nodeID, ok := moduleNodeIDs[moduleID]; ok {
				if _, exists := visibleNodeIDs[nodeID]; exists {
					visible = appendUniqueString(visible, nodeID)
				}
			}
		}
		if len(visible) == 0 {
			continue
		}
		sort.Strings(visible)
		moduleIDs := append([]string{}, layer.ModuleIDs...)
		sort.Strings(moduleIDs)
		result = append(result, LayerLabel{
			Layer:          layer.Layer,
			Label:          fmt.Sprintf("Layer %d", layer.Layer),
			ModuleIDs:      moduleIDs,
			VisibleNodeIDs: visible,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Layer < result[j].Layer })
	return result
}

func buildEvidenceLinks(sourcesByID map[string]analysis.SourceReference, nodes []VisibleNode, relationships []VisibleRelationship, diagnostics []DiagnosticIndicator, referenceDetails ...[]ReferenceDetail) []EvidenceLink {
	ids := map[string]struct{}{}
	for _, node := range nodes {
		for _, sourceID := range node.EvidenceIDs {
			ids[sourceID] = struct{}{}
		}
	}
	for _, relationship := range relationships {
		for _, sourceID := range relationship.SourceReferenceIDs {
			ids[sourceID] = struct{}{}
		}
	}
	for _, diagnostic := range diagnostics {
		for _, sourceID := range diagnostic.SourceReferenceIDs {
			ids[sourceID] = struct{}{}
		}
	}
	if len(referenceDetails) > 0 {
		for _, detail := range referenceDetails[0] {
			for _, sourceID := range detail.SourceReferenceIDs {
				ids[sourceID] = struct{}{}
			}
		}
	}
	result := make([]EvidenceLink, 0, len(ids))
	for sourceID := range ids {
		source, ok := sourcesByID[sourceID]
		if !ok {
			continue
		}
		result = append(result, EvidenceLink{ID: source.ID, Path: source.Path, Start: source.Start, End: source.End, Symbol: source.Symbol, Kind: source.Kind, ReadOnly: true})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func projectionNodeLabel(nodes []model.ProjectionNode, nodeID string) string {
	for _, node := range nodes {
		if node.ID == nodeID {
			return node.Label
		}
	}
	return nodeID
}
