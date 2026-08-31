package distribution

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	appversion "github.com/buffo/arch-view/internal/version"
)

const (
	ReleaseManifestSchemaVersion = "arch-view.release/v1"
	releaseManifestFileName      = "release.json"
)

// ReleaseManifest identifies one complete host-and-analyzer publication.
// Analyzer package details remain in analyzers/index.json.
type ReleaseManifest struct {
	SchemaVersion       string `json:"schema_version"`
	Application         string `json:"application"`
	Version             string `json:"version"`
	Commit              string `json:"commit"`
	BuildDate           string `json:"build_date"`
	BuildID             string `json:"build_id"`
	Platform            string `json:"platform"`
	HostExecutable      string `json:"host_executable"`
	HostSHA256          string `json:"host_sha256"`
	AnalyzerIndexPath   string `json:"analyzer_index_path"`
	AnalyzerIndexSHA256 string `json:"analyzer_index_sha256"`
}

// MarshalReleaseManifest encodes the canonical human-readable release
// manifest representation.
func MarshalReleaseManifest(manifest ReleaseManifest) ([]byte, error) {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal release manifest: %w", err)
	}
	return append(data, '\n'), nil
}

// DecodeReleaseManifest decodes exactly one strict release manifest object.
func DecodeReleaseManifest(data []byte) (ReleaseManifest, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var manifest ReleaseManifest
	if err := decoder.Decode(&manifest); err != nil {
		return ReleaseManifest{}, fmt.Errorf("decode release manifest: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return ReleaseManifest{}, fmt.Errorf("decode release manifest: trailing JSON value")
		}
		return ReleaseManifest{}, fmt.Errorf("decode release manifest: trailing data: %w", err)
	}
	return manifest, nil
}

// ReadReleaseManifest reads and strictly decodes release.json from a release
// root.
func ReadReleaseManifest(root string) (ReleaseManifest, error) {
	absolute, err := absoluteDirectory(root)
	if err != nil {
		return ReleaseManifest{}, err
	}
	manifestPath, err := resolveSafePath(absolute, releaseManifestFileName)
	if err != nil {
		return ReleaseManifest{}, fmt.Errorf("resolve release manifest: %w", err)
	}
	if err := validateRegularFile(manifestPath); err != nil {
		return ReleaseManifest{}, fmt.Errorf("release manifest: %w", err)
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return ReleaseManifest{}, fmt.Errorf("read release manifest: %w", err)
	}
	return DecodeReleaseManifest(data)
}

// ValidateReleaseManifest verifies release metadata, canonical paths, the
// host and analyzer index, and their recorded digests.
func ValidateReleaseManifest(root string, manifest ReleaseManifest) error {
	absolute, err := absoluteDirectory(root)
	if err != nil {
		return err
	}
	if manifest.SchemaVersion != ReleaseManifestSchemaVersion {
		return fmt.Errorf("release manifest schema version %q is unsupported", manifest.SchemaVersion)
	}
	info := appversion.Info{
		Application: manifest.Application,
		Version:     manifest.Version,
		Commit:      manifest.Commit,
		BuildDate:   manifest.BuildDate,
		BuildID:     manifest.BuildID,
	}
	if err := info.Validate(); err != nil {
		return fmt.Errorf("release manifest metadata: %w", err)
	}
	target, err := PlatformTargetFor(manifest.Platform)
	if err != nil {
		return fmt.Errorf("release manifest platform: %w", err)
	}
	expectedHost := "arch-view" + hostExecutableSuffix(target)
	if manifest.HostExecutable != expectedHost {
		return fmt.Errorf("release manifest host executable %q is not %q", manifest.HostExecutable, expectedHost)
	}
	if manifest.AnalyzerIndexPath != path.Join("analyzers", "index.json") {
		return fmt.Errorf("release manifest analyzer index path %q is not canonical", manifest.AnalyzerIndexPath)
	}
	if err := validateDigest(manifest.HostSHA256, "host", manifest.Application); err != nil {
		return err
	}
	if err := validateDigest(manifest.AnalyzerIndexSHA256, "analyzer index", manifest.Application); err != nil {
		return err
	}
	hostPath, err := resolveSafePath(absolute, manifest.HostExecutable)
	if err != nil {
		return fmt.Errorf("resolve release host: %w", err)
	}
	indexPath, err := resolveSafePath(absolute, manifest.AnalyzerIndexPath)
	if err != nil {
		return fmt.Errorf("resolve release analyzer index: %w", err)
	}
	if err := validateRegularFile(hostPath); err != nil {
		return fmt.Errorf("release host: %w", err)
	}
	if err := validateRegularFile(indexPath); err != nil {
		return fmt.Errorf("release analyzer index: %w", err)
	}
	if digest, err := fileSHA256(hostPath); err != nil {
		return fmt.Errorf("hash release host: %w", err)
	} else if digest != manifest.HostSHA256 {
		return fmt.Errorf("release host digest mismatch")
	}
	if digest, err := fileSHA256(indexPath); err != nil {
		return fmt.Errorf("hash release analyzer index: %w", err)
	} else if digest != manifest.AnalyzerIndexSHA256 {
		return fmt.Errorf("release analyzer index digest mismatch")
	}
	index, err := ReadIndex(indexPath)
	if err != nil {
		return fmt.Errorf("read release analyzer index: %w", err)
	}
	if err := ValidateIndex(filepath.Dir(indexPath), index, nil, target.Name); err != nil {
		return fmt.Errorf("validate release analyzer index: %w", err)
	}
	if index.GeneratedBy != "arch-view-build-"+manifest.BuildID {
		return fmt.Errorf("release build ID %q does not match analyzer index %q", manifest.BuildID, index.GeneratedBy)
	}
	return nil
}
