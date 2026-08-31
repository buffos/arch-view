// Package live contains the transport-neutral live analysis boundary.
//
// The package coordinates sessions, freshness, immutable revisions, bounded
// queries, and quality delegation. Analyzer semantics, source-fact extraction,
// architecture-model derivation, and quality-rule semantics remain owned by
// their existing packages.
package live

import (
	"context"
	"time"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/orchestration"
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/quality"
)

const (
	LiveSchemaVersion  = "arch-view.live/v1"
	QuerySchemaVersion = "arch-view.query/v1"

	DefaultDebounce          = 250 * time.Millisecond
	DefaultMaxPendingEvents  = 10_000
	DefaultMaxParallelScopes = 4
	DefaultFreshnessMaxWait  = 5 * time.Second
	DefaultStabilityRetries  = 2
	DefaultQueryMaxBytes     = 16 * 1024
	DefaultQueryMaxItems     = 50
	DefaultQueryHardMaxBytes = 1 << 20
	DefaultQueryHardMaxItems = 5_000
	DefaultContextLines      = 40
	DefaultContextHardLines  = 1_000
)

type Consistency string

const (
	ConsistencyLatestReady    Consistency = "latest_ready"
	ConsistencyRequireCurrent Consistency = "require_current"
	ConsistencySpecific       Consistency = "specific_revision"
)

type SessionState string

const (
	SessionInitializing SessionState = "initializing"
	SessionReady        SessionState = "ready"
	SessionDegraded     SessionState = "degraded"
	SessionFailed       SessionState = "failed"
)

type FreshnessStatus string

const (
	FreshnessCurrent       FreshnessStatus = "current"
	FreshnessStale         FreshnessStatus = "stale"
	FreshnessUpdating      FreshnessStatus = "updating"
	FreshnessFailed        FreshnessStatus = "failed"
	FreshnessInitializing  FreshnessStatus = "initializing"
	FreshnessInputUnstable FreshnessStatus = "input_unstable"
)

type ReconciliationStatus string

const (
	ReconciliationNotRequested ReconciliationStatus = "not_requested"
	ReconciliationPassed       ReconciliationStatus = "passed"
	ReconciliationChanged      ReconciliationStatus = "changed"
	ReconciliationFailed       ReconciliationStatus = "failed"
	ReconciliationUnstable     ReconciliationStatus = "unstable"
)

type LiveSessionConfig struct {
	SchemaVersion      string             `json:"schema_version"`
	SessionID          string             `json:"session_id"`
	RepositoryRoot     string             `json:"repository_root"`
	WatchRoots         []WatchRoot        `json:"watch_roots"`
	AnalyzerIDs        []string           `json:"analyzer_ids,omitempty"`
	SourceIndexRequest SourceIndexRequest `json:"source_index_request"`
	QualityRequest     *QualityRequest    `json:"quality_request,omitempty"`
	WatchPolicy        WatchPolicy        `json:"watch_policy"`
	FreshnessPolicy    FreshnessPolicy    `json:"freshness_policy"`
	QueryPolicy        QueryPolicy        `json:"query_policy"`
	PermissionPolicy   PermissionPolicy   `json:"permission_policy"`
	Extensions         []ExtensionBlock   `json:"extensions"`
}

type WatchRoot struct {
	Path      string `json:"path"`
	Recursive bool   `json:"recursive"`
}

type SourceIndexRequest struct {
	Enabled      bool     `json:"enabled"`
	Capabilities []string `json:"capabilities"`
}

type QualityRequest struct {
	ProfileID      string `json:"profile_id"`
	ProfileVersion string `json:"profile_version"`
}

type WatchPolicy struct {
	DebounceMS        int `json:"debounce_ms"`
	MaxPendingEvents  int `json:"max_pending_events"`
	MaxParallelScopes int `json:"max_parallel_scopes"`
	RescanIntervalMS  int `json:"rescan_interval_ms,omitempty"`
}

