package scene

import (
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
)

func BuildScene(value model.Model, selectedPath []string, displayMode string) (SceneSnapshot, error) {
	return BuildSceneWithOptions(value, selectedPath, displayMode, SceneOptions{ReferenceVisibility: ReferenceVisibilityHidden})
}

func BuildSceneWithOptions(value model.Model, selectedPath []string, displayMode string, options SceneOptions) (SceneSnapshot, error) {
	return buildScene(value, selectedPath, displayMode, options)
}

func buildScene(value model.Model, selectedPath []string, displayMode string, options SceneOptions) (SceneSnapshot, error) {
	if err := model.Validate(value); err != nil {
		return SceneSnapshot{}, err
	}
	request, err := normalizeSceneRequest(selectedPath, displayMode, options)
	if err != nil {
		return SceneSnapshot{}, err
	}
	projection, err := model.BuildHierarchyProjection(value, selectedPath)
	if err != nil {
		return SceneSnapshot{}, err
	}
	if len(selectedPath) > 0 && len(projection.Nodes) == 0 {
		return SceneSnapshot{}, analysis.NewHostError(analysis.ErrInvalidRequest, "viewer hierarchy path does not resolve", map[string]any{"path": selectedPath})
	}

	state := newSceneBuildState(value, projection, request.options, request.allowedScopes)
	relationships := state.buildVisibleRelationships()
	referenceDetails := buildReferenceDetails(state.referenceAccumulators, state.relationshipsByID)
	visibleNodes := state.buildVisibleNodes(relationships, referenceDetails)
	cycleIndicators := buildCycleIndicators(value.Derived.Cycles, state.moduleNodeIDs, state.visibleNodeIDs)
	diagnosticIndicators := buildDiagnosticIndicators(value.Diagnostics, state.diagnosticNodeIDs, state.visibleNodeIDs)
	layerLabels := buildLayerLabels(value.Derived.Layers, state.moduleNodeIDs, state.visibleNodeIDs)
	evidenceLinks := buildEvidenceLinks(state.sourcesByID, visibleNodes, relationships.visible, diagnosticIndicators, referenceDetails)
	accessibility := buildAccessibility(visibleNodes, relationships.visible, cycleIndicators, diagnosticIndicators, referenceDetails)

	return SceneSnapshot{
		SchemaVersion:        SceneSchemaVersion,
		ModelID:              value.ModelID,
		ModelRevision:        value.ModelID,
		Status:               value.Status,
		Project:              SceneProject{RootLabel: value.Project.RootLabel, Boundary: value.Project.Boundary, Language: value.Project.Language},
		HierarchyPath:        append([]string{}, selectedPath...),
		DisplayMode:          request.displayMode,
		ReferenceVisibility:  request.options.ReferenceVisibility,
		VisibleNodes:         visibleNodes,
		VisibleRelationships: relationships.visible,
		CycleIndicators:      cycleIndicators,
		DiagnosticIndicators: diagnosticIndicators,
		LayerLabels:          layerLabels,
		ReferenceSummary:     buildReferenceSummary(state.referenceAccumulators, request.options.ReferenceVisibility),
		ReferenceDetails:     referenceDetails,
		EvidenceLinks:        evidenceLinks,
		Accessibility:        accessibility,
		Summary: SceneSummary{
			VisibleNodeCount:         len(visibleNodes),
			VisibleRelationshipCount: len(relationships.visible),
			ModuleCount:              len(value.Modules),
			ReferenceCount:           len(value.References),
			CycleCount:               len(cycleIndicators),
			DiagnosticCount:          len(diagnosticIndicators),
			EvidenceCount:            len(evidenceLinks),
		},
	}, nil
}

type normalizedSceneRequest struct {
	displayMode   string
	options       SceneOptions
	allowedScopes map[string]struct{}
}

func normalizeSceneRequest(selectedPath []string, displayMode string, options SceneOptions) (normalizedSceneRequest, error) {
	if displayMode == "" {
		displayMode = "overview"
	}
	if !validDisplayMode(displayMode) {
		return normalizedSceneRequest{}, analysis.NewHostError(analysis.ErrInvalidRequest, "viewer display mode is unsupported", map[string]any{"mode": displayMode})
	}
	if options.ReferenceVisibility == "" {
		options.ReferenceVisibility = ReferenceVisibilityHidden
	}
	if !validReferenceVisibility(options.ReferenceVisibility) {
		return normalizedSceneRequest{}, analysis.NewHostError(analysis.ErrInvalidRequest, "viewer reference visibility is unsupported", map[string]any{"reference_visibility": options.ReferenceVisibility})
	}
	allowedScopes, err := referenceScopeFilter(options.ReferenceScopes)
	if err != nil {
		return normalizedSceneRequest{}, err
	}
	for _, segment := range selectedPath {
		if strings.TrimSpace(segment) == "" {
			return normalizedSceneRequest{}, analysis.NewHostError(analysis.ErrInvalidRequest, "viewer hierarchy path contains an empty segment", nil)
		}
	}
	return normalizedSceneRequest{displayMode: displayMode, options: options, allowedScopes: allowedScopes}, nil
}
