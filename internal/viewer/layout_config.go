package viewer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

const (
	layoutConfigSchemaVersion = "arch-view.config/v1"
	layoutConfigFileName      = ".archview.json"
)

// LayoutProfile is the user-controlled presentation layout. It deliberately
// contains no model, analyzer, or source-editing fields.
type LayoutProfile struct {
	Algorithm string         `json:"algorithm"`
	Options   map[string]any `json:"options"`
}

type layoutConfigFile struct {
	SchemaVersion string        `json:"schema_version"`
	Layout        LayoutProfile `json:"layout"`
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
}

type LayoutOptionsResponse struct {
	SchemaVersion string                      `json:"schema_version"`
	Adapter       LayoutAdapter               `json:"adapter"`
	Algorithms    []LayoutAlgorithmDefinition `json:"algorithms"`
	Categories    []LayoutCategoryDefinition  `json:"categories"`
	Options       []LayoutOptionDefinition    `json:"options"`
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

type layoutSession struct {
	profile      LayoutProfile
	activePath   string
	activeOrigin string
	origin       string
	status       string
	canSave      bool
	canSaveAs    bool
	diagnostics  []LayoutDiagnostic
}

type layoutApplyRequest struct {
	SchemaVersion string        `json:"schema_version"`
	Layout        LayoutProfile `json:"layout"`
}

type layoutSaveAsRequest struct {
	SchemaVersion  string        `json:"schema_version"`
	Layout         LayoutProfile `json:"layout"`
	DestinationDir string        `json:"destination_dir"`
	Confirm        bool          `json:"confirm"`
}

func defaultLayoutProfile() LayoutProfile {
	return LayoutProfile{Algorithm: "layered", Options: map[string]any{}}
}

func cloneLayoutProfile(profile LayoutProfile) LayoutProfile {
	options := make(map[string]any, len(profile.Options))
	for key, value := range profile.Options {
		options[key] = value
	}
	if profile.Algorithm == "" {
		profile.Algorithm = "layered"
	}
	profile.Options = options
	return profile
}

func (session layoutSession) response() LayoutConfigResponse {
	diagnostics := append([]LayoutDiagnostic(nil), session.diagnostics...)
	if diagnostics == nil {
		diagnostics = []LayoutDiagnostic{}
	}
	return LayoutConfigResponse{
		SchemaVersion: layoutConfigSchemaVersion,
		Layout:        cloneLayoutProfile(session.profile),
		Origin:        session.origin,
		ActivePath:    session.activePath,
		Status:        session.status,
		CanSave:       session.canSave,
		CanSaveAs:     session.canSaveAs,
		Diagnostics:   diagnostics,
	}
}

func layoutCatalog() LayoutOptionsResponse {
	algorithms := append([]LayoutAlgorithmDefinition(nil), pinnedELKAlgorithms...)
	categories := append([]LayoutCategoryDefinition(nil), pinnedELKCategories...)
	options := make([]LayoutOptionDefinition, len(pinnedELKOptions))
	copy(options, pinnedELKOptions)
	for index := range options {
		options[index] = enrichLayoutOption(options[index])
	}
	return LayoutOptionsResponse{
		SchemaVersion: layoutConfigSchemaVersion,
		Adapter:       LayoutAdapter{ID: "elkjs", Version: "pinned-bundle", Source: "embedded"},
		Algorithms:    algorithms,
		Categories:    categories,
		Options:       options,
	}
}

func enrichLayoutOption(option LayoutOptionDefinition) LayoutOptionDefinition {
	option = cloneLayoutOption(option)
	if strings.TrimSpace(option.Description) == "" {
		option.Description = "ELK layout option: " + option.Name + "."
	}
	if option.AllowedValues == nil {
		option.AllowedValues = []any{}
	}
	option.DefaultValue = "engine default"
	option.Editable = false
	option.RendererSupport = "unsupported"
	setMinimum := func(value float64) { option.Minimum = &value }
	setMaximum := func(value float64) { option.Maximum = &value }
	setEnum := func(values ...string) {
		option.AllowedValues = make([]any, len(values))
		for index, value := range values {
			option.AllowedValues[index] = value
		}
	}
	setSupported := func(defaultValue any) {
		if !layoutOptionTargetsParent(option) {
			return
		}
		option.DefaultValue = defaultValue
		option.Editable = true
		option.RendererSupport = "supported"
	}
	switch option.ID {
	case "org.eclipse.elk.direction":
		setSupported("RIGHT")
		setEnum("RIGHT", "LEFT", "DOWN", "UP")
	case "org.eclipse.elk.edgeRouting":
		setSupported("ORTHOGONAL")
		setEnum("NONE", "POLYLINE", "ORTHOGONAL", "SPLINES")
	case "org.eclipse.elk.aspectRatio":
		setSupported("engine default")
		setMinimum(0)
		option.MinimumExclusive = true
	case "org.eclipse.elk.spacing.nodeNode":
		setSupported(35.0)
		setMinimum(0)
	case "org.eclipse.elk.spacing.edgeNode":
		setSupported(10.0)
		setMinimum(0)
	case "org.eclipse.elk.spacing.edgeEdge":
		setSupported(5.0)
		setMinimum(0)
	case "org.eclipse.elk.layered.spacing.nodeNodeBetweenLayers":
		setSupported(84.0)
		setMinimum(0)
	case "org.eclipse.elk.layered.spacing.edgeNodeBetweenLayers":
		setSupported(10.0)
		setMinimum(0)
	case "org.eclipse.elk.layered.spacing.baseValue":
		setSupported("engine default")
		setMinimum(0)
	case "org.eclipse.elk.layered.spacing.edgeEdgeBetweenLayers":
		setSupported(10.0)
		setMinimum(0)
	case "org.eclipse.elk.layered.layering.strategy":
		setSupported("NETWORK_SIMPLEX")
		setEnum("NETWORK_SIMPLEX", "LONGEST_PATH", "LONGEST_PATH_SOURCE", "COFFMAN_GRAHAM", "INTERACTIVE", "STRETCH_WIDTH", "MIN_WIDTH", "BF_MODEL_ORDER", "DF_MODEL_ORDER")
	case "org.eclipse.elk.layered.cycleBreaking.strategy":
		setSupported("GREEDY")
		setEnum("GREEDY", "DEPTH_FIRST", "INTERACTIVE", "MODEL_ORDER", "GREEDY_MODEL_ORDER", "SCC_CONNECTIVITY", "SCC_NODE_TYPE", "DFS_NODE_ORDER", "BFS_NODE_ORDER")
	case "org.eclipse.elk.layered.crossingMinimization.strategy":
		setSupported("LAYER_SWEEP")
		setEnum("LAYER_SWEEP", "MEDIAN_LAYER_SWEEP", "INTERACTIVE", "NONE")
	case "org.eclipse.elk.layered.nodePlacement.strategy":
		setSupported("BRANDES_KOEPF")
		setEnum("SIMPLE", "INTERACTIVE", "LINEAR_SEGMENTS", "BRANDES_KOEPF", "NETWORK_SIMPLEX")
	case "org.eclipse.elk.layered.compaction.connectedComponents":
		setSupported(false)
	case "org.eclipse.elk.layered.thoroughness":
		setSupported(7.0)
		setMinimum(1)
		setMaximum(100)
	case "org.eclipse.elk.layered.mergeEdges":
		setSupported(false)
	case "org.eclipse.elk.layered.mergeHierarchyEdges":
		setSupported(false)
	case "org.eclipse.elk.layered.feedbackEdges":
		setSupported(false)
	case "org.eclipse.elk.layered.crossingMinimization.forceNodeModelOrder":
		setSupported(false)
	case "org.eclipse.elk.separateConnectedComponents":
		setSupported(true)
	case "org.eclipse.elk.interactive":
		setSupported(false)
	case "org.eclipse.elk.interactiveLayout":
		setSupported(false)
	case "org.eclipse.elk.randomSeed":
		setSupported(1.0)
		setMinimum(0)
	}
	return option
}

func cloneLayoutOption(option LayoutOptionDefinition) LayoutOptionDefinition {
	option.Targets = append([]string(nil), option.Targets...)
	option.Algorithms = append([]string(nil), option.Algorithms...)
	option.AllowedValues = append([]any(nil), option.AllowedValues...)
	return option
}

func layoutOptionTargetsParent(option LayoutOptionDefinition) bool {
	for _, target := range option.Targets {
		if target == "PARENTS" {
			return true
		}
	}
	return false
}

func layoutOptionByID(id string) (LayoutOptionDefinition, bool) {
	for _, option := range pinnedELKOptions {
		if option.ID == id {
			return enrichLayoutOption(option), true
		}
	}
	return LayoutOptionDefinition{}, false
}

func layoutAlgorithmByID(id string) (LayoutAlgorithmDefinition, bool) {
	for _, algorithm := range pinnedELKAlgorithms {
		if algorithm.ID == id {
			return algorithm, true
		}
	}
	return LayoutAlgorithmDefinition{}, false
}

func validateLayoutProfile(profile LayoutProfile) (LayoutProfile, error) {
	if strings.TrimSpace(profile.Algorithm) == "" {
		return LayoutProfile{}, analysis.NewHostError(analysis.ErrInvalidOptions, "layout algorithm is required", map[string]any{"field": "layout.algorithm"})
	}
	profile = cloneLayoutProfile(profile)
	profile.Algorithm = strings.TrimSpace(profile.Algorithm)
	if _, ok := layoutAlgorithmByID(profile.Algorithm); !ok {
		return LayoutProfile{}, analysis.NewHostError(analysis.ErrUnsupportedOption, "layout algorithm is not provided by the pinned ELK adapter", map[string]any{"algorithm": profile.Algorithm})
	}
	if profile.Options == nil {
		profile.Options = map[string]any{}
	}
	normalizedOptions := make(map[string]any, len(profile.Options))
	for key, value := range profile.Options {
		key = canonicalLayoutOptionID(key)
		normalizedOptions[key] = value
	}
	profile.Options = normalizedOptions
	for key, value := range profile.Options {
		option, ok := layoutOptionByID(key)
		if !ok {
			return LayoutProfile{}, analysis.NewHostError(analysis.ErrUnsupportedOption, "layout option is not in the pinned ELK catalog", map[string]any{"option": key})
		}
		if !option.Editable || option.RendererSupport != "supported" {
			return LayoutProfile{}, analysis.NewHostError(analysis.ErrUnsupportedOption, "layout option is cataloged but not supported by the viewer renderer", map[string]any{"option": key})
		}
		if !layoutOptionApplies(option, profile.Algorithm) {
			return LayoutProfile{}, analysis.NewHostError(analysis.ErrInvalidOptions, "layout option does not apply to the selected algorithm", map[string]any{"option": key, "algorithm": profile.Algorithm})
		}
		if err := validateLayoutOptionValue(option, value); err != nil {
			return LayoutProfile{}, err
		}
	}
	return profile, nil
}

func canonicalLayoutOptionID(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "elk.") {
		return "org.eclipse." + value
	}
	return value
}

