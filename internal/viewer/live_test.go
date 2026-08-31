package viewer

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/live"
	"github.com/buffo/arch-view/internal/model/canonical"
	"github.com/buffo/arch-view/internal/quality"
)

func TestLiveViewerServesSessionStatusAndLatestModel(t *testing.T) {
	root := t.TempDir()
	value, err := canonical.Normalize(analysis.AnalysisResult{
		Status:   analysis.StatusComplete,
		Analyzer: analysis.AnalyzerInfo{ID: "analyzer:test", Version: "1.0.0", Language: "go", APIVersion: analysis.AnalyzerAPIVersion},
		Project:  analysis.ProjectInfo{RootLabel: "viewer", Boundary: "repository"},
		Modules:  []analysis.ModuleObservation{}, References: []analysis.Reference{}, SourceReferences: []analysis.SourceReference{}, Relationships: []analysis.RelationshipObservation{}, Diagnostics: []analysis.Diagnostic{},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := live.StartLiveSession(context.Background(), live.LiveSessionConfig{
		SchemaVersion: live.LiveSchemaVersion, SessionID: "session:viewer", RepositoryRoot: ".", WatchRoots: []live.WatchRoot{{Path: ".", Recursive: true}}, SourceIndexRequest: live.SourceIndexRequest{Enabled: false},
	}, root, live.SessionOptions{Scanner: live.StaticScanner{Result: live.ScanResult{Model: value}}, Fingerprinter: live.StaticFingerprinter{Value: live.InputFingerprint{ManifestFingerprint: live.ContentDigest{Algorithm: "hash:sha-256", Value: strings.Repeat("a", 64)}, ContentFingerprint: live.ContentDigest{Algorithm: "hash:sha-256", Value: strings.Repeat("b", 64)}, Files: []live.FileFingerprint{}}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	if err := session.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	server, err := NewLiveServer(session)
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	rootResponse, err := http.Get(httpServer.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	rootBody := readViewerBody(t, rootResponse)
	if rootResponse.StatusCode != http.StatusOK || !strings.Contains(rootBody, `name="live-enabled" content="true"`) || !strings.Contains(rootBody, `name="live-session-id" content="session:viewer"`) {
		t.Fatalf("live root status=%d body contains live metadata=%v", rootResponse.StatusCode, strings.Contains(rootBody, "live-enabled"))
	}

	statusResponse, err := http.Get(httpServer.URL + "/v1/live/session:viewer/status")
	if err != nil {
		t.Fatal(err)
	}
	statusBody := readViewerBody(t, statusResponse)
	if statusResponse.StatusCode != http.StatusOK || !strings.Contains(statusBody, `"schema_version":"arch-view.query/v1"`) || !strings.Contains(statusBody, `"revision":1`) {
		t.Fatalf("live status response status=%d body=%s", statusResponse.StatusCode, statusBody)
	}

	modelResponse, err := http.Get(httpServer.URL + "/v1/models/session%3Aviewer")
	if err != nil {
		t.Fatal(err)
	}
	modelBody := readViewerBody(t, modelResponse)
	if modelResponse.StatusCode != http.StatusOK || !strings.Contains(modelBody, `"schema_version":"arch-view.model/v1"`) {
		t.Fatalf("live model response status=%d body=%s", modelResponse.StatusCode, modelBody)
	}

	oldRevisionResponse, err := http.Get(httpServer.URL + "/v1/models/session%3Aviewer?revision=1")
	if err != nil {
		t.Fatal(err)
	}
	oldRevisionBody := readViewerBody(t, oldRevisionResponse)
	if oldRevisionResponse.StatusCode != http.StatusOK || !strings.Contains(oldRevisionBody, `"schema_version":"arch-view.model/v1"`) {
		t.Fatalf("revision-pinned model response status=%d body=%s", oldRevisionResponse.StatusCode, oldRevisionBody)
	}

	missingRevisionResponse, err := http.Get(httpServer.URL + "/v1/models/session%3Aviewer?revision=2")
	if err != nil {
		t.Fatal(err)
	}
	missingRevisionBody := readViewerBody(t, missingRevisionResponse)
	if missingRevisionResponse.StatusCode != http.StatusConflict || !strings.Contains(missingRevisionBody, "requested live revision is unavailable") {
		t.Fatalf("missing revision response status=%d body=%s", missingRevisionResponse.StatusCode, missingRevisionBody)
	}

	invalidRevisionResponse, err := http.Get(httpServer.URL + "/v1/models/session%3Aviewer?revision=not-a-number")
	if err != nil {
		t.Fatal(err)
	}
	invalidRevisionBody := readViewerBody(t, invalidRevisionResponse)
	if invalidRevisionResponse.StatusCode != http.StatusBadRequest || !strings.Contains(invalidRevisionBody, "positive integer") {
		t.Fatalf("invalid revision response status=%d body=%s", invalidRevisionResponse.StatusCode, invalidRevisionBody)
	}
}

func TestLiveViewerRejectsLegacyPolicyWritesAndUsesSharedQualityEvaluation(t *testing.T) {
	root := t.TempDir()
	writeViewerSourceFixture(t, root)
	profile := bridgeFileSizeProfile("profile:live", 1)
	writeQualityBridgeProfile(t, root, "live.json", profile)
	value := sourceIndexedViewerModel(t, root)
	resolver := live.NewMemoryQualityProfileResolver(profile)
	catalog := quality.NewDefaultCatalog()
	session, err := live.StartLiveSession(context.Background(), live.LiveSessionConfig{
		SchemaVersion:      live.LiveSchemaVersion,
		SessionID:          "session:live-viewer",
		RepositoryRoot:     ".",
		WatchRoots:         []live.WatchRoot{{Path: ".", Recursive: true}},
		SourceIndexRequest: live.SourceIndexRequest{Enabled: true},
		QualityRequest:     &live.QualityRequest{ProfileID: profile.ProfileID, ProfileVersion: profile.ProfileVersion},
	}, root, live.SessionOptions{
		Scanner:        live.StaticScanner{Result: live.ScanResult{Model: value}},
		Fingerprinter:  live.StaticFingerprinter{Value: live.InputFingerprint{ManifestFingerprint: live.ContentDigest{Algorithm: "hash:sha-256", Value: strings.Repeat("a", 64)}, ContentFingerprint: live.ContentDigest{Algorithm: "hash:sha-256", Value: strings.Repeat("b", 64)}}},
		QualityCatalog: catalog,
		Profiles:       resolver,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	if err := session.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	server, err := NewLiveServer(session)
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	profileBody := []byte(`{"schema_version":"arch-view.quality-profile-save/v1","profile_id":"profile:live","profile_version":"1.0.0","rule_bindings":[]}`)
	profileRequest := httptest.NewRequest(http.MethodPut, "/v1/quality/profiles/save", bytes.NewReader(profileBody))
	profileResponse := httptest.NewRecorder()
	server.Handler().ServeHTTP(profileResponse, profileRequest)
	if profileResponse.Code != http.StatusForbidden || !strings.Contains(profileResponse.Body.String(), "quality policy writes are disabled") {
		t.Fatalf("live profile write status=%d body=%s", profileResponse.Code, profileResponse.Body.String())
	}

	baselineRequest := httptest.NewRequest(http.MethodPost, "/v1/quality/baselines/create", bytes.NewReader([]byte(`{}`)))
	baselineResponse := httptest.NewRecorder()
	server.Handler().ServeHTTP(baselineResponse, baselineRequest)
	if baselineResponse.Code != http.StatusForbidden || !strings.Contains(baselineResponse.Body.String(), "quality policy writes are disabled") {
		t.Fatalf("live baseline write status=%d body=%s", baselineResponse.Code, baselineResponse.Body.String())
	}

	evaluationBody, err := json.Marshal(qualityEvaluationRequest{SchemaVersion: qualityEvaluationRequestSchemaVersion, ProfileID: profile.ProfileID, ProfileVersion: profile.ProfileVersion})
	if err != nil {
		t.Fatal(err)
	}
	evaluationResponse, err := http.Post(httpServer.URL+"/v1/quality/evaluate", "application/json", bytes.NewReader(evaluationBody))
	if err != nil {
		t.Fatal(err)
	}
	var evaluated qualityEvaluationHTTPResponse
	if err := json.NewDecoder(evaluationResponse.Body).Decode(&evaluated); err != nil {
		_ = evaluationResponse.Body.Close()
		t.Fatal(err)
	}
	_ = evaluationResponse.Body.Close()
	if evaluationResponse.StatusCode != http.StatusOK || evaluated.Report == nil || evaluated.Report.ProfileID != profile.ProfileID {
		t.Fatalf("live shared evaluation status=%d report=%#v", evaluationResponse.StatusCode, evaluated.Report)
	}

	findingsURL := httpServer.URL + "/v1/models/" + url.PathEscape(session.Config().SessionID) + "/quality/findings?revision=1&limit=25"
	findingsResponse, err := http.Get(findingsURL)
	if err != nil {
		t.Fatal(err)
	}
	var findings qualityFindingsHTTPResponse
	if err := json.NewDecoder(findingsResponse.Body).Decode(&findings); err != nil {
		_ = findingsResponse.Body.Close()
		t.Fatal(err)
	}
	_ = findingsResponse.Body.Close()
	if findingsResponse.StatusCode != http.StatusOK || findings.Status != "available" || findings.ReportID != evaluated.Report.EvaluationID {
		t.Fatalf("live shared findings status=%d findings=%#v", findingsResponse.StatusCode, findings)
	}
}

func readViewerBody(t *testing.T, response *http.Response) string {
	t.Helper()
	defer func() {
		_ = response.Body.Close()
	}()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
