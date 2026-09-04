// Package config owns the persisted .archview.json boundary. It validates
// analysis settings before planning or executing an analyzer while leaving
// layout semantics in the viewer layout package.
package config

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/orchestration"
	"github.com/buffo/arch-view/internal/viewer/layout"
)

const (
	SchemaVersionV1     = "arch-view.config/v1"
	SchemaVersionV2     = "arch-view.config/v2"
	FileName            = ".archview.json"
	ConfigSchemaVersion = SchemaVersionV1
	ConfigFileName      = FileName
)

// Configuration is one complete nearest-ancestor profile. A v1 profile has
// no Analysis value; a v2 profile has both layout and analysis sections.
type Configuration struct {
	SchemaVersion  string                 `json:"schema_version"`
	Layout         layout.LayoutProfile   `json:"layout"`
	Analysis       *AnalysisConfiguration `json:"analysis,omitempty"`
	Path           string                 `json:"-"`
	Origin         string                 `json:"-"`
	Found          bool                   `json:"-"`
	Fingerprint    string                 `json:"-"`
	repositoryRoot string
	invocationRoot string
}

// AnalysisConfiguration is the normalized analysis portion of a v2 profile.
// Diagnostics are planning inputs for unavailable assignments and never carry
// persisted option values.
type AnalysisConfiguration struct {
	Exclude     []string                            `json:"exclude,omitempty"`
	Include     []orchestration.AnalyzerIncludeRule `json:"include,omitempty"`
	Assignments []orchestration.AnalyzerAssignment  `json:"assignments,omitempty"`
	Diagnostics []analysis.Diagnostic               `json:"diagnostics,omitempty"`
	Fingerprint string                              `json:"-"`
}

type rawConfiguration struct {
	SchemaVersion string          `json:"schema_version"`
	Layout        json.RawMessage `json:"layout"`
	Analysis      json.RawMessage `json:"analysis"`
}

type rawAnalysisConfiguration struct {
	Exclude     []string         `json:"exclude"`
	Include     []rawIncludeRule `json:"include"`
	Assignments []rawAssignment  `json:"assignments"`
}

type rawIncludeRule struct {
	AnalyzerID string   `json:"analyzer_id"`
	Globs      []string `json:"globs"`
}

type rawAssignment struct {
	Path       string          `json:"path"`
	AnalyzerID string          `json:"analyzer_id"`
	Options    json.RawMessage `json:"options"`
}

// LoadNearest loads the nearest complete profile starting at repositoryRoot.
// A malformed nearest file is returned as an error; the search never falls
// through to an ancestor after a file has been found.
func LoadNearest(repositoryRoot string, registry *analysis.Registry) (Configuration, error) {
	return LoadNearestAt(repositoryRoot, repositoryRoot, registry)
}

// LoadNearestAt is LoadNearest with a possibly nested invocation root. Paths
// in assignments remain repository-relative while source globs are anchored
// to invocationRoot.
func LoadNearestAt(repositoryRoot, invocationRoot string, registry *analysis.Registry) (Configuration, error) {
	repository, err := normalizeDirectory(repositoryRoot)
	if err != nil {
		return Configuration{}, err
	}
	invocation, invocationRelative, err := normalizeInvocationRoot(repository, invocationRoot)
	if err != nil {
		return Configuration{}, err
	}
	for directory := invocation; ; directory = filepath.Dir(directory) {
		candidate := filepath.Join(directory, FileName)
		data, readErr := os.ReadFile(candidate)
		if readErr == nil {
			value, decodeErr := DecodeAt(data, repository, invocationRelative, registry)
			if decodeErr != nil {
				return Configuration{}, analysis.WrapHostError(analysis.ErrAnalysisConfigInvalid, "nearest .archview.json is invalid", decodeErr, map[string]any{
					"path": candidate,
				})
			}
			value.Path = candidate
			value.Origin = "ancestor"
			value.repositoryRoot = repository
			value.invocationRoot = invocationRelative
			if samePath(filepath.Dir(candidate), invocation) {
				value.Origin = "project"
			}
			value.Found = true
			return value, nil
		}
		if !os.IsNotExist(readErr) {
			return Configuration{}, analysis.WrapHostError(analysis.ErrAnalysisConfigInvalid, "nearest .archview.json could not be read", readErr, map[string]any{
				"path": candidate,
			})
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			break
		}
	}
	value := Configuration{
		SchemaVersion:  SchemaVersionV1,
		Layout:         layout.DefaultProfile(),
		Origin:         "default",
		Found:          false,
		repositoryRoot: repository,
		invocationRoot: invocationRelative,
	}
	value.Fingerprint = configurationFingerprint(value)
	return value, nil
}

