// Package treesitter adapts Tree-sitter's native parser and nodes to the
// backend-neutral syntax contract.
package treesitter

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/buffo/arch-view/internal/analysis/syntax"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// LanguageSelector chooses the grammar for one source file. It allows a
// language provider to select dialects such as TypeScript, TSX, or JavaScript
// without adding language switches to the generic Tree-sitter adapter.
type LanguageSelector func(syntax.Source) (*tree_sitter.Language, error)

// Provider is a Tree-sitter-backed syntax provider.
type Provider struct {
	selectLanguage LanguageSelector
}

var _ syntax.Provider = (*Provider)(nil)

// NewProvider constructs a Tree-sitter provider using selector to choose a
// grammar for each source file.
func NewProvider(selector LanguageSelector) *Provider {
	return &Provider{selectLanguage: selector}
}

// Parse parses one source file and retains any recoverable error or missing
// nodes as backend-neutral syntax issues.
func (provider *Provider) Parse(ctx context.Context, source syntax.Source) (syntax.ParseResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return syntax.ParseResult{}, err
	}
	if provider == nil || provider.selectLanguage == nil {
		return syntax.ParseResult{}, errors.New("tree-sitter language selector is not configured")
	}
	language, err := provider.selectLanguage(source)
	if err != nil {
		return syntax.ParseResult{}, err
	}
	if language == nil {
		return syntax.ParseResult{}, errors.New("tree-sitter language selector returned nil")
	}

	parser := tree_sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(language); err != nil {
		return syntax.ParseResult{}, fmt.Errorf("tree-sitter language could not be assigned: %w", err)
	}
	tree := parser.ParseWithOptions(func(offset int, _ tree_sitter.Point) []byte {
		if offset < 0 || offset >= len(source.Content) {
			return nil
		}
		return source.Content[offset:]
	}, nil, &tree_sitter.ParseOptions{
		ProgressCallback: func(tree_sitter.ParseState) bool {
			return ctx.Err() != nil
		},
	})
	if err := ctx.Err(); err != nil {
		if tree != nil {
			tree.Close()
		}
		return syntax.ParseResult{}, err
	}
	if tree == nil {
		return syntax.ParseResult{}, errors.New("tree-sitter returned no syntax tree")
	}

	result := syntax.ParseResult{Tree: &syntaxTree{inner: tree, content: source.Content}}
	result.Issues = collectIssues(result.Tree.Root())
	return result, nil
}

func collectIssues(root syntax.Node) []syntax.Issue {
	issues := make([]syntax.Issue, 0)
	syntax.Walk(root, func(node syntax.Node) bool {
		switch {
		case node.IsError():
			issues = append(issues, syntax.Issue{
				Message: fmt.Sprintf("Tree-sitter reported an unexpected %q syntax node.", node.Type()),
				Range:   node.Range(),
			})
		case node.IsMissing():
			issues = append(issues, syntax.Issue{
				Message: fmt.Sprintf("Tree-sitter inserted a missing %q syntax node.", node.Type()),
				Range:   node.Range(),
			})
		}
		return true
	})
	return issues
}

type syntaxTree struct {
	inner   *tree_sitter.Tree
	content []byte
	once    sync.Once
}

func (tree *syntaxTree) Root() syntax.Node {
	if tree == nil || tree.inner == nil {
		return nil
	}
	return newNode(tree.inner.RootNode(), tree.content)
}

func (tree *syntaxTree) Close() {
	if tree == nil {
		return
	}
	tree.once.Do(func() {
		if tree.inner != nil {
			tree.inner.Close()
		}
	})
}

type syntaxNode struct {
	inner   *tree_sitter.Node
	content []byte
}

func newNode(node *tree_sitter.Node, content []byte) syntax.Node {
	if node == nil {
		return nil
	}
	return &syntaxNode{inner: node, content: content}
}

func (node *syntaxNode) Type() string { return node.inner.Kind() }

func (node *syntaxNode) IsNamed() bool { return node.inner.IsNamed() }

func (node *syntaxNode) IsError() bool { return node.inner.IsError() }

func (node *syntaxNode) IsMissing() bool { return node.inner.IsMissing() }

func (node *syntaxNode) Range() syntax.Range {
	rangeValue := node.inner.Range()
	return syntax.Range{
		Start: syntax.Point{
			Row:    uint32(rangeValue.StartPoint.Row),
			Column: uint32(rangeValue.StartPoint.Column),
		},
		End: syntax.Point{
			Row:    uint32(rangeValue.EndPoint.Row),
			Column: uint32(rangeValue.EndPoint.Column),
		},
		StartByte: uint32(rangeValue.StartByte),
		EndByte:   uint32(rangeValue.EndByte),
	}
}

func (node *syntaxNode) Text() string {
	return node.inner.Utf8Text(node.content)
}

func (node *syntaxNode) ChildCount() int { return int(node.inner.ChildCount()) }

func (node *syntaxNode) Child(index int) syntax.Node {
	if index < 0 {
		return nil
	}
	return newNode(node.inner.Child(uint(index)), node.content)
}

func (node *syntaxNode) NamedChildCount() int { return int(node.inner.NamedChildCount()) }

func (node *syntaxNode) NamedChild(index int) syntax.Node {
	if index < 0 {
		return nil
	}
	return newNode(node.inner.NamedChild(uint(index)), node.content)
}

func (node *syntaxNode) ChildByFieldName(field string) syntax.Node {
	return newNode(node.inner.ChildByFieldName(field), node.content)
}
