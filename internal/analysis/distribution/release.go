package distribution

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AssembleRelease builds the host and analyzer tree in one staging directory,
// then publishes the complete release only after every build succeeds.
func AssembleRelease(ctx context.Context, options ReleaseOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(options.GoCommand) == "" {
		return fmt.Errorf("go command is required")
	}
	if strings.TrimSpace(options.CCCommand) == "" || strings.TrimSpace(options.CXXCommand) == "" {
		return fmt.Errorf("c and C++ compiler commands are required")
	}
	repositoryRoot, err := absoluteExistingDirectory(options.RepositoryRoot, "repository root")
	if err != nil {
		return err
	}
	outputRoot, err := absoluteOutputRoot(options.OutputRoot)
	if err != nil {
		return err
	}
	if err := ensureOutputDoesNotContainRepository(repositoryRoot, outputRoot); err != nil {
		return err
	}
	target, err := PlatformTargetFor(options.Platform)
	if err != nil {
		return err
	}
	if err := validateReplaceableRelease(outputRoot); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outputRoot), 0o755); err != nil {
		return fmt.Errorf("create release parent: %w", err)
	}
	stagingRoot, err := os.MkdirTemp(filepath.Dir(outputRoot), "."+filepath.Base(outputRoot)+".release-staging-")
	if err != nil {
		return fmt.Errorf("create release staging root: %w", err)
	}
	stagingOwned := true
	defer func() {
		if stagingOwned {
			_ = os.RemoveAll(stagingRoot)
		}
	}()

	build := options.Build
	if build == nil {
		build = BuildEntrypoint
	}
	hostPath := filepath.Join(stagingRoot, "arch-view"+hostExecutableSuffix(target))
	if err := build(ctx, BuildRequest{
		RepositoryRoot: repositoryRoot,
		Entrypoint:     "./cmd/arch-view",
		OutputPath:     hostPath,
		GoCommand:      options.GoCommand,
		CCCommand:      options.CCCommand,
		CXXCommand:     options.CXXCommand,
		GOOS:           target.GOOS,
		GOARCH:         target.GOARCH,
		CGOEnabled:     "1",
	}); err != nil {
		return fmt.Errorf("build host application: %w", err)
	}
	if err := validateRegularFile(hostPath); err != nil {
		return fmt.Errorf("assembled host application: %w", err)
	}
	if _, err := Assemble(ctx, AssembleOptions{
		RepositoryRoot: repositoryRoot,
		OutputRoot:     filepath.Join(stagingRoot, "analyzers"),
		Platform:       target.Name,
		BuildID:        options.BuildID,
		GoCommand:      options.GoCommand,
		CCCommand:      options.CCCommand,
		CXXCommand:     options.CXXCommand,
		AnalyzerIDs:    options.AnalyzerIDs,
		Build:          build,
	}); err != nil {
		return fmt.Errorf("assemble release analyzers: %w", err)
	}
	if err := PublishDirectory(stagingRoot, outputRoot); err != nil {
		return fmt.Errorf("publish release: %w", err)
	}
	stagingOwned = false
	return nil
}

func validateReplaceableRelease(outputRoot string) error {
	if _, err := os.Lstat(outputRoot); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("stat existing release: %w", err)
	}
	hostCount := 0
	for _, name := range []string{"arch-view", "arch-view.exe"} {
		if err := validateRegularFile(filepath.Join(outputRoot, name)); err == nil {
			hostCount++
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("refuse to replace unrecognized release root %s: %w", outputRoot, err)
		}
	}
	if hostCount != 1 {
		return fmt.Errorf("refuse to replace unrecognized release root %s: expected exactly one host executable", outputRoot)
	}
	if err := validateReplaceableDistribution(filepath.Join(outputRoot, "analyzers")); err != nil {
		return fmt.Errorf("refuse to replace invalid release root %s: %w", outputRoot, err)
	}
	return nil
}

func hostExecutableSuffix(target PlatformTarget) string {
	if target.GOOS == "windows" {
		return ".exe"
	}
	return ""
}
