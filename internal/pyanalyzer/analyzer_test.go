package pyanalyzer

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
)

func TestManifestMatchesPythonContract(t *testing.T) {
	manifest := New().Manifest()
	if err := analysis.ValidateManifest(manifest); err != nil {
		t.Fatalf("validate manifest: %v", err)
	}
	if manifest.ID != "org.archview.python" || manifest.Language != "python" || manifest.APIVersion != analysis.AnalyzerAPIVersion {
		t.Fatalf("manifest identity = %#v", manifest)
	}
	if len(manifest.DetectionMarkers) != 3 || manifest.DetectionMarkers[0].Value != "pyproject.toml" || manifest.DetectionMarkers[1].Value != "setup.cfg" || manifest.DetectionMarkers[2].Value != "setup.py" {
		t.Fatalf("manifest markers = %#v", manifest.DetectionMarkers)
	}
	if strings.Join(manifest.Capabilities, ",") != "detect,static_dependencies,dynamic_diagnostics" {
		t.Fatalf("manifest capabilities = %#v", manifest.Capabilities)
	}
	expectedOptions := map[string]analysis.OptionDescriptor{
		"source_roots":   {Name: "source_roots", Type: "string[]", Default: []string{}},
		"python_version": {Name: "python_version", Type: "string", Default: nil},
		"include_stubs":  {Name: "include_stubs", Type: "boolean", Default: false},
		"include_tests":  {Name: "include_tests", Type: "boolean", Default: false},
		"exclude":        {Name: "exclude", Type: "string[]", Default: []string{}},
	}
	if len(manifest.Options) != len(expectedOptions) {
		t.Fatalf("manifest option count = %d, want %d", len(manifest.Options), len(expectedOptions))
	}
	for _, option := range manifest.Options {
		expected, ok := expectedOptions[option.Name]
		if !ok || option.Type != expected.Type || !reflect.DeepEqual(option.Default, expected.Default) {
			t.Fatalf("manifest option %q = %#v", option.Name, option)
		}
	}
}

func TestDetectUsesPyprojectPrecedenceWithoutExecutingSetup(t *testing.T) {
	root := t.TempDir()
	writePythonFixture(t, filepath.Join(root, "pyproject.toml"), "[project]\nname = 'fixture'\n")
	writePythonFixture(t, filepath.Join(root, "setup.cfg"), "[metadata]\nname = fixture\n")
	writePythonFixture(t, filepath.Join(root, "setup.py"), "open('must-not-be-created', 'w').write('executed')\n")
	candidate, err := New().Detect(context.Background(), analysis.DetectRequest{ProjectRoot: root})
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if candidate.BoundaryHint != "pyproject.toml" || candidate.Confidence != 1 {
		t.Fatalf("candidate = %#v", candidate)
	}
	if strings.Join(candidate.MatchedMarkers, ",") != "pyproject.toml,setup.cfg,setup.py" {
		t.Fatalf("matched markers = %#v", candidate.MatchedMarkers)
	}
	if _, err := os.Stat(filepath.Join(root, "must-not-be-created")); !os.IsNotExist(err) {
		t.Fatalf("setup.py appears to have executed: %v", err)
	}
}

