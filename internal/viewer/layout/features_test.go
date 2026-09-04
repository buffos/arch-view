package layout

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
)

func TestFeatureMetadataAndStageOneAvailability(t *testing.T) {
	features := FeatureCatalog()
	if len(features) != 5 {
		t.Fatal(features)
	}
	owned := map[string]bool{}
	for _, feature := range features {
		if feature.Status != "not-implemented" || feature.Stage < 2 || len(feature.Algorithms) == 0 || len(feature.Surfaces) != 3 || feature.Fallback == "" {
			t.Fatal(feature)
		}
		for _, field := range feature.Owns {
			if owned[field] {
				t.Fatal("ownership conflict", field)
			}
			owned[field] = true
		}
		_, err := ValidateProfile(LayoutProfile{Algorithm: "layered", Features: []string{feature.ID}})
		if analysis.ErrorCodeOf(err) != "renderer_feature_unavailable" {
			t.Fatal(err)
		}
	}
	_, err := ValidateProfile(LayoutProfile{Algorithm: "layered", Features: []string{"unknown"}})
	if analysis.ErrorCodeOf(err) != "renderer_feature_unknown" {
		t.Fatal(err)
	}
}

func TestSavedUnavailableFeaturesRemainPreferences(t *testing.T) {
	root := t.TempDir()
	data := []byte(`{"schema_version":"arch-view.config/v2","analysis":{},"layout":{"algorithm":"layered","features":["ports","edge_labels"],"options":{"org.eclipse.elk.portConstraints":"FIXED_SIDE"}},"extension":{"keep":true}}`)
	path := filepath.Join(root, ConfigFileName)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	session := NewSession(root)
	response := session.Response()
	if response.Status != "valid" || !reflect.DeepEqual(response.Layout.Features, []string{"edge_labels", "ports"}) || len(response.Diagnostics) != 2 {
		t.Fatalf("%+v", response)
	}
	response.Layout.Features[0] = "mutated"
	if session.Response().Layout.Features[0] != "edge_labels" {
		t.Fatal("mutable snapshot")
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(data) {
		t.Fatal("read rewrote preferences")
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
