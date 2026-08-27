package main

import (
	"bytes"
	"flag"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/viewer"
	"github.com/buffo/arch-view/internal/viewer/scene"
)

func TestHelpListsTypeScriptSelectionAndOptions(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("help exit code = %d, stderr=%s", code, stderr.String())
	}
	for _, flag := range []string{"--config <tsconfig path>", "--include-js", "--include-tests", "--runtime auto|esm|cjs", "--exclude <glob>"} {
		if !strings.Contains(stdout.String(), flag) {
			t.Fatalf("help output is missing %q: %s", flag, stdout.String())
		}
	}
}

func TestTypeScriptCLIOptionsAreForwardedOnlyWhenSelected(t *testing.T) {
	fs := flag.NewFlagSet("typescript-options", flag.ContinueOnError)
	config := fs.String("config", "", "")
	includeJS := fs.Bool("include-js", false, "")
	includeTests := fs.Bool("include-tests", false, "")
	runtime := fs.String("runtime", "auto", "")
	var excludes stringList
	fs.Var(&excludes, "exclude", "")
	if err := fs.Parse([]string{"--config", "tsconfig.app.json", "--include-js", "--include-tests=false", "--runtime", "cjs", "--exclude", "src/generated/**"}); err != nil {
		t.Fatalf("parse options: %v", err)
	}
	emptyString := ""
	falseValue := false
	emptyList := stringList{}
	options := collectAnalyzerCLIOptions(fs, analyzerCLIFlags{
		module:           &emptyString,
		config:           config,
		includeJS:        includeJS,
		includeTests:     includeTests,
		includeGenerated: &falseValue,
		includeExternal:  &falseValue,
		safeMode:         &falseValue,
		buildTags:        &emptyList,
		excludes:         &excludes,
		sourceRoots:      &emptyList,
		pythonVersion:    &emptyString,
		includeStubs:     &falseValue,
		runtime:          runtime,
	})
	if len(options) != 5 || options["config"] != "tsconfig.app.json" || options["include_js"] != true || options["include_tests"] != false || options["runtime"] != "cjs" {
		t.Fatalf("selected TypeScript options = %#v", options)
	}
	if values, ok := options["exclude"].([]string); !ok || len(values) != 1 || values[0] != "src/generated/**" {
		t.Fatalf("selected TypeScript exclusions = %#v", options)
	}
	for _, key := range []string{"module", "source_roots", "python_version", "include_stubs", "safe_mode"} {
		if _, ok := options[key]; ok {
			t.Fatalf("generic option %q leaked into selected TypeScript options: %#v", key, options)
		}
	}
}

