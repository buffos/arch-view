package sourceindex_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/sourceindex"
	gosyntax "github.com/buffo/arch-view/internal/analysis/syntax/go"
	"github.com/buffo/arch-view/internal/analyzers/go/sourcefacts"
)

type fixtureExtractor struct {
	id      string
	version string
	batch   sourceindex.FactBatch
}

func (value fixtureExtractor) ID() string      { return value.id }
func (value fixtureExtractor) Version() string { return value.version }
func (value fixtureExtractor) Capabilities() []analysis.CapabilityDescriptor {
	return []analysis.CapabilityDescriptor{{ID: sourceindex.CapabilityDeclarations, Version: "v1", SupportedLanguages: []string{"go"}}}
}
func (value fixtureExtractor) Extract(sourceindex.SourceFactInput) (sourceindex.FactBatch, error) {
	return value.batch, nil
}

func TestBuildSourceIndexRetainsRawFileFactsAndGoStructure(t *testing.T) {
	registry := sourcefacts.NewRegistry()
	input := sourceindex.BuildInput{
		Scope:    analysis.ScopeContext{ScopeID: "scope-test", ProjectRoot: ".", Mode: analysis.SourceIndexScopeMode},
		Producer: analysis.ProducerContext{AnalyzerID: "org.archview.go", AnalyzerVersion: "1.0.0"},
		Files: []sourceindex.SourceFileInput{
			{Path: "pkg/main.go", Content: []byte("// Main runs the application.\r\npackage app\r\n\r\nconst Answer = 42\r\n\r\nfunc Main() {}\r\n"), Language: analysis.LanguageRef{ID: "language:go"}, Roles: []string{"role:source"}, ModuleID: "example.com/app"},
			{Path: "pkg/invalid.go", Content: []byte{0xff, 0xfe, '\n'}, Language: analysis.LanguageRef{ID: "language:go"}, Roles: []string{"role:source"}, AnalysisStatus: analysis.FileAnalysisUnparsed, ModuleID: "example.com/app"},
		},
		RequestedCapabilities: []string{sourceindex.CapabilityDeclarations, sourceindex.CapabilityDocumentation, sourceindex.CapabilityFiles, sourceindex.CapabilitySize, sourceindex.CapabilityVisibility},
		SyntaxProvider:        gosyntax.NewProvider(),
		Extractors:            registry,
	}
	first, diagnostics, err := sourceindex.BuildSourceIndex(context.Background(), input)
	if err != nil {
		t.Fatalf("build source index: %v", err)
	}
	if len(diagnostics) == 0 {
		t.Fatal("invalid source should retain a recoverable parse diagnostic")
	}
	if len(first.Snapshots) != 1 {
		t.Fatalf("snapshots = %d, want one", len(first.Snapshots))
	}
	snapshot := first.Snapshots[0]
	if len(snapshot.Files) != 2 {
		t.Fatalf("files = %d, want two", len(snapshot.Files))
	}
	var mainFile, invalidFile analysis.FileRecord
	for _, file := range snapshot.Files {
		switch file.Path {
		case "pkg/main.go":
			mainFile = file
		case "pkg/invalid.go":
			invalidFile = file
		}
	}
	if mainFile.Size.LineCount != 6 || mainFile.Size.ByteCount == 0 || mainFile.Size.ContentHash.Value == "" {
		t.Fatalf("main file size facts = %#v", mainFile.Size)
	}
	if invalidFile.AnalysisStatus != analysis.FileAnalysisUnparsed && invalidFile.AnalysisStatus != analysis.FileAnalysisPartial {
		t.Fatalf("invalid file status = %q, want unparsed or partial", invalidFile.AnalysisStatus)
	}
	if len(snapshot.Symbols) < 2 {
		t.Fatalf("symbols = %#v, want const and function declarations", snapshot.Symbols)
	}
	if len(snapshot.Documentation) < len(snapshot.Symbols) {
		t.Fatalf("documentation records = %d, symbols = %d", len(snapshot.Documentation), len(snapshot.Symbols))
	}
	if len(snapshot.Relations) < len(snapshot.Symbols)+1 {
		t.Fatalf("relations = %d, want module/file plus file/symbol relations", len(snapshot.Relations))
	}
	if err := analysis.ValidateSourceIndex(first); err != nil {
		t.Fatalf("validate source index: %v", err)
	}
}

