package gosyntax

import (
	"context"
	"errors"
	"testing"

	"github.com/buffo/arch-view/internal/analysis/syntax"
)

func TestProviderParsesGoPackageAndImports(t *testing.T) {
	result, err := NewProvider().Parse(context.Background(), syntax.Source{
		Path: "cmd/app/main.go",
		Content: []byte(`package app

import (
	"fmt"
	service "example.com/app/internal/service"
)

func main() { fmt.Println(service.Name) }
`),
	})
	if err != nil {
		t.Fatalf("parse Go source: %v", err)
	}
	defer result.Close()
	if result.Tree == nil || result.Tree.Root() == nil {
		t.Fatal("Go provider returned no syntax tree")
	}
	if got := result.Tree.Root().Type(); got != "source_file" {
		t.Fatalf("root type = %q, want source_file", got)
	}
	for _, nodeType := range []string{"package_clause", "import_declaration", "import_spec", "interpreted_string_literal"} {
		if !containsNodeType(result.Tree.Root(), nodeType) {
			t.Errorf("syntax tree does not contain %q", nodeType)
		}
	}
	if len(result.Issues) != 0 {
		t.Fatalf("valid Go source issues = %#v", result.Issues)
	}
}

func TestProviderReportsRecoverableGoSyntaxIssues(t *testing.T) {
	result, err := NewProvider().Parse(context.Background(), syntax.Source{
		Path:    "broken.go",
		Content: []byte("package broken\n\nfunc broken( {\n"),
	})
	if err != nil {
		t.Fatalf("parse malformed Go source: %v", err)
	}
	defer result.Close()
	if result.Tree == nil || result.Tree.Root() == nil {
		t.Fatal("malformed Go source returned no recoverable tree")
	}
	if len(result.Issues) == 0 {
		t.Fatal("malformed Go source produced no Tree-sitter issues")
	}
}

func TestProviderHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewProvider().Parse(ctx, syntax.Source{Path: "canceled.go", Content: []byte("package canceled\n")})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled parse error = %v, want context.Canceled", err)
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
