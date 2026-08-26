package scene

import (
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
