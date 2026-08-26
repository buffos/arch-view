// Package export renders validated architecture models into deterministic
// machine-readable and visual artifacts.
package export

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/viewer/scene"
)

const (
	FormatJSON = "json"
	FormatHTML = "html"
	FormatSVG  = "svg"
)

var visualFormats = []string{FormatHTML, FormatSVG}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

// Request describes one artifact render or write. Context is checked between
// the major phases; cancellation before the final commit prevents a completed
// artifact, while a commit already in progress is allowed to finish.
type Request struct {
	Format              string
	OutputPath          string
	ViewPath            []string
	ReferenceVisibility string
	ReferenceScopes     []string
	Overwrite           bool
	EmbedSource         bool
	Context             context.Context
}

// ArtifactMetadata is the stable status returned after an artifact is
// rendered and, for Write, atomically committed to its target path.
type ArtifactMetadata struct {
	Format            string            `json:"format"`
	SchemaVersion     string            `json:"schema_version,omitempty"`
	ModelID           string            `json:"model_id"`
	ModelRevision     string            `json:"model_revision,omitempty"`
	Status            model.Status      `json:"status"`
	ContentHash       string            `json:"content_hash"`
	LayoutProvenance  map[string]any    `json:"layout_provenance,omitempty"`
	DiagnosticSummary DiagnosticSummary `json:"diagnostic_summary"`
	Bytes             int               `json:"bytes"`
}

type DiagnosticSummary struct {
	Total    int `json:"total"`
	Info     int `json:"info"`
	Warnings int `json:"warnings"`
	Errors   int `json:"errors"`
}

type htmlBundle struct {
	Model                      model.Model                    `json:"model"`
	Scenes                     map[string]scene.SceneSnapshot `json:"scenes"`
	Layouts                    map[string]deterministicLayout `json:"layouts"`
	InitialPath                []string                       `json:"initial_path"`
	InitialReferenceVisibility string                         `json:"initial_reference_visibility"`
}

// Render validates the model and returns deterministic artifact bytes without
// touching the filesystem.
func Render(value model.Model, request Request) (ArtifactMetadata, []byte, error) {
	request, err := normalizeRequest(request)
	if err != nil {
		return ArtifactMetadata{}, nil, err
	}
	if err := request.Context.Err(); err != nil {
		return ArtifactMetadata{}, nil, err
	}
	if err := model.Validate(value); err != nil {
		return ArtifactMetadata{}, nil, err
	}
	if value.Status == model.StatusFailed {
		return ArtifactMetadata{}, nil, analysis.NewHostError(analysis.ErrInvalidModel, "failed models cannot be exported", map[string]any{"status": value.Status})
	}

	var (
		data       []byte
		provenance map[string]any
	)
	switch request.Format {
	case FormatJSON:
		data, err = canonicalJSON(value)
	case FormatHTML:
		data, provenance, err = renderHTML(value, request)
	case FormatSVG:
		data, provenance, err = renderSVG(value, request)
	}
	if err != nil {
		return ArtifactMetadata{}, nil, err
	}
	if err := request.Context.Err(); err != nil {
		return ArtifactMetadata{}, nil, err
	}
	hash := sha256.Sum256(data)
	return ArtifactMetadata{
		Format:            request.Format,
		SchemaVersion:     value.SchemaVersion,
		ModelID:           value.ModelID,
		ModelRevision:     value.ModelID,
		Status:            value.Status,
		ContentHash:       hex.EncodeToString(hash[:]),
		LayoutProvenance:  provenance,
		DiagnosticSummary: summarizeDiagnostics(value),
		Bytes:             len(data),
	}, data, nil
}

// Write renders and atomically writes one artifact. Existing targets are
// protected unless Overwrite is explicitly set.
func Write(value model.Model, request Request) (ArtifactMetadata, error) {
	request, err := normalizeRequest(request)
	if err != nil {
		return ArtifactMetadata{}, err
	}
	if strings.TrimSpace(request.OutputPath) == "" || request.OutputPath == "-" {
		return ArtifactMetadata{}, analysis.NewHostError(analysis.ErrInvalidRequest, "export requires a file output path", map[string]any{"output": request.OutputPath})
	}
	metadata, data, err := Render(value, request)
	if err != nil {
		return ArtifactMetadata{}, err
	}
	if err := writeAtomically(request.OutputPath, data, request.Overwrite, request.Context); err != nil {
		return ArtifactMetadata{}, err
	}
	return metadata, nil
}

func normalizeRequest(request Request) (Request, error) {
	request.Format = strings.ToLower(strings.TrimSpace(request.Format))
	if !isSupportedFormat(request.Format) {
		return Request{}, analysis.NewHostError(analysis.ErrUnsupportedOption, "export format is unsupported", map[string]any{"format": request.Format, "supported": []string{FormatJSON, FormatHTML, FormatSVG}})
	}
	if request.Context == nil {
		request.Context = context.Background()
	}
	if request.ReferenceVisibility == "" {
		request.ReferenceVisibility = scene.ReferenceVisibilityHidden
	}
	if !validReferenceVisibility(request.ReferenceVisibility) {
		return Request{}, analysis.NewHostError(analysis.ErrInvalidRequest, "export reference visibility is unsupported", map[string]any{"reference_visibility": request.ReferenceVisibility})
	}
	request.ViewPath = append([]string{}, request.ViewPath...)
	for _, segment := range request.ViewPath {
		if strings.TrimSpace(segment) == "" {
			return Request{}, analysis.NewHostError(analysis.ErrInvalidRequest, "export view path contains an empty segment", nil)
		}
	}
	request.ReferenceScopes = sortedUnique(request.ReferenceScopes)
	for _, scope := range request.ReferenceScopes {
		if !validReferenceScope(scope) {
			return Request{}, analysis.NewHostError(analysis.ErrInvalidRequest, "export reference scope is unsupported", map[string]any{"reference_scope": scope})
		}
	}
	if request.EmbedSource {
		return Request{}, analysis.NewHostError(analysis.ErrUnsupportedOption, "source embedding is unsupported in v1", map[string]any{"embed_source": true})
	}
	return request, nil
}

