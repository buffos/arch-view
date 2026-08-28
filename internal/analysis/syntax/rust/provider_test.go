package rust

import (
	"context"
	"testing"

	"github.com/buffo/arch-view/internal/analysis/syntax"
)

func TestProviderParsesRustDeclarationsAndAttributes(t *testing.T) {
	result, err := NewProvider().Parse(context.Background(), syntax.Source{
		Path: "src/lib.rs",
		Content: []byte(`#![allow(dead_code)]
#[path = "components/api.rs"]
pub mod api;
pub use crate::api::Thing as ApiThing;
extern crate serde;
`),
	})
	if err != nil {
		t.Fatalf("parse Rust source: %v", err)
	}
	defer result.Close()
	if result.Tree == nil || result.Tree.Root() == nil {
		t.Fatal("Rust provider returned no syntax tree")
	}
	if got := result.Tree.Root().Type(); got != "source_file" {
		t.Fatalf("root type = %q", got)
	}
	types := map[string]bool{}
	syntax.Walk(result.Tree.Root(), func(node syntax.Node) bool {
		types[node.Type()] = true
		return true
	})
	for _, expected := range []string{"inner_attribute_item", "attribute_item", "mod_item", "use_declaration", "extern_crate_declaration"} {
		if !types[expected] {
			t.Errorf("Tree-sitter tree did not contain %q: %#v", expected, types)
		}
	}
	if len(result.Issues) != 0 {
		t.Fatalf("valid Rust source issues = %#v", result.Issues)
	}
}

func TestProviderReportsRecoverableRustSyntaxIssues(t *testing.T) {
	result, err := NewProvider().Parse(context.Background(), syntax.Source{
		Path:    "broken.rs",
		Content: []byte("mod broken {\n    use crate::Thing;\n"),
	})
	if err != nil {
		t.Fatalf("parse malformed Rust source: %v", err)
	}
	defer result.Close()
	if result.Tree == nil {
		t.Fatal("malformed Rust source returned no recoverable tree")
	}
	if len(result.Issues) == 0 {
		t.Fatal("malformed Rust source produced no Tree-sitter issues")
	}
}

func TestProviderHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewProvider().Parse(ctx, syntax.Source{Path: "canceled.rs", Content: []byte("fn main() {}")})
	if err != context.Canceled {
		t.Fatalf("canceled parse error = %v", err)
	}
}
