package rustanalyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

// Project is the resolved, single-crate Cargo boundary used by the analyzer.
// All paths are absolute internally; result construction converts them to
// project-relative paths before crossing the common contract.
type Project struct {
	Root                  string
	ManifestPath          string
	RelativeManifestPath  string
	CrateRoot             string
	RelativeCrateRoot     string
	PackageName           string
	Edition               string
	WorkspacePath         string
	RelativeWorkspacePath string
	Boundary              string
	Features              []string
	DeclaredFeatures      []string
	Target                string
	IncludeTests          bool
	IncludeExamples       bool
	ExcludePatterns       []string
	Targets               []RustTarget
	Dependencies          []CargoDependency
	Diagnostics           []analysis.Diagnostic
}

// RustTarget is a source root declared or inferred for the selected package.
type RustTarget struct {
	Name         string
	Kind         string
	RelativePath string
	AbsolutePath string
}

// CargoDependency is a data-only dependency declaration from Cargo.toml.
type CargoDependency struct {
	Name            string
	Kind            string
	Version         string
	Path            string
	Registry        string
	Target          string
	Optional        bool
	Workspace       bool
	DefaultFeatures *bool
	Features        []string
	Source          analysis.SourceReference
}

// ResolveProject selects exactly one Cargo package. It reads Cargo manifests
// as data and never asks Cargo to resolve the workspace.
func ResolveProject(root string, options analysis.EffectiveOptions) (Project, error) {
	absoluteRoot, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return Project{}, analysis.WrapHostError(analysis.ErrInvalidRequest, "Rust project root could not be normalized", err, nil)
	}
	absoluteRoot = filepath.Clean(absoluteRoot)
	info, err := os.Stat(absoluteRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return Project{}, analysis.NewHostError(analysis.ErrUnreadableProject, "Rust project root does not exist", map[string]any{"project_root": root})
		}
		return Project{}, analysis.WrapHostError(analysis.ErrUnreadableProject, "Rust project root could not be read", err, map[string]any{"project_root": root})
	}
	if !info.IsDir() {
		return Project{}, analysis.NewHostError(analysis.ErrInvalidRequest, "Rust project root must be a directory", map[string]any{"project_root": root})
	}

	rootManifestPath := filepath.Join(absoluteRoot, "Cargo.toml")
	if !safeProjectFile(absoluteRoot, "Cargo.toml") {
		return Project{}, analysis.NewHostError(analysis.ErrUnsupportedProject, "Rust project requires Cargo.toml at the selected root", map[string]any{"project_root": root})
	}
	rootManifest, err := readCargoManifest(rootManifestPath)
	if err != nil {
		return Project{}, err
	}

	workspace := rootManifest.workspaceMembers != nil || len(rootManifest.workspaceExclude) > 0
	candidates := []cargoCandidate{}
	if rootManifest.packageName != "" {
		candidates = append(candidates, cargoCandidate{ManifestPath: rootManifestPath, Manifest: rootManifest})
	}
	if workspace {
		members, memberErr := workspaceCandidates(absoluteRoot, rootManifest)
		if memberErr != nil {
			return Project{}, memberErr
		}
		for _, member := range members {
			if sameRustPath(member.ManifestPath, rootManifestPath) {
				continue
			}
			candidates = append(candidates, member)
		}
	}
	if len(candidates) == 0 {
		return Project{}, analysis.NewHostError(analysis.ErrUnsupportedProject, "Cargo.toml does not declare a package or readable workspace members", map[string]any{"project_root": root})
	}

	selector := optionString(options, "crate")
	if selector != "" && !safeCrateSelector(absoluteRoot, selector) {
		return Project{}, analysis.NewHostError(analysis.ErrUnsupportedProject, "requested Cargo crate selection escapes the selected project root", map[string]any{"crate": selector})
	}
	selected, err := chooseCandidate(candidates, selector, absoluteRoot)
	if err != nil {
		return Project{}, err
	}
	selectedManifest := selected.Manifest
	if selected.ManifestPath != rootManifestPath {
		// The member manifest was already read for selection; keep the explicit
		// read boundary here so future manifest fields cannot leak from a stale
		// workspace candidate.
		selectedManifest, err = readCargoManifest(selected.ManifestPath)
		if err != nil {
			return Project{}, err
		}
	}

	workspacePath := ""
	if workspace {
		workspacePath = rootManifestPath
	}
	features := optionStrings(options, "features")
	target := optionString(options, "target")
	includeTests := optionBool(options, "include_tests")
	includeExamples := optionBool(options, "include_examples")
	excludes := optionStrings(options, "exclude")
	project := makeProject(absoluteRoot, selected, selectedManifest, workspacePath, features, target, includeTests, includeExamples, excludes)
	project.Dependencies = mergeWorkspaceDependencies(project.Dependencies, rootManifest.workspaceDependencies)
	project.Diagnostics = append(project.Diagnostics, validateDependencyPaths(project)...)
	project.Targets, project.Diagnostics = resolveTargets(project, selectedManifest, project.Diagnostics)
	return project, nil
}

