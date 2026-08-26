package scene

import (
	"sort"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
)

type sceneBuildState struct {
	value                  model.Model
	projection             model.HierarchyProjection
	options                SceneOptions
	allowedScopes          map[string]struct{}
	modulesByID            map[string]analysis.ModuleObservation
	relationshipsByID      map[string]analysis.RelationshipObservation
	referencesByID         map[string]analysis.Reference
	sourcesByID            map[string]analysis.SourceReference
	diagnosticsByID        map[string]model.Diagnostic
	moduleNodeIDs          map[string]string
	localProjectionNodes   []model.ProjectionNode
	referenceAccumulators  map[string]*referenceAccumulator
	visibleProjectionNodes []model.ProjectionNode
	referenceNodeScopes    map[string]string
	visibleNodeIDs         map[string]struct{}
	visibleProjectionByID  map[string]model.ProjectionNode
	cycleModules           map[string]struct{}
	cycleRelationships     map[string]struct{}
	feedbackRelationships  map[string]struct{}
	layersByModule         map[string]int
	sourceModuleIDs        map[string][]string
	diagnosticNodeIDs      map[string][]string
}

func newSceneBuildState(value model.Model, projection model.HierarchyProjection, options SceneOptions, allowedScopes map[string]struct{}) *sceneBuildState {
	state := &sceneBuildState{
		value:                 value,
		projection:            projection,
		options:               options,
		allowedScopes:         allowedScopes,
		modulesByID:           make(map[string]analysis.ModuleObservation, len(value.Modules)),
		relationshipsByID:     make(map[string]analysis.RelationshipObservation, len(value.Relationships)),
		referencesByID:        make(map[string]analysis.Reference, len(value.References)),
		sourcesByID:           make(map[string]analysis.SourceReference, len(value.SourceReferences)),
		diagnosticsByID:       make(map[string]model.Diagnostic, len(value.Diagnostics)),
		moduleNodeIDs:         make(map[string]string),
		referenceAccumulators: make(map[string]*referenceAccumulator),
		referenceNodeScopes:   make(map[string]string),
		cycleModules:          make(map[string]struct{}),
		cycleRelationships:    make(map[string]struct{}),
		feedbackRelationships: make(map[string]struct{}, len(value.Derived.FeedbackRelationshipIDs)),
		layersByModule:        make(map[string]int, len(value.Modules)),
		sourceModuleIDs:       make(map[string][]string),
		diagnosticNodeIDs:     make(map[string][]string, len(value.Diagnostics)),
	}
	state.indexModel()
	state.indexProjection()
	state.collectReferenceAccumulators()
	state.addVisibleReferenceNodes()
	state.indexVisibleProjection()
	state.indexDerivedFacts()
	state.indexDiagnosticNodes()
	return state
}

func (state *sceneBuildState) indexModel() {
	for _, module := range state.value.Modules {
		state.modulesByID[module.ID] = module
	}
	for _, relationship := range state.value.Relationships {
		state.relationshipsByID[relationship.ID] = relationship
	}
	for _, reference := range state.value.References {
		state.referencesByID[reference.ID] = reference
	}
	for _, source := range state.value.SourceReferences {
		state.sourcesByID[source.ID] = source
	}
	for _, diagnostic := range state.value.Diagnostics {
		state.diagnosticsByID[diagnostic.ID] = diagnostic
	}
}

func (state *sceneBuildState) indexProjection() {
	state.localProjectionNodes = make([]model.ProjectionNode, 0, len(state.projection.Nodes))
	for _, node := range state.projection.Nodes {
		if node.Kind == "reference" {
			continue
		}
		state.localProjectionNodes = append(state.localProjectionNodes, node)
		for _, moduleID := range node.ModuleIDs {
			state.moduleNodeIDs[moduleID] = node.ID
		}
	}
}

