package distribution

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/processanalyzer"
	"github.com/buffo/arch-view/internal/analysis/processprotocol"
)

const indexFileName = "index.json"

var (
	packageIDPattern         = regexp.MustCompile(`^[a-z0-9]+(?:[.-][a-z0-9]+)*$`)
	packageVersionPattern    = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$`)
	packageLanguagePattern   = regexp.MustCompile(`^[a-z][a-z0-9+.-]*$`)
	packageAPIVersionPattern = regexp.MustCompile(`^arch-view\.analyzer/v([0-9]+)(?:\.([0-9]+))?$`)
)

var errPackagePathUnsafe = errors.New("package path is unsafe")

// PackageStatus is the safe query status for one indexed package.
type PackageStatus string

const (
	PackageAvailable   PackageStatus = "available"
	PackageUnavailable PackageStatus = "unavailable"
	PackageRejected    PackageStatus = "rejected"
)

// PackageAvailability is a listing-safe projection of one index entry. It
// contains no live process or registry state and can therefore be used by
// startup/listing paths without launching an analyzer.
type PackageAvailability struct {
	AnalyzerPackage
	Status       PackageStatus      `json:"status"`
	ErrorCode    analysis.ErrorCode `json:"error_code,omitempty"`
	ErrorMessage string             `json:"error_message,omitempty"`
}

// VerifiedPackage is the only package value accepted by the process adapter.
// Its paths and descriptor have passed index, filesystem, digest, and
// manifest checks.
type VerifiedPackage struct {
	Root           string
	Index          AnalyzerPackage
	Descriptor     processprotocol.Descriptor
	ExecutablePath string
	DescriptorPath string
}

// PackagedAnalyzer adds the package launch failure contract to the existing
// process adapter. Verification errors happen before this wrapper is created;
// invocation errors are mapped to stable package/runtime outcomes and are
// never converted into an in-process fallback.
type PackagedAnalyzer struct {
	catalog  *Catalog
	entry    AnalyzerPackage
	manifest analysis.Manifest
	config   processanalyzer.Config
}

var _ analysis.Analyzer = (*PackagedAnalyzer)(nil)

// Catalog is an immutable, application-managed analyzer index loaded from one
// installation root. Loading a catalog validates the index itself; individual
// package files are verified when queried or selected so a listing can report
// per-package rejection reasons without making the whole catalog unusable.
type Catalog struct {
	root  string
	index AnalyzerIndex
}

// LoadCatalog reads only root/index.json and validates its schema and package
// metadata. It never reads a target repository and never starts a process.
func LoadCatalog(root string) (*Catalog, error) {
	absolute, err := absoluteDirectory(root)
	if err != nil {
		return nil, indexError(err)
	}
	indexPath, err := resolveTrustedPath(absolute, indexFileName)
	if err != nil {
		return nil, indexError(err)
	}
	if err := validateRegularFile(indexPath); err != nil {
		return nil, indexError(err)
	}
	index, err := ReadIndex(indexPath)
	if err != nil {
		return nil, indexError(err)
	}
	if err := validateIndexMetadata(index); err != nil {
		return nil, err
	}
	return &Catalog{root: absolute, index: index}, nil
}

// ReadCatalog is a descriptive alias for LoadCatalog.
func ReadCatalog(root string) (*Catalog, error) {
	return LoadCatalog(root)
}

// ListAvailableAnalyzers loads an application-managed catalog and returns a
// deterministic, process-free availability report for every index entry.
func ListAvailableAnalyzers(root string) ([]PackageAvailability, error) {
	catalog, err := LoadCatalog(root)
	if err != nil {
		return nil, err
	}
	return catalog.Availability(), nil
}

// Root returns the normalized application-managed analyzer root.
func (c *Catalog) Root() string {
	if c == nil {
		return ""
	}
	return c.root
}

// Index returns a defensive copy of the loaded index.
func (c *Catalog) Index() AnalyzerIndex {
	if c == nil {
		return AnalyzerIndex{}
	}
	index := c.index
	index.Packages = append([]AnalyzerPackage(nil), c.index.Packages...)
	return index
}

// Packages returns the immutable index entries in their canonical order.
func (c *Catalog) Packages() []AnalyzerPackage {
	if c == nil {
		return nil
	}
	return append([]AnalyzerPackage(nil), c.index.Packages...)
}

// Availability verifies every package without starting an executable. A
// missing package is unavailable; a present but unsafe, tampered, malformed,
// or incompatible package is rejected with its stable reason.
func (c *Catalog) Availability() []PackageAvailability {
	if c == nil {
		return nil
	}
	result := make([]PackageAvailability, 0, len(c.index.Packages))
	for _, entry := range c.index.Packages {
		availability := PackageAvailability{
			AnalyzerPackage: entry,
			Status:          PackageAvailable,
		}
		if _, err := c.VerifyPackage(entry); err != nil {
			availability.Status = PackageRejected
			availability.ErrorCode = analysis.ErrorCodeOf(err)
			availability.ErrorMessage = hostErrorMessage(err)
			if availability.ErrorCode == analysis.ErrAnalyzerPackageNotFound {
				availability.Status = PackageUnavailable
			}
		}
		result = append(result, availability)
	}
	return result
}

// SelectPackage selects one exact logical ID/platform pair. It never chooses
// a nearest or cross-platform package.
func (c *Catalog) SelectPackage(logicalAnalyzerID, platform string) (VerifiedPackage, error) {
	if c == nil {
		return VerifiedPackage{}, indexError(errors.New("analyzer catalog is nil"))
	}
	target, err := PlatformTargetFor(platform)
	if err != nil {
		return VerifiedPackage{}, analysis.NewHostError(analysis.ErrAnalyzerPlatformUnsupported, "requested analyzer platform is unsupported", map[string]any{
			"platform": platform,
		})
	}
	for _, entry := range c.index.Packages {
		if entry.LogicalAnalyzerID == logicalAnalyzerID && entry.Platform == target.Name {
			return c.VerifyPackage(entry)
		}
	}
	return VerifiedPackage{}, analysis.NewHostError(analysis.ErrAnalyzerPackageNotFound, "no packaged analyzer is available for the requested logical analyzer and platform", map[string]any{
		"analyzer_id": logicalAnalyzerID,
		"platform":    target.Name,
	})
}

// SelectHostPackage selects a package for the current exact host platform.
func (c *Catalog) SelectHostPackage(logicalAnalyzerID string) (VerifiedPackage, error) {
	return c.SelectPackage(logicalAnalyzerID, HostPlatform())
}

// NewAnalyzer verifies and adapts one package for the existing process host.
// The returned adapter has not started a process; verification happens before
// any later Detect or Analyze invocation.
func (c *Catalog) NewAnalyzer(logicalAnalyzerID, platform string, config processanalyzer.Config) (analysis.Analyzer, error) {
	return c.NewPackagedAnalyzer(logicalAnalyzerID, platform, config)
}

// NewPackagedAnalyzer returns a process-backed analyzer with packaged runtime
// failure mapping. It performs the same pre-launch verification as NewAnalyzer.
func (c *Catalog) NewPackagedAnalyzer(logicalAnalyzerID, platform string, config processanalyzer.Config) (analysis.Analyzer, error) {
	packageValue, err := c.SelectPackage(logicalAnalyzerID, platform)
	if err != nil {
		return nil, err
	}
	if _, err := newProcessAnalyzer(packageValue, config); err != nil {
		return nil, packageFailure(analysis.ErrAnalyzerPackageManifestMismatch, "packaged analyzer descriptor could not be adapted", packageValue.Index, "descriptor", err)
	}
	manifest := cloneCatalogManifest(packageValue.Descriptor.Manifest)
	manifest.RuntimeIdentity = packageValue.Index.ExecutableSHA256 + ":" + packageValue.Index.DescriptorSHA256
	return &PackagedAnalyzer{
		catalog:  c,
		entry:    packageValue.Index,
		manifest: manifest,
		config:   config,
	}, nil
}

func (a *PackagedAnalyzer) Manifest() analysis.Manifest {
	if a == nil {
		return analysis.Manifest{}
	}
	return cloneCatalogManifest(a.manifest)
}

func (a *PackagedAnalyzer) Detect(ctx context.Context, request analysis.DetectRequest) (analysis.DetectionCandidate, error) {
	inner, err := a.verifiedAnalyzer()
	if err != nil {
		return analysis.DetectionCandidate{}, err
	}
	candidate, err := inner.Detect(ctx, request)
	if err != nil {
		return candidate, a.mapRuntimeError(err)
	}
	return candidate, nil
}

func (a *PackagedAnalyzer) Analyze(ctx context.Context, request analysis.AnalyzeRequest) (analysis.AnalysisResult, error) {
	inner, err := a.verifiedAnalyzer()
	if err != nil {
		return analysis.AnalysisResult{}, err
	}
	result, err := inner.Analyze(ctx, request)
	if err != nil {
		return result, a.mapRuntimeError(err)
	}
	return result, nil
}

func (a *PackagedAnalyzer) verifiedAnalyzer() (*processanalyzer.Analyzer, error) {
	if a == nil || a.catalog == nil {
		return nil, packageFailure(analysis.ErrAnalyzerPackageLaunchFailed, "packaged analyzer could not be launched", packagedEntry(a), "launch", nil)
	}
	packageValue, err := a.catalog.SelectPackage(a.entry.LogicalAnalyzerID, a.entry.Platform)
	if err != nil {
		return nil, err
	}
	analyzer, err := newProcessAnalyzer(packageValue, a.config)
	if err != nil {
		return nil, packageFailure(analysis.ErrAnalyzerPackageManifestMismatch, "packaged analyzer descriptor could not be adapted", packageValue.Index, "descriptor", err)
	}
	return analyzer, nil
}

func newProcessAnalyzer(packageValue VerifiedPackage, config processanalyzer.Config) (*processanalyzer.Analyzer, error) {
	return processanalyzer.NewWithBaseDirectory(packageValue.Descriptor, filepath.Dir(packageValue.DescriptorPath), config)
}

func (a *PackagedAnalyzer) mapRuntimeError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || analysis.ErrorCodeOf(err) == analysis.ErrCancelled {
		return err
	}
	var protocolErr *processprotocol.ProtocolError
	if errors.As(err, &protocolErr) && protocolErr.FrameType == processprotocol.FrameHello {
		if protocolErr.Kind == processprotocol.ErrorUnsupportedProtocol {
			return packageFailure(analysis.ErrAnalyzerPackageAPIIncompatible, "packaged analyzer hello API is incompatible", packagedEntry(a), "hello_api_version", err)
		}
		return packageFailure(analysis.ErrAnalyzerPackageManifestMismatch, "packaged analyzer hello manifest disagrees with the package", packagedEntry(a), "hello_manifest", err)
	}
	return packageFailure(analysis.ErrAnalyzerPackageLaunchFailed, "packaged analyzer operation failed", packagedEntry(a), "launch", err)
}

func packagedEntry(analyzer *PackagedAnalyzer) AnalyzerPackage {
	if analyzer == nil {
		return AnalyzerPackage{}
	}
	return analyzer.entry
}

// VerifyAnalyzerPackage verifies one index entry against the package files at
// root. It is a free-function form for callers that already have an index
// entry and is intentionally process-free.
func VerifyAnalyzerPackage(root string, entry AnalyzerPackage) (VerifiedPackage, error) {
	catalog, err := LoadCatalog(root)
	if err != nil {
		return VerifiedPackage{}, err
	}
	return catalog.VerifyPackage(entry)
}

// VerifyPackage validates safe paths, regular files, both SHA-256 digests,
// descriptor manifest compatibility, and the descriptor's executable command.
func (c *Catalog) VerifyPackage(entry AnalyzerPackage) (VerifiedPackage, error) {
	if c == nil {
		return VerifiedPackage{}, indexError(errors.New("analyzer catalog is nil"))
	}
	if !c.contains(entry) {
		return VerifiedPackage{}, packageFailure(analysis.ErrAnalyzerPackageNotFound, "analyzer package is not present in the application index", entry, "index_entry", nil)
	}
	if err := validatePackageMetadata(entry); err != nil {
		return VerifiedPackage{}, err
	}
	executablePath, err := resolveTrustedPath(c.root, entry.ExecutablePath)
	if err != nil {
		return VerifiedPackage{}, classifyPackagePathError(entry, err, "executable_path")
	}
	descriptorPath, err := resolveTrustedPath(c.root, entry.DescriptorPath)
	if err != nil {
		return VerifiedPackage{}, classifyPackagePathError(entry, err, "descriptor_path")
	}
	if err := validateRegularFile(executablePath); err != nil {
		return VerifiedPackage{}, classifyPackageFileError(entry, err, "executable")
	}
	if err := validateRegularFile(descriptorPath); err != nil {
		return VerifiedPackage{}, classifyPackageFileError(entry, err, "descriptor")
	}
	if err := validatePackageShape(filepath.Dir(executablePath), filepath.Base(executablePath)); err != nil {
		return VerifiedPackage{}, packageFailure(analysis.ErrAnalyzerPackageIntegrityMismatch, "packaged analyzer directory is not trusted", entry, "package", err)
	}

	executableDigest, err := fileSHA256(executablePath)
	if err != nil {
		return VerifiedPackage{}, classifyPackageFileError(entry, err, "executable")
	}
	descriptorData, err := os.ReadFile(descriptorPath)
	if err != nil {
		return VerifiedPackage{}, classifyPackageFileError(entry, err, "descriptor")
	}
	descriptorDigest := sha256Bytes(descriptorData)
	if executableDigest != entry.ExecutableSHA256 {
		return VerifiedPackage{}, packageFailure(analysis.ErrAnalyzerPackageIntegrityMismatch, "packaged analyzer executable integrity verification failed", entry, "executable_digest", nil)
	}
	if descriptorDigest != entry.DescriptorSHA256 {
		return VerifiedPackage{}, packageFailure(analysis.ErrAnalyzerPackageIntegrityMismatch, "packaged analyzer descriptor integrity verification failed", entry, "descriptor_digest", nil)
	}

	descriptor, err := processprotocol.DecodeDescriptor(descriptorData)
	if err != nil {
		if analysis.ErrorCodeOf(err) == analysis.ErrAPIIncompatible {
			return VerifiedPackage{}, packageFailure(analysis.ErrAnalyzerPackageAPIIncompatible, "packaged analyzer descriptor API is incompatible", entry, "api_version", err)
		}
		return VerifiedPackage{}, packageFailure(analysis.ErrAnalyzerPackageManifestMismatch, "packaged analyzer descriptor manifest is invalid", entry, "manifest", err)
	}
	if descriptor.Manifest.ID != entry.LogicalAnalyzerID {
		return VerifiedPackage{}, packageFailure(analysis.ErrAnalyzerPackageManifestMismatch, "packaged analyzer logical ID disagrees with the index", entry, "logical_analyzer_id", nil)
	}
	if descriptor.Manifest.Version != entry.AnalyzerVersion {
		return VerifiedPackage{}, packageFailure(analysis.ErrAnalyzerPackageManifestMismatch, "packaged analyzer version disagrees with the index", entry, "analyzer_version", nil)
	}
	if descriptor.Manifest.Language != entry.Language {
		return VerifiedPackage{}, packageFailure(analysis.ErrAnalyzerPackageManifestMismatch, "packaged analyzer language disagrees with the index", entry, "language", nil)
	}
	if descriptor.Manifest.APIVersion != entry.APIVersion {
		return VerifiedPackage{}, packageFailure(analysis.ErrAnalyzerPackageManifestMismatch, "packaged analyzer API disagrees with the index", entry, "api_version", nil)
	}
	if descriptor.Command != filepath.Base(executablePath) {
		return VerifiedPackage{}, packageFailure(analysis.ErrAnalyzerPackageManifestMismatch, "packaged analyzer descriptor does not point to the indexed executable", entry, "executable_path", nil)
	}

	return VerifiedPackage{
		Root:           c.root,
		Index:          entry,
		Descriptor:     descriptor,
		ExecutablePath: executablePath,
		DescriptorPath: descriptorPath,
	}, nil
}

func (c *Catalog) contains(entry AnalyzerPackage) bool {
	for _, indexed := range c.index.Packages {
		if indexed == entry {
			return true
		}
	}
	return false
}

func cloneCatalogManifest(value analysis.Manifest) analysis.Manifest {
	value.DetectionMarkers = append([]analysis.DetectionMarker(nil), value.DetectionMarkers...)
	value.Capabilities = append([]string(nil), value.Capabilities...)
	options := append([]analysis.OptionDescriptor(nil), value.Options...)
	value.Options = make([]analysis.OptionDescriptor, len(options))
	for index, option := range options {
		value.Options[index] = option
		value.Options[index].AllowedValues = append([]string(nil), option.AllowedValues...)
	}
	return value
}

// HostPlatform returns the normalized exact platform token used by the
// package index. Unsupported host combinations are returned unchanged so the
// selector can report analyzer_platform_unsupported rather than guessing.
func HostPlatform() string {
	return runtime.GOOS + "-" + runtime.GOARCH
}

func validateIndexMetadata(index AnalyzerIndex) error {
	if index.SchemaVersion != IndexSchemaVersion {
		return indexError(fmt.Errorf("unsupported schema version %q", index.SchemaVersion))
	}
	if code := validateAPIVersion(index.HostAPIVersion); code != nil {
		return indexError(code)
	}
	if strings.TrimSpace(index.GeneratedBy) == "" || strings.ContainsAny(index.GeneratedBy, "\x00\r\n") {
		return indexError(errors.New("generated_by is invalid"))
	}
	if len(index.Packages) == 0 {
		return indexError(errors.New("packages must be a non-empty array"))
	}
	if !sort.SliceIsSorted(index.Packages, func(i, j int) bool { return packageLess(index.Packages[i], index.Packages[j]) }) {
		return indexError(errors.New("packages are not sorted"))
	}
	seen := make(map[string]struct{}, len(index.Packages))
	for _, entry := range index.Packages {
		if err := validatePackageMetadata(entry); err != nil {
			if analysis.ErrorCodeOf(err) == analysis.ErrAnalyzerPackageAPIIncompatible {
				return err
			}
			return indexError(err)
		}
		key := entry.LogicalAnalyzerID + "\x00" + entry.Platform
		if _, exists := seen[key]; exists {
			return indexError(fmt.Errorf("duplicate package %q", key))
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validatePackageMetadata(entry AnalyzerPackage) error {
	if !packageIDPattern.MatchString(entry.LogicalAnalyzerID) {
		return indexError(fmt.Errorf("logical analyzer ID is invalid"))
	}
	if !packageVersionPattern.MatchString(entry.AnalyzerVersion) {
		return indexError(fmt.Errorf("analyzer version is invalid"))
	}
	if !packageLanguagePattern.MatchString(entry.Language) {
		return indexError(fmt.Errorf("analyzer language is invalid"))
	}
	if code := validateAPIVersion(entry.APIVersion); code != nil {
		return indexError(code)
	}
	target, err := PlatformTargetFor(entry.Platform)
	if err != nil {
		return indexError(err)
	}
	if !isSafeRelativePath(entry.ExecutablePath) || !isSafeRelativePath(entry.DescriptorPath) {
		return indexError(errors.New("package paths must be normalized relative paths"))
	}
	if err := validatePackagePaths(entry, target); err != nil {
		return indexError(err)
	}
	if err := validateDigest(entry.ExecutableSHA256, "executable", entry.LogicalAnalyzerID); err != nil {
		return indexError(err)
	}
	if err := validateDigest(entry.DescriptorSHA256, "descriptor", entry.LogicalAnalyzerID); err != nil {
		return indexError(err)
	}
	return nil
}

func validateAPIVersion(value string) error {
	matches := packageAPIVersionPattern.FindStringSubmatch(value)
	if len(matches) == 0 {
		return errors.New("API version is malformed")
	}
	if matches[1] != "1" {
		return analysis.NewHostError(analysis.ErrAnalyzerPackageAPIIncompatible, "analyzer package API major version is not supported", map[string]any{
			"api_version":      value,
			"host_api_version": processprotocol.ProtocolVersion,
		})
	}
	return nil
}

func indexError(cause error) error {
	if analysis.ErrorCodeOf(cause) == analysis.ErrAnalyzerPackageAPIIncompatible {
		return cause
	}
	return analysis.WrapHostError(analysis.ErrAnalyzerPackageIndexInvalid, "analyzer package index is invalid", cause, map[string]any{
		"index": indexFileName,
	})
}

func packageFailure(code analysis.ErrorCode, message string, entry AnalyzerPackage, field string, cause error) error {
	details := map[string]any{
		"analyzer_id": entry.LogicalAnalyzerID,
		"platform":    entry.Platform,
	}
	if field != "" {
		details["validation"] = field
	}
	if cause == nil {
		return analysis.NewHostError(code, message, details)
	}
	return analysis.WrapHostError(code, message, cause, details)
}

func hostErrorMessage(err error) string {
	var hostErr *analysis.HostError
	if errors.As(err, &hostErr) {
		return hostErr.Message
	}
	return "packaged analyzer validation failed"
}

func classifyPackagePathError(entry AnalyzerPackage, err error, field string) error {
	if errors.Is(err, os.ErrNotExist) {
		return packageFailure(analysis.ErrAnalyzerPackageNotFound, "indexed analyzer package file is missing", entry, field, nil)
	}
	return packageFailure(analysis.ErrAnalyzerPackageIntegrityMismatch, "indexed analyzer package path is not trusted", entry, field, err)
}

func classifyPackageFileError(entry AnalyzerPackage, err error, field string) error {
	if errors.Is(err, os.ErrNotExist) {
		return packageFailure(analysis.ErrAnalyzerPackageNotFound, "indexed analyzer package file is missing", entry, field, nil)
	}
	return packageFailure(analysis.ErrAnalyzerPackageIntegrityMismatch, "indexed analyzer package file is not trusted", entry, field, err)
}

func resolveTrustedPath(root, relative string) (string, error) {
	if !isSafeRelativePath(relative) {
		return "", fmt.Errorf("%w: normalized relative path required", errPackagePathUnsafe)
	}
	rootAbs, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return "", fmt.Errorf("resolve package root: %w", err)
	}
	rootInfo, err := os.Lstat(rootAbs)
	if err != nil {
		return "", err
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return "", fmt.Errorf("%w: package root is not a regular directory", errPackagePathUnsafe)
	}

	candidate := rootAbs
	parts := strings.Split(relative, "/")
	for index, part := range parts {
		candidate = filepath.Join(candidate, filepath.FromSlash(part))
		info, statErr := os.Lstat(candidate)
		if statErr != nil {
			return "", statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("%w: package path contains a symlink", errPackagePathUnsafe)
		}
		if index < len(parts)-1 && !info.IsDir() {
			return "", fmt.Errorf("%w: package path component is not a directory", errPackagePathUnsafe)
		}
	}

	// Lstat rejects ordinary symlinks. EvalSymlinks additionally protects the
	// boundary from platform-specific substitutions such as directory junctions.
	rootReal, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return "", err
	}
	candidateReal, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", err
	}
	relativeToRoot, err := filepath.Rel(rootReal, candidateReal)
	if err != nil || filepath.IsAbs(relativeToRoot) || relativeToRoot == ".." || strings.HasPrefix(relativeToRoot, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("%w: package path escapes the application root", errPackagePathUnsafe)
	}
	return candidate, nil
}
