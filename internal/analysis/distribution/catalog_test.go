package distribution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/processanalyzer"
	"github.com/buffo/arch-view/internal/analysis/processprotocol"
)

func TestCatalogVerifiesPackageAndListsWithoutLaunching(t *testing.T) {
	root := t.TempDir()
	entry := writeCatalogPackage(t, root, "org.archview.go", HostPlatform(), "binary")

	catalog, err := LoadCatalog(root)
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	verified, err := catalog.SelectHostPackage(entry.LogicalAnalyzerID)
	if err != nil {
		t.Fatalf("select package: %v", err)
	}
	if verified.Descriptor.Manifest.ID != entry.LogicalAnalyzerID || verified.ExecutablePath == "" {
		t.Fatalf("verified package = %#v", verified)
	}

	availability := catalog.Availability()
	if len(availability) != 1 || availability[0].Status != PackageAvailable || availability[0].ErrorCode != "" {
		t.Fatalf("availability = %#v", availability)
	}

	// Creating the adapter is intentionally the last operation in this test;
	// NewAnalyzer only resolves the already-verified executable and cannot
	// launch it until a caller invokes Detect or Analyze.
	analyzer, err := catalog.NewAnalyzer(entry.LogicalAnalyzerID, HostPlatform(), processAnalyzerTestConfig())
	if err != nil {
		t.Fatalf("create packaged analyzer: %v", err)
	}
	if analyzer.Manifest().ID != entry.LogicalAnalyzerID {
		t.Fatalf("packaged analyzer manifest = %#v", analyzer.Manifest())
	}
}

func TestCatalogRejectsTamperedFilesWithStableErrors(t *testing.T) {
	root := t.TempDir()
	entry := writeCatalogPackage(t, root, "org.archview.go", HostPlatform(), "binary")

	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(entry.ExecutablePath)), []byte("tampered"), 0o755); err != nil {
		t.Fatalf("tamper executable: %v", err)
	}
	catalog, err := LoadCatalog(root)
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	if _, err := catalog.SelectHostPackage(entry.LogicalAnalyzerID); analysis.ErrorCodeOf(err) != analysis.ErrAnalyzerPackageIntegrityMismatch {
		t.Fatalf("tampered executable error code = %q, want %q", analysis.ErrorCodeOf(err), analysis.ErrAnalyzerPackageIntegrityMismatch)
	}
	if got := catalog.Availability()[0].Status; got != PackageRejected {
		t.Fatalf("tampered executable status = %q, want rejected", got)
	}

	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(entry.ExecutablePath)), []byte("binary"), 0o755); err != nil {
		t.Fatalf("restore executable: %v", err)
	}
	entry = rewriteCatalogDescriptor(t, root, entry, func(descriptor *processprotocol.Descriptor) {
		descriptor.Manifest.Version = "9.9.9"
	})
	catalog, err = LoadCatalog(root)
	if err != nil {
		t.Fatalf("reload catalog: %v", err)
	}
	if _, err := catalog.VerifyPackage(entry); analysis.ErrorCodeOf(err) != analysis.ErrAnalyzerPackageManifestMismatch {
		t.Fatalf("manifest mismatch error code = %q, want %q", analysis.ErrorCodeOf(err), analysis.ErrAnalyzerPackageManifestMismatch)
	}
}

func TestCatalogRejectsUnindexedEntries(t *testing.T) {
	root := t.TempDir()
	entry := writeCatalogPackage(t, root, "org.archview.go", HostPlatform(), "binary")
	catalog, err := LoadCatalog(root)
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	unindexed := entry
	unindexed.ExecutableSHA256 = strings.Repeat("0", 64)
	if _, err := catalog.VerifyPackage(unindexed); analysis.ErrorCodeOf(err) != analysis.ErrAnalyzerPackageNotFound {
		t.Fatalf("unindexed entry error code = %q, want %q", analysis.ErrorCodeOf(err), analysis.ErrAnalyzerPackageNotFound)
	}
}

