// Package orchestration coordinates bounded multi-analyzer analysis without
// moving language semantics, package trust, or canonical graph derivation out
// of their existing owners.
package orchestration

import (
	"context"
	"sync"
	"time"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
)

const (
	JobPlanSchemaVersion        = "arch-view.job-plan/v1"
	JobEventSchemaVersion       = "arch-view.job-event/v1"
	AggregateSchemaVersion      = "arch-view.aggregate/v1"
	AggregateModelSchemaVersion = "arch-view.aggregate-model/v1"
	DiscoveryPolicyVersion      = "arch-view.discovery/v1"
	SourceScopePolicyVersion    = "arch-view.source-scope/v1"

	DefaultWorkerCount = 4
	HardMaxWorkerCount = 16
	MaxPlannedJobs     = 128
)

type JobStatus string

const (
	JobPlanned   JobStatus = "planned"
	JobQueued    JobStatus = "queued"
	JobRunning   JobStatus = "running"
	JobComplete  JobStatus = "complete"
	JobPartial   JobStatus = "partial"
	JobFailed    JobStatus = "failed"
	JobCancelled JobStatus = "cancelled"
	JobSkipped   JobStatus = "skipped"
)

type SelectionSource string

const (
	SelectionAssignment SelectionSource = "assignment"
	SelectionCLI        SelectionSource = "cli"
	SelectionAutomatic  SelectionSource = "automatic"
)

// DiscoveryPolicy controls filesystem traversal. The default policy is
// intentionally conservative and never follows symlink targets.
type DiscoveryPolicy struct {
	Version         string   `json:"version"`
	FixedExclusions []string `json:"fixed_exclusions"`
	FollowSymlinks  bool     `json:"follow_symlinks"`
	MaxPlannedJobs  int      `json:"max_planned_jobs"`
	RepositoryScope string   `json:"repository_scope,omitempty"`
}

// AnalyzerIncludeRule is an analyzer-ID-scoped allowlist. Duplicate rules
// are merged during policy normalization.
type AnalyzerIncludeRule struct {
	AnalyzerID string   `json:"analyzer_id"`
	Globs      []string `json:"globs"`
}

// SourceScopePolicy is resolved relative to InvocationRoot before planning.
// It is deliberately independent from persisted assignment configuration.
type SourceScopePolicy struct {
	PolicyVersion  string                `json:"policy_version"`
	InvocationRoot string                `json:"invocation_root"`
	Exclude        []string              `json:"exclude"`
	Include        []AnalyzerIncludeRule `json:"include"`
}

// AnalyzerAssignment is the already-resolved assignment input consumed by
// planning. Reading or writing .archview.json remains outside this package.
type AnalyzerAssignment struct {
	ProjectRoot string         `json:"project_root"`
	AnalyzerID  string         `json:"analyzer_id"`
	Language    string         `json:"language,omitempty"`
	Options     map[string]any `json:"options,omitempty"`
}

type Assignment = AnalyzerAssignment

// ExplicitSelection constrains planning without changing the assignment
// persistence boundary. ProjectRoot may be absolute or repository-relative.
type ExplicitSelection struct {
	ProjectRoot string         `json:"project_root,omitempty"`
	AnalyzerID  string         `json:"analyzer_id,omitempty"`
	Language    string         `json:"language,omitempty"`
	Options     map[string]any `json:"options,omitempty"`
}

// PlanRequest is the complete input to AnalyzerJobPlanner. The two per-
// analyzer option maps let a combined run carry only options supported by the
// selected manifest; the flat maps are convenient for callers with one common
// option layer.
type PlanRequest struct {
	RepositoryRoot     string
	InvocationRoot     string
	SourceScopePolicy  SourceScopePolicy
	DiscoveryPolicy    DiscoveryPolicy
	Assignments        []AnalyzerAssignment
	CLISelection       *ExplicitSelection
	ProjectOptions     map[string]any
	CLIOptions         map[string]any
	ProjectOptionsByID map[string]map[string]any
	CLIOptionsByID     map[string]map[string]any
	Runtime            analysis.RuntimeSelection
}

