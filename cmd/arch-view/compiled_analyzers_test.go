package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/processanalyzer"
	"github.com/buffo/arch-view/internal/analysis/processprotocol"
	clojureanalyzer "github.com/buffo/arch-view/internal/analyzers/clojure"
	goanalyzer "github.com/buffo/arch-view/internal/analyzers/go"
	pyanalyzer "github.com/buffo/arch-view/internal/analyzers/python"
	rustanalyzer "github.com/buffo/arch-view/internal/analyzers/rust"
	tsanalyzer "github.com/buffo/arch-view/internal/analyzers/typescript"
)

func TestCompiledAnalyzerEntrypointsMatchInProcessAnalyzers(t *testing.T) {
	repoRoot := compiledAnalyzerRepositoryRoot(t)
	cases := []compiledAnalyzerCase{
		{
			name:    "go",
			new:     func() analysis.Analyzer { return goanalyzer.New() },
			options: map[string]any{"include_tests": true, "build_tags": []string{"fixture"}},
			fixture: writeCompiledGoFixture,
		},
		{
			name:    "python",
			new:     func() analysis.Analyzer { return pyanalyzer.New() },
			options: map[string]any{"source_roots": []string{"src"}, "python_version": "3.12"},
			fixture: writeCompiledPythonFixture,
		},
		{
			name:    "typescript",
			new:     func() analysis.Analyzer { return tsanalyzer.New() },
			options: map[string]any{"runtime": "esm"},
			fixture: writeCompiledTypeScriptFixture,
		},
		{
			name:    "rust",
			new:     func() analysis.Analyzer { return rustanalyzer.New() },
			options: map[string]any{"features": []string{"fixture"}, "target": "x86_64-unknown-linux-gnu"},
			fixture: writeCompiledRustFixture,
		},
		{
			name:    "clojure",
			new:     func() analysis.Analyzer { return clojureanalyzer.New() },
			options: map[string]any{"source_roots": []string{"src"}, "platform": "clj"},
			fixture: writeCompiledClojureFixture,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			root := t.TempDir()
			testCase.fixture(t, root)
			inProcess := testCase.new()
			manifest := inProcess.Manifest()

			binary := buildCompiledAnalyzer(t, repoRoot, testCase.name)
			descriptor := processprotocol.Descriptor{
				SchemaVersion:    processprotocol.DescriptorSchemaVersion,
				Manifest:         manifest,
				Command:          binary,
				WorkingDirectory: repoRoot,
			}
			external, err := processanalyzer.NewWithBaseDirectory(descriptor, repoRoot, processanalyzer.DefaultConfig())
			if err != nil {
				t.Fatalf("create process analyzer: %v", err)
			}

			candidate, err := external.Detect(context.Background(), analysis.DetectRequest{ProjectRoot: root})
			if err != nil {
				t.Fatalf("compiled detect: %v", err)
			}
			if candidate.AnalyzerID != manifest.ID || candidate.Confidence <= 0 {
				t.Fatalf("compiled candidate = %#v", candidate)
			}

			inProcessResult := runCompiledParityHost(t, root, inProcess, testCase.options)
			externalResult := runCompiledParityHost(t, root, external, testCase.options)
			if err := analysis.ValidateAnalysisResult(externalResult, manifest, root); err != nil {
				t.Fatalf("compiled result validation: %v", err)
			}
			if externalResult.OptionsFingerprint != inProcessResult.OptionsFingerprint {
				t.Fatalf("options fingerprint compiled=%q in-process=%q", externalResult.OptionsFingerprint, inProcessResult.OptionsFingerprint)
			}

			inProcessManifestJSON := mustMarshalManifest(t, manifest)
			externalManifestJSON := mustMarshalManifest(t, external.Manifest())
			if !bytes.Equal(inProcessManifestJSON, externalManifestJSON) {
				t.Fatalf("manifest mismatch:\nin-process=%s\ncompiled=%s", inProcessManifestJSON, externalManifestJSON)
			}

			canonicalizeCompiledProvenance(&inProcessResult)
			canonicalizeCompiledProvenance(&externalResult)
			inProcessJSON, err := json.Marshal(inProcessResult)
			if err != nil {
				t.Fatalf("marshal in-process result: %v", err)
			}
			externalJSON, err := json.Marshal(externalResult)
			if err != nil {
				t.Fatalf("marshal compiled result: %v", err)
			}
			if !bytes.Equal(inProcessJSON, externalJSON) {
				t.Fatalf("compiled result diverges from in-process result:\nin-process=%s\ncompiled=%s", inProcessJSON, externalJSON)
			}
		})
	}
}

