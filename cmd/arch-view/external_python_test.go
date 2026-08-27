package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/viewer"
)

func TestExternalPythonAnalyzerMatchesBuiltInAndUsesSharedPipeline(t *testing.T) {
	if _, err := exec.LookPath("python"); err != nil {
		t.Skip("python is not available on PATH")
	}
	root := writeExternalPythonFixture(t)
	descriptor := externalPythonDescriptorPath(t)

	builtInPath := filepath.Join(t.TempDir(), "builtin.json")
	externalPaths := []string{
		filepath.Join(t.TempDir(), "external-a.json"),
		filepath.Join(t.TempDir(), "external-b.json"),
	}
	common := []string{
		"--project", root,
		"--source-root", "src",
		"--python-version", "3.12",
		"--format", "analysis-json",
	}
	if code, stderr := runExternalTestCommand(t, append([]string{"analyze"}, append(common, "--analyzer", "org.archview.python", "--output", builtInPath)...)); code != 0 {
		t.Fatalf("built-in Python analysis exit code = %d, stderr=%s", code, stderr)
	}
	for _, output := range externalPaths {
		args := append([]string{"analyze"}, append(common, "--plugin", descriptor, "--analyzer", "org.archview.python.external", "--output", output)...)
		if code, stderr := runExternalTestCommand(t, args); code != 0 {
			t.Fatalf("external Python analysis exit code = %d, stderr=%s", code, stderr)
		}
	}

	builtIn := readExternalAnalysis(t, builtInPath)
	external := readExternalAnalysis(t, externalPaths[0])
	builtIn.RunID = ""
	external.RunID = ""
	builtIn.Analyzer = analysis.AnalyzerInfo{}
	external.Analyzer = analysis.AnalyzerInfo{}
	builtInJSON, err := json.Marshal(builtIn)
	if err != nil {
		t.Fatalf("marshal built-in result: %v", err)
	}
	externalJSON, err := json.Marshal(external)
	if err != nil {
		t.Fatalf("marshal external result: %v", err)
	}
	if !bytes.Equal(builtInJSON, externalJSON) {
		t.Fatalf("external Python result diverges from built-in result:\nbuilt-in=%s\nexternal=%s", builtInJSON, externalJSON)
	}

	firstExternal, err := os.ReadFile(externalPaths[0])
	if err != nil {
		t.Fatalf("read first external result: %v", err)
	}
	secondExternal, err := os.ReadFile(externalPaths[1])
	if err != nil {
		t.Fatalf("read second external result: %v", err)
	}
	if !bytes.Equal(firstExternal, secondExternal) {
		t.Fatal("repeated external Python analysis is not byte-stable")
	}
	if _, err := os.Stat(filepath.Join(root, "must-not-be-created")); !os.IsNotExist(err) {
		t.Fatalf("external analysis executed project code: %v", err)
	}

	modelPath := filepath.Join(t.TempDir(), "model.json")
	if code, stderr := runExternalTestCommand(t, []string{"model", "normalize", "--input", externalPaths[0], "--output", modelPath}); code != 0 {
		t.Fatalf("external model normalization exit code = %d, stderr=%s", code, stderr)
	}
	var normalized model.Model
	data, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("read external model: %v", err)
	}
	if err := json.Unmarshal(data, &normalized); err != nil {
		t.Fatalf("decode external model: %v", err)
	}
	if normalized.Project.Language != "python" || len(normalized.Modules) == 0 {
		t.Fatalf("external model = %#v", normalized)
	}
	if code, stderr := runExternalTestCommand(t, []string{"model", "validate", "--input", modelPath}); code != 0 {
		t.Fatalf("external model validation exit code = %d, stderr=%s", code, stderr)
	}
	projectionPath := filepath.Join(t.TempDir(), "projection.json")
	if code, stderr := runExternalTestCommand(t, []string{"model", "projection", "--input", modelPath, "--output", projectionPath}); code != 0 {
		t.Fatalf("external model projection exit code = %d, stderr=%s", code, stderr)
	}
	if projection, err := os.ReadFile(projectionPath); err != nil || len(projection) == 0 {
		t.Fatalf("external projection = %d bytes, error=%v", len(projection), err)
	}

	modelJSON, err := json.Marshal(normalized)
	if err != nil {
		t.Fatalf("marshal normalized external model: %v", err)
	}
	if bytes.Contains(modelJSON, []byte(`"protocol"`)) || bytes.Contains(modelJSON, []byte(`"manifest"`)) {
		t.Fatalf("external protocol fields leaked into canonical model: %s", modelJSON)
	}
	viewerServer, err := viewer.NewServer(normalized, viewer.ServerOptions{SourceRoot: root})
	if err != nil {
		t.Fatalf("create external viewer: %v", err)
	}
	httpServer := httptest.NewServer(viewerServer.Handler())
	defer httpServer.Close()
	rootResponse, err := http.Get(httpServer.URL + "/")
	if err != nil {
		t.Fatalf("GET external viewer root: %v", err)
	}
	_ = rootResponse.Body.Close()
	if rootResponse.StatusCode != http.StatusOK {
		t.Fatalf("external viewer root status = %d", rootResponse.StatusCode)
	}
	modelURL := httpServer.URL + "/v1/models/" + url.PathEscape(normalized.ModelID)
	modelResponse, err := http.Get(modelURL)
	if err != nil {
		t.Fatalf("GET external viewer model: %v", err)
	}
	_ = modelResponse.Body.Close()
	if modelResponse.StatusCode != http.StatusOK {
		t.Fatalf("external viewer model status = %d", modelResponse.StatusCode)
	}
	projectionResponse, err := http.Get(modelURL + "/projection")
	if err != nil {
		t.Fatalf("GET external viewer scene: %v", err)
	}
	_ = projectionResponse.Body.Close()
	if projectionResponse.StatusCode != http.StatusOK {
		t.Fatalf("external viewer scene status = %d", projectionResponse.StatusCode)
	}
	var source analysis.SourceReference
	for _, candidate := range normalized.SourceReferences {
		if candidate.Start != nil {
			source = candidate
			break
		}
	}
	if source.ID == "" {
		t.Fatal("external model has no source evidence with a location")
	}
	query := url.Values{}
	query.Set("model_id", normalized.ModelID)
	query.Set("evidence_id", source.ID)
	query.Set("path", source.Path)
	sourceResponse, err := http.Get(httpServer.URL + "/v1/source?" + query.Encode())
	if err != nil {
		t.Fatalf("GET external viewer source: %v", err)
	}
	defer sourceResponse.Body.Close()
	if sourceResponse.StatusCode != http.StatusOK {
		t.Fatalf("external viewer source status = %d", sourceResponse.StatusCode)
	}
	var excerpt viewer.SourceExcerpt
	if err := json.NewDecoder(sourceResponse.Body).Decode(&excerpt); err != nil {
		t.Fatalf("decode external viewer source: %v", err)
	}
	if excerpt.SourceReferenceID != source.ID || !excerpt.ReadOnly || excerpt.Path != source.Path {
		t.Fatalf("external viewer source excerpt = %#v", excerpt)
	}
	for _, format := range []string{"json", "html", "svg"} {
		output := filepath.Join(t.TempDir(), "external."+format)
		if code, stderr := runExternalTestCommand(t, []string{"export", "--input", modelPath, "--format", format, "--output", output}); code != 0 {
			t.Fatalf("external %s export exit code = %d, stderr=%s", format, code, stderr)
		}
		if info, err := os.Stat(output); err != nil || info.Size() == 0 {
			t.Fatalf("external %s export = %v, stat error=%v", format, info, err)
		}
	}
}