type cargoCandidate struct {
	ManifestPath string
	Manifest     cargoManifest
}

func workspaceCandidates(root string, manifest cargoManifest) ([]cargoCandidate, error) {
	members := append([]string(nil), manifest.workspaceMembers...)
	if len(members) == 0 {
		return []cargoCandidate{}, nil
	}
	result := make([]cargoCandidate, 0, len(members))
	seen := make(map[string]struct{})
	for _, pattern := range members {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if filepath.IsAbs(pattern) || !safeWorkspacePattern(root, pattern) {
			return nil, analysis.NewHostError(analysis.ErrUnsupportedProject, "Cargo workspace member escapes the selected project root", map[string]any{"member": pattern})
		}
		matches, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(pattern)))
		if err != nil {
			return nil, analysis.WrapHostError(analysis.ErrUnsupportedProject, "Cargo workspace member pattern is invalid", err, map[string]any{"member": pattern})
		}
		if len(matches) == 0 && !strings.ContainsAny(pattern, "*?[") {
			matches = []string{filepath.Join(root, filepath.FromSlash(pattern))}
		}
		for _, match := range matches {
			match = filepath.Clean(match)
			if !pathWithin(root, match) || !resolvedPathWithin(root, match) {
				return nil, analysis.NewHostError(analysis.ErrUnsupportedProject, "Cargo workspace member resolves outside the selected project root", map[string]any{"member": pattern})
			}
			info, statErr := os.Stat(match)
			if statErr != nil || !info.IsDir() {
				return nil, analysis.NewHostError(analysis.ErrUnreadableProject, "Cargo workspace member directory could not be read", map[string]any{"member": pattern, "path": match})
			}
			if workspaceExcluded(root, match, manifest.workspaceExclude) {
				continue
			}
			manifestPath := filepath.Join(match, "Cargo.toml")
			if !safeProjectFile(root, filepath.ToSlash(mustRelative(root, manifestPath))) {
				return nil, analysis.NewHostError(analysis.ErrUnreadableProject, "Cargo workspace member does not contain a safe Cargo.toml", map[string]any{"member": pattern, "path": manifestPath})
			}
			memberManifest, readErr := readCargoManifest(manifestPath)
			if readErr != nil {
				return nil, readErr
			}
			if memberManifest.packageName == "" {
				return nil, analysis.NewHostError(analysis.ErrUnsupportedProject, "Cargo workspace member does not declare a package", map[string]any{"path": manifestPath})
			}
			if _, exists := seen[filepath.Clean(manifestPath)]; exists {
				continue
			}
			seen[filepath.Clean(manifestPath)] = struct{}{}
			result = append(result, cargoCandidate{ManifestPath: manifestPath, Manifest: memberManifest})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ManifestPath < result[j].ManifestPath })
	return result, nil
}

