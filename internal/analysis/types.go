package analysis

import (
	"context"

	"github.com/buffo/arch-view/internal/quality"
)

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
	ProjectRoot        string              `json:"project_root"`
	Selection          AnalyzerSelection   `json:"selection"`
	Options            EffectiveOptions    `json:"options"`
	SourceScope        *SourceScope        `json:"source_scope,omitempty"`
	SourceIndexRequest *SourceIndexRequest `json:"source_index_request,omitempty"`
}

// SourceIndexRequest is the optional analyzer-facing source-index policy.
// A nil request preserves the legacy analyzer contract: an analyzer may use
// its historical default behavior. A non-nil disabled request must avoid
// building a source index, while an enabled request carries the exact
// capability set requested by the live coordinator.
type SourceIndexRequest struct {
	Enabled      bool     `json:"enabled"`
	Capabilities []string `json:"capabilities,omitempty"`
}

func cloneSourceIndexRequest(value *SourceIndexRequest) *SourceIndexRequest {
	if value == nil {
		return nil
	}
	clone := *value
	clone.Capabilities = append([]string(nil), value.Capabilities...)
	return &clone
}

type RunRequest struct {
	ProjectRoot        string
	Language           string
	AnalyzerID         string
	ProjectOptions     map[string]any
	CLIOptions         map[string]any
	SourceScope        *SourceScope
	SourceIndexRequest *SourceIndexRequest
}

// PlannedRunRequest is the host boundary used by the multi-analyzer
// scheduler. Selection and options are resolved by the planner once and are
// passed through unchanged; the host still owns analyzer lookup, runtime
// provenance, result validation, and process cleanup.
type PlannedRunRequest struct {
	ProjectRoot        string
	AnalyzerID         string
	Selection          AnalyzerSelection
	Options            EffectiveOptions
	SourceScope        *SourceScope
	SourceIndexRequest *SourceIndexRequest
}

// RuntimeSelection describes how the host obtained the analyzer used for a
// run. The zero value preserves the original direct/in-process Host behavior;
// command entrypoints set it explicitly so runtime provenance is observable.
type RuntimeSelection struct {
	Mode     string `json:"runtime_mode,omitempty"`
	Source   string `json:"runtime_source,omitempty"`
	Platform string `json:"runtime_platform,omitempty"`
}

const (
	RuntimeModePackaged  = "packaged"
	RuntimeModeInProcess = "in-process"
	RuntimeModeExplicit  = "explicit"
)

type AnalyzerSelection struct {
	AnalyzerID      string   `json:"analyzer_id"`
	Mode            string   `json:"mode"`
	Confidence      float64  `json:"confidence"`
	MatchedMarkers  []string `json:"matched_markers,omitempty"`
	Reason          string   `json:"reason,omitempty"`
	BoundaryHint    string   `json:"boundary_hint,omitempty"`
	RuntimeMode     string   `json:"runtime_mode,omitempty"`
	RuntimeSource   string   `json:"runtime_source,omitempty"`
	RuntimePlatform string   `json:"runtime_platform,omitempty"`
}

type Manifest struct {
	ID               string             `json:"id"`
	Version          string             `json:"version"`
	Language         string             `json:"language"`
	APIVersion       string             `json:"api_version"`
	DetectionMarkers []DetectionMarker  `json:"detection_markers"`
	Capabilities     []string           `json:"capabilities"`
	Options          []OptionDescriptor `json:"options"`
	// RuntimeIdentity is host-owned package identity used for session-cache
	// invalidation. It is not part of the analyzer protocol manifest.
	RuntimeIdentity string `json:"-"`
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

// SourceScope is the normalized source input handed to one analyzer job. All
// paths are repository-relative to the invocation root unless explicitly
// documented otherwise by the containing field. The planner computes the
// fingerprints before execution so source filtering participates in cache and
// job identity.
type SourceScope struct {
	PolicyVersion               string   `json:"policy_version"`
	InvocationRoot              string   `json:"invocation_root"`
	ProjectRoot                 string   `json:"project_root"`
	NestedRootExclusions        []string `json:"nested_root_exclusions,omitempty"`
	IncludeGlobs                []string `json:"include_globs,omitempty"`
	ExcludeGlobs                []string `json:"exclude_globs,omitempty"`
	MatchedPaths                []string `json:"matched_paths"`
	MatchedLocalPaths           []string `json:"matched_local_paths,omitempty"`
	ExcludedPaths               []string `json:"excluded_paths,omitempty"`
	PolicyFingerprint           string   `json:"policy_fingerprint"`
	MatchedSourceSetFingerprint string   `json:"matched_source_set_fingerprint"`
}

type AnalysisStatus string

const (
	StatusComplete  AnalysisStatus = "complete"
	StatusPartial   AnalysisStatus = "partial"
	StatusFailed    AnalysisStatus = "failed"
	StatusCancelled AnalysisStatus = "cancelled"
)

type AnalysisResult struct {
	RunID              string                     `json:"run_id"`
	Status             AnalysisStatus             `json:"status"`
	Analyzer           AnalyzerInfo               `json:"analyzer"`
	Project            ProjectInfo                `json:"project"`
	OptionsFingerprint string                     `json:"options_fingerprint,omitempty"`
	Modules            []ModuleObservation        `json:"modules"`
	Relationships      []RelationshipObservation  `json:"relationships"`
	References         []Reference                `json:"references"`
	SourceReferences   []SourceReference          `json:"source_references"`
	Diagnostics        []Diagnostic               `json:"diagnostics"`
	Summary            AnalysisSummary            `json:"summary"`
	SourceIndex        *SourceIndex               `json:"source_index,omitempty"`
	QualityReport      *quality.QualityEvaluation `json:"quality_report,omitempty"`
}

type AnalyzerInfo struct {
	ID              string `json:"id"`
	Version         string `json:"version"`
	Language        string `json:"language"`
	APIVersion      string `json:"api_version"`
	RuntimeMode     string `json:"runtime_mode,omitempty"`
	RuntimeSource   string `json:"runtime_source,omitempty"`
	RuntimePlatform string `json:"runtime_platform,omitempty"`
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
