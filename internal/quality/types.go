// Package quality contains the versioned, language-neutral quality contract
// and evaluation engine. It deliberately does not import analyzer or model
// packages: those packages may attach a QualityEvaluation without creating a
// dependency cycle, while adapters can translate their facts at the edge.
package quality

const (
	SchemaVersion         = "arch-view.quality/v1"
	BaselineSchemaVersion = "arch-view.quality-baseline/v1"

	AssessmentExact  = "exact"
	AssessmentSignal = "signal"

	StatusActive       = "active"
	StatusSuppressed   = "suppressed"
	StatusBaseline     = "baseline"
	StatusResolved     = "resolved"
	StatusNotEvaluable = "not_evaluable"

	CoverageObserved     = "observed"
	CoverageAbsent       = "absent"
	CoverageUnknown      = "unknown"
	CoverageUnsupported  = "unsupported"
	CoveragePartial      = "partial"
	CoverageNotEvaluable = "not_evaluable"

	ValueInteger = "integer"
	ValueDecimal = "decimal"
	ValueBoolean = "boolean"
	ValueText    = "text"

	OperatorGreaterThanOrEqual = "greater_or_equal"
	OperatorGreaterThan        = "greater_than"
	OperatorLessThanOrEqual    = "less_or_equal"
	OperatorLessThan           = "less_than"
	OperatorEqual              = "equal"
	OperatorNotEqual           = "not_equal"

	SeverityInfo    = "info"
	SeverityWarning = "warning"
	SeverityError   = "error"
	SeverityBlocker = "blocker"
)

// QualityProfile is independently versioned from analyzer and model
// contracts. Its rule configuration is typed by each rule's namespace and
// schema version.
type QualityProfile struct {
	SchemaVersion  string                   `json:"schema_version"`
	ProfileID      string                   `json:"profile_id"`
	ProfileVersion string                   `json:"profile_version"`
	EnabledRules   []RuleBinding            `json:"enabled_rules"`
	SeverityPolicy TypedConfigBlock         `json:"severity_policy"`
	Constraints    []ArchitectureConstraint `json:"constraints"`
	Baseline       *BaselineRef             `json:"baseline,omitempty"`
	Extensions     []ExtensionBlock         `json:"extensions"`
}

type RuleBinding struct {
	RuleID      string           `json:"rule_id"`
	RuleVersion string           `json:"rule_version"`
	Enabled     bool             `json:"enabled"`
	Parameters  TypedConfigBlock `json:"parameters"`
	Severity    string           `json:"severity,omitempty"`
}

type TypedConfigBlock struct {
	Namespace     string `json:"namespace"`
	SchemaVersion string `json:"schema_version"`
	Payload       any    `json:"payload"`
}

type ExtensionBlock struct {
	Namespace     string `json:"namespace"`
	SchemaVersion string `json:"schema_version"`
	Capability    string `json:"capability"`
	Payload       any    `json:"payload"`
}

type BaselineRef struct {
	BaselineID string `json:"baseline_id"`
	Revision   string `json:"revision,omitempty"`
}

type Baseline struct {
	SchemaVersion string           `json:"schema_version"`
	BaselineID    string           `json:"baseline_id"`
	Revision      string           `json:"revision,omitempty"`
	Entries       []BaselineEntry  `json:"entries"`
	Extensions    []ExtensionBlock `json:"extensions"`
}

type BaselineEntry struct {
	FindingKey      string           `json:"finding_key"`
	RuleID          string           `json:"rule_id"`
	RuleVersion     string           `json:"rule_version"`
	ProfileID       string           `json:"profile_id"`
	ProfileVersion  string           `json:"profile_version"`
	FormulaVersions []FormulaVersion `json:"formula_versions"`
	Reason          string           `json:"reason"`
	Owner           string           `json:"owner,omitempty"`
}

type FormulaVersion struct {
	MetricID string `json:"metric_id"`
	Version  string `json:"version"`
}

type SuppressionInfo struct {
	BaselineID        string `json:"baseline_id,omitempty"`
	Reason            string `json:"reason"`
	ExactVersionMatch bool   `json:"exact_version_match"`
}

