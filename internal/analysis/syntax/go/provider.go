// Package gosyntax provides Tree-sitter syntax parsing for Go source files.
package gosyntax

import (
	"context"
	"errors"

	"github.com/buffo/arch-view/internal/analysis/syntax"
	"github.com/buffo/arch-view/internal/analysis/syntax/treesitter"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

// Provider parses Go source files with the Tree-sitter Go grammar.
type Provider struct {
	backend *treesitter.Provider
}

var _ syntax.Provider = (*Provider)(nil)

// NewProvider returns a Tree-sitter provider for Go source files.
func NewProvider() *Provider {
	language := tree_sitter.NewLanguage(tree_sitter_go.Language())
	return &Provider{
		backend: treesitter.NewProvider(func(syntax.Source) (*tree_sitter.Language, error) {
			return language, nil
		}),
	}
}

// Parse delegates to the generic Tree-sitter adapter.
func (provider *Provider) Parse(ctx context.Context, source syntax.Source) (syntax.ParseResult, error) {
	if provider == nil || provider.backend == nil {
		return syntax.ParseResult{}, errors.New("go tree-sitter provider is not configured")
	}
	return provider.backend.Parse(ctx, source)
}
