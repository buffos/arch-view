package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/orchestration"
	"github.com/buffo/arch-view/internal/viewer"
)

type configuredScopeAnalyzer struct {
	manifest analysis.Manifest
	calls    atomic.Int32
	detects  atomic.Int32
}

func (a *configuredScopeAnalyzer) Manifest() analysis.Manifest { return a.manifest }

func (a *configuredScopeAnalyzer) Detect(_ context.Context, request analysis.DetectRequest) (analysis.DetectionCandidate, error) {
	a.detects.Add(1)
	for _, marker := range a.manifest.DetectionMarkers {
		if info, err := os.Stat(filepath.Join(request.ProjectRoot, marker.Value)); err == nil && !info.IsDir() {
			return analysis.DetectionCandidate{
				AnalyzerID:     a.manifest.ID,
				Confidence:     marker.Weight,
				MatchedMarkers: []string{marker.Value},
				Reason:         "configured fixture marker",
			}, nil
		}
	}
	return analysis.DetectionCandidate{AnalyzerID: a.manifest.ID, Reason: "configured fixture marker"}, nil
}

func (a *configuredScopeAnalyzer) Analyze(_ context.Context, request analysis.AnalyzeRequest) (analysis.AnalysisResult, error) {
	a.calls.Add(1)
	sourcePath := "main.go"
	if a.manifest.Language == "typescript" {
		sourcePath = "src/main.ts"
	}
	return analysis.AnalysisResult{
		RunID:              a.manifest.ID + "-run",
		Status:             analysis.StatusComplete,
		Analyzer:           analysis.AnalyzerInfo{ID: a.manifest.ID, Version: a.manifest.Version, Language: a.manifest.Language, APIVersion: a.manifest.APIVersion},
		Project:            analysis.ProjectInfo{RootLabel: filepath.Base(request.ProjectRoot), Boundary: a.manifest.DetectionMarkers[0].Value},
		OptionsFingerprint: request.Options.Fingerprint,
		Modules: []analysis.ModuleObservation{{
			ID: "module", Language: a.manifest.Language, Kind: "module", Name: "module", DisplayName: "module",
			Hierarchy: []string{"module"}, SourceReferenceIDs: []string{"source"},
		}},
		SourceReferences: []analysis.SourceReference{{ID: "source", Path: sourcePath, Kind: "definition"}},
	}, nil
}

