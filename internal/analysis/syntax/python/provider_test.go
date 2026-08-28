package python

import (
	"context"
	"errors"
	"testing"

	"github.com/buffo/arch-view/internal/analysis/syntax"
)

func TestProviderParsesPythonNodesAndSourceText(t *testing.T) {
	result, err := NewProvider().Parse(context.Background(), syntax.Source{
		Path:    "src/app.py",
		Content: []byte("\"\"\"Loads dependencies.\"\"\"\nfrom .service import Service as PublicService\nload = importlib.import_module(\"dep\")\n"),
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	defer result.Close()
	if len(result.Issues) != 0 {
		t.Fatalf("Parse() issues = %#v, want none", result.Issues)
	}
	if result.Tree == nil || result.Tree.Root() == nil {
		t.Fatal("Parse() returned no syntax tree")
	}
	if got := result.Tree.Root().Type(); got != "module" {
		t.Fatalf("root type = %q, want module", got)
	}
	for _, nodeType := range []string{"import_from_statement", "call", "string"} {
		if !containsNodeType(result.Tree.Root(), nodeType) {
			t.Errorf("syntax tree does not contain %q node", nodeType)
		}
	}
}

func TestProviderReportsRecoverableSyntaxIssues(t *testing.T) {
	result, err := NewProvider().Parse(context.Background(), syntax.Source{
		Path:    "broken.py",
		Content: []byte("from broken import (\n"),
	})
	if err != nil {
		t.Fatalf("Parse() error = %v, want recoverable tree", err)
	}
	defer result.Close()
	if len(result.Issues) == 0 {
		t.Fatal("Parse() returned no syntax issues for malformed source")
	}
	if result.Tree == nil || result.Tree.Root() == nil {
		t.Fatal("Parse() discarded the recoverable tree")
	}
}

func TestProviderHonorsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := NewProvider().Parse(ctx, syntax.Source{Path: "cancelled.py", Content: []byte("value = 1\n")})
	if !errors.Is(err, context.Canceled) {
		result.Close()
		t.Fatalf("Parse() error = %v, want context.Canceled", err)
	}
}

func containsNodeType(root syntax.Node, wanted string) bool {
	found := false
	syntax.Walk(root, func(node syntax.Node) bool {
		if node.Type() == wanted {
			found = true
			return false
		}
		return true
	})
	return found
}
