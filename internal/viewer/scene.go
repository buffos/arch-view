package viewer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
)

const SceneSchemaVersion = "arch-view.scene/v1"

const (
	ReferenceVisibilityHidden     = "hidden"
	ReferenceVisibilityAggregated = "aggregated"
	ReferenceVisibilityExpanded   = "expanded"
)

var referenceScopeOrder = []string{"standard_library", "external", "unresolved", "dynamic"}

// SceneSnapshot is the renderer-neutral projection consumed by the local web
// viewer. It deliberately carries semantic IDs and contributor IDs so a
// renderer never has to infer architecture facts from presentation geometry.
type SceneSnapshot struct {
	SchemaVersion        string                `json:"schema_version"`
	ModelID              string                `json:"model_id"`
	ModelRevision        string                `json:"model_revision"`
	Status               model.Status          `json:"status"`
	Project              SceneProject          `json:"project"`
	HierarchyPath        []string              `json:"hierarchy_path"`
	DisplayMode          string                `json:"display_mode"`
	ReferenceVisibility  string                `json:"reference_visibility"`
	VisibleNodes         []VisibleNode         `json:"visible_nodes"`
	VisibleRelationships []VisibleRelationship `json:"visible_relationships"`
	CycleIndicators      []CycleIndicator      `json:"cycle_indicators"`
	DiagnosticIndicators []DiagnosticIndicator `json:"diagnostic_indicators"`
	LayerLabels          []LayerLabel          `json:"layer_labels"`
	ReferenceSummary     ReferenceSummary      `json:"reference_summary"`
	ReferenceDetails     []ReferenceDetail     `json:"reference_details"`
	EvidenceLinks        []EvidenceLink        `json:"evidence_links"`
	Accessibility        Accessibility         `json:"accessibility"`
	Summary              SceneSummary          `json:"summary"`
}

type SceneOptions struct {
	ReferenceVisibility string
	ReferenceScopes     []string
}

type SceneProject struct {
	RootLabel string `json:"root_label"`
	Boundary  string `json:"boundary"`
	Language  string `json:"language"`
}

type VisibleNode struct {
	ID                      string     `json:"id"`
	Kind                    string     `json:"kind"`
	Label                   string     `json:"label"`
	ModuleIDs               []string   `json:"module_ids"`
	HierarchyPath           []string   `json:"hierarchy_path"`
	ReferenceScope          string     `json:"reference_scope,omitempty"`
	Tags                    []string   `json:"tags"`
	CycleState              string     `json:"cycle_state"`
	DiagnosticState         string     `json:"diagnostic_state"`
	IdentityState           string     `json:"identity_state"`
	Layer                   *int       `json:"layer,omitempty"`
	Layers                  []int      `json:"layers,omitempty"`
	ConfidenceState         string     `json:"confidence_state"`
	InternalRelationshipIDs []string   `json:"internal_relationship_ids,omitempty"`
	EvidenceIDs             []string   `json:"evidence_ids"`
	Counts                  NodeCounts `json:"counts"`
	AccessibleLabel         string     `json:"accessible_label"`
}

type NodeCounts struct {
	ModuleCount               int `json:"module_count"`
	RelationshipCount         int `json:"relationship_count"`
	InternalRelationshipCount int `json:"internal_relationship_count,omitempty"`
	EvidenceCount             int `json:"evidence_count"`
	DiagnosticCount           int `json:"diagnostic_count"`
}

type VisibleRelationship struct {
	ID                 string   `json:"id"`
	Type               string   `json:"type"`
	FromVisibleID      string   `json:"from_visible_id"`
	ToVisibleID        string   `json:"to_visible_id"`
	RelationshipIDs    []string `json:"contributor_relationship_ids"`
	SourceReferenceIDs []string `json:"evidence_ids"`
	Count              int      `json:"count"`
	Directed           bool     `json:"directed"`
	CycleState         string   `json:"cycle_state"`
	TargetScope        string   `json:"target_scope,omitempty"`
	ConfidenceState    string   `json:"confidence_state"`
	ConfidenceBasis    string   `json:"confidence_basis,omitempty"`
	ConfidenceScore    *float64 `json:"confidence_score,omitempty"`
	AccessibleLabel    string   `json:"accessible_label"`
}

