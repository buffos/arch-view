package domain

import "time"

const (
	BundleDiscovered  = "discovered"
	BundleValid       = "valid"
	BundleInvalid     = "invalid"
	BundleUnavailable = "unavailable"
	BundleUnreadable  = "unreadable"

	ProfileValid            = "valid"
	ProfileInvalid          = "invalid"
	ProfilePartiallyApplied = "partially_applicable"
	ProfileUnavailable      = "unavailable"

	ProjectionRequested  = "requested"
	ProjectionEvaluating = "evaluating"
	ProjectionReady      = "ready"
	ProjectionTruncated  = "truncated"
	ProjectionFailed     = "failed"
	ProjectionCancelled  = "cancelled"
	ProjectionSuperseded = "superseded"

	RelationshipContainment = "containment"
	RelationshipSemantic    = "semantic_link"
)

type Diagnostic struct {
	Code           string         `json:"code"`
	Severity       string         `json:"severity"`
	Category       string         `json:"category"`
	Message        string         `json:"message"`
	BundleID       string         `json:"bundle_id,omitempty"`
	ConceptID      string         `json:"concept_id,omitempty"`
	RelationshipID string         `json:"relationship_id,omitempty"`
	ProfileID      string         `json:"profile_id,omitempty"`
	OperationID    string         `json:"operation_id,omitempty"`
	Path           string         `json:"path,omitempty"`
	Details        map[string]any `json:"details,omitempty"`
	Recovery       string         `json:"recovery,omitempty"`
}

type Provenance struct {
	Source      string `json:"source"`
	Path        string `json:"path,omitempty"`
	Explanation string `json:"explanation,omitempty"`
}

type BundleCandidate struct {
	BundleID       string       `json:"bundle_id"`
	RelativePath   string       `json:"relative_path"`
	AbsolutePath   string       `json:"-"`
	Status         string       `json:"status"`
	Selectable     bool         `json:"selectable"`
	ConceptCount   int          `json:"concept_count"`
	SourceRevision string       `json:"source_revision,omitempty"`
	Diagnostics    []Diagnostic `json:"diagnostics"`
}

type BundleCatalog struct {
	ProjectID       string            `json:"project_id"`
	DefaultBundleID string            `json:"default_bundle_id,omitempty"`
	Bundles         []BundleCandidate `json:"bundles"`
	Diagnostics     []Diagnostic      `json:"diagnostics"`
	Revision        string            `json:"revision,omitempty"`
}

type BundleSummary struct {
	BundleID       string       `json:"bundle_id"`
	SourceRevision string       `json:"source_revision"`
	ConceptCount   int          `json:"concept_count"`
	LinkCount      int          `json:"link_count"`
	Files          []string     `json:"files"`
	Diagnostics    []Diagnostic `json:"diagnostics"`
}

type Link struct {
	ID         string `json:"id"`
	RawTarget  string `json:"raw_target"`
	TargetPath string `json:"target_path,omitempty"`
	TargetID   string `json:"target_concept_id,omitempty"`
	Fragment   string `json:"fragment,omitempty"`
	Text       string `json:"text,omitempty"`
	Kind       string `json:"kind"`
	Resolved   bool   `json:"resolved"`
	External   bool   `json:"external"`
	Safe       bool   `json:"safe"`
}

type ConceptDocument struct {
	ConceptID          string         `json:"concept_id"`
	SourcePath         string         `json:"source_path"`
	Title              string         `json:"title,omitempty"`
	Description        string         `json:"description,omitempty"`
	Type               string         `json:"type"`
	Tags               []string       `json:"tags,omitempty"`
	Frontmatter        map[string]any `json:"frontmatter"`
	UnknownFrontmatter map[string]any `json:"unknown_frontmatter,omitempty"`
	Markdown           string         `json:"markdown"`
	Links              []Link         `json:"links"`
	ExplicitParents    []string       `json:"explicit_parents,omitempty"`
	ExplicitChildren   []string       `json:"explicit_children,omitempty"`
	Provenance         []Provenance   `json:"provenance"`
	Reserved           bool           `json:"reserved,omitempty"`
}

type Relationship struct {
	RelationshipID string       `json:"relationship_id"`
	Kind           string       `json:"kind"`
	From           string       `json:"from"`
	To             string       `json:"to"`
	Provenance     []Provenance `json:"provenance"`
}

type BundleIndex struct {
	BundleID       string                     `json:"bundle_id"`
	Root           string                     `json:"-"`
	SourceRevision string                     `json:"source_revision"`
	Documents      map[string]ConceptDocument `json:"documents"`
	ConceptOrder   []string                   `json:"concept_order"`
	Relationships  []Relationship             `json:"relationships"`
	Diagnostics    []Diagnostic               `json:"diagnostics"`
}

type RuleInvocation struct {
	RuleID     string         `json:"rule_id"`
	Version    string         `json:"version,omitempty"`
	Priority   int            `json:"priority"`
	Enabled    bool           `json:"enabled"`
	Parameters map[string]any `json:"parameters,omitempty"`
}

