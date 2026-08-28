package distribution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/analysis/processprotocol"
)

func TestAssembleCreatesDeterministicIndexAndPackages(t *testing.T) {
	root := t.TempDir()
	outputRoot := filepath.Join(root, "dist", "analyzers")
	ids := DefaultAnalyzerIDs()
	requests := make([]BuildRequest, 0, len(ids))

	build := func(_ context.Context, request BuildRequest) error {
		requests = append(requests, request)
		if err := os.MkdirAll(filepath.Dir(request.OutputPath), 0o755); err != nil {
			return err
		}
		content := []byte("compiled:" + request.Entrypoint + ":" + request.GOOS + ":" + request.GOARCH)
		return os.WriteFile(request.OutputPath, content, 0o755)
	}

	index, err := Assemble(context.Background(), AssembleOptions{
		RepositoryRoot: root,
		OutputRoot:     outputRoot,
		Platform:       "windows-amd64",
		BuildID:        "deterministic-test",
		GoCommand:      "explicit-test-toolchain",
		CCCommand:      "explicit-test-cc",
		CXXCommand:     "explicit-test-cxx",
		AnalyzerIDs:    ids,
		Build:          build,
	})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}

	if index.SchemaVersion != IndexSchemaVersion {
		t.Fatalf("schema version = %q", index.SchemaVersion)
	}
	if index.HostAPIVersion != processprotocol.ProtocolVersion {
		t.Fatalf("host API version = %q", index.HostAPIVersion)
	}
	if index.GeneratedBy != "arch-view-build-deterministic-test" {
		t.Fatalf("generated_by = %q", index.GeneratedBy)
	}
	if got, want := len(index.Packages), len(ids); got != want {
		t.Fatalf("package count = %d, want %d", got, want)
	}

	wantIDs := append([]string(nil), ids...)
	sort.Strings(wantIDs)
	for i, packageEntry := range index.Packages {
		if packageEntry.LogicalAnalyzerID != wantIDs[i] {
			t.Fatalf("package %d id = %q, want %q", i, packageEntry.LogicalAnalyzerID, wantIDs[i])
		}
		if packageEntry.Platform != "windows-amd64" {
			t.Fatalf("package %d platform = %q", i, packageEntry.Platform)
		}
		if !strings.HasSuffix(packageEntry.ExecutablePath, "/windows-amd64/analyzer.exe") {
			t.Fatalf("package %d executable path = %q", i, packageEntry.ExecutablePath)
		}
		if !strings.HasSuffix(packageEntry.DescriptorPath, "/windows-amd64/descriptor.json") {
			t.Fatalf("package %d descriptor path = %q", i, packageEntry.DescriptorPath)
		}

		descriptorData := mustReadDistributionFile(t, outputRoot, packageEntry.DescriptorPath)
		descriptor, err := processprotocol.DecodeDescriptor(descriptorData)
		if err != nil {
			t.Fatalf("decode %s: %v", packageEntry.DescriptorPath, err)
		}
		if descriptor.Command != "analyzer.exe" || len(descriptor.Args) != 0 {
			t.Fatalf("descriptor command/args = %q/%#v", descriptor.Command, descriptor.Args)
		}
		if descriptor.Manifest.ID != packageEntry.LogicalAnalyzerID {
			t.Fatalf("descriptor id = %q, index id = %q", descriptor.Manifest.ID, packageEntry.LogicalAnalyzerID)
		}
		if got := sha256File(t, filepath.Join(outputRoot, filepath.FromSlash(packageEntry.ExecutablePath))); got != packageEntry.ExecutableSHA256 {
			t.Fatalf("executable digest = %q, index digest = %q", got, packageEntry.ExecutableSHA256)
		}
		if got := sha256.Sum256(descriptorData); hex.EncodeToString(got[:]) != packageEntry.DescriptorSHA256 {
			t.Fatalf("descriptor digest does not match index for %s", packageEntry.DescriptorPath)
		}
	}

	if err := ValidateIndex(outputRoot, index, ids, "windows-amd64"); err != nil {
		t.Fatalf("validate generated index: %v", err)
	}
	firstIndexData := mustReadDistributionFile(t, outputRoot, "index.json")

	secondOutputRoot := filepath.Join(root, "repeat", "analyzers")
	secondIndex, err := Assemble(context.Background(), AssembleOptions{
		RepositoryRoot: root,
		OutputRoot:     secondOutputRoot,
		Platform:       "windows-amd64",
		BuildID:        "deterministic-test",
		GoCommand:      "explicit-test-toolchain",
		CCCommand:      "explicit-test-cc",
		CXXCommand:     "explicit-test-cxx",
		AnalyzerIDs:    ids,
		Build:          build,
	})
	if err != nil {
		t.Fatalf("repeat assemble: %v", err)
	}
	secondIndexData := mustReadDistributionFile(t, secondOutputRoot, "index.json")
	if !reflect.DeepEqual(index, secondIndex) {
		t.Fatalf("repeat index differs:\nfirst=%#v\nsecond=%#v", index, secondIndex)
	}
	if string(firstIndexData) != string(secondIndexData) {
		t.Fatalf("repeat index bytes differ:\nfirst=%s\nsecond=%s", firstIndexData, secondIndexData)
	}

	if got, want := len(requests), len(ids)*2; got != want {
		t.Fatalf("build request count = %d, want %d", got, want)
	}
	for _, request := range requests {
		if request.RepositoryRoot != root {
			t.Fatalf("repository root = %q, want %q", request.RepositoryRoot, root)
		}
		if request.GoCommand != "explicit-test-toolchain" {
			t.Fatalf("go command = %q", request.GoCommand)
		}
		if request.CCCommand != "explicit-test-cc" || request.CXXCommand != "explicit-test-cxx" {
			t.Fatalf("C/C++ commands = %q/%q", request.CCCommand, request.CXXCommand)
		}
		if request.GOOS != "windows" || request.GOARCH != "amd64" || request.CGOEnabled != "1" {
			t.Fatalf("target environment = %q/%q/%q", request.GOOS, request.GOARCH, request.CGOEnabled)
		}
		if !strings.HasPrefix(request.Entrypoint, "./cmd/analyzers/") {
			t.Fatalf("entrypoint = %q", request.Entrypoint)
		}
	}
}