func TestConfiguredMixedAnalysisExposesScopesCacheAndHTTPRecovery(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, "go.mod", "module configured.fixture\n")
	writeCLIFile(t, root, "main.go", "package main\n")
	writeCLIFile(t, root, "frontend/tsconfig.json", "{}\n")
	writeCLIFile(t, root, "frontend/src/main.ts", "export const value = 1;\n")
	writeCLIFile(t, root, "generated/ignored.go", "package generated\n")
	if err := os.MkdirAll(filepath.Join(root, "missing"), 0o755); err != nil {
		t.Fatalf("create unavailable assignment root: %v", err)
	}
	goAnalyzer := &configuredScopeAnalyzer{manifest: configuredScopeManifest("org.example.configured-go", "go", "go.mod")}
	typeScriptAnalyzer := &configuredScopeAnalyzer{manifest: configuredScopeManifest("org.example.configured-typescript", "typescript", "tsconfig.json")}
	registry := analysis.NewRegistry()
	for _, analyzer := range []*configuredScopeAnalyzer{goAnalyzer, typeScriptAnalyzer} {
		if err := registry.Register(analyzer); err != nil {
			t.Fatalf("register %s: %v", analyzer.manifest.ID, err)
		}
	}
	host := analysis.NewHost(registry)
	config := `{
  "schema_version": "arch-view.config/v2",
  "layout": {"algorithm": "layered", "options": {}},
  "analysis": {
    "exclude": ["generated/**"],
    "include": [
      {"analyzer_id": "org.example.configured-go", "globs": ["**/*.go"]},
      {"analyzer_id": "org.example.configured-typescript", "globs": ["frontend/src/**"]}
    ],
    "assignments": [
      {"path": ".", "analyzer_id": "org.example.configured-go", "options": {}},
      {"path": "frontend", "analyzer_id": "org.example.configured-typescript", "options": {}},
      {"path": "missing", "analyzer_id": "org.example.missing", "options": {}}
    ]
  }
}`
	if err := os.WriteFile(filepath.Join(root, ".archview.json"), []byte(config), 0o644); err != nil {
		t.Fatalf("write v2 configuration: %v", err)
	}

	cache := orchestration.NewSessionCache()
	first, err := runCombinedAnalysisWithPolicyAndCache(context.Background(), host, root, map[string]any{}, orchestration.SourceScopePolicy{}, cache)
	if err != nil {
		t.Fatalf("configured combined analysis: %v", err)
	}
	if first.Status != analysis.StatusPartial || len(first.Scopes) != 3 || first.Summary.UsableScopeCount != 2 || first.Summary.FailedScopeCount != 1 {
		t.Fatalf("configured aggregate = %#v", first)
	}
	assertConfiguredScope(t, first, ".", "org.example.configured-go", orchestration.SelectionAssignment, []string{"main.go"})
	assertConfiguredScope(t, first, "frontend", "org.example.configured-typescript", orchestration.SelectionAssignment, []string{"frontend/src/main.ts"})
	missing := findConfiguredScope(t, first, "missing", "org.example.missing")
	if missing.Status != orchestration.JobFailed || !hasDiagnosticCode(missing.Diagnostics, string(analysis.ErrAssignmentAnalyzerUnavailable)) {
		t.Fatalf("unavailable configured scope = %#v", missing)
	}
	if goAnalyzer.calls.Load() != 1 || typeScriptAnalyzer.calls.Load() != 1 {
		t.Fatalf("initial configured analyzer calls = go:%d typescript:%d", goAnalyzer.calls.Load(), typeScriptAnalyzer.calls.Load())
	}

	second, err := runCombinedAnalysisWithPolicyAndCache(context.Background(), host, root, map[string]any{}, orchestration.SourceScopePolicy{}, cache)
	if err != nil {
		t.Fatalf("cached configured combined analysis: %v", err)
	}
	if goAnalyzer.calls.Load() != 1 || typeScriptAnalyzer.calls.Load() != 1 {
		t.Fatalf("cached configured analyzer calls = go:%d typescript:%d", goAnalyzer.calls.Load(), typeScriptAnalyzer.calls.Load())
	}
	for _, scope := range second.Scopes {
		if !scope.CacheHit {
			t.Fatalf("configured scope was not reused: %#v", scope)
		}
	}

	server, err := viewer.NewAggregateServer(second, viewer.ServerOptions{
		SourceRoot: root,
		ReanalyzeCombined: func(ctx context.Context, request viewer.CombinedAnalysisRequest) (orchestration.AnalysisRun, error) {
			return runCombinedAnalysisWithPolicyAndCache(ctx, host, request.ProjectRoot, request.CLIOptions, request.SourceScopePolicy, cache)
		},
	})
	if err != nil {
		t.Fatalf("create configured aggregate viewer: %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	runURL := httpServer.URL + "/v1/analyses/" + url.PathEscape(second.RunID)
	scopesResponse, err := http.Get(runURL + "/scopes")
	if err != nil {
		t.Fatalf("GET configured scopes: %v", err)
	}
	var scopes struct {
		Scopes []orchestration.ScopeSummary `json:"scopes"`
	}
	if decodeErr := json.NewDecoder(scopesResponse.Body).Decode(&scopes); decodeErr != nil {
		_ = scopesResponse.Body.Close()
		t.Fatalf("decode configured scopes: %v", decodeErr)
	}
	_ = scopesResponse.Body.Close()
	if scopesResponse.StatusCode != http.StatusOK || len(scopes.Scopes) != 3 {
		t.Fatalf("configured scopes response = %d %#v", scopesResponse.StatusCode, scopes)
	}
	failedScope := findConfiguredScope(t, second, "missing", "org.example.missing")
	failedProjection, err := http.Get(runURL + "/projection?scope=" + url.QueryEscape(failedScope.ScopeID))
	if err != nil {
		t.Fatalf("GET unavailable scope projection: %v", err)
	}
	failedBody, readErr := io.ReadAll(failedProjection.Body)
	_ = failedProjection.Body.Close()
	if readErr != nil {
		t.Fatalf("read unavailable scope projection: %v", readErr)
	}
	if failedProjection.StatusCode != http.StatusOK || !strings.Contains(string(failedBody), string(analysis.ErrAssignmentAnalyzerUnavailable)) {
		t.Fatalf("unavailable scope projection = %d %s", failedProjection.StatusCode, failedBody)
	}

	reanalysisResponse, err := http.Post(httpServer.URL+"/v1/reanalysis", "application/json", strings.NewReader(`{"project_root":""}`))
	if err != nil {
		t.Fatalf("POST configured reanalysis: %v", err)
	}
	var reanalysis struct {
		Scopes []orchestration.ScopeSummary `json:"scopes"`
	}
	if decodeErr := json.NewDecoder(reanalysisResponse.Body).Decode(&reanalysis); decodeErr != nil {
		_ = reanalysisResponse.Body.Close()
		t.Fatalf("decode configured reanalysis: %v", decodeErr)
	}
	_ = reanalysisResponse.Body.Close()
	if reanalysisResponse.StatusCode != http.StatusOK || len(reanalysis.Scopes) != 3 {
		t.Fatalf("configured reanalysis = %d %#v", reanalysisResponse.StatusCode, reanalysis)
	}
	for _, scope := range reanalysis.Scopes {
		if !scope.CacheHit {
			t.Fatalf("reanalysis did not report cache hit: %#v", scope)
		}
	}
}

func TestConfiguredSingleAnalysisDoesNotExecuteNestedAssignments(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, "go.mod", "module configured.fixture\n")
	writeCLIFile(t, root, "main.go", "package main\n")
	writeCLIFile(t, root, "frontend/tsconfig.json", "{}\n")
	writeCLIFile(t, root, "frontend/src/main.ts", "export const value = 1;\n")
	goAnalyzer := &configuredScopeAnalyzer{manifest: configuredScopeManifest("org.example.configured-go", "go", "go.mod")}
	typeScriptAnalyzer := &configuredScopeAnalyzer{manifest: configuredScopeManifest("org.example.configured-typescript", "typescript", "tsconfig.json")}
	registry := analysis.NewRegistry()
	for _, analyzer := range []*configuredScopeAnalyzer{goAnalyzer, typeScriptAnalyzer} {
		if err := registry.Register(analyzer); err != nil {
			t.Fatalf("register %s: %v", analyzer.manifest.ID, err)
		}
	}
	config := `{
  "schema_version": "arch-view.config/v2",
  "layout": {"algorithm": "layered", "options": {}},
  "analysis": {"assignments": [
    {"path": ".", "analyzer_id": "org.example.configured-go", "options": {}},
    {"path": "frontend", "analyzer_id": "org.example.configured-typescript", "options": {}}
  ]}
}`
	writeCLIFile(t, root, ".archview.json", config)
	result, err := runConfiguredSingleAnalysis(context.Background(), analysis.NewHost(registry), root, goAnalyzer.manifest.ID, "go", map[string]any{})
	if err != nil {
		t.Fatalf("configured single analysis: %v", err)
	}
	if result.Analyzer.ID != goAnalyzer.manifest.ID || goAnalyzer.calls.Load() != 1 || typeScriptAnalyzer.calls.Load() != 0 || typeScriptAnalyzer.detects.Load() != 0 {
		t.Fatalf("single analysis broadened into configured nested jobs: result=%#v calls=go:%d typescript:%d detects=typescript:%d", result.Analyzer, goAnalyzer.calls.Load(), typeScriptAnalyzer.calls.Load(), typeScriptAnalyzer.detects.Load())
	}
}

