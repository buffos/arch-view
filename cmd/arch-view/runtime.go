package main

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/distribution"
	"github.com/buffo/arch-view/internal/analysis/processanalyzer"
)

const analyzerRuntimeAuto = "auto"

type commandRuntimeOptions struct {
	Mode                 string
	ModeProvided         bool
	AllowUntrustedPlugin bool
	PluginPaths          []string
	AnalyzerID           string
	Language             string
	ListOnly             bool
}

// configureCommandHost applies the public runtime policy around a built-in
// host. Packaged mode replaces built-ins with verified application-managed
// process analyzers; in-process and explicit modes retain the existing
// registry with visible provenance.
func configureCommandHost(base *analysis.Host, options commandRuntimeOptions) (*analysis.Host, *distribution.Catalog, error) {
	mode := strings.ToLower(strings.TrimSpace(options.Mode))
	if mode == "" {
		mode = analyzerRuntimeAuto
	}

	root, rootAvailable, rootConfigured := applicationAnalyzerRoot()
	if mode == analyzerRuntimeAuto {
		if len(options.PluginPaths) > 0 {
			if options.ModeProvided {
				return nil, nil, runtimeOverrideError("local analyzer descriptors require --analyzer-runtime explicit", nil)
			}
			// Preserve the pre-distribution Python pilot for source checkouts
			// that explicitly provide --plugin. A release installation has an
			// application-managed catalog and must use the explicit opt-in path.
			if rootConfigured || rootAvailable {
				return nil, nil, runtimeOverrideError("--plugin is rejected by packaged runtime selection; use --analyzer-runtime explicit --allow-untrusted-plugin", nil)
			}
			return configureExplicitHost(base, options, true)
		}
		if rootConfigured {
			if !rootAvailable {
				return nil, nil, loadMissingCatalog(root)
			}
			mode = analysis.RuntimeModePackaged
		} else {
			mode = analysis.RuntimeModeInProcess
		}
	}

	switch mode {
	case analysis.RuntimeModePackaged:
		if len(options.PluginPaths) > 0 {
			return nil, nil, runtimeOverrideError("--plugin is only valid with --analyzer-runtime explicit", nil)
		}
		if options.AllowUntrustedPlugin {
			return nil, nil, runtimeOverrideError("--allow-untrusted-plugin requires an explicit local descriptor", nil)
		}
		if !rootAvailable {
			return nil, nil, loadMissingCatalog(root)
		}
		return configurePackagedHost(root, base, options)
	case analysis.RuntimeModeInProcess:
		if len(options.PluginPaths) > 0 {
			return nil, nil, runtimeOverrideError("--plugin requires --analyzer-runtime explicit", nil)
		}
		if options.AllowUntrustedPlugin {
			return nil, nil, runtimeOverrideError("--allow-untrusted-plugin requires an explicit local descriptor", nil)
		}
		host, err := cloneHostWithRuntime(base, analysis.RuntimeSelection{
			Mode:     analysis.RuntimeModeInProcess,
			Source:   "built-in-registry",
			Platform: distribution.HostPlatform(),
		})
		return host, nil, err
	case analysis.RuntimeModeExplicit:
		return configureExplicitHost(base, options, false)
	default:
		return nil, nil, analysis.NewHostError(analysis.ErrInvalidRequest, "analyzer runtime mode is unsupported", map[string]any{
			"runtime_mode": options.Mode,
			"supported":    []string{analysis.RuntimeModePackaged, analysis.RuntimeModeInProcess, analysis.RuntimeModeExplicit},
		})
	}
}

