package distribution

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis/processprotocol"
)

// MarshalIndex encodes an index using the canonical human-readable JSON
// representation used for deterministic distribution output.
func MarshalIndex(index AnalyzerIndex) ([]byte, error) {
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal analyzer index: %w", err)
	}
	return append(data, '\n'), nil
}

// DecodeIndex decodes exactly one strict analyzer index JSON object.
func DecodeIndex(data []byte) (AnalyzerIndex, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var index AnalyzerIndex
	if err := decoder.Decode(&index); err != nil {
		return AnalyzerIndex{}, fmt.Errorf("decode analyzer index: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return AnalyzerIndex{}, fmt.Errorf("decode analyzer index: trailing JSON value")
		}
		return AnalyzerIndex{}, fmt.Errorf("decode analyzer index: trailing data: %w", err)
	}
	return index, nil
}

// ReadIndex reads and strictly decodes an index file.
func ReadIndex(path string) (AnalyzerIndex, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return AnalyzerIndex{}, fmt.Errorf("read analyzer index %s: %w", path, err)
	}
	return DecodeIndex(data)
}

// ValidateIndex verifies the index shape, canonical package tree, descriptors,
// file digests, and expected enabled-analyzer completeness.
func ValidateIndex(outputRoot string, index AnalyzerIndex, expectedIDs []string, expectedPlatform string) error {
	root, err := absoluteDirectory(outputRoot)
	if err != nil {
		return err
	}
	if index.SchemaVersion != IndexSchemaVersion {
		return fmt.Errorf("index schema version %q is unsupported", index.SchemaVersion)
	}
	if index.HostAPIVersion != processprotocol.ProtocolVersion {
		return fmt.Errorf("index host API version %q does not match %q", index.HostAPIVersion, processprotocol.ProtocolVersion)
	}
	if strings.TrimSpace(index.GeneratedBy) == "" {
		return fmt.Errorf("index generated_by is required")
	}
	if index.Packages == nil {
		return fmt.Errorf("index packages must be an array")
	}
	if len(index.Packages) == 0 {
		return fmt.Errorf("index must contain at least one analyzer package")
	}
	if err := validateIndexMetadata(index); err != nil {
		return err
	}

	targetName := ""
	if expectedPlatform != "" {
		target, targetErr := PlatformTargetFor(expectedPlatform)
		if targetErr != nil {
			return targetErr
		}
		targetName = target.Name
	}

	expectedSet := make(map[string]struct{}, len(expectedIDs))
	for _, rawID := range expectedIDs {
		id := strings.TrimSpace(rawID)
		if id == "" {
			return fmt.Errorf("expected analyzer ID cannot be blank")
		}
		if _, exists := expectedSet[id]; exists {
			return fmt.Errorf("expected analyzer ID %q is duplicated", id)
		}
		expectedSet[id] = struct{}{}
	}
	if len(expectedSet) > 0 {
		if targetName != "" && len(index.Packages) != len(expectedSet) {
			return fmt.Errorf("index contains %d packages, expected %d for %s", len(index.Packages), len(expectedSet), targetName)
		}
		if targetName == "" && len(index.Packages) < len(expectedSet) {
			return fmt.Errorf("index contains %d packages, expected at least %d", len(index.Packages), len(expectedSet))
		}
	}

	if !sort.SliceIsSorted(index.Packages, func(i, j int) bool { return packageLess(index.Packages[i], index.Packages[j]) }) {
		return fmt.Errorf("index packages are not sorted by logical analyzer ID and platform")
	}

	seenPackages := make(map[string]struct{}, len(index.Packages))
	seenIDs := make(map[string]struct{}, len(index.Packages))
	for i, packageEntry := range index.Packages {
		if packageEntry.LogicalAnalyzerID == "" {
			return fmt.Errorf("index package %d has no logical analyzer ID", i)
		}
		if len(expectedSet) > 0 {
			if _, ok := expectedSet[packageEntry.LogicalAnalyzerID]; !ok {
				return fmt.Errorf("index contains unexpected analyzer ID %q", packageEntry.LogicalAnalyzerID)
			}
		}
		seenIDs[packageEntry.LogicalAnalyzerID] = struct{}{}
		if packageEntry.AnalyzerVersion == "" || packageEntry.Language == "" || packageEntry.APIVersion == "" {
			return fmt.Errorf("index package %q has incomplete analyzer metadata", packageEntry.LogicalAnalyzerID)
		}
		platform, platformErr := PlatformTargetFor(packageEntry.Platform)
		if platformErr != nil {
			return fmt.Errorf("index package %q: %w", packageEntry.LogicalAnalyzerID, platformErr)
		}
		if targetName != "" && packageEntry.Platform != targetName {
			return fmt.Errorf("index package %q targets %q, expected %q", packageEntry.LogicalAnalyzerID, packageEntry.Platform, targetName)
		}
		packageKey := packageEntry.LogicalAnalyzerID + "\x00" + packageEntry.Platform
		if _, exists := seenPackages[packageKey]; exists {
			return fmt.Errorf("index contains duplicate package %q", packageKey)
		}
		seenPackages[packageKey] = struct{}{}

		if err := validateDigest(packageEntry.ExecutableSHA256, "executable", packageEntry.LogicalAnalyzerID); err != nil {
			return err
		}
		if err := validateDigest(packageEntry.DescriptorSHA256, "descriptor", packageEntry.LogicalAnalyzerID); err != nil {
			return err
		}
		if err := validatePackagePaths(packageEntry, platform); err != nil {
			return err
		}

		executablePath, err := resolveSafePath(root, packageEntry.ExecutablePath)
		if err != nil {
			return fmt.Errorf("index package %q executable path: %w", packageEntry.LogicalAnalyzerID, err)
		}
		descriptorPath, err := resolveSafePath(root, packageEntry.DescriptorPath)
		if err != nil {
			return fmt.Errorf("index package %q descriptor path: %w", packageEntry.LogicalAnalyzerID, err)
		}
		if err := validateRegularFile(executablePath); err != nil {
			return fmt.Errorf("index package %q executable: %w", packageEntry.LogicalAnalyzerID, err)
		}
		if err := validateRegularFile(descriptorPath); err != nil {
			return fmt.Errorf("index package %q descriptor: %w", packageEntry.LogicalAnalyzerID, err)
		}
		if err := validatePackageShape(filepath.Dir(executablePath), filepath.Base(executablePath)); err != nil {
			return fmt.Errorf("index package %q: %w", packageEntry.LogicalAnalyzerID, err)
		}

		if digest, digestErr := fileSHA256(executablePath); digestErr != nil {
			return fmt.Errorf("hash executable for %q: %w", packageEntry.LogicalAnalyzerID, digestErr)
		} else if digest != packageEntry.ExecutableSHA256 {
			return fmt.Errorf("executable digest mismatch for %q", packageEntry.LogicalAnalyzerID)
		}
		descriptorData, readErr := os.ReadFile(descriptorPath)
		if readErr != nil {
			return fmt.Errorf("read descriptor for %q: %w", packageEntry.LogicalAnalyzerID, readErr)
		}
		if digest := sha256Bytes(descriptorData); digest != packageEntry.DescriptorSHA256 {
			return fmt.Errorf("descriptor digest mismatch for %q", packageEntry.LogicalAnalyzerID)
		}
		descriptor, decodeErr := processprotocol.DecodeDescriptor(descriptorData)
		if decodeErr != nil {
			return fmt.Errorf("decode descriptor for %q: %w", packageEntry.LogicalAnalyzerID, decodeErr)
		}
		if descriptor.Manifest.ID != packageEntry.LogicalAnalyzerID ||
			descriptor.Manifest.Version != packageEntry.AnalyzerVersion ||
			descriptor.Manifest.Language != packageEntry.Language ||
			descriptor.Manifest.APIVersion != packageEntry.APIVersion {
			return fmt.Errorf("descriptor manifest does not match index for %q", packageEntry.LogicalAnalyzerID)
		}
		if descriptor.Command != filepath.Base(executablePath) {
			return fmt.Errorf("descriptor command %q does not point to executable %q for %q", descriptor.Command, filepath.Base(executablePath), packageEntry.LogicalAnalyzerID)
		}
	}

	if len(expectedSet) > 0 {
		for id := range expectedSet {
			if _, ok := seenIDs[id]; !ok {
				return fmt.Errorf("index is missing enabled analyzer %q", id)
			}
		}
	}
	if err := validateDistributionShape(root, index.Packages); err != nil {
		return err
	}
	return nil
}

