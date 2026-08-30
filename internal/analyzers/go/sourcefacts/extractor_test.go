package sourcefacts_test

import (
	"context"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/sourceindex"
	gosyntax "github.com/buffo/arch-view/internal/analysis/syntax/go"
	"github.com/buffo/arch-view/internal/analyzers/go/sourcefacts"
)

func TestExtractorOnlyEmitsDeclaredValueNamesAndUsesPackageVisibility(t *testing.T) {
	index, _, err := sourceindex.BuildSourceIndex(context.Background(), sourceindex.BuildInput{
		Scope:                 analysis.ScopeContext{ScopeID: "scope-values", ProjectRoot: ".", Mode: analysis.SourceIndexScopeMode},
		Producer:              analysis.ProducerContext{AnalyzerID: "org.archview.go", AnalyzerVersion: "1.0.0"},
		Files:                 []sourceindex.SourceFileInput{{Path: "main.go", Content: []byte("package main\nvar Referenced = 1\nvar declared, Exported = Referenced, 2\nvar _hidden = 3\n"), Language: analysis.LanguageRef{ID: "language:go"}}},
		RequestedCapabilities: []string{sourceindex.CapabilityDeclarations, sourceindex.CapabilityVisibility}, SyntaxProvider: gosyntax.NewProvider(), Extractors: sourcefacts.NewRegistry(),
	})
	if err != nil {
		t.Fatalf("build source index: %v", err)
	}
	symbols := make(map[string]analysis.SymbolRecord)
	for _, symbol := range index.Snapshots[0].Symbols {
		symbols[symbol.Name] = symbol
	}
	if len(symbols) != 4 {
		t.Fatalf("symbols = %#v, want only the four declared values", symbols)
	}
	if symbols["declared"].Visibility.Classification != "package" || symbols["_hidden"].Visibility.Classification != "package" || symbols["Exported"].Visibility.Classification != "public" {
		t.Fatalf("Go visibility facts = declared:%#v _hidden:%#v Exported:%#v", symbols["declared"].Visibility, symbols["_hidden"].Visibility, symbols["Exported"].Visibility)
	}
}

func TestExtractorAttachesDocumentationToTypeAndValueDeclarations(t *testing.T) {
	index, _, err := sourceindex.BuildSourceIndex(context.Background(), sourceindex.BuildInput{
		Scope:                 analysis.ScopeContext{ScopeID: "scope-docs", ProjectRoot: ".", Mode: analysis.SourceIndexScopeMode},
		Producer:              analysis.ProducerContext{AnalyzerID: "org.archview.go", AnalyzerVersion: "1.0.0"},
		Files:                 []sourceindex.SourceFileInput{{Path: "main.go", Content: []byte("package main\n\n// Thing docs.\ntype Thing struct{}\n\n// Value docs.\nvar Value = 1\n"), Language: analysis.LanguageRef{ID: "language:go"}}},
		RequestedCapabilities: []string{sourceindex.CapabilityDeclarations, sourceindex.CapabilityDocumentation}, SyntaxProvider: gosyntax.NewProvider(), Extractors: sourcefacts.NewRegistry(),
	})
	if err != nil {
		t.Fatalf("build source index: %v", err)
	}
	docs := make(map[string]string)
	symbolNames := make(map[string]string)
	for _, symbol := range index.Snapshots[0].Symbols {
		symbolNames[symbol.ID] = symbol.Name
	}
	for _, doc := range index.Snapshots[0].Documentation {
		docs[symbolNames[doc.SubjectRef.ID]] = doc.NormalizedText
	}
	if docs["Thing"] != "Thing docs." || docs["Value"] != "Value docs." {
		t.Fatalf("declaration documentation = %#v", docs)
	}
}

