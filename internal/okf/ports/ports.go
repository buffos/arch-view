package ports

import (
	"context"

	"github.com/buffo/arch-view/internal/okf/domain"
)

// BundleScanner discovers candidates without interpreting their contents.
type BundleScanner interface {
	Scan(context.Context, string) ([]domain.BundleCandidate, []domain.Diagnostic)
}

// BundleValidator validates one candidate against the OKF conformance rules.
type BundleValidator interface {
	Validate(context.Context, domain.BundleCandidate) (domain.BundleCandidate, []domain.Diagnostic)
}

// BundleIndexer creates an immutable source snapshot for one valid candidate.
type BundleIndexer interface {
	Index(context.Context, domain.BundleCandidate) (domain.BundleIndex, error)
}

// RuleStrategy is an in-process, renderer-neutral extension point. Profile
// data may select a strategy but can never provide executable code.
type RuleStrategy interface {
	ID() string
	Version() string
	Description() string
	Evaluate(context.Context, domain.ConceptDocument, domain.RuleInvocation) (RuleResult, error)
}

type RuleResult struct {
	Visible        *bool
	Role           string
	Token          string
	Shape          string
	Label          string
	EffectiveState string
	Annotations    map[string]any
	Diagnostics    []domain.Diagnostic
}

// RuleEvaluator interprets a concept without exposing registry configuration.
type RuleEvaluator interface {
	Evaluate(context.Context, domain.ConceptDocument, domain.Profile) (RuleResult, []domain.Diagnostic)
}

// RuleParameterValidator lets a strategy validate its own declarative schema.
type RuleParameterValidator interface {
	ValidateParameters(map[string]any) error
}

type ParameterSchemaProvider interface {
	ParameterSchema() map[string]any
}

type ExtensionDescriber interface {
	Description() string
}

// ShapeProvider contributes declarative geometry in normalized coordinates.
type ShapeProvider interface {
	Definition() domain.ShapeDefinition
}

// DetailRenderer prepares display Markdown, never trusted HTML. The caller
// owns sanitization, source metadata and source-link boundary enforcement.
type DetailRenderer interface {
	ID() string
	Version() string
	Description() string
	ParameterSchema() map[string]any
	ValidateParameters(map[string]any) error
	Render(context.Context, domain.ConceptDocument, map[string]any) (string, error)
}

type DetailRendererResolver interface {
	ResolveDetailRenderer(string, string) (DetailRenderer, bool)
}

// DiagnosticProvider inspects an owned snapshot, without editing source or
// deciding navigation. Metadata describes its versioned diagnostic vocabulary.
type DiagnosticProvider interface {
	Metadata() Extension
	Diagnose(context.Context, domain.BundleIndex) ([]domain.Diagnostic, error)
}

// PresentationPropertyProvider contributes display properties only. The normal
// rule engine owns priority/conflict handling and the renderer owns safe output.
type PresentationPropertyProvider interface {
	Metadata() Extension
	ValidateParameters(map[string]any) error
	Properties(context.Context, domain.ConceptDocument, map[string]any) (PresentationProperties, error)
}

type PresentationProperties struct {
	Label       string         `json:"label,omitempty"`
	Token       string         `json:"token,omitempty"`
	Shape       string         `json:"shape,omitempty"`
	Annotations map[string]any `json:"annotations,omitempty"`
}

// RuleRegistry resolves stable strategy IDs and exposes metadata to clients.
type RuleRegistry interface {
	Register(RuleStrategy) error
	Resolve(string, string) (RuleStrategy, bool)
	Catalog() []Extension
}

type Extension struct {
	ID               string                  `json:"id"`
	Version          string                  `json:"version"`
	Kind             string                  `json:"kind"`
	Description      string                  `json:"description"`
	Capabilities     []string                `json:"capabilities"`
	ParameterSchema  map[string]any          `json:"parameter_schema,omitempty"`
	DefinitionSchema map[string]any          `json:"definition_schema,omitempty"`
	ShapeDefinition  *domain.ShapeDefinition `json:"shape_definition,omitempty"`
}

// RelationshipAdapter contributes source relationships without owning
// containment traversal or presentation policy.
type RelationshipAdapter interface {
	ID() string
	Version() string
	Relationships(context.Context, domain.BundleIndex) ([]domain.Relationship, []domain.Diagnostic)
}

// ConfigurationStore is the persistence seam used by the application layer.
type ConfigurationStore interface {
	Load(context.Context, string) (domain.ProjectConfiguration, error)
	Save(context.Context, string, domain.ProjectConfiguration, string, string, []byte) (domain.ProjectConfiguration, error)
}

// LayoutValidator keeps renderer-specific layout catalogs outside the OKF
// domain while allowing profile writes to validate their layout section.
type LayoutValidator interface {
	Validate(domain.LayoutSettings) error
}