func TestAssembleSupportsFirstTargetMatrix(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		goos   string
		goarch string
		exe    string
	}{
		{name: "windows", value: "windows-amd64", goos: "windows", goarch: "amd64", exe: "analyzer.exe"},
		{name: "linux", value: "linux-amd64", goos: "linux", goarch: "amd64", exe: "analyzer"},
		{name: "darwin", value: "darwin-arm64", goos: "darwin", goarch: "arm64", exe: "analyzer"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			var request BuildRequest
			index, err := Assemble(context.Background(), AssembleOptions{
				RepositoryRoot: root,
				OutputRoot:     filepath.Join(root, "analyzers"),
				Platform:       test.value,
				BuildID:        "matrix-test",
				GoCommand:      "matrix-toolchain",
				CCCommand:      "matrix-cc",
				CXXCommand:     "matrix-cxx",
				AnalyzerIDs:    []string{"org.archview.go"},
				Build: func(_ context.Context, got BuildRequest) error {
					request = got
					if err := os.MkdirAll(filepath.Dir(got.OutputPath), 0o755); err != nil {
						return err
					}
					return os.WriteFile(got.OutputPath, []byte("binary"), 0o755)
				},
			})
			if err != nil {
				t.Fatalf("assemble: %v", err)
			}
			if request.GOOS != test.goos || request.GOARCH != test.goarch {
				t.Fatalf("target = %q/%q, want %q/%q", request.GOOS, request.GOARCH, test.goos, test.goarch)
			}
			if filepath.Base(index.Packages[0].ExecutablePath) != test.exe {
				t.Fatalf("executable path = %q, want base %q", index.Packages[0].ExecutablePath, test.exe)
			}
		})
	}
}

func TestAssembleRejectsInvalidEnabledSetAndPlatform(t *testing.T) {
	base := AssembleOptions{
		RepositoryRoot: t.TempDir(),
		OutputRoot:     filepath.Join(t.TempDir(), "analyzers"),
		BuildID:        "validation-test",
		GoCommand:      "explicit-toolchain",
		CCCommand:      "explicit-cc",
		CXXCommand:     "explicit-cxx",
		Build: func(context.Context, BuildRequest) error {
			return errors.New("should not build")
		},
	}

	tests := []struct {
		name string
		edit func(*AssembleOptions)
	}{
		{name: "unsupported platform", edit: func(options *AssembleOptions) { options.Platform = "linux-arm64" }},
		{name: "blank analyzer", edit: func(options *AssembleOptions) {
			options.Platform = "windows-amd64"
			options.AnalyzerIDs = []string{"org.archview.go", ""}
		}},
		{name: "duplicate analyzer", edit: func(options *AssembleOptions) {
			options.Platform = "windows-amd64"
			options.AnalyzerIDs = []string{"org.archview.go", "org.archview.go"}
		}},
		{name: "unknown analyzer", edit: func(options *AssembleOptions) {
			options.Platform = "windows-amd64"
			options.AnalyzerIDs = []string{"org.archview.unknown"}
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options := base
			test.edit(&options)
			if _, err := Assemble(context.Background(), options); err == nil {
				t.Fatal("assemble succeeded for invalid input")
			}
		})
	}
}

