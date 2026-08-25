package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
)

func TestAnalyzersCommandListsBuiltInGoManifest(t *testing.T) {
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
	if len(response.Analyzers) != 1 || response.Analyzers[0].ID != "org.archview.go" {
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
