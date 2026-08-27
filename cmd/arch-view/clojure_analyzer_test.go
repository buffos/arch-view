package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
)

func TestClojureCLIExplicitAndAutomaticAnalysisJSON(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "deps.edn"), []byte("{:paths [\"src\"]}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "src", "sample"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "sample", "core.clj"), []byte("(ns sample.core)\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, selection := range [][]string{{"--language", "clojure"}, {}} {
		output := filepath.Join(t.TempDir(), "analysis.json")
		args := []string{"analyze", "--project", root, "--output", output, "--format", "analysis-json"}
		args = append(args, selection...)
		var stdout, stderr bytes.Buffer
		if code := run(args, &stdout, &stderr); code != 0 {
			t.Fatalf("selection %v exit code = %d, stderr = %s", selection, code, stderr.String())
		}
		data, err := os.ReadFile(output)
		if err != nil {
			t.Fatal(err)
		}
		var result analysis.AnalysisResult
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("selection %v result: %v", selection, err)
		}
		if result.Status != analysis.StatusComplete || result.Analyzer.ID != "org.archview.clojure" || result.Project.Boundary != "deps.edn" || !hasClojureModule(result.Modules, "clj:sample.core") {
			t.Fatalf("selection %v result = %#v", selection, result)
		}
	}
}

func hasClojureModule(modules []analysis.ModuleObservation, id string) bool {
	for _, module := range modules {
		if module.ID == id {
			return true
		}
	}
	return false
}
