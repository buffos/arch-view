package distribution

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssembleBuildsCompleteDefaultAnalyzerMatrix(t *testing.T) {
	parent := t.TempDir()
	repositoryRoot := filepath.Join(parent, "repository")
	if err := os.Mkdir(repositoryRoot, 0o755); err != nil {
		t.Fatalf("create repository: %v", err)
	}
	ids := DefaultAnalyzerIDs()
	for _, target := range SupportedPlatformTargets() {
		t.Run(target.Name, func(t *testing.T) {
			outputRoot := filepath.Join(parent, "analyzers-"+strings.ReplaceAll(target.Name, "-", "_"))
			index, err := Assemble(context.Background(), AssembleOptions{
				RepositoryRoot: repositoryRoot,
				OutputRoot:     outputRoot,
				Platform:       target.Name,
				BuildID:        "matrix-" + target.Name,
				GoCommand:      "matrix-go",
				CCCommand:      "matrix-cc",
				CXXCommand:     "matrix-cxx",
				AnalyzerIDs:    ids,
				Build: func(_ context.Context, request BuildRequest) error {
					if err := os.MkdirAll(filepath.Dir(request.OutputPath), 0o755); err != nil {
						return err
					}
					return os.WriteFile(request.OutputPath, []byte("compiled:"+request.Entrypoint+":"+request.GOOS+":"+request.GOARCH), 0o755)
				},
			})
			if err != nil {
				t.Fatalf("assemble %s: %v", target.Name, err)
			}
			if len(index.Packages) != len(ids) {
				t.Fatalf("package count = %d, want %d", len(index.Packages), len(ids))
			}
			if err := ValidateIndex(outputRoot, index, ids, target.Name); err != nil {
				t.Fatalf("validate %s matrix package: %v", target.Name, err)
			}
		})
	}
}

