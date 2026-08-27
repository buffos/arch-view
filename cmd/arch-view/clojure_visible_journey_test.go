package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/viewer"
	"github.com/buffo/arch-view/internal/viewer/scene"
)

func TestHelpListsClojurePlatformOption(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("help exit code = %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "--platform <clj|cljs|both>") {
		t.Fatalf("help output is missing Clojure platform option: %s", stdout.String())
	}
}

func TestClojureCLIOptionsAndSharedVisibleJourney(t *testing.T) {
	root := writeVisibleClojureProject(t)
	firstPath := filepath.Join(t.TempDir(), "first-analysis.json")
	secondPath := filepath.Join(t.TempDir(), "second-analysis.json")
	args := clojureAnalysisArgs(root, firstPath)
	if code, _, stderr := runClojureCommand(args...); code != 0 {
		t.Fatalf("Clojure analyze exit code = %d, stderr=%s", code, stderr)
	}
	for index := range args {
		if args[index] == "--output" && index+1 < len(args) {
			args[index+1] = secondPath
			break
		}
	}
	if code, _, stderr := runClojureCommand(args...); code != 0 {
		t.Fatalf("repeated Clojure analyze exit code = %d, stderr=%s", code, stderr)
	}
	firstData := readTestFile(t, firstPath)
	secondData := readTestFile(t, secondPath)
	if string(firstData) != string(secondData) {
		t.Fatal("repeated Clojure analysis is not byte-stable")
	}
	var result analysis.AnalysisResult
	decodeTestJSON(t, firstData, &result)
	if result.Analyzer.ID != "org.archview.clojure" || result.Analyzer.Language != "clojure" || result.Project.Boundary != "deps.edn" || result.RunID == "" || result.OptionsFingerprint == "" {
		t.Fatalf("Clojure selection metadata = %#v", result)
	}
	if result.Status != analysis.StatusPartial || !hasAnalysisModule(result.Modules, "clj:app.core") || !hasAnalysisModule(result.Modules, "clj:app.server") || !hasAnalysisModule(result.Modules, "clj:app.browser") {
		t.Fatalf("Clojure analysis observations = %#v", result)
	}
	if !hasAnalysisDiagnostic(result.Diagnostics, "clojure_unresolved_dependency") || !hasAnalysisDiagnostic(result.Diagnostics, "clojure_dynamic_reference") {
		t.Fatalf("Clojure diagnostics = %#v", result.Diagnostics)
	}
	if hasAnalysisModule(result.Modules, "clj:app.ignored") || !hasAnalysisModuleTag(result.Modules, "clj:app.core", "polymorphic") {
		t.Fatalf("Clojure source-root/exclusion/polymorphic options were not applied: %#v", result.Modules)
	}
	if !hasClojureAnalysisMetadata(result.Modules, "clj:app.core", "platform", "both") {
		t.Fatalf("Clojure platform metadata missing: %#v", result.Modules)
	}
	if _, err := os.Stat(filepath.Join(root, "must-not-be-created")); !os.IsNotExist(err) {
		t.Fatalf("static Clojure analysis executed project code: %v", err)
	}

	autoOutput := filepath.Join(t.TempDir(), "auto-analysis.json")
	if code, _, stderr := runClojureCommand("analyze", "--project", root, "--output", autoOutput, "--format", "analysis-json"); code != 0 {
		t.Fatalf("auto-detected Clojure analyze exit code = %d, stderr=%s", code, stderr)
	}
	var autoResult analysis.AnalysisResult
	decodeTestJSON(t, readTestFile(t, autoOutput), &autoResult)
	if autoResult.Analyzer.ID != "org.archview.clojure" || autoResult.Project.Boundary != "deps.edn" {
		t.Fatalf("auto-detected Clojure metadata = %#v", autoResult)
	}
	idOutput := filepath.Join(t.TempDir(), "id-analysis.json")
	if code, _, stderr := runClojureCommand("analyze", "--project", root, "--analyzer", "org.archview.clojure", "--output", idOutput, "--format", "analysis-json"); code != 0 {
		t.Fatalf("explicit analyzer-id Clojure analyze exit code = %d, stderr=%s", code, stderr)
	}
	var idResult analysis.AnalysisResult
	decodeTestJSON(t, readTestFile(t, idOutput), &idResult)
	if idResult.Analyzer.ID != "org.archview.clojure" || idResult.Project.Boundary != "deps.edn" {
		t.Fatalf("explicit analyzer-id Clojure metadata = %#v", idResult)
	}
}

