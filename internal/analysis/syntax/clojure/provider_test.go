package clojure

import (
	"context"
	"testing"

	"github.com/buffo/arch-view/internal/analysis/syntax"
)

func TestProviderParsesClojureFormsAndReaderConditionals(t *testing.T) {
	result, err := NewProvider().Parse(context.Background(), syntax.Source{
		Path: "src/core.cljc",
		Content: []byte(`(ns demo.core
  (:require [clojure.string :as str :refer [join]]))
(defn greet [name] (str/join " " ["hello" name]))
#?(:clj (load "runtime.clj") :cljs (require 'demo.cljs))
`),
	})
	if err != nil {
		t.Fatalf("parse Clojure source: %v", err)
	}
	defer result.Close()
	if result.Tree == nil || result.Tree.Root() == nil {
		t.Fatal("Clojure provider returned no syntax tree")
	}
	if got := result.Tree.Root().Type(); got != "source" {
		t.Fatalf("root type = %q", got)
	}
	types := map[string]bool{}
	syntax.Walk(result.Tree.Root(), func(node syntax.Node) bool {
		types[node.Type()] = true
		return true
	})
	for _, expected := range []string{"list_lit", "sym_lit", "kwd_lit", "vec_lit", "read_cond_lit"} {
		if !types[expected] {
			t.Errorf("Tree-sitter tree did not contain %q: %#v", expected, types)
		}
	}
	if len(result.Issues) != 0 {
		t.Fatalf("valid Clojure source issues = %#v", result.Issues)
	}
}

func TestProviderReportsRecoverableClojureSyntaxIssues(t *testing.T) {
	result, err := NewProvider().Parse(context.Background(), syntax.Source{
		Path:    "broken.clj",
		Content: []byte("(ns broken\n(def value 1)\n"),
	})
	if err != nil {
		t.Fatalf("parse malformed Clojure source: %v", err)
	}
	defer result.Close()
	if result.Tree == nil {
		t.Fatal("malformed Clojure source returned no recoverable tree")
	}
	if len(result.Issues) == 0 {
		t.Fatal("malformed Clojure source produced no Tree-sitter issues")
	}
}

func TestProviderHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewProvider().Parse(ctx, syntax.Source{Path: "canceled.clj", Content: []byte("(ns demo)")})
	if err != context.Canceled {
		t.Fatalf("canceled parse error = %v", err)
	}
}
