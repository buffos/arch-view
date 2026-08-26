package viewer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
)

func TestLayoutEndpointsExposeCatalogAndModelOnlySessionLimits(t *testing.T) {
	server, err := NewServer(fixtureModel(t))
	if err != nil {
		t.Fatal(err)
	}
	optionsResponse := requestLayout(t, server, http.MethodGet, "/v1/layout/options", nil)
	if optionsResponse.Code != http.StatusOK {
		t.Fatalf("GET options status = %d, body=%s", optionsResponse.Code, optionsResponse.Body.String())
	}
	var options LayoutOptionsResponse
	if err := json.Unmarshal(optionsResponse.Body.Bytes(), &options); err != nil {
		t.Fatal(err)
	}
	if len(options.Options) != 235 || len(options.Algorithms) != 11 {
		t.Fatalf("catalog sizes = %d options, %d algorithms", len(options.Options), len(options.Algorithms))
	}
	configResponse := requestLayout(t, server, http.MethodGet, "/v1/layout/config", nil)
	var config LayoutConfigResponse
	if err := json.Unmarshal(configResponse.Body.Bytes(), &config); err != nil {
		t.Fatal(err)
	}
	if config.Origin != "session" || config.CanSave || config.CanSaveAs || len(config.Diagnostics) != 1 {
		t.Fatalf("model-only config = %#v", config)
	}
	payload := layoutJSON(t, layoutApplyRequest{SchemaVersion: layoutConfigSchemaVersion, Layout: LayoutProfile{Algorithm: "layered", Options: map[string]any{"org.eclipse.elk.direction": "LEFT"}}})
	applyResponse := requestLayout(t, server, http.MethodPost, "/v1/layout/apply", payload)
	if applyResponse.Code != http.StatusOK {
		t.Fatalf("POST apply status = %d, body=%s", applyResponse.Code, applyResponse.Body.String())
	}
	if response := requestLayout(t, server, http.MethodPut, "/v1/layout/config", payload); response.Code != http.StatusConflict || analysis.ErrorCodeOf(decodeHTTPError(t, response.Body.Bytes())) != analysis.ErrSaveAsRequired {
		t.Fatalf("model-only Save response = %d, body=%s", response.Code, response.Body.String())
	}
	saveAsPayload := layoutJSON(t, layoutSaveAsRequest{SchemaVersion: layoutConfigSchemaVersion, Layout: LayoutProfile{Algorithm: "layered", Options: map[string]any{}}, DestinationDir: t.TempDir(), Confirm: true})
	if response := requestLayout(t, server, http.MethodPut, "/v1/layout/config/save-as", saveAsPayload); response.Code != http.StatusForbidden {
		t.Fatalf("model-only Save As status = %d, body=%s", response.Code, response.Body.String())
	}
}

