package rustanalyzer

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
		t.Fatal("Rust analyzer has no syntax provider")
	}
}

func TestRustDiscoveryReportsSyntaxBackendFailureWithoutLegacyFallback(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "lib.rs")
	if err := os.WriteFile(sourcePath, []byte("pub mod api;\n"), 0o644); err != nil {
		t.Fatalf("write Rust source: %v", err)
	}
	project := Project{
		Root:        root,
		CrateRoot:   root,
		PackageName: "demo",
		Targets:     []RustTarget{{Kind: "lib", AbsolutePath: sourcePath}},
	}
	discovery, err := DiscoverWithSyntaxProvider(context.Background(), project, analysis.EffectiveOptions{}, failingRustSyntaxProvider{})
	if err != nil {
		t.Fatalf("Rust discovery: %v", err)
	}
	if _, ok := discovery.Modules["api"]; ok {
		t.Fatalf("legacy parser produced a module after Tree-sitter failure: %#v", discovery.Modules)
	}
	found := false
	for _, diagnostic := range discovery.Diagnostics {
		if diagnostic.Code == "rust_syntax_backend" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Tree-sitter backend failure was not reported: %#v", discovery.Diagnostics)
	}
}

type failingRustSyntaxProvider struct{}

func (failingRustSyntaxProvider) Parse(context.Context, syntax.Source) (syntax.ParseResult, error) {
	return syntax.ParseResult{}, errors.New("test backend failure")
}
