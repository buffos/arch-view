package main

import (
	"bytes"
	"encoding/json"
	"io"
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

func TestHelpListsPythonSelectionAndOptions(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("help exit code = %d, stderr=%s", code, stderr.String())
	}
	for _, flag := range []string{"--analyzer <id>", "--source-root <path>", "--python-version <3.x>", "--include-stubs", "--include-tests", "--exclude <glob>"} {
		if !strings.Contains(stdout.String(), flag) {
			t.Fatalf("help output is missing %q: %s", flag, stdout.String())
		}
	}
}

func TestPythonCLIOptionsSelectionAndDeterminism(t *testing.T) {
	root := writeVisiblePythonProject(t)
	firstPath := filepath.Join(t.TempDir(), "first-analysis.json")
	secondPath := filepath.Join(t.TempDir(), "second-analysis.json")
	args := pythonAnalysisArgs(root, firstPath)
	args = append(args, "--analyzer", "org.archview.python")
	if code, _, stderr := runPythonCommand(args...); code != 0 {
		t.Fatalf("explicit Python analyze exit code = %d, stderr=%s", code, stderr)
	}
	for index := range args {
		if args[index] == "--output" && index+1 < len(args) {
			args[index+1] = secondPath
			break
		}
	}
	if code, _, stderr := runPythonCommand(args...); code != 0 {
		t.Fatalf("repeated explicit Python analyze exit code = %d, stderr=%s", code, stderr)
	}

	firstData := readTestFile(t, firstPath)
	secondData := readTestFile(t, secondPath)
	if string(firstData) != string(secondData) {
		t.Fatalf("repeated Python CLI analysis is not byte-stable")
	}
	var result analysis.AnalysisResult
	decodeTestJSON(t, firstData, &result)
	if result.Status != analysis.StatusPartial || result.Analyzer.ID != "org.archview.python" || result.Analyzer.Language != "python" {
		t.Fatalf("Python selection metadata = %#v", result)
	}
	if result.Project.Boundary != "pyproject.toml" || result.OptionsFingerprint == "" || result.RunID == "" {
		t.Fatalf("Python boundary/fingerprint metadata = %#v", result)
	}
	for _, moduleID := range []string{
		"py:package:archdemo",
		"py:module:archdemo.service",
		"py:module:archdemo.contracts",
		"py:module:tests.test_service",
		"py:module:extension",
	} {
		if !hasAnalysisModule(result.Modules, moduleID) {
			t.Fatalf("Python option fixture did not include %q; modules=%#v", moduleID, analysisModuleIDs(result.Modules))
		}
	}
	if hasAnalysisModule(result.Modules, "py:module:configured") || hasAnalysisModule(result.Modules, "py:module:archdemo.ignored") {
		t.Fatalf("CLI source-root/exclude options were not applied: %#v", analysisModuleIDs(result.Modules))
	}
	if !hasAnalysisModuleTag(result.Modules, "py:module:archdemo.contracts", "stub") || !hasAnalysisModuleTag(result.Modules, "py:module:tests.test_service", "test") {
		t.Fatalf("CLI stub/test options were not applied: %#v", result.Modules)
	}
	if !hasPythonMetadataValue(result.Modules, "py:module:archdemo.service", "python_version", "3.12") {
		t.Fatalf("CLI Python version did not override project metadata: %#v", result.Modules)
	}
	if !hasAnalysisDiagnostic(result.Diagnostics, "python_dynamic_import") || !hasAnalysisDiagnostic(result.Diagnostics, "python_unresolved_import") {
		t.Fatalf("partial Python diagnostics are missing: %#v", result.Diagnostics)
	}
	if _, err := os.Stat(filepath.Join(root, "must-not-be-created")); !os.IsNotExist(err) {
		t.Fatalf("static Python analysis executed project code: %v", err)
	}

	autoOutput := filepath.Join(t.TempDir(), "auto-analysis.json")
	if code, _, stderr := runPythonCommand("analyze", "--project", root, "--format", "analysis-json", "--output", autoOutput); code != 0 {
		t.Fatalf("auto-detected Python analyze exit code = %d, stderr=%s", code, stderr)
	}
	var autoResult analysis.AnalysisResult
	decodeTestJSON(t, readTestFile(t, autoOutput), &autoResult)
	if autoResult.Analyzer.ID != "org.archview.python" || autoResult.Project.Boundary != "pyproject.toml" {
		t.Fatalf("auto-detected Python metadata = %#v", autoResult)
	}
	idOutput := filepath.Join(t.TempDir(), "id-analysis.json")
	if code, _, stderr := runPythonCommand("analyze", "--project", root, "--analyzer", "org.archview.python", "--output", idOutput, "--format", "analysis-json"); code != 0 {
		t.Fatalf("explicit analyzer-id Python analyze exit code = %d, stderr=%s", code, stderr)
	}
	var idResult analysis.AnalysisResult
	decodeTestJSON(t, readTestFile(t, idOutput), &idResult)
	if idResult.Analyzer.ID != "org.archview.python" || idResult.Project.Boundary != "pyproject.toml" {
		t.Fatalf("explicit analyzer-id Python metadata = %#v", idResult)
	}
}

