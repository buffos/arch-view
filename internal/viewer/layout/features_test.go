package layout

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
)

func TestFeatureMetadataAndStagedAvailability(t *testing.T) {
	features := FeatureCatalog()
	if len(features) != 5 {
		t.Fatal(features)
	}
	owned := map[string]bool{}
	for _, feature := range features {
		if feature.Stage < 2 || len(feature.Algorithms) == 0 || len(feature.Surfaces) != 3 || feature.Fallback == "" {
			t.Fatal(feature)
		}
		for _, field := range feature.Owns {
			if owned[field] {
				t.Fatal("ownership conflict", field)
			}
			owned[field] = true
		}
		profile := LayoutProfile{Algorithm: "layered", Features: []string{feature.ID}, Options: map[string]any{}}
		if feature.ID == "spline_refinement" {
			profile.Options["org.eclipse.elk.edgeRouting"] = "SPLINES"
		}
		_, err := ValidateProfile(profile)
		if feature.Stage <= 4 && err != nil {
			t.Fatalf("supported stage %d feature %s: %v", feature.Stage, feature.ID, err)
		}
		if feature.Stage > 4 && analysis.ErrorCodeOf(err) != "renderer_feature_unavailable" {
			t.Fatalf("future feature %s: %v", feature.ID, err)
		}
	}
	_, err := ValidateProfile(LayoutProfile{Algorithm: "layered", Features: []string{"unknown"}})
	if analysis.ErrorCodeOf(err) != "renderer_feature_unknown" {
		t.Fatal(err)
	}
}

func TestSavedFeaturesRemainPreferences(t *testing.T) {
	root := t.TempDir()
	data := []byte(`{"schema_version":"arch-view.config/v2","analysis":{},"layout":{"algorithm":"layered","features":["compound","edge_labels"],"options":{}},"extension":{"keep":true}}`)
	path := filepath.Join(root, ConfigFileName)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	session := NewSession(root)
	response := session.Response()
	if response.Status != "valid" || !reflect.DeepEqual(response.Layout.Features, []string{"compound", "edge_labels"}) || len(response.Diagnostics) != 0 {
		t.Fatalf("%+v", response)
	}
	response.Layout.Features[0] = "mutated"
	if session.Response().Layout.Features[0] != "compound" {
		t.Fatal("mutable snapshot")
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(data) {
		t.Fatal("read rewrote preferences")
	}
}

func TestPresentationPortSettingsRequireFeatureAndFixedSides(t *testing.T) {
	profile := LayoutProfile{Algorithm: "layered", Features: []string{"ports"}, Options: map[string]any{
		"org.eclipse.elk.portConstraints": "FIXED_SIDE",
	}}
	if _, err := ValidateProfile(profile); err != nil {
		t.Fatal(err)
	}
	profile.Features = nil
	if _, err := ValidateProfile(profile); analysis.ErrorCodeOf(err) != "renderer_feature_dependency" {
		t.Fatal(err)
	}
	profile.Features = []string{"ports"}
	profile.Options["org.eclipse.elk.portConstraints"] = "FREE"
	if _, err := ValidateProfile(profile); err == nil {
		t.Fatal("unverified port constraint accepted")
	}
}

func TestCompoundSettingsRequireFeatureAndSingleRunHierarchy(t *testing.T) {
	profile := LayoutProfile{Algorithm: "layered", Features: []string{"compound"}, Options: map[string]any{
		"org.eclipse.elk.hierarchyHandling": "INCLUDE_CHILDREN",
	}}
	if _, err := ValidateProfile(profile); err != nil {
		t.Fatal(err)
	}
	profile.Features = nil
	if _, err := ValidateProfile(profile); analysis.ErrorCodeOf(err) != "renderer_feature_dependency" {
		t.Fatal(err)
	}
	profile.Features = []string{"compound"}
	profile.Options["org.eclipse.elk.hierarchyHandling"] = "SEPARATE_CHILDREN"
	if _, err := ValidateProfile(profile); err == nil {
		t.Fatal("unverified hierarchy mode accepted")
	}
}

func TestAdvancedEdgeSettingsRequireTheirFeatures(t *testing.T) {
	profile := LayoutProfile{Algorithm: "layered", Features: []string{"edge_labels", "spline_refinement"}, Options: map[string]any{
		"org.eclipse.elk.edgeRouting":                      "SPLINES",
		"org.eclipse.elk.edgeLabels.placement":             "TAIL",
		"org.eclipse.elk.layered.edgeRouting.splines.mode": "SLOPPY",
	}}
	if _, err := ValidateProfile(profile); err != nil {
		t.Fatal(err)
	}
	profile.Features = []string{"edge_labels"}
	if _, err := ValidateProfile(profile); err == nil {
		t.Fatal("spline setting accepted without its renderer feature")
	}
}

func TestUsefulSettingsValidateAndRoundTrip(t *testing.T) {
	padding := map[string]any{"top": 10.0, "right": 20.0, "bottom": 30.0, "left": 40.0}
	for _, algorithm := range []string{"mrtree", "layered"} {
		profile := LayoutProfile{Algorithm: algorithm, Options: map[string]any{"org.eclipse.elk.padding": padding}}
		data, err := EncodeConfig(profile)
		if err != nil {
			t.Fatal(err)
		}
		restored, err := DecodeConfig(data)
		if err != nil || !reflect.DeepEqual(profile, restored) {
			t.Fatalf("%+v %v", restored, err)
		}
	}
	for _, value := range []any{"10", map[string]any{"top": 1}, map[string]any{"top": -1, "right": 0, "bottom": 0, "left": 0}} {
		_, err := ValidateProfile(LayoutProfile{Algorithm: "mrtree", Options: map[string]any{"org.eclipse.elk.padding": value}})
		if err == nil {
			t.Fatal("accepted", value)
		}
	}
	for key, value := range map[string]any{"org.eclipse.elk.mrtree.edgeRoutingMode": "NONE", "org.eclipse.elk.mrtree.searchOrder": "BFS", "org.eclipse.elk.mrtree.weighting": "CONSTRAINT"} {
		_, err := ValidateProfile(LayoutProfile{Algorithm: "mrtree", Options: map[string]any{key: value}})
		if err == nil {
			t.Fatal("unsafe/unimplemented value accepted", key)
		}
	}
}

func TestLayoutSavePreservesOtherSections(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ConfigFileName)
	if err := os.WriteFile(path, []byte(`{"schema_version":"arch-view.config/v2","analysis":{},"layout":{"algorithm":"layered","options":{}},"okf":{"profiles":[{"layout":{"algorithm":"mrtree"}}]},"custom":[1,2]}`), 0600); err != nil {
		t.Fatal(err)
	}
	session := NewSession(root)
	profile := LayoutProfile{Algorithm: "mrtree", Options: map[string]any{"org.eclipse.elk.mrtree.weighting": "FAN"}}
	if err := session.Apply(profile); err != nil {
		t.Fatal(err)
	}
	if NewSession(root).Response().Layout.Algorithm != "layered" {
		t.Fatal("Apply persisted")
	}
	if err := session.SaveActive(profile); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(NewSession(root).Response().Layout, profile) {
		t.Fatal("save/reload mismatch")
	}
	data, _ := os.ReadFile(path)
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(data, &raw)
	if len(raw["okf"]) == 0 || len(raw["custom"]) == 0 {
		t.Fatal("unrelated settings lost")
	}
}