func TestBuildSourceIndexIsDeterministicAndQueriesFollowContainment(t *testing.T) {
	input := sourceindex.BuildInput{
		Scope:                 analysis.ScopeContext{ScopeID: "scope-query", ProjectRoot: ".", Mode: analysis.SourceIndexScopeMode},
		Producer:              analysis.ProducerContext{AnalyzerID: "org.archview.go", AnalyzerVersion: "1.0.0"},
		Files:                 []sourceindex.SourceFileInput{{Path: "main.go", Content: []byte("package main\n\n// Main docs\nfunc Main() {}\n"), Language: analysis.LanguageRef{ID: "language:go"}, Roles: []string{"role:source"}, ModuleID: "example.com/app"}},
		RequestedCapabilities: []string{sourceindex.CapabilityDeclarations, sourceindex.CapabilityDocumentation, sourceindex.CapabilityFiles, sourceindex.CapabilitySize, sourceindex.CapabilityVisibility},
		SyntaxProvider:        gosyntax.NewProvider(),
		Extractors:            sourcefacts.NewRegistry(),
	}
	first, _, err := sourceindex.BuildSourceIndex(context.Background(), input)
	if err != nil {
		t.Fatalf("first build: %v", err)
	}
	second, _, err := sourceindex.BuildSourceIndex(context.Background(), input)
	if err != nil {
		t.Fatalf("second build: %v", err)
	}
	firstJSON, _ := json.Marshal(first)
	secondJSON, _ := json.Marshal(second)
	if string(firstJSON) != string(secondJSON) {
		t.Fatalf("equal source inputs produced different source-index payloads")
	}
	service := sourceindex.NewQueryService(&first)
	inspected, err := service.InspectModule("example.com/app", sourceindex.QueryOptions{})
	if err != nil {
		t.Fatalf("inspect module: %v", err)
	}
	if len(inspected.Files) != 1 || len(inspected.Symbols) != 1 {
		t.Fatalf("module inspection = files %d symbols %d, want one each", len(inspected.Files), len(inspected.Symbols))
	}
	if len(inspected.Documentation) != 1 || inspected.Documentation[0].RawText != "" || inspected.Documentation[0].NormalizedText != "Main docs" {
		t.Fatalf("default module documentation = %#v, want normalized text without raw text", inspected.Documentation)
	}
	withText, err := service.InspectModule("example.com/app", sourceindex.QueryOptions{IncludeDocumentationText: true})
	if err != nil || len(withText.Documentation) != 1 || withText.Documentation[0].RawText != "Main docs" {
		t.Fatalf("module documentation with explicit text = %#v, err=%v", withText.Documentation, err)
	}
	page, err := service.FindSymbols(sourceindex.QueryOptions{Limit: 1})
	if err != nil {
		t.Fatalf("find symbols: %v", err)
	}
	if len(page.Items) != 1 || page.NextCursor != "" {
		t.Fatalf("symbol page = %#v, want one item and no continuation", page)
	}
	evidence, err := service.GetEvidence(page.Items[0].ID, sourceindex.QueryOptions{})
	if err != nil || len(evidence.Spans) == 0 {
		t.Fatalf("symbol evidence = %#v, err=%v", evidence, err)
	}
}

