package viewer

import (
	"encoding/json"
	"github.com/buffo/arch-view/internal/viewer/layout"
	"net/http"
	"testing"
)

func TestLayoutFeatureHTTPValidation(t *testing.T) {
	server, err := NewServer(fixtureModel(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, feature := range []string{"unknown", "compound"} {
		payload, _ := json.Marshal(layoutApplyRequest{SchemaVersion: "arch-view.config/v2", Layout: layout.LayoutProfile{Algorithm: "layered", Features: []string{feature}}})
		response := requestLayout(t, server, http.MethodPost, "/v1/layout/apply", payload)
		if response.Code != http.StatusUnprocessableEntity {
			t.Fatalf("%d %s", response.Code, response.Body.String())
		}
	}
	portPayload, _ := json.Marshal(layoutApplyRequest{SchemaVersion: "arch-view.config/v2", Layout: layout.LayoutProfile{Algorithm: "layered", Features: []string{"ports"}}})
	if response := requestLayout(t, server, http.MethodPost, "/v1/layout/apply", portPayload); response.Code != http.StatusOK {
		t.Fatalf("%d %s", response.Code, response.Body.String())
	}
	payload, _ := json.Marshal(layoutApplyRequest{SchemaVersion: "arch-view.config/v2", Layout: layout.LayoutProfile{Algorithm: "mrtree", Options: map[string]any{"org.eclipse.elk.mrtree.weighting": "FAN"}}})
	response := requestLayout(t, server, http.MethodPost, "/v1/layout/apply", payload)
	if response.Code != http.StatusOK {
		t.Fatalf("%d %s", response.Code, response.Body.String())
	}
}