func TestPackagedAnalyzerReverifiesBeforeOperation(t *testing.T) {
	root := t.TempDir()
	entry := writeCatalogPackage(t, root, "org.archview.go", HostPlatform(), "binary")
	catalog, err := LoadCatalog(root)
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	analyzer, err := catalog.NewPackagedAnalyzer(entry.LogicalAnalyzerID, entry.Platform, processAnalyzerTestConfig())
	if err != nil {
		t.Fatalf("create packaged analyzer: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(entry.ExecutablePath)), []byte("tampered-after-selection"), 0o755); err != nil {
		t.Fatalf("tamper selected executable: %v", err)
	}
	if _, err := analyzer.Detect(context.Background(), analysis.DetectRequest{ProjectRoot: t.TempDir()}); analysis.ErrorCodeOf(err) != analysis.ErrAnalyzerPackageIntegrityMismatch {
		t.Fatalf("post-selection tamper error code = %q, want %q", analysis.ErrorCodeOf(err), analysis.ErrAnalyzerPackageIntegrityMismatch)
	}
}

func TestCatalogRejectsSymlinkedIndex(t *testing.T) {
	root := t.TempDir()
	writeCatalogPackage(t, root, "org.archview.go", HostPlatform(), "binary")
	indexPath := filepath.Join(root, indexFileName)
	outside := filepath.Join(t.TempDir(), indexFileName)
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read index: %v", err)
	}
	if err := os.WriteFile(outside, data, 0o644); err != nil {
		t.Fatalf("write outside index: %v", err)
	}
	if err := os.Remove(indexPath); err != nil {
		t.Fatalf("remove original index: %v", err)
	}
	if err := os.Symlink(outside, indexPath); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}
	if _, err := LoadCatalog(root); analysis.ErrorCodeOf(err) != analysis.ErrAnalyzerPackageIndexInvalid {
		t.Fatalf("symlinked index error code = %q, want %q", analysis.ErrorCodeOf(err), analysis.ErrAnalyzerPackageIndexInvalid)
	}
}

func TestCatalogReportsMissingPackageFiles(t *testing.T) {
	root := t.TempDir()
	entry := writeCatalogPackage(t, root, "org.archview.go", HostPlatform(), "binary")
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(entry.DescriptorPath))); err != nil {
		t.Fatalf("remove descriptor: %v", err)
	}
	catalog, err := LoadCatalog(root)
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	if _, err := catalog.SelectHostPackage(entry.LogicalAnalyzerID); analysis.ErrorCodeOf(err) != analysis.ErrAnalyzerPackageNotFound {
		t.Fatalf("missing descriptor error code = %q, want %q", analysis.ErrorCodeOf(err), analysis.ErrAnalyzerPackageNotFound)
	}
	availability := catalog.Availability()
	if len(availability) != 1 || availability[0].Status != PackageUnavailable || availability[0].ErrorCode != analysis.ErrAnalyzerPackageNotFound {
		t.Fatalf("missing descriptor availability = %#v", availability)
	}
}

