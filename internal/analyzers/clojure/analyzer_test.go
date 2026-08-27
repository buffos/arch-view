package clojureanalyzer

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
)

func TestManifestAndDetection(t *testing.T) {
	analyzer := New()
	manifest := analyzer.Manifest()
	if manifest.ID != "org.archview.clojure" || manifest.Language != "clojure" || manifest.APIVersion != analysis.AnalyzerAPIVersion {
		t.Fatalf("manifest = %#v", manifest)
	}
	if len(manifest.DetectionMarkers) != 3 || manifest.Capabilities == nil {
		t.Fatalf("manifest markers/capabilities = %#v", manifest)
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "project.clj"), []byte("(defproject sample \"0.1.0\")\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	candidate, err := analyzer.Detect(context.Background(), analysis.DetectRequest{ProjectRoot: root})
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if candidate.AnalyzerID != manifest.ID || candidate.Confidence != 0.9 || candidate.BoundaryHint != "project.clj" || len(candidate.MatchedMarkers) != 1 {
		t.Fatalf("candidate = %#v", candidate)
	}
}

func TestResolveProjectUsesBoundaryAndSourceRootPrecedence(t *testing.T) {
	root := t.TempDir()
	writeClojureFile(t, root, "deps.edn", "{:paths [\"configured\"]}\n")
	writeClojureFile(t, root, "project.clj", "(defproject sample \"0.1.0\" :source-paths [\"project-src\"])\n")
	writeClojureFile(t, root, "shadow-cljs.edn", "{:source-paths [\"shadow-src\"]}\n")
	for _, directory := range []string{"configured", "explicit"} {
		if err := os.MkdirAll(filepath.Join(root, directory), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	project, err := ResolveProject(root, testClojureOptions(t, nil))
	if err != nil {
		t.Fatalf("ResolveProject() error = %v", err)
	}
	if project.Boundary != "deps.edn" || len(project.SourceRoots) != 1 || project.SourceRoots[0].Relative != "configured" {
		t.Fatalf("project = %#v", project)
	}

	project, err = ResolveProject(root, testClojureOptions(t, map[string]any{"source_roots": []string{"explicit"}}))
	if err != nil {
		t.Fatalf("ResolveProject(explicit) error = %v", err)
	}
	if len(project.SourceRoots) != 1 || project.SourceRoots[0].Relative != "explicit" {
		t.Fatalf("explicit project roots = %#v", project.SourceRoots)
	}

	projectOnly := t.TempDir()
	writeClojureFile(t, projectOnly, "project.clj", "(defproject sample \"0.1.0\" :source-paths [\"project-src\"])\n")
	if err := os.MkdirAll(filepath.Join(projectOnly, "project-src"), 0o755); err != nil {
		t.Fatal(err)
	}
	project, err = ResolveProject(projectOnly, testClojureOptions(t, nil))
	if err != nil || project.Boundary != "project.clj" || len(project.SourceRoots) != 1 || project.SourceRoots[0].Relative != "project-src" {
		t.Fatalf("project.clj resolution = %#v, error = %v", project, err)
	}

	shadowOnly := t.TempDir()
	writeClojureFile(t, shadowOnly, "shadow-cljs.edn", "{:source-paths [\"shadow-src\"]}\n")
	if err := os.MkdirAll(filepath.Join(shadowOnly, "shadow-src"), 0o755); err != nil {
		t.Fatal(err)
	}
	project, err = ResolveProject(shadowOnly, testClojureOptions(t, nil))
	if err != nil || project.Boundary != "shadow-cljs.edn" || len(project.SourceRoots) != 1 || project.SourceRoots[0].Relative != "shadow-src" {
		t.Fatalf("shadow-cljs.edn resolution = %#v, error = %v", project, err)
	}
}

func TestResolveProjectRejectsEscapingAndSymlinkedRoots(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeClojureFile(t, root, "deps.edn", "{:paths [\"src\"]}\n")
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	options := testClojureOptions(t, map[string]any{"source_roots": []string{"../outside"}})
	project, err := ResolveProject(root, options)
	if err != nil {
		t.Fatalf("ResolveProject() error = %v", err)
	}
	if hasSourceRoot(project.SourceRoots, "../outside") || !hasDiagnostic(project.ConfigDiagnostics, "clojure_source_root_invalid") {
		t.Fatalf("escaping root was not rejected: %#v", project)
	}

	link := filepath.Join(root, "linked")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("directory symlinks are unavailable: %v", err)
	}
	project, err = ResolveProject(root, testClojureOptions(t, map[string]any{"source_roots": []string{"linked"}}))
	if err != nil {
		t.Fatalf("ResolveProject(symlink) error = %v", err)
	}
	if hasSourceRoot(project.SourceRoots, "linked") || !hasDiagnostic(project.ConfigDiagnostics, "clojure_source_root_invalid") {
		t.Fatalf("outside symlink root was not rejected: %#v", project)
	}
}

func TestDiscoverNamespacesFlavorsTestsExclusionsAndEvidence(t *testing.T) {
	root := t.TempDir()
	writeClojureFile(t, root, "deps.edn", "{:paths [\"src\" \"test\"]}\n")
	writeClojureFile(t, root, "src/app/core.clj", "(ns app.core)\n(def value 1)\n")
	writeClojureFile(t, root, "src/app/web.cljs", "(ns app.web)\n")
	writeClojureFile(t, root, "src/app/shared.cljc", "(ns app.shared)\n")
	writeClojureFile(t, root, "src/app/no_ns.clj", "(def value (str \"not evaluated\"))\n")
	writeClojureFile(t, root, "src/generated/app/generated.clj", "(ns app.generated)\n")
	writeClojureFile(t, root, "src/app/excluded.clj", "(ns app.excluded)\n")
	writeClojureFile(t, root, "src/app/recursive/nested.clj", "(ns app.recursive.nested)\n")
	writeClojureFile(t, root, "src/app/quoted_ns.clj", "'(ns app.faked)\n")
	writeClojureFile(t, root, "test/app/core_test.clj", "(ns app.core)\n")

	excludeOptions := map[string]any{"exclude": []string{"**/excluded.clj", "src/app/recursive/**"}}
	project, err := ResolveProject(root, testClojureOptions(t, excludeOptions))
	if err != nil {
		t.Fatalf("ResolveProject() error = %v", err)
	}
	discovery, err := Discover(context.Background(), project, testClojureOptions(t, excludeOptions))
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(discovery.Modules) != 3 {
		t.Fatalf("modules = %#v", discovery.Modules)
	}
	if hasModule(discovery.Modules, "clj:app.generated") || hasModule(discovery.Modules, "clj:app.excluded") || hasModule(discovery.Modules, "clj:app.recursive.nested") || hasModule(discovery.Modules, "clj:app.faked") {
		t.Fatalf("excluded module was discovered: %#v", discovery.Modules)
	}
	if !hasDiagnostic(discovery.Diagnostics, "clojure_missing_ns") {
		t.Fatalf("missing ns diagnostic absent: %#v", discovery.Diagnostics)
	}
	shared := findModule(discovery.Modules, "clj:app.shared")
	if shared == nil || !metadataContainsString(shared.Metadata, "flavors", "cljc") || len(shared.SourceReferenceIDs) != 2 {
		t.Fatalf("shared module = %#v", shared)
	}
	if hasPath(discovery.SourceReferences, "test/app/core_test.clj") {
		t.Fatalf("test file should be opt-in: %#v", discovery.SourceReferences)
	}
	if !hasPath(discovery.SourceReferences, "src/app/core.clj") {
		t.Fatalf("source evidence missing: %#v", discovery.SourceReferences)
	}

	testOptions := testClojureOptions(t, map[string]any{"include_tests": true, "exclude": []string{"src/app/excluded.clj"}})
	testDiscovery, err := Discover(context.Background(), project, testOptions)
	if err != nil {
		t.Fatalf("Discover(include_tests) error = %v", err)
	}
	if !hasPath(testDiscovery.SourceReferences, "test/app/core_test.clj") {
		t.Fatalf("test file was not included: %#v", testDiscovery.SourceReferences)
	}
}

func TestDiscoverPlatformSelectionAndDeterminism(t *testing.T) {
	root := t.TempDir()
	writeClojureFile(t, root, "deps.edn", "{:paths [\"src\"]}\n")
	writeClojureFile(t, root, "src/app/core.clj", "(ns app.core)\n")
	writeClojureFile(t, root, "src/app/web.cljs", "(ns app.web)\n")
	writeClojureFile(t, root, "src/app/shared.cljc", "(ns app.shared)\n")

	cljOptions := testClojureOptions(t, map[string]any{"platform": "clj"})
	project, err := ResolveProject(root, cljOptions)
	if err != nil {
		t.Fatal(err)
	}
	first, err := Discover(context.Background(), project, cljOptions)
	if err != nil {
		t.Fatal(err)
	}
	if hasModule(first.Modules, "clj:app.web") || !hasModule(first.Modules, "clj:app.core") || !hasModule(first.Modules, "clj:app.shared") {
		t.Fatalf("clj modules = %#v", first.Modules)
	}

	cljsOptions := testClojureOptions(t, map[string]any{"platform": "cljs"})
	cljsProject, err := ResolveProject(root, cljsOptions)
	if err != nil {
		t.Fatal(err)
	}
	cljsDiscovery, err := Discover(context.Background(), cljsProject, cljsOptions)
	if err != nil {
		t.Fatal(err)
	}
	if hasModule(cljsDiscovery.Modules, "clj:app.core") || !hasModule(cljsDiscovery.Modules, "clj:app.web") || !hasModule(cljsDiscovery.Modules, "clj:app.shared") {
		t.Fatalf("cljs modules = %#v", cljsDiscovery.Modules)
	}

	second, err := Discover(context.Background(), project, cljOptions)
	if err != nil {
		t.Fatal(err)
	}
	firstJSON, _ := json.Marshal(BuildResult(project, first, New().Manifest()))
	secondJSON, _ := json.Marshal(BuildResult(project, second, New().Manifest()))
	if string(firstJSON) != string(secondJSON) {
		t.Fatalf("repeated discovery differs:\n%s\n%s", firstJSON, secondJSON)
	}
}

func TestDiscoverHonorsCancellationAndDoesNotEvaluateSource(t *testing.T) {
	root := t.TempDir()
	writeClojureFile(t, root, "deps.edn", "{:paths [\"src\"]}\n")
	writeClojureFile(t, root, "src/app/safe.clj", "(ns app.safe)\n(spit \"SHOULD-NOT-EXIST\" \"executed\")\n")
	options := testClojureOptions(t, nil)
	project, err := ResolveProject(root, options)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Discover(ctx, project, options); !errors.Is(err, context.Canceled) {
		t.Fatalf("Discover(cancelled) error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "SHOULD-NOT-EXIST")); !os.IsNotExist(err) {
		t.Fatalf("source appears to have been evaluated: %v", err)
	}
	discovery, err := Discover(context.Background(), project, options)
	if err != nil {
		t.Fatal(err)
	}
	if !hasModule(discovery.Modules, "clj:app.safe") {
		t.Fatalf("safe source result = %#v", discovery)
	}
}

func testClojureOptions(t *testing.T, cli map[string]any) analysis.EffectiveOptions {
	t.Helper()
	options, err := analysis.ResolveOptions(New().Manifest(), nil, cli)
	if err != nil {
		t.Fatalf("ResolveOptions() error = %v", err)
	}
	return options
}

func writeClojureFile(t *testing.T, root, relative, content string) {
	t.Helper()
	pathValue := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(pathValue), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pathValue, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func hasSourceRoot(roots []SourceRoot, relative string) bool {
	for _, root := range roots {
		if root.Relative == relative {
			return true
		}
	}
	return false
}

func hasDiagnostic(diagnostics []analysis.Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func hasModule(modules []analysis.ModuleObservation, id string) bool {
	return findModule(modules, id) != nil
}

func findModule(modules []analysis.ModuleObservation, id string) *analysis.ModuleObservation {
	for index := range modules {
		if modules[index].ID == id {
			return &modules[index]
		}
	}
	return nil
}

func metadataContainsString(metadata map[string]any, key, expected string) bool {
	values, ok := metadata[key].([]string)
	if !ok {
		return false
	}
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func hasPath(sources []analysis.SourceReference, expected string) bool {
	for _, source := range sources {
		if source.Path == expected {
			return true
		}
	}
	return false
}
