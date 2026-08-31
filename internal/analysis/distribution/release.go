package distribution

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	appversion "github.com/buffo/arch-view/internal/version"
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
	metadata, err := releaseMetadata(options)
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
		LinkerFlags:    metadata.LinkerFlags(),
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
	if err := writeReleaseManifest(stagingRoot, target, metadata, hostPath); err != nil {
		return fmt.Errorf("write release manifest: %w", err)
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
	manifest, err := ReadReleaseManifest(outputRoot)
	if err != nil {
		return fmt.Errorf("refuse to replace unrecognized release root %s: %w", outputRoot, err)
	}
	if err := ValidateReleaseManifest(outputRoot, manifest); err != nil {
		return fmt.Errorf("refuse to replace invalid release root %s: %w", outputRoot, err)
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

func releaseMetadata(options ReleaseOptions) (appversion.Info, error) {
	if strings.TrimSpace(options.BuildID) == "" || strings.TrimSpace(options.BuildID) != options.BuildID {
		return appversion.Info{}, fmt.Errorf("build ID must be non-empty normalized text")
	}
	metadata := appversion.Current()
	if options.Version != "" {
		metadata.Version = options.Version
	}
	if options.Commit != "" {
		metadata.Commit = options.Commit
	}
	if options.BuildDate != "" {
		metadata.BuildDate = options.BuildDate
	}
	metadata.BuildID = options.BuildID
	if err := metadata.Validate(); err != nil {
		return appversion.Info{}, err
	}
	return metadata, nil
}

func writeReleaseManifest(stagingRoot string, target PlatformTarget, metadata appversion.Info, hostPath string) error {
	hostDigest, err := fileSHA256(hostPath)
	if err != nil {
		return fmt.Errorf("hash host executable: %w", err)
	}
	indexPath := filepath.Join(stagingRoot, "analyzers", "index.json")
	indexDigest, err := fileSHA256(indexPath)
	if err != nil {
		return fmt.Errorf("hash analyzer index: %w", err)
	}
	manifest := ReleaseManifest{
		SchemaVersion:       ReleaseManifestSchemaVersion,
		Application:         metadata.Application,
		Version:             metadata.Version,
		Commit:              metadata.Commit,
		BuildDate:           metadata.BuildDate,
		BuildID:             metadata.BuildID,
		Platform:            target.Name,
		HostExecutable:      filepath.Base(hostPath),
		HostSHA256:          hostDigest,
		AnalyzerIndexPath:   filepath.ToSlash(filepath.Join("analyzers", "index.json")),
		AnalyzerIndexSHA256: indexDigest,
	}
	data, err := MarshalReleaseManifest(manifest)
	if err != nil {
		return err
	}
	manifestPath := filepath.Join(stagingRoot, releaseManifestFileName)
	if err := os.WriteFile(manifestPath, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", releaseManifestFileName, err)
	}
	decoded, err := ReadReleaseManifest(stagingRoot)
	if err != nil {
		return err
	}
	return ValidateReleaseManifest(stagingRoot, decoded)
}

func hostExecutableSuffix(target PlatformTarget) string {
	if target.GOOS == "windows" {
		return ".exe"
	}
	return ""
}