func TestClojureSharedModelViewerAndDeterministicExports(t *testing.T) {
	root := writeVisibleClojureProject(t)
	analysisPath := filepath.Join(t.TempDir(), "analysis.json")
	if code, _, stderr := runClojureCommand(clojureAnalysisArgs(root, analysisPath)...); code != 0 {
		t.Fatalf("Clojure analysis exit code = %d, stderr=%s", code, stderr)
	}

	modelPath := filepath.Join(t.TempDir(), "model.json")
	if code, _, stderr := runClojureCommand("model", "normalize", "--input", analysisPath, "--output", modelPath); code != 0 {
		t.Fatalf("Clojure model normalize exit code = %d, stderr=%s", code, stderr)
	}
	var normalized model.Model
	decodeTestJSON(t, readTestFile(t, modelPath), &normalized)
	if normalized.Project.Language != "clojure" || normalized.Status != model.StatusPartial || len(normalized.Relationships) == 0 || len(normalized.References) == 0 {
		t.Fatalf("normalized Clojure model lost shared observations: %#v", normalized)
	}
	if code, stdout, stderr := runClojureCommand("model", "validate", "--input", modelPath); code != 0 || !strings.Contains(stdout, `"valid": true`) {
		t.Fatalf("Clojure model validate exit code = %d, stdout=%s, stderr=%s", code, stdout, stderr)
	}
	projectionPath := filepath.Join(t.TempDir(), "projection.json")
	if code, _, stderr := runClojureCommand("model", "projection", "--input", modelPath, "--path", "app", "--output", projectionPath); code != 0 {
		t.Fatalf("Clojure model projection exit code = %d, stderr=%s", code, stderr)
	}
	var projection model.HierarchyProjection
	decodeTestJSON(t, readTestFile(t, projectionPath), &projection)
	if len(projection.Nodes) == 0 || !hasProjectionModule(projection.Nodes, "clj:app.core") {
		t.Fatalf("Clojure hierarchy projection = %#v", projection)
	}

	htmlPath := filepath.Join(t.TempDir(), "architecture.html")
	if code, _, stderr := runClojureCommand(clojureExportAnalyzeArgs(root, htmlPath, "html")...); code != 0 {
		t.Fatalf("Clojure HTML export exit code = %d, stderr=%s", code, stderr)
	}
	htmlData := readTestFile(t, htmlPath)
	html := string(htmlData)
	for _, marker := range []string{"window.__ARCH_VIEW_EXPORT__", "app.core", "clojure_dynamic_reference", "download-svg", "Full canvas"} {
		if !strings.Contains(html, marker) {
			t.Fatalf("Clojure HTML export is missing %q", marker)
		}
	}
	if strings.Contains(html, "<script src=") || strings.Contains(html, `<link rel="stylesheet"`) {
		t.Fatal("Clojure HTML export is not self-contained")
	}
	if strings.Contains(html, "must-not-be-created") {
		t.Fatal("Clojure HTML export embedded dynamic/source content")
	}
	repeatedHTMLPath := filepath.Join(t.TempDir(), "architecture-repeat.html")
	if code, _, stderr := runClojureCommand(clojureExportAnalyzeArgs(root, repeatedHTMLPath, "html")...); code != 0 {
		t.Fatalf("repeated Clojure HTML export exit code = %d, stderr=%s", code, stderr)
	}
	if string(htmlData) != string(readTestFile(t, repeatedHTMLPath)) {
		t.Fatal("repeated Clojure HTML export is not byte-stable")
	}

	jsonPath := filepath.Join(t.TempDir(), "architecture.json")
	if code, _, stderr := runClojureCommand(clojureExportAnalyzeArgs(root, jsonPath, "json")...); code != 0 {
		t.Fatalf("Clojure canonical JSON export exit code = %d, stderr=%s", code, stderr)
	}
	repeatedJSONPath := filepath.Join(t.TempDir(), "architecture-repeat.json")
	if code, _, stderr := runClojureCommand(clojureExportAnalyzeArgs(root, repeatedJSONPath, "json")...); code != 0 {
		t.Fatalf("repeated Clojure canonical JSON export exit code = %d, stderr=%s", code, stderr)
	}
	if string(readTestFile(t, jsonPath)) != string(readTestFile(t, repeatedJSONPath)) {
		t.Fatal("repeated Clojure canonical JSON export is not byte-stable")
	}

	svgPath := filepath.Join(t.TempDir(), "architecture.svg")
	if code, _, stderr := runClojureCommand(clojureExportAnalyzeArgs(root, svgPath, "svg")...); code != 0 {
		t.Fatalf("Clojure SVG export exit code = %d, stderr=%s", code, stderr)
	}
	svg := string(readTestFile(t, svgPath))
	for _, marker := range []string{"<svg ", `data-module-id=`, "clojure_dynamic_reference", `scope="dynamic"`} {
		if !strings.Contains(svg, marker) {
			t.Fatalf("Clojure SVG export is missing %q", marker)
		}
	}
	repeatedSVGPath := filepath.Join(t.TempDir(), "architecture-repeat.svg")
	if code, _, stderr := runClojureCommand(clojureExportAnalyzeArgs(root, repeatedSVGPath, "svg")...); code != 0 {
		t.Fatalf("repeated Clojure SVG export exit code = %d, stderr=%s", code, stderr)
	}
	if svg != string(readTestFile(t, repeatedSVGPath)) {
		t.Fatal("repeated Clojure SVG export is not byte-stable")
	}

	viewerServer, err := viewer.NewServer(normalized, viewer.ServerOptions{SourceRoot: root})
	if err != nil {
		t.Fatalf("NewServer(Clojure model): %v", err)
	}
	httpServer := httptest.NewServer(viewerServer.Handler())
	defer httpServer.Close()
	rootResponse, rootBody := getViewerResponse(t, httpServer.URL+"/")
	if rootResponse.StatusCode != http.StatusOK || !strings.Contains(string(rootBody), "Download SVG") || !strings.Contains(string(rootBody), "Full canvas") || !strings.Contains(string(rootBody), normalized.ModelID) {
		t.Fatalf("Clojure viewer root response = %d %s", rootResponse.StatusCode, rootBody)
	}
	modelResponse, modelBody := getViewerResponse(t, httpServer.URL+"/v1/models/"+url.PathEscape(normalized.ModelID))
	var viewerModel model.Model
	decodeTestJSON(t, modelBody, &viewerModel)
	if modelResponse.StatusCode != http.StatusOK || viewerModel.Project.Language != "clojure" {
		t.Fatalf("Clojure viewer model response = %d %#v", modelResponse.StatusCode, viewerModel)
	}
	query := url.Values{"mode": []string{"list"}, "reference_visibility": []string{"expanded"}}
	sceneResponse, sceneBody := getViewerResponse(t, httpServer.URL+"/v1/models/"+url.PathEscape(normalized.ModelID)+"/projection?"+query.Encode())
	var snapshot scene.SceneSnapshot
	decodeTestJSON(t, sceneBody, &snapshot)
	if sceneResponse.StatusCode != http.StatusOK || snapshot.Project.Language != "clojure" || len(snapshot.VisibleNodes) == 0 || len(snapshot.VisibleRelationships) == 0 || len(snapshot.ReferenceDetails) == 0 || len(snapshot.EvidenceLinks) == 0 {
		t.Fatalf("Clojure viewer expanded scene = %d %#v", sceneResponse.StatusCode, snapshot)
	}
	if snapshot.ReferenceSummary.ExpandedCount == 0 || !hasSceneReferenceScope(snapshot.ReferenceDetails, "dynamic") || !hasSceneReferenceScope(snapshot.ReferenceDetails, "unresolved") || !hasDirectedSceneRelationship(snapshot) {
		t.Fatalf("Clojure viewer did not expose shared reference/edge facts: %#v", snapshot)
	}
	if !hasSceneDescription(snapshot, "app") || !hasSceneDescription(snapshot, "depends_on") {
		t.Fatalf("Clojure viewer accessibility facts = %#v", snapshot.Accessibility)
	}

	drillQuery := url.Values{"mode": []string{"list"}, "path": []string{"app"}}
	drillResponse, drillBody := getViewerResponse(t, httpServer.URL+"/v1/models/"+url.PathEscape(normalized.ModelID)+"/projection?"+drillQuery.Encode())
	var drilled scene.SceneSnapshot
	decodeTestJSON(t, drillBody, &drilled)
	if drillResponse.StatusCode != http.StatusOK || len(drilled.HierarchyPath) != 1 || drilled.HierarchyPath[0] != "app" || len(drilled.VisibleNodes) == 0 {
		t.Fatalf("Clojure viewer drilled scene = %d %#v", drillResponse.StatusCode, drilled)
	}

	sourceID, sourcePath := firstClojureEvidence(normalized, "src/app/core.cljc")
	if sourceID == "" || !hasEvidenceLocation(normalized, sourceID) {
		t.Fatalf("Clojure namespace evidence was not retained: %#v", normalized.SourceReferences)
	}
	sourceQuery := url.Values{"model_id": []string{normalized.ModelID}, "evidence_id": []string{sourceID}, "path": []string{sourcePath}, "start_line": []string{"1"}, "end_line": []string{"20"}}
	sourceResponse, sourceBody := getViewerResponse(t, httpServer.URL+"/v1/source?"+sourceQuery.Encode())
	if sourceResponse.StatusCode != http.StatusOK || !strings.Contains(string(sourceBody), "defprotocol") {
		t.Fatalf("Clojure source evidence response = %d %s", sourceResponse.StatusCode, sourceBody)
	}
	crossRootResponse, _ := getViewerResponse(t, httpServer.URL+"/v1/source?model_id="+url.QueryEscape(normalized.ModelID)+"&path=../outside.clj")
	if crossRootResponse.StatusCode != http.StatusForbidden {
		t.Fatalf("Clojure cross-root evidence status = %d, want %d", crossRootResponse.StatusCode, http.StatusForbidden)
	}
}

