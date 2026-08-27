package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
)

func TestAnalyzersCommandListsBuiltInManifests(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"analyzers"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response struct {
		Analyzers []analysis.Manifest `json:"analyzers"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v; output=%s", err, stdout.String())
	}
	if len(response.Analyzers) != 2 || response.Analyzers[0].ID != "org.archview.go" || response.Analyzers[1].ID != "org.archview.python" {
		t.Fatalf("analyzers = %#v", response.Analyzers)
	}
}

func TestAnalyzeCommandSelectsGoBoundaryAndWritesPartialResult(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/service\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	output := filepath.Join(t.TempDir(), "analysis.json")
	var stdout, stderr bytes.Buffer
	code := run([]string{
		"analyze",
		"--project", root,
		"--language", "go",
		"--include-tests",
		"--output", output,
		"--format", "analysis-json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	var result analysis.AnalysisResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("decode result: %v; output=%s", err, string(data))
	}
	if result.Status != analysis.StatusPartial || result.Analyzer.ID != "org.archview.go" {
		t.Fatalf("result = %#v", result)
	}
	if result.Modules == nil || result.Relationships == nil || result.References == nil || result.SourceReferences == nil {
		t.Fatalf("empty observation collections must serialize as arrays: %#v", result)
	}
	if result.OptionsFingerprint == "" || result.Project.ModulePath != "example.com/service" {
		t.Fatalf("result metadata = %#v", result)
	}
}

func TestAnalyzeCommandSelectsPythonAndNormalizesModuleOnlyResult(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte("[tool.setuptools.packages.find]\nwhere = ['src']\n"), 0o644); err != nil {
		t.Fatalf("write pyproject.toml: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "src", "service"), 0o755); err != nil {
		t.Fatalf("create Python package: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "service", "__init__.py"), []byte("NAME = 'service'\n"), 0o644); err != nil {
		t.Fatalf("write __init__.py: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "service", "api.py"), []byte("def handle():\n    return True\n"), 0o644); err != nil {
		t.Fatalf("write api.py: %v", err)
	}
	analysisPath := filepath.Join(t.TempDir(), "python-analysis.json")
	modelPath := filepath.Join(t.TempDir(), "python-model.json")
	var stdout, stderr bytes.Buffer
	code := run([]string{
		"analyze",
		"--project", root,
		"--language", "python",
		"--source-root", "src",
		"--include-stubs=false",
		"--format", "analysis-json",
		"--output", analysisPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Python analyze exit code = %d, stderr = %s", code, stderr.String())
	}
	data, err := os.ReadFile(analysisPath)
	if err != nil {
		t.Fatalf("read Python analysis: %v", err)
	}
	var result analysis.AnalysisResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("decode Python analysis: %v", err)
	}
	if result.Analyzer.ID != "org.archview.python" || result.Status != analysis.StatusComplete || len(result.Modules) != 2 {
		t.Fatalf("Python analysis result = %#v", result)
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"model", "normalize", "--input", analysisPath, "--output", modelPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("Python model normalize exit code = %d, stderr = %s", code, stderr.String())
	}
	modelData, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("read Python model: %v", err)
	}
	var normalized model.Model
	if err := json.Unmarshal(modelData, &normalized); err != nil {
		t.Fatalf("decode Python model: %v", err)
	}
	if normalized.Project.Language != "python" || normalized.SchemaVersion != model.SchemaVersion || len(normalized.Modules) != 2 {
		t.Fatalf("normalized Python model = %#v", normalized)
	}
}

func TestAnalyzeCommandAutoDetectsPythonProject(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "setup.cfg"), []byte("[options.packages.find]\nwhere = src\n"), 0o644); err != nil {
		t.Fatalf("write setup.cfg: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "src", "auto"), 0o755); err != nil {
		t.Fatalf("create Python package: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "auto", "module.py"), []byte("VALUE = 1\n"), 0o644); err != nil {
		t.Fatalf("write module.py: %v", err)
	}
	output := filepath.Join(t.TempDir(), "auto-python-analysis.json")
	var stdout, stderr bytes.Buffer
	if code := run([]string{"analyze", "--project", root, "--output", output, "--format", "analysis-json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("auto-detect exit code = %d, stderr = %s", code, stderr.String())
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read auto-detected analysis: %v", err)
	}
	var result analysis.AnalysisResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("decode auto-detected analysis: %v", err)
	}
	if result.Analyzer.ID != "org.archview.python" || result.Project.Boundary != "setup.cfg" {
		t.Fatalf("auto-detected result = %#v", result)
	}
}

func TestAnalyzeCommandReturnsUnsupportedExitCode(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := run([]string{
		"analyze",
		"--project", root,
		"--language", "go",
		"--output", "-",
		"--format", "analysis-json",
	}, &stdout, &stderr)
	if code != 3 {
		t.Fatalf("exit code = %d, want 3; stderr=%s", code, stderr.String())
	}
}

func TestOpenCommandRequiresExactlyOneInput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"open"}, &stdout, &stderr); code != 2 {
		t.Fatalf("open without input exit code = %d, want 2; stderr=%s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"open", "--model", "model.json", "--project", "."}, &stdout, &stderr); code != 2 {
		t.Fatalf("open with two inputs exit code = %d, want 2; stderr=%s", code, stderr.String())
	}
}

func TestModelNormalizeAndValidateCommands(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/service\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package service\n\nimport \"example.com/service/internal/worker\"\n\nvar _ = worker.Name\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	workerDir := filepath.Join(root, "internal", "worker")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("create worker directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workerDir, "worker.go"), []byte("package worker\n\nconst Name = \"worker\"\n"), 0o644); err != nil {
		t.Fatalf("write worker.go: %v", err)
	}
	analysisPath := filepath.Join(t.TempDir(), "analysis.json")
	modelPath := filepath.Join(t.TempDir(), "model.json")
	var stdout, stderr bytes.Buffer
	if code := run([]string{"analyze", "--project", root, "--language", "go", "--output", analysisPath, "--format", "analysis-json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("analyze exit code = %d, stderr=%s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"model", "normalize", "--input", analysisPath, "--output", modelPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("normalize exit code = %d, stderr=%s", code, stderr.String())
	}
	data, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("read model: %v", err)
	}
	var normalized model.Model
	if err := json.Unmarshal(data, &normalized); err != nil {
		t.Fatalf("decode model: %v", err)
	}
	if normalized.SchemaVersion != model.SchemaVersion || normalized.ModelID == "" || len(normalized.Modules) != 2 || len(normalized.Relationships) != 1 {
		t.Fatalf("normalized model = %#v", normalized)
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"model", "validate", "--input", modelPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("validate exit code = %d, stderr=%s", code, stderr.String())
	}
	var validation struct {
		Valid bool `json:"valid"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &validation); err != nil || !validation.Valid {
		t.Fatalf("validation response = %s; error=%v", stdout.String(), err)
	}
	stdout.Reset()
	stderr.Reset()
	projectionPath := filepath.Join(t.TempDir(), "projection.json")
	if code := run([]string{"model", "projection", "--input", modelPath, "--output", projectionPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("projection exit code = %d, stderr=%s", code, stderr.String())
	}
	projectionData, err := os.ReadFile(projectionPath)
	if err != nil {
		t.Fatalf("read projection: %v", err)
	}
	var projection model.HierarchyProjection
	if err := json.Unmarshal(projectionData, &projection); err != nil {
		t.Fatalf("decode projection: %v", err)
	}
	if len(projection.Nodes) != 2 {
		t.Fatalf("projection = %#v", projection)
	}
	stdout.Reset()
	stderr.Reset()
	pathProjection := filepath.Join(t.TempDir(), "worker-projection.json")
	if code := run([]string{"model", "projection", "--input", modelPath, "--path", "internal", "--output", pathProjection}, &stdout, &stderr); code != 0 {
		t.Fatalf("path projection exit code = %d, stderr=%s", code, stderr.String())
	}
	pathProjectionData, err := os.ReadFile(pathProjection)
	if err != nil {
		t.Fatalf("read path projection: %v", err)
	}
	var selectedProjection model.HierarchyProjection
	if err := json.Unmarshal(pathProjectionData, &selectedProjection); err != nil {
		t.Fatalf("decode path projection: %v", err)
	}
	if len(selectedProjection.Nodes) != 1 || selectedProjection.Nodes[0].Kind != "group" || selectedProjection.Nodes[0].ID != "group:internal/worker" {
		t.Fatalf("selected projection = %#v", selectedProjection)
	}
}

func TestAnalyzeCommandRejectsPositionalArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"analyze", "unexpected"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2; stderr=%s", code, stderr.String())
	}
}