func chooseCandidate(candidates []cargoCandidate, selector, root string) (cargoCandidate, error) {
	if selector != "" {
		for _, candidate := range candidates {
			if cargoCandidateMatches(candidate, selector, root) {
				return candidate, nil
			}
		}
		return cargoCandidate{}, analysis.NewHostError(analysis.ErrModuleSelection, "requested Cargo crate was not found in the selected project", map[string]any{"crate": selector, "candidates": candidateNames(candidates, root)})
	}
	if len(candidates) != 1 {
		return cargoCandidate{}, analysis.NewHostError(analysis.ErrModuleSelection, "Cargo project exposes multiple crates; an explicit crate selection is required", map[string]any{"crate_count": len(candidates), "candidates": candidateNames(candidates, root)})
	}
	return candidates[0], nil
}

func cargoCandidateMatches(candidate cargoCandidate, selector, root string) bool {
	if candidate.Manifest.packageName == selector || normalizeCargoName(candidate.Manifest.packageName) == normalizeCargoName(selector) {
		return true
	}
	selectorPath := selector
	if strings.HasSuffix(strings.ToLower(selectorPath), "cargo.toml") {
		selectorPath = filepath.Dir(selectorPath)
	}
	if filepath.IsAbs(selectorPath) {
		selectorPath = filepath.Clean(selectorPath)
	} else {
		selectorPath = filepath.Join(root, filepath.FromSlash(selectorPath))
	}
	return sameRustPath(filepath.Dir(candidate.ManifestPath), selectorPath) || sameRustPath(candidate.ManifestPath, selectorPath)
}

func sameRustPath(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func safeCrateSelector(root, selector string) bool {
	selectorPath := selector
	if strings.HasSuffix(strings.ToLower(selectorPath), "cargo.toml") {
		selectorPath = filepath.Dir(selectorPath)
	}
	if filepath.IsAbs(selectorPath) {
		return pathWithin(root, filepath.Clean(selectorPath))
	}
	return pathWithin(root, filepath.Clean(filepath.Join(root, filepath.FromSlash(selectorPath))))
}

func candidateNames(candidates []cargoCandidate, root string) []string {
	result := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		relative, err := filepath.Rel(root, filepath.Dir(candidate.ManifestPath))
		if err != nil || relative == "." {
			relative = candidate.Manifest.packageName
		}
		result = append(result, candidate.Manifest.packageName+" ("+filepath.ToSlash(relative)+")")
	}
	sort.Strings(result)
	return result
}

func makeProject(root string, candidate cargoCandidate, manifest cargoManifest, workspacePath string, features []string, target string, includeTests, includeExamples bool, excludes []string) Project {
	crateRoot := filepath.Dir(candidate.ManifestPath)
	relativeManifest := mustRelative(root, candidate.ManifestPath)
	relativeCrate := mustRelative(root, crateRoot)
	if relativeCrate == "." {
		relativeCrate = "."
	}
	relativeWorkspace := ""
	if workspacePath != "" {
		relativeWorkspace = mustRelative(root, workspacePath)
	}
	dependencies := append([]CargoDependency(nil), manifest.dependencies...)
	sort.Slice(dependencies, func(i, j int) bool {
		if dependencies[i].Kind == dependencies[j].Kind {
			return dependencies[i].Name < dependencies[j].Name
		}
		return dependencies[i].Kind < dependencies[j].Kind
	})
	sort.Strings(features)
	sort.Strings(excludes)
	return Project{
		Root:                  root,
		ManifestPath:          candidate.ManifestPath,
		RelativeManifestPath:  filepath.ToSlash(relativeManifest),
		CrateRoot:             crateRoot,
		RelativeCrateRoot:     filepath.ToSlash(relativeCrate),
		PackageName:           manifest.packageName,
		Edition:               manifest.edition,
		WorkspacePath:         workspacePath,
		RelativeWorkspacePath: filepath.ToSlash(relativeWorkspace),
		Boundary:              "Cargo.toml",
		Features:              append([]string{}, features...),
		DeclaredFeatures:      sortedCargoFeatureNames(manifest.features),
		Target:                target,
		IncludeTests:          includeTests,
		IncludeExamples:       includeExamples,
		ExcludePatterns:       append([]string{}, excludes...),
		Dependencies:          dependencies,
		Diagnostics:           append([]analysis.Diagnostic(nil), manifest.diagnostics...),
	}
}

