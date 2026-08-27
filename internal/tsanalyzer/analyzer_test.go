package tsanalyzer

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
)

func TestManifestMatchesTypeScriptContract(t *testing.T) {
	manifest := New().Manifest()
	if err := analysis.ValidateManifest(manifest); err != nil {
		t.Fatalf("validate manifest: %v", err)
	}
	if manifest.ID != "org.archview.typescript" || manifest.Language != "typescript" || manifest.APIVersion != analysis.AnalyzerAPIVersion {
		t.Fatalf("manifest identity = %#v", manifest)
	}
	if len(manifest.DetectionMarkers) != 2 || manifest.DetectionMarkers[0].Value != "tsconfig.json" || manifest.DetectionMarkers[1].Value != "package.json" {
		t.Fatalf("manifest markers = %#v", manifest.DetectionMarkers)
	}
	if strings.Join(manifest.Capabilities, ",") != "detect,static_dependencies,aliases,exports" {
		t.Fatalf("manifest capabilities = %#v", manifest.Capabilities)
	}
	expectedOptions := map[string]analysis.OptionDescriptor{
		"config":        {Name: "config", Type: "string", Default: nil},
		"include_js":    {Name: "include_js", Type: "boolean", Default: false},
		"include_tests": {Name: "include_tests", Type: "boolean", Default: false},
		"runtime":       {Name: "runtime", Type: "string", Default: "auto", AllowedValues: []string{"auto", "esm", "cjs"}},
		"exclude":       {Name: "exclude", Type: "string[]", Default: []string{}},
	}
	if len(manifest.Options) != len(expectedOptions) {
		t.Fatalf("manifest option count = %d, want %d", len(manifest.Options), len(expectedOptions))
	}
	for _, option := range manifest.Options {
		expected, ok := expectedOptions[option.Name]
		if !ok || option.Type != expected.Type || !reflect.DeepEqual(option.Default, expected.Default) || !reflect.DeepEqual(option.AllowedValues, expected.AllowedValues) {
			t.Fatalf("manifest option %q = %#v", option.Name, option)
		}
	}
}

func TestDetectRecognizesTypeScriptProjectMarkers(t *testing.T) {
	root := t.TempDir()
	writeTSFixture(t, filepath.Join(root, "tsconfig.json"), `{ "compilerOptions": {} }`)
	writeTSFixture(t, filepath.Join(root, "package.json"), `{ "name": "fixture" }`)

	candidate, err := New().Detect(context.Background(), analysis.DetectRequest{ProjectRoot: root})
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if candidate.AnalyzerID != "org.archview.typescript" || candidate.Confidence != 1 || candidate.BoundaryHint != "tsconfig.json" {
		t.Fatalf("candidate = %#v", candidate)
	}
	if strings.Join(candidate.MatchedMarkers, ",") != "tsconfig.json,package.json" {
		t.Fatalf("matched markers = %#v", candidate.MatchedMarkers)
	}
}

func TestResolveProjectRequiresExplicitConfigForAmbiguousCandidates(t *testing.T) {
	root := t.TempDir()
	writeTSFixture(t, filepath.Join(root, "tsconfig.json"), `{ "compilerOptions": {} }`)
	writeTSFixture(t, filepath.Join(root, "tsconfig.app.json"), `{ "compilerOptions": {} }`)

	_, err := ResolveProject(root, tsOptions(t, nil))
	if analysis.ErrorCodeOf(err) != analysis.ErrModuleSelection {
		t.Fatalf("error code = %q, want %q; err=%v", analysis.ErrorCodeOf(err), analysis.ErrModuleSelection, err)
	}
	project, err := ResolveProject(root, tsOptions(t, map[string]any{"config": "tsconfig.app.json"}))
	if err != nil {
		t.Fatalf("resolve explicit config: %v", err)
	}
	if project.Boundary != "tsconfig.app.json" || project.ConfigPath != "tsconfig.app.json" {
		t.Fatalf("project boundary = %#v", project)
	}
}

