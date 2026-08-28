package syntax

import (
	"context"
	"reflect"
	"testing"
)

func TestWalkVisitsNodesInSourceOrderAndPrunes(t *testing.T) {
	leaf := &testNode{kind: "identifier", text: "leaf"}
	pruned := &testNode{kind: "pruned", children: []*testNode{{kind: "hidden"}}}
	root := &testNode{
		kind: "source_file",
		children: []*testNode{
			{kind: "comment", text: "doc"},
			pruned,
			leaf,
		},
	}

	var visited []string
	Walk(root, func(node Node) bool {
		visited = append(visited, node.Type())
		return node.Type() != "pruned"
	})

	want := []string{"source_file", "comment", "pruned", "identifier"}
	if !reflect.DeepEqual(visited, want) {
		t.Fatalf("visited nodes = %#v, want %#v", visited, want)
	}
}

func TestWalkIgnoresNilInputs(t *testing.T) {
	called := false
	Walk(nil, func(Node) bool {
		called = true
		return true
	})
	Walk(&testNode{kind: "root"}, nil)
	if called {
		t.Fatal("nil root invoked visitor")
	}
}

func TestParseResultCloseClosesOwnedTree(t *testing.T) {
	tree := &testTree{root: &testNode{kind: "source_file"}}
	result := ParseResult{Tree: tree}

	result.Close()

	if !tree.closed {
		t.Fatal("ParseResult.Close did not close the tree")
	}
}

func TestProviderContractCarriesSourceAndIssues(t *testing.T) {
	provider := testProvider{
		tree:   &testTree{root: &testNode{kind: "source_file"}},
		issues: []Issue{{Message: "recoverable", Range: Range{StartByte: 2, EndByte: 3}}},
	}
	result, err := provider.Parse(context.Background(), Source{Path: "main.go", Content: []byte("go")})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if provider.source.Path != "main.go" || string(provider.source.Content) != "go" {
		t.Fatalf("provider source = %#v, want path/content", provider.source)
	}
	if len(result.Issues) != 1 || result.Issues[0].Message != "recoverable" {
		t.Fatalf("parse issues = %#v, want one recoverable issue", result.Issues)
	}
}

type testProvider struct {
	source Source
	tree   *testTree
	issues []Issue
}

func (provider *testProvider) Parse(_ context.Context, source Source) (ParseResult, error) {
	provider.source = source
	return ParseResult{Tree: provider.tree, Issues: provider.issues}, nil
}

type testTree struct {
	root   *testNode
	closed bool
}

func (tree *testTree) Root() Node { return tree.root }

func (tree *testTree) Close() { tree.closed = true }

type testNode struct {
	kind     string
	named    bool
	err      bool
	missing  bool
	text     string
	rangeVal Range
	children []*testNode
	fields   map[string]*testNode
}

func (node *testNode) Type() string { return node.kind }

func (node *testNode) IsNamed() bool { return node.named }

func (node *testNode) IsError() bool { return node.err }

func (node *testNode) IsMissing() bool { return node.missing }

func (node *testNode) Range() Range { return node.rangeVal }

func (node *testNode) Text() string { return node.text }

func (node *testNode) ChildCount() int { return len(node.children) }

func (node *testNode) Child(index int) Node {
	if index < 0 || index >= len(node.children) {
		return nil
	}
	return node.children[index]
}

func (node *testNode) NamedChildCount() int {
	count := 0
	for _, child := range node.children {
		if child != nil && child.named {
			count++
		}
	}
	return count
}

func (node *testNode) NamedChild(index int) Node {
	if index < 0 {
		return nil
	}
	for _, child := range node.children {
		if child == nil || !child.named {
			continue
		}
		if index == 0 {
			return child
		}
		index--
	}
	return nil
}

func (node *testNode) ChildByFieldName(field string) Node {
	return node.fields[field]
}