type FreshnessPolicy struct {
	DefaultConsistency  Consistency `json:"default_consistency"`
	SettleMS            int         `json:"settle_ms"`
	MaxWaitMS           int         `json:"max_wait_ms"`
	MaxStabilityRetries int         `json:"max_stability_retries"`
	ReconcileIntervalMS int         `json:"reconcile_interval_ms,omitempty"`
}

type QueryPolicy struct {
	DefaultMaxBytes     int `json:"default_max_bytes"`
	DefaultMaxItems     int `json:"default_max_items"`
	HardMaxBytes        int `json:"hard_max_bytes"`
	HardMaxItems        int `json:"hard_max_items"`
	DefaultContextLines int `json:"default_context_lines"`
	HardContextLines    int `json:"hard_context_lines"`
}

type Operation string

const (
	OperationStatus              Operation = "status"
	OperationSearch              Operation = "search"
	OperationEvidence            Operation = "evidence"
	OperationSourceContext       Operation = "source_context"
	OperationEnsureCurrent       Operation = "ensure_current"
	OperationQualityEvaluate     Operation = "quality_evaluate"
	OperationQualityProfileRead  Operation = "quality_profile_read"
	OperationQualityProfileWrite Operation = "quality_profile_write"
	OperationBaselineRead        Operation = "baseline_read"
	OperationBaselineWrite       Operation = "baseline_write"
)

type BaselineMode string

const (
	BaselineModeProfile  BaselineMode = "profile"
	BaselineModeNone     BaselineMode = "none"
	BaselineModeSelected BaselineMode = "selected"
)

type PermissionPolicy struct {
	DefaultMode          string      `json:"default_mode"`
	AllowedOperations    []Operation `json:"allowed_operations"`
	Transport            string      `json:"transport"`
	AllowTargetExecution bool        `json:"allow_target_execution"`
	AllowShell           bool        `json:"allow_shell"`
	AuditPolicyWrites    bool        `json:"audit_policy_writes"`
}

type ExtensionBlock struct {
	Namespace     string `json:"namespace"`
	SchemaVersion string `json:"schema_version"`
	Capability    string `json:"capability"`
	Payload       any    `json:"payload"`
}

type LiveDiagnostic struct {
	Code        string         `json:"code"`
	Message     string         `json:"message"`
	Severity    string         `json:"severity"`
	Path        string         `json:"path,omitempty"`
	Recoverable bool           `json:"recoverable"`
	Details     map[string]any `json:"details,omitempty"`
}

type Freshness struct {
	Status               FreshnessStatus      `json:"status"`
	RequestedConsistency Consistency          `json:"requested_consistency,omitempty"`
	LastReadyRevision    int                  `json:"last_ready_revision,omitempty"`
	PendingEventGroupID  string               `json:"pending_event_group_id,omitempty"`
	ChangedPaths         []string             `json:"changed_paths,omitempty"`
	Reconciliation       ReconciliationStatus `json:"reconciliation"`
}

type InputVerification struct {
	Status              string         `json:"status"`
	ManifestFingerprint ContentDigest  `json:"manifest_fingerprint"`
	ContentFingerprint  *ContentDigest `json:"content_fingerprint,omitempty"`
	Attempts            int            `json:"attempts"`
}

type ContentDigest struct {
	Algorithm string `json:"algorithm"`
	Value     string `json:"value"`
}

type OpaqueRef struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type LiveSnapshot struct {
	SchemaVersion          string            `json:"schema_version"`
	SnapshotID             string            `json:"snapshot_id"`
	Revision               int               `json:"revision"`
	State                  SessionState      `json:"state"`
	SourceInputFingerprint ContentDigest     `json:"source_input_fingerprint"`
	InputVerification      InputVerification `json:"input_verification"`
	SourceIndexRef         *OpaqueRef        `json:"source_index_ref,omitempty"`
	ModelRef               *OpaqueRef        `json:"model_ref,omitempty"`
	QualityReportRef       *OpaqueRef        `json:"quality_report_ref,omitempty"`
	QualityPolicyRef       *OpaqueRef        `json:"quality_policy_ref,omitempty"`
	ScopeIDs               []string          `json:"scope_ids"`
	Diagnostics            []LiveDiagnostic  `json:"diagnostics"`
	Freshness              Freshness         `json:"freshness"`
	SemanticDigest         ContentDigest     `json:"semantic_digest"`
	Extensions             []ExtensionBlock  `json:"extensions"`
}

