package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/distribution"
	"github.com/buffo/arch-view/internal/analysis/processplugin"
	"github.com/buffo/arch-view/internal/analysis/processprotocol"
	clojureanalyzer "github.com/buffo/arch-view/internal/analyzers/clojure"
	goanalyzer "github.com/buffo/arch-view/internal/analyzers/go"
	pyanalyzer "github.com/buffo/arch-view/internal/analyzers/python"
	rustanalyzer "github.com/buffo/arch-view/internal/analyzers/rust"
	tsaanalyzer "github.com/buffo/arch-view/internal/analyzers/typescript"
	"github.com/buffo/arch-view/internal/viewer"
)

const (
	packagedFixtureEnv      = "ARCH_VIEW_CLI_PACKAGED_FIXTURE"
	packagedFixtureAnalyzer = "ARCH_VIEW_CLI_PACKAGED_ANALYZER_ID"
	packagedFixtureBuildID  = "cli-runtime-fixture"
)

// TestPackagedAnalyzerFixtureProcess is launched from a copied test binary by
// the packaged-runtime tests. The child uses the same analyzer implementation
// as the in-process path, so the test exercises only the distribution and
// NDJSON boundary.
func TestPackagedAnalyzerFixtureProcess(t *testing.T) {
	if os.Getenv(packagedFixtureEnv) != "1" {
		return
	}
	analyzer := packagedFixtureAnalyzerForID(os.Getenv(packagedFixtureAnalyzer))
	if analyzer == nil {
		_, _ = fmt.Fprintln(os.Stderr, "unknown packaged analyzer fixture")
		os.Exit(1)
	}
	if err := processplugin.Run(context.Background(), analyzer, os.Stdin, os.Stdout, os.Stderr); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "packaged analyzer fixture failed: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func TestPackagedRuntimeListsWithoutLaunching(t *testing.T) {
	root := buildCLIFixtureCatalog(t, []string{"org.archview.go"})
	setAnalyzerRoot(t, root)

	var stdout, stderr bytes.Buffer
	if code := run([]string{"analyzers", "--analyzer-runtime", "packaged"}, &stdout, &stderr); code != 0 {
		t.Fatalf("packaged listing exit code = %d, stderr=%s", code, stderr.String())
	}
	var response struct {
		Analyzers []analyzerListing `json:"analyzers"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode packaged listing: %v; output=%s", err, stdout.String())
	}
	if len(response.Analyzers) != 1 {
		t.Fatalf("packaged listing = %#v", response.Analyzers)
	}
	listing := response.Analyzers[0]
	if listing.ID != "org.archview.go" || listing.Availability != string(distribution.PackageAvailable) || listing.RuntimeMode != analysis.RuntimeModePackaged || listing.RuntimeSource != "application-index" {
		t.Fatalf("packaged listing metadata = %#v", listing)
	}
}

func TestPackagedRuntimeListingReportsRejectedPackages(t *testing.T) {
	root := buildCLIFixtureCatalog(t, []string{"org.archview.go"})
	setAnalyzerRoot(t, root)
	index, err := distribution.ReadIndex(filepath.Join(root, "index.json"))
	if err != nil {
		t.Fatalf("read fixture index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(index.Packages[0].ExecutablePath)), []byte("tampered"), 0o755); err != nil {
		t.Fatalf("tamper packaged executable: %v", err)
	}

	var stdout, stderr bytes.Buffer
	if code := run([]string{"analyzers", "--analyzer-runtime", "packaged"}, &stdout, &stderr); code != 0 {
		t.Fatalf("rejected package listing exit code = %d, stderr=%s", code, stderr.String())
	}
	var response struct {
		Analyzers []analyzerListing `json:"analyzers"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode rejected package listing: %v", err)
	}
	if len(response.Analyzers) != 1 || response.Analyzers[0].Availability != string(distribution.PackageRejected) || response.Analyzers[0].ErrorCode != string(analysis.ErrAnalyzerPackageIntegrityMismatch) {
		t.Fatalf("rejected package listing = %#v", response.Analyzers)
	}
}

