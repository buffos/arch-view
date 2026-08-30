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