type HierarchySettings struct {
	UseExplicit          bool `json:"use_explicit"`
	UseFilesystem        bool `json:"use_filesystem_fallback"`
	Configured           bool `json:"-"`
	ExplicitConfigured   bool `json:"-"`
	FilesystemConfigured bool `json:"-"`
}

type RelationshipSettings struct {
	ShowContainment       bool `json:"show_containment"`
	ShowSemantic          bool `json:"show_semantic_links"`
	Configured            bool `json:"-"`
	ContainmentConfigured bool `json:"-"`
	SemanticConfigured    bool `json:"-"`
}

type StateSettings struct {
	Field                  string            `json:"field,omitempty"`
	Mapping                map[string]string `json:"mapping,omitempty"`
	RollUp                 bool              `json:"roll_up"`
	ShowDeclared           bool              `json:"show_declared"`
	Configured             bool              `json:"-"`
	FieldConfigured        bool              `json:"-"`
	MappingConfigured      bool              `json:"-"`
	RollUpConfigured       bool              `json:"-"`
	ShowDeclaredConfigured bool              `json:"-"`
}

type NodeField struct {
	Source    string `json:"source"`
	Label     string `json:"label,omitempty"`
	MaxLength int    `json:"max_length,omitempty"`
}

type NavigationSettings struct {
	DefaultDepth            int  `json:"default_depth"`
	MaxNodes                int  `json:"max_nodes"`
	MaxRelationships        int  `json:"max_relationships"`
	DepthConfigured         bool `json:"-"`
	NodesConfigured         bool `json:"-"`
	RelationshipsConfigured bool `json:"-"`
}

type StyleToken struct {
	ID              string  `json:"id"`
	Fill            string  `json:"fill,omitempty"`
	Stroke          string  `json:"stroke,omitempty"`
	Text            string  `json:"text,omitempty"`
	Shape           string  `json:"shape,omitempty"`
	Emphasis        string  `json:"emphasis,omitempty"`
	StrokeWidth     float64 `json:"stroke_width,omitempty"`
	StrokeDasharray string  `json:"stroke_dasharray,omitempty"`
}

// StyleDecoration is a structural overlay applied after the selected token.
// Empty values intentionally leave the token's corresponding value intact.
type StyleDecoration struct {
	Fill            string  `json:"fill,omitempty"`
	Stroke          string  `json:"stroke,omitempty"`
	Text            string  `json:"text,omitempty"`
	StrokeWidth     float64 `json:"stroke_width,omitempty"`
	StrokeDasharray string  `json:"stroke_dasharray,omitempty"`
}

type StyleSettings struct {
	DefaultToken string                     `json:"default_token"`
	Tokens       map[string]StyleToken      `json:"tokens,omitempty"`
	StateTokens  map[string]string          `json:"state_tokens,omitempty"`
	Decorations  map[string]StyleDecoration `json:"decorations,omitempty"`
}

type DetailSettings struct {
	Renderer          *DetailRendererSelection `json:"renderer,omitempty"`
	ShowRawMarkdown   bool                     `json:"show_raw_markdown"`
	ShowUnknown       bool                     `json:"show_unknown_frontmatter"`
	RawConfigured     bool                     `json:"-"`
	UnknownConfigured bool                     `json:"-"`
}

type DetailRendererSelection struct {
	ID         string         `json:"id"`
	Version    string         `json:"version"`
	Parameters map[string]any `json:"parameters,omitempty"`
}

type LayoutSettings struct {
	Algorithm string         `json:"algorithm,omitempty"`
	Options   map[string]any `json:"options,omitempty"`
}

type Profile struct {
	ProfileID     string               `json:"profile_id"`
	Name          string               `json:"name"`
	Origin        string               `json:"origin"`
	Bases         []string             `json:"bases"`
	Rules         []RuleInvocation     `json:"rules"`
	Hierarchy     HierarchySettings    `json:"hierarchy"`
	Relationships RelationshipSettings `json:"relationships"`
	State         StateSettings        `json:"state"`
	NodeFields    []NodeField          `json:"node_fields,omitempty"`
	Navigation    NavigationSettings   `json:"navigation"`
	Style         StyleSettings        `json:"style"`
	Details       DetailSettings       `json:"details"`
	Layout        LayoutSettings       `json:"layout"`
	Revision      string               `json:"revision"`
	Status        string               `json:"status"`
	Immutable     bool                 `json:"immutable"`
	extensions    profileExtensions
}

type ProfileBinding struct {
	BundleID  string `json:"bundle_id"`
	ProfileID string `json:"profile_id,omitempty"`
}

type ProfileCatalog struct {
	Profiles              []Profile        `json:"profiles"`
	Bindings              []ProfileBinding `json:"bindings"`
	RegistryRevision      string           `json:"registry_revision"`
	ConfigurationRevision string           `json:"configuration_revision,omitempty"`
	Diagnostics           []Diagnostic     `json:"diagnostics"`
}

type ProjectionSource struct {
	BundleID       string `json:"bundle_id"`
	SourceRevision string `json:"source_revision"`
}

