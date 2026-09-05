package layout

import (
	_ "embed"
	"encoding/json"
	"reflect"
	"slices"
	"sort"

	"github.com/buffo/arch-view/internal/analysis"
)

// FeatureDefinition is shared declarative policy, not an executable plugin.
// Browser handlers register against these IDs; stages publish support only
// after their implementation and verification are complete.
type FeatureDefinition struct {
	OptionIDs       []string       `json:"option_ids"`
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Status          string         `json:"status"`
	Order           int            `json:"order"`
	Stage           int            `json:"stage"`
	Algorithms      []string       `json:"algorithms"`
	Surfaces        []string       `json:"surfaces"`
	Prerequisites   []string       `json:"prerequisites"`
	Owns            []string       `json:"owns"`
	RequiredOptions map[string]any `json:"required_options"`
	Fallback        string         `json:"fallback"`
}

//go:embed features.json
var featureMetadata []byte

func FeatureCatalog() []FeatureDefinition {
	var result []FeatureDefinition
	if err := json.Unmarshal(featureMetadata, &result); err != nil {
		panic("invalid compiled layout feature metadata: " + err.Error())
	}
	return result
}

func validateFeatures(profile LayoutProfile, allowUnavailable bool) ([]string, error) {
	if profile.Features == nil {
		return nil, nil
	}
	features := append([]string{}, profile.Features...)
	sort.Strings(features)
	seen := make(map[string]bool)
	catalog := make(map[string]FeatureDefinition)
	for _, feature := range FeatureCatalog() {
		catalog[feature.ID] = feature
	}
	for _, id := range features {
		feature, ok := catalog[id]
		if !ok {
			return nil, featureError("renderer_feature_unknown", id, "Unknown renderer feature.")
		}
		if seen[id] {
			return nil, featureError("renderer_feature_duplicate", id, "Duplicate renderer feature.")
		}
		seen[id] = true
		if allowUnavailable && feature.Status != "supported" {
			continue
		}
		if feature.Status != "supported" {
			return nil, featureError("renderer_feature_unavailable", id, "This renderer feature is not implemented in this build.")
		}
		if !slices.Contains(feature.Algorithms, profile.Algorithm) {
			return nil, featureError("renderer_feature_incompatible", id, "Renderer feature does not support the selected algorithm.")
		}
		for key, expected := range feature.RequiredOptions {
			if !reflect.DeepEqual(profile.Options[key], expected) {
				return nil, featureError("renderer_feature_incompatible", id, "Required layout option is not selected: "+key)
			}
		}
		for _, dependency := range feature.Prerequisites {
			if !slices.Contains(features, dependency) {
				return nil, featureError("renderer_feature_dependency", id, "Missing renderer feature prerequisite: "+dependency)
			}
		}
	}
	return features, nil
}

func featureError(code analysis.ErrorCode, id, message string) error {
	return analysis.NewHostError(code, message, map[string]any{"feature": id})
}

func unavailableFeatureDiagnostics(profile LayoutProfile) []LayoutDiagnostic {
	var diagnostics []LayoutDiagnostic
	for _, feature := range FeatureCatalog() {
		if slices.Contains(profile.Features, feature.ID) && feature.Status != "supported" {
			diagnostics = append(diagnostics, LayoutDiagnostic{
				Code: "renderer_feature_unavailable", Severity: "warning",
				Message: feature.Name + " is saved but unavailable in this build; ordinary geometry is used without changing saved preferences.",
			})
		}
	}
	return diagnostics
}

func unavailableFeatureOwnsOption(profile LayoutProfile, id string) bool {
	for _, feature := range FeatureCatalog() {
		if feature.Status != "supported" && slices.Contains(profile.Features, feature.ID) && slices.Contains(feature.OptionIDs, id) {
			return true
		}
	}
	return false
}