func TestExtractorProvidesCallableBodySpanAndVersionedMetrics(t *testing.T) {
	content := []byte("package main\n\n// DecodeAt validates a document.\nfunc DecodeAt(data []byte) error {\n\tif len(data) > 0 {\n\t\tfor _, value := range data {\n\t\t\tif value == 0 {\n\t\t\t\treturn nil\n\t\t\t}\n\t\t}\n\t}\n\treturn nil\n}\n")
	index, _, err := sourceindex.BuildSourceIndex(context.Background(), sourceindex.BuildInput{
		Scope:    analysis.ScopeContext{ScopeID: "scope-callable-metrics", ProjectRoot: ".", Mode: analysis.SourceIndexScopeMode},
		Producer: analysis.ProducerContext{AnalyzerID: "org.archview.go", AnalyzerVersion: "1.0.0"},
		Files: []sourceindex.SourceFileInput{{
			Path:     "main.go",
			Content:  content,
			Language: analysis.LanguageRef{ID: "language:go"},
		}},
		RequestedCapabilities: []string{sourceindex.CapabilityCallableMetrics, sourceindex.CapabilityDeclarations, sourceindex.CapabilityDocumentation, sourceindex.CapabilityVisibility},
		SyntaxProvider:        gosyntax.NewProvider(),
		Extractors:            sourcefacts.NewRegistry(),
	})
	if err != nil {
		t.Fatalf("build source index: %v", err)
	}
	var callable analysis.SymbolRecord
	for _, symbol := range index.Snapshots[0].Symbols {
		if symbol.Name == "DecodeAt" {
			callable = symbol
			break
		}
	}
	if callable.ID == "" || callable.BodySpan == nil {
		t.Fatalf("callable = %#v, want an explicit body span", callable)
	}
	if callable.BodySpan.Start.Line != 4 || callable.BodySpan.End.Line != 13 {
		t.Fatalf("body span = %#v, want lines 4-13", callable.BodySpan)
	}
	metrics := make(map[string]analysis.MetricFact)
	for _, metric := range index.Snapshots[0].Metrics {
		if metric.SubjectRef.ID == callable.ID {
			metrics[metric.MetricID] = metric
		}
	}
	if got := metrics["source:callable.cyclomatic_complexity"].Value.Value; got != 4 {
		t.Fatalf("complexity metric = %#v, want 4", got)
	}
	if got := metrics["source:callable.max_nesting_depth"].Value.Value; got != 3 {
		t.Fatalf("nesting metric = %#v, want 3", got)
	}
	for metricID, metric := range metrics {
		if metricID == "source:callable.cyclomatic_complexity" || metricID == "source:callable.max_nesting_depth" {
			if metric.FormulaVersion != sourcefacts.ExtractorVersion || len(metric.Extensions) != 1 {
				t.Fatalf("metric %s metadata = %#v, want extractor formula and span vocabulary", metricID, metric)
			}
		}
	}
	for _, documentation := range index.Snapshots[0].Documentation {
		if documentation.SubjectRef.ID == callable.ID && documentation.Status != analysis.DocumentationPresent {
			t.Fatalf("DecodeAt documentation = %#v, want present", documentation)
		}
	}
}

func TestExtractorCountsDeclaredGoDecisionVocabulary(t *testing.T) {
	content := []byte("package main\n\nfunc classify(value int, input <-chan int) int {\n\tif value > 0 && value < 10 {\n\t\tvalue++\n\t}\n\tfor value > 10 {\n\t\tvalue--\n\t}\n\tswitch value {\n\tcase 1:\n\t\treturn 1\n\tcase 2, 3:\n\t\treturn 2\n\tdefault:\n\t\tvalue = 0\n\t}\n\tselect {\n\tcase <-input:\n\t\treturn value\n\tdefault:\n\t\treturn 0\n\t}\n}\n")
	index, _, err := sourceindex.BuildSourceIndex(context.Background(), sourceindex.BuildInput{
		Scope:                 analysis.ScopeContext{ScopeID: "scope-decision-vocabulary", ProjectRoot: ".", Mode: analysis.SourceIndexScopeMode},
		Producer:              analysis.ProducerContext{AnalyzerID: "org.archview.go", AnalyzerVersion: "1.0.0"},
		Files:                 []sourceindex.SourceFileInput{{Path: "main.go", Content: content, Language: analysis.LanguageRef{ID: "language:go"}}},
		RequestedCapabilities: []string{sourceindex.CapabilityCallableMetrics, sourceindex.CapabilityDeclarations},
		SyntaxProvider:        gosyntax.NewProvider(),
		Extractors:            sourcefacts.NewRegistry(),
	})
	if err != nil {
		t.Fatalf("build source index: %v", err)
	}
	for _, metric := range index.Snapshots[0].Metrics {
		if metric.MetricID == "source:callable.cyclomatic_complexity" {
			if metric.Value.Value != 7 {
				t.Fatalf("complexity metric = %#v, want base + if + boolean + for + two switch cases + select case = 7", metric.Value.Value)
			}
			return
		}
	}
	t.Fatal("callable complexity metric was not emitted")
}