func layoutOptionApplies(option LayoutOptionDefinition, algorithm string) bool {
	if len(option.Algorithms) == 0 {
		return option.Editable && option.RendererSupport == "supported"
	}
	for _, candidate := range option.Algorithms {
		if candidate == "all" || candidate == algorithm {
			return true
		}
	}
	return false
}

func validateLayoutOptionValue(option LayoutOptionDefinition, value any) error {
	invalid := func(message string) error {
		return analysis.NewHostError(analysis.ErrInvalidOptions, message, map[string]any{"option": option.ID, "type": option.Type, "value": value})
	}
	switch option.Type {
	case "BOOLEAN":
		if _, ok := value.(bool); !ok {
			return invalid("layout option must be a boolean")
		}
	case "INT":
		number, ok := jsonNumber(value)
		if !ok || !isFiniteNumber(number) || number != math.Trunc(number) {
			return invalid("layout option must be an integer")
		}
		if outsideLayoutOptionBounds(option, number) {
			return invalid("layout option is outside its supported range")
		}
	case "DOUBLE":
		number, ok := jsonNumber(value)
		if !ok || !isFiniteNumber(number) {
			return invalid("layout option must be a number")
		}
		if outsideLayoutOptionBounds(option, number) {
			return invalid("layout option is outside its supported range")
		}
	case "ENUM", "STRING":
		text, ok := value.(string)
		if !ok {
			return invalid("layout option must be a string")
		}
		if len(option.AllowedValues) > 0 {
			allowed := false
			for _, candidate := range option.AllowedValues {
				if text == candidate {
					allowed = true
					break
				}
			}
			if !allowed {
				return invalid("layout option value is not allowed")
			}
		}
	default:
		return invalid("layout option type is not editable by this viewer")
	}
	return nil
}

func isFiniteNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func outsideLayoutOptionBounds(option LayoutOptionDefinition, value float64) bool {
	if option.Minimum != nil && (value < *option.Minimum || option.MinimumExclusive && value == *option.Minimum) {
		return true
	}
	if option.Maximum != nil && (value > *option.Maximum || option.MaximumExclusive && value == *option.Maximum) {
		return true
	}
	return false
}

func jsonNumber(value any) (float64, bool) {
	switch number := value.(type) {
	case float64:
		return number, true
	case float32:
		return float64(number), true
	case int:
		return float64(number), true
	case int64:
		return float64(number), true
	case json.Number:
		parsed, err := number.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

func decodeLayoutConfig(data []byte) (LayoutProfile, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var config layoutConfigFile
	if err := decoder.Decode(&config); err != nil {
		return LayoutProfile{}, analysis.WrapHostError(analysis.ErrInvalidOptions, "layout configuration is not valid JSON", err, nil)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return LayoutProfile{}, analysis.WrapHostError(analysis.ErrInvalidOptions, "layout configuration contains trailing data", err, nil)
	}
	if config.SchemaVersion != layoutConfigSchemaVersion {
		return LayoutProfile{}, analysis.NewHostError(analysis.ErrInvalidOptions, "layout configuration schema is unsupported", map[string]any{"schema_version": config.SchemaVersion, "expected": layoutConfigSchemaVersion})
	}
	return validateLayoutProfile(config.Layout)
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("more than one JSON value")
		}
		return err
	}
	return nil
}