func TestTypeScriptCLISelectionAndAnalysisAreDeterministic(t *testing.T) {
	root := writeVisibleTypeScriptProject(t)
	firstPath := filepath.Join(t.TempDir(), "first-analysis.json")
	secondPath := filepath.Join(t.TempDir(), "second-analysis.json")
	args := typescriptAnalysisArgs(root, firstPath)
	if code, _, stderr := runTypeScriptCommand(args...); code != 0 {
		t.Fatalf("explicit TypeScript analyze exit code = %d, stderr=%s", code, stderr)
	}
	for index := range args {
		if args[index] == "--output" && index+1 < len(args) {
			args[index+1] = secondPath
			break
		}
	}
	if code, _, stderr := runTypeScriptCommand(args...); code != 0 {
		t.Fatalf("repeated TypeScript analyze exit code = %d, stderr=%s", code, stderr)
	}
	if string(readTestFile(t, firstPath)) != string(readTestFile(t, secondPath)) {
		t.Fatalf("repeated TypeScript CLI analysis is not byte-stable")
	}

	var result analysis.AnalysisResult
	decodeTestJSON(t, readTestFile(t, firstPath), &result)
	if result.Status != analysis.StatusPartial || result.Analyzer.ID != "org.archview.typescript" || result.Analyzer.Language != "typescript" {
		t.Fatalf("TypeScript selection metadata = %#v", result)
	}
	if result.Project.Boundary != "tsconfig.json" || result.OptionsFingerprint == "" || result.RunID == "" {
		t.Fatalf("TypeScript boundary/fingerprint metadata = %#v", result)
	}
	for _, moduleID := range []string{"ts:module:src/main", "ts:module:src/client", "ts:module:src/visible.test"} {
		if !hasAnalysisModule(result.Modules, moduleID) {
			t.Fatalf("TypeScript option fixture did not include %q; modules=%#v", moduleID, analysisModuleIDs(result.Modules))
		}
	}
	if hasAnalysisModule(result.Modules, "ts:module:src/excluded") {
		t.Fatalf("TypeScript --exclude option was not applied: %#v", analysisModuleIDs(result.Modules))
	}
	if !hasAnalysisDiagnostic(result.Diagnostics, "typescript_dynamic_import") || !hasAnalysisDiagnostic(result.Diagnostics, "typescript_unresolved_relative_import") {
		t.Fatalf("TypeScript uncertainty diagnostics are missing: %#v", result.Diagnostics)
	}

	autoPath := filepath.Join(t.TempDir(), "auto-analysis.json")
	if code, _, stderr := runTypeScriptCommand("analyze", "--project", root, "--format", "analysis-json", "--output", autoPath); code != 0 {
		t.Fatalf("auto-detected TypeScript analyze exit code = %d, stderr=%s", code, stderr)
	}
	var autoResult analysis.AnalysisResult
	decodeTestJSON(t, readTestFile(t, autoPath), &autoResult)
	if autoResult.Analyzer.ID != "org.archview.typescript" || autoResult.Project.Boundary != "tsconfig.json" {
		t.Fatalf("auto-detected TypeScript metadata = %#v", autoResult)
	}

	idPath := filepath.Join(t.TempDir(), "id-analysis.json")
	if code, _, stderr := runTypeScriptCommand("analyze", "--project", root, "--analyzer", "org.archview.typescript", "--output", idPath, "--format", "analysis-json"); code != 0 {
		t.Fatalf("explicit TypeScript analyzer-id exit code = %d, stderr=%s", code, stderr)
	}
	var idResult analysis.AnalysisResult
	decodeTestJSON(t, readTestFile(t, idPath), &idResult)
	if idResult.Analyzer.ID != "org.archview.typescript" || idResult.Project.Boundary != "tsconfig.json" {
		t.Fatalf("explicit TypeScript analyzer-id metadata = %#v", idResult)
	}
}