func TestExternalPluginListingAndSelectionAreDeterministic(t *testing.T) {
	if _, err := exec.LookPath("python"); err != nil {
		t.Skip("python is not available on PATH")
	}
	descriptor := externalPythonDescriptorPath(t)

	stdout, stderr := runExternalTestCommandWithOutput(t, []string{"analyzers", "--plugin", descriptor})
	if stderr != "" {
		t.Fatalf("plugin listing stderr = %s", stderr)
	}
	var response struct {
		Analyzers []analysis.Manifest `json:"analyzers"`
	}
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		t.Fatalf("decode plugin listing: %v; output=%s", err, stdout)
	}
	if len(response.Analyzers) != 6 || response.Analyzers[3].ID != "org.archview.python.external" {
		t.Fatalf("plugin listing = %#v", response.Analyzers)
	}
	for index := 1; index < len(response.Analyzers); index++ {
		if response.Analyzers[index-1].ID >= response.Analyzers[index].ID {
			t.Fatalf("plugin listing is not sorted: %#v", response.Analyzers)
		}
	}

	duplicateStdout, duplicateStderr := runExternalTestCommandWithOutput(t, []string{"analyzers", "--plugin", descriptor, "--plugin", descriptor})
	if duplicateStdout != "" || !strings.Contains(duplicateStderr, "analyzer id is already registered") {
		t.Fatalf("duplicate plugin registration output=%q stderr=%q", duplicateStdout, duplicateStderr)
	}

	root := writeExternalPythonFixture(t)
	output := filepath.Join(t.TempDir(), "selected.json")
	args := []string{
		"analyze", "--project", root,
		"--plugin", descriptor,
		"--analyzer", "org.archview.python.external",
		"--source-root", "src",
		"--output", output,
		"--format", "analysis-json",
	}
	if code, stderr := runExternalTestCommand(t, args); code != 0 {
		t.Fatalf("explicit external selection exit code = %d, stderr=%s", code, stderr)
	}
	result := readExternalAnalysis(t, output)
	if result.Analyzer.ID != "org.archview.python.external" || result.OptionsFingerprint == "" {
		t.Fatalf("selected external result = %#v", result)
	}

	autoOutput := filepath.Join(t.TempDir(), "ambiguous.json")
	autoArgs := []string{"analyze", "--project", root, "--plugin", descriptor, "--output", autoOutput, "--format", "analysis-json"}
	if code, stderr := runExternalTestCommand(t, autoArgs); code != 2 || !strings.Contains(stderr, "multiple analyzers have the same highest detection confidence") {
		t.Fatalf("ambiguous external auto-selection code=%d stderr=%s", code, stderr)
	}
}