func TestAnalyzeCommandExportsSelfContainedHTML(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/export-cli\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"ok\") }\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".archview.json"), []byte(`{"schema_version":"arch-view.config/v1","layout":{"algorithm":"layered","options":{"org.eclipse.elk.edgeRouting":"SPLINES"}}}`), 0o644); err != nil {
		t.Fatalf("write layout config: %v", err)
	}
	output := filepath.Join(t.TempDir(), "architecture.html")
	var stdout, stderr bytes.Buffer
	code := run([]string{
		"analyze",
		"--project", root,
		"--language", "go",
		"--format", "html",
		"--output", output,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("HTML analyze/export exit code = %d, stderr=%s", code, stderr.String())
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read HTML export: %v", err)
	}
	html := string(data)
	if !strings.Contains(html, "window.__ARCH_VIEW_EXPORT__") || !strings.Contains(html, `"org.eclipse.elk.edgeRouting":"SPLINES"`) || strings.Contains(html, "<script src=") || strings.Contains(html, "<link rel=\"stylesheet\"") {
		t.Fatalf("HTML export is not self-contained: %s", html[:minTestStringLength(len(html), 500)])
	}
	var metadata struct {
		Format string `json:"format"`
		Status string `json:"status"`
		Bytes  int    `json:"bytes"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &metadata); err != nil {
		t.Fatalf("decode export metadata: %v; output=%s", err, stdout.String())
	}
	if metadata.Format != "html" || metadata.Status == "" || metadata.Bytes != len(data) {
		t.Fatalf("export metadata = %#v, file bytes=%d", metadata, len(data))
	}
}

func TestExportCommandWritesAllFormatsAndRejectsSourceEmbedding(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/export-command\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"ok\") }\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	analysisPath := filepath.Join(t.TempDir(), "analysis.json")
	modelPath := filepath.Join(t.TempDir(), "model.json")
	var stdout, stderr bytes.Buffer
	if code := run([]string{"analyze", "--project", root, "--language", "go", "--format", "analysis-json", "--output", analysisPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("analyze exit code = %d, stderr=%s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"model", "normalize", "--input", analysisPath, "--output", modelPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("normalize exit code = %d, stderr=%s", code, stderr.String())
	}
	for _, format := range []string{"json", "html", "svg"} {
		stdout.Reset()
		stderr.Reset()
		output := filepath.Join(t.TempDir(), "architecture."+format)
		if code := run([]string{"export", "--input", modelPath, "--format", format, "--output", output}, &stdout, &stderr); code != 0 {
			t.Fatalf("%s export exit code = %d, stderr=%s", format, code, stderr.String())
		}
		if info, err := os.Stat(output); err != nil || info.Size() == 0 {
			t.Fatalf("%s export output = %v, stat error=%v", format, info, err)
		}
	}
	unsupportedOutput := filepath.Join(t.TempDir(), "unsupported.json")
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"export", "--input", modelPath, "--format", "json", "--output", unsupportedOutput, "--embed-source"}, &stdout, &stderr); code != 2 {
		t.Fatalf("embed-source exit code = %d, want 2; stderr=%s", code, stderr.String())
	}
	if _, err := os.Stat(unsupportedOutput); !os.IsNotExist(err) {
		t.Fatalf("unsupported export left an output file: %v", err)
	}
}

func TestExportCommandRejectsNonDeterministicOutput(t *testing.T) {
	output := filepath.Join(t.TempDir(), "architecture.json")
	var stdout, stderr bytes.Buffer
	code := run([]string{
		"export",
		"--input", filepath.Join(t.TempDir(), "missing-model.json"),
		"--format", "json",
		"--output", output,
		"--deterministic=false",
	}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("non-deterministic export exit code = %d, want 2; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "non-deterministic export is unsupported") {
		t.Fatalf("non-deterministic export error = %s", stderr.String())
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("rejected export left an output file: %v", err)
	}
}

func minTestStringLength(value, maximum int) int {
	if value < maximum {
		return value
	}
	return maximum
}