func isSupportedFormat(value string) bool {
	return value == FormatJSON || value == FormatHTML || value == FormatSVG
}

func validReferenceVisibility(value string) bool {
	return value == scene.ReferenceVisibilityHidden || value == scene.ReferenceVisibilityAggregated || value == scene.ReferenceVisibilityExpanded
}

func validReferenceScope(value string) bool {
	switch value {
	case "standard_library", "external", "unresolved", "dynamic":
		return true
	default:
		return false
	}
}

func canonicalJSON(value model.Model) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, analysis.WrapHostError(analysis.ErrInvalidModel, "model could not be serialized for export", err, nil)
	}
	return append(data, '\n'), nil
}

func summarizeDiagnostics(value model.Model) DiagnosticSummary {
	result := DiagnosticSummary{Total: len(value.Diagnostics)}
	for _, diagnostic := range value.Diagnostics {
		switch diagnostic.Severity {
		case "info":
			result.Info++
		case "warning":
			result.Warnings++
		case "error":
			result.Errors++
		}
	}
	return result
}

func sceneCatalog(value model.Model, request Request) (htmlBundle, map[string]any, error) {
	paths := hierarchyPaths(value)
	pathSet := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		pathSet[pathKey(path)] = struct{}{}
	}
	if _, exists := pathSet[pathKey(request.ViewPath)]; !exists {
		return htmlBundle{}, nil, analysis.NewHostError(analysis.ErrInvalidRequest, "export view path does not resolve", map[string]any{"view_path": request.ViewPath})
	}

	scenes := make(map[string]scene.SceneSnapshot, len(paths)*len(visualFormats)+len(paths))
	layouts := make(map[string]deterministicLayout, len(scenes))
	var provenance map[string]any
	for _, path := range paths {
		for _, visibility := range []string{scene.ReferenceVisibilityHidden, scene.ReferenceVisibilityAggregated, scene.ReferenceVisibilityExpanded} {
			if err := request.Context.Err(); err != nil {
				return htmlBundle{}, nil, err
			}
			snapshot, err := scene.BuildSceneWithOptions(value, path, "overview", scene.SceneOptions{
				ReferenceVisibility: visibility,
				ReferenceScopes:     request.ReferenceScopes,
			})
			if err != nil {
				return htmlBundle{}, nil, err
			}
			key := sceneCatalogKey(path, visibility)
			scenes[key] = snapshot
			layout := buildDeterministicLayout(snapshot)
			layouts[sceneLayoutKey(snapshot)] = layout
			if provenance == nil {
				provenance = layoutProvenance()
			}
		}
	}
	return htmlBundle{
		Model:                      value,
		Scenes:                     scenes,
		Layouts:                    layouts,
		InitialPath:                append([]string{}, request.ViewPath...),
		InitialReferenceVisibility: request.ReferenceVisibility,
	}, provenance, nil
}

func hierarchyPaths(value model.Model) [][]string {
	paths := [][]string{[]string{}}
	seen := map[string]struct{}{pathKey([]string{}): {}}
	for _, module := range value.Modules {
		for length := 1; length <= len(module.Hierarchy); length++ {
			path := append([]string{}, module.Hierarchy[:length]...)
			key := pathKey(path)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			paths = append(paths, path)
		}
	}
	sort.Slice(paths, func(left, right int) bool { return pathKey(paths[left]) < pathKey(paths[right]) })
	return paths
}

func pathKey(value []string) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func sceneCatalogKey(path []string, visibility string) string {
	return pathKey(path) + "|" + visibility
}

func sceneLayoutKey(scene scene.SceneSnapshot) string {
	nodes := make([]string, 0, len(scene.VisibleNodes))
	for _, node := range scene.VisibleNodes {
		nodes = append(nodes, node.ID)
	}
	sort.Strings(nodes)
	relationships := make([]string, 0, len(scene.VisibleRelationships))
	for _, relationship := range scene.VisibleRelationships {
		relationships = append(relationships, relationship.ID+":"+relationship.FromVisibleID+":"+relationship.ToVisibleID)
	}
	sort.Strings(relationships)
	return strings.Join([]string{
		nonEmpty(scene.ModelRevision, scene.ModelID),
		strings.Join(scene.HierarchyPath, "/"),
		scene.ReferenceVisibility,
		strings.Join(nodes, ","),
		strings.Join(relationships, ","),
	}, "|")
}

func nonEmpty(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}

func sortedUnique(values []string) []string {
	result := append([]string{}, values...)
	sort.Strings(result)
	if len(result) < 2 {
		return result
	}
	write := 1
	for _, value := range result[1:] {
		if value != result[write-1] {
			result[write] = value
			write++
		}
	}
	return result[:write]
}

func layoutProvenance() map[string]any {
	return map[string]any{
		"engine":                  "deterministic-export",
		"algorithm":               "layered-orthogonal-v1",
		"algorithm_version":       "1",
		"direction":               "right",
		"node_width":              deterministicNodeWidth,
		"node_height":             deterministicNodeHeight,
		"column_gap":              deterministicColumnGap,
		"row_gap":                 deterministicRowGap,
		"reference_layout_policy": "scene-reference-visibility",
	}
}