type ProjectionProfile struct {
	ProfileID       string         `json:"profile_id"`
	ProfileRevision string         `json:"profile_revision"`
	Layout          LayoutSettings `json:"layout"`
}

type NavigationState struct {
	CanGoBack   bool     `json:"can_go_back"`
	FocusRoot   string   `json:"focus_root,omitempty"`
	Depth       int      `json:"depth"`
	Full        bool     `json:"full"`
	Breadcrumbs []string `json:"breadcrumbs"`
}

type SceneNode struct {
	ID                 string           `json:"id"`
	ConceptID          string           `json:"concept_id"`
	SourcePath         string           `json:"source_path"`
	Title              string           `json:"title"`
	Label              string           `json:"label"`
	Type               string           `json:"type,omitempty"`
	Role               string           `json:"role,omitempty"`
	DeclaredState      string           `json:"declared_state,omitempty"`
	EffectiveState     string           `json:"effective_state,omitempty"`
	PresentationFields []NodeFieldValue `json:"presentation_fields"`
	PresentationStyle  StyleToken       `json:"presentation_style"`
	Shape              string           `json:"shape"`
	ShapeDefinition    *ShapeDefinition `json:"shape_definition,omitempty"`
	PresentationToken  string           `json:"presentation_token"`
	IsRoot             bool             `json:"is_root"`
	IsRollup           bool             `json:"is_rollup"`
	Annotations        map[string]any   `json:"annotations,omitempty"`
	HierarchyPath      []string         `json:"hierarchy_path"`
	ChildrenVisible    bool             `json:"children_visible"`
	Depth              int              `json:"depth"`
}

type NodeFieldValue struct {
	Source string `json:"source"`
	Label  string `json:"label"`
	Value  string `json:"value"`
}

type SceneRelationship struct {
	ID              string       `json:"id"`
	RelationshipID  string       `json:"relationship_id"`
	Kind            string       `json:"kind"`
	From            string       `json:"from"`
	To              string       `json:"to"`
	FromVisibleID   string       `json:"from_visible_id"`
	ToVisibleID     string       `json:"to_visible_id"`
	Label           string       `json:"label,omitempty"`
	AccessibleLabel string       `json:"accessible_label"`
	Provenance      []Provenance `json:"provenance"`
}

type ProjectionCounts struct {
	VisibleNodes         int `json:"visible_nodes"`
	HiddenNodes          int `json:"hidden_nodes"`
	VisibleRelationships int `json:"visible_relationships"`
	HiddenRelationships  int `json:"hidden_relationships"`
}

type LegendEntry struct {
	Token string `json:"token"`
	Label string `json:"label"`
	Shape string `json:"shape"`
	Fill  string `json:"fill,omitempty"`
}

type ProjectionSnapshot struct {
	Status             string              `json:"status"`
	ProjectionRevision string              `json:"projection_revision"`
	Source             ProjectionSource    `json:"source"`
	Profile            ProjectionProfile   `json:"profile"`
	Navigation         NavigationState     `json:"navigation"`
	Nodes              []SceneNode         `json:"nodes"`
	Relationships      []SceneRelationship `json:"relationships"`
	Counts             ProjectionCounts    `json:"counts"`
	Legend             []LegendEntry       `json:"legend"`
	Diagnostics        []Diagnostic        `json:"diagnostics"`
	GeneratedAt        time.Time           `json:"generated_at"`
}

type RenderedMarkdown struct {
	Format  string `json:"format"`
	Content string `json:"content"`
	Links   []Link `json:"links"`
}

type ConceptDetail struct {
	ConceptID        string            `json:"concept_id"`
	BundleID         string            `json:"bundle_id"`
	SourceRevision   string            `json:"source_revision"`
	Overview         map[string]string `json:"overview"`
	MappedMetadata   map[string]any    `json:"mapped_metadata"`
	DeclaredState    string            `json:"declared_state,omitempty"`
	EffectiveState   string            `json:"effective_state,omitempty"`
	RenderedMarkdown RenderedMarkdown  `json:"rendered_markdown"`
	RawMarkdown      string            `json:"raw_markdown,omitempty"`
	Frontmatter      map[string]any    `json:"frontmatter"`
	Containment      map[string]any    `json:"containment"`
	SemanticLinks    []Link            `json:"semantic_links"`
	Provenance       []Provenance      `json:"provenance"`
	Diagnostics      []Diagnostic      `json:"diagnostics"`
}

type ProjectConfiguration struct {
	SchemaVersion string           `json:"schema_version"`
	DefaultGraph  string           `json:"default_graph,omitempty"`
	Bindings      []ProfileBinding `json:"bindings,omitempty"`
	Profiles      []Profile        `json:"profiles,omitempty"`
	Revision      string           `json:"revision,omitempty"`
}

type Session struct {
	SessionID    string
	BundleID     string
	ProfileID    string
	FocusRoot    string
	Depth        int
	Full         bool
	History      []NavigationState
	LastSnapshot *ProjectionSnapshot
	Request      uint64
}
