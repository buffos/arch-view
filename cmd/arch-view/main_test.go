package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
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

func TestAnalyzeCommandRejectsPositionalArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"analyze", "unexpected"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2; stderr=%s", code, stderr.String())
	}
}