func TestClojureCLIRejectsMixedMarkerAutoDetection(t *testing.T) {
	root := writeVisibleClojureProject(t)
	writeTestFile(t, filepath.Join(root, "go.mod"), "module mixed.example\n")
	output := filepath.Join(t.TempDir(), "mixed-analysis.json")
	code, _, stderr := runClojureCommand("analyze", "--project", root, "--output", output, "--format", "analysis-json")
	if code != 2 || !strings.Contains(stderr, "multiple analyzers have the same highest detection confidence") {
		t.Fatalf("mixed-marker auto-detection = code %d, stderr=%s", code, stderr)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("ambiguous selection left an output artifact: %v", err)
	}
}

func writeVisibleClojureProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "deps.edn"), "{:paths [\"src\" \"test\"]}\n")
	writeTestFile(t, filepath.Join(root, "src", "app", "core.cljc"), `(ns app.core
  #?(:clj (:require [app.server :as server])
     :cljs (:require [app.browser :as browser]))
  (:require [app.missing :as missing]))
(defprotocol Service (run [this value]))
(defmulti render (fn [value] (:kind value)))
(load-string "(spit \\\"must-not-be-created\\\" \\\"executed\\\")")
`)
	writeTestFile(t, filepath.Join(root, "src", "app", "server.clj"), "(ns app.server)\n")
	writeTestFile(t, filepath.Join(root, "src", "app", "browser.cljs"), "(ns app.browser)\n")
	writeTestFile(t, filepath.Join(root, "src", "app", "ignored.clj"), "(ns app.ignored)\n")
	writeTestFile(t, filepath.Join(root, "test", "app", "core_test.clj"), "(ns app.core-test)\n")
	return root
}