// Decode validates a configuration document without filesystem discovery.
// It is useful for callers that already resolved the nearest file.
func Decode(data []byte, registry *analysis.Registry) (Configuration, error) {
	return DecodeAt(data, "", ".", registry)
}

// DecodeAt validates a document with repository and invocation-root context.
func DecodeAt(data []byte, repositoryRoot, invocationRoot string, registry *analysis.Registry) (Configuration, error) {
	var raw rawConfiguration
	if err := json.Unmarshal(data, &raw); err != nil {
		return Configuration{}, configError("configuration document is not valid JSON", map[string]any{"field": "document"}, err)
	}
	if raw.SchemaVersion != SchemaVersionV1 && raw.SchemaVersion != SchemaVersionV2 {
		return Configuration{}, configError("configuration schema version is unsupported", map[string]any{"field": "schema_version", "schema_version": raw.SchemaVersion}, nil)
	}
	if len(raw.Layout) == 0 || bytes.Equal(bytes.TrimSpace(raw.Layout), []byte("null")) {
		return Configuration{}, configError("configuration requires a layout object", map[string]any{"field": "layout"}, nil)
	}
	var profile layout.LayoutProfile
	if err := decodeStrict(raw.Layout, &profile); err != nil {
		return Configuration{}, configError("layout profile is invalid", map[string]any{"field": "layout"}, err)
	}
	profile, err := layout.ValidateProfile(profile)
	if err != nil {
		return Configuration{}, configError("layout profile is invalid", map[string]any{"field": "layout"}, err)
	}

	value := Configuration{SchemaVersion: raw.SchemaVersion, Layout: profile, repositoryRoot: repositoryRoot, invocationRoot: invocationRoot}
	if raw.SchemaVersion == SchemaVersionV1 {
		if len(raw.Analysis) > 0 && !bytes.Equal(bytes.TrimSpace(raw.Analysis), []byte("null")) {
			return Configuration{}, configError("v1 configuration cannot contain an analysis section", map[string]any{"field": "analysis", "schema_version": raw.SchemaVersion}, nil)
		}
		value.Fingerprint = configurationFingerprint(value)
		return value, nil
	}
	if len(raw.Analysis) == 0 || bytes.Equal(bytes.TrimSpace(raw.Analysis), []byte("null")) {
		return Configuration{}, configError("v2 configuration requires an analysis object", map[string]any{"field": "analysis"}, nil)
	}
	analysisValue, err := decodeAnalysis(raw.Analysis, repositoryRoot, invocationRoot, registry)
	if err != nil {
		return Configuration{}, err
	}
	value.Analysis = &analysisValue
	value.Fingerprint = configurationFingerprint(value)
	return value, nil
}

// SourceScopePolicy converts the normalized configuration filters into the
// planner policy for the requested invocation root.
func (value Configuration) SourceScopePolicy(invocationRoot string) (orchestration.SourceScopePolicy, error) {
	if strings.TrimSpace(invocationRoot) == "" {
		invocationRoot = value.invocationRoot
	}
	if strings.TrimSpace(invocationRoot) == "" {
		invocationRoot = "."
	}
	if filepath.IsAbs(invocationRoot) && value.repositoryRoot != "" {
		relative, err := filepath.Rel(value.repositoryRoot, filepath.Clean(invocationRoot))
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return orchestration.SourceScopePolicy{}, analysis.NewHostError(analysis.ErrAnalysisScopeFilterInvalid, "invocation root must remain inside the repository", map[string]any{"invocation_root": invocationRoot})
		}
		if relative == "" {
			relative = "."
		}
		invocationRoot = filepath.ToSlash(relative)
	}
	policy := orchestration.SourceScopePolicy{InvocationRoot: invocationRoot}
	if value.Analysis != nil {
		policy.Exclude = append([]string(nil), value.Analysis.Exclude...)
		policy.Include = append([]orchestration.AnalyzerIncludeRule(nil), value.Analysis.Include...)
	}
	return orchestration.NormalizeSourceScopePolicy(policy, invocationRoot)
}