type ProjectRootCandidate struct {
	AbsolutePath         string   `json:"absolute_path"`
	RelativePath         string   `json:"relative_path"`
	StrongMarkers        []string `json:"strong_markers"`
	NestedRootExclusions []string `json:"nested_root_exclusions"`
	OwnedSourcePaths     []string `json:"owned_source_paths"`
}

type DiscoveryResult struct {
	RepositoryRoot      string                 `json:"repository_root"`
	RepositoryRootLabel string                 `json:"repository_root_label"`
	Policy              DiscoveryPolicy        `json:"policy"`
	Roots               []ProjectRootCandidate `json:"roots"`
	AllSourcePaths      []string               `json:"all_source_paths"`
	Diagnostics         []analysis.Diagnostic  `json:"diagnostics"`
}

type ProjectDiscoveryService struct {
	Registry *analysis.Registry
	Policy   DiscoveryPolicy
}

type CandidateEvaluation struct {
	Root               ProjectRootCandidate
	Analyzer           analysis.Analyzer
	Manifest           analysis.Manifest
	Language           string
	Candidate          analysis.DetectionCandidate
	Source             SelectionSource
	Options            analysis.EffectiveOptions
	InitialDiagnostics []analysis.Diagnostic
}

type AnalyzerJobPlanner struct {
	Registry  *analysis.Registry
	Discovery *ProjectDiscoveryService
}

type AnalyzerJob struct {
	JobID                       string                     `json:"job_id"`
	ScopeID                     string                     `json:"scope_id"`
	ProjectRoot                 string                     `json:"project_root"`
	RelativeProjectRoot         string                     `json:"relative_project_root"`
	LogicalAnalyzerID           string                     `json:"logical_analyzer_id"`
	AnalyzerVersion             string                     `json:"analyzer_version"`
	Language                    string                     `json:"language"`
	RuntimeMode                 string                     `json:"runtime_mode,omitempty"`
	RuntimeSource               string                     `json:"runtime_source"`
	RuntimePlatform             string                     `json:"runtime_platform,omitempty"`
	SelectionSource             SelectionSource            `json:"selection_source"`
	Selection                   analysis.AnalyzerSelection `json:"selection"`
	NestedRootExclusions        []string                   `json:"nested_root_exclusions"`
	EffectiveSourceScope        analysis.SourceScope       `json:"effective_source_scope"`
	SourceScopeFingerprint      string                     `json:"source_scope_fingerprint"`
	MatchedSourceSetFingerprint string                     `json:"matched_source_set_fingerprint"`
	EffectiveOptionsFingerprint string                     `json:"effective_options_fingerprint"`
	Options                     analysis.EffectiveOptions  `json:"options"`
	Status                      JobStatus                  `json:"status"`
	StartedAt                   *time.Time                 `json:"-"`
	FinishedAt                  *time.Time                 `json:"-"`
	Result                      *analysis.AnalysisResult   `json:"result,omitempty"`
	Diagnostics                 []analysis.Diagnostic      `json:"diagnostics"`

	// Manifest is planner/scheduler metadata and is kept out of the wire plan.
	// The scheduler uses it to perform a second canonical result check for
	// custom executors; Host.RunPlanned remains the authoritative runtime path.
	Manifest analysis.Manifest `json:"-"`
}

type JobPlan struct {
	PlanVersion            string                `json:"plan_version"`
	RepositoryRoot         string                `json:"repository_root"`
	InvocationRoot         string                `json:"invocation_root"`
	DiscoveryPolicyVersion string                `json:"discovery_policy_version"`
	SourceScopePolicy      SourceScopePolicy     `json:"source_scope_policy"`
	Jobs                   []AnalyzerJob         `json:"jobs"`
	DiscoveryDiagnostics   []analysis.Diagnostic `json:"discovery_diagnostics"`
}

type JobLifecycleEvent struct {
	SchemaVersion string               `json:"schema_version"`
	Sequence      int                  `json:"sequence"`
	RunID         string               `json:"run_id"`
	Type          string               `json:"type"`
	JobID         string               `json:"job_id,omitempty"`
	ScopeID       string               `json:"scope_id,omitempty"`
	Status        JobStatus            `json:"status,omitempty"`
	CompletedJobs int                  `json:"completed_jobs"`
	TotalJobs     int                  `json:"total_jobs"`
	Diagnostic    *analysis.Diagnostic `json:"diagnostic,omitempty"`
}

