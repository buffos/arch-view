package analysis

// SourceIndexSchemaVersion is the independently versioned source-facts
// attachment. It deliberately does not change AnalyzerAPIVersion or the
// canonical architecture-model schema.
const SourceIndexSchemaVersion = "arch-view.source-index/v1"

const (
	SourceIndexSnapshotScope    = "scope"
	SourceIndexSnapshotCombined = "combined_projection"

	SourceIndexScopeMode    = "scope"
	SourceIndexCombinedMode = "combined"

	FactStatusObserved    = "observed"
	FactStatusAbsent      = "absent"
	FactStatusUnknown     = "unknown"
	FactStatusUnsupported = "unsupported"
	FactStatusPartial     = "partial"

	FileAnalysisComplete = "complete"
	FileAnalysisPartial  = "partial"
	FileAnalysisUnparsed = "unparsed"
	FileAnalysisUnknown  = "unknown"

	SymbolCategoryCallable  = "callable"
	SymbolCategoryType      = "type"
	SymbolCategoryNamespace = "namespace"
	SymbolCategoryValue     = "value"
	SymbolCategoryMember    = "member"
	SymbolCategoryMacro     = "macro"
	SymbolCategoryUnknown   = "unknown"

	DocumentationPresent     = "present"
	DocumentationAbsent      = "absent"
	DocumentationUnknown     = "unknown"
	DocumentationUnsupported = "unsupported"
	DocumentationPartial     = "partial"

	DocumentationComplete    = "complete"
	DocumentationSummaryOnly = "summary_only"
	DocumentationTruncated   = "truncated"
	DocumentationUnknownSize = "unknown"

	RelationContains = "contains"
	RelationDeclares = "declares"
	RelationUnknown  = "unknown"

	SourceSpanCoordinateSystem = "utf8-byte"
	SourceHashAlgorithm        = "hash:sha-256"
)

// SourceIndex is the optional, independently versioned source-facts
// attachment on an analysis result and canonical model. Snapshots are the
// authority; Projection is only a derived combined read model.
type SourceIndex struct {
	SchemaVersion string                `json:"schema_version"`
	Snapshots     []SourceIndexSnapshot `json:"snapshots"`
	Projection    *SourceIndexSnapshot  `json:"projection,omitempty"`
	Extensions    []ExtensionBlock      `json:"extensions"`
}

// SourceIndexSnapshot is one deterministic fact set for one analyzer scope or
// an explicitly derived combined projection.
type SourceIndexSnapshot struct {
	SnapshotID     string                 `json:"snapshot_id"`
	SnapshotKind   string                 `json:"snapshot_kind"`
	ScopeContext   ScopeContext           `json:"scope_context"`
	Producer       ProducerContext        `json:"producer"`
	Input          InputContext           `json:"input"`
	Capabilities   []CapabilityDescriptor `json:"capabilities"`
	Coverage       []CoverageRecord       `json:"coverage"`
	Files          []FileRecord           `json:"files"`
	Symbols        []SymbolRecord         `json:"symbols"`
	Documentation  []DocumentationRecord  `json:"documentation"`
	Occurrences    []SymbolOccurrence     `json:"occurrences"`
	Relations      []CodeRelation         `json:"relations"`
	Metrics        []MetricFact           `json:"metrics"`
	SnapshotDigest ContentDigest          `json:"snapshot_digest"`
	Extensions     []ExtensionBlock       `json:"extensions"`
}

type ScopeContext struct {
	ScopeID                 string             `json:"scope_id"`
	ProjectRoot             string             `json:"project_root"`
	SourceScopeFingerprint  ContentDigest      `json:"source_scope_fingerprint"`
	SourcePolicyFingerprint *ContentDigest     `json:"source_policy_fingerprint,omitempty"`
	RepositoryContext       *RepositoryContext `json:"repository_context,omitempty"`
	Mode                    string             `json:"mode"`
	SourceSnapshotIDs       []string           `json:"source_snapshot_ids,omitempty"`
}

type RepositoryContext struct {
	RootLabel string `json:"root_label,omitempty"`
}

type ProducerContext struct {
	AnalyzerID      string              `json:"analyzer_id"`
	AnalyzerVersion string              `json:"analyzer_version"`
	Extractors      []ExtractorIdentity `json:"extractors"`
	ProtocolVersion string              `json:"protocol_version,omitempty"`
}

type ExtractorIdentity struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type InputContext struct {
	EligibleFileCount     int           `json:"eligible_file_count"`
	SourceSetDigest       ContentDigest `json:"source_set_digest"`
	RequestedCapabilities []string      `json:"requested_capabilities"`
}

type CapabilityDescriptor struct {
	ID                 string   `json:"id"`
	Version            string   `json:"version,omitempty"`
	SupportedLanguages []string `json:"supported_languages,omitempty"`
	Description        string   `json:"description,omitempty"`
}

type CoverageRecord struct {
	Capability    string         `json:"capability"`
	SubjectKind   string         `json:"subject_kind"`
	Status        string         `json:"status"`
	EligibleCount *int           `json:"eligible_count,omitempty"`
	ObservedCount *int           `json:"observed_count,omitempty"`
	Reason        string         `json:"reason,omitempty"`
	Provenance    FactProvenance `json:"provenance"`
}

type FileRecord struct {
	ID             string           `json:"id"`
	Path           string           `json:"path"`
	Language       LanguageRef      `json:"language"`
	Roles          []string         `json:"roles"`
	Size           FileSize         `json:"size"`
	AnalysisStatus string           `json:"analysis_status"`
	Provenance     FactProvenance   `json:"provenance"`
	Extensions     []ExtensionBlock `json:"extensions"`
}

