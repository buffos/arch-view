package layout

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

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

func (session Session) response() LayoutConfigResponse {
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

func Catalog() LayoutOptionsResponse {
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

func layoutCatalog() LayoutOptionsResponse {
	return Catalog()
}

func cloneLayoutOption(option LayoutOptionDefinition) LayoutOptionDefinition {
	option.Targets = append([]string(nil), option.Targets...)
	option.Algorithms = append([]string(nil), option.Algorithms...)
	option.AllowedValues = append([]any(nil), option.AllowedValues...)
	return option
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
		canonicalKey := canonicalLayoutOptionID(key)
		if _, exists := normalizedOptions[canonicalKey]; exists {
			return LayoutProfile{}, analysis.NewHostError(analysis.ErrInvalidOptions, "layout profile contains duplicate option aliases", map[string]any{"option": canonicalKey})
		}
		normalizedOptions[canonicalKey] = value
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
	if handler, ok := layoutOptionHandlers[option.ID]; ok && handler.algorithmApplies != nil {
		return handler.algorithmApplies(option, algorithm)
	}
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
	if handler, ok := layoutOptionHandlers[option.ID]; ok && handler.validate != nil {
		return handler.validate(option, value)
	}
	return validateCatalogOptionValue(option, value)
}

func validateCatalogOptionValue(option LayoutOptionDefinition, value any) error {
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
	profile, _, err := decodeLayoutDocument(data)
	return profile, err
}

func decodeLayoutDocument(data []byte) (LayoutProfile, json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var config layoutConfigFile
	if err := decoder.Decode(&config); err != nil {
		return LayoutProfile{}, nil, analysis.WrapHostError(analysis.ErrInvalidOptions, "layout configuration is not valid JSON", err, nil)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return LayoutProfile{}, nil, analysis.WrapHostError(analysis.ErrInvalidOptions, "layout configuration contains trailing data", err, nil)
	}
	if config.SchemaVersion != layoutConfigSchemaVersion && config.SchemaVersion != "arch-view.config/v2" {
		return LayoutProfile{}, nil, analysis.NewHostError(analysis.ErrInvalidOptions, "layout configuration schema is unsupported", map[string]any{"schema_version": config.SchemaVersion, "expected": []string{layoutConfigSchemaVersion, "arch-view.config/v2"}})
	}
	if config.SchemaVersion == layoutConfigSchemaVersion && len(config.Analysis) > 0 && !bytes.Equal(bytes.TrimSpace(config.Analysis), []byte("null")) {
		return LayoutProfile{}, nil, analysis.NewHostError(analysis.ErrInvalidOptions, "v1 layout configuration cannot contain an analysis section", map[string]any{"field": "analysis"})
	}
	if config.SchemaVersion == "arch-view.config/v2" && (len(config.Analysis) == 0 || bytes.Equal(bytes.TrimSpace(config.Analysis), []byte("null"))) {
		return LayoutProfile{}, nil, analysis.NewHostError(analysis.ErrInvalidOptions, "v2 layout configuration requires an analysis section", map[string]any{"field": "analysis"})
	}
	if config.SchemaVersion == "arch-view.config/v2" && !jsonObject(config.Analysis) {
		return LayoutProfile{}, nil, analysis.NewHostError(analysis.ErrInvalidOptions, "v2 layout configuration analysis section must be an object", map[string]any{"field": "analysis"})
	}
	profile, err := validateLayoutProfile(config.Layout)
	if err != nil {
		return LayoutProfile{}, nil, err
	}
	return profile, append(json.RawMessage(nil), config.Analysis...), nil
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
	return encodeLayoutConfigWithAnalysis(profile, nil)
}

func encodeLayoutConfigWithAnalysis(profile LayoutProfile, analysisRaw json.RawMessage) ([]byte, error) {
	profile, err := validateLayoutProfile(profile)
	if err != nil {
		return nil, err
	}
	schemaVersion := layoutConfigSchemaVersion
	if len(bytes.TrimSpace(analysisRaw)) > 0 {
		if !jsonObject(analysisRaw) {
			return nil, analysis.NewHostError(analysis.ErrInvalidOptions, "v2 analysis configuration cannot be preserved because it is invalid", map[string]any{"field": "analysis"})
		}
		schemaVersion = "arch-view.config/v2"
	}
	data, err := json.MarshalIndent(layoutConfigFile{SchemaVersion: schemaVersion, Layout: profile, Analysis: append(json.RawMessage(nil), analysisRaw...)}, "", "  ")
	if err != nil {
		return nil, analysis.WrapHostError(analysis.ErrHostFailure, "layout configuration could not be encoded", err, nil)
	}
	return append(data, '\n'), nil
}

func jsonObject(data json.RawMessage) bool {
	if len(bytes.TrimSpace(data)) == 0 || bytes.Equal(bytes.TrimSpace(data), []byte("null")) || !json.Valid(data) {
		return false
	}
	var value map[string]json.RawMessage
	return json.Unmarshal(data, &value) == nil && value != nil
}

func discoverLayoutSession(sourceRoot string) Session {
	profile := defaultLayoutProfile()
	if sourceRoot == "" {
		return Session{
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
			profile, analysisRaw, decodeErr := decodeLayoutDocument(data)
			origin := layoutOriginForPath(sourceRoot, candidate)
			if decodeErr != nil {
				return Session{
					profile:      profileOrDefault(profile),
					activePath:   candidate,
					activeOrigin: origin,
					origin:       origin,
					status:       "invalid",
					canSaveAs:    true,
					diagnostics:  []LayoutDiagnostic{{Code: "invalid_config", Severity: "error", Message: decodeErr.Error(), Path: candidate, Source: "nearest configuration"}},
				}
			}
			return Session{profile: profile, activePath: candidate, activeOrigin: origin, origin: origin, status: "valid", canSave: true, canSaveAs: true, analysisRaw: analysisRaw}
		}
		if !os.IsNotExist(err) {
			return Session{profile: profile, activePath: candidate, activeOrigin: layoutOriginForPath(sourceRoot, candidate), origin: layoutOriginForPath(sourceRoot, candidate), status: "invalid", canSaveAs: true, diagnostics: []LayoutDiagnostic{{Code: "unreadable_config", Severity: "error", Message: "The nearest .archview.json could not be read: " + err.Error(), Path: candidate, Source: "nearest configuration"}}}
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			break
		}
	}
	return Session{profile: profile, origin: "default", status: "valid", canSaveAs: true}
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

func samePath(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}