func TestPythonVisibleJourneyUsesSharedModelViewerAndExportPaths(t *testing.T) {
	root := writeVisiblePythonProject(t)
	analysisPath := filepath.Join(t.TempDir(), "analysis.json")
	if code, _, stderr := runPythonCommand(pythonAnalysisArgs(root, analysisPath)...); code != 0 {
		t.Fatalf("Python analysis exit code = %d, stderr=%s", code, stderr)
	}

	modelPath := filepath.Join(t.TempDir(), "model.json")
	if code, _, stderr := runPythonCommand("model", "normalize", "--input", analysisPath, "--output", modelPath); code != 0 {
		t.Fatalf("Python model normalize exit code = %d, stderr=%s", code, stderr)
	}
	var normalized model.Model
	decodeTestJSON(t, readTestFile(t, modelPath), &normalized)
	if normalized.Project.Language != "python" || normalized.Status != model.StatusPartial || len(normalized.Relationships) == 0 || len(normalized.References) == 0 {
		t.Fatalf("normalized Python model lost shared observations: %#v", normalized)
	}

	if code, stdout, stderr := runPythonCommand("model", "validate", "--input", modelPath); code != 0 || !strings.Contains(stdout, `"valid": true`) {
		t.Fatalf("Python model validate exit code = %d, stdout=%s, stderr=%s", code, stdout, stderr)
	}
	projectionPath := filepath.Join(t.TempDir(), "projection.json")
	if code, _, stderr := runPythonCommand("model", "projection", "--input", modelPath, "--path", "archdemo", "--output", projectionPath); code != 0 {
		t.Fatalf("Python model projection exit code = %d, stderr=%s", code, stderr)
	}
	var projection model.HierarchyProjection
	decodeTestJSON(t, readTestFile(t, projectionPath), &projection)
	if len(projection.Nodes) == 0 || !hasProjectionNode(projection.Nodes, "py:package:archdemo") {
		t.Fatalf("Python hierarchy projection = %#v", projection)
	}

	htmlPath := filepath.Join(t.TempDir(), "architecture.html")
	if code, _, stderr := runPythonCommand(pythonExportAnalyzeArgs(root, htmlPath, "html")...); code != 0 {
		t.Fatalf("Python HTML export exit code = %d, stderr=%s", code, stderr)
	}
	htmlData := readTestFile(t, htmlPath)
	html := string(htmlData)
	for _, marker := range []string{"window.__ARCH_VIEW_EXPORT__", "archdemo.service", "python_dynamic_import", "download-svg", "Full canvas"} {
		if !strings.Contains(html, marker) {
			t.Fatalf("Python HTML export is missing %q", marker)
		}
	}
	if strings.Contains(html, "<script src=") || strings.Contains(html, `<link rel="stylesheet"`) {
		t.Fatalf("Python HTML export is not self-contained")
	}
	repeatedHTMLPath := filepath.Join(t.TempDir(), "architecture-repeat.html")
	if code, _, stderr := runPythonCommand(pythonExportAnalyzeArgs(root, repeatedHTMLPath, "html")...); code != 0 {
		t.Fatalf("repeated Python HTML export exit code = %d, stderr=%s", code, stderr)
	}
	if string(htmlData) != string(readTestFile(t, repeatedHTMLPath)) {
		t.Fatalf("repeated Python HTML export is not byte-stable")
	}

	jsonPath := filepath.Join(t.TempDir(), "architecture.json")
	if code, _, stderr := runPythonCommand(pythonExportAnalyzeArgs(root, jsonPath, "json")...); code != 0 {
		t.Fatalf("Python canonical JSON export exit code = %d, stderr=%s", code, stderr)
	}
	repeatedJSONPath := filepath.Join(t.TempDir(), "architecture-repeat.json")
	if code, _, stderr := runPythonCommand(pythonExportAnalyzeArgs(root, repeatedJSONPath, "json")...); code != 0 {
		t.Fatalf("repeated Python canonical JSON export exit code = %d, stderr=%s", code, stderr)
	}
	if string(readTestFile(t, jsonPath)) != string(readTestFile(t, repeatedJSONPath)) {
		t.Fatalf("repeated Python canonical JSON export is not byte-stable")
	}

	svgPath := filepath.Join(t.TempDir(), "architecture.svg")
	if code, _, stderr := runPythonCommand(pythonExportAnalyzeArgs(root, svgPath, "svg")...); code != 0 {
		t.Fatalf("Python SVG export exit code = %d, stderr=%s", code, stderr)
	}
	svg := string(readTestFile(t, svgPath))
	for _, marker := range []string{"<svg ", `data-module-id="`, "python_dynamic_import", `scope="dynamic"`} {
		if !strings.Contains(svg, marker) {
			t.Fatalf("Python SVG export is missing %q", marker)
		}
	}
	repeatedSVGPath := filepath.Join(t.TempDir(), "architecture-repeat.svg")
	if code, _, stderr := runPythonCommand(pythonExportAnalyzeArgs(root, repeatedSVGPath, "svg")...); code != 0 {
		t.Fatalf("repeated Python SVG export exit code = %d, stderr=%s", code, stderr)
	}
	if svg != string(readTestFile(t, repeatedSVGPath)) {
		t.Fatalf("repeated Python SVG export is not byte-stable")
	}

	viewerServer, err := viewer.NewServer(normalized, viewer.ServerOptions{SourceRoot: root})
	if err != nil {
		t.Fatalf("NewServer(Python model): %v", err)
	}
	httpServer := httptest.NewServer(viewerServer.Handler())
	defer httpServer.Close()

	rootResponse, rootBody := getViewerResponse(t, httpServer.URL+"/")
	if rootResponse.StatusCode != http.StatusOK || !strings.Contains(string(rootBody), "Download SVG") || !strings.Contains(string(rootBody), "Full canvas") || !strings.Contains(string(rootBody), normalized.ModelID) {
		t.Fatalf("Python viewer root response = %d %s", rootResponse.StatusCode, rootBody)
	}
	exportResponse, exportBody := getViewerResponse(t, httpServer.URL+"/assets/app/export.js")
	if exportResponse.StatusCode != http.StatusOK || !strings.Contains(string(exportBody), "downloadCurrentSVG") || !strings.Contains(string(exportBody), "serializeSVG") {
		t.Fatalf("Python viewer current-canvas export asset = %d %s", exportResponse.StatusCode, exportBody)
	}

	modelResponse, modelBody := getViewerResponse(t, httpServer.URL+"/v1/models/"+url.PathEscape(normalized.ModelID))
	var viewerModel model.Model
	decodeTestJSON(t, modelBody, &viewerModel)
	if modelResponse.StatusCode != http.StatusOK || viewerModel.Project.Language != "python" {
		t.Fatalf("Python viewer model response = %d %#v", modelResponse.StatusCode, viewerModel)
	}

	query := url.Values{"mode": []string{"list"}, "reference_visibility": []string{"expanded"}}
	sceneResponse, sceneBody := getViewerResponse(t, httpServer.URL+"/v1/models/"+url.PathEscape(normalized.ModelID)+"/projection?"+query.Encode())
	var snapshot scene.SceneSnapshot
	decodeTestJSON(t, sceneBody, &snapshot)
	if sceneResponse.StatusCode != http.StatusOK || snapshot.Project.Language != "python" || len(snapshot.VisibleNodes) == 0 || len(snapshot.VisibleRelationships) == 0 || len(snapshot.ReferenceDetails) == 0 || len(snapshot.EvidenceLinks) == 0 {
		t.Fatalf("Python viewer expanded scene = %d %#v", sceneResponse.StatusCode, snapshot)
	}
	if snapshot.ReferenceSummary.ExpandedCount == 0 || !hasSceneReferenceScope(snapshot.ReferenceDetails, "dynamic") || !hasSceneReferenceScope(snapshot.ReferenceDetails, "unresolved") {
		t.Fatalf("Python viewer did not expose reference scopes: %#v", snapshot.ReferenceSummary)
	}
	dynamicDetail := findSceneReference(snapshot.ReferenceDetails, "dynamic")
	if dynamicDetail == nil || dynamicDetail.ConfidenceState != "low" || dynamicDetail.ConfidenceBasis != "dynamic" {
		t.Fatalf("Python viewer did not expose dynamic confidence: %#v", dynamicDetail)
	}
	if !hasDirectedSceneRelationship(snapshot) {
		t.Fatalf("Python viewer did not expose directed relationships: %#v", snapshot.VisibleRelationships)
	}
	if !hasSceneDescription(snapshot, "archdemo") || !hasSceneDescription(snapshot, "depends_on") {
		t.Fatalf("Python viewer accessibility descriptions lost module/relationship facts: %#v", snapshot.Accessibility)
	}

	drillQuery := url.Values{"mode": []string{"list"}, "path": []string{"archdemo"}}
	drillResponse, drillBody := getViewerResponse(t, httpServer.URL+"/v1/models/"+url.PathEscape(normalized.ModelID)+"/projection?"+drillQuery.Encode())
	var drilled scene.SceneSnapshot
	decodeTestJSON(t, drillBody, &drilled)
	if drillResponse.StatusCode != http.StatusOK || len(drilled.HierarchyPath) != 1 || drilled.HierarchyPath[0] != "archdemo" || len(drilled.VisibleNodes) == 0 {
		t.Fatalf("Python viewer drilled scene = %d %#v", drillResponse.StatusCode, drilled)
	}
	if !hasSceneDescription(drilled, "service") || !hasSceneDescription(drilled, "depends_on") {
		t.Fatalf("Python viewer drilled accessibility facts = %#v", drilled.Accessibility)
	}

	sourceID, sourcePath := firstPythonEvidence(normalized, "src/archdemo/service.py")
	if sourceID == "" {
		t.Fatalf("Python service evidence was not retained: %#v", normalized.SourceReferences)
	}
	if !hasEvidenceLocation(normalized, sourceID) {
		t.Fatalf("Python service evidence did not retain a source location: %#v", normalized.SourceReferences)
	}
	sourceQuery := url.Values{"model_id": []string{normalized.ModelID}, "evidence_id": []string{sourceID}, "path": []string{sourcePath}, "start_line": []string{"1"}, "end_line": []string{"10"}}
	sourceResponse, sourceBody := getViewerResponse(t, httpServer.URL+"/v1/source?"+sourceQuery.Encode())
	if sourceResponse.StatusCode != http.StatusOK || !strings.Contains(string(sourceBody), "importlib.import_module") {
		t.Fatalf("Python source evidence response = %d %s", sourceResponse.StatusCode, sourceBody)
	}

	crossRootResponse, _ := getViewerResponse(t, httpServer.URL+"/v1/source?model_id="+url.QueryEscape(normalized.ModelID)+"&path=../outside.py")
	if crossRootResponse.StatusCode != http.StatusForbidden {
		t.Fatalf("Python cross-root evidence status = %d, want %d", crossRootResponse.StatusCode, http.StatusForbidden)
	}
	staleResponse, _ := getViewerResponse(t, httpServer.URL+"/v1/source?model_id=stale-python-model&path="+url.QueryEscape(sourcePath))
	if staleResponse.StatusCode != http.StatusNotFound {
		t.Fatalf("Python stale evidence status = %d, want %d", staleResponse.StatusCode, http.StatusNotFound)
	}
}