type ProviderIdentity struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type QualityEvaluation struct {
	SchemaVersion         string              `json:"schema_version"`
	EvaluationID          string              `json:"evaluation_id"`
	SourceSnapshotIDs     []string            `json:"source_snapshot_ids"`
	ModelRevision         string              `json:"model_revision,omitempty"`
	ProfileID             string              `json:"profile_id"`
	ProfileVersion        string              `json:"profile_version"`
	ProfileDigest         *ContentDigest      `json:"profile_digest,omitempty"`
	OptionsDigest         *ContentDigest      `json:"options_digest,omitempty"`
	Baseline              *BaselineRef        `json:"baseline,omitempty"`
	BaselineDigest        ContentDigest       `json:"baseline_digest,omitempty"`
	ProviderIdentities    []ProviderIdentity  `json:"provider_identities"`
	Coverage              []QualityCoverage   `json:"coverage"`
	Metrics               []MetricFact        `json:"metrics"`
	Findings              []QualityFinding    `json:"findings"`
	Diagnostics           []QualityDiagnostic `json:"diagnostics"`
	EvaluationFingerprint ContentDigest       `json:"evaluation_fingerprint"`
	ReportDigest          ContentDigest       `json:"report_digest"`
	Extensions            []ExtensionBlock    `json:"extensions"`
}

type ContentDigest struct {
	Algorithm string `json:"algorithm"`
	Value     string `json:"value"`
}

type MetricFact struct {
	ID             string           `json:"id"`
	SubjectRef     EntityRef        `json:"subject_ref"`
	MetricID       string           `json:"metric_id"`
	Value          MetricValue      `json:"value"`
	Unit           string           `json:"unit,omitempty"`
	FormulaID      string           `json:"formula_id"`
	FormulaVersion string           `json:"formula_version"`
	Provenance     FactProvenance   `json:"provenance"`
	Extensions     []ExtensionBlock `json:"extensions"`
}

type MetricValue struct {
	Kind  string `json:"kind"`
	Value any    `json:"value"`
}

type QualityFinding struct {
	ID                string           `json:"id"`
	FindingKey        string           `json:"finding_key"`
	RuleID            string           `json:"rule_id"`
	RuleVersion       string           `json:"rule_version"`
	AssessmentKind    string           `json:"assessment_kind"`
	Status            string           `json:"status"`
	Severity          string           `json:"severity"`
	SubjectRef        EntityRef        `json:"subject_ref"`
	MessageCode       string           `json:"message_code"`
	Message           string           `json:"message"`
	ObservedMetricIDs []string         `json:"observed_metric_ids"`
	Comparison        *Comparison      `json:"comparison,omitempty"`
	Evidence          FindingEvidence  `json:"evidence"`
	Limitations       []string         `json:"limitations,omitempty"`
	Suppression       *SuppressionInfo `json:"suppression,omitempty"`
	Provenance        FactProvenance   `json:"provenance"`
	Extensions        []ExtensionBlock `json:"extensions"`
}

type Comparison struct {
	Operator         string      `json:"operator"`
	ObservedMetricID string      `json:"observed_metric_id"`
	Limit            MetricValue `json:"limit"`
	Unit             string      `json:"unit,omitempty"`
}

type FindingEvidence struct {
	SourceSpans    []SourceSpan `json:"source_spans"`
	EntityRefs     []EntityRef  `json:"entity_refs"`
	RelationRefs   []EntityRef  `json:"relation_refs"`
	MetricRefs     []string     `json:"metric_refs"`
	DiagnosticRefs []string     `json:"diagnostic_refs"`
}

type FactProvenance struct {
	Status          string   `json:"status"`
	Basis           string   `json:"basis"`
	EvidenceIDs     []string `json:"evidence_ids"`
	Provider        string   `json:"provider"`
	ProviderVersion string   `json:"provider_version"`
}

type EntityRef struct {
	Kind       string `json:"kind"`
	ID         string `json:"id"`
	StableKey  string `json:"stable_key,omitempty"`
	SnapshotID string `json:"snapshot_id,omitempty"`
	ScopeID    string `json:"scope_id,omitempty"`
}

type SourceSpan struct {
	FileID           string        `json:"file_id"`
	Start            SpanPosition  `json:"start"`
	End              SpanPosition  `json:"end"`
	CoordinateSystem string        `json:"coordinate_system"`
	ContentHash      ContentDigest `json:"content_hash"`
}

