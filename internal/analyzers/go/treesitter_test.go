package goanalyzer

import (
	"context"
	"errors"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/syntax"
	gosyntax "github.com/buffo/arch-view/internal/analysis/syntax/go"
)

func TestNewUsesTreeSitterProvider(t *testing.T) {
	analyzer := New()
	if analyzer.syntaxProvider == nil {
		t.Fatal("Go analyzer has no syntax provider")
	}
	if _, ok := analyzer.syntaxProvider.(*gosyntax.Provider); !ok {
		t.Fatalf("default Go syntax provider = %T, want Tree-sitter Go provider", analyzer.syntaxProvider)
	}
}

func TestGoAnalyzerReportsSyntaxBackendFailureWithoutLegacyFallback(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root+"/go.mod", "module example.com/app\ngo 1.22\n")
	writeFixture(t, root+"/main.go", "package app\n\nimport \"example.com/app/service\"\n")

	result, err := NewWithSyntaxProvider(failingGoSyntaxProvider{}).Analyze(context.Background(), analysis.AnalyzeRequest{
		ProjectRoot: root,
		Options:     goOptions(t, nil),
	})
	if err != nil {
		t.Fatalf("analyze with failed syntax backend: %v", err)
	}
	if result.Status != analysis.StatusPartial {
		t.Fatalf("failed syntax backend status = %q, want partial", result.Status)
	}
	if !hasDiagnostic(result.Diagnostics, "go_syntax_backend") {
		t.Fatalf("missing syntax backend diagnostic: %#v", result.Diagnostics)
	}
	if len(result.Modules) != 0 || len(result.Relationships) != 0 || len(result.References) != 0 {
		t.Fatalf("failed syntax backend unexpectedly produced observations: %#v", result)
	}
}

type failingGoSyntaxProvider struct{}

func (failingGoSyntaxProvider) Parse(context.Context, syntax.Source) (syntax.ParseResult, error) {
	return syntax.ParseResult{}, errors.New("test Go syntax backend failure")
}