type compiledAnalyzerCase struct {
	name    string
	new     func() analysis.Analyzer
	options map[string]any
	fixture func(*testing.T, string)
}

func runCompiledParityHost(t *testing.T, root string, analyzer analysis.Analyzer, options map[string]any) analysis.AnalysisResult {
	t.Helper()
	registry := analysis.NewRegistry()
	if err := registry.Register(analyzer); err != nil {
		t.Fatalf("register analyzer %s: %v", analyzer.Manifest().ID, err)
	}
	host := analysis.NewHost(registry)
	result, err := host.Run(context.Background(), analysis.RunRequest{
		ProjectRoot: root,
		AnalyzerID:  analyzer.Manifest().ID,
		CLIOptions:  options,
	})
	if err != nil {
		t.Fatalf("run analyzer %s: %v", analyzer.Manifest().ID, err)
	}
	return result
}

func compiledAnalyzerRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve compiled analyzer test path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func buildCompiledAnalyzer(t *testing.T, repoRoot, name string) string {
	t.Helper()
	binaryName := "arch-view-analyzer-" + name
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binary := filepath.Join(t.TempDir(), binaryName)
	command := exec.Command("go", "build", "-o", binary, "./cmd/analyzers/"+name)
	command.Dir = repoRoot
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("build compiled %s entrypoint: %v\nstderr=%s", name, err, stderr.String())
	}
	return binary
}

func mustMarshalManifest(t *testing.T, manifest analysis.Manifest) []byte {
	t.Helper()
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest %s: %v", manifest.ID, err)
	}
	return data
}

func canonicalizeCompiledProvenance(result *analysis.AnalysisResult) {
	result.RunID = ""
	result.Analyzer = analysis.AnalyzerInfo{}
}

func writeCompiledFixtureFile(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create fixture directory %s: %v", relative, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture file %s: %v", relative, err)
	}
}

func writeCompiledGoFixture(t *testing.T, root string) {
	writeCompiledFixtureFile(t, root, "go.mod", "module example.com/compiled-fixture\n\ngo 1.22\n")
	writeCompiledFixtureFile(t, root, "main.go", "package main\n\nfunc main() {}\n")
	writeCompiledFixtureFile(t, root, "main_test.go", "package main\n\nimport \"testing\"\n\nfunc TestFixture(t *testing.T) {}\n")
}

func writeCompiledPythonFixture(t *testing.T, root string) {
	writeCompiledFixtureFile(t, root, "pyproject.toml", "[project]\nname = 'compiled-fixture'\nrequires-python = '>=3.11'\n\n[tool.setuptools.packages.find]\nwhere = ['src']\n")
	writeCompiledFixtureFile(t, root, "src/app.py", "from . import models\n\nVALUE = models.VALUE\n")
	writeCompiledFixtureFile(t, root, "src/models.py", "VALUE = 1\n")
}

func writeCompiledTypeScriptFixture(t *testing.T, root string) {
	writeCompiledFixtureFile(t, root, "tsconfig.json", "{\"include\":[\"src/**/*\"]}\n")
	writeCompiledFixtureFile(t, root, "src/index.ts", "import { value } from './value';\nexport const result = value;\n")
	writeCompiledFixtureFile(t, root, "src/value.ts", "export const value = 1;\n")
}

func writeCompiledRustFixture(t *testing.T, root string) {
	writeCompiledFixtureFile(t, root, "Cargo.toml", "[package]\nname = \"compiled-fixture\"\nversion = \"0.1.0\"\nedition = \"2021\"\n\n[features]\nfixture = []\n")
	writeCompiledFixtureFile(t, root, "src/lib.rs", "pub fn value() -> i32 { 1 }\n")
}

func writeCompiledClojureFixture(t *testing.T, root string) {
	writeCompiledFixtureFile(t, root, "deps.edn", "{:paths [\"src\"]}\n")
	writeCompiledFixtureFile(t, root, "src/fixture/core.clj", "(ns fixture.core)\n(def value 1)\n")
}
