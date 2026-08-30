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
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/orchestration"
	goanalyzer "github.com/buffo/arch-view/internal/analyzers/go"
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/model/canonical"
)

func TestSourceIndexHTTPQueriesAndBoundedEvidence(t *testing.T) {
	root := t.TempDir()
	writeViewerSourceFixture(t, root)
	value := sourceIndexedViewerModel(t, root)
	server, err := NewServer(value, ServerOptions{SourceRoot: root})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	base := httpServer.URL + "/v1/models/" + url.PathEscape(value.ModelID) + "/source-index"

	var files sourceIndexPage[analysis.FileRecord]
	response, err := http.Get(base + "/files?limit=1")
	if err != nil {
		t.Fatalf("GET source files: %v", err)
	}
	if err := json.NewDecoder(response.Body).Decode(&files); err != nil {
		_ = response.Body.Close()
		t.Fatalf("decode source files: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || files.Total != 1 || len(files.Items) != 1 || files.Items[0].Path != "main.go" {
		t.Fatalf("source files response = %d %#v", response.StatusCode, files)
	}

	var symbols sourceIndexPage[analysis.SymbolRecord]
	response, err = http.Get(base + "/symbols?limit=1")
	if err != nil {
		t.Fatalf("GET source symbols: %v", err)
	}
	if err := json.NewDecoder(response.Body).Decode(&symbols); err != nil {
		_ = response.Body.Close()
		t.Fatalf("decode source symbols: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || symbols.Total != 1 || len(symbols.Items) != 1 || symbols.Items[0].Name != "Main" {
		t.Fatalf("source symbols response = %d %#v", response.StatusCode, symbols)
	}

	var kindSymbols sourceIndexPage[analysis.SymbolRecord]
	response, err = http.Get(base + "/symbols?language_kind=go:function")
	if err != nil {
		t.Fatalf("GET filtered source symbols: %v", err)
	}
	if err := json.NewDecoder(response.Body).Decode(&kindSymbols); err != nil {
		_ = response.Body.Close()
		t.Fatalf("decode filtered source symbols: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || kindSymbols.Total != 1 || len(kindSymbols.Items) != 1 || kindSymbols.Items[0].Name != "Main" {
		t.Fatalf("filtered source symbols response = %d %#v", response.StatusCode, kindSymbols)
	}

	var documentation sourceIndexPage[analysis.DocumentationRecord]
	response, err = http.Get(base + "/documentation?subject_id=" + url.QueryEscape(symbols.Items[0].ID))
	if err != nil {
		t.Fatalf("GET linked documentation: %v", err)
	}
	if err := json.NewDecoder(response.Body).Decode(&documentation); err != nil {
		_ = response.Body.Close()
		t.Fatalf("decode linked documentation: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || documentation.Total != 1 || len(documentation.Items) != 1 || documentation.Items[0].SubjectRef.ID != symbols.Items[0].ID {
		t.Fatalf("linked documentation response = %d %#v", response.StatusCode, documentation)
	}

	var evidence sourceFactEvidenceResponse
	response, err = http.Get(base + "/evidence/" + url.PathEscape(symbols.Items[0].ID))
	if err != nil {
		t.Fatalf("GET source evidence: %v", err)
	}
	if err := json.NewDecoder(response.Body).Decode(&evidence); err != nil {
		_ = response.Body.Close()
		t.Fatalf("decode source evidence: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || len(evidence.Spans) == 0 || evidence.SourceContext != nil {
		t.Fatalf("default source evidence = %d %#v, want bounded context omitted", response.StatusCode, evidence)
	}
	for _, documentation := range evidence.Documentation {
		if documentation.RawText != "" || documentation.NormalizedText == "" {
			t.Fatalf("default source evidence documentation detail = %#v, want normalized text without raw text", documentation)
		}
	}

	response, err = http.Get(base + "/evidence/" + url.PathEscape(symbols.Items[0].ID) + "?include_source=true")
	if err != nil {
		t.Fatalf("GET bounded source evidence: %v", err)
	}
	var bounded sourceFactEvidenceResponse
	if err := json.NewDecoder(response.Body).Decode(&bounded); err != nil {
		_ = response.Body.Close()
		t.Fatalf("decode bounded source evidence: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || bounded.SourceContext == nil || !bounded.SourceContext.ReadOnly || len(bounded.SourceContext.Lines) > maxSourceSpan {
		t.Fatalf("bounded source evidence = %d %#v", response.StatusCode, bounded)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package viewer\n\nfunc Changed() {}\n"), 0o644); err != nil {
		t.Fatalf("mutate indexed source: %v", err)
	}
	response, err = http.Get(base + "/evidence/" + url.PathEscape(symbols.Items[0].ID) + "?include_source=true")
	if err != nil {
		t.Fatalf("GET stale source evidence: %v", err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(body), string(analysis.ErrSourceIndexDigestMismatch)) {
		t.Fatalf("stale source evidence = %d %s", response.StatusCode, body)
	}

	response, err = http.Get(base + "/files?limit=201")
	if err != nil {
		t.Fatalf("GET invalid source query: %v", err)
	}
	body, _ = io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), string(analysis.ErrInvalidRequest)) {
		t.Fatalf("invalid source query = %d %s", response.StatusCode, body)
	}
}

func TestSplitPhysicalSourceLinesSupportsAllLineEndings(t *testing.T) {
	lines := splitPhysicalSourceLines([]byte("one\rtwo\r\nthree\nfour"))
	if strings.Join(lines, "|") != "one|two|three|four" {
		t.Fatalf("physical source lines = %#v", lines)
	}
}

func TestSourceIndexQueryOptionsKeepRepeatedContainmentIDs(t *testing.T) {
	options, err := sourceIndexQueryOptions(url.Values{
		"module_id":     {"module-b", " module-a ", "module-b", ""},
		"file_id":       {"file-b", "file-a", "file-a"},
		"subject_id":    {"symbol-b", " symbol-a ", "symbol-b", ""},
		"language_kind": {"go:struct"},
	})
	if err != nil {
		t.Fatalf("sourceIndexQueryOptions() error = %v", err)
	}
	if strings.Join(options.ModuleIDs, ",") != "module-b,module-a" {
		t.Fatalf("module ids = %#v", options.ModuleIDs)
	}
	if strings.Join(options.FileIDs, ",") != "file-b,file-a" {
		t.Fatalf("file ids = %#v", options.FileIDs)
	}
	if strings.Join(options.SubjectIDs, ",") != "symbol-b,symbol-a" {
		t.Fatalf("subject ids = %#v", options.SubjectIDs)
	}
	if options.SubjectID != "" {
		t.Fatalf("singular subject id = %q, want empty for repeated values", options.SubjectID)
	}
	if options.LanguageKind != "go:struct" {
		t.Fatalf("language kind = %q", options.LanguageKind)
	}
}

func TestModelHTTPCanOmitSourceIndexWithoutChangingDefault(t *testing.T) {
	root := t.TempDir()
	writeViewerSourceFixture(t, root)
	value := sourceIndexedViewerModel(t, root)
	server, err := NewServer(value, ServerOptions{SourceRoot: root})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	endpoint := httpServer.URL + "/v1/models/" + url.PathEscape(value.ModelID)

	response, err := http.Get(endpoint)
	if err != nil {
		t.Fatalf("GET default model: %v", err)
	}
	var full model.Model
	if err := json.NewDecoder(response.Body).Decode(&full); err != nil {
		_ = response.Body.Close()
		t.Fatalf("decode default model: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || full.SourceIndex == nil {
		t.Fatalf("default model response = %d source-index=%v, want source-index", response.StatusCode, full.SourceIndex != nil)
	}

	response, err = http.Get(endpoint + "?include_source_index=false")
	if err != nil {
		t.Fatalf("GET compact model: %v", err)
	}
	var compact model.Model
	if err := json.NewDecoder(response.Body).Decode(&compact); err != nil {
		_ = response.Body.Close()
		t.Fatalf("decode compact model: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || compact.SourceIndex != nil {
		t.Fatalf("compact model response = %d source-index=%v, want omitted", response.StatusCode, compact.SourceIndex != nil)
	}
}

func TestAggregateSourceIndexHTTPResolvesConcreteScopeBeforeQuerying(t *testing.T) {
	root := t.TempDir()
	writeViewerSourceFixture(t, root)
	result := sourceIndexedAnalysisResult(t, root)
	manifest := goanalyzer.New().Manifest()
	job := orchestration.AnalyzerJob{
		ScopeID:             "scope-go",
		ProjectRoot:         root,
		RelativeProjectRoot: ".",
		LogicalAnalyzerID:   manifest.ID,
		AnalyzerVersion:     manifest.Version,
		Language:            manifest.Language,
		Status:              orchestration.JobComplete,
		Result:              &result,
		Manifest:            manifest,
	}
	plan := orchestration.JobPlan{PlanVersion: orchestration.JobPlanSchemaVersion, RepositoryRoot: root, InvocationRoot: ".", DiscoveryPolicyVersion: orchestration.DiscoveryPolicyVersion, SourceScopePolicy: orchestration.SourceScopePolicy{PolicyVersion: orchestration.SourceScopePolicyVersion, InvocationRoot: "."}, Jobs: []orchestration.AnalyzerJob{job}}
	run, err := orchestration.AggregateScopeResults(orchestration.ExecutionSnapshot{RunID: "run-source-index", Plan: plan, Jobs: []orchestration.AnalyzerJob{job}})
	if err != nil {
		t.Fatalf("aggregate source-index fixture: %v", err)
	}
	server, err := NewAggregateServer(run, ServerOptions{SourceRoot: root})
	if err != nil {
		t.Fatalf("NewAggregateServer() error = %v", err)
	}
	if run.Model == nil {
		t.Fatalf("aggregate run has no combined model: status=%q scopes=%#v diagnostics=%#v", run.Status, run.Scopes, run.Diagnostics)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	base := httpServer.URL + "/v1/models/" + url.PathEscape(run.Model.ModelID) + "/source-index/files"
	response, err := http.Get(base + "?scope=scope-go")
	if err != nil {
		t.Fatalf("GET concrete aggregate source scope: %v", err)
	}
	var files sourceIndexPage[analysis.FileRecord]
	if err := json.NewDecoder(response.Body).Decode(&files); err != nil {
		_ = response.Body.Close()
		t.Fatalf("decode concrete aggregate source scope: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || files.ScopeID == "" || len(files.Items) != 1 || files.Items[0].Path != "main.go" {
		t.Fatalf("concrete aggregate source scope = %d %#v", response.StatusCode, files)
	}
}

func TestLegacyModelSourceIndexRemainsOptional(t *testing.T) {
	value := fixtureModel(t)
	server, err := NewServer(value)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	response, err := http.Get(httpServer.URL + "/v1/models/" + url.PathEscape(value.ModelID) + "/source-index")
	if err != nil {
		t.Fatalf("GET legacy source-index: %v", err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNotFound || !strings.Contains(string(body), string(analysis.ErrSourceScopeUnavailable)) {
		t.Fatalf("legacy source-index response = %d %s", response.StatusCode, body)
	}
}

type sourceIndexPage[T any] struct {
	SnapshotID string `json:"snapshot_id"`
	ScopeID    string `json:"scope_id"`
	Items      []T    `json:"items"`
	Total      int    `json:"total"`
}

func writeViewerSourceFixture(t *testing.T, root string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/viewer\ngo 1.22\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package viewer\n\n// Main is the viewer entrypoint.\nfunc Main() {}\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
}

func sourceIndexedViewerModel(t *testing.T, root string) model.Model {
	t.Helper()
	return canonicalModelFromSourceResult(t, sourceIndexedAnalysisResult(t, root))
}

func sourceIndexedAnalysisResult(t *testing.T, root string) analysis.AnalysisResult {
	t.Helper()
	options, err := analysis.ResolveOptions(goanalyzer.New().Manifest(), nil, nil)
	if err != nil {
		t.Fatalf("resolve Go options: %v", err)
	}
	result, err := goanalyzer.New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: options})
	if err != nil {
		t.Fatalf("analyze source fixture: %v", err)
	}
	if result.SourceIndex == nil {
		t.Fatal("source fixture has no source-index attachment")
	}
	result.RunID = "scope-viewer"
	return result
}

func canonicalModelFromSourceResult(t *testing.T, result analysis.AnalysisResult) model.Model {
	t.Helper()
	value, err := canonical.Normalize(result)
	if err != nil {
		t.Fatalf("normalize source fixture: %v", err)
	}
	return value
}