// CanonicalJSON returns the deterministic configuration representation used
// for fingerprints and cache provenance.
func (value Configuration) CanonicalJSON() ([]byte, error) {
	canonical := struct {
		SchemaVersion string                 `json:"schema_version"`
		Layout        layout.LayoutProfile   `json:"layout"`
		Analysis      *AnalysisConfiguration `json:"analysis,omitempty"`
	}{value.SchemaVersion, layoutProfileCopy(value.Layout), canonicalAnalysis(value.Analysis)}
	return json.Marshal(canonical)
}

func decodeAnalysis(data []byte, repositoryRoot, invocationRoot string, registry *analysis.Registry) (AnalysisConfiguration, error) {
	var raw rawAnalysisConfiguration
	if err := decodeStrict(data, &raw); err != nil {
		return AnalysisConfiguration{}, configError("analysis configuration is invalid", map[string]any{"field": "analysis"}, err)
	}
	if invocationRoot == "" {
		invocationRoot = "."
	}

	include := make([]orchestration.AnalyzerIncludeRule, 0, len(raw.Include))
	seenInclude := make(map[string]int, len(raw.Include))
	for index, rule := range raw.Include {
		analyzerID := strings.TrimSpace(rule.AnalyzerID)
		if analyzerID == "" || strings.ContainsAny(analyzerID, "\x00\r\n") {
			return AnalysisConfiguration{}, configError("analysis include rule has an invalid analyzer id", map[string]any{
				"field":       fmt.Sprintf("analysis.include[%d].analyzer_id", index),
				"analyzer_id": rule.AnalyzerID,
			}, nil)
		}
		if previous, exists := seenInclude[analyzerID]; exists {
			return AnalysisConfiguration{}, analysis.NewHostError(analysis.ErrAnalysisScopeFilterInvalid, "analysis include rules must contain one rule per analyzer id", map[string]any{
				"field":        fmt.Sprintf("analysis.include[%d]", index),
				"analyzer_id":  analyzerID,
				"duplicate_of": fmt.Sprintf("analysis.include[%d]", previous),
			})
		}
		seenInclude[analyzerID] = index
		if registry != nil {
			if _, exists := registry.Get(analyzerID); !exists {
				return AnalysisConfiguration{}, analysis.NewHostError(analysis.ErrAnalysisScopeFilterInvalid, "analysis include rule targets an unavailable analyzer", map[string]any{
					"field":       fmt.Sprintf("analysis.include[%d].analyzer_id", index),
					"analyzer_id": analyzerID,
				})
			}
		}
		if len(rule.Globs) == 0 {
			return AnalysisConfiguration{}, analysis.NewHostError(analysis.ErrAnalysisScopeFilterInvalid, "analysis include rule requires at least one glob", map[string]any{
				"field":       fmt.Sprintf("analysis.include[%d].globs", index),
				"analyzer_id": analyzerID,
			})
		}
		include = append(include, orchestration.AnalyzerIncludeRule{AnalyzerID: analyzerID, Globs: append([]string(nil), rule.Globs...)})
	}

	assignments := make([]orchestration.AnalyzerAssignment, 0, len(raw.Assignments))
	seenAssignments := make(map[string]int, len(raw.Assignments))
	diagnostics := []analysis.Diagnostic{}
	for index, rawAssignment := range raw.Assignments {
		assignmentPath, pathErr := normalizeAssignmentPath(rawAssignment.Path)
		if pathErr != nil {
			return AnalysisConfiguration{}, analysis.WrapHostError(analysis.ErrAnalysisAssignmentInvalid, "analysis assignment path is invalid", pathErr, map[string]any{
				"field": fmt.Sprintf("analysis.assignments[%d].path", index),
				"path":  rawAssignment.Path,
			})
		}
		if previous, exists := seenAssignments[assignmentPath]; exists {
			return AnalysisConfiguration{}, analysis.NewHostError(analysis.ErrAssignmentDuplicatePath, "analysis assignments contain a duplicate normalized path", map[string]any{
				"field":        fmt.Sprintf("analysis.assignments[%d].path", index),
				"path":         assignmentPath,
				"duplicate_of": fmt.Sprintf("analysis.assignments[%d].path", previous),
			})
		}
		seenAssignments[assignmentPath] = index
		analyzerID := strings.TrimSpace(rawAssignment.AnalyzerID)
		if analyzerID == "" || strings.ContainsAny(analyzerID, "\x00\r\n") {
			return AnalysisConfiguration{}, analysis.NewHostError(analysis.ErrAnalysisAssignmentInvalid, "analysis assignment requires a safe analyzer id", map[string]any{
				"field": fmt.Sprintf("analysis.assignments[%d].analyzer_id", index),
				"path":  assignmentPath,
			})
		}
		options, optionsErr := decodeAssignmentOptions(rawAssignment.Options, index, assignmentPath)
		if optionsErr != nil {
			return AnalysisConfiguration{}, optionsErr
		}
		if registry != nil {
			if analyzerValue, exists := registry.Get(analyzerID); exists {
				manifest := analyzerValue.Manifest()
				if validationErr := validateAssignmentOptions(manifest, options, index, assignmentPath); validationErr != nil {
					return AnalysisConfiguration{}, validationErr
				}
			} else {
				diagnostics = append(diagnostics, analysis.Diagnostic{
					Code:        string(analysis.ErrAssignmentAnalyzerUnavailable),
					Severity:    "error",
					Message:     fmt.Sprintf("assigned analyzer %q is unavailable for this scope", analyzerID),
					Subject:     analyzerID,
					Path:        assignmentPath,
					Recoverable: true,
					Metadata:    map[string]any{"assignment_path": assignmentPath},
				})
			}
		}
		assignments = append(assignments, orchestration.AnalyzerAssignment{ProjectRoot: assignmentPath, AnalyzerID: analyzerID, Options: options})
	}
	sort.SliceStable(assignments, func(i, j int) bool {
		if assignments[i].ProjectRoot != assignments[j].ProjectRoot {
			return assignments[i].ProjectRoot < assignments[j].ProjectRoot
		}
		return assignments[i].AnalyzerID < assignments[j].AnalyzerID
	})

	policy, policyErr := orchestration.NormalizeSourceScopePolicy(orchestration.SourceScopePolicy{
		PolicyVersion:  orchestration.SourceScopePolicyVersion,
		InvocationRoot: invocationRoot,
		Exclude:        raw.Exclude,
		Include:        include,
	}, invocationRoot)
	if policyErr != nil {
		return AnalysisConfiguration{}, policyErr
	}
	value := AnalysisConfiguration{
		Exclude:     append([]string{}, policy.Exclude...),
		Include:     append([]orchestration.AnalyzerIncludeRule{}, policy.Include...),
		Assignments: assignments,
		Diagnostics: diagnostics,
	}
	value.Fingerprint = analysisConfigurationFingerprint(value)
	return value, nil
}