func TestTypeScriptVisibleJourneyUsesSharedModelViewerAndExportPaths(t *testing.T) {
	root := writeVisibleTypeScriptProject(t)
	analysisPath := filepath.Join(t.TempDir(), "analysis.json")
	if code, _, stderr := runTypeScriptCommand(typescriptAnalysisArgs(root, analysisPath)...); code != 0 {
		t.Fatalf("TypeScript analysis exit code = %d, stderr=%s", code, stderr)
	}

	modelPath := filepath.Join(t.TempDir(), "model.json")
	if code, _, stderr := runTypeScriptCommand("model", "normalize", "--input", analysisPath, "--output", modelPath); code != 0 {
		t.Fatalf("TypeScript model normalize exit code = %d, stderr=%s", code, stderr)
	}
	var normalized model.Model
	decodeTestJSON(t, readTestFile(t, modelPath), &normalized)
	if normalized.Project.Language != "typescript" || normalized.Status != model.StatusPartial || len(normalized.Relationships) == 0 || len(normalized.References) == 0 || len(normalized.SourceReferences) == 0 || len(normalized.Diagnostics) == 0 {
		t.Fatalf("normalized TypeScript model lost shared observations: %#v", normalized)
	}
	if code, stdout, stderr := runTypeScriptCommand("model", "validate", "--input", modelPath); code != 0 || !strings.Contains(stdout, `"valid": true`) {
		t.Fatalf("TypeScript model validate exit code = %d, stdout=%s, stderr=%s", code, stdout, stderr)
	}
	projectionPath := filepath.Join(t.TempDir(), "projection.json")
	if code, _, stderr := runTypeScriptCommand("model", "projection", "--input", modelPath, "--output", projectionPath); code != 0 {
		t.Fatalf("TypeScript model projection exit code = %d, stderr=%s", code, stderr)
	}
	var projection model.HierarchyProjection
	decodeTestJSON(t, readTestFile(t, projectionPath), &projection)
	if len(projection.Nodes) == 0 || len(projection.Relationships) == 0 {
		t.Fatalf("TypeScript hierarchy projection = %#v", projection)
	}

	for _, format := range []string{"json", "html", "svg"} {
		firstPath := filepath.Join(t.TempDir(), "architecture."+format)
		secondPath := filepath.Join(t.TempDir(), "architecture-repeat."+format)
		if code, _, stderr := runTypeScriptCommand(typescriptExportAnalyzeArgs(root, firstPath, format)...); code != 0 {
			t.Fatalf("TypeScript %s export exit code = %d, stderr=%s", format, code, stderr)
		}
		if code, _, stderr := runTypeScriptCommand(typescriptExportAnalyzeArgs(root, secondPath, format)...); code != 0 {
			t.Fatalf("repeated TypeScript %s export exit code = %d, stderr=%s", format, code, stderr)
		}
		if string(readTestFile(t, firstPath)) != string(readTestFile(t, secondPath)) {
			t.Fatalf("repeated TypeScript %s export is not byte-stable", format)
		}
		data := readTestFile(t, firstPath)
		switch format {
		case "json":
			var exported model.Model
			decodeTestJSON(t, data, &exported)
			if exported.Project.Language != "typescript" || len(exported.Relationships) == 0 || len(exported.References) == 0 {
				t.Fatalf("TypeScript canonical JSON export lost semantics: %#v", exported)
			}
		case "html":
			html := string(data)
			for _, marker := range []string{"window.__ARCH_VIEW_EXPORT__", "ts:module:src/main", "Download SVG", "Full canvas"} {
				if !strings.Contains(html, marker) {
					t.Fatalf("TypeScript HTML export is missing %q", marker)
				}
			}
			if strings.Contains(html, "<script src=") || strings.Contains(html, `<link rel="stylesheet"`) {
				t.Fatalf("TypeScript HTML export is not self-contained")
			}
		case "svg":
			svg := string(data)
			for _, marker := range []string{"<svg ", `data-module-id="`, `data-relationship-id="`, "ts:module:src/main"} {
				if !strings.Contains(svg, marker) {
					t.Fatalf("TypeScript SVG export is missing %q", marker)
				}
			}
		}
	}

	viewerServer, err := viewer.NewServer(normalized, viewer.ServerOptions{SourceRoot: root})
	if err != nil {
		t.Fatalf("NewServer(TypeScript model): %v", err)
	}
	httpServer := httptest.NewServer(viewerServer.Handler())
	defer httpServer.Close()

	rootResponse, rootBody := getViewerResponse(t, httpServer.URL+"/")
	if rootResponse.StatusCode != http.StatusOK || !strings.Contains(string(rootBody), "Download SVG") || !strings.Contains(string(rootBody), "Full canvas") || !strings.Contains(string(rootBody), normalized.ModelID) {
		t.Fatalf("TypeScript viewer root response = %d %s", rootResponse.StatusCode, rootBody)
	}
	modelResponse, modelBody := getViewerResponse(t, httpServer.URL+"/v1/models/"+url.PathEscape(normalized.ModelID))
	var viewerModel model.Model
	decodeTestJSON(t, modelBody, &viewerModel)
	if modelResponse.StatusCode != http.StatusOK || viewerModel.Project.Language != "typescript" {
		t.Fatalf("TypeScript viewer model response = %d %#v", modelResponse.StatusCode, viewerModel)
	}

	query := url.Values{"mode": []string{"list"}, "reference_visibility": []string{"expanded"}}
	sceneResponse, sceneBody := getViewerResponse(t, httpServer.URL+"/v1/models/"+url.PathEscape(normalized.ModelID)+"/projection?"+query.Encode())
	var snapshot scene.SceneSnapshot
	decodeTestJSON(t, sceneBody, &snapshot)
	if sceneResponse.StatusCode != http.StatusOK || snapshot.Project.Language != "typescript" || len(snapshot.VisibleNodes) == 0 || len(snapshot.VisibleRelationships) == 0 || len(snapshot.ReferenceDetails) == 0 || len(snapshot.EvidenceLinks) == 0 || len(snapshot.DiagnosticIndicators) == 0 {
		t.Fatalf("TypeScript viewer expanded scene = %d %#v", sceneResponse.StatusCode, snapshot)
	}
	if !hasSceneReferenceScope(snapshot.ReferenceDetails, "dynamic") || !hasSceneReferenceScope(snapshot.ReferenceDetails, "unresolved") || !hasDirectedSceneRelationship(snapshot) {
		t.Fatalf("TypeScript viewer lost reference scope or directed relationships: %#v", snapshot)
	}
	if !hasSceneDescription(snapshot, "group src") || !hasSceneDescription(snapshot, "depends_on") {
		t.Fatalf("TypeScript viewer accessibility descriptions lost semantic facts: %#v", snapshot.Accessibility)
	}

	sourceID, sourcePath := firstTypeScriptEvidence(normalized, "src/main.ts")
	if sourceID == "" || !hasEvidenceLocation(normalized, sourceID) {
		t.Fatalf("TypeScript source evidence was not retained: %#v", normalized.SourceReferences)
	}
	sourceQuery := url.Values{"model_id": []string{normalized.ModelID}, "evidence_id": []string{sourceID}, "path": []string{sourcePath}, "start_line": []string{"1"}, "end_line": []string{"20"}}
	sourceResponse, sourceBody := getViewerResponse(t, httpServer.URL+"/v1/source?"+sourceQuery.Encode())
	if sourceResponse.StatusCode != http.StatusOK || !strings.Contains(string(sourceBody), "importedFeature") {
		t.Fatalf("TypeScript source evidence response = %d %s", sourceResponse.StatusCode, sourceBody)
	}
}

func writeVisibleTypeScriptProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "package.json"), `{"name":"visible-ts","type":"module","imports":{"#feature":"./src/feature.ts"},"exports":{".":"./src/main.ts"}}`)
	writeTestFile(t, filepath.Join(root, "tsconfig.json"), `{
  "compilerOptions": {"baseUrl":".","paths":{"@/*":["src/*"]}},
  "include": ["src/**/*"]
}`)
	writeTestFile(t, filepath.Join(root, "src", "main.ts"), `import { feature as importedFeature } from "@/feature";
import type { Contract } from "./types";
export { feature } from "./feature";
const service = require("./service");
const lazy = import("./lazy");
const computed = import("./" + moduleName);
import fs from "node:fs";
import external from "react";
import "./missing";
import "#feature";
export const main: Contract = { importedFeature, service, lazy, computed, fs, external };
`)
	writeTestFile(t, filepath.Join(root, "src", "feature.ts"), `export const feature = true;
`)
	writeTestFile(t, filepath.Join(root, "src", "types.ts"), `export type Contract = { importedFeature: boolean };
`)
	writeTestFile(t, filepath.Join(root, "src", "service.ts"), `export const service = true;
`)
	writeTestFile(t, filepath.Join(root, "src", "lazy.ts"), `export const lazy = true;
`)
	writeTestFile(t, filepath.Join(root, "src", "client.js"), `export const client = true;
`)
	writeTestFile(t, filepath.Join(root, "src", "visible.test.ts"), `import "./feature";
`)
	writeTestFile(t, filepath.Join(root, "src", "excluded.ts"), `export const excluded = true;
`)
	return root
}

func typescriptAnalysisArgs(root, output string) []string {
	return []string{
		"analyze",
		"--project", root,
		"--language", "typescript",
		"--config", "tsconfig.json",
		"--include-js",
		"--include-tests",
		"--runtime", "esm",
		"--exclude", "src/excluded.ts",
		"--format", "analysis-json",
		"--output", output,
	}
}

func typescriptExportAnalyzeArgs(root, output, format string) []string {
	args := typescriptAnalysisArgs(root, output)
	args = append(args, "--view-path", "src")
	for index := range args {
		if args[index] == "analysis-json" {
			args[index] = format
			break
		}
	}
	return args
}

func runTypeScriptCommand(args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func firstTypeScriptEvidence(value model.Model, path string) (string, string) {
	for _, source := range value.SourceReferences {
		if source.Path == path && source.Start != nil {
			return source.ID, source.Path
		}
	}
	return "", ""
}
