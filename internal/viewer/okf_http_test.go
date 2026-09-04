package viewer

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestOKFHTTPExposesCanonicalReadAndSessionEnvelopes(t *testing.T) {
	root := t.TempDir()
	writeOKFHTTPFixture(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: area\ntitle: Root\n---\n")
	writeOKFHTTPFixture(t, filepath.Join(root, ".okf", "root", "child.md"), "---\ntype: concept\ntitle: Child\nparent: root\ncustom: retained\n---\n# Child\n<script>alert(1)</script>\n[Root](/root.md) [unsafe](javascript:alert(1)) [web](https://example.com).\n")
	writeOKFHTTPFixture(t, filepath.Join(root, "invalid", ".okf", "bad.md"), "missing frontmatter\n")
	server, err := NewServer(fixtureModel(t), ServerOptions{SourceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	rootResponse := getOKFHTTP(t, httpServer.URL+"/")
	if rootResponse.status != http.StatusOK || !strings.Contains(string(rootResponse.body), `content="true"`) || !strings.Contains(string(rootResponse.body), "okf-session-id") {
		t.Fatalf("root = %d %s", rootResponse.status, rootResponse.body)
	}

	catalogResponse := getOKFHTTP(t, httpServer.URL+"/v1/okf/catalog")
	var catalogEnvelope struct {
		Data domain.BundleCatalog `json:"data"`
		Meta map[string]any       `json:"meta"`
	}
	decodeOKFHTTP(t, catalogResponse, &catalogEnvelope)
	if catalogResponse.status != http.StatusOK || len(catalogEnvelope.Data.Bundles) != 2 || catalogEnvelope.Data.Bundles[1].Status != domain.BundleInvalid || catalogEnvelope.Meta["request_id"] == nil {
		t.Fatalf("catalog = %d %#v", catalogResponse.status, catalogEnvelope)
	}

	profilesResponse := getOKFHTTP(t, httpServer.URL+"/v1/okf/profiles")
	var profilesEnvelope struct {
		Data domain.ProfileCatalog `json:"data"`
	}
	decodeOKFHTTP(t, profilesResponse, &profilesEnvelope)
	if profilesResponse.status != http.StatusOK || len(profilesEnvelope.Data.Profiles) < 2 {
		t.Fatalf("profiles = %d %#v", profilesResponse.status, profilesEnvelope)
	}

	selection := postOKFHTTP(t, httpServer.URL+"/v1/okf/sessions/default/bundle", http.MethodPut, map[string]any{"bundle_id": ".okf"})
	var sessionEnvelope struct {
		Data struct {
			Projection domain.ProjectionSnapshot `json:"projection"`
		} `json:"data"`
	}
	decodeOKFHTTP(t, selection, &sessionEnvelope)
	if selection.status != http.StatusOK || sessionEnvelope.Data.Projection.Source.BundleID != ".okf" || len(sessionEnvelope.Data.Projection.Nodes) != 2 {
		t.Fatalf("selection = %d %#v", selection.status, sessionEnvelope)
	}

	projectionResponse := getOKFHTTP(t, httpServer.URL+"/v1/okf/sessions/default/projection")
	var projectionEnvelope struct {
		Data domain.ProjectionSnapshot `json:"data"`
	}
	decodeOKFHTTP(t, projectionResponse, &projectionEnvelope)
	if projectionResponse.status != http.StatusOK || projectionEnvelope.Data.Profile.ProfileID != "builtin:neutral" {
		t.Fatalf("projection = %d %#v", projectionResponse.status, projectionEnvelope)
	}

	detailResponse := getOKFHTTP(t, httpServer.URL+"/v1/okf/sessions/default/concept-detail?concept_id="+url.QueryEscape("root/child"))
	var detailEnvelope struct {
		Data domain.ConceptDetail `json:"data"`
	}
	decodeOKFHTTP(t, detailResponse, &detailEnvelope)
	if detailResponse.status != http.StatusOK || strings.Contains(detailEnvelope.Data.RenderedMarkdown.Content, "script") || !strings.Contains(detailEnvelope.Data.RenderedMarkdown.Content, "okf-concept=root") {
		t.Fatalf("detail = %d %#v", detailResponse.status, detailEnvelope)
	}

	missingResponse := getOKFHTTP(t, httpServer.URL+"/v1/okf/sessions/default/concept-detail?concept_id=missing")
	var failure struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeOKFHTTP(t, missingResponse, &failure)
	if missingResponse.status != http.StatusNotFound || failure.Error.Code != "okf_concept_not_found" {
		t.Fatalf("missing detail = %d %#v", missingResponse.status, failure)
	}

	methodResponse := postOKFHTTP(t, httpServer.URL+"/v1/okf/catalog", http.MethodPost, map[string]any{})
	var methodFailure struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeOKFHTTP(t, methodResponse, &methodFailure)
	if methodResponse.status != http.StatusMethodNotAllowed || methodFailure.Error.Code != "okf_method_not_allowed" {
		t.Fatalf("method failure = %d %#v", methodResponse.status, methodFailure)
	}
}

func TestOKFHTTPProfileSaveAsSaveAndBuiltinProtection(t *testing.T) {
	root := t.TempDir()
	writeOKFHTTPFixture(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: area\n---\n")
	server, err := NewServer(fixtureModel(t), ServerOptions{SourceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	profilesResponse := getOKFHTTP(t, httpServer.URL+"/v1/okf/profiles")
	var profilesEnvelope struct {
		Data domain.ProfileCatalog `json:"data"`
	}
	decodeOKFHTTP(t, profilesResponse, &profilesEnvelope)
	var neutral domain.Profile
	for _, value := range profilesEnvelope.Data.Profiles {
		if value.ProfileID == "builtin:neutral" {
			neutral = value
		}
	}
	if neutral.ProfileID == "" {
		t.Fatal("neutral profile missing")
	}
	saveAs := postOKFHTTP(t, httpServer.URL+"/v1/okf/profiles/save-as", http.MethodPost, map[string]any{"profile": neutral, "new_profile_id": "http-copy", "operation_id": "http-save-as"})
	var savedEnvelope struct {
		Data domain.ProjectConfiguration `json:"data"`
	}
	decodeOKFHTTP(t, saveAs, &savedEnvelope)
	if saveAs.status != http.StatusCreated || len(savedEnvelope.Data.Profiles) != 1 || savedEnvelope.Data.Profiles[0].ProfileID != "project:http-copy" {
		t.Fatalf("Save As = %d %#v", saveAs.status, savedEnvelope)
	}
	retry := postOKFHTTP(t, httpServer.URL+"/v1/okf/profiles/save-as", http.MethodPost, map[string]any{"profile": neutral, "new_profile_id": "http-copy", "operation_id": "http-save-as"})
	var retryEnvelope struct {
		Data domain.ProjectConfiguration `json:"data"`
	}
	decodeOKFHTTP(t, retry, &retryEnvelope)
	if retry.status != http.StatusCreated || retryEnvelope.Data.Revision != savedEnvelope.Data.Revision {
		t.Fatalf("Save As retry = %d %#v", retry.status, retryEnvelope)
	}

	updated := savedEnvelope.Data.Profiles[0]
	updated.Name = "Updated"
	save := postOKFHTTP(t, httpServer.URL+"/v1/okf/profiles/"+url.PathEscape(updated.ProfileID), http.MethodPut, map[string]any{"profile": updated, "expected_revision": savedEnvelope.Data.Revision, "operation_id": "http-save"})
	var updatedEnvelope struct {
		Data domain.ProjectConfiguration `json:"data"`
	}
	decodeOKFHTTP(t, save, &updatedEnvelope)
	if save.status != http.StatusOK || updatedEnvelope.Data.Profiles[0].Name != "Updated" {
		t.Fatalf("Save = %d %#v", save.status, updatedEnvelope)
	}

	builtin := postOKFHTTP(t, httpServer.URL+"/v1/okf/profiles/builtin:neutral", http.MethodPut, neutral)
	var builtinFailure struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeOKFHTTP(t, builtin, &builtinFailure)
	if builtin.status != http.StatusForbidden || builtinFailure.Error.Code != "okf_builtin_immutable" {
		t.Fatalf("builtin save = %d %#v", builtin.status, builtinFailure)
	}
}

func TestOKFHTTPModelOnlySessionReportsUnavailableBoundary(t *testing.T) {
	server, err := NewServer(fixtureModel(t))
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	response := getOKFHTTP(t, httpServer.URL+"/v1/okf/catalog")
	var failure struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeOKFHTTP(t, response, &failure)
	if response.status != http.StatusServiceUnavailable || failure.Error.Code != "okf_project_not_found" {
		t.Fatalf("model-only OKF = %d %#v", response.status, failure)
	}
}

type okfHTTPResponse struct {
	status int
	body   []byte
}

func getOKFHTTP(t *testing.T, endpoint string) okfHTTPResponse {
	t.Helper()
	response, err := http.Get(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return okfHTTPResponse{status: response.StatusCode, body: body}
}

func postOKFHTTP(t *testing.T, endpoint, method string, value any) okfHTTPResponse {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(method, endpoint, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	result, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return okfHTTPResponse{status: response.StatusCode, body: result}
}

func decodeOKFHTTP(t *testing.T, response okfHTTPResponse, target any) {
	t.Helper()
	if err := json.Unmarshal(response.body, target); err != nil {
		t.Fatalf("decode HTTP response %d: %v; body=%s", response.status, err, response.body)
	}
}

func writeOKFHTTPFixture(t *testing.T, pathValue, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(pathValue), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pathValue, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