func TestAnalyzeDiscoversConfiguredModulesAndHonorsSafeDefaults(t *testing.T) {
	root := t.TempDir()
	writeTSFixture(t, filepath.Join(root, "package.json"), `{"name":"fixture","type":"module"}`)
	writeTSFixture(t, filepath.Join(root, "tsconfig.base.json"), `{
  // inherited compiler settings are read as data
  "compilerOptions": {"baseUrl": ".", "outDir": "dist"}
}`)
	writeTSFixture(t, filepath.Join(root, "tsconfig.json"), `{
  "extends": "./tsconfig.base.json",
  "compilerOptions": {"paths": {"@/*": ["src/*"]}},
  "include": ["src/**/*"]
}`)
	writeTSFixture(t, filepath.Join(root, "src", "index.ts"), `export { service } from "./service";
export type { Contract } from "./contract";
`)
	writeTSFixture(t, filepath.Join(root, "src", "service.ts"), `import type { Contract } from "./contract";
export const service: Contract = {} as Contract;
`)
	writeTSFixture(t, filepath.Join(root, "src", "contract.tsx"), `export type Contract = { ready: boolean };
`)
	writeTSFixture(t, filepath.Join(root, "src", "ignored.test.ts"), `export const ignored = true;
`)
	writeTSFixture(t, filepath.Join(root, "src", "client.js"), `export const client = true;
`)
	writeTSFixture(t, filepath.Join(root, "dist", "built.ts"), `export const built = true;
`)
	writeTSFixture(t, filepath.Join(root, "node_modules", "pkg", "index.ts"), `export const external = true;
`)

	options := tsOptions(t, nil)
	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: options})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if result.Status != analysis.StatusComplete || result.Project.Boundary != "tsconfig.json" || result.Project.ModuleRoot != "." {
		t.Fatalf("result metadata = %#v; diagnostics=%#v", result.Project, result.Diagnostics)
	}
	for _, id := range []string{"ts:module:src/index", "ts:module:src/service", "ts:module:src/contract"} {
		if !hasTSModule(result.Modules, id) {
			t.Fatalf("missing module %q: %#v", id, tsModuleIDs(result.Modules))
		}
	}
	for _, id := range []string{"ts:module:src/ignored.test", "ts:module:src/client", "ts:module:dist/built", "ts:module:node_modules/pkg/index"} {
		if hasTSModule(result.Modules, id) {
			t.Fatalf("default-excluded module %q was discovered: %#v", id, tsModuleIDs(result.Modules))
		}
	}
	if result.Modules == nil || result.SourceReferences == nil || result.Relationships == nil || result.References == nil {
		t.Fatalf("observation collections must be non-nil: %#v", result)
	}

	repeat, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: options})
	if err != nil {
		t.Fatalf("repeat analyze: %v", err)
	}
	firstJSON, _ := json.Marshal(result)
	repeatJSON, _ := json.Marshal(repeat)
	if string(firstJSON) != string(repeatJSON) {
		t.Fatalf("analysis is not byte-stable:\n%s\n%s", firstJSON, repeatJSON)
	}

	withJSAndTests := tsOptions(t, map[string]any{"include_js": true, "include_tests": true})
	expanded, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: withJSAndTests})
	if err != nil {
		t.Fatalf("expanded analyze: %v", err)
	}
	if !hasTSModule(expanded.Modules, "ts:module:src/client") || !hasTSModule(expanded.Modules, "ts:module:src/ignored.test") {
		t.Fatalf("explicit scope options were not applied: %#v", tsModuleIDs(expanded.Modules))
	}
}

func TestAnalyzeUsesConfiguredPatternsWithoutEscapingNestedConfigDirectory(t *testing.T) {
	root := t.TempDir()
	writeTSFixture(t, filepath.Join(root, "config", "tsconfig.json"), `{}`)
	writeTSFixture(t, filepath.Join(root, "config", "local.ts"), `export const local = true;`)
	writeTSFixture(t, filepath.Join(root, "sibling.ts"), `export const sibling = true;`)

	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: tsOptions(t, map[string]any{"config": "config/tsconfig.json"})})
	if err != nil {
		t.Fatalf("analyze nested config: %v", err)
	}
	if !hasTSModule(result.Modules, "ts:module:config/local") || hasTSModule(result.Modules, "ts:module:sibling") {
		t.Fatalf("default nested-config scope was incorrect: %#v", result.Modules)
	}
}