type CycleIndicator struct {
	ID              string   `json:"id"`
	ModuleIDs       []string `json:"module_ids"`
	VisibleNodeIDs  []string `json:"visible_node_ids"`
	RelationshipIDs []string `json:"relationship_ids"`
	State           string   `json:"state"`
	Label           string   `json:"label"`
}

type DiagnosticIndicator struct {
	ID                 string   `json:"id"`
	Code               string   `json:"code"`
	Severity           string   `json:"severity"`
	Message            string   `json:"message"`
	Subject            string   `json:"subject,omitempty"`
	Path               string   `json:"path,omitempty"`
	VisibleNodeIDs     []string `json:"visible_node_ids"`
	SourceReferenceIDs []string `json:"evidence_ids"`
	Recoverable        bool     `json:"recoverable"`
	Label              string   `json:"label"`
}

type LayerLabel struct {
	Layer          int      `json:"layer"`
	Label          string   `json:"label"`
	ModuleIDs      []string `json:"module_ids"`
	VisibleNodeIDs []string `json:"visible_node_ids"`
}

type EvidenceLink struct {
	ID       string             `json:"id"`
	Path     string             `json:"path"`
	Start    *analysis.Position `json:"start,omitempty"`
	End      *analysis.Position `json:"end,omitempty"`
	Symbol   string             `json:"symbol,omitempty"`
	Kind     string             `json:"kind"`
	ReadOnly bool               `json:"read_only"`
}

type Accessibility struct {
	ReadingOrder []string          `json:"reading_order"`
	Descriptions map[string]string `json:"descriptions"`
}

type ReferenceSummary struct {
	Total             int                     `json:"total"`
	RelationshipCount int                     `json:"relationship_count"`
	VisibleCount      int                     `json:"visible_count"`
	HiddenCount       int                     `json:"hidden_count"`
	AggregatedCount   int                     `json:"aggregated_count"`
	ExpandedCount     int                     `json:"expanded_count"`
	ByScope           []ReferenceScopeSummary `json:"by_scope"`
}

type ReferenceScopeSummary struct {
	Scope             string `json:"scope"`
	ReferenceCount    int    `json:"reference_count"`
	RelationshipCount int    `json:"relationship_count"`
	VisibleNodeCount  int    `json:"visible_node_count"`
}

type ReferenceDetail struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Scope              string   `json:"scope"`
	FromVisibleIDs     []string `json:"from_visible_ids"`
	RelationshipIDs    []string `json:"relationship_ids"`
	SourceReferenceIDs []string `json:"evidence_ids"`
	Count              int      `json:"count"`
	ConfidenceState    string   `json:"confidence_state"`
	ConfidenceBasis    string   `json:"confidence_basis,omitempty"`
	ConfidenceScore    *float64 `json:"confidence_score,omitempty"`
	AccessibleLabel    string   `json:"accessible_label"`
}

type referenceAccumulator struct {
	Reference          analysis.Reference
	FromVisibleIDs     []string
	RelationshipIDs    []string
	SourceReferenceIDs []string
}

type SceneSummary struct {
	VisibleNodeCount         int `json:"visible_node_count"`
	VisibleRelationshipCount int `json:"visible_relationship_count"`
	ModuleCount              int `json:"module_count"`
	ReferenceCount           int `json:"reference_count"`
	CycleCount               int `json:"cycle_count"`
	DiagnosticCount          int `json:"diagnostic_count"`
	EvidenceCount            int `json:"evidence_count"`
}

