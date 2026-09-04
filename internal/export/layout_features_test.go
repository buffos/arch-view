package export

import (
	"bytes"
	"github.com/buffo/arch-view/internal/viewer/layout"
	"testing"
)

func TestEmbeddedHTMLRetainsUnavailableFeaturePreferences(t *testing.T) {
	profile := layout.LayoutProfile{Algorithm: "layered", Features: []string{"ports"}, Options: map[string]any{"org.eclipse.elk.portConstraints": "FIXED_SIDE"}}
	_, data, err := Render(exportFixtureModel(t), Request{Format: FormatHTML, LayoutProfile: &profile})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`"features":["ports"]`)) || !bytes.Contains(data, []byte("renderer_feature_unavailable")) {
		t.Fatal("missing saved preference or shared diagnostic")
	}
}

func TestStaticSVGReportsAdvancedBrowserFeaturesNotApplied(t *testing.T) {
	profile := layout.LayoutProfile{Algorithm: "layered", Features: []string{"edge_labels"}}
	metadata, data, err := Render(exportFixtureModel(t), Request{Format: FormatSVG, LayoutProfile: &profile})
	if err != nil {
		t.Fatal(err)
	}
	if metadata.LayoutProvenance["diagnostic"] != "advanced_features_not_applied" {
		t.Fatalf("provenance = %#v", metadata.LayoutProvenance)
	}
	if bytes.Contains(data, []byte("geometry-edge-label")) {
		t.Fatal("static SVG used browser-only advanced geometry")
	}
}
