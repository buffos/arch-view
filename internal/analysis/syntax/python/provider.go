// Package python provides Tree-sitter syntax parsing for Python source files.
package python

import (
	"context"
	"errors"

	"github.com/buffo/arch-view/internal/analysis/syntax"
	"github.com/buffo/arch-view/internal/analysis/syntax/treesitter"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"
)

// Provider parses Python source files with the Python Tree-sitter grammar.
type Provider struct {
	backend *treesitter.Provider
}

var _ syntax.Provider = (*Provider)(nil)

// NewProvider returns a Tree-sitter provider for Python source files.
func NewProvider() *Provider {
	language := tree_sitter.NewLanguage(tree_sitter_python.Language())
	return &Provider{
		backend: treesitter.NewProvider(func(syntax.Source) (*tree_sitter.Language, error) {
			return language, nil
		}),
	}
}

// Parse delegates to the generic Tree-sitter adapter.
func (provider *Provider) Parse(ctx context.Context, source syntax.Source) (syntax.ParseResult, error) {
	if provider == nil || provider.backend == nil {
		return syntax.ParseResult{}, errors.New("python tree-sitter provider is not configured")
	}
	return provider.backend.Parse(ctx, source)
}