func TestRequestedSourcePolicyCannotBroadenConfiguredInclude(t *testing.T) {
	configured := orchestration.SourceScopePolicy{
		Include: []orchestration.AnalyzerIncludeRule{{AnalyzerID: "org.example.go", Globs: []string{"src/**"}}},
	}
	requested := orchestration.SourceScopePolicy{
		Include: []orchestration.AnalyzerIncludeRule{
			{AnalyzerID: "org.example.go", Globs: []string{"generated/**"}},
			{AnalyzerID: "org.example.typescript", Globs: []string{"frontend/**"}},
		},
	}
	merged, err := mergeCombinedSourcePolicies(configured, requested, ".")
	if err != nil {
		t.Fatalf("merge source policies: %v", err)
	}
	if len(merged.Include) != 2 || merged.Include[0].AnalyzerID != "org.example.go" || !sameConfiguredStrings(merged.Include[0].Globs, []string{"src/**"}) {
		t.Fatalf("configured include was broadened: %#v", merged.Include)
	}
	if merged.Include[1].AnalyzerID != "org.example.typescript" || !sameConfiguredStrings(merged.Include[1].Globs, []string{"frontend/**"}) {
		t.Fatalf("unconfigured requested include was not retained: %#v", merged.Include)
	}
}

func configuredScopeManifest(id, language, marker string) analysis.Manifest {
	return analysis.Manifest{
		ID: id, Version: "1.0.0", Language: language, APIVersion: analysis.AnalyzerAPIVersion,
		DetectionMarkers: []analysis.DetectionMarker{{Kind: "file", Value: marker, Weight: 1}},
		Capabilities:     []string{"detect"}, Options: []analysis.OptionDescriptor{},
	}
}