func TestPackagedAnalyzerMapsLaunchAndHelloFailures(t *testing.T) {
	entry := AnalyzerPackage{LogicalAnalyzerID: "org.archview.go", Platform: HostPlatform()}
	analyzer := &PackagedAnalyzer{entry: entry}

	unsupported := &processprotocol.ProtocolError{Kind: processprotocol.ErrorUnsupportedProtocol, FrameType: processprotocol.FrameHello}
	if got := analysis.ErrorCodeOf(analyzer.mapRuntimeError(unsupported)); got != analysis.ErrAnalyzerPackageAPIIncompatible {
		t.Fatalf("unsupported hello error code = %q, want %q", got, analysis.ErrAnalyzerPackageAPIIncompatible)
	}
	mismatch := &processprotocol.ProtocolError{Kind: processprotocol.ErrorInvalidPayload, FrameType: processprotocol.FrameHello}
	if got := analysis.ErrorCodeOf(analyzer.mapRuntimeError(mismatch)); got != analysis.ErrAnalyzerPackageManifestMismatch {
		t.Fatalf("mismatched hello error code = %q, want %q", got, analysis.ErrAnalyzerPackageManifestMismatch)
	}
	if got := analysis.ErrorCodeOf(analyzer.mapRuntimeError(errors.New("process exited"))); got != analysis.ErrAnalyzerPackageLaunchFailed {
		t.Fatalf("launch error code = %q, want %q", got, analysis.ErrAnalyzerPackageLaunchFailed)
	}
	var nilAnalyzer *PackagedAnalyzer
	_, nilErr := nilAnalyzer.Analyze(context.Background(), analysis.AnalyzeRequest{})
	if got := analysis.ErrorCodeOf(nilErr); got != analysis.ErrAnalyzerPackageLaunchFailed {
		t.Fatalf("nil packaged analyzer error code = %q, want %q", got, analysis.ErrAnalyzerPackageLaunchFailed)
	}
}

func TestCatalogRejectsMalformedIndexAndDuplicatePackages(t *testing.T) {
	root := t.TempDir()
	first := writeCatalogPackage(t, root, "org.archview.go", HostPlatform(), "binary")
	index := readCatalogIndex(t, root)
	index.Packages = append(index.Packages, first)
	writeCatalogIndex(t, root, index)
	if _, err := LoadCatalog(root); analysis.ErrorCodeOf(err) != analysis.ErrAnalyzerPackageIndexInvalid {
		t.Fatalf("duplicate index error code = %q, want %q", analysis.ErrorCodeOf(err), analysis.ErrAnalyzerPackageIndexInvalid)
	}

	index.Packages = []AnalyzerPackage{first}
	index.Packages[0].ExecutablePath = "../outside/analyzer.exe"
	writeCatalogIndex(t, root, index)
	if _, err := LoadCatalog(root); analysis.ErrorCodeOf(err) != analysis.ErrAnalyzerPackageIndexInvalid {
		t.Fatalf("traversal index error code = %q, want %q", analysis.ErrorCodeOf(err), analysis.ErrAnalyzerPackageIndexInvalid)
	}

	index.Packages[0].ExecutablePath = first.ExecutablePath
	index.Packages[0].APIVersion = "arch-view.analyzer/v2"
	writeCatalogIndex(t, root, index)
	if _, err := LoadCatalog(root); analysis.ErrorCodeOf(err) != analysis.ErrAnalyzerPackageAPIIncompatible {
		t.Fatalf("API mismatch error code = %q, want %q", analysis.ErrorCodeOf(err), analysis.ErrAnalyzerPackageAPIIncompatible)
	}
}