func decodeAssignmentOptions(data json.RawMessage, index int, assignmentPath string) (map[string]any, error) {
	if len(data) == 0 {
		return nil, nil
	}
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil, analysis.NewHostError(analysis.ErrAnalysisAssignmentInvalid, "analysis assignment options must be an object", map[string]any{
			"field": fmt.Sprintf("analysis.assignments[%d].options", index),
			"path":  assignmentPath,
		})
	}
	var options map[string]any
	if err := decodeStrict(data, &options); err != nil || options == nil {
		return nil, analysis.NewHostError(analysis.ErrAnalysisAssignmentInvalid, "analysis assignment options must be an object", map[string]any{
			"field": fmt.Sprintf("analysis.assignments[%d].options", index),
			"path":  assignmentPath,
		})
	}
	return options, nil
}

func validateAssignmentOptions(manifest analysis.Manifest, options map[string]any, index int, assignmentPath string) error {
	if len(options) == 0 {
		return nil
	}
	descriptors := make(map[string]analysis.OptionDescriptor, len(manifest.Options))
	for _, descriptor := range manifest.Options {
		descriptors[descriptor.Name] = descriptor
	}
	keys := make([]string, 0, len(options))
	for key := range options {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		descriptor, exists := descriptors[key]
		if !exists {
			return analysis.NewHostError(analysis.ErrAnalysisAssignmentInvalid, "analysis assignment option is not declared by the analyzer manifest", map[string]any{
				"field":       fmt.Sprintf("analysis.assignments[%d].options.%s", index, key),
				"path":        assignmentPath,
				"analyzer_id": manifest.ID,
				"option":      key,
			})
		}
		if descriptor.Sensitive {
			return analysis.NewHostError(analysis.ErrAnalysisAssignmentInvalid, "sensitive analyzer options cannot be persisted in project configuration", map[string]any{
				"field":       fmt.Sprintf("analysis.assignments[%d].options.%s", index, key),
				"path":        assignmentPath,
				"analyzer_id": manifest.ID,
				"option":      key,
			})
		}
	}
	if _, err := analysis.ResolveOptions(manifest, options, nil); err != nil {
		details := map[string]any{
			"field":       fmt.Sprintf("analysis.assignments[%d].options", index),
			"path":        assignmentPath,
			"analyzer_id": manifest.ID,
		}
		var hostErr *analysis.HostError
		if errorsAsHostError(err, &hostErr) && hostErr.Details != nil {
			if option, ok := hostErr.Details["option"].(string); ok {
				details["option"] = option
			}
		}
		return analysis.NewHostError(analysis.ErrAnalysisAssignmentInvalid, "analysis assignment option value is invalid", details)
	}
	return nil
}