type LanguageRef struct {
	ID      string `json:"id"`
	Dialect string `json:"dialect,omitempty"`
}

type FileSize struct {
	LineCount   int           `json:"line_count"`
	ByteCount   int           `json:"byte_count"`
	ContentHash ContentDigest `json:"content_hash"`
}

type ContentDigest struct {
	Algorithm string `json:"algorithm"`
	Value     string `json:"value"`
}

type SymbolRecord struct {
	ID               string           `json:"id"`
	Name             string           `json:"name"`
	QualifiedName    string           `json:"qualified_name,omitempty"`
	Category         string           `json:"category"`
	LanguageKind     string           `json:"language_kind,omitempty"`
	Visibility       VisibilityFact   `json:"visibility"`
	Locations        []SymbolLocation `json:"locations"`
	BodySpan         *SourceSpan      `json:"body_span,omitempty"`
	DocumentationIDs []string         `json:"documentation_ids,omitempty"`
	// StructuralFacts are optional, extractor-reported counts consumed by
	// advisory quality signals. They are never inferred from symbol names.
	MemberCount             *int             `json:"member_count,omitempty"`
	MethodCount             *int             `json:"method_count,omitempty"`
	DependencyCount         *int             `json:"dependency_count,omitempty"`
	ConcreteDependencyCount *int             `json:"concrete_dependency_count,omitempty"`
	InterfaceMethodCount    *int             `json:"interface_method_count,omitempty"`
	TypeSwitchCount         *int             `json:"type_switch_count,omitempty"`
	HierarchyDepth          *int             `json:"hierarchy_depth,omitempty"`
	DerivedTypeCount        *int             `json:"derived_type_count,omitempty"`
	AbstractionCount        *int             `json:"abstraction_count,omitempty"`
	StructuralFacts         map[string]int   `json:"structural_facts,omitempty"`
	StableKey               string           `json:"stable_key,omitempty"`
	IdentityBasis           string           `json:"identity_basis,omitempty"`
	Provenance              FactProvenance   `json:"provenance"`
	Extensions              []ExtensionBlock `json:"extensions"`
}

type VisibilityFact struct {
	Classification string         `json:"classification"`
	LanguageValue  string         `json:"language_value,omitempty"`
	Provenance     FactProvenance `json:"provenance"`
}

type SymbolLocation struct {
	Kind               string     `json:"kind"`
	Span               SourceSpan `json:"span"`
	SourceReferenceIDs []string   `json:"source_reference_ids,omitempty"`
}

type DocumentationRecord struct {
	ID                 string           `json:"id"`
	SubjectRef         EntityRef        `json:"subject_ref"`
	SelectionGroup     string           `json:"selection_group"`
	Format             string           `json:"format"`
	RawText            string           `json:"raw_text,omitempty"`
	NormalizedText     string           `json:"normalized_text,omitempty"`
	Spans              []SourceSpan     `json:"spans"`
	SourceReferenceIDs []string         `json:"source_reference_ids,omitempty"`
	AttachmentBasis    string           `json:"attachment_basis"`
	PrecedenceRank     *int             `json:"precedence_rank,omitempty"`
	IsPrimary          bool             `json:"is_primary"`
	Status             string           `json:"status"`
	Completeness       string           `json:"completeness"`
	Provenance         FactProvenance   `json:"provenance"`
	Extensions         []ExtensionBlock `json:"extensions"`
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

type FactProvenance struct {
	Status          string     `json:"status"`
	Basis           string     `json:"basis"`
	EvidenceIDs     []string   `json:"evidence_ids"`
	Provider        string     `json:"provider"`
	ProviderVersion string     `json:"provider_version"`
	Score           *FactScore `json:"score,omitempty"`
}

type FactScore struct {
	Value        float64 `json:"value"`
	ScaleID      string  `json:"scale_id"`
	ScaleVersion string  `json:"scale_version"`
}

type EntityRef struct {
	Kind       string `json:"kind"`
	ID         string `json:"id"`
	SnapshotID string `json:"snapshot_id,omitempty"`
	ScopeID    string `json:"scope_id,omitempty"`
}

type CodeRelation struct {
	ID               string            `json:"id"`
	Category         string            `json:"category"`
	LanguageKind     string            `json:"language_kind,omitempty"`
	FromRef          EntityRef         `json:"from_ref"`
	ToRef            *EntityRef        `json:"to_ref,omitempty"`
	UnresolvedTarget *UnresolvedTarget `json:"unresolved_target,omitempty"`
	EvidenceSpans    []SourceSpan      `json:"evidence_spans"`
	Provenance       FactProvenance    `json:"provenance"`
	Extensions       []ExtensionBlock  `json:"extensions"`
}

type UnresolvedTarget struct {
	DisplayName   string `json:"display_name"`
	QualifiedName string `json:"qualified_name,omitempty"`
	LanguageKind  string `json:"language_kind,omitempty"`
}

type SymbolOccurrence struct {
	ID               string           `json:"id"`
	SymbolRef        *EntityRef       `json:"symbol_ref,omitempty"`
	TargetName       string           `json:"target_name"`
	SourceSpan       SourceSpan       `json:"source_span"`
	OccurrenceKind   string           `json:"occurrence_kind"`
	ResolutionStatus string           `json:"resolution_status"`
	Provenance       FactProvenance   `json:"provenance"`
	Extensions       []ExtensionBlock `json:"extensions"`
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

type ExtensionBlock struct {
	Namespace     string `json:"namespace"`
	SchemaVersion string `json:"schema_version"`
	Capability    string `json:"capability"`
	Payload       any    `json:"payload"`
}