func clojureAnalysisArgs(root, output string) []string {
	return []string{
		"analyze",
		"--project", root,
		"--language", "clojure",
		"--source-root", "src",
		"--source-root", "test",
		"--platform", "both",
		"--include-tests",
		"--exclude", "src/app/ignored.clj",
		"--format", "analysis-json",
		"--output", output,
	}
}

func clojureExportAnalyzeArgs(root, output, format string) []string {
	args := clojureAnalysisArgs(root, output)
	for index := range args {
		if args[index] == "analysis-json" {
			args[index] = format
			break
		}
	}
	return args
}

func runClojureCommand(args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func hasClojureAnalysisMetadata(values []analysis.ModuleObservation, id, key, expected string) bool {
	for _, value := range values {
		if value.ID == id && value.Metadata[key] == expected {
			return true
		}
	}
	return false
}

func firstClojureEvidence(value model.Model, path string) (string, string) {
	for _, source := range value.SourceReferences {
		if source.Path == path && source.Start != nil {
			return source.ID, source.Path
		}
	}
	return "", ""
}

func hasProjectionModule(values []model.ProjectionNode, moduleID string) bool {
	for _, value := range values {
		for _, candidate := range value.ModuleIDs {
			if candidate == moduleID {
				return true
			}
		}
	}
	return false
}