func encodeLayoutConfig(profile LayoutProfile) ([]byte, error) {
	profile, err := validateLayoutProfile(profile)
	if err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(layoutConfigFile{SchemaVersion: layoutConfigSchemaVersion, Layout: profile}, "", "  ")
	if err != nil {
		return nil, analysis.WrapHostError(analysis.ErrHostFailure, "layout configuration could not be encoded", err, nil)
	}
	return append(data, '\n'), nil
}

func discoverLayoutSession(sourceRoot string) layoutSession {
	profile := defaultLayoutProfile()
	if sourceRoot == "" {
		return layoutSession{
			profile:     profile,
			origin:      "session",
			status:      "valid",
			canSaveAs:   false,
			diagnostics: []LayoutDiagnostic{{Code: "persistence_unavailable", Severity: "info", Message: "This model-only session can apply layout settings for the current session, but has no project directory for persistence."}},
		}
	}
	for directory := sourceRoot; ; directory = filepath.Dir(directory) {
		candidate := filepath.Join(directory, layoutConfigFileName)
		data, err := os.ReadFile(candidate)
		if err == nil {
			profile, decodeErr := decodeLayoutConfig(data)
			origin := layoutOriginForPath(sourceRoot, candidate)
			if decodeErr != nil {
				return layoutSession{
					profile:      profileOrDefault(profile),
					activePath:   candidate,
					activeOrigin: origin,
					origin:       origin,
					status:       "invalid",
					canSaveAs:    true,
					diagnostics:  []LayoutDiagnostic{{Code: "invalid_config", Severity: "error", Message: decodeErr.Error(), Path: candidate, Source: "nearest configuration"}},
				}
			}
			return layoutSession{profile: profile, activePath: candidate, activeOrigin: origin, origin: origin, status: "valid", canSave: true, canSaveAs: true}
		}
		if !os.IsNotExist(err) {
			return layoutSession{profile: profile, activePath: candidate, activeOrigin: layoutOriginForPath(sourceRoot, candidate), origin: layoutOriginForPath(sourceRoot, candidate), status: "invalid", canSaveAs: true, diagnostics: []LayoutDiagnostic{{Code: "unreadable_config", Severity: "error", Message: "The nearest .archview.json could not be read: " + err.Error(), Path: candidate, Source: "nearest configuration"}}}
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			break
		}
	}
	return layoutSession{profile: profile, origin: "default", status: "valid", canSaveAs: true}
}

func profileOrDefault(profile LayoutProfile) LayoutProfile {
	if profile.Algorithm == "" {
		return defaultLayoutProfile()
	}
	return cloneLayoutProfile(profile)
}

func layoutOriginForPath(sourceRoot, configPath string) string {
	if samePath(filepath.Dir(configPath), sourceRoot) {
		return "project"
	}
	return "ancestor"
}

func pathIsInSourceChain(sourceRoot, configPath string) bool {
	if sourceRoot == "" {
		return false
	}
	for directory := sourceRoot; ; directory = filepath.Dir(directory) {
		if samePath(filepath.Join(directory, layoutConfigFileName), configPath) {
			return true
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return false
		}
	}
}

func activeLayoutOrigin(sourceRoot, configPath string) string {
	if pathIsInSourceChain(sourceRoot, configPath) {
		return layoutOriginForPath(sourceRoot, configPath)
	}
	return "custom"
}