// BuildScene converts the canonical model and its existing hierarchy
// projection into a semantic scene. The model is never mutated and all
// aggregation retains the canonical contributor IDs. The default projection
// is local-first: non-local references remain available as details but are
// hidden from the overview graph.
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
	if displayMode == "" {
		displayMode = "overview"
	}
	if !validDisplayMode(displayMode) {
		return SceneSnapshot{}, analysis.NewHostError(analysis.ErrInvalidRequest, "viewer display mode is unsupported", map[string]any{"mode": displayMode})
	}
	if options.ReferenceVisibility == "" {
		options.ReferenceVisibility = ReferenceVisibilityHidden
	}
	if !validReferenceVisibility(options.ReferenceVisibility) {
		return SceneSnapshot{}, analysis.NewHostError(analysis.ErrInvalidRequest, "viewer reference visibility is unsupported", map[string]any{"reference_visibility": options.ReferenceVisibility})
	}
	allowedScopes, err := referenceScopeFilter(options.ReferenceScopes)
	if err != nil {
		return SceneSnapshot{}, err
	}
	for _, segment := range selectedPath {
		if strings.TrimSpace(segment) == "" {
			return SceneSnapshot{}, analysis.NewHostError(analysis.ErrInvalidRequest, "viewer hierarchy path contains an empty segment", nil)
		}
	}

	projection, err := model.BuildHierarchyProjection(value, selectedPath)
	if err != nil {
		return SceneSnapshot{}, err
	}
	if len(selectedPath) > 0 && len(projection.Nodes) == 0 {
		return SceneSnapshot{}, analysis.NewHostError(analysis.ErrInvalidRequest, "viewer hierarchy path does not resolve", map[string]any{"path": selectedPath})
	}

	modulesByID := make(map[string]analysis.ModuleObservation, len(value.Modules))
	for _, module := range value.Modules {
		modulesByID[module.ID] = module
	}
	relationshipsByID := make(map[string]analysis.RelationshipObservation, len(value.Relationships))
	for _, relationship := range value.Relationships {
		relationshipsByID[relationship.ID] = relationship
	}
	referencesByID := make(map[string]analysis.Reference, len(value.References))
	for _, reference := range value.References {
		referencesByID[reference.ID] = reference
	}
	sourcesByID := make(map[string]analysis.SourceReference, len(value.SourceReferences))
	for _, source := range value.SourceReferences {
		sourcesByID[source.ID] = source
	}
	diagnosticsByID := make(map[string]model.Diagnostic, len(value.Diagnostics))
	for _, diagnostic := range value.Diagnostics {
		diagnosticsByID[diagnostic.ID] = diagnostic
	}

	moduleNodeIDs := make(map[string]string)
	localProjectionNodes := make([]model.ProjectionNode, 0, len(projection.Nodes))
	for _, node := range projection.Nodes {
		if node.Kind == "reference" {
			continue
		}
		localProjectionNodes = append(localProjectionNodes, node)
		for _, moduleID := range node.ModuleIDs {
			moduleNodeIDs[moduleID] = node.ID
		}
	}

	referenceAccumulators := make(map[string]*referenceAccumulator)
	for _, relationship := range value.Relationships {
		fromVisibleID, fromVisible := moduleNodeIDs[relationship.FromModuleID]
		if !fromVisible || relationship.ToReferenceID == "" {
			continue
		}
		reference, exists := referencesByID[relationship.ToReferenceID]
		if !exists || !scopeAllowed(reference.Scope, allowedScopes) {
			continue
		}
		accumulator := referenceAccumulators[reference.ID]
		if accumulator == nil {
			accumulator = &referenceAccumulator{Reference: reference}
			referenceAccumulators[reference.ID] = accumulator
		}
		accumulator.FromVisibleIDs = appendUniqueString(accumulator.FromVisibleIDs, fromVisibleID)
		accumulator.RelationshipIDs = appendUniqueString(accumulator.RelationshipIDs, relationship.ID)
		for _, sourceID := range relationship.SourceReferenceIDs {
			accumulator.SourceReferenceIDs = appendUniqueString(accumulator.SourceReferenceIDs, sourceID)
		}
	}

	visibleProjectionNodes := append([]model.ProjectionNode{}, localProjectionNodes...)
	referenceNodeScopes := make(map[string]string)
	for _, referenceID := range sortedReferenceIDs(referenceAccumulators) {
		accumulator := referenceAccumulators[referenceID]
		switch options.ReferenceVisibility {
		case ReferenceVisibilityExpanded:
			visibleProjectionNodes = append(visibleProjectionNodes, model.ProjectionNode{ID: referenceID, Kind: "reference", Label: accumulator.Reference.Name, ModuleIDs: []string{}})
			referenceNodeScopes[referenceID] = accumulator.Reference.Scope
		case ReferenceVisibilityAggregated:
			boundaryID := referenceBoundaryID(accumulator.Reference.Scope)
			if _, exists := referenceNodeScopes[boundaryID]; !exists {
				visibleProjectionNodes = append(visibleProjectionNodes, model.ProjectionNode{ID: boundaryID, Kind: "reference", Label: referenceBoundaryLabel(accumulator.Reference.Scope), ModuleIDs: []string{}})
				referenceNodeScopes[boundaryID] = accumulator.Reference.Scope
			}
		}
	}

	visibleNodeIDs := make(map[string]struct{}, len(visibleProjectionNodes))
	visibleProjectionNodesByID := make(map[string]model.ProjectionNode, len(visibleProjectionNodes))
	for _, node := range visibleProjectionNodes {
		visibleNodeIDs[node.ID] = struct{}{}
		visibleProjectionNodesByID[node.ID] = node
	}

	cycleModules := make(map[string]struct{})
	cycleRelationships := make(map[string]struct{})
	for _, cycle := range value.Derived.Cycles {
		for _, moduleID := range cycle.ModuleIDs {
			cycleModules[moduleID] = struct{}{}
		}
		for _, relationshipID := range cycle.RelationshipIDs {
			cycleRelationships[relationshipID] = struct{}{}
		}
	}
	feedbackRelationships := make(map[string]struct{}, len(value.Derived.FeedbackRelationshipIDs))
	for _, relationshipID := range value.Derived.FeedbackRelationshipIDs {
		feedbackRelationships[relationshipID] = struct{}{}
	}

	layersByModule := make(map[string]int, len(value.Modules))
	for _, layer := range value.Derived.Layers {
		for _, moduleID := range layer.ModuleIDs {
			layersByModule[moduleID] = layer.Layer
		}
	}

	sourceModuleIDs := make(map[string][]string)
	for _, module := range value.Modules {
		for _, sourceID := range module.SourceReferenceIDs {
			sourceModuleIDs[sourceID] = appendUniqueString(sourceModuleIDs[sourceID], module.ID)
		}
	}
	for sourceID := range sourceModuleIDs {
		sort.Strings(sourceModuleIDs[sourceID])
	}

	diagnosticNodeIDs := make(map[string][]string, len(value.Diagnostics))
	for _, diagnostic := range value.Diagnostics {
		for _, moduleID := range sourceModuleIDsForDiagnostic(diagnostic, sourceModuleIDs) {
			if nodeID, ok := moduleNodeIDs[moduleID]; ok {
				diagnosticNodeIDs[diagnostic.ID] = appendUniqueString(diagnosticNodeIDs[diagnostic.ID], nodeID)
			}
		}
		if diagnostic.Subject != "" {
			if nodeID, ok := moduleNodeIDs[diagnostic.Subject]; ok {
				diagnosticNodeIDs[diagnostic.ID] = appendUniqueString(diagnosticNodeIDs[diagnostic.ID], nodeID)
			}
		}
		sort.Strings(diagnosticNodeIDs[diagnostic.ID])
	}

	nodeRelationshipCounts := make(map[string]int, len(visibleProjectionNodes))
	internalRelationshipIDsByNode := make(map[string][]string, len(visibleProjectionNodes))
	nodeEvidenceIDs := make(map[string][]string, len(visibleProjectionNodes))
	type relationshipAccumulator struct {
		Type               string
		FromVisibleID      string
		ToVisibleID        string
		TargetScope        string
		RelationshipIDs    []string
		SourceReferenceIDs []string
	}
	relationshipAccumulators := make(map[string]*relationshipAccumulator)
	for _, projected := range projection.Relationships {
		if _, exists := visibleNodeIDs[projected.FromNodeID]; !exists {
			continue
		}
		toVisibleID := projected.ToNodeID
		targetScope := ""
		if reference, isReference := referencesByID[projected.ToNodeID]; isReference {
			if !scopeAllowed(reference.Scope, allowedScopes) || options.ReferenceVisibility == ReferenceVisibilityHidden {
				continue
			}
			targetScope = reference.Scope
			if options.ReferenceVisibility == ReferenceVisibilityAggregated {
				toVisibleID = referenceBoundaryID(reference.Scope)
			}
		}
		if _, exists := visibleNodeIDs[toVisibleID]; !exists {
			continue
		}
		key := projected.Type + "\x00" + projected.FromNodeID + "\x00" + toVisibleID
		accumulator := relationshipAccumulators[key]
		if accumulator == nil {
			accumulator = &relationshipAccumulator{Type: projected.Type, FromVisibleID: projected.FromNodeID, ToVisibleID: toVisibleID, TargetScope: targetScope}
			relationshipAccumulators[key] = accumulator
		}
		for _, relationshipID := range projected.RelationshipIDs {
			accumulator.RelationshipIDs = appendUniqueString(accumulator.RelationshipIDs, relationshipID)
		}
		for _, sourceID := range projected.SourceReferenceIDs {
			accumulator.SourceReferenceIDs = appendUniqueString(accumulator.SourceReferenceIDs, sourceID)
		}
	}
	relationshipKeys := make([]string, 0, len(relationshipAccumulators))
	for key := range relationshipAccumulators {
		relationshipKeys = append(relationshipKeys, key)
	}
	sort.Strings(relationshipKeys)
	visibleRelationships := make([]VisibleRelationship, 0, len(relationshipKeys))
	for _, key := range relationshipKeys {
		accumulator := relationshipAccumulators[key]
		contributors := make([]analysis.RelationshipObservation, 0, len(accumulator.RelationshipIDs))
		for _, relationshipID := range accumulator.RelationshipIDs {
			if relationship, ok := relationshipsByID[relationshipID]; ok {
				contributors = append(contributors, relationship)
			}
		}
		confidenceState, confidenceBasis, confidenceScore := aggregateConfidence(contributors)
		cycleState := relationshipCycleState(accumulator.RelationshipIDs, cycleRelationships, feedbackRelationships)
		projectedNode, hasProjectedNode := visibleProjectionNodesByID[accumulator.FromVisibleID]
		if hasProjectedNode && shouldSummarizeInternalRelationship(projectedNode, accumulator.FromVisibleID, accumulator.ToVisibleID, cycleState, contributors) {
			for _, relationshipID := range accumulator.RelationshipIDs {
				internalRelationshipIDsByNode[accumulator.FromVisibleID] = appendUniqueString(internalRelationshipIDsByNode[accumulator.FromVisibleID], relationshipID)
			}
			for _, sourceID := range accumulator.SourceReferenceIDs {
				nodeEvidenceIDs[accumulator.FromVisibleID] = appendUniqueString(nodeEvidenceIDs[accumulator.FromVisibleID], sourceID)
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
		visible.AccessibleLabel = relationshipAccessibleLabel(visible, projectionNodeLabel(visibleProjectionNodes, visible.FromVisibleID), projectionNodeLabel(visibleProjectionNodes, visible.ToVisibleID))
		visibleRelationships = append(visibleRelationships, visible)
		nodeRelationshipCounts[accumulator.FromVisibleID]++
		if accumulator.ToVisibleID != accumulator.FromVisibleID {
			nodeRelationshipCounts[accumulator.ToVisibleID]++
		}
		for _, sourceID := range accumulator.SourceReferenceIDs {
			nodeEvidenceIDs[accumulator.FromVisibleID] = appendUniqueString(nodeEvidenceIDs[accumulator.FromVisibleID], sourceID)
			nodeEvidenceIDs[accumulator.ToVisibleID] = appendUniqueString(nodeEvidenceIDs[accumulator.ToVisibleID], sourceID)
		}
	}
	for nodeID := range nodeEvidenceIDs {
		sort.Strings(nodeEvidenceIDs[nodeID])
	}
	for nodeID := range internalRelationshipIDsByNode {
		sort.Strings(internalRelationshipIDsByNode[nodeID])
	}

	referenceDetails := buildReferenceDetails(referenceAccumulators, relationshipsByID)
	referenceDetailsByID := make(map[string]ReferenceDetail, len(referenceDetails))
	for _, detail := range referenceDetails {
		referenceDetailsByID[detail.ID] = detail
	}

	visibleNodes := make([]VisibleNode, 0, len(visibleProjectionNodes))
	for _, projected := range visibleProjectionNodes {
		tags := []string{}
		sourceIDs := append([]string{}, nodeEvidenceIDs[projected.ID]...)
		for _, moduleID := range projected.ModuleIDs {
			module, ok := modulesByID[moduleID]
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
		cycleState := nodeCycleState(projected.ModuleIDs, cycleModules, feedbackRelationships, value.Relationships)
		diagnosticIDs := diagnosticsForVisibleNode(projected.ID, diagnosticNodeIDs)
		diagnosticState := diagnosticsState(diagnosticIDs, diagnosticsByID)
		internalRelationshipIDs := append([]string{}, internalRelationshipIDsByNode[projected.ID]...)
		layerValues := nodeLayers(projected.ModuleIDs, layersByModule)
		var layer *int
		if len(layerValues) == 1 {
			value := layerValues[0]
			layer = &value
		}
		confidenceState := "not_applicable"
		referenceScope := referenceNodeScopes[projected.ID]
		if referenceScope != "" {
			contributors := []analysis.RelationshipObservation{}
			if strings.HasPrefix(projected.ID, "reference-boundary:") {
				for _, detail := range referenceDetails {
					if detail.Scope != referenceScope {
						continue
					}
					for _, relationshipID := range detail.RelationshipIDs {
						if relationship, ok := relationshipsByID[relationshipID]; ok {
							contributors = append(contributors, relationship)
						}
					}
				}
			} else if detail, ok := referenceDetailsByID[projected.ID]; ok {
				for _, relationshipID := range detail.RelationshipIDs {
					if relationship, ok := relationshipsByID[relationshipID]; ok {
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
				RelationshipCount:         nodeRelationshipCounts[projected.ID],
				InternalRelationshipCount: len(internalRelationshipIDs),
				EvidenceCount:             len(sourceIDs),
				DiagnosticCount:           len(diagnosticIDs),
			},
		}
		visible.AccessibleLabel = nodeAccessibleLabel(visible)
		visibleNodes = append(visibleNodes, visible)
	}

	cycleIndicators := buildCycleIndicators(value.Derived.Cycles, moduleNodeIDs, visibleNodeIDs)
	diagnosticIndicators := buildDiagnosticIndicators(value.Diagnostics, diagnosticNodeIDs, visibleNodeIDs)
	layerLabels := buildLayerLabels(value.Derived.Layers, moduleNodeIDs, visibleNodeIDs)
	evidenceLinks := buildEvidenceLinks(sourcesByID, visibleNodes, visibleRelationships, diagnosticIndicators, referenceDetails)

	accessibility := Accessibility{
		ReadingOrder: []string{},
		Descriptions: map[string]string{},
	}
	for _, node := range visibleNodes {
		accessibility.ReadingOrder = append(accessibility.ReadingOrder, node.ID)
		accessibility.Descriptions[node.ID] = node.AccessibleLabel
	}
	for _, relationship := range visibleRelationships {
		accessibility.ReadingOrder = append(accessibility.ReadingOrder, relationship.ID)
		accessibility.Descriptions[relationship.ID] = relationship.AccessibleLabel
	}
	for _, cycle := range cycleIndicators {
		accessibility.ReadingOrder = append(accessibility.ReadingOrder, cycle.ID)
		accessibility.Descriptions[cycle.ID] = cycle.Label
	}
	for _, diagnostic := range diagnosticIndicators {
		accessibility.ReadingOrder = append(accessibility.ReadingOrder, diagnostic.ID)
		accessibility.Descriptions[diagnostic.ID] = diagnostic.Label
	}
	for _, detail := range referenceDetails {
		accessibilityID := "reference-detail:" + detail.ID
		accessibility.ReadingOrder = append(accessibility.ReadingOrder, accessibilityID)
		accessibility.Descriptions[accessibilityID] = detail.AccessibleLabel
	}

	referenceSummary := buildReferenceSummary(referenceAccumulators, options.ReferenceVisibility)

	snapshot := SceneSnapshot{
		SchemaVersion:        SceneSchemaVersion,
		ModelID:              value.ModelID,
		ModelRevision:        value.ModelID,
		Status:               value.Status,
		Project:              SceneProject{RootLabel: value.Project.RootLabel, Boundary: value.Project.Boundary, Language: value.Project.Language},
		HierarchyPath:        append([]string{}, selectedPath...),
		DisplayMode:          displayMode,
		ReferenceVisibility:  options.ReferenceVisibility,
		VisibleNodes:         visibleNodes,
		VisibleRelationships: visibleRelationships,
		CycleIndicators:      cycleIndicators,
		DiagnosticIndicators: diagnosticIndicators,
		LayerLabels:          layerLabels,
		ReferenceSummary:     referenceSummary,
		ReferenceDetails:     referenceDetails,
		EvidenceLinks:        evidenceLinks,
		Accessibility:        accessibility,
		Summary: SceneSummary{
			VisibleNodeCount:         len(visibleNodes),
			VisibleRelationshipCount: len(visibleRelationships),
			ModuleCount:              len(value.Modules),
			ReferenceCount:           len(value.References),
			CycleCount:               len(cycleIndicators),
			DiagnosticCount:          len(diagnosticIndicators),
			EvidenceCount:            len(evidenceLinks),
		},
	}
	return snapshot, nil
}

func validDisplayMode(mode string) bool {
	switch mode {
	case "overview", "detail", "list":
		return true
	default:
		return false
	}
}

func validReferenceVisibility(value string) bool {
	switch value {
	case ReferenceVisibilityHidden, ReferenceVisibilityAggregated, ReferenceVisibilityExpanded:
		return true
	default:
		return false
	}
}

func referenceScopeFilter(values []string) (map[string]struct{}, error) {
	if len(values) == 0 {
		return nil, nil
	}
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !validReferenceScope(value) {
			return nil, analysis.NewHostError(analysis.ErrInvalidRequest, "viewer reference scope is unsupported", map[string]any{"reference_scope": value})
		}
		result[value] = struct{}{}
	}
	return result, nil
}

func validReferenceScope(value string) bool {
	switch value {
	case "standard_library", "external", "unresolved", "dynamic":
		return true
	default:
		return false
	}
}

func scopeAllowed(scope string, allowed map[string]struct{}) bool {
	if len(allowed) == 0 {
		return true
	}
	_, ok := allowed[scope]
	return ok
}

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

func nodeAccessibleLabel(node VisibleNode) string {
	parts := []string{fmt.Sprintf("%s %s", node.Kind, node.Label)}
	if node.ReferenceScope != "" {
		parts = append(parts, node.ReferenceScope+" reference scope")
	}
	if node.Counts.ModuleCount > 0 {
		parts = append(parts, fmt.Sprintf("%d module(s)", node.Counts.ModuleCount))
	}
	if node.Layer != nil {
		parts = append(parts, fmt.Sprintf("layer %d", *node.Layer))
	} else if len(node.Layers) > 0 {
		parts = append(parts, fmt.Sprintf("layers %s", joinInts(node.Layers)))
	}
	if node.CycleState != "none" {
		parts = append(parts, node.CycleState)
	}
	if node.DiagnosticState != "none" {
		parts = append(parts, node.DiagnosticState+" diagnostics")
	}
	parts = append(parts, node.IdentityState+" identity")
	if node.Counts.InternalRelationshipCount > 0 {
		parts = append(parts, fmt.Sprintf("%d internal relationship(s)", node.Counts.InternalRelationshipCount))
	}
	if node.ReferenceScope != "" {
		parts = append(parts, node.ConfidenceState+" confidence")
	}
	return strings.Join(parts, "; ")
}

func relationshipAccessibleLabel(relationship VisibleRelationship, fromLabel, toLabel string) string {
	parts := []string{fmt.Sprintf("%s from %s to %s", relationship.Type, fromLabel, toLabel), fmt.Sprintf("%d contributor(s)", relationship.Count), relationship.ConfidenceState + " confidence"}
	if relationship.TargetScope != "" {
		parts = append(parts, relationship.TargetScope+" reference scope")
	}
	if relationship.CycleState != "none" {
		parts = append(parts, relationship.CycleState)
	}
	return strings.Join(parts, "; ")
}

func joinInts(values []int) string {
	parts := make([]string, len(values))
	for index, value := range values {
		parts[index] = fmt.Sprint(value)
	}
	return strings.Join(parts, ", ")
}

func appendUniqueString(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
