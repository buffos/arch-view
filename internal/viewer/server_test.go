package viewer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/model/canonical"
	"github.com/buffo/arch-view/internal/viewer/scene"
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
	if !strings.Contains(rootText, "SEMANTIC SCENE") || !strings.Contains(rootText, value.ModelID) || !strings.Contains(rootText, "reference-visibility") || !strings.Contains(rootText, "Accessible scene list") || !strings.Contains(rootText, "HUMAN-ORIENTED INSPECTION") || !strings.Contains(rootText, "type=\"module\"") || !strings.Contains(rootText, "/assets/vendor/elk-worker.min.js") || strings.Contains(rootText, "__ARCH_VIEW_MODEL_ID__") {
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
	if assetResponse.StatusCode != http.StatusOK || !strings.Contains(string(assetBody), "DOMContentLoaded") || !strings.Contains(string(assetBody), "./app/bootstrap.js") {
		t.Fatalf("asset response = %d %q", assetResponse.StatusCode, string(assetBody))
	}

	requestAssetResponse, err := http.Get(httpServer.URL + "/assets/layout_request.js")
	if err != nil {
		t.Fatalf("GET layout_request.js: %v", err)
	}
	requestAssetBody, err := io.ReadAll(requestAssetResponse.Body)
	_ = requestAssetResponse.Body.Close()
	if err != nil {
		t.Fatalf("read layout_request.js: %v", err)
	}
	if requestAssetResponse.StatusCode != http.StatusOK || requestAssetResponse.Header.Get("Content-Type") != "text/javascript; charset=utf-8" || !strings.Contains(string(requestAssetBody), "buildRootLayoutOptions") {
		t.Fatalf("layout request asset = %d %q", requestAssetResponse.StatusCode, requestAssetResponse.Header.Get("Content-Type"))
	}

	routeResponse, err := http.Get(httpServer.URL + "/assets/graph_route.js")
	if err != nil {
		t.Fatalf("GET graph_route.js: %v", err)
	}
	routeBody, err := io.ReadAll(routeResponse.Body)
	_ = routeResponse.Body.Close()
	if err != nil {
		t.Fatalf("read graph route asset: %v", err)
	}
	if routeResponse.StatusCode != http.StatusOK || routeResponse.Header.Get("Content-Type") != "text/javascript; charset=utf-8" || !strings.Contains(string(routeBody), "fromELKSections") {
		t.Fatalf("graph route asset = %d %q", routeResponse.StatusCode, routeResponse.Header.Get("Content-Type"))
	}

	moduleResponse, err := http.Get(httpServer.URL + "/assets/app/bootstrap.js")
	if err != nil {
		t.Fatalf("GET app/bootstrap.js: %v", err)
	}
	moduleBody, err := io.ReadAll(moduleResponse.Body)
	_ = moduleResponse.Body.Close()
	if err != nil {
		t.Fatalf("read app module asset: %v", err)
	}
	if moduleResponse.StatusCode != http.StatusOK || moduleResponse.Header.Get("Content-Type") != "text/javascript; charset=utf-8" || !strings.Contains(string(moduleBody), "export function bootstrap") {
		t.Fatalf("app module asset = %d %q", moduleResponse.StatusCode, moduleResponse.Header.Get("Content-Type"))
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
	stylesText := string(stylesBody)
	if stylesResponse.StatusCode != http.StatusOK || !strings.Contains(stylesText, ".edge-hit") || !strings.Contains(stylesText, "fill: none") || strings.Contains(stylesText, "@import") || !strings.Contains(stylesText, "select option") {
		t.Fatalf("stylesheet response = %d edge=%t fill=%t imports=%t options=%t", stylesResponse.StatusCode, strings.Contains(stylesText, ".edge-hit"), strings.Contains(stylesText, "fill: none"), strings.Contains(stylesText, "@import"), strings.Contains(stylesText, "select option"))
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
	var snapshot scene.SceneSnapshot
	if err := json.NewDecoder(projectionResponse.Body).Decode(&snapshot); err != nil {
		_ = projectionResponse.Body.Close()
		t.Fatalf("decode projection response: %v", err)
	}
	_ = projectionResponse.Body.Close()
	if projectionResponse.StatusCode != http.StatusOK || snapshot.DisplayMode != "list" || len(snapshot.HierarchyPath) != 1 || snapshot.HierarchyPath[0] != "core" {
		t.Fatalf("projection response = %d %#v", projectionResponse.StatusCode, snapshot)
	}
	if snapshot.ReferenceVisibility != scene.ReferenceVisibilityHidden {
		t.Fatalf("default reference visibility = %q", snapshot.ReferenceVisibility)
	}

	aggregatedResponse, err := http.Get(httpServer.URL + "/v1/models/" + url.PathEscape(value.ModelID) + "/projection?reference_visibility=aggregated")
	if err != nil {
		t.Fatalf("GET aggregated projection: %v", err)
	}
	var aggregated scene.SceneSnapshot
	if err := json.NewDecoder(aggregatedResponse.Body).Decode(&aggregated); err != nil {
		_ = aggregatedResponse.Body.Close()
		t.Fatalf("decode aggregated projection: %v", err)
	}
	_ = aggregatedResponse.Body.Close()
	if aggregatedResponse.StatusCode != http.StatusOK || aggregated.ReferenceVisibility != scene.ReferenceVisibilityAggregated || aggregated.ReferenceSummary.AggregatedCount != 1 {
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

func TestServerServesContainedReadOnlySourceAndRejectsTraversal(t *testing.T) {
	value := fixtureModel(t)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "api"), 0o755); err != nil {
		t.Fatalf("create source directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "api", "api.go"), []byte("package api\n\nimport (\n\t\"example.com/app/core\"\n)\n\nfunc Use() {}\n"), 0o644); err != nil {
		t.Fatalf("write source fixture: %v", err)
	}
	server, err := NewServer(value, ServerOptions{SourceRoot: root})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	query := url.Values{}
	query.Set("model_id", value.ModelID)
	query.Set("evidence_id", "src-api-import")
	query.Set("path", "api/api.go")
	query.Set("start_line", "3")
	query.Set("end_line", "5")
	response, err := http.Get(httpServer.URL + "/v1/source?" + query.Encode())
	if err != nil {
		t.Fatalf("GET contained source: %v", err)
	}
	var excerpt SourceExcerpt
	if err := json.NewDecoder(response.Body).Decode(&excerpt); err != nil {
		_ = response.Body.Close()
		t.Fatalf("decode source excerpt: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || !excerpt.ReadOnly || excerpt.Path != "api/api.go" || len(excerpt.Lines) != 3 || excerpt.Lines[0].Number != 3 {
		t.Fatalf("source response = %d %#v", response.StatusCode, excerpt)
	}

	traversal, err := http.Get(httpServer.URL + "/v1/source?model_id=" + url.QueryEscape(value.ModelID) + "&path=../outside.go")
	if err != nil {
		t.Fatalf("GET traversal source: %v", err)
	}
	_ = traversal.Body.Close()
	if traversal.StatusCode != http.StatusForbidden {
		t.Fatalf("traversal status = %d, want %d", traversal.StatusCode, http.StatusForbidden)
	}

	stale, err := http.Get(httpServer.URL + "/v1/source?model_id=old-revision&path=api%2Fapi.go")
	if err != nil {
		t.Fatalf("GET stale source: %v", err)
	}
	_ = stale.Body.Close()
	if stale.StatusCode != http.StatusNotFound {
		t.Fatalf("stale source status = %d, want %d", stale.StatusCode, http.StatusNotFound)
	}

	zeroLine, err := http.Get(httpServer.URL + "/v1/source?model_id=" + url.QueryEscape(value.ModelID) + "&evidence_id=src-api-import&path=api%2Fapi.go&start_line=0&end_line=1")
	if err != nil {
		t.Fatalf("GET zero-line source: %v", err)
	}
	_ = zeroLine.Body.Close()
	if zeroLine.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("zero start line status = %d, want %d", zeroLine.StatusCode, http.StatusUnprocessableEntity)
	}
}

func TestServerReanalysisReplacesOnlyValidRevision(t *testing.T) {
	value := fixtureModel(t)
	root := t.TempDir()
	called := false
	server, err := NewServer(value, ServerOptions{
		SourceRoot: root,
		Reanalyze: func(_ context.Context, request ReanalysisRequest) (model.Model, error) {
			called = request.ProjectRoot == root && request.Language == "go"
			return value, nil
		},
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	request, err := http.NewRequest(http.MethodPost, httpServer.URL+"/v1/reanalysis", bytes.NewBufferString(`{"project_root":"","language":"go","options":{}}`))
	if err != nil {
		t.Fatalf("create reanalysis request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("POST reanalysis: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || !called {
		t.Fatalf("reanalysis response = %d called=%v", response.StatusCode, called)
	}

	failedServer, err := NewServer(value, ServerOptions{
		SourceRoot: root,
		Reanalyze: func(context.Context, ReanalysisRequest) (model.Model, error) {
			return model.Model{}, errors.New("fixture analysis failed")
		},
	})
	if err != nil {
		t.Fatalf("NewServer(failed) error = %v", err)
	}
	failedHTTPServer := httptest.NewServer(failedServer.Handler())
	defer failedHTTPServer.Close()
	failedRequest, err := http.NewRequest(http.MethodPost, failedHTTPServer.URL+"/v1/reanalysis", bytes.NewBufferString(`{"project_root":"","language":"go","options":{}}`))
	if err != nil {
		t.Fatalf("create failed reanalysis request: %v", err)
	}
	failedResponse, err := http.DefaultClient.Do(failedRequest)
	if err != nil {
		t.Fatalf("POST failed reanalysis: %v", err)
	}
	_ = failedResponse.Body.Close()
	if failedResponse.StatusCode != http.StatusUnprocessableEntity || failedServer.snapshot().ModelID != value.ModelID {
		t.Fatalf("failed reanalysis response = %d model=%q", failedResponse.StatusCode, failedServer.snapshot().ModelID)
	}

	failedModel, err := canonical.Normalize(analysis.AnalysisResult{
		Status: analysis.StatusFailed,
		Analyzer: analysis.AnalyzerInfo{
			ID:         "org.archview.go",
			Version:    "1.0.0",
			Language:   "go",
			APIVersion: analysis.AnalyzerAPIVersion,
		},
		Project: analysis.ProjectInfo{RootLabel: "fixture", Boundary: "go.mod"},
	})
	if err != nil {
		t.Fatalf("normalize failed model: %v", err)
	}
	failedRevisionServer, err := NewServer(value, ServerOptions{
		SourceRoot: root,
		Reanalyze: func(context.Context, ReanalysisRequest) (model.Model, error) {
			return failedModel, nil
		},
	})
	if err != nil {
		t.Fatalf("NewServer(failed revision) error = %v", err)
	}
	failedRevisionHTTPServer := httptest.NewServer(failedRevisionServer.Handler())
	defer failedRevisionHTTPServer.Close()
	failedRevisionRequest, err := http.NewRequest(http.MethodPost, failedRevisionHTTPServer.URL+"/v1/reanalysis", bytes.NewBufferString(`{"project_root":"","language":"go","options":{}}`))
	if err != nil {
		t.Fatalf("create failed revision request: %v", err)
	}
	failedRevisionResponse, err := http.DefaultClient.Do(failedRevisionRequest)
	if err != nil {
		t.Fatalf("POST failed revision: %v", err)
	}
	_ = failedRevisionResponse.Body.Close()
	if failedRevisionResponse.StatusCode != http.StatusUnprocessableEntity || failedRevisionServer.snapshot().ModelID != value.ModelID {
		t.Fatalf("failed revision response = %d model=%q", failedRevisionResponse.StatusCode, failedRevisionServer.snapshot().ModelID)
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