func TestCombinedProjectionKeepsEqualPathsScopeQualified(t *testing.T) {
	build := func(scope string) analysis.SourceIndexSnapshot {
		index, _, err := sourceindex.BuildSourceIndex(context.Background(), sourceindex.BuildInput{
			Scope:                 analysis.ScopeContext{ScopeID: scope, ProjectRoot: ".", Mode: analysis.SourceIndexScopeMode},
			Producer:              analysis.ProducerContext{AnalyzerID: "org.archview.go", AnalyzerVersion: "1.0.0"},
			Files:                 []sourceindex.SourceFileInput{{Path: "main.go", Content: []byte("package main\nfunc Main() {}\n"), Language: analysis.LanguageRef{ID: "language:go"}, Roles: []string{"role:source"}, ModuleID: "example.com/app"}},
			RequestedCapabilities: []string{sourceindex.CapabilityDeclarations, sourceindex.CapabilityDocumentation, sourceindex.CapabilityFiles, sourceindex.CapabilitySize, sourceindex.CapabilityVisibility},
			SyntaxProvider:        gosyntax.NewProvider(),
			Extractors:            sourcefacts.NewRegistry(),
		})
		if err != nil {
			t.Fatalf("build %s: %v", scope, err)
		}
		namespaced, err := sourceindex.NamespaceScopeSnapshot(index.Snapshots[0], scope, scope)
		if err != nil {
			t.Fatalf("namespace %s: %v", scope, err)
		}
		return namespaced
	}
	snapshots := []analysis.SourceIndexSnapshot{build("scope-b"), build("scope-a")}
	for _, snapshot := range snapshots {
		if err := analysis.ValidateSourceIndex(analysis.SourceIndex{SchemaVersion: analysis.SourceIndexSchemaVersion, Snapshots: []analysis.SourceIndexSnapshot{snapshot}, Extensions: []analysis.ExtensionBlock{}}); err != nil {
			t.Fatalf("validate source snapshot %s: %v", snapshot.ScopeContext.ScopeID, err)
		}
	}
	projection, err := sourceindex.BuildCombinedProjection(snapshots)
	if err != nil {
		t.Fatalf("combined projection: %v", err)
	}
	if projection == nil || len(projection.Files) != 2 || len(projection.Symbols) != 2 {
		t.Fatalf("projection files/symbols = %d/%d", len(projection.Files), len(projection.Symbols))
	}
	if projection.Files[0].ID == projection.Files[1].ID || projection.Files[0].Path == projection.Files[1].Path {
		t.Fatalf("projection did not preserve scope-qualified equal paths: %#v", projection.Files)
	}
	if err := analysis.ValidateSourceIndex(analysis.SourceIndex{SchemaVersion: analysis.SourceIndexSchemaVersion, Snapshots: []analysis.SourceIndexSnapshot{*projection}, Extensions: []analysis.ExtensionBlock{}}); err != nil {
		t.Fatalf("validate projection: %v", err)
	}
}

func TestBuildSourceIndexUsesRawPhysicalLineRule(t *testing.T) {
	cases := []struct {
		name    string
		content []byte
		lines   int
	}{
		{name: "empty", content: []byte{}, lines: 0},
		{name: "unterminated", content: []byte("a"), lines: 1},
		{name: "lf", content: []byte("a\nb"), lines: 2},
		{name: "crlf", content: []byte("a\r\nb"), lines: 2},
		{name: "lone-cr", content: []byte("a\rb"), lines: 2},
		{name: "invalid-utf8", content: []byte{0xff, '\r', 0xfe}, lines: 2},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			index, _, err := sourceindex.BuildSourceIndex(context.Background(), sourceindex.BuildInput{
				Scope:                 analysis.ScopeContext{ScopeID: "scope-lines", ProjectRoot: ".", Mode: analysis.SourceIndexScopeMode},
				Producer:              analysis.ProducerContext{AnalyzerID: "org.archview.go", AnalyzerVersion: "1.0.0"},
				Files:                 []sourceindex.SourceFileInput{{Path: "file.go", Content: testCase.content, Language: analysis.LanguageRef{ID: "language:go"}}},
				RequestedCapabilities: []string{sourceindex.CapabilityFiles, sourceindex.CapabilitySize},
			})
			if err != nil {
				t.Fatalf("build: %v", err)
			}
			file := index.Snapshots[0].Files[0]
			if file.Size.LineCount != testCase.lines || file.Size.ByteCount != len(testCase.content) || file.Size.ContentHash.Value == "" {
				t.Fatalf("file size = %#v, want %d lines and %d bytes", file.Size, testCase.lines, len(testCase.content))
			}
		})
	}
}