func TestCatalogValidatesEveryIndexMetadataField(t *testing.T) {
	tests := []struct {
		name string
		edit func(*AnalyzerIndex)
		want analysis.ErrorCode
	}{
		{name: "schema", edit: func(index *AnalyzerIndex) { index.SchemaVersion = "arch-view.analyzer-index/v9" }, want: analysis.ErrAnalyzerPackageIndexInvalid},
		{name: "host API", edit: func(index *AnalyzerIndex) { index.HostAPIVersion = "arch-view.analyzer/v9" }, want: analysis.ErrAnalyzerPackageAPIIncompatible},
		{name: "logical ID", edit: func(index *AnalyzerIndex) { index.Packages[0].LogicalAnalyzerID = "INVALID" }, want: analysis.ErrAnalyzerPackageIndexInvalid},
		{name: "semantic version", edit: func(index *AnalyzerIndex) { index.Packages[0].AnalyzerVersion = "latest" }, want: analysis.ErrAnalyzerPackageIndexInvalid},
		{name: "language", edit: func(index *AnalyzerIndex) { index.Packages[0].Language = "Go" }, want: analysis.ErrAnalyzerPackageIndexInvalid},
		{name: "platform", edit: func(index *AnalyzerIndex) { index.Packages[0].Platform = "linux-arm64" }, want: analysis.ErrAnalyzerPackageIndexInvalid},
		{name: "absolute path", edit: func(index *AnalyzerIndex) { index.Packages[0].ExecutablePath = filepath.Join(t.TempDir(), "analyzer") }, want: analysis.ErrAnalyzerPackageIndexInvalid},
		{name: "alternate separator", edit: func(index *AnalyzerIndex) {
			index.Packages[0].DescriptorPath = strings.ReplaceAll(index.Packages[0].DescriptorPath, "/", "\\")
		}, want: analysis.ErrAnalyzerPackageIndexInvalid},
		{name: "uppercase digest", edit: func(index *AnalyzerIndex) {
			index.Packages[0].ExecutableSHA256 = strings.ToUpper(index.Packages[0].ExecutableSHA256)
		}, want: analysis.ErrAnalyzerPackageIndexInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeCatalogPackage(t, root, "org.archview.go", HostPlatform(), "binary")
			index := readCatalogIndex(t, root)
			test.edit(&index)
			writeCatalogIndex(t, root, index)
			if _, err := LoadCatalog(root); analysis.ErrorCodeOf(err) != test.want {
				t.Fatalf("error code = %q, want %q", analysis.ErrorCodeOf(err), test.want)
			}
		})
	}
}

func TestCatalogClassifiesDescriptorTamperAPIAndLaunchFailures(t *testing.T) {
	t.Run("descriptor digest", func(t *testing.T) {
		root := t.TempDir()
		entry := writeCatalogPackage(t, root, "org.archview.go", HostPlatform(), "binary")
		descriptorPath := filepath.Join(root, filepath.FromSlash(entry.DescriptorPath))
		if err := os.WriteFile(descriptorPath, []byte("tampered"), 0o644); err != nil {
			t.Fatalf("tamper descriptor: %v", err)
		}
		catalog, err := LoadCatalog(root)
		if err != nil {
			t.Fatalf("load catalog: %v", err)
		}
		if _, err := catalog.SelectHostPackage(entry.LogicalAnalyzerID); analysis.ErrorCodeOf(err) != analysis.ErrAnalyzerPackageIntegrityMismatch {
			t.Fatalf("descriptor tamper error code = %q, want %q", analysis.ErrorCodeOf(err), analysis.ErrAnalyzerPackageIntegrityMismatch)
		}
	})

	t.Run("descriptor API", func(t *testing.T) {
		root := t.TempDir()
		entry := writeCatalogPackage(t, root, "org.archview.go", HostPlatform(), "binary")
		entry = rewriteCatalogDescriptor(t, root, entry, func(descriptor *processprotocol.Descriptor) {
			descriptor.Manifest.APIVersion = "arch-view.analyzer/v9"
		})
		catalog, err := LoadCatalog(root)
		if err != nil {
			t.Fatalf("load catalog: %v", err)
		}
		if _, err := catalog.VerifyPackage(entry); analysis.ErrorCodeOf(err) != analysis.ErrAnalyzerPackageAPIIncompatible {
			t.Fatalf("descriptor API error code = %q, want %q", analysis.ErrorCodeOf(err), analysis.ErrAnalyzerPackageAPIIncompatible)
		}
	})

	t.Run("launch", func(t *testing.T) {
		root := t.TempDir()
		entry := writeCatalogPackage(t, root, "org.archview.go", HostPlatform(), "not-an-executable")
		catalog, err := LoadCatalog(root)
		if err != nil {
			t.Fatalf("load catalog: %v", err)
		}
		analyzer, err := catalog.NewPackagedAnalyzer(entry.LogicalAnalyzerID, entry.Platform, processAnalyzerTestConfig())
		if err != nil {
			t.Fatalf("create packaged analyzer: %v", err)
		}
		if _, err := analyzer.Detect(context.Background(), analysis.DetectRequest{ProjectRoot: t.TempDir()}); analysis.ErrorCodeOf(err) != analysis.ErrAnalyzerPackageLaunchFailed {
			t.Fatalf("launch error code = %q, want %q", analysis.ErrorCodeOf(err), analysis.ErrAnalyzerPackageLaunchFailed)
		}
	})
}

