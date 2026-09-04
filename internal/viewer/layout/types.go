package layout

import "encoding/json"

const (
	ConfigSchemaVersion = "arch-view.config/v1"
	ConfigFileName      = ".archview.json"

	layoutConfigSchemaVersion = ConfigSchemaVersion
	layoutConfigFileName      = ConfigFileName
)

// LayoutProfile is the user-controlled presentation layout. It deliberately
// contains no model, analyzer, or source-editing fields.
type LayoutProfile struct {
	Algorithm string         `json:"algorithm"`
	Options   map[string]any `json:"options"`
	Features  []string       `json:"features,omitempty"`
}

type LayoutAdapter struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Source  string `json:"source"`
}

type LayoutAlgorithmDefinition struct {
	ID                string   `json:"id"`
	ELKID             string   `json:"elk_id"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Category          string   `json:"category,omitempty"`
	KnownOptions      []string `json:"known_options"`
	SupportedFeatures []string `json:"supported_features,omitempty"`
	RendererSupport   string   `json:"renderer_support"`
}

type LayoutCategoryDefinition struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	KnownLayouters []string `json:"known_layouters,omitempty"`
}

// LayoutOptionDefinition is the normalized, UI-facing catalog entry. The
// bundled ELK API provides names/types/targets, while this application adds
// safe editability, defaults, and renderer support explicitly.
type LayoutOptionDefinition struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Group            string   `json:"group,omitempty"`
	Type             string   `json:"type"`
	Targets          []string `json:"targets"`
	Algorithms       []string `json:"algorithms,omitempty"`
	DefaultValue     any      `json:"default"`
	AllowedValues    []any    `json:"allowed_values"`
	Minimum          *float64 `json:"minimum,omitempty"`
	Maximum          *float64 `json:"maximum,omitempty"`
	MinimumExclusive bool     `json:"minimum_exclusive,omitempty"`
	MaximumExclusive bool     `json:"maximum_exclusive,omitempty"`
	Editable         bool     `json:"editable"`
	RendererSupport  string   `json:"renderer_support"`
	Control          string   `json:"control,omitempty"`
	SupportedTargets []string `json:"supported_targets,omitempty"`
	RequiredFeatures []string `json:"required_features,omitempty"`
	SupportNote      string   `json:"support_note,omitempty"`
}

type LayoutOptionsResponse struct {
	SchemaVersion string                      `json:"schema_version"`
	Adapter       LayoutAdapter               `json:"adapter"`
	Algorithms    []LayoutAlgorithmDefinition `json:"algorithms"`
	Categories    []LayoutCategoryDefinition  `json:"categories"`
	Options       []LayoutOptionDefinition    `json:"options"`
	Features      []FeatureDefinition         `json:"features"`
}

type LayoutDiagnostic struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Path     string `json:"path,omitempty"`
	Source   string `json:"source,omitempty"`
}

type LayoutConfigResponse struct {
	SchemaVersion string             `json:"schema_version"`
	Layout        LayoutProfile      `json:"layout"`
	Origin        string             `json:"origin"`
	ActivePath    string             `json:"active_path,omitempty"`
	Status        string             `json:"status"`
	CanSave       bool               `json:"can_save"`
	CanSaveAs     bool               `json:"can_save_as"`
	Diagnostics   []LayoutDiagnostic `json:"diagnostics"`
}

// Session owns the currently selected profile and its discovered persistence
// target. Callers serialize access when a session is shared by an HTTP server.
type Session struct {
	profile      LayoutProfile
	activePath   string
	activeOrigin string
	origin       string
	status       string
	canSave      bool
	canSaveAs    bool
	diagnostics  []LayoutDiagnostic
	sourceRoot   string
	analysisRaw  json.RawMessage
	rawDocument  json.RawMessage
}

type layoutConfigFile struct {
	SchemaVersion string          `json:"schema_version"`
	Layout        LayoutProfile   `json:"layout"`
	Analysis      json.RawMessage `json:"analysis,omitempty"`
	OKF           json.RawMessage `json:"okf,omitempty"`
}