func TestPackagedRuntimeRequiresExplicitFallbackAndOverride(t *testing.T) {
	missingRoot := filepath.Join(t.TempDir(), "missing-analyzers")
	setAnalyzerRoot(t, missingRoot)

	var stdout, stderr bytes.Buffer
	if code := run([]string{"analyzers", "--analyzer-runtime", "packaged"}, &stdout, &stderr); code != 2 {
		t.Fatalf("missing packaged catalog exit code = %d, want 2; stderr=%s", code, stderr.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte(analysis.ErrAnalyzerPackageIndexInvalid)) {
		t.Fatalf("missing packaged catalog error = %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"analyzers", "--analyzer-runtime", "in-process"}, &stdout, &stderr); code != 0 {
		t.Fatalf("explicit in-process listing exit code = %d, stderr=%s", code, stderr.String())
	}
	var response struct {
		Analyzers []analyzerListing `json:"analyzers"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode in-process listing: %v", err)
	}
	if len(response.Analyzers) != 5 || response.Analyzers[0].RuntimeMode != analysis.RuntimeModeInProcess {
		t.Fatalf("in-process listing = %#v", response.Analyzers)
	}

	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"analyzers", "--analyzer-runtime", "auto", "--plugin", filepath.Join(t.TempDir(), "descriptor.json")}, &stdout, &stderr); code != 2 {
		t.Fatalf("auto plugin override exit code = %d, want 2; stderr=%s", code, stderr.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte(analysis.ErrAnalyzerRuntimeOverrideRequired)) {
		t.Fatalf("auto plugin override error = %s", stderr.String())
	}
}

func TestExplicitDescriptorRuntimeRequiresOptIn(t *testing.T) {
	t.Setenv("ARCH_VIEW_ANALYZER_ROOT", "")
	t.Setenv("ARCH_VIEW_ANALYZERS_ROOT", "")
	descriptor := externalPythonDescriptorPath(t)

	var stdout, stderr bytes.Buffer
	if code := run([]string{"analyzers", "--analyzer-runtime", "explicit", "--plugin", descriptor}, &stdout, &stderr); code != 2 {
		t.Fatalf("unapproved explicit descriptor exit code = %d, want 2; stderr=%s", code, stderr.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte(analysis.ErrAnalyzerRuntimeOverrideRequired)) {
		t.Fatalf("unapproved explicit descriptor error = %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"analyzers", "--analyzer-runtime", "explicit", "--allow-untrusted-plugin", "--plugin", descriptor}, &stdout, &stderr); code != 0 {
		t.Fatalf("approved explicit descriptor exit code = %d, stderr=%s", code, stderr.String())
	}
	var response struct {
		Analyzers []analyzerListing `json:"analyzers"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode explicit descriptor listing: %v", err)
	}
	if len(response.Analyzers) != 1 || response.Analyzers[0].ID != "org.archview.python.external" || response.Analyzers[0].RuntimeMode != analysis.RuntimeModeExplicit {
		t.Fatalf("explicit descriptor listing = %#v", response.Analyzers)
	}
}

func TestTamperedPackagedRuntimeDoesNotFallback(t *testing.T) {
	root := buildCLIFixtureCatalog(t, []string{"org.archview.go"})
	setAnalyzerRoot(t, root)
	index, err := distribution.ReadIndex(filepath.Join(root, "index.json"))
	if err != nil {
		t.Fatalf("read tampered fixture index: %v", err)
	}
	executablePath := filepath.Join(root, filepath.FromSlash(index.Packages[0].ExecutablePath))
	if err := os.WriteFile(executablePath, []byte("tampered"), 0o755); err != nil {
		t.Fatalf("tamper packaged executable: %v", err)
	}
	project := writeCLIParityProject(t, "org.archview.go")
	var stdout, stderr bytes.Buffer
	if code := run([]string{"analyze", "--project", project, "--analyzer", "org.archview.go", "--analyzer-runtime", "packaged", "--output", filepath.Join(t.TempDir(), "rejected.json")}, &stdout, &stderr); code != 4 {
		t.Fatalf("tampered packaged exit code = %d, want 4; stderr=%s", code, stderr.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte(analysis.ErrAnalyzerPackageIntegrityMismatch)) {
		t.Fatalf("tampered packaged error = %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"analyze", "--project", project, "--analyzer-runtime", "packaged", "--output", filepath.Join(t.TempDir(), "auto-rejected.json")}, &stdout, &stderr); code != 4 {
		t.Fatalf("auto-selected tampered package exit code = %d, want 4; stderr=%s", code, stderr.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte(analysis.ErrAnalyzerPackageIntegrityMismatch)) {
		t.Fatalf("auto-selected tampered package error = %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"analyze", "--project", project, "--analyzer", "org.archview.go", "--analyzer-runtime", "in-process", "--output", filepath.Join(t.TempDir(), "fallback.json")}, &stdout, &stderr); code != 0 {
		t.Fatalf("explicit in-process fallback exit code = %d, stderr=%s", code, stderr.String())
	}
}

func TestPackagedRuntimeIgnoresProjectLocalDescriptors(t *testing.T) {
	root := buildCLIFixtureCatalog(t, []string{"org.archview.go"})
	setAnalyzerRoot(t, root)
	t.Setenv(packagedFixtureEnv, "1")
	t.Setenv(packagedFixtureAnalyzer, "org.archview.go")
	project := writeCLIParityProject(t, "org.archview.go")
	writeCLIFile(t, project, "external-plugin.json", "this file must not be read as a descriptor\n")
	writeCLIFile(t, project, "analyzer.exe", "this file must not be executed\n")

	result := runCLIAnalysis(t, project, "org.archview.go", analysis.RuntimeModePackaged)
	if result.Analyzer.ID != "org.archview.go" || result.Analyzer.RuntimeSource != "application-index" {
		t.Fatalf("packaged result provenance = %#v", result.Analyzer)
	}
}

func TestPackagedAndInProcessRuntimeParityForAllAnalyzers(t *testing.T) {
	ids := distribution.DefaultAnalyzerIDs()
	root := buildCLIFixtureCatalog(t, ids)
	setAnalyzerRoot(t, root)
	t.Setenv(packagedFixtureEnv, "1")

	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			t.Setenv(packagedFixtureAnalyzer, id)
			project := writeCLIParityProject(t, id)
			inProcess := runCLIAnalysis(t, project, id, analysis.RuntimeModeInProcess)
			packaged := runCLIAnalysis(t, project, id, analysis.RuntimeModePackaged)

			inProcess.Analyzer.RuntimeMode = ""
			inProcess.Analyzer.RuntimeSource = ""
			inProcess.Analyzer.RuntimePlatform = ""
			packaged.Analyzer.RuntimeMode = ""
			packaged.Analyzer.RuntimeSource = ""
			packaged.Analyzer.RuntimePlatform = ""
			if !reflect.DeepEqual(packaged, inProcess) {
				t.Fatalf("packaged and in-process results differ\npackaged=%#v\nin-process=%#v", packaged, inProcess)
			}
		})
	}
}

func TestProjectViewerReanalysisRetainsRuntimeSelection(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "runtime.marker"), []byte("fixture\n"), 0o644); err != nil {
		t.Fatalf("write runtime marker: %v", err)
	}
	analyzer := &runtimeCaptureAnalyzer{}
	registry := analysis.NewRegistry()
	if err := registry.Register(analyzer); err != nil {
		t.Fatalf("register capture analyzer: %v", err)
	}
	host := analysis.NewHostWithRuntime(registry, analysis.RuntimeSelection{
		Mode:     analysis.RuntimeModePackaged,
		Source:   "application-index",
		Platform: distribution.HostPlatform(),
	})
	options := projectViewerOptions(host, root, "fixture", "org.example.runtime-capture", map[string]any{})
	value, err := options.Reanalyze(context.Background(), viewer.ReanalysisRequest{ProjectRoot: root, Language: "fixture", Options: map[string]any{}})
	if err != nil {
		t.Fatalf("reanalysis: %v", err)
	}
	if value.Project.Language != "fixture" {
		t.Fatalf("reanalysis model language = %q", value.Project.Language)
	}
	if analyzer.selection.RuntimeMode != analysis.RuntimeModePackaged || analyzer.selection.RuntimeSource != "application-index" {
		t.Fatalf("reanalysis selection = %#v", analyzer.selection)
	}
}

func buildCLIFixtureCatalog(t *testing.T, ids []string) string {
	t.Helper()
	repositoryRoot := t.TempDir()
	outputRoot := filepath.Join(t.TempDir(), "analyzers")
	binary, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatalf("read test binary: %v", err)
	}
	_, err = distribution.Assemble(context.Background(), distribution.AssembleOptions{
		RepositoryRoot: repositoryRoot,
		OutputRoot:     outputRoot,
		Platform:       distribution.HostPlatform(),
		BuildID:        packagedFixtureBuildID,
		GoCommand:      "test-go",
		CCCommand:      "test-cc",
		CXXCommand:     "test-cxx",
		AnalyzerIDs:    ids,
		Build: func(_ context.Context, request distribution.BuildRequest) error {
			if err := os.MkdirAll(filepath.Dir(request.OutputPath), 0o755); err != nil {
				return err
			}
			return os.WriteFile(request.OutputPath, binary, 0o755)
		},
	})
	if err != nil {
		t.Fatalf("assemble CLI fixture catalog: %v", err)
	}
	addPackagedFixtureArguments(t, outputRoot)
	return outputRoot
}