func TestPythonCLIRejectsMixedMarkerAutoDetection(t *testing.T) {
	root := writeVisiblePythonProject(t)
	writeTestFile(t, filepath.Join(root, "go.mod"), "module mixed.example\n")
	output := filepath.Join(t.TempDir(), "mixed-analysis.json")
	code, _, stderr := runPythonCommand("analyze", "--project", root, "--output", output, "--format", "analysis-json")
	if code != 2 || !strings.Contains(stderr, "multiple analyzers have the same highest detection confidence") {
		t.Fatalf("mixed-marker auto-detection = code %d, stderr=%s", code, stderr)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("ambiguous selection left an output artifact: %v", err)
	}
}

func writeVisiblePythonProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "pyproject.toml"), "[project]\nname = 'visible-fixture'\nrequires-python = '>=3.10'\n\n[tool.setuptools.packages.find]\nwhere = ['configured_src']\n")
	writeTestFile(t, filepath.Join(root, "configured_src", "configured.py"), "VALUE = 'wrong root'\n")
	writeTestFile(t, filepath.Join(root, "src", "archdemo", "__init__.py"), "from .service import Service\n")
	writeTestFile(t, filepath.Join(root, "src", "archdemo", "service.py"), "import archdemo.models\nfrom . import models\nfrom .missing import Missing\nimport os\nimport requests\nimport importlib\nif TYPE_CHECKING:\n    from .optional import Optional\nloaded = importlib.import_module(module_name)\nclass Service:\n    pass\n")
	writeTestFile(t, filepath.Join(root, "src", "archdemo", "models.py"), "class Model:\n    pass\n")
	writeTestFile(t, filepath.Join(root, "src", "archdemo", "optional.py"), "class Optional:\n    pass\n")
	writeTestFile(t, filepath.Join(root, "src", "archdemo", "contracts.pyi"), "class Contract: ...\n")
	writeTestFile(t, filepath.Join(root, "src", "archdemo", "ignored.py"), "VALUE = 'excluded'\n")
	writeTestFile(t, filepath.Join(root, "src", "archdemo", "side_effect.py"), "open('must-not-be-created', 'w').write('executed')\n")
	writeTestFile(t, filepath.Join(root, "src", "tests", "test_service.py"), "from archdemo.service import Service\n")
	writeTestFile(t, filepath.Join(root, "extensions", "extension.py"), "from archdemo import Service\n")
	return root
}

