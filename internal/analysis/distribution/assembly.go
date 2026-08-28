package distribution

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/buffo/arch-view/internal/analysis/processprotocol"
)

// Assemble builds and atomically publishes one complete analyzer distribution.
// No output under the final root is changed until every package, descriptor,
// digest, and index validation has succeeded.
func Assemble(ctx context.Context, options AssembleOptions) (AnalyzerIndex, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(options.RepositoryRoot) == "" {
		return AnalyzerIndex{}, fmt.Errorf("repository root is required")
	}
	if strings.TrimSpace(options.BuildID) == "" || strings.TrimSpace(options.BuildID) != options.BuildID {
		return AnalyzerIndex{}, fmt.Errorf("build ID must be non-empty normalized text")
	}
	if strings.ContainsAny(options.BuildID, "\x00\r\n") {
		return AnalyzerIndex{}, fmt.Errorf("build ID contains forbidden control characters")
	}
	if strings.TrimSpace(options.GoCommand) == "" {
		return AnalyzerIndex{}, fmt.Errorf("go command is required")
	}
	if strings.TrimSpace(options.CCCommand) == "" || strings.TrimSpace(options.CXXCommand) == "" {
		return AnalyzerIndex{}, fmt.Errorf("c and C++ compiler commands are required")
	}
	repositoryRoot, err := absoluteExistingDirectory(options.RepositoryRoot, "repository root")
	if err != nil {
		return AnalyzerIndex{}, err
	}
	outputRoot, err := absoluteOutputRoot(options.OutputRoot)
	if err != nil {
		return AnalyzerIndex{}, err
	}
	if err := ensureOutputDoesNotContainRepository(repositoryRoot, outputRoot); err != nil {
		return AnalyzerIndex{}, err
	}
	if err := validateReplaceableDistribution(outputRoot); err != nil {
		return AnalyzerIndex{}, err
	}
	target, err := PlatformTargetFor(options.Platform)
	if err != nil {
		return AnalyzerIndex{}, err
	}
	specs, err := resolveAnalyzerSpecs(options.AnalyzerIDs)
	if err != nil {
		return AnalyzerIndex{}, err
	}
	if len(specs) == 0 {
		return AnalyzerIndex{}, fmt.Errorf("at least one analyzer must be enabled")
	}

	if err := os.MkdirAll(filepath.Dir(outputRoot), 0o755); err != nil {
		return AnalyzerIndex{}, fmt.Errorf("create distribution parent: %w", err)
	}
	stagingRoot, err := os.MkdirTemp(filepath.Dir(outputRoot), "."+filepath.Base(outputRoot)+".staging-")
	if err != nil {
		return AnalyzerIndex{}, fmt.Errorf("create distribution staging root: %w", err)
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
	packages := make([]AnalyzerPackage, 0, len(specs))
	for _, spec := range specs {
		packageDir := filepath.Join(stagingRoot, filepath.FromSlash(spec.ID), target.Name)
		if err := os.MkdirAll(packageDir, 0o755); err != nil {
			return AnalyzerIndex{}, fmt.Errorf("create package directory for %q: %w", spec.ID, err)
		}
		executablePath := filepath.Join(packageDir, target.ExecutableName)
		request := BuildRequest{
			RepositoryRoot: repositoryRoot,
			Entrypoint:     spec.Entrypoint,
			OutputPath:     executablePath,
			GoCommand:      options.GoCommand,
			CCCommand:      options.CCCommand,
			CXXCommand:     options.CXXCommand,
			GOOS:           target.GOOS,
			GOARCH:         target.GOARCH,
			CGOEnabled:     "1",
		}
		if err := build(ctx, request); err != nil {
			return AnalyzerIndex{}, fmt.Errorf("assemble analyzer %q: %w", spec.ID, err)
		}
		if err := validateRegularFile(executablePath); err != nil {
			return AnalyzerIndex{}, fmt.Errorf("assembled executable for %q: %w", spec.ID, err)
		}

		descriptor := processprotocol.Descriptor{
			SchemaVersion: processprotocol.DescriptorSchemaVersion,
			Manifest:      spec.Manifest,
			Command:       target.ExecutableName,
		}
		descriptorData, err := json.MarshalIndent(descriptor, "", "  ")
		if err != nil {
			return AnalyzerIndex{}, fmt.Errorf("marshal descriptor for %q: %w", spec.ID, err)
		}
		descriptorData = append(descriptorData, '\n')
		if err := processprotocol.ValidateDescriptor(descriptor); err != nil {
			return AnalyzerIndex{}, fmt.Errorf("validate descriptor for %q: %w", spec.ID, err)
		}
		descriptorPath := filepath.Join(packageDir, descriptorFileName)
		if err := os.WriteFile(descriptorPath, descriptorData, 0o644); err != nil {
			return AnalyzerIndex{}, fmt.Errorf("write descriptor for %q: %w", spec.ID, err)
		}
		if err := validateRegularFile(descriptorPath); err != nil {
			return AnalyzerIndex{}, fmt.Errorf("assembled descriptor for %q: %w", spec.ID, err)
		}
		if err := validatePackageShape(packageDir, target.ExecutableName); err != nil {
			return AnalyzerIndex{}, fmt.Errorf("assembled package for %q: %w", spec.ID, err)
		}
		executableDigest, err := fileSHA256(executablePath)
		if err != nil {
			return AnalyzerIndex{}, fmt.Errorf("hash executable for %q: %w", spec.ID, err)
		}
		descriptorDigest, err := fileSHA256(descriptorPath)
		if err != nil {
			return AnalyzerIndex{}, fmt.Errorf("hash descriptor for %q: %w", spec.ID, err)
		}

		packages = append(packages, AnalyzerPackage{
			LogicalAnalyzerID: spec.Manifest.ID,
			AnalyzerVersion:   spec.Manifest.Version,
			Language:          spec.Manifest.Language,
			APIVersion:        spec.Manifest.APIVersion,
			Platform:          target.Name,
			ExecutablePath:    filepath.ToSlash(filepath.Join(spec.ID, target.Name, target.ExecutableName)),
			DescriptorPath:    filepath.ToSlash(filepath.Join(spec.ID, target.Name, descriptorFileName)),
			ExecutableSHA256:  executableDigest,
			DescriptorSHA256:  descriptorDigest,
		})
	}

	index := AnalyzerIndex{
		SchemaVersion:  IndexSchemaVersion,
		HostAPIVersion: processprotocol.ProtocolVersion,
		GeneratedBy:    "arch-view-build-" + options.BuildID,
		Packages:       packages,
	}
	indexData, err := MarshalIndex(index)
	if err != nil {
		return AnalyzerIndex{}, err
	}
	indexPath := filepath.Join(stagingRoot, "index.json")
	if err := os.WriteFile(indexPath, indexData, 0o644); err != nil {
		return AnalyzerIndex{}, fmt.Errorf("write analyzer index: %w", err)
	}
	decodedIndex, err := ReadIndex(indexPath)
	if err != nil {
		return AnalyzerIndex{}, err
	}
	if err := ValidateIndex(stagingRoot, decodedIndex, analyzerIDs(specs), target.Name); err != nil {
		return AnalyzerIndex{}, fmt.Errorf("validate written analyzer index: %w", err)
	}

	if err := PublishDirectory(stagingRoot, outputRoot); err != nil {
		return AnalyzerIndex{}, fmt.Errorf("publish analyzer distribution: %w", err)
	}
	stagingOwned = false
	return decodedIndex, nil
}

func validateReplaceableDistribution(outputRoot string) error {
	if _, err := os.Lstat(outputRoot); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("stat existing distribution: %w", err)
	}
	index, err := ReadIndex(filepath.Join(outputRoot, "index.json"))
	if err != nil {
		return fmt.Errorf("refuse to replace unrecognized distribution root %s: %w", outputRoot, err)
	}
	if err := ValidateIndex(outputRoot, index, nil, ""); err != nil {
		return fmt.Errorf("refuse to replace invalid distribution root %s: %w", outputRoot, err)
	}
	return nil
}

func analyzerIDs(specs []AnalyzerSpec) []string {
	ids := make([]string, 0, len(specs))
	for _, spec := range specs {
		ids = append(ids, spec.ID)
	}
	return ids
}

func absoluteExistingDirectory(value, label string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s is required", label)
	}
	absolute, err := filepath.Abs(filepath.Clean(value))
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", label, err)
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return "", fmt.Errorf("stat %s %s: %w", label, absolute, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s %s is not a directory", label, absolute)
	}
	return absolute, nil
}

func absoluteOutputRoot(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("output root is required")
	}
	absolute, err := filepath.Abs(filepath.Clean(value))
	if err != nil {
		return "", fmt.Errorf("resolve output root: %w", err)
	}
	if info, statErr := os.Lstat(absolute); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return "", fmt.Errorf("output root %s must be a directory when it exists", absolute)
		}
	} else if !os.IsNotExist(statErr) {
		return "", fmt.Errorf("stat output root %s: %w", absolute, statErr)
	}
	return absolute, nil
}
