package main

import (
	"flag"
	"testing"
)

func TestCollectRustAnalyzerCLIOptionsKeepsOnlyExplicitValues(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	crate := fs.String("crate", "", "")
	target := fs.String("target", "", "")
	includeTests := fs.Bool("include-tests", false, "")
	includeExamples := fs.Bool("include-examples", false, "")
	excludes := stringList{}
	features := stringList{}
	buildTags := stringList{}
	sourceRoots := stringList{}
	pythonVersion := ""
	includeGenerated := false
	includeExternal := false
	safeMode := true
	includeStubs := false
	fs.Var(&excludes, "exclude", "")
	fs.Var(&features, "feature", "")
	if err := fs.Parse([]string{"--crate", "crates/api", "--feature", "serde", "--feature", "api", "--target", "wasm32-unknown-unknown", "--include-tests", "--include-examples", "--exclude", "generated/**"}); err != nil {
		t.Fatalf("parse Rust options: %v", err)
	}
	values := collectAnalyzerCLIOptions(fs, analyzerCLIFlags{
		crate:            crate,
		features:         &features,
		target:           target,
		includeTests:     includeTests,
		includeExamples:  includeExamples,
		includeGenerated: &includeGenerated,
		includeExternal:  &includeExternal,
		safeMode:         &safeMode,
		buildTags:        &buildTags,
		excludes:         &excludes,
		sourceRoots:      &sourceRoots,
		pythonVersion:    &pythonVersion,
		includeStubs:     &includeStubs,
	})
	if values["crate"] != "crates/api" || values["target"] != "wasm32-unknown-unknown" || values["include_tests"] != true || values["include_examples"] != true {
		t.Fatalf("scalar Rust options = %#v", values)
	}
	if got, ok := values["features"].([]string); !ok || len(got) != 2 || got[0] != "serde" || got[1] != "api" {
		t.Fatalf("feature options = %#v", values["features"])
	}
	if got, ok := values["exclude"].([]string); !ok || len(got) != 1 || got[0] != "generated/**" {
		t.Fatalf("exclude options = %#v", values["exclude"])
	}
	if _, exists := values["module"]; exists {
		t.Fatalf("unset Go option leaked into Rust options: %#v", values)
	}
}