type SchedulerOptions struct {
	WorkerCount int
}

type AnalyzerExecutor func(context.Context, AnalyzerJob) (analysis.AnalysisResult, error)

type ExecutionSnapshot struct {
	RunID  string
	Plan   JobPlan
	Jobs   []AnalyzerJob
	Events []JobLifecycleEvent
}

type AnalyzerJobScheduler struct {
	Executor    AnalyzerExecutor
	WorkerCount int
	cancelMu    sync.Mutex
	cancel      context.CancelFunc
}

type ScopeSourceSummary struct {
	PolicyFingerprint           string `json:"policy_fingerprint"`
	MatchedSourceSetFingerprint string `json:"matched_source_set_fingerprint"`
}

type ScopeSummary struct {
	ScopeID       string                   `json:"scope_id"`
	ProjectRoot   string                   `json:"project_root"`
	Analyzer      analysis.AnalyzerInfo    `json:"analyzer"`
	RuntimeSource string                   `json:"runtime_source,omitempty"`
	SourceScope   ScopeSourceSummary       `json:"source_scope"`
	Status        JobStatus                `json:"status"`
	Summary       analysis.AnalysisSummary `json:"summary"`
	Diagnostics   []analysis.Diagnostic    `json:"diagnostics"`
}

type ScopedDiagnostic struct {
	ID          string             `json:"id"`
	ScopeID     string             `json:"scope_id,omitempty"`
	Code        string             `json:"code"`
	Severity    string             `json:"severity"`
	Message     string             `json:"message"`
	Subject     string             `json:"subject,omitempty"`
	Path        string             `json:"path,omitempty"`
	Location    *analysis.Position `json:"location,omitempty"`
	Recoverable bool               `json:"recoverable"`
	Metadata    map[string]any     `json:"metadata,omitempty"`
}

type AggregateSummary struct {
	JobCount         int `json:"job_count"`
	UsableScopeCount int `json:"usable_scope_count"`
	FailedScopeCount int `json:"failed_scope_count"`
}

type RepositoryMetadata struct {
	RootLabel string `json:"root_label"`
}

type AggregateModel struct {
	SchemaVersion    string                             `json:"schema_version"`
	ModelID          string                             `json:"model_id"`
	Status           model.Status                       `json:"status"`
	Project          model.Project                      `json:"project"`
	Analyzer         analysis.AnalyzerInfo              `json:"analyzer"`
	Scopes           []ScopeSummary                     `json:"scopes"`
	Modules          []analysis.ModuleObservation       `json:"modules"`
	References       []analysis.Reference               `json:"references"`
	SourceReferences []analysis.SourceReference         `json:"source_references"`
	Relationships    []analysis.RelationshipObservation `json:"relationships"`
	Diagnostics      []model.Diagnostic                 `json:"diagnostics"`
	Derived          model.Derived                      `json:"derived"`
}

type AnalysisRun struct {
	SchemaVersion string                  `json:"schema_version"`
	RunID         string                  `json:"run_id"`
	Status        analysis.AnalysisStatus `json:"status"`
	Repository    RepositoryMetadata      `json:"repository"`
	JobPlan       JobPlan                 `json:"job_plan"`
	Scopes        []ScopeSummary          `json:"scopes"`
	Model         *AggregateModel         `json:"model,omitempty"`
	Diagnostics   []ScopedDiagnostic      `json:"diagnostics"`
	Summary       AggregateSummary        `json:"summary"`
	Events        []JobLifecycleEvent     `json:"-"`

	// These caches back scope selection and the local viewer. They are not
	// serialized as part of the aggregate response.
	combinedCanonical model.Model
	scopeModels       map[string]model.Model
	scopeResults      map[string]analysis.AnalysisResult
}

type SelectedScope struct {
	RunID      string
	Scope      string
	Summary    ScopeSummary
	Model      model.Model
	Projection model.HierarchyProjection
}

type Projector interface {
	SelectAnalysisScope(scope string) (SelectedScope, error)
}
