package viewer

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/orchestration"
)

func TestAggregateServerExposesCachedScopesAndProjection(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("write source fixture: %v", err)
	}
	run := aggregateFixtureRun(t, root)
	server, err := NewAggregateServer(run, ServerOptions{SourceRoot: root})
	if err != nil {
		t.Fatalf("NewAggregateServer() error = %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	rootResponse, err := http.Get(httpServer.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	var rootBody []byte
	rootBody, err = readBody(rootResponse)
	if err != nil {
		t.Fatalf("read root: %v", err)
	}
	if rootResponse.StatusCode != http.StatusOK || !strings.Contains(string(rootBody), "Analysis scope") || !strings.Contains(string(rootBody), run.RunID) || !strings.Contains(string(rootBody), "aggregate-enabled") {
		t.Fatalf("aggregate root response = %d %s", rootResponse.StatusCode, rootBody)
	}

	scopesResponse, err := http.Get(httpServer.URL + "/v1/analyses/" + url.PathEscape(run.RunID) + "/scopes")
	if err != nil {
		t.Fatalf("GET scopes: %v", err)
	}
	var scopes struct {
		SchemaVersion string                       `json:"schema_version"`
		ActiveScope   string                       `json:"active_scope"`
		RunID         string                       `json:"run_id"`
		Scopes        []orchestration.ScopeSummary `json:"scopes"`
	}
	if err := json.NewDecoder(scopesResponse.Body).Decode(&scopes); err != nil {
		_ = scopesResponse.Body.Close()
		t.Fatalf("decode scopes: %v", err)
	}
	_ = scopesResponse.Body.Close()
	if scopesResponse.StatusCode != http.StatusOK || scopes.SchemaVersion != scopeListSchemaVersion || scopes.ActiveScope != "all" || scopes.RunID != run.RunID || len(scopes.Scopes) != 1 {
		t.Fatalf("scopes response = %d %#v", scopesResponse.StatusCode, scopes)
	}

	allResponse, err := http.Get(httpServer.URL + "/v1/analyses/" + url.PathEscape(run.RunID) + "/projection?scope=all")
	if err != nil {
		t.Fatalf("GET all projection: %v", err)
	}
	var allProjection struct {
		ScopeID         string `json:"scope_id"`
		AggregateStatus string `json:"aggregate_status"`
	}
	if err := json.NewDecoder(allResponse.Body).Decode(&allProjection); err != nil {
		_ = allResponse.Body.Close()
		t.Fatalf("decode all projection: %v", err)
	}
	_ = allResponse.Body.Close()
	if allResponse.StatusCode != http.StatusOK || allProjection.ScopeID != "all" || allProjection.AggregateStatus != string(analysis.StatusComplete) {
		t.Fatalf("all projection = %d %#v", allResponse.StatusCode, allProjection)
	}

	scopeID := scopes.Scopes[0].ScopeID
	scopeResponse, err := http.Get(httpServer.URL + "/v1/analyses/" + url.PathEscape(run.RunID) + "/projection?scope=" + url.QueryEscape(scopeID))
	if err != nil {
		t.Fatalf("GET scope projection: %v", err)
	}
	var scopeProjection struct {
		ScopeID string `json:"scope_id"`
	}
	if err := json.NewDecoder(scopeResponse.Body).Decode(&scopeProjection); err != nil {
		_ = scopeResponse.Body.Close()
		t.Fatalf("decode scope projection: %v", err)
	}
	_ = scopeResponse.Body.Close()
	if scopeResponse.StatusCode != http.StatusOK || scopeProjection.ScopeID != scopeID {
		t.Fatalf("scope projection = %d %#v", scopeResponse.StatusCode, scopeProjection)
	}

	eventsResponse, err := http.Get(httpServer.URL + "/v1/analyses/" + url.PathEscape(run.RunID) + "/events")
	if err != nil {
		t.Fatalf("GET events: %v", err)
	}
	var events struct {
		Events []orchestration.JobLifecycleEvent `json:"events"`
	}
	if err := json.NewDecoder(eventsResponse.Body).Decode(&events); err != nil {
		_ = eventsResponse.Body.Close()
		t.Fatalf("decode events: %v", err)
	}
	_ = eventsResponse.Body.Close()
	if eventsResponse.StatusCode != http.StatusOK || len(events.Events) == 0 {
		t.Fatalf("events response = %d %#v", eventsResponse.StatusCode, events)
	}

	unknownResponse, err := http.Get(httpServer.URL + "/v1/analyses/" + url.PathEscape(run.RunID) + "/projection?scope=scope-missing")
	if err != nil {
		t.Fatalf("GET unknown scope: %v", err)
	}
	unknownBody, err := readBody(unknownResponse)
	if err != nil {
		t.Fatalf("read unknown scope: %v", err)
	}
	if unknownResponse.StatusCode != http.StatusNotFound || !strings.Contains(string(unknownBody), string(analysis.ErrAnalysisScopeNotFound)) {
		t.Fatalf("unknown scope response = %d %s", unknownResponse.StatusCode, unknownBody)
	}
}

func TestAggregateServerAnalysisPOSTIsTheOnlyExecutionEntryPoint(t *testing.T) {
	root := t.TempDir()
	run := aggregateFixtureRun(t, root)
	var calls atomic.Int32
	server, err := NewAggregateServer(run, ServerOptions{
		SourceRoot: root,
		AnalyzeCombined: func(_ context.Context, _ CombinedAnalysisRequest) (orchestration.AnalysisRun, error) {
			calls.Add(1)
			return run, nil
		},
	})
	if err != nil {
		t.Fatalf("NewAggregateServer() error = %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Post(httpServer.URL+"/v1/analyses", "application/json", strings.NewReader(`{"project_root":""}`))
	if err != nil {
		t.Fatalf("POST analyses: %v", err)
	}
	if body, readErr := readBody(response); readErr != nil {
		t.Fatalf("read POST response: %v", readErr)
	} else if response.StatusCode != http.StatusOK || !strings.Contains(string(body), orchestration.AggregateSchemaVersion) {
		t.Fatalf("POST response = %d %s", response.StatusCode, body)
	}
	if calls.Load() != 1 {
		t.Fatalf("analysis callback calls after POST = %d, want 1", calls.Load())
	}

	projectionURL := httpServer.URL + "/v1/analyses/" + url.PathEscape(run.RunID) + "/projection?scope=all"
	projection, err := http.Get(projectionURL)
	if err != nil {
		t.Fatalf("GET cached projection: %v", err)
	}
	_ = projection.Body.Close()
	if projection.StatusCode != http.StatusOK || calls.Load() != 1 {
		t.Fatalf("cached projection status=%d callback calls=%d", projection.StatusCode, calls.Load())
	}

	outside := t.TempDir()
	outsideResponse, err := http.Post(httpServer.URL+"/v1/analyses", "application/json", strings.NewReader(`{"project_root":`+strconv.Quote(outside)+`}`))
	if err != nil {
		t.Fatalf("POST outside analysis root: %v", err)
	}
	outsideBody, err := readBody(outsideResponse)
	if err != nil {
		t.Fatalf("read outside analysis response: %v", err)
	}
	if outsideResponse.StatusCode != http.StatusForbidden || calls.Load() != 1 || !strings.Contains(string(outsideBody), string(analysis.ErrInvalidRequest)) {
		t.Fatalf("outside analysis response=%d calls=%d body=%s", outsideResponse.StatusCode, calls.Load(), outsideBody)
	}
}

func TestAggregateSourceProjectionDoesNotMutateCachedScope(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "nested", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run := aggregateFixtureRunAtProjectRoot(t, root, "nested")
	selected, err := run.SelectAnalysisScope("scope-fixture")
	if err != nil {
		t.Fatalf("select scope before source lookup: %v", err)
	}
	if got := selected.Model.SourceReferences[0].Path; got != "main.go" {
		t.Fatalf("initial cached source path = %q", got)
	}
	query := url.Values{"scope": []string{"scope-fixture"}, "evidence_id": []string{"source:main"}}
	if _, err := aggregateSourceModel(run, query); err != nil {
		t.Fatalf("aggregate source model: %v", err)
	}
	selected, err = run.SelectAnalysisScope("scope-fixture")
	if err != nil {
		t.Fatalf("select scope after source lookup: %v", err)
	}
	if got := selected.Model.SourceReferences[0].Path; got != "main.go" {
		t.Fatalf("source lookup mutated cached path to %q", got)
	}
}

func TestFailedCombinedReanalysisPreservesActiveRevision(t *testing.T) {
	root := t.TempDir()
	current := aggregateFixtureRun(t, root)
	failed := orchestration.AnalysisRun{SchemaVersion: orchestration.AggregateSchemaVersion, RunID: "run-failed-reanalysis", Status: analysis.StatusFailed}
	server, err := NewAggregateServer(current, ServerOptions{
		SourceRoot: root,
		ReanalyzeCombined: func(context.Context, CombinedAnalysisRequest) (orchestration.AnalysisRun, error) {
			return failed, nil
		},
	})
	if err != nil {
		t.Fatalf("NewAggregateServer() error = %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Post(httpServer.URL+"/v1/reanalysis", "application/json", strings.NewReader(`{"project_root":""}`))
	if err != nil {
		t.Fatalf("POST failed reanalysis: %v", err)
	}
	_, _ = readBody(response)
	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("failed reanalysis status = %d", response.StatusCode)
	}
	if server.analysisRunID() != current.RunID || server.lookupAggregateRun(current.RunID) == nil || server.lookupAggregateRun(failed.RunID) != nil {
		t.Fatalf("failed reanalysis replaced active run: active=%q", server.analysisRunID())
	}
}

func TestAggregateServerRetainsFailedScopeDiagnosticsInProjection(t *testing.T) {
	root := t.TempDir()
	run := aggregateFixtureRunWithFailedScope(t, root)
	server, err := NewAggregateServer(run, ServerOptions{SourceRoot: root})
	if err != nil {
		t.Fatalf("NewAggregateServer() error = %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	projectionURL := httpServer.URL + "/v1/analyses/" + url.PathEscape(run.RunID) + "/projection?scope=scope-failed"
	response, err := http.Get(projectionURL)
	if err != nil {
		t.Fatalf("GET failed scope projection: %v", err)
	}
	var projection struct {
		ScopeID         string `json:"scope_id"`
		AggregateStatus string `json:"aggregate_status"`
		ScopeStatus     string `json:"scope_status"`
		Summary         struct {
			DiagnosticCount int `json:"diagnostic_count"`
		} `json:"summary"`
		DiagnosticIndicators []struct {
			Code string `json:"code"`
		} `json:"diagnostic_indicators"`
	}
	if err := json.NewDecoder(response.Body).Decode(&projection); err != nil {
		_ = response.Body.Close()
		t.Fatalf("decode failed scope projection: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || projection.ScopeID != "scope-failed" || projection.AggregateStatus != string(analysis.StatusPartial) || projection.ScopeStatus != string(orchestration.JobFailed) {
		t.Fatalf("failed scope projection = %d %#v", response.StatusCode, projection)
	}
	if projection.Summary.DiagnosticCount == 0 || len(projection.DiagnosticIndicators) == 0 || projection.DiagnosticIndicators[0].Code != "fixture_failed" {
		t.Fatalf("failed scope diagnostics = %#v", projection)
	}
}

func aggregateFixtureRun(t *testing.T, root string) orchestration.AnalysisRun {
	return aggregateFixtureRunAtProjectRoot(t, root, ".")
}

func aggregateFixtureRunAtProjectRoot(t *testing.T, root, projectRoot string) orchestration.AnalysisRun {
	t.Helper()
	manifest := analysis.Manifest{ID: "org.archview.go", Version: "1.0.0", Language: "go", APIVersion: analysis.AnalyzerAPIVersion}
	result := analysis.AnalysisResult{
		RunID:            "scope-run",
		Status:           analysis.StatusComplete,
		Analyzer:         analysis.AnalyzerInfo{ID: manifest.ID, Version: manifest.Version, Language: manifest.Language, APIVersion: manifest.APIVersion},
		Project:          analysis.ProjectInfo{RootLabel: "fixture", Boundary: "go.mod"},
		Modules:          []analysis.ModuleObservation{{ID: "go:main", Language: "go", Kind: "package", Name: "main", DisplayName: "main", Hierarchy: []string{"main"}, SourceReferenceIDs: []string{"source:main"}}},
		SourceReferences: []analysis.SourceReference{{ID: "source:main", Path: "main.go", Kind: "file"}},
	}
	absoluteProjectRoot := root
	if projectRoot != "." {
		absoluteProjectRoot = filepath.Join(root, filepath.FromSlash(projectRoot))
	}
	job := orchestration.AnalyzerJob{ScopeID: "scope-fixture", RelativeProjectRoot: projectRoot, ProjectRoot: absoluteProjectRoot, LogicalAnalyzerID: manifest.ID, AnalyzerVersion: manifest.Version, Language: manifest.Language, Status: orchestration.JobComplete, Result: &result, Manifest: manifest}
	plan := orchestration.JobPlan{PlanVersion: orchestration.JobPlanSchemaVersion, RepositoryRoot: root, InvocationRoot: ".", DiscoveryPolicyVersion: orchestration.DiscoveryPolicyVersion, SourceScopePolicy: orchestration.SourceScopePolicy{PolicyVersion: orchestration.SourceScopePolicyVersion, InvocationRoot: "."}, Jobs: []orchestration.AnalyzerJob{job}}
	snapshot := orchestration.ExecutionSnapshot{RunID: "run-fixture", Plan: plan, Jobs: []orchestration.AnalyzerJob{job}, Events: []orchestration.JobLifecycleEvent{{Type: "run.started"}, {Type: "run.completed"}}}
	run, err := orchestration.AggregateScopeResults(snapshot)
	if err != nil {
		t.Fatalf("AggregateScopeResults() error = %v", err)
	}
	return run
}

func aggregateFixtureRunWithFailedScope(t *testing.T, root string) orchestration.AnalysisRun {
	t.Helper()
	manifest := analysis.Manifest{ID: "org.archview.go", Version: "1.0.0", Language: "go", APIVersion: analysis.AnalyzerAPIVersion}
	result := analysis.AnalysisResult{
		RunID:            "scope-run",
		Status:           analysis.StatusComplete,
		Analyzer:         analysis.AnalyzerInfo{ID: manifest.ID, Version: manifest.Version, Language: manifest.Language, APIVersion: manifest.APIVersion},
		Project:          analysis.ProjectInfo{RootLabel: "fixture", Boundary: "go.mod"},
		Modules:          []analysis.ModuleObservation{{ID: "go:main", Language: "go", Kind: "package", Name: "main", DisplayName: "main", Hierarchy: []string{"main"}, SourceReferenceIDs: []string{"source:main"}}},
		SourceReferences: []analysis.SourceReference{{ID: "source:main", Path: "main.go", Kind: "file"}},
	}
	success := orchestration.AnalyzerJob{ScopeID: "scope-success", RelativeProjectRoot: ".", ProjectRoot: root, LogicalAnalyzerID: manifest.ID, AnalyzerVersion: manifest.Version, Language: manifest.Language, Status: orchestration.JobComplete, Result: &result, Manifest: manifest}
	failure := orchestration.AnalyzerJob{ScopeID: "scope-failed", RelativeProjectRoot: "broken", ProjectRoot: filepath.Join(root, "broken"), LogicalAnalyzerID: "org.archview.rust", AnalyzerVersion: "1.0.0", Language: "rust", Status: orchestration.JobFailed, Diagnostics: []analysis.Diagnostic{{Code: "fixture_failed", Severity: "error", Message: "fixture scope failed", Subject: "org.archview.rust", Path: "broken", Recoverable: true}}}
	plan := orchestration.JobPlan{PlanVersion: orchestration.JobPlanSchemaVersion, RepositoryRoot: root, InvocationRoot: ".", DiscoveryPolicyVersion: orchestration.DiscoveryPolicyVersion, SourceScopePolicy: orchestration.SourceScopePolicy{PolicyVersion: orchestration.SourceScopePolicyVersion, InvocationRoot: "."}, Jobs: []orchestration.AnalyzerJob{success, failure}}
	snapshot := orchestration.ExecutionSnapshot{RunID: "run-failed-scope", Plan: plan, Jobs: []orchestration.AnalyzerJob{success, failure}, Events: []orchestration.JobLifecycleEvent{{Type: "run.started"}, {Type: "run.completed"}}}
	run, err := orchestration.AggregateScopeResults(snapshot)
	if err != nil {
		t.Fatalf("AggregateScopeResults() error = %v", err)
	}
	return run
}

func readBody(response *http.Response) ([]byte, error) {
	defer response.Body.Close()
	return io.ReadAll(response.Body)
}