func (state *sceneBuildState) collectReferenceAccumulators() {
	for _, relationship := range state.value.Relationships {
		fromVisibleID, fromVisible := state.moduleNodeIDs[relationship.FromModuleID]
		if !fromVisible || relationship.ToReferenceID == "" {
			continue
		}
		reference, exists := state.referencesByID[relationship.ToReferenceID]
		if !exists || !scopeAllowed(reference.Scope, state.allowedScopes) {
			continue
		}
		accumulator := state.referenceAccumulators[reference.ID]
		if accumulator == nil {
			accumulator = &referenceAccumulator{Reference: reference}
			state.referenceAccumulators[reference.ID] = accumulator
		}
		accumulator.FromVisibleIDs = appendUniqueString(accumulator.FromVisibleIDs, fromVisibleID)
		accumulator.RelationshipIDs = appendUniqueString(accumulator.RelationshipIDs, relationship.ID)
		for _, sourceID := range relationship.SourceReferenceIDs {
			accumulator.SourceReferenceIDs = appendUniqueString(accumulator.SourceReferenceIDs, sourceID)
		}
	}
}

func (state *sceneBuildState) addVisibleReferenceNodes() {
	state.visibleProjectionNodes = append([]model.ProjectionNode{}, state.localProjectionNodes...)
	for _, referenceID := range sortedReferenceIDs(state.referenceAccumulators) {
		accumulator := state.referenceAccumulators[referenceID]
		switch state.options.ReferenceVisibility {
		case ReferenceVisibilityExpanded:
			state.visibleProjectionNodes = append(state.visibleProjectionNodes, model.ProjectionNode{
				ID: referenceID, Kind: "reference", Label: accumulator.Reference.Name, ModuleIDs: []string{},
			})
			state.referenceNodeScopes[referenceID] = accumulator.Reference.Scope
		case ReferenceVisibilityAggregated:
			boundaryID := referenceBoundaryID(accumulator.Reference.Scope)
			if _, exists := state.referenceNodeScopes[boundaryID]; !exists {
				state.visibleProjectionNodes = append(state.visibleProjectionNodes, model.ProjectionNode{
					ID: boundaryID, Kind: "reference", Label: referenceBoundaryLabel(accumulator.Reference.Scope), ModuleIDs: []string{},
				})
				state.referenceNodeScopes[boundaryID] = accumulator.Reference.Scope
			}
		}
	}
}

func (state *sceneBuildState) indexVisibleProjection() {
	state.visibleNodeIDs = make(map[string]struct{}, len(state.visibleProjectionNodes))
	state.visibleProjectionByID = make(map[string]model.ProjectionNode, len(state.visibleProjectionNodes))
	for _, node := range state.visibleProjectionNodes {
		state.visibleNodeIDs[node.ID] = struct{}{}
		state.visibleProjectionByID[node.ID] = node
	}
}

func (state *sceneBuildState) indexDerivedFacts() {
	for _, cycle := range state.value.Derived.Cycles {
		for _, moduleID := range cycle.ModuleIDs {
			state.cycleModules[moduleID] = struct{}{}
		}
		for _, relationshipID := range cycle.RelationshipIDs {
			state.cycleRelationships[relationshipID] = struct{}{}
		}
	}
	for _, relationshipID := range state.value.Derived.FeedbackRelationshipIDs {
		state.feedbackRelationships[relationshipID] = struct{}{}
	}
	for _, layer := range state.value.Derived.Layers {
		for _, moduleID := range layer.ModuleIDs {
			state.layersByModule[moduleID] = layer.Layer
		}
	}
	for _, module := range state.value.Modules {
		for _, sourceID := range module.SourceReferenceIDs {
			state.sourceModuleIDs[sourceID] = appendUniqueString(state.sourceModuleIDs[sourceID], module.ID)
		}
	}
	for sourceID := range state.sourceModuleIDs {
		sort.Strings(state.sourceModuleIDs[sourceID])
	}
}

func (state *sceneBuildState) indexDiagnosticNodes() {
	for _, diagnostic := range state.value.Diagnostics {
		for _, moduleID := range sourceModuleIDsForDiagnostic(diagnostic, state.sourceModuleIDs) {
			if nodeID, ok := state.moduleNodeIDs[moduleID]; ok {
				state.diagnosticNodeIDs[diagnostic.ID] = appendUniqueString(state.diagnosticNodeIDs[diagnostic.ID], nodeID)
			}
		}
		if diagnostic.Subject != "" {
			if nodeID, ok := state.moduleNodeIDs[diagnostic.Subject]; ok {
				state.diagnosticNodeIDs[diagnostic.ID] = appendUniqueString(state.diagnosticNodeIDs[diagnostic.ID], nodeID)
			}
		}
		sort.Strings(state.diagnosticNodeIDs[diagnostic.ID])
	}
}