func TestAnalyzeDoesNotTreatRootDirAsTheOnlyConfiguredInputRoot(t *testing.T) {
	root := t.TempDir()
	writeTSFixture(t, filepath.Join(root, "tsconfig.json"), `{"compilerOptions":{"rootDir":"src"},"include":["shared/**/*"]}`)
	writeTSFixture(t, filepath.Join(root, "src", "ignored.ts"), `export const ignored = true;`)
	writeTSFixture(t, filepath.Join(root, "shared", "included.ts"), `export const included = true;`)

	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: tsOptions(t, nil)})
	if err != nil {
		t.Fatalf("analyze rootDir fixture: %v", err)
	}
	if !hasTSModule(result.Modules, "ts:module:shared/included") || hasTSModule(result.Modules, "ts:module:src/ignored") {
		t.Fatalf("rootDir incorrectly constrained configured inputs: %#v", result.Modules)
	}
}

func TestAnalyzeResolvesStaticDependenciesAndRetainsUncertainty(t *testing.T) {
	root := t.TempDir()
	writeTSFixture(t, filepath.Join(root, "package.json"), `{"name":"fixture","type":"module","imports":{"#feature":"./src/feature.ts"},"exports":{".":"./src/index.ts","./feature":"./src/feature.ts"}}`)
	writeTSFixture(t, filepath.Join(root, "tsconfig.json"), `{
  "compilerOptions": {"baseUrl":".","paths":{"@/*":["src/*"]}},
  "include": ["src/**/*"]
}`)
	writeTSFixture(t, filepath.Join(root, "src", "index.ts"), `import { feature as importedFeature } from "@/feature";
import type { Contract } from "./types";
export { feature } from "./feature";
export type { Contract as PublicContract } from "./types";
const service = require("./service");
const lazy = import("./lazy");
const computed = import("./" + name);
import fs from "node:fs";
import external from "react";
import "./missing";
import "fixture/feature";
import "fixture/private";
import "#feature";
import "exported/private";
`)
	writeTSFixture(t, filepath.Join(root, "src", "feature.ts"), `export const feature = true;`)
	writeTSFixture(t, filepath.Join(root, "src", "types.ts"), `export type Contract = { ready: boolean };`)
	writeTSFixture(t, filepath.Join(root, "src", "service.ts"), `export const service = true;`)
	writeTSFixture(t, filepath.Join(root, "src", "lazy.ts"), `export const lazy = true;`)
	writeTSFixture(t, filepath.Join(root, "node_modules", "exported", "package.json"), `{"name":"exported","exports":{".":"./index.js"}}`)
	writeTSFixture(t, filepath.Join(root, "node_modules", "exported", "index.js"), `export const exported = true;`)

	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: tsOptions(t, nil)})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if result.Status != analysis.StatusPartial {
		t.Fatalf("status = %q, diagnostics=%#v", result.Status, result.Diagnostics)
	}
	if !hasTSRelationshipToModule(result, "ts:module:src/index", "ts:module:src/feature") {
		t.Fatalf("alias/reexport/self-package/imports local edges missing: %#v", result.Relationships)
	}
	if !hasTSRelationshipToModule(result, "ts:module:src/index", "ts:module:src/types") || !hasTSRelationshipToModule(result, "ts:module:src/index", "ts:module:src/service") || !hasTSRelationshipToModule(result, "ts:module:src/index", "ts:module:src/lazy") {
		t.Fatalf("static local edges missing: %#v", result.Relationships)
	}
	aliasEdge := findTSRelationshipToModule(result, "ts:module:src/index", "ts:module:src/feature")
	if !metadataContains(aliasEdge.Metadata, "resolution_kinds", "path_alias") || !metadataContains(aliasEdge.Metadata, "aliases", "@/*") {
		t.Fatalf("alias provenance = %#v", aliasEdge.Metadata)
	}
	typeEdge := findTSRelationshipToModule(result, "ts:module:src/index", "ts:module:src/types")
	if typeEdge.Metadata["type_only"] != true {
		t.Fatalf("type-only provenance = %#v", typeEdge.Metadata)
	}
	if !metadataContains(typeEdge.Metadata, "import_kinds", "reexport") || !metadataContains(typeEdge.Metadata, "import_kinds", "type_import") {
		t.Fatalf("type/re-export kinds = %#v", typeEdge.Metadata)
	}
	if !hasTSReference(result, "standard_library", "node:fs") || !hasTSReference(result, "external", "react") || !hasTSReference(result, "unresolved", "./missing") || !hasTSReference(result, "unresolved", "fixture/private") || !hasTSReference(result, "unresolved", "exported/private") || !hasTSReference(result, "dynamic", `"./"+name`) {
		t.Fatalf("reference scopes = %#v", result.References)
	}
	if !hasTSDiagnostic(result, "typescript_dynamic_import") || !hasTSDiagnostic(result, "typescript_unresolved_relative_import") || !hasTSDiagnostic(result, "typescript_package_exports_unresolved") {
		t.Fatalf("uncertainty diagnostics = %#v", result.Diagnostics)
	}
	for _, source := range result.SourceReferences {
		if source.Start != nil && (source.Start.Line < 1 || source.Start.Column < 1) {
			t.Fatalf("invalid source position = %#v", source)
		}
	}
	firstJSON, _ := json.Marshal(result)
	repeat, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: tsOptions(t, nil)})
	if err != nil {
		t.Fatalf("repeat analyze: %v", err)
	}
	repeatJSON, _ := json.Marshal(repeat)
	if string(firstJSON) != string(repeatJSON) {
		t.Fatalf("dependency analysis is not byte-stable:\n%s\n%s", firstJSON, repeatJSON)
	}
}