func resolveTargets(project Project, manifest cargoManifest, diagnostics []analysis.Diagnostic) ([]RustTarget, []analysis.Diagnostic) {
	seen := make(map[string]struct{})
	targets := make([]RustTarget, 0)
	add := func(name, kind, relative string) {
		relative = filepath.ToSlash(filepath.Clean(relative))
		if name == "" {
			name = project.PackageName
		}
		if relative == "." {
			diagnostics = append(diagnostics, analysis.Diagnostic{Code: "rust_target_invalid", Severity: "warning", Message: "Cargo declared a Rust target without a usable source path.", Subject: relative, Recoverable: true})
			return
		}
		if filepath.IsAbs(relative) || strings.HasPrefix(relative, "../") || strings.Contains(relative, ":") {
			diagnostics = append(diagnostics, analysis.Diagnostic{Code: "rust_target_outside_root", Severity: "error", Message: fmt.Sprintf("Rust target %q resolves outside the selected project root and was ignored.", relative), Subject: relative, Recoverable: true})
			return
		}
		absolute := filepath.Join(project.CrateRoot, filepath.FromSlash(relative))
		if !pathWithin(project.CrateRoot, absolute) || !resolvedPathWithin(project.Root, absolute) {
			diagnostics = append(diagnostics, analysis.Diagnostic{Code: "rust_target_outside_root", Severity: "error", Message: fmt.Sprintf("Rust target %q resolves outside the selected project root and was ignored.", relative), Subject: relative, Recoverable: true})
			return
		}
		if _, exists := seen[absolute]; exists {
			return
		}
		if info, err := os.Stat(absolute); err != nil || info.IsDir() {
			diagnostics = append(diagnostics, analysis.Diagnostic{Code: "rust_target_missing", Severity: "warning", Message: fmt.Sprintf("Rust target source %q could not be read.", relative), Path: projectRelative(project.Root, absolute), Recoverable: true})
			return
		}
		seen[absolute] = struct{}{}
		targets = append(targets, RustTarget{Name: name, Kind: kind, RelativePath: relative, AbsolutePath: absolute})
	}
	if manifest.libPath != "" {
		add(manifest.libName, "lib", manifest.libPath)
	} else if fileExists(filepath.Join(project.CrateRoot, "src", "lib.rs")) {
		add(manifest.packageName, "lib", filepath.ToSlash(filepath.Join("src", "lib.rs")))
	}
	for _, target := range manifest.binTargets {
		targetPath := target.Path
		if targetPath == "" {
			if target.Name == "" || normalizeCargoName(target.Name) == normalizeCargoName(project.PackageName) {
				targetPath = filepath.ToSlash(filepath.Join("src", "main.rs"))
			} else {
				targetPath = filepath.ToSlash(filepath.Join("src", "bin", target.Name+".rs"))
			}
		}
		add(target.Name, "bin", targetPath)
	}
	if len(manifest.binTargets) == 0 && fileExists(filepath.Join(project.CrateRoot, "src", "main.rs")) {
		add(manifest.packageName, "bin", filepath.ToSlash(filepath.Join("src", "main.rs")))
	}
	if binDir := filepath.Join(project.CrateRoot, "src", "bin"); directoryExists(binDir) {
		entries, err := os.ReadDir(binDir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && filepath.Ext(entry.Name()) == ".rs" {
					add(strings.TrimSuffix(entry.Name(), ".rs"), "bin", filepath.ToSlash(filepath.Join("src", "bin", entry.Name())))
				}
			}
		}
	}
	if project.IncludeTests {
		addRustDirectoryTargets(add, project, "tests", "test")
	}
	if project.IncludeExamples {
		addRustDirectoryTargets(add, project, "examples", "example")
		addRustDirectoryTargets(add, project, "benches", "bench")
	}
	sort.Slice(targets, func(i, j int) bool {
		if targets[i].RelativePath == targets[j].RelativePath {
			return targets[i].Kind < targets[j].Kind
		}
		return targets[i].RelativePath < targets[j].RelativePath
	})
	return targets, diagnostics
}