type WatchEvent struct {
	BackendID string `json:"backend_id"`
	Kind      string `json:"kind"`
	Path      string `json:"path"`
	OldPath   string `json:"old_path,omitempty"`
	Sequence  string `json:"sequence,omitempty"`
}

type NormalizedEvent struct {
	BackendID    string `json:"backend_id"`
	Kind         string `json:"kind"`
	Path         string `json:"path,omitempty"`
	OldPath      string `json:"old_path,omitempty"`
	Sequence     string `json:"sequence,omitempty"`
	EventGroupID string `json:"event_group_id"`
}

type EventGroup struct {
	ID            string            `json:"event_group_id"`
	Events        []NormalizedEvent `json:"events"`
	ChangedPaths  []string          `json:"changed_paths"`
	FullRescan    bool              `json:"full_rescan"`
	Reason        string            `json:"reason,omitempty"`
	FirstSequence string            `json:"first_sequence,omitempty"`
	LastSequence  string            `json:"last_sequence,omitempty"`
}

type ReconciliationResult struct {
	Status              ReconciliationStatus `json:"status"`
	ManifestFingerprint ContentDigest        `json:"manifest_fingerprint"`
	ContentFingerprint  *ContentDigest       `json:"content_fingerprint,omitempty"`
	ChangedPaths        []string             `json:"changed_paths"`
	Attempts            int                  `json:"attempts"`
	Reason              string               `json:"reason,omitempty"`
}

type InvalidationPlan struct {
	EventGroupID            string         `json:"event_group_id"`
	AffectedPaths           []string       `json:"affected_paths"`
	AffectedScopeIDs        []string       `json:"affected_scope_ids"`
	Mode                    string         `json:"mode"`
	SourceFingerprintBefore *ContentDigest `json:"source_fingerprint_before,omitempty"`
	Reason                  string         `json:"reason,omitempty"`
}

type ScopeIdentity struct {
	ScopeID     string `json:"scope_id"`
	ProjectRoot string `json:"project_root"`
	AnalyzerID  string `json:"analyzer_id"`
}

type StructuralQuery struct {
	ScopeIDs          []string `json:"scope_ids,omitempty"`
	PathPrefix        string   `json:"path_prefix,omitempty"`
	PathGlob          string   `json:"path_glob,omitempty"`
	Language          string   `json:"language,omitempty"`
	Roles             []string `json:"roles,omitempty"`
	Name              string   `json:"name,omitempty"`
	QualifiedName     string   `json:"qualified_name,omitempty"`
	SymbolCategories  []string `json:"symbol_categories,omitempty"`
	DocumentationText string   `json:"documentation_text,omitempty"`
	RuleIDs           []string `json:"rule_ids,omitempty"`
	AssessmentKinds   []string `json:"assessment_kinds,omitempty"`
	Severities        []string `json:"severities,omitempty"`
	Statuses          []string `json:"statuses,omitempty"`
	ModuleIDs         []string `json:"module_ids,omitempty"`
	FileIDs           []string `json:"file_ids,omitempty"`
	SubjectID         string   `json:"subject_id,omitempty"`
	CaseSensitive     bool     `json:"case_sensitive"`
}

type QueryRequest struct {
	SessionID   string          `json:"session_id"`
	Consistency Consistency     `json:"consistency"`
	Revision    int             `json:"revision,omitempty"`
	Query       StructuralQuery `json:"query"`
	Projection  []string        `json:"projection,omitempty"`
	MaxBytes    int             `json:"max_bytes,omitempty"`
	MaxItems    int             `json:"max_items,omitempty"`
	Cursor      string          `json:"cursor,omitempty"`
}

