package live

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	clojureanalyzer "github.com/buffo/arch-view/internal/analyzers/clojure"
	goanalyzer "github.com/buffo/arch-view/internal/analyzers/go"
	pyanalyzer "github.com/buffo/arch-view/internal/analyzers/python"
	rustanalyzer "github.com/buffo/arch-view/internal/analyzers/rust"
	tsAnalyzer "github.com/buffo/arch-view/internal/analyzers/typescript"
	"github.com/buffo/arch-view/internal/quality"
)

func TestMixedAnalyzerLiveSessionExposesLanguageNeutralScopes(t *testing.T) {
	root := t.TempDir()
	writeCrossAnalyzerFixture(t, root)
	registry := analysis.NewRegistry()
	for _, analyzer := range []analysis.Analyzer{
		clojureanalyzer.New(),
		goanalyzer.New(),
		pyanalyzer.New(),
		rustanalyzer.New(),
		tsAnalyzer.New(),
	} {
		if err := registry.Register(analyzer); err != nil {
			t.Fatalf("register %s: %v", analyzer.Manifest().ID, err)
		}
	}

	config := testLiveConfig("session:mixed-analyzers")
	config.SourceIndexRequest = SourceIndexRequest{Enabled: true}
	config.FreshnessPolicy.MaxWaitMS = 30_000
	session, err := StartLiveSession(context.Background(), config, root, SessionOptions{
		AnalyzerRegistry: registry,
		StartWatcher:     false,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	if err := session.Wait(context.Background()); err != nil {
		t.Fatalf("mixed analyzer session: %v", err)
	}

	status := session.Status()
	if status.Snapshot == nil || status.Snapshot.Revision != 1 {
		t.Fatalf("mixed analyzer status = %#v", status)
	}
	envelope, err := session.ListScopes(context.Background(), QueryRequest{Consistency: ConsistencySpecific, Revision: 1, MaxItems: 100, MaxBytes: 1 << 20})
	if err != nil {
		t.Fatalf("list mixed scopes: %v", err)
	}
	page, ok := envelope.Result.(ItemPage[ScopeInfo])
	if !ok {
		t.Fatalf("scope result type = %T", envelope.Result)
	}
	languages := make(map[string]bool, len(page.Items))
	capableScopes := 0
	unsupportedScopes := 0
	unsupportedScopeID := ""
	for _, item := range page.Items {
		languages[item.Language] = true
		if len(item.Capabilities) > 0 {
			capableScopes++
		} else {
			unsupportedScopes++
			if unsupportedScopeID == "" {
				unsupportedScopeID = item.ScopeID
			}
		}
	}
	for _, language := range []string{"clojure", "go", "python", "rust", "typescript"} {
		if !languages[language] {
			t.Errorf("mixed session omitted %s scope; scopes=%#v", language, page.Items)
		}
	}
	if len(page.Items) != 5 {
		t.Fatalf("mixed session scopes = %d, want 5: %#v", len(page.Items), page.Items)
	}
	if capableScopes == 0 || unsupportedScopes == 0 {
		t.Fatalf("mixed session did not expose both capable and unsupported source surfaces: %#v", page.Items)
	}
	unsupportedFiles, err := session.FindFiles(context.Background(), QueryRequest{Consistency: ConsistencySpecific, Revision: 1, Query: StructuralQuery{ScopeIDs: []string{unsupportedScopeID}}, MaxItems: 100, MaxBytes: 1 << 20})
	if err != nil {
		t.Fatalf("query unsupported mixed scope: %v", err)
	}
	unsupportedCoverage := false
	for _, coverage := range unsupportedFiles.Capabilities {
		if coverage.Status == "unsupported" || coverage.Status == "unavailable" || coverage.Status == "unknown" {
			unsupportedCoverage = true
			break
		}
	}
	if !unsupportedCoverage {
		t.Fatalf("unsupported mixed scope did not expose explicit coverage: %#v", unsupportedFiles.Capabilities)
	}

	files, err := session.FindFiles(context.Background(), QueryRequest{Consistency: ConsistencySpecific, Revision: 1, MaxItems: 100, MaxBytes: 1 << 20})
	if err != nil {
		t.Fatalf("find mixed files: %v", err)
	}
	filePage, ok := files.Result.(ItemPage[analysis.FileRecord])
	if !ok || len(filePage.Items) == 0 {
		t.Fatalf("mixed file result = %#v", files.Result)
	}
	if len(files.Capabilities) == 0 || files.Freshness.Status != FreshnessCurrent {
		t.Fatalf("mixed file envelope lost coverage or freshness: %#v", files)
	}
}

func TestMixedAnalyzerAgentLoopReconcilesExternalEditAndComparesQuality(t *testing.T) {
	root := t.TempDir()
	writeCrossAnalyzerFixture(t, root)
	registry := analysis.NewRegistry()
	for _, analyzer := range []analysis.Analyzer{clojureanalyzer.New(), goanalyzer.New(), pyanalyzer.New(), rustanalyzer.New(), tsAnalyzer.New()} {
		if err := registry.Register(analyzer); err != nil {
			t.Fatal(err)
		}
	}
	profile := policyThresholdProfile()
	config := testLiveConfig("session:mixed-agent-loop")
	config.SourceIndexRequest = SourceIndexRequest{Enabled: true}
	config.QualityRequest = &QualityRequest{ProfileID: profile.ProfileID, ProfileVersion: profile.ProfileVersion}
	config.FreshnessPolicy.MaxWaitMS = 30_000
	session, err := StartLiveSession(context.Background(), config, root, SessionOptions{
		AnalyzerRegistry: registry,
		QualityCatalog:   quality.NewDefaultCatalog(),
		Profiles:         NewMemoryQualityProfileResolver(profile),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	if err := session.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}

	symbols, err := session.FindSymbols(context.Background(), QueryRequest{Consistency: ConsistencySpecific, Revision: 1, Query: StructuralQuery{Name: "Main"}, Projection: []string{"locations"}, MaxItems: 10, MaxBytes: 64 * 1024})
	if err != nil {
		t.Fatal(err)
	}
	symbolPage, ok := symbols.Result.(ItemPage[analysis.SymbolRecord])
	if !ok || len(symbolPage.Items) != 1 {
		t.Fatalf("Main symbol query = %#v", symbols.Result)
	}
	textMatches, err := session.FindText(context.Background(), TextSearchQuery{Consistency: ConsistencySpecific, Revision: 1, Pattern: "return 1", Mode: "literal", MaxItems: 10, MaxBytes: 64 * 1024})
	if err != nil {
		t.Fatal(err)
	}
	textPage, ok := textMatches.Result.(ItemPage[TextMatch])
	if !ok || len(textPage.Items) == 0 {
		t.Fatalf("exact text query = %#v", textMatches.Result)
	}
	contextEnvelope, err := session.GetSourceContext(context.Background(), SourceContextRequest{Consistency: ConsistencySpecific, Revision: 1, EntityID: symbolPage.Items[0].ID, MaxLines: 5, MaxBytes: 1024})
	if err != nil {
		t.Fatal(err)
	}
	contextResult, ok := contextEnvelope.Result.(SourceContextResult)
	if !ok || contextResult.Context.Path != "main.go" || contextResult.Context.Content == "" {
		t.Fatalf("bounded source context = %#v", contextEnvelope.Result)
	}

	previousFindings, err := session.QualityGateway().GetQualityFindings(context.Background(), QualityFindingsRequest{QueryRequest: QueryRequest{Consistency: ConsistencySpecific, Revision: 1, MaxItems: 100, MaxBytes: 1 << 20}})
	if err != nil {
		t.Fatal(err)
	}
	previousPage, ok := previousFindings.Result.(QualityFindingsResult)
	if !ok || len(previousPage.Items) == 0 {
		t.Fatalf("initial findings = %#v", previousFindings.Result)
	}

	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main; func Main() int { return 1 }"), 0o644); err != nil {
		t.Fatal(err)
	}
	current, err := session.EnsureCurrentSnapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if current.Snapshot.Revision != 2 || current.Snapshot.InputVerification.Status != "verified" {
		t.Fatalf("current revision = %#v", current.Snapshot)
	}
	comparisonEnvelope, err := session.QualityGateway().CompareQualityReports(context.Background(), QualityCompareRequest{PreviousRevision: 1, CurrentRevision: 2, MaxItems: 100, MaxBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	comparisonResult, ok := comparisonEnvelope.Result.(QualityComparisonResult)
	if !ok || comparisonResult.Comparison == nil || len(comparisonResult.Comparison.Resolved) == 0 {
		t.Fatalf("quality comparison = %#v", comparisonEnvelope.Result)
	}
}

func writeCrossAnalyzerFixture(t *testing.T, root string) {
	t.Helper()
	files := map[string]string{
		"go.mod":               "module example.com/mixed\ngo 1.22\n",
		"main.go":              "package main\n\nfunc Main() int { return 1 }\n",
		"pyproject.toml":       "[project]\nname = 'mixed-python'\n",
		"service.py":           "def service():\n    return 1\n",
		"tsconfig.json":        "{\"compilerOptions\":{\"target\":\"ES2020\"}}\n",
		"index.ts":             "export const answer: number = 42;\n",
		"Cargo.toml":           "[package]\nname = 'mixed-rust'\nversion = '0.1.0'\nedition = '2021'\n",
		"src/lib.rs":           "pub fn answer() -> i32 { 42 }\n",
		"deps.edn":             "{:paths [\"clj-src\"]}\n",
		"clj-src/app/core.clj": "(ns app.core)\n(defn answer [] 42)\n",
	}
	for relative, content := range files {
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create fixture directory for %s: %v", relative, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write fixture %s: %v", relative, err)
		}
	}
}