func TestAnalyzeUsesPackageConditionsRootDirsAndJavaScriptExtensionMapping(t *testing.T) {
	root := t.TempDir()
	writeTSFixture(t, filepath.Join(root, "package.json"), `{"name":"app","type":"module","exports":{".":{"import":"./src/esm.ts","require":"./src/cjs.ts"}}}`)
	writeTSFixture(t, filepath.Join(root, "tsconfig.json"), `{
  "compilerOptions": {"rootDirs":["src","overlay"]},
  "include": ["src/**/*", "overlay/**/*"]
}`)
	writeTSFixture(t, filepath.Join(root, "src", "main.ts"), `import "./mirror";
import "./client.js";
import "mirror";
import "app";
`)
	writeTSFixture(t, filepath.Join(root, "src", "client.ts"), `export const client = true;`)
	writeTSFixture(t, filepath.Join(root, "src", "esm.ts"), `export const mode = "esm";`)
	writeTSFixture(t, filepath.Join(root, "src", "cjs.ts"), `export const mode = "cjs";`)
	writeTSFixture(t, filepath.Join(root, "overlay", "mirror.ts"), `export const mirror = true;`)

	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: tsOptions(t, map[string]any{"include_js": true, "runtime": "esm"})})
	if err != nil {
		t.Fatalf("analyze esm: %v", err)
	}
	for _, target := range []string{"ts:module:overlay/mirror", "ts:module:src/client", "ts:module:src/esm"} {
		if !hasTSRelationshipToModule(result, "ts:module:src/main", target) {
			t.Fatalf("missing ESM/rootDirs edge to %q: %#v", target, result.Relationships)
		}
	}
	if !hasTSReference(result, "external", "mirror") {
		t.Fatalf("bare import incorrectly reused rootDirs as a package resolver: %#v", result.Relationships)
	}
	if hasTSRelationshipToModule(result, "ts:module:src/main", "ts:module:src/cjs") {
		t.Fatalf("CJS export condition was selected for ESM runtime: %#v", result.Relationships)
	}
	result, err = New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: tsOptions(t, map[string]any{"runtime": "cjs"})})
	if err != nil {
		t.Fatalf("analyze cjs: %v", err)
	}
	if !hasTSRelationshipToModule(result, "ts:module:src/main", "ts:module:src/cjs") {
		t.Fatalf("CJS export condition was not selected: %#v", result.Relationships)
	}
}