type TextSearchQuery struct {
	Pattern       string      `json:"pattern"`
	Mode          string      `json:"mode"`
	Consistency   Consistency `json:"consistency,omitempty"`
	Revision      int         `json:"revision,omitempty"`
	PathGlob      string      `json:"path_glob,omitempty"`
	Language      string      `json:"language,omitempty"`
	ScopeIDs      []string    `json:"scope_ids,omitempty"`
	CaseSensitive bool        `json:"case_sensitive"`
	MaxLineBytes  int         `json:"max_line_bytes,omitempty"`
	MaxBytes      int         `json:"max_bytes,omitempty"`
	MaxItems      int         `json:"max_items,omitempty"`
	Cursor        string      `json:"cursor,omitempty"`
}

type TextMatch struct {
	ScopeID     string `json:"scope_id"`
	Path        string `json:"path"`
	Line        int    `json:"line"`
	Column      int    `json:"column"`
	EndColumn   int    `json:"end_column"`
	Text        string `json:"text,omitempty"`
	MatchLength int    `json:"match_length"`
}

type SourceContextRequest struct {
	SessionID   string               `json:"session_id"`
	Consistency Consistency          `json:"consistency"`
	Revision    int                  `json:"revision,omitempty"`
	ScopeID     string               `json:"scope_id,omitempty"`
	EntityID    string               `json:"entity_id,omitempty"`
	Span        *analysis.SourceSpan `json:"span,omitempty"`
	MaxLines    int                  `json:"max_lines"`
	MaxBytes    int                  `json:"max_bytes"`
}

type SourceContext struct {
	ScopeID     string        `json:"scope_id"`
	SnapshotID  string        `json:"snapshot_id"`
	Path        string        `json:"path"`
	StartLine   int           `json:"start_line"`
	EndLine     int           `json:"end_line"`
	Content     string        `json:"content"`
	ContentHash ContentDigest `json:"content_hash"`
	Truncated   bool          `json:"truncated"`
}

type CapabilityCoverage struct {
	Capability  string   `json:"capability"`
	Status      string   `json:"status"`
	ProviderIDs []string `json:"provider_ids,omitempty"`
	Reason      string   `json:"reason,omitempty"`
}

type BudgetUsage struct {
	MaxBytes     int  `json:"max_bytes"`
	MaxItems     int  `json:"max_items"`
	EmittedBytes int  `json:"emitted_bytes"`
	EmittedItems int  `json:"emitted_items"`
	Truncated    bool `json:"truncated"`
}

type QueryDiagnostic struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type QueryEnvelope struct {
	SchemaVersion        string               `json:"schema_version"`
	SessionID            string               `json:"session_id"`
	SnapshotID           string               `json:"snapshot_id,omitempty"`
	Revision             int                  `json:"revision,omitempty"`
	RequestedConsistency Consistency          `json:"requested_consistency"`
	ReturnedConsistency  string               `json:"returned_consistency"`
	Freshness            Freshness            `json:"freshness"`
	ScopeIDs             []string             `json:"scope_ids"`
	Result               any                  `json:"result"`
	ResultCount          int                  `json:"result_count"`
	OmittedFields        []string             `json:"omitted_fields"`
	Capabilities         []CapabilityCoverage `json:"capabilities"`
	Budget               BudgetUsage          `json:"budget"`
	NextCursor           string               `json:"next_cursor,omitempty"`
	Diagnostics          []QueryDiagnostic    `json:"diagnostics"`
}

type SnapshotStatusResult struct {
	State       SessionState     `json:"state"`
	Snapshot    *LiveSnapshot    `json:"snapshot,omitempty"`
	Diagnostics []LiveDiagnostic `json:"diagnostics"`
}

type QueryError struct {
	Code    string
	Message string
	Details map[string]any
}