func TestAnalyzeDiscoversRegularAndNamespacePackagesDeterministically(t *testing.T) {
	root := t.TempDir()
	writePythonFixture(t, filepath.Join(root, "pyproject.toml"), "[project]\nrequires-python = '>=3.11'\n\n[tool.setuptools.packages.find]\nwhere = ['src',]\n")
	writePythonFixture(t, filepath.Join(root, "src", "acme", "__init__.py"), "VALUE = 1\n")
	writePythonFixture(t, filepath.Join(root, "src", "acme", "service.py"), "def serve():\n    return 1\n")
	writePythonFixture(t, filepath.Join(root, "src", "acme", "service.pyi"), "def serve() -> int: ...\n")
	writePythonFixture(t, filepath.Join(root, "src", "ns", "deep", "worker.py"), "WORKER = True\n")
	writePythonFixture(t, filepath.Join(root, "src", "tests", "test_ignored.py"), "BROKEN = (\n")
	writePythonFixture(t, filepath.Join(root, "src", "__pycache__", "ignored.py"), "CACHE = True\n")
	writePythonFixture(t, filepath.Join(root, "src", "external", "ignored.py"), "EXTERNAL = True\n")
	writePythonFixture(t, filepath.Join(root, "src", "generated", "ignored.py"), "GENERATED = True\n")

	analyzer := New()
	options := pythonOptions(t, nil)
	result, err := analyzer.Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: options})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if result.Status != analysis.StatusComplete || result.Project.Boundary != "pyproject.toml" || result.Project.ModuleRoot != "src" {
		t.Fatalf("result metadata = %#v; diagnostics=%#v", result.Project, result.Diagnostics)
	}
	if result.Project.RootLabel == "" || len(result.Relationships) != 0 || len(result.References) != 0 {
		t.Fatalf("module-only result = %#v", result)
	}
	if hasModule(result.Modules, "py:module:acme.service") == false || hasModule(result.Modules, "py:package:acme") == false || hasModule(result.Modules, "py:package:ns") == false || hasModule(result.Modules, "py:package:ns.deep") == false || hasModule(result.Modules, "py:module:ns.deep.worker") == false {
		t.Fatalf("discovered modules = %#v", moduleIDs(result.Modules))
	}
	if hasModule(result.Modules, "py:module:acme.service.pyi") || hasModule(result.Modules, "py:module:tests.test_ignored") {
		t.Fatalf("default exclusions were ignored: %#v", moduleIDs(result.Modules))
	}
	acme := findModule(result.Modules, "py:package:acme")
	if acme.Metadata["package_kind"] != "regular" || !containsString(acme.SourceReferenceIDs, stableID("file", "src/acme/__init__.py")) {
		t.Fatalf("regular package evidence = %#v", acme)
	}
	namespace := findModule(result.Modules, "py:package:ns")
	if namespace.Metadata["package_kind"] != "namespace" || !containsString(namespace.Tags, "namespace") {
		t.Fatalf("namespace package metadata = %#v", namespace)
	}
	for _, source := range result.SourceReferences {
		if filepath.IsAbs(source.Path) || !strings.HasPrefix(source.Path, "src/") || source.Kind != "file" {
			t.Fatalf("invalid source evidence = %#v", source)
		}
	}

	repeat, err := analyzer.Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: options})
	if err != nil {
		t.Fatalf("repeat analyze: %v", err)
	}
	firstJSON, _ := json.Marshal(result)
	repeatJSON, _ := json.Marshal(repeat)
	if string(firstJSON) != string(repeatJSON) {
		t.Fatalf("analysis is not byte-stable:\n%s\n%s", firstJSON, repeatJSON)
	}

	withStubsAndTests := pythonOptions(t, map[string]any{"include_stubs": true, "include_tests": true})
	expanded, err := analyzer.Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: withStubsAndTests})
	if err != nil {
		t.Fatalf("expanded analyze: %v", err)
	}
	if !hasModule(expanded.Modules, "py:module:acme.service") || !hasModule(expanded.Modules, "py:module:tests.test_ignored") || len(expanded.Diagnostics) == 0 {
		t.Fatalf("expanded result should retain the invalid test diagnostic: %#v", expanded)
	}
	if !hasModule(expanded.Modules, "py:module:acme.service") || !containsString(findModule(expanded.Modules, "py:module:acme.service").Tags, "stub") {
		t.Fatalf("stub evidence was not included: %#v", expanded.Modules)
	}
}

func TestResolveProjectHonorsConfigurationPrecedenceAndSafeRoots(t *testing.T) {
	root := t.TempDir()
	writePythonFixture(t, filepath.Join(root, "pyproject.toml"), "[tool.setuptools.packages.find]\nwhere = ['src']\n")
	writePythonFixture(t, filepath.Join(root, "setup.cfg"), "[options.packages.find]\nwhere = lib\n")
	writePythonFixture(t, filepath.Join(root, "setup.py"), "from setuptools import find_packages\npackages=find_packages(where='other')\n")
	writePythonFixture(t, filepath.Join(root, "src", "pkg", "__init__.py"), "")
	writePythonFixture(t, filepath.Join(root, "lib", "wrong.py"), "")
	writePythonFixture(t, filepath.Join(root, "other", "wrong.py"), "")
	project, err := ResolveProject(root, pythonOptions(t, nil))
	if err != nil {
		t.Fatalf("resolve pyproject: %v", err)
	}
	if len(project.SourceRoots) != 1 || project.SourceRoots[0].Relative != "src" {
		t.Fatalf("pyproject roots = %#v", project.SourceRoots)
	}

	setupRoot := t.TempDir()
	writePythonFixture(t, filepath.Join(setupRoot, "setup.cfg"), "[options.packages.find]\nwhere = lib, source\n")
	writePythonFixture(t, filepath.Join(setupRoot, "lib", "pkg.py"), "")
	project, err = ResolveProject(setupRoot, pythonOptions(t, nil))
	if err != nil {
		t.Fatalf("resolve setup.cfg: %v", err)
	}
	if len(project.SourceRoots) != 1 || project.SourceRoots[0].Relative != "lib" || !hasDiagnostic(project.ConfigDiagnostics, "python_source_root_invalid") {
		t.Fatalf("setup.cfg roots = %#v", project.SourceRoots)
	}

	outside := filepath.Join(root, "..", "python-outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatalf("mkdir outside: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(outside) })
	writePythonFixture(t, filepath.Join(outside, "secret.py"), "")
	unsafeOptions := pythonOptions(t, map[string]any{"source_roots": []string{"../python-outside"}})
	unsafeProject, err := ResolveProject(root, unsafeOptions)
	if err != nil {
		t.Fatalf("resolve unsafe root: %v", err)
	}
	if len(unsafeProject.SourceRoots) != 1 || unsafeProject.SourceRoots[0].Relative != "." {
		t.Fatalf("unsafe source root fallback = %#v", unsafeProject.SourceRoots)
	}
	unsafeResult, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: unsafeOptions})
	if err != nil {
		t.Fatalf("analyze unsafe root: %v", err)
	}
	for _, source := range unsafeResult.SourceReferences {
		if strings.Contains(source.Path, "secret") {
			t.Fatalf("outside source was read: %#v", source)
		}
	}
	if !hasDiagnostic(unsafeResult.Diagnostics, "python_source_root_invalid") {
		t.Fatalf("unsafe root diagnostic missing: %#v", unsafeResult.Diagnostics)
	}
}