func TestAnalyzeRejectsPackageMetadataTargetsOutsidePackageDirectory(t *testing.T) {
	root := t.TempDir()
	writeTSFixture(t, filepath.Join(root, "tsconfig.json"), `{"include":["**/*"]}`)
	writeTSFixture(t, filepath.Join(root, "pkg", "package.json"), `{"main":"../secret.ts"}`)
	writeTSFixture(t, filepath.Join(root, "pkg", "index.ts"), `export const local = true;`)
	writeTSFixture(t, filepath.Join(root, "secret.ts"), `export const secret = true;`)
	writeTSFixture(t, filepath.Join(root, "main.ts"), `import "./pkg";`)

	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: tsOptions(t, nil)})
	if err != nil {
		t.Fatalf("analyze package metadata escape: %v", err)
	}
	if hasTSRelationshipToModule(result, "ts:module:main", "ts:module:secret") {
		t.Fatalf("package metadata escaped its package directory: %#v", result.Relationships)
	}
}

func TestAnalyzeDoesNotFallBackThroughBlockedDirectoryExports(t *testing.T) {
	root := t.TempDir()
	writeTSFixture(t, filepath.Join(root, "tsconfig.json"), `{"include":["**/*"]}`)
	writeTSFixture(t, filepath.Join(root, "pkg", "package.json"), `{"exports":{".":null},"main":"./index.ts"}`)
	writeTSFixture(t, filepath.Join(root, "pkg", "index.ts"), `export const local = true;`)
	writeTSFixture(t, filepath.Join(root, "main.ts"), `import "./pkg";`)

	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: tsOptions(t, nil)})
	if err != nil {
		t.Fatalf("analyze blocked directory exports: %v", err)
	}
	if hasTSRelationshipToModule(result, "ts:module:main", "ts:module:pkg/index") || !hasTSDiagnostic(result, "typescript_package_exports_unresolved") {
		t.Fatalf("blocked package exports were bypassed: relationships=%#v diagnostics=%#v", result.Relationships, result.Diagnostics)
	}
}

func TestAnalyzeRetainsModulesForMalformedConfigurationAndSyntax(t *testing.T) {
	root := t.TempDir()
	writeTSFixture(t, filepath.Join(root, "tsconfig.json"), `{"compilerOptions":"invalid","extends":true,"include":["src/**/*"]}`)
	writeTSFixture(t, filepath.Join(root, "src", "main.ts"), `import { broken from "./valid";
export const main = true;
`)
	writeTSFixture(t, filepath.Join(root, "src", "valid.ts"), `export const valid = true;`)

	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: tsOptions(t, nil)})
	if err != nil {
		t.Fatalf("analyze malformed fixture: %v", err)
	}
	if result.Status != analysis.StatusPartial || !hasTSDiagnostic(result, "typescript_compiler_options_invalid") || !hasTSDiagnostic(result, "typescript_extends_invalid") || !hasTSDiagnostic(result, "typescript_import_syntax") {
		t.Fatalf("malformed result = %#v", result)
	}
	if !hasTSModule(result.Modules, "ts:module:src/main") || !hasTSModule(result.Modules, "ts:module:src/valid") {
		t.Fatalf("malformed fixture lost usable modules: %#v", result.Modules)
	}
}

func TestAnalyzeIncludesJavaScriptWhenAllowJSIsConfigured(t *testing.T) {
	root := t.TempDir()
	writeTSFixture(t, filepath.Join(root, "tsconfig.json"), `{"compilerOptions":{"allowJs":true},"include":["src/**/*"]}`)
	writeTSFixture(t, filepath.Join(root, "src", "main.ts"), `import "./client";`)
	writeTSFixture(t, filepath.Join(root, "src", "client.js"), `export const client = true;`)
	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: tsOptions(t, nil)})
	if err != nil {
		t.Fatalf("analyze allowJs fixture: %v", err)
	}
	if !hasTSModule(result.Modules, "ts:module:src/client") || !hasTSRelationshipToModule(result, "ts:module:src/main", "ts:module:src/client") {
		t.Fatalf("allowJs did not include or resolve JavaScript: modules=%#v relationships=%#v", result.Modules, result.Relationships)
	}
}