func (err *QueryError) Error() string {
	if err == nil {
		return ""
	}
	return err.Code + ": " + err.Message
}

type ScanRequest struct {
	SessionID        string
	RepositoryRoot   string
	Config           LiveSessionConfig
	Invalidation     InvalidationPlan
	InputFingerprint InputFingerprint
	PolicyService    QualityPolicyService
}

type ScanResult struct {
	Run           orchestration.AnalysisRun
	Model         model.Model
	QualityReport *quality.QualityEvaluation
	Diagnostics   []LiveDiagnostic
}

type Scanner interface {
	Scan(context.Context, ScanRequest) (ScanResult, error)
}

type QualityProfileResolver interface {
	ResolveProfile(context.Context, string, string) (quality.QualityProfile, error)
	ListProfiles(context.Context) ([]QualityProfileInfo, error)
}

// QualityPolicyService is the transport-neutral write seam for quality
// profiles and baselines. The live package performs permission, revision, and
// audit checks; the service performs typed document validation and safe file
// persistence.
type QualityPolicyService interface {
	ValidateProfile(quality.QualityProfile) (quality.QualityProfile, error)
	ResolveProfile(context.Context, string, string) (quality.QualityProfile, error)
	ProfileDocument(context.Context, string, string) (quality.QualityProfile, QualityPolicyProfileInfo, error)
	SaveProfile(context.Context, quality.QualityProfile, string, bool) (QualityPolicyWriteResult, error)
	SaveBaseline(context.Context, quality.Baseline, string, bool) (QualityPolicyWriteResult, error)
	ListBaselines(context.Context) ([]QualityBaselineInfo, error)
	ReadBaseline(context.Context, string) (quality.Baseline, QualityBaselineInfo, error)
	ResolveBaseline(context.Context, string, string) (quality.Baseline, QualityBaselineInfo, error)
	AppendBaseline(context.Context, QualityBaselineAppendRequest) (QualityBaselineAppendResult, error)
}

// QualityPolicyProfileInfo and QualityPolicyWriteResult are deliberately
// declared in live so an adapter need not expose the policy package's file
// implementation to callers or transports.
type QualityPolicyProfileInfo struct {
	ProfileID      string `json:"profile_id"`
	ProfileVersion string `json:"profile_version"`
	FileName       string `json:"file_name"`
	Status         string `json:"status"`
	Reason         string `json:"reason,omitempty"`
}

type QualityPolicyWriteResult struct {
	FileName     string                `json:"file_name"`
	RelativePath string                `json:"relative_path"`
	Digest       quality.ContentDigest `json:"digest"`
	Overwritten  bool                  `json:"overwritten"`
}

type QualityBaselineInfo struct {
	BaselineID string   `json:"baseline_id,omitempty"`
	Revision   string   `json:"revision,omitempty"`
	FileName   string   `json:"file_name"`
	Status     string   `json:"status"`
	EntryCount int      `json:"entry_count"`
	Profiles   []string `json:"profiles"`
	Reason     string   `json:"reason,omitempty"`
}

type QualityBaselineAppendRequest struct {
	Profile          quality.QualityProfile
	ProfileFileName  string
	BaselineFileName string
	BaselineID       string
	Entries          []quality.BaselineEntry
	Revision         string
	ExpectedRevision string
}

type QualityBaselineAppendResult struct {
	Profile       quality.QualityProfile
	Baseline      quality.Baseline
	Added         []quality.BaselineEntry
	Existing      []quality.BaselineEntry
	BaselineWrite QualityPolicyWriteResult
	ProfileWrite  QualityPolicyWriteResult
	Changed       bool
}

// PolicyAuthorization is passed to the host authorizer only for policy
// writes. An authorization string is opaque to live and is never interpreted
// as a filesystem path or command.
type PolicyAuthorization struct {
	SessionID     string    `json:"session_id"`
	Operation     Operation `json:"operation"`
	Authorization string    `json:"authorization"`
}