func pythonAnalysisArgs(root, output string) []string {
	return []string{
		"analyze",
		"--project", root,
		"--language", "python",
		"--source-root", "src",
		"--source-root", "extensions",
		"--python-version", "3.12",
		"--include-stubs",
		"--include-tests",
		"--exclude", "src/archdemo/ignored.py",
		"--format", "analysis-json",
		"--output", output,
	}
}

func pythonExportAnalyzeArgs(root, output, format string) []string {
	args := pythonAnalysisArgs(root, output)
	for index := range args {
		if args[index] == "analysis-json" {
			args[index] = format
			break
		}
	}
	return args
}

func runPythonCommand(args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func readTestFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func decodeTestJSON(t *testing.T, data []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("decode JSON: %v; data=%s", err, data)
	}
}

func hasAnalysisModule(values []analysis.ModuleObservation, id string) bool {
	for _, value := range values {
		if value.ID == id {
			return true
		}
	}
	return false
}

func hasAnalysisModuleTag(values []analysis.ModuleObservation, id, tag string) bool {
	for _, value := range values {
		if value.ID != id {
			continue
		}
		for _, candidate := range value.Tags {
			if candidate == tag {
				return true
			}
		}
	}
	return false
}

func hasPythonMetadataValue(values []analysis.ModuleObservation, id, key, want string) bool {
	for _, value := range values {
		if value.ID == id && value.Metadata[key] == want {
			return true
		}
	}
	return false
}