func validateDependencyPaths(project Project) []analysis.Diagnostic {
	diagnostics := make([]analysis.Diagnostic, 0)
	for _, dependency := range project.Dependencies {
		if dependency.Path == "" {
			continue
		}
		candidate := filepath.Clean(filepath.Join(project.CrateRoot, filepath.FromSlash(dependency.Path)))
		if filepath.IsAbs(dependency.Path) || !pathWithin(project.Root, candidate) || !resolvedPathWithin(project.Root, candidate) {
			diagnostics = append(diagnostics, analysis.Diagnostic{
				Code:        "rust_dependency_path_outside_root",
				Severity:    "warning",
				Message:     fmt.Sprintf("Cargo dependency %q points outside the selected project root and was retained only as metadata.", dependency.Name),
				Path:        project.RelativeManifestPath,
				Recoverable: true,
				Metadata:    map[string]any{"dependency": dependency.Name, "path": dependency.Path},
			})
		}
	}
	return diagnostics
}

func addRustDirectoryTargets(add func(string, string, string), project Project, directory, kind string) {
	root := filepath.Join(project.CrateRoot, directory)
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".rs" {
			continue
		}
		add(strings.TrimSuffix(entry.Name(), ".rs"), kind, filepath.ToSlash(filepath.Join(directory, entry.Name())))
	}
}

func mergeWorkspaceDependencies(dependencies []CargoDependency, workspace map[string]dependencySpec) []CargoDependency {
	if len(workspace) == 0 {
		return dependencies
	}
	for index := range dependencies {
		if !dependencies[index].Workspace {
			continue
		}
		if spec, ok := workspace[dependencies[index].Name]; ok {
			if dependencies[index].Version == "" {
				dependencies[index].Version = spec.Version
			}
			if dependencies[index].Path == "" {
				dependencies[index].Path = spec.Path
			}
			if dependencies[index].Registry == "" {
				dependencies[index].Registry = spec.Registry
			}
			if dependencies[index].DefaultFeatures == nil && spec.DefaultFeatures != nil {
				defaultFeatures := *spec.DefaultFeatures
				dependencies[index].DefaultFeatures = &defaultFeatures
			}
			for _, feature := range spec.Features {
				dependencies[index].Features = appendUniqueString(dependencies[index].Features, feature)
			}
			sort.Strings(dependencies[index].Features)
		}
	}
	return dependencies
}

func safeWorkspacePattern(root, pattern string) bool {
	clean := filepath.Clean(filepath.Join(root, filepath.FromSlash(pattern)))
	return pathWithin(root, clean)
}

func workspaceExcluded(root, path string, patterns []string) bool {
	relative := filepath.ToSlash(mustRelative(root, path))
	for _, pattern := range patterns {
		if matchesAnyRustExclude(relative, []string{pattern}) {
			return true
		}
	}
	return false
}

func optionString(options analysis.EffectiveOptions, name string) string {
	value := options.Values[name]
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func optionStrings(options analysis.EffectiveOptions, name string) []string {
	value := options.Values[name]
	values, _ := value.([]string)
	return append([]string{}, values...)
}

func optionBool(options analysis.EffectiveOptions, name string) bool {
	value := options.Values[name]
	result, _ := value.(bool)
	return result
}

func mustRelative(root, value string) string {
	relative, err := filepath.Rel(root, value)
	if err != nil || relative == "" {
		return "."
	}
	return filepath.Clean(relative)
}

func projectRelative(root, value string) string {
	return filepath.ToSlash(mustRelative(root, value))
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func directoryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func normalizeCargoName(value string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(value)), "_", "-")
}

func sortedCargoFeatureNames(features map[string]struct{}) []string {
	result := make([]string, 0, len(features))
	for feature := range features {
		result = append(result, feature)
	}
	sort.Strings(result)
	return result
}