type PolicyAuthorizer interface {
	Authorize(context.Context, PolicyAuthorization) error
}

type PolicyAuthorizerFunc func(context.Context, PolicyAuthorization) error

func (authorizer PolicyAuthorizerFunc) Authorize(ctx context.Context, value PolicyAuthorization) error {
	if authorizer == nil {
		return newLiveError(ErrorQualityPolicyPermissionDenied, "quality policy authorization is unavailable", nil)
	}
	return authorizer(ctx, value)
}

type QualityPolicyCommand struct {
	Operation                string                  `json:"operation"`
	SessionID                string                  `json:"session_id"`
	ReportID                 string                  `json:"report_id,omitempty"`
	ReportRevision           int                     `json:"report_revision,omitempty"`
	ProfileID                string                  `json:"profile_id,omitempty"`
	ProfileVersion           string                  `json:"profile_version,omitempty"`
	SourceProfileID          string                  `json:"source_profile_id,omitempty"`
	SourceProfileVersion     string                  `json:"source_profile_version,omitempty"`
	Profile                  *quality.QualityProfile `json:"profile,omitempty"`
	RuleBindings             []quality.RuleBinding   `json:"rule_bindings,omitempty"`
	FileName                 string                  `json:"file_name,omitempty"`
	BaselineID               string                  `json:"baseline_id,omitempty"`
	BaselineRevision         string                  `json:"baseline_revision,omitempty"`
	ExpectedBaselineRevision string                  `json:"expected_baseline_revision,omitempty"`
	FindingKeys              []string                `json:"finding_keys,omitempty"`
	Reason                   string                  `json:"reason,omitempty"`
	Owner                    string                  `json:"owner,omitempty"`
	Authorization            string                  `json:"authorization,omitempty"`
	Overwrite                bool                    `json:"overwrite"`
}

type QualityPolicyIdentity struct {
	ProfileID       string                 `json:"profile_id"`
	ProfileVersion  string                 `json:"profile_version"`
	ProfileDigest   *quality.ContentDigest `json:"profile_digest,omitempty"`
	ReportID        string                 `json:"report_id,omitempty"`
	ReportRevision  int                    `json:"report_revision,omitempty"`
	ReportDigest    quality.ContentDigest  `json:"report_digest,omitempty"`
	RuleVersions    []string               `json:"rule_versions"`
	FormulaVersions []string               `json:"formula_versions"`
}

type QualityPolicyAudit struct {
	AuditID      string `json:"audit_id"`
	Operation    string `json:"operation"`
	SessionID    string `json:"session_id"`
	Authorized   bool   `json:"authorized"`
	At           string `json:"at"`
	RelativePath string `json:"relative_path,omitempty"`
	Details      string `json:"details,omitempty"`
}

type QualityBaselinePreview struct {
	BaselineID     string                  `json:"baseline_id"`
	Revision       string                  `json:"revision,omitempty"`
	FindingKeys    []string                `json:"finding_keys"`
	Entries        []quality.BaselineEntry `json:"entries"`
	PolicyIdentity QualityPolicyIdentity   `json:"policy_identity"`
}

type QualityPolicyResult struct {
	Status               string                    `json:"status"`
	Operation            string                    `json:"operation"`
	Message              string                    `json:"message,omitempty"`
	Profile              *quality.QualityProfile   `json:"profile,omitempty"`
	Baseline             *quality.Baseline         `json:"baseline,omitempty"`
	Preview              *QualityBaselinePreview   `json:"preview,omitempty"`
	FindingKeys          []string                  `json:"finding_keys,omitempty"`
	PolicyIdentity       QualityPolicyIdentity     `json:"policy_identity"`
	Write                *QualityPolicyWriteResult `json:"write,omitempty"`
	ProfileWrite         *QualityPolicyWriteResult `json:"profile_write,omitempty"`
	Audit                *QualityPolicyAudit       `json:"audit,omitempty"`
	AddedFindingKeys     []string                  `json:"added_finding_keys,omitempty"`
	ExistingFindingKeys  []string                  `json:"existing_finding_keys,omitempty"`
	ReevaluationRequired bool                      `json:"reevaluation_required,omitempty"`
}