func TestAssembleIsByteStableForUnchangedInputs(t *testing.T) {
	parent := t.TempDir()
	repositoryRoot := filepath.Join(parent, "repository")
	if err := os.Mkdir(repositoryRoot, 0o755); err != nil {
		t.Fatalf("create repository: %v", err)
	}
	build := func(_ context.Context, request BuildRequest) error {
		if err := os.MkdirAll(filepath.Dir(request.OutputPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(request.OutputPath, []byte("stable:"+request.Entrypoint), 0o755)
	}
	options := AssembleOptions{
		RepositoryRoot: repositoryRoot,
		Platform:       "windows-amd64",
		BuildID:        "stable-build",
		GoCommand:      "stable-go",
		CCCommand:      "stable-cc",
		CXXCommand:     "stable-cxx",
		AnalyzerIDs:    []string{"org.archview.go", "org.archview.python"},
		Build:          build,
	}
	firstRoot := filepath.Join(parent, "first")
	secondRoot := filepath.Join(parent, "second")
	options.OutputRoot = firstRoot
	first, err := Assemble(context.Background(), options)
	if err != nil {
		t.Fatalf("first stable assembly: %v", err)
	}
	options.OutputRoot = secondRoot
	second, err := Assemble(context.Background(), options)
	if err != nil {
		t.Fatalf("second stable assembly: %v", err)
	}
	firstIndex, err := os.ReadFile(filepath.Join(firstRoot, "index.json"))
	if err != nil {
		t.Fatalf("read first index: %v", err)
	}
	secondIndex, err := os.ReadFile(filepath.Join(secondRoot, "index.json"))
	if err != nil {
		t.Fatalf("read second index: %v", err)
	}
	if !bytes.Equal(firstIndex, secondIndex) || len(first.Packages) != len(second.Packages) {
		t.Fatalf("stable indexes differ")
	}
	for index := range first.Packages {
		firstExecutable, err := os.ReadFile(filepath.Join(firstRoot, filepath.FromSlash(first.Packages[index].ExecutablePath)))
		if err != nil {
			t.Fatalf("read first executable: %v", err)
		}
		secondExecutable, err := os.ReadFile(filepath.Join(secondRoot, filepath.FromSlash(second.Packages[index].ExecutablePath)))
		if err != nil {
			t.Fatalf("read second executable: %v", err)
		}
		if !bytes.Equal(firstExecutable, secondExecutable) {
			t.Fatalf("executable %q is not byte-stable", first.Packages[index].LogicalAnalyzerID)
		}
	}
}

func TestAssembleReleasePublishesHostAndAnalyzerTree(t *testing.T) {
	repositoryRoot := t.TempDir()
	outputRoot := filepath.Join(repositoryRoot, "dist", "release")
	var requests []BuildRequest
	build := func(_ context.Context, request BuildRequest) error {
		requests = append(requests, request)
		if err := os.MkdirAll(filepath.Dir(request.OutputPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(request.OutputPath, []byte("compiled:"+request.Entrypoint), 0o755)
	}

	if err := AssembleRelease(context.Background(), ReleaseOptions{
		RepositoryRoot: repositoryRoot,
		OutputRoot:     outputRoot,
		Platform:       "windows-amd64",
		Version:        "0.1.0",
		Commit:         "abc123",
		BuildDate:      "2026-09-01T12:00:00Z",
		BuildID:        "release-test",
		GoCommand:      "explicit-go",
		CCCommand:      "explicit-cc",
		CXXCommand:     "explicit-cxx",
		AnalyzerIDs:    []string{"org.archview.go"},
		Build:          build,
	}); err != nil {
		t.Fatalf("assemble release: %v", err)
	}

	if got := string(mustReadDistributionFile(t, outputRoot, "arch-view.exe")); got != "compiled:./cmd/arch-view" {
		t.Fatalf("host content = %q", got)
	}
	index, err := ReadIndex(filepath.Join(outputRoot, "analyzers", "index.json"))
	if err != nil {
		t.Fatalf("read release index: %v", err)
	}
	if err := ValidateIndex(filepath.Join(outputRoot, "analyzers"), index, []string{"org.archview.go"}, "windows-amd64"); err != nil {
		t.Fatalf("validate release index: %v", err)
	}
	if got, want := len(requests), 2; got != want {
		t.Fatalf("build request count = %d, want %d", got, want)
	}
	for _, request := range requests {
		if request.GoCommand != "explicit-go" || request.CCCommand != "explicit-cc" || request.CXXCommand != "explicit-cxx" {
			t.Fatalf("toolchain = %q/%q/%q", request.GoCommand, request.CCCommand, request.CXXCommand)
		}
	}
}

func TestAssembleReleasePublishesVersionedManifestAndHostMetadata(t *testing.T) {
	repositoryRoot := t.TempDir()
	outputRoot := filepath.Join(repositoryRoot, "dist", "release")
	requests := make([]BuildRequest, 0, 2)
	build := func(_ context.Context, request BuildRequest) error {
		requests = append(requests, request)
		if err := os.MkdirAll(filepath.Dir(request.OutputPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(request.OutputPath, []byte("compiled:"+request.Entrypoint), 0o755)
	}

	if err := AssembleRelease(context.Background(), ReleaseOptions{
		RepositoryRoot: repositoryRoot,
		OutputRoot:     outputRoot,
		Platform:       "windows-amd64",
		Version:        "0.1.0",
		Commit:         "abc123",
		BuildDate:      "2026-09-01T12:00:00Z",
		BuildID:        "ci-42",
		GoCommand:      "explicit-go",
		CCCommand:      "explicit-cc",
		CXXCommand:     "explicit-cxx",
		AnalyzerIDs:    []string{"org.archview.go"},
		Build:          build,
	}); err != nil {
		t.Fatalf("assemble release: %v", err)
	}

	manifest, err := ReadReleaseManifest(outputRoot)
	if err != nil {
		t.Fatalf("read release manifest: %v", err)
	}
	if manifest.SchemaVersion != ReleaseManifestSchemaVersion || manifest.Application != "arch-view" || manifest.Version != "0.1.0" || manifest.Commit != "abc123" || manifest.BuildDate != "2026-09-01T12:00:00Z" || manifest.BuildID != "ci-42" || manifest.Platform != "windows-amd64" {
		t.Fatalf("release manifest = %#v", manifest)
	}
	if manifest.HostExecutable != "arch-view.exe" || manifest.AnalyzerIndexPath != "analyzers/index.json" {
		t.Fatalf("release manifest paths = %#v", manifest)
	}
	if err := ValidateReleaseManifest(outputRoot, manifest); err != nil {
		t.Fatalf("validate release manifest: %v", err)
	}
	if len(requests) != 2 {
		t.Fatalf("build requests = %d, want 2", len(requests))
	}
	if requests[0].Entrypoint != "./cmd/arch-view" {
		t.Fatalf("host entrypoint = %q", requests[0].Entrypoint)
	}
	if got, want := requests[0].LinkerFlags, "-X=github.com/buffo/arch-view/internal/version.Version=0.1.0 -X=github.com/buffo/arch-view/internal/version.Commit=abc123 -X=github.com/buffo/arch-view/internal/version.BuildDate=2026-09-01T12:00:00Z -X=github.com/buffo/arch-view/internal/version.BuildID=ci-42"; got != want {
		t.Fatalf("host linker flags = %q, want %q", got, want)
	}
	if requests[1].LinkerFlags != "" {
		t.Fatalf("analyzer linker flags = %q, want empty", requests[1].LinkerFlags)
	}
}

func TestAssembleReleaseRejectsInvalidVersionBeforeBuilding(t *testing.T) {
	repositoryRoot := t.TempDir()
	built := false
	err := AssembleRelease(context.Background(), ReleaseOptions{
		RepositoryRoot: repositoryRoot,
		OutputRoot:     filepath.Join(t.TempDir(), "release"),
		Platform:       "windows-amd64",
		Version:        "not-a-version",
		Commit:         "abc123",
		BuildDate:      "2026-09-01T12:00:00Z",
		BuildID:        "ci-42",
		GoCommand:      "explicit-go",
		CCCommand:      "explicit-cc",
		CXXCommand:     "explicit-cxx",
		AnalyzerIDs:    []string{"org.archview.go"},
		Build: func(context.Context, BuildRequest) error {
			built = true
			return nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "semantic-version") {
		t.Fatalf("invalid version error = %v", err)
	}
	if built {
		t.Fatal("invalid version started a build")
	}
}

func TestAssembleReleaseFailurePreservesLastCompleteRelease(t *testing.T) {
	repositoryRoot := t.TempDir()
	outputRoot := filepath.Join(repositoryRoot, "dist", "release")
	options := ReleaseOptions{
		RepositoryRoot: repositoryRoot,
		OutputRoot:     outputRoot,
		Platform:       "windows-amd64",
		BuildID:        "first",
		GoCommand:      "explicit-go",
		CCCommand:      "explicit-cc",
		CXXCommand:     "explicit-cxx",
		AnalyzerIDs:    []string{"org.archview.go"},
		Build: func(_ context.Context, request BuildRequest) error {
			if err := os.MkdirAll(filepath.Dir(request.OutputPath), 0o755); err != nil {
				return err
			}
			return os.WriteFile(request.OutputPath, []byte("first:"+request.Entrypoint), 0o755)
		},
	}
	if err := AssembleRelease(context.Background(), options); err != nil {
		t.Fatalf("first release: %v", err)
	}
	oldHost := mustReadDistributionFile(t, outputRoot, "arch-view.exe")
	oldIndex := mustReadDistributionFile(t, filepath.Join(outputRoot, "analyzers"), "index.json")

	options.BuildID = "second"
	options.Build = func(_ context.Context, request BuildRequest) error {
		if strings.HasPrefix(request.Entrypoint, "./cmd/analyzers/") {
			return errors.New("simulated analyzer build failure")
		}
		if err := os.MkdirAll(filepath.Dir(request.OutputPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(request.OutputPath, []byte("second-host"), 0o755)
	}
	if err := AssembleRelease(context.Background(), options); err == nil {
		t.Fatal("failed release unexpectedly succeeded")
	}
	if got := mustReadDistributionFile(t, outputRoot, "arch-view.exe"); string(got) != string(oldHost) {
		t.Fatal("failed release replaced the host application")
	}
	if got := mustReadDistributionFile(t, filepath.Join(outputRoot, "analyzers"), "index.json"); string(got) != string(oldIndex) {
		t.Fatal("failed release replaced the analyzer index")
	}
}

func TestAssemblyRejectsOutputThatContainsRepository(t *testing.T) {
	parent := t.TempDir()
	repositoryRoot := filepath.Join(parent, "repository")
	if err := os.Mkdir(repositoryRoot, 0o755); err != nil {
		t.Fatalf("create repository: %v", err)
	}
	marker := filepath.Join(repositoryRoot, "keep.txt")
	if err := os.WriteFile(marker, []byte("keep"), 0o644); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	_, err := Assemble(context.Background(), AssembleOptions{
		RepositoryRoot: repositoryRoot,
		OutputRoot:     parent,
		Platform:       "windows-amd64",
		BuildID:        "unsafe-output",
		GoCommand:      "explicit-go",
		CCCommand:      "explicit-cc",
		CXXCommand:     "explicit-cxx",
		AnalyzerIDs:    []string{"org.archview.go"},
		Build: func(context.Context, BuildRequest) error {
			return errors.New("build must not run")
		},
	})
	if err == nil {
		t.Fatal("assembly accepted an output root containing the repository")
	}
	if got := string(mustReadDistributionFile(t, repositoryRoot, "keep.txt")); got != "keep" {
		t.Fatalf("repository marker = %q", got)
	}
}

func TestAssemblyRefusesToReplaceUnrecognizedOutput(t *testing.T) {
	repositoryRoot := t.TempDir()
	outputRoot := filepath.Join(repositoryRoot, "dist", "analyzers")
	if err := os.MkdirAll(outputRoot, 0o755); err != nil {
		t.Fatalf("create output root: %v", err)
	}
	marker := filepath.Join(outputRoot, "keep.txt")
	if err := os.WriteFile(marker, []byte("keep"), 0o644); err != nil {
		t.Fatalf("write output marker: %v", err)
	}

	_, err := Assemble(context.Background(), AssembleOptions{
		RepositoryRoot: repositoryRoot,
		OutputRoot:     outputRoot,
		Platform:       "windows-amd64",
		BuildID:        "unsafe-replacement",
		GoCommand:      "explicit-go",
		CCCommand:      "explicit-cc",
		CXXCommand:     "explicit-cxx",
		AnalyzerIDs:    []string{"org.archview.go"},
		Build: func(context.Context, BuildRequest) error {
			return errors.New("build must not run")
		},
	})
	if err == nil {
		t.Fatal("assembly replaced an unrecognized output directory")
	}
	if got := string(mustReadDistributionFile(t, outputRoot, "keep.txt")); got != "keep" {
		t.Fatalf("output marker = %q", got)
	}
}

func TestPublishDirectoryRejectsOverlappingRoots(t *testing.T) {
	stagingRoot := t.TempDir()
	outputRoot := filepath.Join(stagingRoot, "output")
	if err := os.Mkdir(outputRoot, 0o755); err != nil {
		t.Fatalf("create nested output: %v", err)
	}
	if err := PublishDirectory(stagingRoot, outputRoot); err == nil {
		t.Fatal("publication accepted overlapping roots")
	}
}