func externalPythonDescriptorPath(t *testing.T) string {
	t.Helper()
	path := filepath.Join("..", "..", "plugins", "python-analyzer", "external-plugin.json")
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("resolve external descriptor: %v", err)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("external descriptor %q: %v", abs, err)
	}
	return abs
}

func runExternalTestCommand(t *testing.T, args []string) (int, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	return code, stderr.String()
}

func runExternalTestCommandWithOutput(t *testing.T, args []string) (string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	_ = run(args, &stdout, &stderr)
	return stdout.String(), stderr.String()
}

func readExternalAnalysis(t *testing.T, path string) analysis.AnalysisResult {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read analysis %q: %v", path, err)
	}
	var result analysis.AnalysisResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("decode analysis %q: %v", path, err)
	}
	return result
}

func writeExternalPythonFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"pyproject.toml":           "[project]\nname = 'fixture'\nrequires-python = '>=3.11'\n\n[tool.setuptools.packages.find]\nwhere = ['src']\n",
		"src/app/__init__.py":      "from .sub.service import Service\nfrom . import models\n",
		"src/app/sub/service.py":   "import app.models\nfrom . import models\nfrom ..shared import helper\nimport os\nimport requests\nimport app.missing\nimport importlib\n\nif TYPE_CHECKING:\n    from app import models\n\nloaded = importlib.import_module(module_name)\nother = __import__('app.models')\n",
		"src/app/models.py":        "MODEL = True\n",
		"src/app/shared/helper.py": "HELPER = True\n",
		"src/consumer.py":          "from app import Service\nfrom app import models\nfrom app.sub.service import Service\nfrom app import missing_name\n",
		"src/side_effect.py":       "open('must-not-be-created', 'w').write('executed')\n",
	}
	for relative, content := range files {
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create fixture directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write fixture %q: %v", relative, err)
		}
	}
	return root
}