type QualityProfileInfo struct {
	ProfileID      string `json:"profile_id"`
	ProfileVersion string `json:"profile_version"`
	FileName       string `json:"file_name,omitempty"`
	Status         string `json:"status"`
	Reason         string `json:"reason,omitempty"`
}

// QualityCatalogRequest selects the immutable revision used to anchor a
// catalog read. ProfileID is optional: without it, rule entries contain the
// catalog defaults and are marked as catalog-only.
type QualityCatalogRequest struct {
	QueryRequest
	ProfileID      string `json:"profile_id,omitempty"`
	ProfileVersion string `json:"profile_version,omitempty"`
}

type QualityBaselinesRequest struct {
	QueryRequest
	FileNames        []string `json:"file_names,omitempty"`
	BaselineID       string   `json:"baseline_id,omitempty"`
	BaselineRevision string   `json:"baseline_revision,omitempty"`
	IncludeEntries   bool     `json:"include_entries"`
}

type QualityBaselineDocument struct {
	Info         QualityBaselineInfo     `json:"info"`
	Entries      []quality.BaselineEntry `json:"entries,omitempty"`
	TotalEntries int                     `json:"total_entries"`
	NextCursor   string                  `json:"next_cursor,omitempty"`
}

type QualityBaselinesResult struct {
	Status     string                    `json:"status"`
	Items      []QualityBaselineDocument `json:"items"`
	Total      int                       `json:"total"`
	NextCursor string                    `json:"next_cursor,omitempty"`
}

type QualityRuleCatalogEntry struct {
	RuleID               string                   `json:"rule_id"`
	RuleVersion          string                   `json:"rule_version"`
	AssessmentKind       string                   `json:"assessment_kind"`
	RequiredCapabilities []string                 `json:"required_capabilities"`
	ParameterSchema      quality.ParameterSchema  `json:"parameter_schema"`
	DefaultSeverity      string                   `json:"default_severity"`
	Description          string                   `json:"description,omitempty"`
	Limitations          []string                 `json:"limitations,omitempty"`
	Status               string                   `json:"status"`
	Reason               string                   `json:"reason,omitempty"`
	Enabled              bool                     `json:"enabled"`
	Parameters           quality.TypedConfigBlock `json:"parameters"`
	Severity             string                   `json:"severity,omitempty"`
}

type QualityCatalogResult struct {
	Profiles     []QualityProfileInfo           `json:"profiles"`
	Rules        []QualityRuleCatalogEntry      `json:"rules"`
	Capabilities []quality.CapabilityDescriptor `json:"capabilities"`
}

type QualityEvaluationRequest struct {
	SessionID      string                `json:"session_id"`
	Consistency    Consistency           `json:"consistency"`
	Revision       int                   `json:"revision,omitempty"`
	ScopeIDs       []string              `json:"scope_ids,omitempty"`
	ProfileID      string                `json:"profile_id"`
	ProfileVersion string                `json:"profile_version"`
	RuleBindings   []quality.RuleBinding `json:"rule_bindings,omitempty"`
	BaselineMode   BaselineMode          `json:"baseline_mode,omitempty"`
	BaselineFiles  []string              `json:"baseline_files,omitempty"`
	Persist        bool                  `json:"persist"`
	MaxBytes       int                   `json:"max_bytes,omitempty"`
	MaxItems       int                   `json:"max_items,omitempty"`
}

type QualityEvaluationResult struct {
	Status    string                     `json:"status"`
	Temporary bool                       `json:"temporary"`
	Report    *quality.QualityEvaluation `json:"report,omitempty"`
	Reason    string                     `json:"reason,omitempty"`
}

type QualityFindingsRequest struct {
	QueryRequest
	ReportID string `json:"report_id,omitempty"`
}