type SpanPosition struct {
	ByteOffset int `json:"byte_offset"`
	Line       int `json:"line"`
	Column     int `json:"column"`
}

type QualityCoverage struct {
	RuleID                string         `json:"rule_id"`
	RuleVersion           string         `json:"rule_version"`
	Status                string         `json:"status"`
	RequiredCapabilities  []string       `json:"required_capabilities"`
	AvailableCapabilities []string       `json:"available_capabilities"`
	SubjectCount          *int           `json:"subject_count,omitempty"`
	EvaluatedCount        *int           `json:"evaluated_count,omitempty"`
	Reason                string         `json:"reason,omitempty"`
	Provenance            FactProvenance `json:"provenance"`
}

type QualityDiagnostic struct {
	Code        string         `json:"code"`
	Message     string         `json:"message"`
	Severity    string         `json:"severity"`
	RuleID      string         `json:"rule_id,omitempty"`
	RuleVersion string         `json:"rule_version,omitempty"`
	Field       string         `json:"field,omitempty"`
	SubjectRef  *EntityRef     `json:"subject_ref,omitempty"`
	Details     map[string]any `json:"details,omitempty"`
}

type ArchitectureConstraint struct {
	ID         string           `json:"id"`
	Kind       string           `json:"kind"`
	Parameters TypedConfigBlock `json:"parameters"`
	Provenance FactProvenance   `json:"provenance"`
	Extensions []ExtensionBlock `json:"extensions"`
}

// SourceSnapshot is the quality engine's neutral view of one authoritative
// source-index snapshot. Adapters translate the repository's source-index
// types into this input without giving quality ownership of extraction.
type SourceSnapshot struct {
	SnapshotID    string                 `json:"snapshot_id"`
	ScopeID       string                 `json:"scope_id"`
	Capabilities  []CapabilityDescriptor `json:"capabilities"`
	Coverage      []SourceCoverage       `json:"coverage"`
	Files         []SourceFile           `json:"files"`
	Symbols       []SourceSymbol         `json:"symbols"`
	Documentation []SourceDocumentation  `json:"documentation"`
	Relations     []SourceRelation       `json:"relations"`
	Metrics       []MetricFact           `json:"metrics"`
}

type CapabilityDescriptor struct {
	ID                 string   `json:"id"`
	Version            string   `json:"version,omitempty"`
	SupportedLanguages []string `json:"supported_languages,omitempty"`
	Description        string   `json:"description,omitempty"`
}

type SourceCoverage struct {
	Capability    string         `json:"capability"`
	SubjectKind   string         `json:"subject_kind"`
	Status        string         `json:"status"`
	EligibleCount *int           `json:"eligible_count,omitempty"`
	ObservedCount *int           `json:"observed_count,omitempty"`
	Reason        string         `json:"reason,omitempty"`
	Provenance    FactProvenance `json:"provenance"`
}

type SourceFile struct {
	ID             string         `json:"id"`
	StableKey      string         `json:"stable_key,omitempty"`
	Path           string         `json:"path"`
	Language       string         `json:"language"`
	LineCount      int            `json:"line_count"`
	ByteCount      int            `json:"byte_count"`
	ContentHash    ContentDigest  `json:"content_hash"`
	AnalysisStatus string         `json:"analysis_status"`
	Provenance     FactProvenance `json:"provenance"`
}