func addPackagedFixtureArguments(t *testing.T, root string) {
	t.Helper()
	indexPath := filepath.Join(root, "index.json")
	index, err := distribution.ReadIndex(indexPath)
	if err != nil {
		t.Fatalf("read fixture index: %v", err)
	}
	for indexNumber := range index.Packages {
		entry := &index.Packages[indexNumber]
		descriptorPath := filepath.Join(root, filepath.FromSlash(entry.DescriptorPath))
		data, err := os.ReadFile(descriptorPath)
		if err != nil {
			t.Fatalf("read fixture descriptor: %v", err)
		}
		var descriptor processprotocol.Descriptor
		if err := json.Unmarshal(data, &descriptor); err != nil {
			t.Fatalf("decode fixture descriptor: %v", err)
		}
		descriptor.Args = []string{"-test.run", "^TestPackagedAnalyzerFixtureProcess$", "--"}
		data, err = json.MarshalIndent(descriptor, "", "  ")
		if err != nil {
			t.Fatalf("marshal fixture descriptor: %v", err)
		}
		data = append(data, '\n')
		if err := os.WriteFile(descriptorPath, data, 0o644); err != nil {
			t.Fatalf("write fixture descriptor: %v", err)
		}
		digest := sha256.Sum256(data)
		entry.DescriptorSHA256 = hex.EncodeToString(digest[:])
	}
	data, err := distribution.MarshalIndex(index)
	if err != nil {
		t.Fatalf("marshal fixture index: %v", err)
	}
	if err := os.WriteFile(indexPath, data, 0o644); err != nil {
		t.Fatalf("write fixture index: %v", err)
	}
	if _, err := distribution.LoadCatalog(root); err != nil {
		t.Fatalf("verify fixture catalog after arguments: %v", err)
	}
}