func TestCatalogUsesExactPlatformOnly(t *testing.T) {
	root := t.TempDir()
	current := HostPlatform()
	target, err := PlatformTargetFor(current)
	if err != nil {
		t.Skipf("current host platform is outside the first release matrix: %v", err)
	}
	other := "linux-amd64"
	if current == other {
		other = "darwin-arm64"
	}
	first := writeCatalogPackage(t, root, "org.archview.go", current, "current")
	second := writeCatalogPackage(t, root, "org.archview.go", other, "other")
	index := readCatalogIndex(t, root)
	index.Packages = []AnalyzerPackage{second, first}
	// Canonical package order is ID then platform.
	if packageLess(index.Packages[1], index.Packages[0]) {
		index.Packages[0], index.Packages[1] = index.Packages[1], index.Packages[0]
	}
	writeCatalogIndex(t, root, index)
	catalog, err := LoadCatalog(root)
	if err != nil {
		t.Fatalf("load multi-platform catalog: %v", err)
	}
	verified, err := catalog.SelectHostPackage(first.LogicalAnalyzerID)
	if err != nil {
		t.Fatalf("select host package: %v", err)
	}
	if got := mustReadCatalogFile(t, verified.ExecutablePath); string(got) != "current" {
		t.Fatalf("selected executable = %q, want current", got)
	}
	if _, err := catalog.SelectPackage(first.LogicalAnalyzerID, "freebsd-amd64"); analysis.ErrorCodeOf(err) != analysis.ErrAnalyzerPlatformUnsupported {
		t.Fatalf("unsupported platform error code = %q, want %q", analysis.ErrorCodeOf(err), analysis.ErrAnalyzerPlatformUnsupported)
	}
	if _, err := catalog.SelectPackage("org.archview.missing", target.Name); analysis.ErrorCodeOf(err) != analysis.ErrAnalyzerPackageNotFound {
		t.Fatalf("missing package error code = %q, want %q", analysis.ErrorCodeOf(err), analysis.ErrAnalyzerPackageNotFound)
	}
}

func TestCatalogRejectsSymlinkSubstitution(t *testing.T) {
	root := t.TempDir()
	entry := writeCatalogPackage(t, root, "org.archview.go", HostPlatform(), "binary")
	packagePath := filepath.Join(root, filepath.FromSlash(entry.LogicalAnalyzerID))
	outside := filepath.Join(t.TempDir(), entry.LogicalAnalyzerID)
	if err := os.Rename(packagePath, outside); err != nil {
		t.Fatalf("move package for symlink test: %v", err)
	}
	if err := os.Symlink(outside, packagePath); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}
	catalog, err := LoadCatalog(root)
	if err != nil {
		t.Fatalf("load symlink catalog: %v", err)
	}
	if _, err := catalog.SelectHostPackage(entry.LogicalAnalyzerID); analysis.ErrorCodeOf(err) != analysis.ErrAnalyzerPackageIntegrityMismatch {
		t.Fatalf("symlink error code = %q, want %q", analysis.ErrorCodeOf(err), analysis.ErrAnalyzerPackageIntegrityMismatch)
	}
}