func configurePackagedHost(root string, base *analysis.Host, options commandRuntimeOptions) (*analysis.Host, *distribution.Catalog, error) {
	catalog, err := distribution.LoadCatalog(root)
	if err != nil {
		return nil, nil, err
	}
	platform := distribution.HostPlatform()
	if _, err := distribution.PlatformTargetFor(platform); err != nil {
		return nil, nil, analysis.NewHostError(analysis.ErrAnalyzerPlatformUnsupported, "the current host platform is outside the packaged analyzer matrix", map[string]any{
			"platform": platform,
		})
	}

	registry := analysis.NewRegistry()
	availability := catalog.Availability()
	if !options.ListOnly && options.AnalyzerID == "" && options.Language == "" {
		for _, packageInfo := range availability {
			if packageInfo.Platform == platform && packageInfo.Status != distribution.PackageAvailable {
				_, err := catalog.SelectPackage(packageInfo.LogicalAnalyzerID, platform)
				return nil, nil, err
			}
		}
	}
	for _, packageInfo := range availability {
		if packageInfo.Platform != platform || packageInfo.Status != distribution.PackageAvailable {
			continue
		}
		analyzer, err := catalog.NewPackagedAnalyzer(packageInfo.LogicalAnalyzerID, platform, processanalyzer.DefaultConfig())
		if err != nil {
			if packageInfo.LogicalAnalyzerID == options.AnalyzerID {
				return nil, nil, err
			}
			continue
		}
		if err := registry.Register(analyzer); err != nil {
			return nil, nil, analysis.WrapHostError(analysis.ErrAnalyzerPackageManifestMismatch, "packaged analyzer could not be registered", err, map[string]any{
				"analyzer_id": packageInfo.LogicalAnalyzerID,
				"platform":    platform,
			})
		}
	}

	if options.AnalyzerID != "" {
		if _, err := catalog.SelectHostPackage(options.AnalyzerID); err != nil {
			return nil, nil, err
		}
	}
	if options.Language != "" {
		if err := validatePackagedLanguage(catalog, availability, options.Language, platform); err != nil {
			return nil, nil, err
		}
	}

	host := analysis.NewHostWithRuntime(registry, analysis.RuntimeSelection{
		Mode:     analysis.RuntimeModePackaged,
		Source:   "application-index",
		Platform: platform,
	})
	return host, catalog, nil
}

func configureExplicitHost(base *analysis.Host, options commandRuntimeOptions, legacy bool) (*analysis.Host, *distribution.Catalog, error) {
	if len(options.PluginPaths) == 0 {
		return nil, nil, runtimeOverrideError("explicit runtime selection requires at least one --plugin descriptor", nil)
	}
	if !legacy && !options.AllowUntrustedPlugin {
		return nil, nil, runtimeOverrideError("explicit local descriptors require --allow-untrusted-plugin", nil)
	}
	selection := analysis.RuntimeSelection{
		Mode:     analysis.RuntimeModeExplicit,
		Source:   "explicit-descriptor",
		Platform: distribution.HostPlatform(),
	}
	var host *analysis.Host
	var err error
	if legacy {
		host, err = base.WithRuntime(analysis.RuntimeSelection{})
	} else {
		host = analysis.NewHostWithRuntime(analysis.NewRegistry(), selection)
	}
	if err != nil {
		return nil, nil, err
	}
	if err := loadExternalPlugins(host, options.PluginPaths); err != nil {
		return nil, nil, err
	}
	return host, nil, nil
}

func cloneHostWithRuntime(base *analysis.Host, selection analysis.RuntimeSelection) (*analysis.Host, error) {
	if base == nil {
		return nil, errors.New("base analysis host is nil")
	}
	return base.WithRuntime(selection)
}

func validatePackagedLanguage(catalog *distribution.Catalog, values []distribution.PackageAvailability, language, platform string) error {
	language = strings.ToLower(strings.TrimSpace(language))
	var rejected *distribution.PackageAvailability
	for _, value := range values {
		if value.Platform != platform || value.Language != language {
			continue
		}
		if value.Status == distribution.PackageAvailable {
			return nil
		}
		if rejected == nil {
			candidate := value
			rejected = &candidate
		}
	}
	if rejected != nil {
		_, err := catalog.SelectPackage(rejected.LogicalAnalyzerID, platform)
		if err != nil {
			return err
		}
	}
	return analysis.NewHostError(analysis.ErrAnalyzerPackageNotFound, "no packaged analyzer is available for the requested language and platform", map[string]any{
		"language": language,
		"platform": platform,
	})
}