func TestLayoutSaveAndSaveAsUseOnlyTheSelectedDestination(t *testing.T) {
	root := t.TempDir()
	custom := t.TempDir()
	server, err := NewServer(fixtureModel(t), ServerOptions{SourceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	profile := LayoutProfile{Algorithm: "layered", Options: map[string]any{"org.eclipse.elk.direction": "LEFT"}}
	payload := layoutJSON(t, layoutApplyRequest{SchemaVersion: layoutConfigSchemaVersion, Layout: profile})
	if response := requestLayout(t, server, http.MethodPut, "/v1/layout/config", payload); response.Code != http.StatusConflict {
		t.Fatalf("Save without active file status = %d, body=%s", response.Code, response.Body.String())
	}
	saveAsPayload := layoutJSON(t, layoutSaveAsRequest{SchemaVersion: layoutConfigSchemaVersion, Layout: profile, DestinationDir: custom, Confirm: true})
	saveAsResponse := requestLayout(t, server, http.MethodPut, "/v1/layout/config/save-as", saveAsPayload)
	if saveAsResponse.Code != http.StatusOK {
		t.Fatalf("Save As status = %d, body=%s", saveAsResponse.Code, saveAsResponse.Body.String())
	}
	customPath := filepath.Join(custom, layoutConfigFileName)
	if _, err := os.Stat(customPath); err != nil {
		t.Fatalf("Save As did not create selected file: %v", err)
	}
	var config LayoutConfigResponse
	if err := json.Unmarshal(saveAsResponse.Body.Bytes(), &config); err != nil {
		t.Fatal(err)
	}
	if config.Origin != "custom" || !samePath(config.ActivePath, customPath) || !config.CanSave {
		t.Fatalf("Save As config = %#v", config)
	}
	updated := layoutJSON(t, layoutApplyRequest{SchemaVersion: layoutConfigSchemaVersion, Layout: LayoutProfile{Algorithm: "layered", Options: map[string]any{"org.eclipse.elk.direction": "DOWN"}}})
	if response := requestLayout(t, server, http.MethodPut, "/v1/layout/config", updated); response.Code != http.StatusOK {
		t.Fatalf("Save active status = %d, body=%s", response.Code, response.Body.String())
	}
	data, err := os.ReadFile(customPath)
	if err != nil {
		t.Fatal(err)
	}
	var stored layoutConfigFile
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.Layout.Options["org.eclipse.elk.direction"] != "DOWN" {
		t.Fatalf("active file was not overwritten: %#v", stored.Layout)
	}
	if _, err := os.Stat(filepath.Join(root, layoutConfigFileName)); !os.IsNotExist(err) {
		t.Fatalf("Save As unexpectedly created a project-root file: %v", err)
	}
}

func TestLayoutSaveTargetsNearestDiscoveredAncestor(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "project")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	profile := LayoutProfile{Algorithm: "layered", Options: map[string]any{"org.eclipse.elk.direction": "RIGHT"}}
	data, err := encodeLayoutConfig(profile)
	if err != nil {
		t.Fatal(err)
	}
	ancestorPath := filepath.Join(parent, layoutConfigFileName)
	if err := os.WriteFile(ancestorPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(fixtureModel(t), ServerOptions{SourceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	updated := layoutJSON(t, layoutApplyRequest{SchemaVersion: layoutConfigSchemaVersion, Layout: LayoutProfile{Algorithm: "layered", Options: map[string]any{"org.eclipse.elk.direction": "UP"}}})
	response := requestLayout(t, server, http.MethodPut, "/v1/layout/config", updated)
	if response.Code != http.StatusOK {
		t.Fatalf("ancestor Save status = %d, body=%s", response.Code, response.Body.String())
	}
	stored, err := os.ReadFile(ancestorPath)
	if err != nil {
		t.Fatal(err)
	}
	var decoded layoutConfigFile
	if err := json.Unmarshal(stored, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Layout.Options["org.eclipse.elk.direction"] != "UP" {
		t.Fatalf("ancestor was not updated: %#v", decoded.Layout)
	}
	if _, err := os.Stat(filepath.Join(root, layoutConfigFileName)); !os.IsNotExist(err) {
		t.Fatalf("ordinary Save unexpectedly created a root config: %v", err)
	}
}

func TestLayoutEndpointsRejectOversizedRequests(t *testing.T) {
	server, err := NewServer(fixtureModel(t))
	if err != nil {
		t.Fatal(err)
	}
	oversized := []byte(fmt.Sprintf(`{"schema_version":"%s","layout":{"algorithm":"layered","options":{}}}%s`, layoutConfigSchemaVersion, strings.Repeat(" ", maxLayoutRequestBytes)))
	for _, request := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/v1/layout/apply"},
		{method: http.MethodPut, path: "/v1/layout/config/save-as"},
	} {
		response := requestLayout(t, server, request.method, request.path, oversized)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("oversized %s status = %d, body=%s", request.path, response.Code, response.Body.String())
		}
	}
}

func requestLayout(t *testing.T, server *Server, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	request := httptest.NewRequest(method, path, reader)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	return recorder
}

func layoutJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func decodeHTTPError(t *testing.T, data []byte) error {
	t.Helper()
	var envelope struct {
		Error struct {
			Code    analysis.ErrorCode `json:"code"`
			Message string             `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	return analysis.NewHostError(envelope.Error.Code, envelope.Error.Message, nil)
}
