package pyanalyzer

import (
	"context"
	"errors"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/syntax"
	pysyntax "github.com/buffo/arch-view/internal/analysis/syntax/python"
)

func TestNewUsesTreeSitterProvider(t *testing.T) {
	analyzer := New()
	if analyzer.syntaxProvider == nil {
		t.Fatal("default Python analyzer has no syntax provider")
	}
	if _, ok := analyzer.syntaxProvider.(*pysyntax.Provider); !ok {
		t.Fatalf("default Python syntax provider = %T, want Tree-sitter Python provider", analyzer.syntaxProvider)
	}
}

func TestPythonAnalyzerReportsSyntaxBackendFailure(t *testing.T) {
	root := t.TempDir()
	writePythonFixture(t, root+"/pyproject.toml", "[project]\nname = 'fixture'\n")
	writePythonFixture(t, root+"/consumer.py", "import dependency\n")

	result, err := NewWithSyntaxProvider(failingPythonSyntaxProvider{}).Analyze(context.Background(), analysis.AnalyzeRequest{
		ProjectRoot: root,
		Options:     pythonOptions(t, nil),
	})
	if err != nil {
		t.Fatalf("analyze with failed syntax backend: %v", err)
	}
	if result.Status != analysis.StatusPartial {
		t.Fatalf("failed syntax backend status = %q, want partial", result.Status)
	}
	if !hasDiagnostic(result.Diagnostics, "python_syntax_backend") {
		t.Fatalf("missing syntax backend diagnostic: %#v", result.Diagnostics)
	}
	if len(result.Relationships) != 0 || len(result.References) != 0 {
		t.Fatalf("failed syntax backend unexpectedly produced dependency observations: %#v", result)
	}
}

type failingPythonSyntaxProvider struct{}

func (failingPythonSyntaxProvider) Parse(context.Context, syntax.Source) (syntax.ParseResult, error) {
	return syntax.ParseResult{}, errors.New("test Python syntax backend failure")
}
