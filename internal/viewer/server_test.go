package viewer

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestServerServesReadOnlyModelSceneAndBrowserAssets(t *testing.T) {
	value := fixtureModel(t)
	server, err := NewServer(value)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	rootResponse, err := http.Get(httpServer.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	rootBody, err := io.ReadAll(rootResponse.Body)
	_ = rootResponse.Body.Close()
	if err != nil {
		t.Fatalf("read root response: %v", err)
	}
	if rootResponse.StatusCode != http.StatusOK || rootResponse.Header.Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("root response = %d %q", rootResponse.StatusCode, rootResponse.Header.Get("Content-Type"))
	}
	rootText := string(rootBody)
	if !strings.Contains(rootText, "SEMANTIC SCENE") || !strings.Contains(rootText, value.ModelID) || !strings.Contains(rootText, "reference-visibility") || !strings.Contains(rootText, "Accessible list &amp; imports") || strings.Contains(rootText, "__ARCH_VIEW_MODEL_ID__") {
		t.Fatalf("root page did not contain the model bootstrap: %s", rootText)
	}

	assetResponse, err := http.Get(httpServer.URL + "/assets/app.js")
	if err != nil {
		t.Fatalf("GET app.js: %v", err)
	}
	assetBody, err := io.ReadAll(assetResponse.Body)
	_ = assetResponse.Body.Close()
	if err != nil {
		t.Fatalf("read app.js: %v", err)
	}
	if assetResponse.StatusCode != http.StatusOK || !strings.Contains(string(assetBody), "DOMContentLoaded") || !strings.Contains(string(assetBody), "reference-detail") || !strings.Contains(string(assetBody), "reference_visibility") {
		t.Fatalf("asset response = %d %q", assetResponse.StatusCode, string(assetBody))
	}

	elkResponse, err := http.Get(httpServer.URL + "/assets/vendor/elk.bundled.js")
	if err != nil {
		t.Fatalf("GET elk.bundled.js: %v", err)
	}
	elkBody, err := io.ReadAll(elkResponse.Body)
	_ = elkResponse.Body.Close()
	if err != nil {
		t.Fatalf("read elk.bundled.js: %v", err)
	}
	if elkResponse.StatusCode != http.StatusOK || elkResponse.Header.Get("Content-Type") != "text/javascript; charset=utf-8" || !strings.Contains(string(elkBody), "ELK") {
		t.Fatalf("ELK asset response = %d %q", elkResponse.StatusCode, elkResponse.Header.Get("Content-Type"))
	}

	workerResponse, err := http.Get(httpServer.URL + "/assets/vendor/elk-worker.min.js")
	if err != nil {
		t.Fatalf("GET elk-worker.min.js: %v", err)
	}
	workerBody, err := io.ReadAll(workerResponse.Body)
	_ = workerResponse.Body.Close()
	if err != nil {
		t.Fatalf("read elk-worker.min.js: %v", err)
	}
	if workerResponse.StatusCode != http.StatusOK || workerResponse.Header.Get("Content-Type") != "text/javascript; charset=utf-8" || len(workerBody) == 0 {
		t.Fatalf("ELK worker response = %d %q", workerResponse.StatusCode, workerResponse.Header.Get("Content-Type"))
	}

	stylesResponse, err := http.Get(httpServer.URL + "/assets/styles.css")
	if err != nil {
		t.Fatalf("GET styles.css: %v", err)
	}
	stylesBody, err := io.ReadAll(stylesResponse.Body)
	_ = stylesResponse.Body.Close()
	if err != nil {
		t.Fatalf("read styles.css: %v", err)
	}
	if stylesResponse.StatusCode != http.StatusOK || !strings.Contains(string(stylesBody), ".edge-hit") || !strings.Contains(string(stylesBody), "fill: none") {
		t.Fatalf("edge hit-area style missing from response = %d", stylesResponse.StatusCode)
	}

	modelResponse, err := http.Get(httpServer.URL + "/v1/models/" + url.PathEscape(value.ModelID))
	if err != nil {
		t.Fatalf("GET model: %v", err)
	}
	var decodedModel map[string]any
	if err := json.NewDecoder(modelResponse.Body).Decode(&decodedModel); err != nil {
		_ = modelResponse.Body.Close()
		t.Fatalf("decode model response: %v", err)
	}
	_ = modelResponse.Body.Close()
	if modelResponse.StatusCode != http.StatusOK || decodedModel["model_id"] != value.ModelID {
		t.Fatalf("model response = %d %#v", modelResponse.StatusCode, decodedModel)
	}

	projectionResponse, err := http.Get(httpServer.URL + "/v1/models/" + url.PathEscape(value.ModelID) + "/projection?path=core&mode=list")
	if err != nil {
		t.Fatalf("GET projection: %v", err)
	}
	var scene SceneSnapshot
	if err := json.NewDecoder(projectionResponse.Body).Decode(&scene); err != nil {
		_ = projectionResponse.Body.Close()
		t.Fatalf("decode projection response: %v", err)
	}
	_ = projectionResponse.Body.Close()
	if projectionResponse.StatusCode != http.StatusOK || scene.DisplayMode != "list" || len(scene.HierarchyPath) != 1 || scene.HierarchyPath[0] != "core" {
		t.Fatalf("projection response = %d %#v", projectionResponse.StatusCode, scene)
	}
	if scene.ReferenceVisibility != ReferenceVisibilityHidden {
		t.Fatalf("default reference visibility = %q", scene.ReferenceVisibility)
	}

	aggregatedResponse, err := http.Get(httpServer.URL + "/v1/models/" + url.PathEscape(value.ModelID) + "/projection?reference_visibility=aggregated")
	if err != nil {
		t.Fatalf("GET aggregated projection: %v", err)
	}
	var aggregated SceneSnapshot
	if err := json.NewDecoder(aggregatedResponse.Body).Decode(&aggregated); err != nil {
		_ = aggregatedResponse.Body.Close()
		t.Fatalf("decode aggregated projection: %v", err)
	}
	_ = aggregatedResponse.Body.Close()
	if aggregatedResponse.StatusCode != http.StatusOK || aggregated.ReferenceVisibility != ReferenceVisibilityAggregated || aggregated.ReferenceSummary.AggregatedCount != 1 {
		t.Fatalf("aggregated response = %d %#v", aggregatedResponse.StatusCode, aggregated)
	}

	invalidVisibility, err := http.Get(httpServer.URL + "/v1/models/" + url.PathEscape(value.ModelID) + "/projection?reference_visibility=invalid")
	if err != nil {
		t.Fatalf("GET invalid visibility: %v", err)
	}
	_ = invalidVisibility.Body.Close()
	if invalidVisibility.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid visibility status = %d", invalidVisibility.StatusCode)
	}

	invalidScope, err := http.Get(httpServer.URL + "/v1/models/" + url.PathEscape(value.ModelID) + "/projection?reference_scope=not-a-scope")
	if err != nil {
		t.Fatalf("GET invalid reference scope: %v", err)
	}
	_ = invalidScope.Body.Close()
	if invalidScope.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid reference scope status = %d", invalidScope.StatusCode)
	}

	methodRequest, err := http.NewRequest(http.MethodPost, httpServer.URL+"/", nil)
	if err != nil {
		t.Fatalf("create method request: %v", err)
	}
	methodResponse, err := http.DefaultClient.Do(methodRequest)
	if err != nil {
		t.Fatalf("POST /: %v", err)
	}
	_ = methodResponse.Body.Close()
	if methodResponse.StatusCode != http.StatusMethodNotAllowed || methodResponse.Header.Get("Allow") != http.MethodGet {
		t.Fatalf("read-only response = %d allow=%q", methodResponse.StatusCode, methodResponse.Header.Get("Allow"))
	}

	sourceResponse, err := http.Get(httpServer.URL + "/v1/source?path=../outside.go")
	if err != nil {
		t.Fatalf("GET unsupported source route: %v", err)
	}
	_ = sourceResponse.Body.Close()
	if sourceResponse.StatusCode != http.StatusNotFound {
		t.Fatalf("source route status = %d, want 404 until the evidence slice owns it", sourceResponse.StatusCode)
	}
}

func TestServerRejectsUnknownModelAndHierarchy(t *testing.T) {
	server, err := NewServer(fixtureModel(t))
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	unknown, err := http.Get(httpServer.URL + "/v1/models/unknown")
	if err != nil {
		t.Fatalf("GET unknown model: %v", err)
	}
	_ = unknown.Body.Close()
	if unknown.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown model status = %d", unknown.StatusCode)
	}

	invalidPath, err := http.Get(httpServer.URL + "/v1/models/" + url.PathEscape(server.model.ModelID) + "/projection?path=missing")
	if err != nil {
		t.Fatalf("GET invalid path: %v", err)
	}
	_ = invalidPath.Body.Close()
	if invalidPath.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("invalid hierarchy status = %d", invalidPath.StatusCode)
	}
}
