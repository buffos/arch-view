package typescript

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/analysis/syntax"
)

func TestProviderParsesTypeScriptNodesAndSourceText(t *testing.T) {
	content := []byte("/** Loads the dependency. */\nimport { value as local } from \"dep\";\nlocal();\n")
	result, err := NewProvider().Parse(context.Background(), syntax.Source{Path: "src/app.ts", Content: content})
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
	if got := result.Tree.Root().Type(); got != "program" {
		t.Fatalf("root type = %q, want program", got)
	}

	types := map[string]bool{}
	var importText string
	syntax.Walk(result.Tree.Root(), func(node syntax.Node) bool {
		types[node.Type()] = true
		if node.Type() == "import_statement" {
			importText = node.Text()
		}
		return true
	})
	for _, nodeType := range []string{"comment", "import_statement", "call_expression"} {
		if !types[nodeType] {
			t.Errorf("syntax tree does not contain %q node", nodeType)
		}
	}
	if !strings.Contains(importText, `from "dep"`) {
		t.Fatalf("import node text = %q, want source text", importText)
	}
}

func TestProviderSelectsTSXGrammarForTSXFiles(t *testing.T) {
	result, err := NewProvider().Parse(context.Background(), syntax.Source{
		Path:    "src/component.tsx",
		Content: []byte("export const App = () => <div data-id=\"app\" />;\n"),
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	defer result.Close()
	if len(result.Issues) != 0 {
		t.Fatalf("Parse() issues = %#v, want none", result.Issues)
	}
	if !containsNodeType(result.Tree.Root(), "jsx_self_closing_element") {
		t.Fatal("TSX syntax tree does not contain jsx_self_closing_element")
	}
}

func TestProviderSelectsJavaScriptGrammarForJavaScriptFiles(t *testing.T) {
	result, err := NewProvider().Parse(context.Background(), syntax.Source{
		Path:    "src/loader.js",
		Content: []byte("const value = require(\"dep\");\n"),
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	defer result.Close()
	if len(result.Issues) != 0 {
		t.Fatalf("Parse() issues = %#v, want none", result.Issues)
	}
	if result.Tree.Root().Type() != "program" || !containsNodeType(result.Tree.Root(), "call_expression") {
		t.Fatal("JavaScript syntax tree did not parse the require call")
	}
}

func TestProviderReportsRecoverableSyntaxIssues(t *testing.T) {
	result, err := NewProvider().Parse(context.Background(), syntax.Source{
		Path:    "broken.ts",
		Content: []byte("const value = ;\n"),
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

	result, err := NewProvider().Parse(ctx, syntax.Source{Path: "cancelled.ts", Content: []byte("const value = 1;\n")})
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