func TestAssembleFailurePreservesLastCompleteDistribution(t *testing.T) {
	root := t.TempDir()
	outputRoot := filepath.Join(root, "analyzers")
	firstBuild := func(_ context.Context, request BuildRequest) error {
		if err := os.MkdirAll(filepath.Dir(request.OutputPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(request.OutputPath, []byte("first-complete"), 0o755)
	}
	if _, err := Assemble(context.Background(), AssembleOptions{
		RepositoryRoot: root,
		OutputRoot:     outputRoot,
		Platform:       "windows-amd64",
		BuildID:        "first",
		GoCommand:      "explicit-toolchain",
		CCCommand:      "explicit-cc",
		CXXCommand:     "explicit-cxx",
		AnalyzerIDs:    []string{"org.archview.go", "org.archview.python"},
		Build:          firstBuild,
	}); err != nil {
		t.Fatalf("first assemble: %v", err)
	}
	oldIndex := mustReadDistributionFile(t, outputRoot, "index.json")
	oldExecutable := mustReadDistributionFile(t, outputRoot, "org.archview.go/windows-amd64/analyzer.exe")

	failingBuild := func(_ context.Context, request BuildRequest) error {
		if err := os.MkdirAll(filepath.Dir(request.OutputPath), 0o755); err != nil {
			return err
		}
		if strings.HasSuffix(request.Entrypoint, "/python") {
			return errors.New("simulated analyzer compilation failure")
		}
		return os.WriteFile(request.OutputPath, []byte("second-partial"), 0o755)
	}
	if _, err := Assemble(context.Background(), AssembleOptions{
		RepositoryRoot: root,
		OutputRoot:     outputRoot,
		Platform:       "windows-amd64",
		BuildID:        "second",
		GoCommand:      "explicit-toolchain",
		CCCommand:      "explicit-cc",
		CXXCommand:     "explicit-cxx",
		AnalyzerIDs:    []string{"org.archview.go", "org.archview.python"},
		Build:          failingBuild,
	}); err == nil {
		t.Fatal("failed assemble unexpectedly succeeded")
	}

	if got := mustReadDistributionFile(t, outputRoot, "index.json"); string(got) != string(oldIndex) {
		t.Fatal("failed assemble replaced the last complete index")
	}
	if got := mustReadDistributionFile(t, outputRoot, "org.archview.go/windows-amd64/analyzer.exe"); string(got) != string(oldExecutable) {
		t.Fatal("failed assemble replaced a package in the last complete distribution")
	}
	entries, err := os.ReadDir(filepath.Dir(outputRoot))
	if err != nil {
		t.Fatalf("read distribution parent: %v", err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".analyzers.") {
			t.Fatalf("staging or backup directory remains after failed assemble: %s", entry.Name())
		}
	}
}

func TestValidateIndexRejectsUnsafePathsAndDigestFailures(t *testing.T) {
	root := t.TempDir()
	outputRoot := filepath.Join(root, "analyzers")
	index, err := Assemble(context.Background(), AssembleOptions{
		RepositoryRoot: root,
		OutputRoot:     outputRoot,
		Platform:       "windows-amd64",
		BuildID:        "validation-test",
		GoCommand:      "explicit-toolchain",
		CCCommand:      "explicit-cc",
		CXXCommand:     "explicit-cxx",
		AnalyzerIDs:    []string{"org.archview.go"},
		Build: func(_ context.Context, request BuildRequest) error {
			if err := os.MkdirAll(filepath.Dir(request.OutputPath), 0o755); err != nil {
				return err
			}
			return os.WriteFile(request.OutputPath, []byte("binary"), 0o755)
		},
	})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}

	unsafe := index
	unsafe.Packages = append([]AnalyzerPackage(nil), index.Packages...)
	unsafe.Packages[0].ExecutablePath = "../outside/analyzer.exe"
	if err := ValidateIndex(outputRoot, unsafe, []string{"org.archview.go"}, "windows-amd64"); err == nil {
		t.Fatal("unsafe executable path was accepted")
	}

	badDigest := index
	badDigest.Packages = append([]AnalyzerPackage(nil), index.Packages...)
	badDigest.Packages[0].ExecutableSHA256 = strings.Repeat("0", 64)
	if err := ValidateIndex(outputRoot, badDigest, []string{"org.archview.go"}, "windows-amd64"); err == nil {
		t.Fatal("incorrect executable digest was accepted")
	}

	if err := os.WriteFile(filepath.Join(outputRoot, "unexpected.txt"), []byte("unexpected"), 0o644); err != nil {
		t.Fatalf("write unexpected artifact: %v", err)
	}
	if err := ValidateIndex(outputRoot, index, []string{"org.archview.go"}, "windows-amd64"); err == nil {
		t.Fatal("unexpected distribution artifact was accepted")
	}
}

func mustReadDistributionFile(t *testing.T, root, relative string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		t.Fatalf("read %s: %v", relative, err)
	}
	return data
}

func sha256File(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