func TestExtractImportsSupportsMultilineLiteralsAndSkipsMemberRequire(t *testing.T) {
	content := `const lazy = import(
  "./lazy",
  { with: { type: "json" } },
);
const computed = import("./" + moduleName);
const member = object.require("./member");
const regular = require(
  "./regular"
);
const regex = /require\("\.\/regex"\)/;
const regexDynamic = /import\("\.\/dynamic"\)/;
const memberImport = object.import("./member-import");
if (ready) /require\("\.\/control-regex"\)/;
const ratio = /a/ / require("./regular");
const numericRatio = 10 / require("./numeric");`
	observations, diagnostics := extractTSImports("src/main.ts", content, "ts:module:src/main")
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected import diagnostics: %#v", diagnostics)
	}
	if len(observations) != 5 {
		t.Fatalf("observations = %#v", observations)
	}
	if observations[0].Specifier != "./lazy" || !observations[0].Dynamic || observations[0].Computed {
		t.Fatalf("multiline literal dynamic import = %#v", observations[0])
	}
	if observations[1].Specifier != "<computed>" || !observations[1].Dynamic || !observations[1].Computed {
		t.Fatalf("computed dynamic import = %#v", observations[1])
	}
	if observations[2].Specifier != "./regular" || observations[2].Kind != "require" || observations[2].Dynamic {
		t.Fatalf("multiline require = %#v", observations[2])
	}
	if observations[3].Specifier != "./regular" || observations[3].Kind != "require" || observations[3].Dynamic {
		t.Fatalf("require after regex division = %#v", observations[3])
	}
	if observations[4].Specifier != "./numeric" || observations[4].Kind != "require" || observations[4].Dynamic {
		t.Fatalf("require after numeric division = %#v", observations[4])
	}
}

func TestExtractImportsTreatsDelimiterTextInsideStringsAsModuleSpecifiers(t *testing.T) {
	content := `import "(";
import ")";
import ";";
const close = import(")");
const member = object.require(")");`
	observations, diagnostics := extractTSImports("src/main.ts", content, "ts:module:src/main")
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected delimiter-text diagnostics: %#v", diagnostics)
	}
	if len(observations) != 4 {
		t.Fatalf("observations = %#v", observations)
	}
	for index, expected := range []string{"(", ")", ";", ")"} {
		if observations[index].Specifier != expected {
			t.Fatalf("observation %d = %#v", index, observations[index])
		}
	}
	if observations[3].Kind != "dynamic_import" || !observations[3].Dynamic || observations[3].Computed {
		t.Fatalf("literal delimiter dynamic import = %#v", observations[3])
	}
}

func TestExtractImportsTreatsSideEffectTypeSpecifierAsValueImport(t *testing.T) {
	observations, diagnostics := extractTSImports("src/main.ts", `import "type";
	import type from "./default";
import type { Value } from "./types";`, "ts:module:src/main")
	if len(diagnostics) != 0 || len(observations) != 3 {
		t.Fatalf("type side-effect observations=%#v diagnostics=%#v", observations, diagnostics)
	}
	if observations[0].Kind != "import" || observations[0].TypeOnly || observations[0].Specifier != "type" {
		t.Fatalf("side-effect import was classified as a type import: %#v", observations[0])
	}
	if observations[1].Kind != "import" || observations[1].TypeOnly || !containsStringValue(observations[1].ImportNames, "type") {
		t.Fatalf("default binding named type was classified as a type import: %#v", observations[1])
	}
}

func TestExtractImportsReportsUnterminatedCalls(t *testing.T) {
	observations, diagnostics := extractTSImports("src/main.ts", `const lazy = import("./lazy");
const broken = require("./broken"`, "ts:module:src/main")
	if len(observations) != 1 || observations[0].Specifier != "./lazy" {
		t.Fatalf("valid call was not retained: %#v", observations)
	}
	if len(diagnostics) != 1 || diagnostics[0].Code != "typescript_import_syntax" {
		t.Fatalf("unterminated call diagnostic = %#v", diagnostics)
	}
}