type SourceSymbol struct {
	ID               string           `json:"id"`
	StableKey        string           `json:"stable_key,omitempty"`
	Name             string           `json:"name"`
	QualifiedName    string           `json:"qualified_name,omitempty"`
	Category         string           `json:"category"`
	LanguageKind     string           `json:"language_kind,omitempty"`
	Visibility       string           `json:"visibility"`
	VisibilityStatus string           `json:"visibility_status,omitempty"`
	Locations        []SourceLocation `json:"locations"`
	BodySpan         *SourceSpan      `json:"body_span,omitempty"`
	DocumentationIDs []string         `json:"documentation_ids,omitempty"`
	// Structural facts are optional extractor observations. They are kept
	// explicit so SOLID signals can cite reported counts without pretending to
	// infer design intent from names or paths.
	MemberCount             *int           `json:"member_count,omitempty"`
	MethodCount             *int           `json:"method_count,omitempty"`
	DependencyCount         *int           `json:"dependency_count,omitempty"`
	ConcreteDependencyCount *int           `json:"concrete_dependency_count,omitempty"`
	InterfaceMethodCount    *int           `json:"interface_method_count,omitempty"`
	TypeSwitchCount         *int           `json:"type_switch_count,omitempty"`
	HierarchyDepth          *int           `json:"hierarchy_depth,omitempty"`
	DerivedTypeCount        *int           `json:"derived_type_count,omitempty"`
	AbstractionCount        *int           `json:"abstraction_count,omitempty"`
	StructuralFacts         map[string]int `json:"structural_facts,omitempty"`
	Provenance              FactProvenance `json:"provenance"`
}

type SourceLocation struct {
	Kind               string     `json:"kind"`
	Span               SourceSpan `json:"span"`
	SourceReferenceIDs []string   `json:"source_reference_ids,omitempty"`
}

type SourceDocumentation struct {
	ID         string         `json:"id"`
	SubjectRef EntityRef      `json:"subject_ref"`
	Status     string         `json:"status"`
	Spans      []SourceSpan   `json:"spans"`
	Provenance FactProvenance `json:"provenance"`
}

type SourceRelation struct {
	ID            string         `json:"id"`
	Category      string         `json:"category"`
	FromRef       EntityRef      `json:"from_ref"`
	ToRef         *EntityRef     `json:"to_ref,omitempty"`
	EvidenceSpans []SourceSpan   `json:"evidence_spans"`
	Provenance    FactProvenance `json:"provenance"`
}

type ArchitectureModel struct {
	ScopeID           string                     `json:"scope_id"`
	Aggregate         bool                       `json:"aggregate"`
	SourceSnapshotIDs []string                   `json:"source_snapshot_ids"`
	Modules           []ArchitectureModule       `json:"modules"`
	Relationships     []ArchitectureRelationship `json:"relationships"`
	Cycles            []ArchitectureCycle        `json:"cycles"`
	Layers            []ArchitectureLayer        `json:"layers"`
}

type ArchitectureModule struct {
	ID        string   `json:"id"`
	StableKey string   `json:"stable_key,omitempty"`
	Name      string   `json:"name"`
	Hierarchy []string `json:"hierarchy"`
	Tags      []string `json:"tags,omitempty"`
}

type ArchitectureRelationship struct {
	ID                 string   `json:"id"`
	Type               string   `json:"type"`
	FromModuleID       string   `json:"from_module_id"`
	ToModuleID         string   `json:"to_module_id,omitempty"`
	ToReferenceID      string   `json:"to_reference_id,omitempty"`
	SourceReferenceIDs []string `json:"source_reference_ids,omitempty"`
}

type ArchitectureCycle struct {
	ID              string   `json:"id"`
	ModuleIDs       []string `json:"module_ids"`
	RelationshipIDs []string `json:"relationship_ids"`
}

type ArchitectureLayer struct {
	Layer     int      `json:"layer"`
	ModuleIDs []string `json:"module_ids"`
}

// EvaluationInput keeps source and architecture facts explicit. Multiple
// source snapshots are evaluated independently unless a caller supplies an
// explicit combined architecture model.
type EvaluationInput struct {
	SourceSnapshots []SourceSnapshot   `json:"source_snapshots"`
	Architecture    *ArchitectureModel `json:"architecture,omitempty"`
	Baseline        *Baseline          `json:"baseline,omitempty"`
	Options         map[string]any     `json:"options,omitempty"`
}

type EvaluationContext struct {
	Profile      QualityProfile
	Input        EvaluationInput
	RuleBinding  *RuleBinding
	Snapshot     *SourceSnapshot
	Architecture *ArchitectureModel
}

type MetricBatch struct {
	Metrics     []MetricFact
	Coverage    []QualityCoverage
	Diagnostics []QualityDiagnostic
}

type RuleResult struct {
	Findings    []QualityFinding
	Coverage    []QualityCoverage
	Diagnostics []QualityDiagnostic
}
