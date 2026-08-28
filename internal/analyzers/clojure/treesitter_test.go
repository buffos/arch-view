package clojureanalyzer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/syntax"
)

func TestNewUsesTreeSitterProvider(t *testing.T) {
	if New().syntaxProvider == nil {
		t.Fatal("Clojure analyzer has no syntax provider")
	}
}

func TestClojureDiscoveryReportsSyntaxBackendFailureWithoutLegacyFallback(t *testing.T) {
	root := t.TempDir()
	sourceRoot := filepath.Join(root, "src")
	if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
		t.Fatalf("create source root: %v", err)
	}
	sourcePath := filepath.Join(sourceRoot, "core.clj")
	if err := os.WriteFile(sourcePath, []byte("(ns demo.core)\n"), 0o644); err != nil {
		t.Fatalf("write Clojure source: %v", err)
	}
	project := Project{
		Root:        root,
		SourceRoots: []SourceRoot{{Absolute: sourceRoot, Relative: "src"}},
		Platform:    "both",
	}
	discovery, err := DiscoverWithSyntaxProvider(context.Background(), project, analysis.EffectiveOptions{}, failingClojureSyntaxProvider{})
	if err != nil {
		t.Fatalf("Clojure discovery: %v", err)
	}
	if len(discovery.Modules) != 0 {
		t.Fatalf("legacy parser produced modules after Tree-sitter failure: %#v", discovery.Modules)
	}
	found := false
	for _, diagnostic := range discovery.Diagnostics {
		if diagnostic.Code == "clojure_syntax_backend" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Tree-sitter backend failure was not reported: %#v", discovery.Diagnostics)
	}
}

type failingClojureSyntaxProvider struct{}

func (failingClojureSyntaxProvider) Parse(context.Context, syntax.Source) (syntax.ParseResult, error) {
	return syntax.ParseResult{}, errors.New("test backend failure")
}