func TestExtractImportsDistinguishesTypeBindingFromTypeModifier(t *testing.T) {
	content := `import { type } from "./value";
import { type Foo, type Bar as Baz } from "./types";`
	observations, diagnostics := extractTSImports("src/main.ts", content, "ts:module:src/main")
	if len(diagnostics) != 0 || len(observations) != 2 {
		t.Fatalf("type import observations=%#v diagnostics=%#v", observations, diagnostics)
	}
	if observations[0].TypeOnly || observations[0].Kind != "import" || !containsStringValue(observations[0].ImportNames, "type") {
		t.Fatalf("named type binding was classified as type-only: %#v", observations[0])
	}
	if !observations[1].TypeOnly || observations[1].Kind != "type_import" {
		t.Fatalf("type modifiers were not classified as type-only: %#v", observations[1])
	}
}

func containsStringValue(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func TestAnalyzeUsesImportKindForAutoPackageConditions(t *testing.T) {
	root := t.TempDir()
	writeTSFixture(t, filepath.Join(root, "package.json"), `{"name":"app","exports":{".":{"import":"./src/esm.ts","require":"./src/cjs.ts"}}}`)
	writeTSFixture(t, filepath.Join(root, "tsconfig.json"), `{"include":["src/**/*"]}`)
	writeTSFixture(t, filepath.Join(root, "src", "main.ts"), `import "app";
const cjs = require("app");`)
	writeTSFixture(t, filepath.Join(root, "src", "esm.ts"), `export const mode = "esm";`)
	writeTSFixture(t, filepath.Join(root, "src", "cjs.ts"), `export const mode = "cjs";`)

	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: tsOptions(t, nil)})
	if err != nil {
		t.Fatalf("analyze auto package conditions: %v", err)
	}
	if !hasTSRelationshipToModule(result, "ts:module:src/main", "ts:module:src/esm") || !hasTSRelationshipToModule(result, "ts:module:src/main", "ts:module:src/cjs") {
		t.Fatalf("auto package conditions did not follow import kind: %#v", result.Relationships)
	}
	esm := findTSRelationshipToModule(result, "ts:module:src/main", "ts:module:src/esm")
	cjs := findTSRelationshipToModule(result, "ts:module:src/main", "ts:module:src/cjs")
	if esm.Metadata["runtime_context"] != "esm" || cjs.Metadata["runtime_context"] != "cjs" {
		t.Fatalf("auto package runtime provenance = esm=%#v cjs=%#v", esm.Metadata, cjs.Metadata)
	}
}

func TestAnalyzeDoesNotUseCaseSensitiveModuleLookupOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("filesystem case semantics differ outside Windows")
	}
	root := t.TempDir()
	writeTSFixture(t, filepath.Join(root, "tsconfig.json"), `{"include":["src/**/*"]}`)
	writeTSFixture(t, filepath.Join(root, "src", "main.ts"), `import "./feature";`)
	writeTSFixture(t, filepath.Join(root, "src", "Feature.ts"), `export const feature = true;`)

	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: tsOptions(t, nil)})
	if err != nil {
		t.Fatalf("analyze case-insensitive import: %v", err)
	}
	if !hasTSRelationshipToModule(result, "ts:module:src/main", "ts:module:src/Feature") {
		t.Fatalf("case-insensitive import was not resolved locally: %#v", result.Relationships)
	}
}

func TestAnalyzeDoesNotReportCaseSensitiveConfiguredFileMissingOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("filesystem case semantics differ outside Windows")
	}
	root := t.TempDir()
	writeTSFixture(t, filepath.Join(root, "tsconfig.json"), `{"files":["src/feature.ts"]}`)
	writeTSFixture(t, filepath.Join(root, "src", "Feature.ts"), `export const feature = true;`)

	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: tsOptions(t, nil)})
	if err != nil {
		t.Fatalf("analyze case-insensitive configured file: %v", err)
	}
	if hasTSDiagnostic(result, "typescript_configured_file_unavailable") {
		t.Fatalf("case-insensitive configured file was reported missing: %#v", result.Diagnostics)
	}
}