func writeCatalogPackage(t *testing.T, root, id, platform, executableContent string) AnalyzerPackage {
	t.Helper()
	target, err := PlatformTargetFor(platform)
	if err != nil {
		t.Fatalf("platform target: %v", err)
	}
	manifest := analysis.Manifest{
		ID:               id,
		Version:          "1.0.0",
		Language:         "go",
		APIVersion:       analysis.AnalyzerAPIVersion,
		DetectionMarkers: []analysis.DetectionMarker{{Kind: "file", Value: "go.mod", Weight: 1}},
		Capabilities:     []string{"detect", "static_dependencies"},
		Options:          []analysis.OptionDescriptor{},
	}
	descriptor := processprotocol.Descriptor{
		SchemaVersion: processprotocol.DescriptorSchemaVersion,
		Manifest:      manifest,
		Command:       target.ExecutableName,
	}
	descriptorData, err := json.MarshalIndent(descriptor, "", "  ")
	if err != nil {
		t.Fatalf("marshal descriptor: %v", err)
	}
	descriptorData = append(descriptorData, '\n')
	packageDir := filepath.Join(root, filepath.FromSlash(id), filepath.FromSlash(platform))
	if err := os.MkdirAll(packageDir, 0o755); err != nil {
		t.Fatalf("create package directory: %v", err)
	}
	executablePath := filepath.Join(packageDir, target.ExecutableName)
	descriptorPath := filepath.Join(packageDir, descriptorFileName)
	if err := os.WriteFile(executablePath, []byte(executableContent), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}
	if err := os.WriteFile(descriptorPath, descriptorData, 0o644); err != nil {
		t.Fatalf("write descriptor: %v", err)
	}
	entry := AnalyzerPackage{
		LogicalAnalyzerID: id,
		AnalyzerVersion:   manifest.Version,
		Language:          manifest.Language,
		APIVersion:        manifest.APIVersion,
		Platform:          platform,
		ExecutablePath:    filepath.ToSlash(filepath.Join(id, platform, target.ExecutableName)),
		DescriptorPath:    filepath.ToSlash(filepath.Join(id, platform, descriptorFileName)),
		ExecutableSHA256:  catalogSHA256([]byte(executableContent)),
		DescriptorSHA256:  catalogSHA256(descriptorData),
	}
	index := AnalyzerIndex{SchemaVersion: IndexSchemaVersion, HostAPIVersion: processprotocol.ProtocolVersion, GeneratedBy: "arch-view-build-test", Packages: []AnalyzerPackage{entry}}
	writeCatalogIndex(t, root, index)
	return entry
}

func rewriteCatalogDescriptor(t *testing.T, root string, entry AnalyzerPackage, mutate func(*processprotocol.Descriptor)) AnalyzerPackage {
	t.Helper()
	descriptorPath := filepath.Join(root, filepath.FromSlash(entry.DescriptorPath))
	data, err := os.ReadFile(descriptorPath)
	if err != nil {
		t.Fatalf("read descriptor: %v", err)
	}
	descriptor, err := processprotocol.DecodeDescriptor(data)
	if err != nil {
		t.Fatalf("decode descriptor: %v", err)
	}
	mutate(&descriptor)
	data, err = json.MarshalIndent(descriptor, "", "  ")
	if err != nil {
		t.Fatalf("marshal changed descriptor: %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(descriptorPath, data, 0o644); err != nil {
		t.Fatalf("write changed descriptor: %v", err)
	}
	entry.DescriptorSHA256 = catalogSHA256(data)
	index := readCatalogIndex(t, root)
	index.Packages[0] = entry
	writeCatalogIndex(t, root, index)
	return entry
}

func readCatalogIndex(t *testing.T, root string) AnalyzerIndex {
	t.Helper()
	index, err := ReadIndex(filepath.Join(root, indexFileName))
	if err != nil {
		t.Fatalf("read catalog index: %v", err)
	}
	return index
}

func writeCatalogIndex(t *testing.T, root string, index AnalyzerIndex) {
	t.Helper()
	data, err := MarshalIndex(index)
	if err != nil {
		t.Fatalf("marshal catalog index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, indexFileName), data, 0o644); err != nil {
		t.Fatalf("write catalog index: %v", err)
	}
}

func mustReadCatalogFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read catalog file: %v", err)
	}
	return data
}

func catalogSHA256(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func processAnalyzerTestConfig() processanalyzer.Config {
	return processanalyzer.DefaultConfig()
}
