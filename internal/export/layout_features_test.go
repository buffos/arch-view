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