func normalizeAssignmentPath(value string) (string, error) {
	if value == "" || strings.TrimSpace(value) != value {
		return "", fmt.Errorf("path must be non-empty and must not contain surrounding whitespace")
	}
	if strings.ContainsAny(value, "\\\x00\r\n") || filepath.IsAbs(value) || strings.HasPrefix(value, "/") || windowsAbsolute(value) {
		return "", fmt.Errorf("path must be a repository-relative POSIX path")
	}
	if strings.ContainsAny(value, "*?[]{}") {
		return "", fmt.Errorf("path cannot contain glob characters")
	}
	value = strings.TrimPrefix(value, "./")
	if value == "" || value == "." {
		return ".", nil
	}
	parts := strings.Split(value, "/")
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		switch part {
		case "":
			return "", fmt.Errorf("path cannot contain empty segments")
		case "..":
			return "", fmt.Errorf("path cannot contain parent traversal")
		case ".":
			continue
		default:
			clean = append(clean, part)
		}
	}
	if len(clean) == 0 {
		return ".", nil
	}
	return path.Join(clean...), nil
}

func normalizeDirectory(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", analysis.NewHostError(analysis.ErrUnreadableProject, "repository root is required", nil)
	}
	absolute, err := filepath.Abs(value)
	if err != nil {
		return "", analysis.WrapHostError(analysis.ErrUnreadableProject, "repository root could not be normalized", err, nil)
	}
	absolute = filepath.Clean(absolute)
	info, err := os.Stat(absolute)
	if err != nil {
		return "", analysis.WrapHostError(analysis.ErrUnreadableProject, "repository root could not be read", err, map[string]any{"project_root": absolute})
	}
	if !info.IsDir() {
		return "", analysis.NewHostError(analysis.ErrUnreadableProject, "repository root must be a directory", map[string]any{"project_root": absolute})
	}
	return absolute, nil
}

