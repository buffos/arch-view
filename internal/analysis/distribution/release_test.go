package distribution

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