func TestModuleFileRelationUsesExactEndExclusivePosition(t *testing.T) {
	index, _, err := sourceindex.BuildSourceIndex(context.Background(), sourceindex.BuildInput{
		Scope:                 analysis.ScopeContext{ScopeID: "scope-span", ProjectRoot: ".", Mode: analysis.SourceIndexScopeMode},
		Producer:              analysis.ProducerContext{AnalyzerID: "fixture", AnalyzerVersion: "1.0.0"},
		Files:                 []sourceindex.SourceFileInput{{Path: "file.go", Content: []byte("a"), Language: analysis.LanguageRef{ID: "language:go"}, ModuleID: "module"}},
		RequestedCapabilities: []string{sourceindex.CapabilityFiles},
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	span := index.Snapshots[0].Relations[0].EvidenceSpans[0]
	if span.End.ByteOffset != 1 || span.End.Line != 1 || span.End.Column != 2 {
		t.Fatalf("full-file span end = %#v", span.End)
	}
}

func TestBuildSourceIndexRejectsMalformedExtractorBatchAtomically(t *testing.T) {
	registry := sourceindex.NewRegistry()
	if err := registry.Register(fixtureExtractor{id: "fixture", version: "1.0.0", batch: sourceindex.FactBatch{
		Symbols:     []analysis.SymbolRecord{{Name: "Valid", Category: analysis.SymbolCategoryValue, Locations: []analysis.SymbolLocation{{Kind: "declaration"}}}},
		Occurrences: []analysis.SymbolOccurrence{{TargetName: "target", OccurrenceKind: "reference", ResolutionStatus: "unresolved", SourceSpan: analysis.SourceSpan{Start: analysis.SpanPosition{Line: 1, Column: 1}, End: analysis.SpanPosition{ByteOffset: 100, Line: 1, Column: 101}}}},
	}}); err != nil {
		t.Fatalf("register extractor: %v", err)
	}
	index, diagnostics, err := sourceindex.BuildSourceIndex(context.Background(), sourceindex.BuildInput{
		Scope:                 analysis.ScopeContext{ScopeID: "scope-invalid-batch", ProjectRoot: ".", Mode: analysis.SourceIndexScopeMode},
		Producer:              analysis.ProducerContext{AnalyzerID: "fixture", AnalyzerVersion: "1.0.0"},
		Files:                 []sourceindex.SourceFileInput{{Path: "main.go", Content: []byte("package main\n"), Language: analysis.LanguageRef{ID: "language:go"}}},
		RequestedCapabilities: []string{sourceindex.CapabilityDeclarations}, SyntaxProvider: gosyntax.NewProvider(), Extractors: registry,
	})
	if err != nil {
		t.Fatalf("build source index: %v", err)
	}
	if len(index.Snapshots[0].Symbols) != 0 {
		t.Fatalf("rejected batch leaked symbols: %#v", index.Snapshots[0].Symbols)
	}
	if len(diagnostics) != 1 || index.Snapshots[0].Coverage[0].Status != analysis.FactStatusPartial {
		t.Fatalf("malformed batch diagnostics/coverage = %#v / %#v", diagnostics, index.Snapshots[0].Coverage)
	}
}

func TestBuildSourceIndexReportsMixedLanguageCoverageAsPartial(t *testing.T) {
	index, _, err := sourceindex.BuildSourceIndex(context.Background(), sourceindex.BuildInput{
		Scope:    analysis.ScopeContext{ScopeID: "scope-mixed", ProjectRoot: ".", Mode: analysis.SourceIndexScopeMode},
		Producer: analysis.ProducerContext{AnalyzerID: "fixture", AnalyzerVersion: "1.0.0"},
		Files: []sourceindex.SourceFileInput{
			{Path: "main.go", Content: []byte("package main\nfunc Main() {}\n"), Language: analysis.LanguageRef{ID: "language:go"}},
			{Path: "main.ts", Content: []byte("export function main() {}\n"), Language: analysis.LanguageRef{ID: "language:typescript"}},
		},
		RequestedCapabilities: []string{sourceindex.CapabilityDeclarations}, SyntaxProvider: gosyntax.NewProvider(), Extractors: sourcefacts.NewRegistry(),
	})
	if err != nil {
		t.Fatalf("build source index: %v", err)
	}
	coverage := index.Snapshots[0].Coverage[0]
	if coverage.Status != analysis.FactStatusPartial || coverage.EligibleCount == nil || *coverage.EligibleCount != 2 || coverage.ObservedCount == nil || *coverage.ObservedCount != 1 {
		t.Fatalf("mixed-language coverage = %#v", coverage)
	}
}

func TestBuildSourceIndexRejectsAmbiguousExtractorStrategies(t *testing.T) {
	registry := sourceindex.NewRegistry()
	for _, extractor := range []fixtureExtractor{{id: "first", version: "1"}, {id: "second", version: "1"}} {
		if err := registry.Register(extractor); err != nil {
			t.Fatalf("register extractor: %v", err)
		}
	}
	_, _, err := sourceindex.BuildSourceIndex(context.Background(), sourceindex.BuildInput{
		Scope:                 analysis.ScopeContext{ScopeID: "scope-conflict", ProjectRoot: ".", Mode: analysis.SourceIndexScopeMode},
		Files:                 []sourceindex.SourceFileInput{{Path: "main.go", Content: []byte("package main\n"), Language: analysis.LanguageRef{ID: "language:go"}}},
		RequestedCapabilities: []string{sourceindex.CapabilityDeclarations}, Extractors: registry,
	})
	if analysis.ErrorCodeOf(err) != analysis.ErrSourceExtractorFailed {
		t.Fatalf("ambiguous extractor error = %v", err)
	}
}

func TestSnapshotIdentityIncludesExtractorVersion(t *testing.T) {
	build := func(version string) string {
		registry := sourceindex.NewRegistry()
		if err := registry.Register(fixtureExtractor{id: "fixture", version: version}); err != nil {
			t.Fatalf("register extractor %s: %v", version, err)
		}
		index, _, err := sourceindex.BuildSourceIndex(context.Background(), sourceindex.BuildInput{
			Scope:                 analysis.ScopeContext{ScopeID: "scope-version", ProjectRoot: ".", Mode: analysis.SourceIndexScopeMode},
			Producer:              analysis.ProducerContext{AnalyzerID: "fixture", AnalyzerVersion: "1.0.0"},
			Files:                 []sourceindex.SourceFileInput{{Path: "main.go", Content: []byte("package main\n"), Language: analysis.LanguageRef{ID: "language:go"}}},
			RequestedCapabilities: []string{sourceindex.CapabilityDeclarations}, SyntaxProvider: gosyntax.NewProvider(), Extractors: registry,
		})
		if err != nil {
			t.Fatalf("build with extractor %s: %v", version, err)
		}
		return index.Snapshots[0].SnapshotID
	}
	if first, second := build("1.0.0"), build("2.0.0"); first == second {
		t.Fatalf("extractor version change retained snapshot id %q", first)
	}
}

func TestCombinedCoverageKeepsProvenanceStatusAligned(t *testing.T) {
	build := func(scope, language, content string, registry *sourceindex.Registry) analysis.SourceIndexSnapshot {
		index, _, err := sourceindex.BuildSourceIndex(context.Background(), sourceindex.BuildInput{
			Scope:                 analysis.ScopeContext{ScopeID: scope, ProjectRoot: ".", Mode: analysis.SourceIndexScopeMode},
			Producer:              analysis.ProducerContext{AnalyzerID: "fixture", AnalyzerVersion: "1.0.0"},
			Files:                 []sourceindex.SourceFileInput{{Path: "main." + language, Content: []byte(content), Language: analysis.LanguageRef{ID: "language:" + language}}},
			RequestedCapabilities: []string{sourceindex.CapabilityDeclarations}, SyntaxProvider: gosyntax.NewProvider(), Extractors: registry,
		})
		if err != nil {
			t.Fatalf("build %s: %v", scope, err)
		}
		return index.Snapshots[0]
	}
	projection, err := sourceindex.BuildCombinedProjection([]analysis.SourceIndexSnapshot{
		build("scope-go", "go", "package main\nfunc Main() {}\n", sourcefacts.NewRegistry()),
		build("scope-ts", "typescript", "export function main() {}\n", nil),
	})
	if err != nil {
		t.Fatalf("build projection: %v", err)
	}
	coverage := projection.Coverage[0]
	if coverage.Status != analysis.FactStatusPartial || coverage.Provenance.Status != analysis.FactStatusPartial {
		t.Fatalf("combined coverage = %#v", coverage)
	}
}

func TestQueryRejectsMalformedGlobAndCompactsDocumentation(t *testing.T) {
	longText := strings.Repeat("λ", sourceindex.CompactDocumentationRuneLimit+10)
	index := analysis.SourceIndex{SchemaVersion: analysis.SourceIndexSchemaVersion, Snapshots: []analysis.SourceIndexSnapshot{{
		SnapshotID: "snapshot-query-validation", ScopeContext: analysis.ScopeContext{ScopeID: "scope-query-validation"},
		Documentation: []analysis.DocumentationRecord{{ID: "doc", NormalizedText: longText, RawText: longText, Completeness: analysis.DocumentationComplete}},
	}}}
	service := sourceindex.NewQueryService(&index)
	if _, err := service.FindFiles(sourceindex.QueryOptions{PathGlob: "["}); analysis.ErrorCodeOf(err) != analysis.ErrInvalidRequest {
		t.Fatalf("malformed glob error = %v", err)
	}
	page, err := service.FindDocumentation(sourceindex.QueryOptions{})
	if err != nil {
		t.Fatalf("find documentation: %v", err)
	}
	if got := page.Items[0]; got.RawText != "" || len([]rune(got.NormalizedText)) != sourceindex.CompactDocumentationRuneLimit || got.Completeness != analysis.DocumentationTruncated {
		t.Fatalf("compact documentation = %#v", got)
	}
}

func TestQueryFiltersResolveRepeatedModuleAndFileMembershipThroughRelations(t *testing.T) {
	index := analysis.SourceIndex{
		SchemaVersion: analysis.SourceIndexSchemaVersion,
		Snapshots: []analysis.SourceIndexSnapshot{{
			SnapshotID:   "snapshot-query",
			ScopeContext: analysis.ScopeContext{ScopeID: "scope-query", Mode: analysis.SourceIndexScopeMode},
			Files: []analysis.FileRecord{
				{ID: "file-a", Path: "shared/main.go", Language: analysis.LanguageRef{ID: "language:go"}},
				{ID: "file-b", Path: "shared/main.go", Language: analysis.LanguageRef{ID: "language:go"}},
				{ID: "file-unrelated", Path: "shared/other.go", Language: analysis.LanguageRef{ID: "language:go"}},
			},
			Symbols: []analysis.SymbolRecord{
				{ID: "symbol-a", Name: "A", LanguageKind: "go:function"},
				{ID: "symbol-b", Name: "B", LanguageKind: "go:struct"},
				{ID: "symbol-unrelated", Name: "Unrelated"},
			},
			Documentation: []analysis.DocumentationRecord{
				{ID: "doc-file-a", SubjectRef: analysis.EntityRef{Kind: "file", ID: "file-a"}},
				{ID: "doc-symbol-a", SubjectRef: analysis.EntityRef{Kind: "symbol", ID: "symbol-a"}},
				{ID: "doc-unrelated", SubjectRef: analysis.EntityRef{Kind: "file", ID: "file-unrelated"}},
			},
			Relations: []analysis.CodeRelation{
				{ID: "module-a-file-a", Category: analysis.RelationContains, FromRef: analysis.EntityRef{Kind: "module", ID: "module-a"}, ToRef: &analysis.EntityRef{Kind: "file", ID: "file-a"}},
				{ID: "module-b-file-b", Category: analysis.RelationContains, FromRef: analysis.EntityRef{Kind: "module", ID: "module-b"}, ToRef: &analysis.EntityRef{Kind: "file", ID: "file-b"}},
				{ID: "file-a-symbol-a", Category: analysis.RelationDeclares, FromRef: analysis.EntityRef{Kind: "file", ID: "file-a"}, ToRef: &analysis.EntityRef{Kind: "symbol", ID: "symbol-a"}},
				{ID: "file-b-symbol-b", Category: analysis.RelationContains, FromRef: analysis.EntityRef{Kind: "file", ID: "file-b"}, ToRef: &analysis.EntityRef{Kind: "symbol", ID: "symbol-b"}},
			},
		}},
	}
	service := sourceindex.NewQueryService(&index)

	files, err := service.FindFiles(sourceindex.QueryOptions{ModuleIDs: []string{"module-b", "module-a"}, Limit: 25})
	if err != nil {
		t.Fatalf("module-filtered files: %v", err)
	}
	if files.Total != 2 || files.Items[0].ID != "file-a" || files.Items[1].ID != "file-b" {
		t.Fatalf("module-filtered files = %#v, want file-a and file-b only", files)
	}

	symbols, err := service.FindSymbols(sourceindex.QueryOptions{ModuleIDs: []string{"module-a"}, Limit: 25})
	if err != nil {
		t.Fatalf("module-filtered symbols: %v", err)
	}
	if symbols.Total != 1 || symbols.Items[0].ID != "symbol-a" {
		t.Fatalf("module-filtered symbols = %#v, want symbol-a only", symbols)
	}

	fileSymbols, err := service.FindSymbols(sourceindex.QueryOptions{FileIDs: []string{"file-b"}, Limit: 25})
	if err != nil {
		t.Fatalf("file-filtered symbols: %v", err)
	}
	if fileSymbols.Total != 1 || fileSymbols.Items[0].ID != "symbol-b" {
		t.Fatalf("file-filtered symbols = %#v, want symbol-b only", fileSymbols)
	}

	kindFiltered, err := service.FindSymbols(sourceindex.QueryOptions{LanguageKind: "go:struct", Limit: 25})
	if err != nil {
		t.Fatalf("language-kind-filtered symbols: %v", err)
	}
	if kindFiltered.Total != 1 || kindFiltered.Items[0].ID != "symbol-b" {
		t.Fatalf("language-kind-filtered symbols = %#v, want symbol-b only", kindFiltered)
	}

	documentation, err := service.FindDocumentation(sourceindex.QueryOptions{ModuleIDs: []string{"module-a"}, Limit: 25})
	if err != nil {
		t.Fatalf("module-filtered documentation: %v", err)
	}
	if documentation.Total != 2 || documentation.Items[0].ID != "doc-file-a" || documentation.Items[1].ID != "doc-symbol-a" {
		t.Fatalf("module-filtered documentation = %#v, want both explicit subjects from file-a", documentation)
	}

	linkedDocumentation, err := service.FindDocumentation(sourceindex.QueryOptions{SubjectIDs: []string{"symbol-a"}, Limit: 25})
	if err != nil {
		t.Fatalf("subject-filtered documentation: %v", err)
	}
	if linkedDocumentation.Total != 1 || linkedDocumentation.Items[0].ID != "doc-symbol-a" {
		t.Fatalf("subject-filtered documentation = %#v, want doc-symbol-a only", linkedDocumentation)
	}

	unknown, err := service.FindFiles(sourceindex.QueryOptions{ModuleIDs: []string{"module-a"}, FileIDs: []string{"unknown-file"}, Limit: 25})
	if err != nil {
		t.Fatalf("unknown explicit membership: %v", err)
	}
	if unknown.Total != 1 || unknown.Items[0].ID != "file-a" {
		t.Fatalf("unknown explicit membership widened result = %#v", unknown)
	}
}