func validateDistributionShape(root string, packages []AnalyzerPackage) error {
	expectedPlatforms := make(map[string]map[string]struct{})
	for _, packageEntry := range packages {
		platforms := expectedPlatforms[packageEntry.LogicalAnalyzerID]
		if platforms == nil {
			platforms = make(map[string]struct{})
			expectedPlatforms[packageEntry.LogicalAnalyzerID] = platforms
		}
		platforms[packageEntry.Platform] = struct{}{}
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return fmt.Errorf("read distribution root: %w", err)
	}
	if len(entries) != len(expectedPlatforms)+1 {
		return fmt.Errorf("distribution root must contain only index.json and indexed analyzer directories")
	}
	seenIndex := false
	for _, entry := range entries {
		if entry.Name() == "index.json" {
			if err := validateRegularFile(filepath.Join(root, entry.Name())); err != nil {
				return fmt.Errorf("distribution index: %w", err)
			}
			seenIndex = true
			continue
		}
		platforms, ok := expectedPlatforms[entry.Name()]
		if !ok {
			return fmt.Errorf("distribution root contains unexpected entry %q", entry.Name())
		}
		analyzerRoot := filepath.Join(root, entry.Name())
		info, err := os.Lstat(analyzerRoot)
		if err != nil {
			return fmt.Errorf("stat analyzer directory %q: %w", entry.Name(), err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("analyzer entry %q is not a regular directory", entry.Name())
		}
		platformEntries, err := os.ReadDir(analyzerRoot)
		if err != nil {
			return fmt.Errorf("read analyzer directory %q: %w", entry.Name(), err)
		}
		if len(platformEntries) != len(platforms) {
			return fmt.Errorf("analyzer directory %q does not match indexed platforms", entry.Name())
		}
		for _, platformEntry := range platformEntries {
			if _, ok := platforms[platformEntry.Name()]; !ok {
				return fmt.Errorf("analyzer directory %q contains unexpected platform %q", entry.Name(), platformEntry.Name())
			}
			platformPath := filepath.Join(analyzerRoot, platformEntry.Name())
			platformInfo, err := os.Lstat(platformPath)
			if err != nil {
				return fmt.Errorf("stat platform directory %q: %w", platformEntry.Name(), err)
			}
			if platformInfo.Mode()&os.ModeSymlink != 0 || !platformInfo.IsDir() {
				return fmt.Errorf("platform entry %q is not a regular directory", platformEntry.Name())
			}
		}
	}
	if !seenIndex {
		return fmt.Errorf("distribution root is missing index.json")
	}
	return nil
}

func packageLess(left, right AnalyzerPackage) bool {
	if left.LogicalAnalyzerID != right.LogicalAnalyzerID {
		return left.LogicalAnalyzerID < right.LogicalAnalyzerID
	}
	if left.Platform != right.Platform {
		return left.Platform < right.Platform
	}
	return left.ExecutablePath < right.ExecutablePath
}

func validatePackagePaths(packageEntry AnalyzerPackage, target PlatformTarget) error {
	packageDir := path.Join(packageEntry.LogicalAnalyzerID, packageEntry.Platform)
	if packageEntry.ExecutablePath != path.Join(packageDir, target.ExecutableName) {
		return fmt.Errorf("executable path %q is not the canonical package path", packageEntry.ExecutablePath)
	}
	if packageEntry.DescriptorPath != path.Join(packageDir, descriptorFileName) {
		return fmt.Errorf("descriptor path %q is not the canonical package path", packageEntry.DescriptorPath)
	}
	return nil
}

func validateDigest(digest, label, analyzerID string) error {
	if len(digest) != sha256.Size*2 || strings.ToLower(digest) != digest {
		return fmt.Errorf("%s digest for %q must be lowercase SHA-256", label, analyzerID)
	}
	if _, err := hex.DecodeString(digest); err != nil {
		return fmt.Errorf("%s digest for %q is not hexadecimal: %w", label, analyzerID, err)
	}
	return nil
}

func validatePackageShape(packageDir, executableName string) error {
	entries, err := os.ReadDir(packageDir)
	if err != nil {
		return fmt.Errorf("read package directory: %w", err)
	}
	if len(entries) != 2 {
		return fmt.Errorf("package directory must contain exactly executable and descriptor")
	}
	seenExecutable := false
	seenDescriptor := false
	for _, entry := range entries {
		switch entry.Name() {
		case executableName:
			seenExecutable = true
		case descriptorFileName:
			seenDescriptor = true
		default:
			return fmt.Errorf("package directory contains unexpected entry %q", entry.Name())
		}
	}
	if !seenExecutable || !seenDescriptor {
		return fmt.Errorf("package directory is missing executable or descriptor")
	}
	return nil
}

func resolveSafePath(root, relative string) (string, error) {
	return resolveTrustedPath(root, relative)
}

func isSafeRelativePath(value string) bool {
	if value == "" || strings.ContainsAny(value, "\\\x00\r\n") || strings.Contains(value, ":") || path.IsAbs(value) || path.Clean(value) != value {
		return false
	}
	return value != "." && value != ".."
}

func absoluteDirectory(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("distribution root is required")
	}
	absolute, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("resolve distribution root: %w", err)
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return "", fmt.Errorf("stat distribution root %s: %w", absolute, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("distribution root %s is not a directory", absolute)
	}
	return absolute, nil
}

func validateRegularFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", path)
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		_ = file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func sha256Bytes(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