func runtimeOverrideError(message string, details map[string]any) error {
	return analysis.NewHostError(analysis.ErrAnalyzerRuntimeOverrideRequired, message, details)
}

func loadMissingCatalog(root string) error {
	return analysis.NewHostError(analysis.ErrAnalyzerPackageIndexInvalid, "application-managed analyzer index could not be loaded", map[string]any{
		"index":      filepath.Join(root, "index.json"),
		"root":       root,
		"validation": "missing",
	})
}

// applicationAnalyzerRoot intentionally considers only an explicitly
// configured root or the directory beside the running Arch View executable.
// It never searches the target repository for analyzers.
func applicationAnalyzerRoot() (root string, available, configured bool) {
	for _, name := range []string{"ARCH_VIEW_ANALYZER_ROOT", "ARCH_VIEW_ANALYZERS_ROOT"} {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			absolute, err := filepath.Abs(value)
			if err != nil {
				return filepath.Clean(value), false, true
			}
			_, statErr := os.Stat(absolute)
			return filepath.Clean(absolute), statErr == nil, true
		}
	}

	executable, err := os.Executable()
	if err != nil {
		return "", false, false
	}
	candidate := filepath.Join(filepath.Dir(executable), "analyzers")
	if _, statErr := os.Stat(candidate); statErr == nil {
		return candidate, true, true
	}
	return candidate, false, false
}

func runtimeModeWasProvided(fs interface{ Visit(func(*flag.Flag)) }) bool {
	provided := false
	fs.Visit(func(value *flag.Flag) {
		if value.Name == "analyzer-runtime" {
			provided = true
		}
	})
	return provided
}

type analyzerListing struct {
	analysis.Manifest
	Availability  string `json:"availability,omitempty"`
	Platform      string `json:"platform,omitempty"`
	RuntimeMode   string `json:"runtime_mode,omitempty"`
	RuntimeSource string `json:"runtime_source,omitempty"`
	ErrorCode     string `json:"error_code,omitempty"`
	ErrorMessage  string `json:"error_message,omitempty"`
}

func packageListings(host *analysis.Host, catalog *distribution.Catalog) []analyzerListing {
	if catalog == nil {
		manifests := host.ListManifests()
		selection := host.Runtime()
		result := make([]analyzerListing, 0, len(manifests))
		for _, manifest := range manifests {
			result = append(result, analyzerListing{
				Manifest:      manifest,
				Availability:  string(distribution.PackageAvailable),
				RuntimeMode:   selection.Mode,
				RuntimeSource: selection.Source,
			})
		}
		return result
	}

	manifestByID := make(map[string]analysis.Manifest)
	for _, manifest := range host.ListManifests() {
		manifestByID[manifest.ID] = manifest
	}
	selection := host.Runtime()
	values := catalog.Availability()
	result := make([]analyzerListing, 0, len(values))
	for _, value := range values {
		manifest := manifestByID[value.LogicalAnalyzerID]
		if manifest.ID == "" {
			manifest = analysis.Manifest{
				ID:         value.LogicalAnalyzerID,
				Version:    value.AnalyzerVersion,
				Language:   value.Language,
				APIVersion: value.APIVersion,
			}
		}
		result = append(result, analyzerListing{
			Manifest:      manifest,
			Availability:  string(value.Status),
			Platform:      value.Platform,
			RuntimeMode:   selection.Mode,
			RuntimeSource: selection.Source,
			ErrorCode:     string(value.ErrorCode),
			ErrorMessage:  value.ErrorMessage,
		})
	}
	return result
}

func sortAnalyzerListings(values []analyzerListing) {
	sort.Slice(values, func(i, j int) bool {
		if values[i].ID != values[j].ID {
			return values[i].ID < values[j].ID
		}
		return values[i].Platform < values[j].Platform
	})
}
