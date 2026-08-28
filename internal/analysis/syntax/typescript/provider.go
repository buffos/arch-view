// Package typescript provides Tree-sitter syntax parsing for TypeScript,
// TSX, and JavaScript-family source files.
package typescript

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/buffo/arch-view/internal/analysis/syntax"
	"github.com/buffo/arch-view/internal/analysis/syntax/treesitter"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	tree_sitter_typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

// Provider selects the appropriate Tree-sitter grammar from a source path.
type Provider struct {
	backend *treesitter.Provider
}

var _ syntax.Provider = (*Provider)(nil)

// NewProvider returns a Tree-sitter provider for TypeScript, TSX, and
// JavaScript-family files.
func NewProvider() *Provider {
	typescriptLanguage := tree_sitter.NewLanguage(tree_sitter_typescript.LanguageTypescript())
	tsxLanguage := tree_sitter.NewLanguage(tree_sitter_typescript.LanguageTSX())
	javascriptLanguage := tree_sitter.NewLanguage(tree_sitter_javascript.Language())
	return &Provider{
		backend: treesitter.NewProvider(func(source syntax.Source) (*tree_sitter.Language, error) {
			switch strings.ToLower(filepath.Ext(source.Path)) {
			case ".tsx", ".jsx":
				return tsxLanguage, nil
			case ".js", ".mjs", ".cjs":
				return javascriptLanguage, nil
			default:
				return typescriptLanguage, nil
			}
		}),
	}
}

// Parse delegates to the generic Tree-sitter adapter.
func (provider *Provider) Parse(ctx context.Context, source syntax.Source) (syntax.ParseResult, error) {
	if provider == nil || provider.backend == nil {
		return syntax.ParseResult{}, errors.New("typescript tree-sitter provider is not configured")
	}
	return provider.backend.Parse(ctx, source)
}