func assertConfiguredScope(t *testing.T, run orchestration.AnalysisRun, root, analyzerID string, source orchestration.SelectionSource, matched []string) {
	t.Helper()
	scope := findConfiguredScope(t, run, root, analyzerID)
	if scope.SelectionSource != source || scope.Status != orchestration.JobComplete {
		t.Fatalf("configured scope %s/%s = %#v", root, analyzerID, scope)
	}
	if scope.SourceScope.MatchedSourceSetFingerprint == "" || scope.EffectiveOptionsFingerprint == "" {
		t.Fatalf("configured scope fingerprints = %#v", scope)
	}
	job, ok := findConfiguredJob(run, scope.ScopeID)
	if !ok || !sameConfiguredStrings(job.EffectiveSourceScope.MatchedPaths, matched) {
		t.Fatalf("configured scope source paths = %#v, want %#v", job.EffectiveSourceScope.MatchedPaths, matched)
	}
}

func sameConfiguredStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func findConfiguredScope(t *testing.T, run orchestration.AnalysisRun, root, analyzerID string) orchestration.ScopeSummary {
	t.Helper()
	for _, scope := range run.Scopes {
		if scope.ProjectRoot == root && scope.Analyzer.ID == analyzerID {
			return scope
		}
	}
	t.Fatalf("configured scope %s/%s was not found: %#v", root, analyzerID, run.Scopes)
	return orchestration.ScopeSummary{}
}

func findConfiguredJob(run orchestration.AnalysisRun, scopeID string) (orchestration.AnalyzerJob, bool) {
	for _, job := range run.JobPlan.Jobs {
		if job.ScopeID == scopeID {
			return job, true
		}
	}
	return orchestration.AnalyzerJob{}, false
}

func hasDiagnosticCode(values []analysis.Diagnostic, code string) bool {
	for _, value := range values {
		if value.Code == code {
			return true
		}
	}
	return false
}