type QualityFindingsResult struct {
	Status       string                    `json:"status"`
	ReportID     string                    `json:"report_id,omitempty"`
	EvaluationID string                    `json:"evaluation_id,omitempty"`
	SnapshotIDs  []string                  `json:"source_snapshot_ids,omitempty"`
	Items        []quality.QualityFinding  `json:"items"`
	Total        int                       `json:"total"`
	NextCursor   string                    `json:"next_cursor,omitempty"`
	Coverage     []quality.QualityCoverage `json:"coverage"`
}

type QualityEvidenceRequest struct {
	QueryRequest
	ReportID             string `json:"report_id,omitempty"`
	FindingID            string `json:"finding_id"`
	IncludeSourceContext bool   `json:"include_source_context"`
	MaxLines             int    `json:"max_lines,omitempty"`
	MaxContextBytes      int    `json:"max_context_bytes,omitempty"`
}

type QualityEvidenceResult struct {
	Status   string                          `json:"status"`
	Evidence *quality.QualityFindingEvidence `json:"evidence,omitempty"`
	Contexts []SourceContext                 `json:"contexts,omitempty"`
	Reason   string                          `json:"reason,omitempty"`
}

type QualityCompareRequest struct {
	SessionID        string      `json:"session_id"`
	Consistency      Consistency `json:"consistency,omitempty"`
	PreviousRevision int         `json:"previous_revision"`
	CurrentRevision  int         `json:"current_revision"`
	PreviousReportID string      `json:"previous_report_id,omitempty"`
	CurrentReportID  string      `json:"current_report_id,omitempty"`
	MaxBytes         int         `json:"max_bytes,omitempty"`
	MaxItems         int         `json:"max_items,omitempty"`
}

type QualityComparisonResult struct {
	Status     string                           `json:"status"`
	Comparison *quality.QualityReportComparison `json:"comparison,omitempty"`
	Reason     string                           `json:"reason,omitempty"`
}

type SessionOptions struct {
	AnalyzerRegistry *analysis.Registry
	// AnalyzerOptionsByID carries already parsed CLI/session options to the
	// shared planner. The map is keyed by logical analyzer ID so options for
	// one language can never leak into another analyzer.
	AnalyzerOptionsByID map[string]map[string]any
	Scanner             Scanner
	QualityCatalog      *quality.Catalog
	Profiles            QualityProfileResolver
	PolicyService       QualityPolicyService
	PolicyAuthorizer    PolicyAuthorizer
	Watcher             WatchBackend
	Fingerprinter       Fingerprinter
	SnapshotStore       SnapshotStore
	StartWatcher        bool
}

type Fingerprinter interface {
	Fingerprint(context.Context, string, []WatchRoot) (InputFingerprint, error)
}

type FileFingerprint struct {
	Path string        `json:"path"`
	Size int64         `json:"size"`
	Hash ContentDigest `json:"hash"`
}

type InputFingerprint struct {
	ManifestFingerprint ContentDigest     `json:"manifest_fingerprint"`
	ContentFingerprint  ContentDigest     `json:"content_fingerprint"`
	Files               []FileFingerprint `json:"files"`
}

type WatchBackend interface {
	Start(context.Context, string, []WatchRoot, WatchPolicy) (<-chan WatchEvent, error)
	Stop() error
}

type SnapshotStore interface {
	ReadLatestReady(string) (*RevisionRecord, bool)
	ReadRevision(string, int) (*RevisionRecord, bool)
	PublishAtomically(*RevisionRecord) (*RevisionRecord, error)
	WaitForRevision(context.Context, string, int) (*RevisionRecord, error)
}

type RevisionRecord struct {
	SessionID      string
	Input          InputFingerprint
	Snapshot       LiveSnapshot
	Run            orchestration.AnalysisRun
	Model          model.Model
	ScopeModels    map[string]model.Model
	ScopeResults   map[string]analysis.AnalysisResult
	QualityReports map[string]quality.QualityEvaluation
}
