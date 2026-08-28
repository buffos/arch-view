package tsanalyzer

import (
	"context"
	"errors"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/syntax"
)

func TestTreeSitterAnalyzerExtractsSupportedDependencyForms(t *testing.T) {
	root := t.TempDir()
	writeTSFixture(t, root+"/tsconfig.json", `{"include":["src/**/*"]}`)
	writeTSFixture(t, root+"/src/index.ts", `import { feature as importedFeature } from "./feature";
import type { Contract } from "./types";
export { feature } from "./feature";
export type { Contract as PublicContract } from "./types";
const service = require("./service");
const lazy = import("./lazy", { with: { type: "json" } });
const computed = import("./" + moduleName);
const required = require("./" + moduleName);
import "node:fs";
`)
	writeTSFixture(t, root+"/src/feature.ts", `export const feature = true;`)
	writeTSFixture(t, root+"/src/types.ts", `export type Contract = { ready: boolean };`)
	writeTSFixture(t, root+"/src/service.ts", `export const service = true;`)
	writeTSFixture(t, root+"/src/lazy.ts", `export const lazy = true;`)
	options := tsOptions(t, nil)
	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: options})
	if err != nil {
		t.Fatalf("analyze supported dependency forms: %v", err)
	}
	for _, target := range []string{
		"ts:module:src/feature",
		"ts:module:src/types",
		"ts:module:src/service",
		"ts:module:src/lazy",
	} {
		if !hasTSRelationshipToModule(result, "ts:module:src/index", target) {
			t.Fatalf("missing Tree-sitter dependency edge to %s: %#v", target, result.Relationships)
		}
	}
	typeEdge := findTSRelationshipToModule(result, "ts:module:src/index", "ts:module:src/types")
	if typeEdge.Metadata["type_only"] != true {
		t.Fatalf("Tree-sitter type-only metadata = %#v", typeEdge.Metadata)
	}
	lazyEdge := findTSRelationshipToModule(result, "ts:module:src/index", "ts:module:src/lazy")
	if lazyEdge.Metadata["dynamic"] != true {
		t.Fatalf("Tree-sitter dynamic metadata = %#v", lazyEdge.Metadata)
	}
	if !hasTSReference(result, "standard_library", "node:fs") {
		t.Fatalf("Tree-sitter standard-library reference missing: %#v", result.References)
	}
	if !hasTSDiagnostic(result, "typescript_dynamic_import") {
		t.Fatalf("Tree-sitter computed dynamic import diagnostic missing: %#v", result.Diagnostics)
	}
}

func TestTreeSitterAnalyzerRecoversModulesForMalformedSyntax(t *testing.T) {
	root := t.TempDir()
	writeTSFixture(t, root+"/tsconfig.json", `{"include":["src/**/*"]}`)
	writeTSFixture(t, root+"/src/main.ts", `import { broken from "./valid";
export const main = true;
`)
	writeTSFixture(t, root+"/src/valid.ts", `export const valid = true;`)

	request := analysis.AnalyzeRequest{ProjectRoot: root, Options: tsOptions(t, nil)}
	treeSitter, err := New().Analyze(context.Background(), request)
	if err != nil {
		t.Fatalf("Tree-sitter analyze malformed syntax: %v", err)
	}
	if treeSitter.Status != analysis.StatusPartial {
		t.Fatalf("malformed syntax status = %q", treeSitter.Status)
	}
	if !hasTSModule(treeSitter.Modules, "ts:module:src/main") || !hasTSModule(treeSitter.Modules, "ts:module:src/valid") {
		t.Fatalf("malformed syntax lost usable modules: %#v", treeSitter.Modules)
	}
	if !hasTSDiagnostic(treeSitter, "typescript_import_syntax") {
		t.Fatalf("Tree-sitter malformed syntax diagnostics = %#v", treeSitter.Diagnostics)
	}
}

func TestTreeSitterAnalyzerReportsBackendFailure(t *testing.T) {
	root := t.TempDir()
	writeTSFixture(t, root+"/tsconfig.json", `{"include":["src/**/*"]}`)
	writeTSFixture(t, root+"/src/main.ts", `import "./dependency";`)
	writeTSFixture(t, root+"/src/dependency.ts", `export const dependency = true;`)

	options := tsOptions(t, nil)
	result, err := NewWithSyntaxProvider(failingSyntaxProvider{}).Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: options})
	if err != nil {
		t.Fatalf("analyze with failed syntax backend: %v", err)
	}
	if hasTSRelationshipToModule(result, "ts:module:src/main", "ts:module:src/dependency") {
		t.Fatalf("failed syntax backend unexpectedly produced a local dependency: %#v", result.Relationships)
	}
	if result.Status != analysis.StatusPartial {
		t.Fatalf("failed syntax backend status = %q, want partial", result.Status)
	}
	if !hasTSDiagnostic(result, "typescript_syntax_backend") {
		t.Fatalf("missing syntax backend diagnostic: %#v", result.Diagnostics)
	}
}

func TestNewUsesTreeSitter(t *testing.T) {
	analyzer := New()
	if _, ok := analyzer.importExtractor.(treeSitterTSImportExtractor); !ok {
		t.Fatalf("default import extractor = %T, want direct Tree-sitter extractor", analyzer.importExtractor)
	}
}

type failingSyntaxProvider struct{}

func (failingSyntaxProvider) Parse(context.Context, syntax.Source) (syntax.ParseResult, error) {
	return syntax.ParseResult{}, errors.New("test syntax backend failure")
}