func TestResolveProjectReadsSetupPyMetadataAsData(t *testing.T) {
	root := t.TempDir()
	writePythonFixture(t, filepath.Join(root, "setup.py"), `from setuptools import find_namespace_packages
packages = find_namespace_packages(where="src")
open("must-not-run", "w").write("side effect")
`)
	writePythonFixture(t, filepath.Join(root, "src", "package", "module.py"), "VALUE = 1\n")
	project, err := ResolveProject(root, pythonOptions(t, nil))
	if err != nil {
		t.Fatalf("resolve setup.py: %v", err)
	}
	if project.Boundary != "setup.py" || len(project.SourceRoots) != 1 || project.SourceRoots[0].Relative != "src" {
		t.Fatalf("setup.py project = %#v", project)
	}
	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: pythonOptions(t, nil)})
	if err != nil {
		t.Fatalf("analyze setup.py: %v", err)
	}
	if !hasModule(result.Modules, "py:package:package") || !hasModule(result.Modules, "py:module:package.module") {
		t.Fatalf("setup.py modules = %#v", moduleIDs(result.Modules))
	}
	if _, err := os.Stat(filepath.Join(root, "must-not-run")); !os.IsNotExist(err) {
		t.Fatalf("setup.py appears to have executed: %v", err)
	}
}

func TestAnalyzeRetainsUsableModulesForConfigurationAndVersionProblems(t *testing.T) {
	root := t.TempDir()
	writePythonFixture(t, filepath.Join(root, "pyproject.toml"), "[tool.setuptools.packages.find\nwhere = ['src']\n")
	writePythonFixture(t, filepath.Join(root, "fallback.py"), "VALUE = 1\n")
	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: pythonOptions(t, map[string]any{"python_version": "2.7"})})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if result.Status != analysis.StatusPartial || !hasDiagnostic(result.Diagnostics, "python_configuration_invalid") || !hasDiagnostic(result.Diagnostics, "python_unsupported_version") || !hasModule(result.Modules, "py:module:fallback") {
		t.Fatalf("partial result = %#v", result)
	}
}

func TestAnalyzeConflictingLayoutsIsByteStable(t *testing.T) {
	root := t.TempDir()
	writePythonFixture(t, filepath.Join(root, "pyproject.toml"), "[project]\nname = 'fixture'\n")
	for _, name := range []string{"alpha", "beta", "gamma"} {
		writePythonFixture(t, filepath.Join(root, name, "__init__.py"), "")
		writePythonFixture(t, filepath.Join(root, name+".py"), "")
	}
	analyzer := New()
	options := pythonOptions(t, nil)
	var first []byte
	for attempt := 0; attempt < 20; attempt++ {
		result, err := analyzer.Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: options})
		if err != nil {
			t.Fatalf("analyze attempt %d: %v", attempt, err)
		}
		encoded, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("marshal attempt %d: %v", attempt, err)
		}
		if attempt == 0 {
			first = encoded
			continue
		}
		if string(encoded) != string(first) {
			t.Fatalf("conflicting-layout analysis changed on attempt %d:\n%s\n%s", attempt, first, encoded)
		}
	}
}

func pythonOptions(t *testing.T, values map[string]any) analysis.EffectiveOptions {
	t.Helper()
	options, err := analysis.ResolveOptions(New().Manifest(), nil, values)
	if err != nil {
		t.Fatalf("resolve Python options: %v", err)
	}
	return options
}

func writePythonFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}

func hasModule(values []analysis.ModuleObservation, id string) bool {
	for _, value := range values {
		if value.ID == id {
			return true
		}
	}
	return false
}

func findModule(values []analysis.ModuleObservation, id string) analysis.ModuleObservation {
	for _, value := range values {
		if value.ID == id {
			return value
		}
	}
	return analysis.ModuleObservation{}
}

func moduleIDs(values []analysis.ModuleObservation) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.ID)
	}
	return result
}

func hasDiagnostic(values []analysis.Diagnostic, code string) bool {
	for _, value := range values {
		if value.Code == code {
			return true
		}
	}
	return false
}