func runCLIAnalysis(t *testing.T, project, analyzerID, runtimeMode string) analysis.AnalysisResult {
	t.Helper()
	output := filepath.Join(t.TempDir(), "analysis.json")
	var stdout, stderr bytes.Buffer
	args := []string{
		"analyze", "--project", project, "--analyzer", analyzerID,
		"--analyzer-runtime", runtimeMode, "--format", "analysis-json", "--output", output,
	}
	if code := run(args, &stdout, &stderr); code != 0 {
		t.Fatalf("%s analysis exit code = %d, stderr=%s", runtimeMode, code, stderr.String())
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read %s analysis: %v", runtimeMode, err)
	}
	var result analysis.AnalysisResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("decode %s analysis: %v", runtimeMode, err)
	}
	if result.Analyzer.RuntimeMode != runtimeMode {
		t.Fatalf("%s runtime provenance = %#v", runtimeMode, result.Analyzer)
	}
	return result
}

func writeCLIParityProject(t *testing.T, analyzerID string) string {
	t.Helper()
	root := t.TempDir()
	switch analyzerID {
	case "org.archview.clojure":
		writeCLIFile(t, root, "deps.edn", "{:paths [\"src\"]}\n")
		writeCLIFile(t, root, filepath.Join("src", "app", "core.clj"), "(ns app.core)\n")
	case "org.archview.go":
		writeCLIFile(t, root, "go.mod", "module example.com/fixture\n")
		writeCLIFile(t, root, "main.go", "package fixture\n\nconst Answer = 42\n")
	case "org.archview.python":
		writeCLIFile(t, root, "pyproject.toml", "[project]\nname = 'fixture'\n")
		writeCLIFile(t, root, "module.py", "VALUE = 42\n")
	case "org.archview.rust":
		writeCLIFile(t, root, "Cargo.toml", "[package]\nname = \"fixture\"\nedition = \"2021\"\n")
		writeCLIFile(t, root, filepath.Join("src", "lib.rs"), "pub fn answer() -> u32 { 42 }\n")
	case "org.archview.typescript":
		writeCLIFile(t, root, "tsconfig.json", `{"include":["src/**/*"]}`)
		writeCLIFile(t, root, filepath.Join("src", "main.ts"), "export const answer = 42;\n")
	default:
		t.Fatalf("unknown analyzer ID %q", analyzerID)
	}
	return root
}

func writeCLIFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture file %s: %v", name, err)
	}
}

func setAnalyzerRoot(t *testing.T, root string) {
	t.Helper()
	t.Setenv("ARCH_VIEW_ANALYZER_ROOT", root)
	t.Setenv("ARCH_VIEW_ANALYZERS_ROOT", "")
}

func packagedFixtureAnalyzerForID(id string) analysis.Analyzer {
	switch id {
	case "org.archview.clojure":
		return clojureanalyzer.New()
	case "org.archview.go":
		return goanalyzer.New()
	case "org.archview.python":
		return pyanalyzer.New()
	case "org.archview.rust":
		return rustanalyzer.New()
	case "org.archview.typescript":
		return tsaanalyzer.New()
	default:
		return nil
	}
}

type runtimeCaptureAnalyzer struct {
	selection analysis.AnalyzerSelection
}

func (*runtimeCaptureAnalyzer) Manifest() analysis.Manifest {
	return analysis.Manifest{
		ID:               "org.example.runtime-capture",
		Version:          "1.0.0",
		Language:         "fixture",
		APIVersion:       analysis.AnalyzerAPIVersion,
		DetectionMarkers: []analysis.DetectionMarker{{Kind: "file", Value: "runtime.marker", Weight: 1}},
		Capabilities:     []string{"detect"},
		Options:          []analysis.OptionDescriptor{},
	}
}

func (a *runtimeCaptureAnalyzer) Detect(context.Context, analysis.DetectRequest) (analysis.DetectionCandidate, error) {
	return analysis.DetectionCandidate{AnalyzerID: a.Manifest().ID, Confidence: 1, Reason: "runtime marker"}, nil
}

func (a *runtimeCaptureAnalyzer) Analyze(_ context.Context, request analysis.AnalyzeRequest) (analysis.AnalysisResult, error) {
	a.selection = request.Selection
	manifest := a.Manifest()
	return analysis.AnalysisResult{
		Status:             analysis.StatusComplete,
		Analyzer:           analysis.AnalyzerInfo{ID: manifest.ID, Version: manifest.Version, Language: manifest.Language, APIVersion: manifest.APIVersion},
		Project:            analysis.ProjectInfo{RootLabel: "fixture", Boundary: "runtime.marker"},
		OptionsFingerprint: request.Options.Fingerprint,
	}, nil
}
