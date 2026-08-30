package model

import "github.com/buffo/arch-view/internal/analysis"

const SchemaVersion = "arch-view.model/v1"

type Status string

const (
	StatusComplete Status = "complete"
	StatusPartial  Status = "partial"
	StatusFailed   Status = "failed"
)

type Model struct {
	SchemaVersion    string                             `json:"schema_version"`
	ModelID          string                             `json:"model_id"`
	Status           Status                             `json:"status"`
	Project          Project                            `json:"project"`
	Analyzer         analysis.AnalyzerInfo              `json:"analyzer"`
	Modules          []analysis.ModuleObservation       `json:"modules"`
	References       []analysis.Reference               `json:"references"`
	SourceReferences []analysis.SourceReference         `json:"source_references"`
	Relationships    []analysis.RelationshipObservation `json:"relationships"`
	Diagnostics      []Diagnostic                       `json:"diagnostics"`
	Derived          Derived                            `json:"derived"`
	SourceIndex      *analysis.SourceIndex              `json:"source_index,omitempty"`
}

type Project struct {
	RootLabel string `json:"root_label"`
	Boundary  string `json:"boundary"`
	Language  string `json:"language"`
}

type Diagnostic struct {
	ID                 string             `json:"id"`
	Code               string             `json:"code"`
	Severity           string             `json:"severity"`
	Message            string             `json:"message"`
	Subject            string             `json:"subject,omitempty"`
	Path               string             `json:"path,omitempty"`
	Location           *analysis.Position `json:"location,omitempty"`
	SourceReferenceIDs []string           `json:"source_reference_ids"`
	Recoverable        bool               `json:"recoverable"`
	Metadata           map[string]any     `json:"metadata,omitempty"`
}

type Derived struct {
	Cycles                  []CycleGroup   `json:"cycles"`
	FeedbackRelationshipIDs []string       `json:"feedback_relationship_ids"`
	Layers                  []Layer        `json:"layers"`
	AlgorithmProvenance     map[string]any `json:"algorithm_provenance"`
}

type CycleGroup struct {
	ID              string   `json:"id"`
	ModuleIDs       []string `json:"module_ids"`
	RelationshipIDs []string `json:"relationship_ids"`
}

type Layer struct {
	Layer     int      `json:"layer"`
	ModuleIDs []string `json:"module_ids"`
}

type HierarchyProjection struct {
	Path          []string                 `json:"path"`
	Nodes         []ProjectionNode         `json:"nodes"`
	Relationships []ProjectionRelationship `json:"relationships"`
}

type ProjectionNode struct {
	ID        string   `json:"id"`
	Kind      string   `json:"kind"`
	Label     string   `json:"label"`
	Hierarchy []string `json:"hierarchy"`
	ModuleIDs []string `json:"module_ids"`
}

type ProjectionRelationship struct {
	ID                 string   `json:"id"`
	Type               string   `json:"type"`
	FromNodeID         string   `json:"from_node_id"`
	ToNodeID           string   `json:"to_node_id"`
	RelationshipIDs    []string `json:"relationship_ids"`
	SourceReferenceIDs []string `json:"source_reference_ids"`
	Count              int      `json:"count"`
}