func hasAnalysisDiagnostic(values []analysis.Diagnostic, code string) bool {
	for _, value := range values {
		if value.Code == code {
			return true
		}
	}
	return false
}

func analysisModuleIDs(values []analysis.ModuleObservation) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.ID)
	}
	return result
}

func hasProjectionNode(values []model.ProjectionNode, id string) bool {
	for _, value := range values {
		if value.ID == id {
			return true
		}
	}
	return false
}

func getViewerResponse(t *testing.T, target string) (*http.Response, []byte) {
	t.Helper()
	response, err := http.Get(target)
	if err != nil {
		t.Fatalf("GET %s: %v", target, err)
	}
	data, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil {
		t.Fatalf("read %s: %v", target, err)
	}
	return response, data
}

func hasSceneReferenceScope(values []scene.ReferenceDetail, scope string) bool {
	for _, value := range values {
		if value.Scope == scope {
			return true
		}
	}
	return false
}

func findSceneReference(values []scene.ReferenceDetail, scope string) *scene.ReferenceDetail {
	for index := range values {
		if values[index].Scope == scope {
			return &values[index]
		}
	}
	return nil
}

func hasDirectedSceneRelationship(snapshot scene.SceneSnapshot) bool {
	for _, relationship := range snapshot.VisibleRelationships {
		if relationship.Directed && relationship.Type == "depends_on" {
			return true
		}
	}
	return false
}

func hasSceneDescription(snapshot scene.SceneSnapshot, fragment string) bool {
	for _, description := range snapshot.Accessibility.Descriptions {
		if strings.Contains(description, fragment) {
			return true
		}
	}
	return false
}

func firstPythonEvidence(value model.Model, path string) (string, string) {
	for _, source := range value.SourceReferences {
		if source.Path == path && source.Start != nil {
			return source.ID, source.Path
		}
	}
	return "", ""
}

func hasEvidenceLocation(value model.Model, id string) bool {
	for _, source := range value.SourceReferences {
		if source.ID == id && source.Start != nil && source.Start.Line > 0 && source.Start.Column > 0 {
			return true
		}
	}
	return false
}