func TestExtractorReportsGoStructuralFactsForSolidSignals(t *testing.T) {
	content := []byte(`package app

// Dependency is the service boundary.
type Dependency interface {
	Read() error
	Write() error
}

type Base struct {
	Value int
}

// Service coordinates its dependencies.
type Service struct {
	Base
	dep    Dependency
	client *Client
}

type Client struct{}

func (s *Service) Decode(value any) error {
	switch value.(type) {
	case string:
		return nil
	default:
		return nil
	}
}
`)
	index, _, err := sourceindex.BuildSourceIndex(context.Background(), sourceindex.BuildInput{
		Scope:                 analysis.ScopeContext{ScopeID: "scope-solid", ProjectRoot: ".", Mode: analysis.SourceIndexScopeMode},
		Producer:              analysis.ProducerContext{AnalyzerID: "org.archview.go", AnalyzerVersion: "1.0.0"},
		Files:                 []sourceindex.SourceFileInput{{Path: "service.go", Content: content, Language: analysis.LanguageRef{ID: "language:go"}}},
		RequestedCapabilities: []string{sourceindex.CapabilityDeclarations, sourceindex.CapabilityDocumentation, "source:solid.structure"},
		SyntaxProvider:        gosyntax.NewProvider(),
		Extractors:            sourcefacts.NewRegistry(),
	})
	if err != nil {
		t.Fatalf("build source index: %v", err)
	}
	snapshot := index.Snapshots[0]
	if !hasCapability(snapshot.Capabilities, "source:solid.structure") {
		t.Fatalf("capabilities = %#v, want source:solid.structure", snapshot.Capabilities)
	}
	if coverage := capabilityCoverage(snapshot.Coverage, "source:solid.structure"); coverage.Status != analysis.FactStatusObserved {
		t.Fatalf("SOLID coverage = %#v, want observed", coverage)
	}
	symbols := symbolsByName(snapshot.Symbols)
	service := symbols["Service"]
	if service.MemberCount == nil || *service.MemberCount != 4 || service.MethodCount == nil || *service.MethodCount != 1 || service.DependencyCount == nil || *service.DependencyCount != 3 || service.ConcreteDependencyCount == nil || *service.ConcreteDependencyCount != 2 || service.HierarchyDepth == nil || *service.HierarchyDepth != 1 {
		t.Fatalf("Service structural facts = %#v", service)
	}
	dependency := symbols["Dependency"]
	if dependency.InterfaceMethodCount == nil || *dependency.InterfaceMethodCount != 2 || dependency.AbstractionCount == nil || *dependency.AbstractionCount != 1 {
		t.Fatalf("Dependency structural facts = %#v", dependency)
	}
	base := symbols["Base"]
	if base.DerivedTypeCount == nil || *base.DerivedTypeCount != 1 {
		t.Fatalf("Base structural facts = %#v, want one embedding type", base)
	}
	decode := symbols["Decode"]
	if decode.TypeSwitchCount == nil || *decode.TypeSwitchCount != 1 {
		t.Fatalf("Decode structural facts = %#v, want one type switch", decode)
	}
}

func hasCapability(values []analysis.CapabilityDescriptor, id string) bool {
	for _, value := range values {
		if value.ID == id {
			return true
		}
	}
	return false
}

func capabilityCoverage(values []analysis.CoverageRecord, id string) analysis.CoverageRecord {
	for _, value := range values {
		if value.Capability == id {
			return value
		}
	}
	return analysis.CoverageRecord{Capability: id, Status: "missing"}
}

func symbolsByName(values []analysis.SymbolRecord) map[string]analysis.SymbolRecord {
	result := make(map[string]analysis.SymbolRecord, len(values))
	for _, value := range values {
		result[value.Name] = value
	}
	return result
}