func normalizeInvocationRoot(repositoryRoot, value string) (string, string, error) {
	if strings.TrimSpace(value) == "" {
		return repositoryRoot, ".", nil
	}
	absolute := value
	if !filepath.IsAbs(absolute) {
		absolute = filepath.Join(repositoryRoot, filepath.FromSlash(value))
	}
	absolute, err := filepath.Abs(absolute)
	if err != nil {
		return "", "", analysis.WrapHostError(analysis.ErrAnalysisScopeFilterInvalid, "invocation root could not be normalized", err, nil)
	}
	absolute = filepath.Clean(absolute)
	relative, err := filepath.Rel(repositoryRoot, absolute)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return "", "", analysis.NewHostError(analysis.ErrAnalysisScopeFilterInvalid, "invocation root must remain inside the repository", map[string]any{"invocation_root": value})
	}
	info, err := os.Stat(absolute)
	if err != nil || !info.IsDir() {
		return "", "", analysis.WrapHostError(analysis.ErrAnalysisScopeFilterInvalid, "invocation root could not be read", err, map[string]any{"invocation_root": value})
	}
	if relative == "" {
		relative = "."
	}
	return absolute, filepath.ToSlash(relative), nil
}

func decodeStrict(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return fmt.Errorf("more than one JSON value")
	} else if !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func configError(message string, details map[string]any, cause error) error {
	if cause == nil {
		return analysis.NewHostError(analysis.ErrAnalysisConfigInvalid, message, details)
	}
	return analysis.WrapHostError(analysis.ErrAnalysisConfigInvalid, message, cause, details)
}

func configurationFingerprint(value Configuration) string {
	data, err := value.CanonicalJSON()
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func analysisConfigurationFingerprint(value AnalysisConfiguration) string {
	data, err := json.Marshal(canonicalAnalysis(&value))
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func canonicalAnalysis(value *AnalysisConfiguration) *AnalysisConfiguration {
	if value == nil {
		return nil
	}
	copyValue := *value
	copyValue.Exclude = append([]string(nil), value.Exclude...)
	copyValue.Include = append([]orchestration.AnalyzerIncludeRule(nil), value.Include...)
	for index := range copyValue.Include {
		copyValue.Include[index].Globs = append([]string(nil), value.Include[index].Globs...)
	}
	copyValue.Assignments = append([]orchestration.AnalyzerAssignment(nil), value.Assignments...)
	for index := range copyValue.Assignments {
		copyValue.Assignments[index].Options = cloneMap(value.Assignments[index].Options)
	}
	copyValue.Diagnostics = nil
	sort.SliceStable(copyValue.Exclude, func(i, j int) bool { return copyValue.Exclude[i] < copyValue.Exclude[j] })
	sort.SliceStable(copyValue.Include, func(i, j int) bool { return copyValue.Include[i].AnalyzerID < copyValue.Include[j].AnalyzerID })
	for index := range copyValue.Include {
		sort.Strings(copyValue.Include[index].Globs)
	}
	sort.SliceStable(copyValue.Assignments, func(i, j int) bool {
		return copyValue.Assignments[i].ProjectRoot < copyValue.Assignments[j].ProjectRoot
	})
	return &copyValue
}

func layoutProfileCopy(value layout.LayoutProfile) layout.LayoutProfile {
	copyValue := value
	copyValue.Options = cloneMap(value.Options)
	return copyValue
}

func cloneMap(value map[string]any) map[string]any {
	if value == nil {
		return nil
	}
	result := make(map[string]any, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}

func errorsAsHostError(err error, target **analysis.HostError) bool {
	return errors.As(err, target)
}

func windowsAbsolute(value string) bool {
	return len(value) >= 2 && ((value[0] >= 'a' && value[0] <= 'z') || (value[0] >= 'A' && value[0] <= 'Z')) && value[1] == ':'
}

func samePath(left, right string) bool {
	return filepath.Clean(left) == filepath.Clean(right)
}
