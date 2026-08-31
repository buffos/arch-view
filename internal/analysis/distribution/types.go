package distribution

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	clojureanalyzer "github.com/buffo/arch-view/internal/analyzers/clojure"
	goanalyzer "github.com/buffo/arch-view/internal/analyzers/go"
	pyanalyzer "github.com/buffo/arch-view/internal/analyzers/python"
	rustanalyzer "github.com/buffo/arch-view/internal/analyzers/rust"
	tsanalyzer "github.com/buffo/arch-view/internal/analyzers/typescript"
)

const (
	IndexSchemaVersion = "arch-view.analyzer-index/v1"
	descriptorFileName = "descriptor.json"
)

// AnalyzerPackage is the immutable package metadata stored in index.json.
type AnalyzerPackage struct {
	LogicalAnalyzerID string `json:"logical_analyzer_id"`
	AnalyzerVersion   string `json:"analyzer_version"`
	Language          string `json:"language"`
	APIVersion        string `json:"api_version"`
	Platform          string `json:"platform"`
	ExecutablePath    string `json:"executable_path"`
	DescriptorPath    string `json:"descriptor_path"`
	ExecutableSHA256  string `json:"executable_sha256"`
	DescriptorSHA256  string `json:"descriptor_sha256"`
}

// AnalyzerIndex is the deterministic manifest for one compiled analyzer
// distribution. A distribution is assembled for one exact target platform.
type AnalyzerIndex struct {
	SchemaVersion  string            `json:"schema_version"`
	HostAPIVersion string            `json:"host_api_version"`
	GeneratedBy    string            `json:"generated_by"`
	Packages       []AnalyzerPackage `json:"packages"`
}

// AnalyzerSpec connects a logical analyzer identity to its compiled command
// entrypoint and canonical in-process manifest.
type AnalyzerSpec struct {
	ID         string
	Entrypoint string
	Manifest   analysis.Manifest
}

// PlatformTarget is one supported release target and its executable naming
// convention.
type PlatformTarget struct {
	Name           string
	GOOS           string
	GOARCH         string
	ExecutableName string
}

var supportedPlatforms = map[string]PlatformTarget{
	"windows-amd64": {Name: "windows-amd64", GOOS: "windows", GOARCH: "amd64", ExecutableName: "analyzer.exe"},
	"linux-amd64":   {Name: "linux-amd64", GOOS: "linux", GOARCH: "amd64", ExecutableName: "analyzer"},
	"darwin-arm64":  {Name: "darwin-arm64", GOOS: "darwin", GOARCH: "arm64", ExecutableName: "analyzer"},
}

// SupportedPlatformTargets returns the supported target matrix in stable
// platform-name order.
func SupportedPlatformTargets() []PlatformTarget {
	result := make([]PlatformTarget, 0, len(supportedPlatforms))
	for _, target := range supportedPlatforms {
		result = append(result, target)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

// PlatformTargetFor resolves one exact supported platform name.
func PlatformTargetFor(name string) (PlatformTarget, error) {
	if strings.TrimSpace(name) != name || name == "" {
		return PlatformTarget{}, fmt.Errorf("platform target must be non-empty normalized text: %q", name)
	}
	target, ok := supportedPlatforms[name]
	if !ok {
		return PlatformTarget{}, fmt.Errorf("unsupported platform target %q", name)
	}
	return target, nil
}

func analyzerSpecs() []AnalyzerSpec {
	specs := []AnalyzerSpec{
		{Entrypoint: "./cmd/analyzers/clojure", Manifest: clojureanalyzer.New().Manifest()},
		{Entrypoint: "./cmd/analyzers/go", Manifest: goanalyzer.New().Manifest()},
		{Entrypoint: "./cmd/analyzers/python", Manifest: pyanalyzer.New().Manifest()},
		{Entrypoint: "./cmd/analyzers/rust", Manifest: rustanalyzer.New().Manifest()},
		{Entrypoint: "./cmd/analyzers/typescript", Manifest: tsanalyzer.New().Manifest()},
	}
	for i := range specs {
		specs[i].ID = specs[i].Manifest.ID
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].ID < specs[j].ID })
	return specs
}

// AnalyzerSpecs returns the enabled analyzer catalog in stable ID order.
func AnalyzerSpecs() []AnalyzerSpec {
	return analyzerSpecs()
}

// DefaultAnalyzerIDs returns all five compiled analyzer IDs in stable order.
func DefaultAnalyzerIDs() []string {
	specs := analyzerSpecs()
	ids := make([]string, 0, len(specs))
	for _, spec := range specs {
		ids = append(ids, spec.ID)
	}
	return ids
}

func resolveAnalyzerSpecs(ids []string) ([]AnalyzerSpec, error) {
	specs := analyzerSpecs()
	byID := make(map[string]AnalyzerSpec, len(specs))
	for _, spec := range specs {
		byID[spec.ID] = spec
	}
	if len(ids) == 0 {
		return specs, nil
	}

	result := make([]AnalyzerSpec, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, rawID := range ids {
		id := strings.TrimSpace(rawID)
		if id == "" {
			return nil, fmt.Errorf("enabled analyzer ID cannot be blank")
		}
		if _, exists := seen[id]; exists {
			return nil, fmt.Errorf("enabled analyzer ID %q is duplicated", id)
		}
		spec, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("enabled analyzer ID %q is not registered", id)
		}
		seen[id] = struct{}{}
		result = append(result, spec)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

// BuildRequest is the explicit input passed to a compiled analyzer build.
type BuildRequest struct {
	RepositoryRoot string
	Entrypoint     string
	OutputPath     string
	GoCommand      string
	CCCommand      string
	CXXCommand     string
	GOOS           string
	GOARCH         string
	CGOEnabled     string
	LinkerFlags    string
}

// BuildFunc allows assembly tests and release tooling to supply a controlled
// compiler invocation.
type BuildFunc func(context.Context, BuildRequest) error

// AssembleOptions contains every input that affects an analyzer distribution.
type AssembleOptions struct {
	RepositoryRoot string
	OutputRoot     string
	Platform       string
	BuildID        string
	GoCommand      string
	CCCommand      string
	CXXCommand     string
	AnalyzerIDs    []string
	Build          BuildFunc
}

// ReleaseOptions contains every input needed to assemble a host application
// and its analyzer distribution as one publication unit.
type ReleaseOptions struct {
	RepositoryRoot string
	OutputRoot     string
	Platform       string
	Version        string
	Commit         string
	BuildDate      string
	BuildID        string
	GoCommand      string
	CCCommand      string
	CXXCommand     string
	AnalyzerIDs    []string
	Build          BuildFunc
}
