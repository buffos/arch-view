package analysis

import "context"

const AnalyzerAPIVersion = "arch-view.analyzer/v1"

type Analyzer interface {
	Manifest() Manifest
	Detect(context.Context, DetectRequest) (DetectionCandidate, error)
	Analyze(context.Context, AnalyzeRequest) (AnalysisResult, error)
}

type DetectRequest struct {
	ProjectRoot string `json:"project_root"`
}

type AnalyzeRequest struct {
	ProjectRoot string            `json:"project_root"`
	Selection   AnalyzerSelection `json:"selection"`
	Options     EffectiveOptions  `json:"options"`
}

type RunRequest struct {
	ProjectRoot    string
	Language       string
	AnalyzerID     string
	ProjectOptions map[string]any
	CLIOptions     map[string]any
}

type AnalyzerSelection struct {
	AnalyzerID     string   `json:"analyzer_id"`
	Mode           string   `json:"mode"`
	Confidence     float64  `json:"confidence"`
	MatchedMarkers []string `json:"matched_markers,omitempty"`
	Reason         string   `json:"reason,omitempty"`
	BoundaryHint   string   `json:"boundary_hint,omitempty"`
}

type Manifest struct {
	ID               string             `json:"id"`
	Version          string             `json:"version"`
	Language         string             `json:"language"`
	APIVersion       string             `json:"api_version"`
	DetectionMarkers []DetectionMarker  `json:"detection_markers"`
	Capabilities     []string           `json:"capabilities"`
	Options          []OptionDescriptor `json:"options"`
}

type DetectionMarker struct {
	Kind   string  `json:"kind"`
	Value  string  `json:"value"`
	Weight float64 `json:"weight"`
}

type OptionDescriptor struct {
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	Default       any      `json:"default"`
	AllowedValues []string `json:"allowed_values,omitempty"`
	Description   string   `json:"description,omitempty"`
	Sensitive     bool     `json:"sensitive"`
}

type DetectionCandidate struct {
	AnalyzerID     string   `json:"analyzer_id"`
	Confidence     float64  `json:"confidence"`
	MatchedMarkers []string `json:"matched_markers,omitempty"`
	BoundaryHint   string   `json:"boundary_hint,omitempty"`
	Reason         string   `json:"reason"`
}

type EffectiveOptions struct {
	Values      map[string]any    `json:"values"`
	Sources     map[string]string `json:"sources,omitempty"`
	Fingerprint string            `json:"fingerprint"`
}

type AnalysisStatus string

const (
	StatusComplete  AnalysisStatus = "complete"
	StatusPartial   AnalysisStatus = "partial"
	StatusFailed    AnalysisStatus = "failed"
	StatusCancelled AnalysisStatus = "cancelled"
)

type AnalysisResult struct {
	RunID              string                    `json:"run_id"`
	Status             AnalysisStatus            `json:"status"`
	Analyzer           AnalyzerInfo              `json:"analyzer"`
	Project            ProjectInfo               `json:"project"`
	OptionsFingerprint string                    `json:"options_fingerprint,omitempty"`
	Modules            []ModuleObservation       `json:"modules"`
	Relationships      []RelationshipObservation `json:"relationships"`
	References         []Reference               `json:"references"`
	SourceReferences   []SourceReference         `json:"source_references"`
	Diagnostics        []Diagnostic              `json:"diagnostics"`
	Summary            AnalysisSummary           `json:"summary"`
}

type AnalyzerInfo struct {
	ID         string `json:"id"`
	Version    string `json:"version"`
	Language   string `json:"language"`
	APIVersion string `json:"api_version"`
}

type ProjectInfo struct {
	RootLabel     string `json:"root_label"`
	Boundary      string `json:"boundary"`
	ModulePath    string `json:"module_path,omitempty"`
	ModuleRoot    string `json:"module_root,omitempty"`
	WorkspacePath string `json:"workspace_path,omitempty"`
}

type ModuleObservation struct {
	ID                 string         `json:"id"`
	Language           string         `json:"language"`
	Kind               string         `json:"kind"`
	Name               string         `json:"name"`
	DisplayName        string         `json:"display_name"`
	Hierarchy          []string       `json:"hierarchy"`
	SourceReferenceIDs []string       `json:"source_reference_ids"`
	Tags               []string       `json:"tags"`
	Metadata           map[string]any `json:"metadata,omitempty"`
}

type RelationshipObservation struct {
	ID                 string         `json:"id"`
	Type               string         `json:"type"`
	FromModuleID       string         `json:"from_module_id"`
	ToModuleID         string         `json:"to_module_id,omitempty"`
	ToReferenceID      string         `json:"to_reference_id,omitempty"`
	SourceReferenceIDs []string       `json:"source_reference_ids"`
	Confidence         *Confidence    `json:"confidence,omitempty"`
	Metadata           map[string]any `json:"metadata,omitempty"`
}

type Reference struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Scope    string         `json:"scope"`
	Language string         `json:"language"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type SourceReference struct {
	ID     string    `json:"id"`
	Path   string    `json:"path"`
	Start  *Position `json:"start,omitempty"`
	End    *Position `json:"end,omitempty"`
	Symbol string    `json:"symbol,omitempty"`
	Kind   string    `json:"kind"`
}

type Position struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type Confidence struct {
	Basis string  `json:"basis"`
	Score float64 `json:"score,omitempty"`
}

type Diagnostic struct {
	Code        string         `json:"code"`
	Severity    string         `json:"severity"`
	Message     string         `json:"message"`
	Subject     string         `json:"subject,omitempty"`
	Path        string         `json:"path,omitempty"`
	Location    *Position      `json:"location,omitempty"`
	Recoverable bool           `json:"recoverable"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type AnalysisSummary struct {
	ModuleCount          int `json:"module_count"`
	RelationshipCount    int `json:"relationship_count"`
	ReferenceCount       int `json:"reference_count"`
	SourceReferenceCount int `json:"source_reference_count"`
	DiagnosticCount      int `json:"diagnostic_count"`
}