func TestPackageMetadataCannotEscapeProjectRootThroughSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	outsidePackage := filepath.Join(outside, "package.json")
	writeTSFixture(t, outsidePackage, `{"name":"outside"}`)
	linkPath := filepath.Join(root, "package.json")
	if err := os.Symlink(outsidePackage, linkPath); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}

	context, diagnostics := readPackageContext(root, root)
	if context.Name != "" || context.Path != "" {
		t.Fatalf("escaped package metadata was read: %#v", context)
	}
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "typescript_package_json_outside_root" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing escaped package diagnostic: %#v", diagnostics)
	}
}

func TestAnalyzeTreatsNullConfigValuesAsMalformed(t *testing.T) {
	root := t.TempDir()
	writeTSFixture(t, filepath.Join(root, "tsconfig.json"), `{"extends":null,"compilerOptions":null,"include":null}`)
	writeTSFixture(t, filepath.Join(root, "main.ts"), `export const main = true;`)

	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: tsOptions(t, nil)})
	if err != nil {
		t.Fatalf("analyze null config: %v", err)
	}
	if result.Status != analysis.StatusPartial || !hasTSDiagnostic(result, "typescript_configuration_invalid") {
		t.Fatalf("null config was not reported as partial invalid configuration: %#v", result)
	}
}

func TestAnalyzeTreatsNullConfigRootAsMalformed(t *testing.T) {
	root := t.TempDir()
	writeTSFixture(t, filepath.Join(root, "tsconfig.json"), `null`)
	writeTSFixture(t, filepath.Join(root, "main.ts"), `export const main = true;`)

	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: tsOptions(t, nil)})
	if err != nil {
		t.Fatalf("analyze null config root: %v", err)
	}
	if result.Status != analysis.StatusPartial || !hasTSDiagnostic(result, "typescript_configuration_invalid") {
		t.Fatalf("null config root was not reported as partial invalid configuration: %#v", result)
	}
}

func TestAnalyzeKeepsDistinctIDsForSameStemAcrossLanguages(t *testing.T) {
	root := t.TempDir()
	writeTSFixture(t, filepath.Join(root, "tsconfig.json"), `{"compilerOptions":{"allowJs":true},"include":["**/*"]}`)
	writeTSFixture(t, filepath.Join(root, "feature.ts"), `export const typed = true;`)
	writeTSFixture(t, filepath.Join(root, "feature.js"), `export const untyped = true;`)

	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: tsOptions(t, nil)})
	if err != nil {
		t.Fatalf("analyze same-stem fixture: %v", err)
	}
	if len(result.Modules) != 2 || result.Modules[0].ID == result.Modules[1].ID {
		t.Fatalf("same-stem modules did not receive unique IDs: %#v", result.Modules)
	}
}

func tsOptions(t *testing.T, values map[string]any) analysis.EffectiveOptions {
	t.Helper()
	options, err := analysis.ResolveOptions(New().Manifest(), nil, values)
	if err != nil {
		t.Fatalf("resolve TypeScript options: %v", err)
	}
	return options
}

func writeTSFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}

func hasTSModule(values []analysis.ModuleObservation, id string) bool {
	for _, value := range values {
		if value.ID == id {
			return true
		}
	}
	return false
}

func tsModuleIDs(values []analysis.ModuleObservation) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.ID)
	}
	return result
}

func hasTSRelationshipToModule(result analysis.AnalysisResult, from, to string) bool {
	return findTSRelationshipToModule(result, from, to).ID != ""
}

func findTSRelationshipToModule(result analysis.AnalysisResult, from, to string) analysis.RelationshipObservation {
	for _, relationship := range result.Relationships {
		if relationship.FromModuleID == from && relationship.ToModuleID == to {
			return relationship
		}
	}
	return analysis.RelationshipObservation{}
}

func hasTSReference(result analysis.AnalysisResult, scope, name string) bool {
	for _, reference := range result.References {
		if reference.Scope == scope && reference.Name == name {
			return true
		}
	}
	return false
}

func hasTSDiagnostic(result analysis.AnalysisResult, code string) bool {
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func metadataContains(metadata map[string]any, key, expected string) bool {
	values, ok := metadata[key].([]string)
	if !ok {
		return false
	}
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
