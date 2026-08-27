package tsanalyzer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

type tsModuleIndex struct {
	ByAbsolute map[string]string
	ByRelative map[string]string
	Files      map[string]tsFileObservation
}

type tsResolution struct {
	Scope          string
	Name           string
	ModuleID       string
	ResolutionKind string
	Alias          string
	Package        string
	Runtime        string
	Confidence     *analysis.Confidence
	Metadata       map[string]any
	Diagnostic     *analysis.Diagnostic
}

func newTSModuleIndex(discovery discoveryResult) tsModuleIndex {
	index := tsModuleIndex{ByAbsolute: map[string]string{}, ByRelative: map[string]string{}, Files: discovery.Files}
	for relative, file := range discovery.Files {
		index.ByAbsolute[tsPathKey(file.AbsolutePath)] = file.ModuleID
		index.ByRelative[tsPathKey(relative)] = file.ModuleID
	}
	return index
}

func resolveTSImport(project Project, discovery discoveryResult, observation tsImportObservation, index tsModuleIndex) tsResolution {
	resolutionProject := project
	resolutionProject.Runtime = runtimeForTSObservation(project, observation)
	base := tsResolution{
		Name:       observation.Specifier,
		Runtime:    resolutionProject.Runtime,
		Metadata:   map[string]any{},
		Confidence: &analysis.Confidence{Basis: "external", Score: 0.8},
	}
	if observation.Dynamic {
		base.Scope = "dynamic"
		base.Name = observation.Specifier
		base.ResolutionKind = "dynamic"
		base.Confidence = &analysis.Confidence{Basis: "dynamic", Score: 0.2}
		base.Metadata["dynamic"] = true
		if observation.Computed {
			base.Metadata["computed"] = true
			base.Metadata["dynamic_expressions"] = []string{observation.Expression}
			base.Diagnostic = tsImportDiagnostic(observation, "typescript_dynamic_import", fmt.Sprintf("TypeScript dynamic loading expression %q could not be resolved statically; it was retained as a dynamic reference.", observation.Expression), "warning", true, base.Metadata)
			return base
		}
	}

	specifier := strings.TrimSpace(observation.Specifier)
	if specifier == "" || specifier == "<computed>" {
		base.Scope = "unresolved"
		base.ResolutionKind = "unresolved"
		base.Confidence = &analysis.Confidence{Basis: "unresolved", Score: 0.2}
		base.Diagnostic = tsImportDiagnostic(observation, "typescript_unresolved_import", "TypeScript dependency has no statically resolvable module specifier.", "warning", true, base.Metadata)
		return base
	}
	fromFile, ok := discovery.Files[relativePathForModule(discovery, observation.FromModuleID)]
	if !ok {
		base.Scope = "unresolved"
		base.ResolutionKind = "missing_source"
		base.Confidence = &analysis.Confidence{Basis: "unresolved", Score: 0.2}
		base.Diagnostic = tsImportDiagnostic(observation, "typescript_source_missing", "The source module for a dependency observation was not present in the selected file set.", "warning", true, base.Metadata)
		return base
	}
	fromPath := fromFile.AbsolutePath

	if strings.HasPrefix(specifier, ".") {
		resolution := resolveTSPath(resolutionProject, index, filepath.Join(filepath.Dir(fromPath), filepath.FromSlash(specifier)), "relative", "")
		if resolution.Scope == "" {
			if rootResolution, rootDirsOK := resolveTSRootDirs(resolutionProject, index, fromPath, specifier); rootDirsOK {
				resolution = rootResolution
			} else {
				resolution = unresolvedTSPath(observation, "relative", "typescript_unresolved_relative_import", "Relative TypeScript dependency could not be resolved within the selected project boundary.")
			}
		}
		return finalizeTSPathResolution(resolutionProject, observation, resolution, specifier)
	}
	if alias, resolution, matched := resolveTSPathAlias(resolutionProject, index, specifier); matched {
		if resolution.Scope == "" {
			resolution = unresolvedTSPath(observation, "path_alias", "typescript_unresolved_alias", fmt.Sprintf("Configured TypeScript path alias %q did not resolve to a selected local module.", alias))
		}
		resolution.Alias = alias
		return finalizeTSPathResolution(resolutionProject, observation, resolution, specifier)
	}
	if resolutionProject.CompilerOptions.BaseURL != "" {
		resolution := resolveTSPath(resolutionProject, index, filepath.Join(resolutionProject.CompilerOptions.BaseURL, filepath.FromSlash(specifier)), "base_url", "")
		if resolution.Scope == "local" {
			return finalizeTSPathResolution(resolutionProject, observation, resolution, specifier)
		}
	}
	if resolution, ok := resolveTSRootDirs(resolutionProject, index, fromPath, specifier); ok {
		return finalizeTSPathResolution(resolutionProject, observation, resolution, specifier)
	}
	if strings.HasPrefix(specifier, "#") && resolutionProject.PackageRoot != "" {
		if resolution, ok := resolveTSInternalImport(resolutionProject, index, specifier); ok {
			return finalizeTSPathResolution(resolutionProject, observation, resolution, specifier)
		}
		return unresolvedTSPath(observation, "package_imports", "typescript_package_import_unresolved", fmt.Sprintf("Package import %q could not be resolved safely.", specifier))
	}

	if standardLibrarySpecifier(specifier) {
		base.Scope = "standard_library"
		base.ResolutionKind = "standard_library"
		base.Confidence = &analysis.Confidence{Basis: "standard-library", Score: 0.95}
		base.Metadata["runtime"] = resolutionProject.Runtime
		return base
	}

	packageName, subpath := splitPackageSpecifier(specifier)
	if packageName != "" && resolutionProject.PackageName != "" && (packageName == resolutionProject.PackageName) {
		resolution := resolvePackageImport(resolutionProject, index, resolutionProject.PackageRoot, packageName, subpath, "self_package")
		if resolution.Scope == "" {
			return unresolvedTSPath(observation, "package_exports", "typescript_package_exports_unresolved", fmt.Sprintf("Self-package export %q could not be resolved safely.", specifier))
		}
		if resolution.Runtime == "" {
			resolution.Runtime = resolutionProject.Runtime
		}
		if resolution.Scope == "unresolved" && resolution.Diagnostic == nil {
			resolution.Diagnostic = tsImportDiagnostic(observation, "typescript_package_exports_unresolved", fmt.Sprintf("Self-package export %q could not be resolved safely.", specifier), "warning", true, resolution.Metadata)
		}
		return finalizeTSPathResolution(resolutionProject, observation, resolution, specifier)
	}

	base.Scope = "external"
	base.ResolutionKind = "package"
	base.Package = packageName
	base.Metadata["package"] = packageName
	if packageRoot, found := findPackageRoot(resolutionProject.Root, filepath.Dir(fromPath), packageName); found {
		base.Metadata["package_root"] = relativeProjectPath(project.Root, packageRoot)
		if manifest, err := readPackageManifest(resolutionProject.Root, filepath.Join(packageRoot, "package.json")); err == nil && len(manifest.Exports) > 0 {
			if _, ok := selectPackageExport(manifest.Exports, subpath, effectiveProjectRuntime(resolutionProject)); !ok {
				base.Scope = "unresolved"
				base.ResolutionKind = "package_exports_unresolved"
				base.Confidence = &analysis.Confidence{Basis: "unresolved", Score: 0.3}
				base.Diagnostic = tsImportDiagnostic(observation, "typescript_package_exports_unresolved", fmt.Sprintf("Package export %q is not statically available for the selected runtime context.", specifier), "warning", true, base.Metadata)
			}
		}
	}
	return base
}

func resolveTSInternalImport(project Project, index tsModuleIndex, specifier string) (tsResolution, bool) {
	manifest, err := readPackageManifest(project.Root, filepath.Join(project.PackageRoot, "package.json"))
	if err != nil || len(manifest.Imports) == 0 {
		return tsResolution{}, false
	}
	target, ok := selectPackageMapTarget(manifest.Imports, specifier, effectiveProjectRuntime(project))
	if !ok || !strings.HasPrefix(target, "./") {
		return tsResolution{}, false
	}
	targetPath := filepath.Clean(filepath.Join(project.PackageRoot, filepath.FromSlash(strings.TrimPrefix(target, "./"))))
	if !pathWithin(project.PackageRoot, targetPath) {
		return tsResolution{}, false
	}
	resolution := resolveTSPath(project, index, targetPath, "package_imports", "")
	if resolution.Scope == "" {
		return tsResolution{}, false
	}
	resolution.Metadata["package_import"] = specifier
	return resolution, true
}

func relativePathForModule(discovery discoveryResult, moduleID string) string {
	for relative, file := range discovery.Files {
		if file.ModuleID == moduleID {
			return relative
		}
	}
	return ""
}

func resolveTSPath(project Project, index tsModuleIndex, basePath, resolutionKind, alias string) tsResolution {
	if fileInfo, err := os.Stat(basePath); err == nil && fileInfo.IsDir() && safeFileWithinRoot(project.Root, basePath) {
		if manifest, manifestErr := readPackageManifest(project.Root, filepath.Join(basePath, "package.json")); manifestErr == nil {
			if len(manifest.Exports) > 0 {
				target, ok := selectPackageExport(manifest.Exports, ".", effectiveProjectRuntime(project))
				if !ok {
					return tsPackageExportsUnresolved(basePath)
				}
				if !strings.HasPrefix(target, "./") {
					return tsPackageExportsUnresolved(basePath)
				}
				if targetPath, valid := packageMetadataTarget(basePath, target); valid {
					if resolution := resolveTSPath(project, index, targetPath, "package_exports", alias); resolution.Scope != "" {
						return resolution
					}
				}
				return tsPackageExportsUnresolved(basePath)
			}
			for _, value := range []string{manifest.Types, manifest.Typings, manifest.Module, manifest.Main} {
				if value == "" {
					continue
				}
				if targetPath, valid := packageMetadataTarget(basePath, value); valid {
					if resolution := resolveTSPath(project, index, targetPath, "package_manifest", alias); resolution.Scope != "" {
						return resolution
					}
				}
			}
		}
	}
	candidates := tsFileCandidates(basePath, project.IncludeJavaScript)
	for _, candidate := range candidates {
		if !pathWithin(project.Root, candidate) {
			continue
		}
		if moduleID, ok := index.ByAbsolute[tsPathKey(candidate)]; ok {
			return tsLocalResolution(moduleID, resolutionKind, alias)
		}
	}
	for _, candidate := range tsFileCandidates(basePath, true) {
		if !pathWithin(project.Root, candidate) {
			continue
		}
		if fileInfo, err := os.Stat(candidate); err == nil && !fileInfo.IsDir() && safeFileWithinRoot(project.Root, candidate) {
			return tsExcludedResolution(relativeProjectPath(project.Root, candidate), resolutionKind, alias)
		}
	}
	return tsResolution{}
}

func tsPackageExportsUnresolved(packageRoot string) tsResolution {
	return tsResolution{
		Scope:          "unresolved",
		Name:           filepath.ToSlash(filepath.Clean(packageRoot)),
		ResolutionKind: "package_exports",
		Confidence:     &analysis.Confidence{Basis: "unresolved", Score: 0.2},
		Metadata:       map[string]any{"package_exports": true},
	}
}

func packageMetadataTarget(packageRoot, value string) (string, bool) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" || filepath.IsAbs(value) || (len(value) >= 2 && value[1] == ':') {
		return "", false
	}
	target := filepath.Clean(filepath.Join(packageRoot, filepath.FromSlash(value)))
	return target, pathWithin(packageRoot, target)
}

func tsFileCandidates(basePath string, allowJS bool) []string {
	basePath = filepath.Clean(basePath)
	extension := strings.ToLower(filepath.Ext(basePath))
	result := make([]string, 0, 10)
	appendUnique := func(value string) {
		value = filepath.Clean(value)
		for _, current := range result {
			if current == value {
				return
			}
		}
		result = append(result, value)
	}
	if extension != "" {
		switch extension {
		case ".js", ".jsx", ".mjs", ".cjs":
			without := strings.TrimSuffix(basePath, filepath.Ext(basePath))
			appendUnique(without + ".ts")
			appendUnique(without + ".tsx")
			appendUnique(without + ".d.ts")
			if allowJS {
				appendUnique(basePath)
			}
		default:
			appendUnique(basePath)
		}
	} else {
		for _, suffix := range []string{".ts", ".tsx", ".d.ts"} {
			appendUnique(basePath + suffix)
		}
		if allowJS {
			for _, suffix := range []string{".js", ".jsx"} {
				appendUnique(basePath + suffix)
			}
		}
	}
	if extension == "" {
		for _, suffix := range []string{".ts", ".tsx", ".d.ts"} {
			appendUnique(filepath.Join(basePath, "index"+suffix))
		}
		if allowJS {
			for _, suffix := range []string{".js", ".jsx"} {
				appendUnique(filepath.Join(basePath, "index"+suffix))
			}
		}
	}
	return result
}

func resolveTSPathAlias(project Project, index tsModuleIndex, specifier string) (string, tsResolution, bool) {
	keys := make([]string, 0, len(project.CompilerOptions.Paths))
	for key := range project.CompilerOptions.Paths {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if len(keys[i]) == len(keys[j]) {
			return keys[i] < keys[j]
		}
		return len(keys[i]) > len(keys[j])
	})
	for _, key := range keys {
		capture, matched := matchTSAlias(key, specifier)
		if !matched {
			continue
		}
		for _, target := range project.CompilerOptions.Paths[key] {
			candidate := strings.ReplaceAll(target, "*", capture)
			if resolution := resolveTSPath(project, index, candidate, "path_alias", key); resolution.Scope != "" {
				return key, resolution, true
			}
		}
		return key, tsResolution{}, true
	}
	return "", tsResolution{}, false
}

func matchTSAlias(pattern, value string) (string, bool) {
	if !strings.Contains(pattern, "*") {
		return "", pattern == value
	}
	parts := strings.SplitN(pattern, "*", 2)
	if !strings.HasPrefix(value, parts[0]) || !strings.HasSuffix(value, parts[1]) || len(value) < len(parts[0])+len(parts[1]) {
		return "", false
	}
	return value[len(parts[0]) : len(value)-len(parts[1])], true
}

func resolveTSRootDirs(project Project, index tsModuleIndex, fromPath, specifier string) (tsResolution, bool) {
	if !strings.HasPrefix(specifier, ".") {
		return tsResolution{}, false
	}
	for _, root := range project.CompilerOptions.RootDirs {
		if !pathWithin(root, fromPath) {
			continue
		}
		relative, err := filepath.Rel(root, filepath.Dir(fromPath))
		if err != nil {
			continue
		}
		for _, otherRoot := range project.CompilerOptions.RootDirs {
			candidate := filepath.Join(otherRoot, relative, filepath.FromSlash(specifier))
			if resolution := resolveTSPath(project, index, candidate, "root_dirs", ""); resolution.Scope != "" {
				return resolution, true
			}
		}
	}
	return tsResolution{}, false
}

func tsLocalResolution(moduleID, kind, alias string) tsResolution {
	return tsResolution{
		Scope: "local", ModuleID: moduleID, ResolutionKind: kind, Alias: alias,
		Confidence: &analysis.Confidence{Basis: "resolved", Score: 1}, Metadata: map[string]any{},
	}
}

func tsExcludedResolution(name, kind, alias string) tsResolution {
	return tsResolution{
		Scope: "excluded", Name: name, ResolutionKind: kind, Alias: alias,
		Confidence: &analysis.Confidence{Basis: "excluded", Score: 0.1}, Metadata: map[string]any{"excluded_path": name},
	}
}

func unresolvedTSPath(observation tsImportObservation, kind, code, message string) tsResolution {
	resolution := tsResolution{Scope: "unresolved", Name: observation.Specifier, ResolutionKind: kind, Confidence: &analysis.Confidence{Basis: "unresolved", Score: 0.2}, Metadata: map[string]any{}}
	resolution.Diagnostic = tsImportDiagnostic(observation, code, message, "warning", true, resolution.Metadata)
	return resolution
}

func finalizeTSPathResolution(project Project, observation tsImportObservation, resolution tsResolution, specifier string) tsResolution {
	if resolution.Name == "" {
		resolution.Name = specifier
	}
	if resolution.Metadata == nil {
		resolution.Metadata = map[string]any{}
	}
	if resolution.Runtime == "" && (resolution.Scope == "local" || resolution.Scope == "unresolved") {
		_, packageExport := resolution.Metadata["package_exports"]
		_, packageImport := resolution.Metadata["package_import"]
		if packageExport || packageImport || strings.HasPrefix(resolution.ResolutionKind, "package_") {
			resolution.Runtime = effectiveProjectRuntime(project)
		}
	}
	if observation.Dynamic {
		resolution.Metadata["dynamic"] = true
	}
	if observation.Computed {
		resolution.Metadata["computed"] = true
	}
	if resolution.Scope == "excluded" && resolution.Diagnostic == nil {
		resolution.Diagnostic = tsImportDiagnostic(observation, "typescript_excluded_import", fmt.Sprintf("TypeScript dependency %q points to a file excluded from the selected project scope.", specifier), "warning", true, resolution.Metadata)
	}
	if resolution.Scope == "unresolved" && resolution.Diagnostic == nil && strings.HasPrefix(resolution.ResolutionKind, "package_exports") {
		resolution.Diagnostic = tsImportDiagnostic(observation, "typescript_package_exports_unresolved", fmt.Sprintf("Package export %q could not be resolved safely.", specifier), "warning", true, resolution.Metadata)
	}
	return resolution
}

func standardLibrarySpecifier(specifier string) bool {
	if strings.HasPrefix(specifier, "node:") {
		return true
	}
	root := strings.SplitN(specifier, "/", 2)[0]
	for _, value := range []string{"assert", "buffer", "child_process", "cluster", "console", "constants", "crypto", "dgram", "diagnostics_channel", "dns", "domain", "events", "fs", "http", "http2", "https", "module", "net", "os", "path", "perf_hooks", "process", "punycode", "querystring", "readline", "repl", "stream", "string_decoder", "sys", "timers", "tls", "trace_events", "tty", "url", "util", "v8", "vm", "wasi", "worker_threads", "zlib"} {
		if root == value {
			return true
		}
	}
	return false
}

func splitPackageSpecifier(specifier string) (string, string) {
	parts := strings.Split(strings.Trim(specifier, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return "", ""
	}
	packageParts := 1
	if strings.HasPrefix(parts[0], "@") {
		if len(parts) < 2 {
			return strings.Join(parts, "/"), "."
		}
		packageParts = 2
	}
	if len(parts) <= packageParts {
		return strings.Join(parts[:packageParts], "/"), "."
	}
	return strings.Join(parts[:packageParts], "/"), "./" + strings.Join(parts[packageParts:], "/")
}

type tsPackageManifest struct {
	Name    string
	Type    string
	Main    string
	Module  string
	Types   string
	Typings string
	Exports json.RawMessage
	Imports json.RawMessage
}

func readPackageManifest(root, manifestPath string) (tsPackageManifest, error) {
	if !safeFileWithinRoot(root, manifestPath) {
		return tsPackageManifest{}, fmt.Errorf("package manifest resolves outside the selected project root")
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return tsPackageManifest{}, err
	}
	var value struct {
		Name    string          `json:"name"`
		Type    string          `json:"type"`
		Main    string          `json:"main"`
		Module  string          `json:"module"`
		Types   string          `json:"types"`
		Typings string          `json:"typings"`
		Exports json.RawMessage `json:"exports"`
		Imports json.RawMessage `json:"imports"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return tsPackageManifest{}, err
	}
	return tsPackageManifest{Name: value.Name, Type: value.Type, Main: value.Main, Module: value.Module, Types: value.Types, Typings: value.Typings, Exports: value.Exports, Imports: value.Imports}, nil
}

func findPackageRoot(projectRoot, startDirectory, packageName string) (string, bool) {
	if packageName == "" {
		return "", false
	}
	for directory := filepath.Clean(startDirectory); pathWithin(projectRoot, directory); directory = filepath.Dir(directory) {
		candidate := filepath.Join(directory, "node_modules", filepath.FromSlash(packageName))
		if pathWithin(projectRoot, candidate) {
			if info, err := os.Stat(candidate); err == nil && info.IsDir() && safeFileWithinRoot(projectRoot, candidate) {
				return candidate, true
			}
		}
		if directory == projectRoot {
			break
		}
	}
	return "", false
}

func resolvePackageImport(project Project, index tsModuleIndex, packageRoot, packageName, subpath, resolutionKind string) tsResolution {
	manifest, err := readPackageManifest(project.Root, filepath.Join(packageRoot, "package.json"))
	if err != nil {
		return tsResolution{}
	}
	target := subpath
	usedExports := len(manifest.Exports) > 0
	if len(manifest.Exports) > 0 {
		selected, ok := selectPackageExport(manifest.Exports, subpath, effectiveProjectRuntime(project))
		if !ok {
			return tsResolution{Scope: "unresolved", Name: packageName + strings.TrimPrefix(subpath, "."), ResolutionKind: "package_exports", Confidence: &analysis.Confidence{Basis: "unresolved", Score: 0.2}, Metadata: map[string]any{"package": packageName}}
		}
		target = selected
	} else if target == "." {
		for _, value := range []string{manifest.Types, manifest.Typings, manifest.Module, manifest.Main} {
			if value != "" {
				target = value
				break
			}
		}
		if target == "." {
			target = "./index"
		}
	}
	if !strings.HasPrefix(target, "./") {
		return tsResolution{}
	}
	targetPath := filepath.Clean(filepath.Join(packageRoot, filepath.FromSlash(strings.TrimPrefix(target, "./"))))
	if !pathWithin(packageRoot, targetPath) {
		return tsResolution{}
	}
	resolution := resolveTSPath(project, index, targetPath, resolutionKind, "")
	if resolution.Scope == "local" {
		resolution.Metadata["package"] = packageName
		if usedExports {
			resolution.Metadata["package_exports"] = true
		}
	}
	return resolution
}

func selectPackageExport(raw json.RawMessage, subpath, runtime string) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return "", false
	}
	if _, isString := value.(string); isString && subpath != "." {
		return "", false
	}
	return selectPackageExportValue(value, subpath, runtime)
}

func selectPackageMapTarget(raw json.RawMessage, specifier, runtime string) (string, bool) {
	var value map[string]any
	if len(raw) == 0 || json.Unmarshal(raw, &value) != nil {
		return "", false
	}
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i] == keys[j] {
			return false
		}
		if keys[i] == specifier {
			return true
		}
		if keys[j] == specifier {
			return false
		}
		if len(keys[i]) == len(keys[j]) {
			return keys[i] < keys[j]
		}
		return len(keys[i]) > len(keys[j])
	})
	for _, key := range keys {
		capture, matched := matchTSAlias(key, specifier)
		if !matched {
			continue
		}
		target, ok := selectPackageExportValue(value[key], ".", runtime)
		if ok {
			return strings.ReplaceAll(target, "*", capture), true
		}
	}
	return "", false
}

func effectiveProjectRuntime(project Project) string {
	if project.Runtime == "esm" || project.Runtime == "cjs" {
		return project.Runtime
	}
	if strings.EqualFold(project.PackageType, "module") {
		return "esm"
	}
	if strings.EqualFold(project.PackageType, "commonjs") || strings.EqualFold(project.CompilerOptions.Module, "commonjs") {
		return "cjs"
	}
	return "auto"
}

func runtimeForTSObservation(project Project, observation tsImportObservation) string {
	if project.Runtime == "esm" || project.Runtime == "cjs" {
		return project.Runtime
	}
	if effective := effectiveProjectRuntime(project); effective != "auto" {
		return effective
	}
	switch observation.Kind {
	case "require":
		return "cjs"
	case "import", "type_import", "reexport", "dynamic_import":
		return "esm"
	default:
		return "auto"
	}
}

func selectPackageExportValue(value any, subpath, runtime string) (string, bool) {
	switch typed := value.(type) {
	case string:
		return typed, true
	case []any:
		for _, candidate := range typed {
			if result, ok := selectPackageExportValue(candidate, subpath, runtime); ok {
				return result, true
			}
		}
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		hasSubpaths := false
		for _, key := range keys {
			if strings.HasPrefix(key, ".") {
				hasSubpaths = true
				break
			}
		}
		if hasSubpaths {
			sort.Slice(keys, func(i, j int) bool {
				if keys[i] == keys[j] {
					return false
				}
				if keys[i] == subpath {
					return true
				}
				if keys[j] == subpath {
					return false
				}
				if len(keys[i]) == len(keys[j]) {
					return keys[i] < keys[j]
				}
				return len(keys[i]) > len(keys[j])
			})
			for _, key := range keys {
				capture, matched := matchTSAlias(key, subpath)
				if !matched {
					continue
				}
				target, ok := selectPackageExportValue(typed[key], subpath, runtime)
				if ok {
					return strings.ReplaceAll(target, "*", capture), true
				}
			}
			return "", false
		}
		for _, key := range packageConditionOrder(runtime) {
			if candidate, ok := typed[key]; ok {
				if result, selected := selectPackageExportValue(candidate, subpath, runtime); selected {
					return result, true
				}
			}
		}
	}
	return "", false
}

func packageConditionOrder(runtime string) []string {
	if runtime == "cjs" {
		return []string{"types", "require", "node", "default", "import"}
	}
	if runtime == "esm" {
		return []string{"types", "import", "node", "default", "require"}
	}
	return []string{"types", "import", "require", "node", "default"}
}

func tsImportDiagnostic(observation tsImportObservation, code, message, severity string, recoverable bool, metadata map[string]any) *analysis.Diagnostic {
	if metadata == nil {
		metadata = map[string]any{}
	}
	copyMetadata := cloneTSMetadata(metadata)
	copyMetadata["source_reference_id"] = observation.Source.ID
	return &analysis.Diagnostic{Code: code, Severity: severity, Message: message, Path: observation.Source.Path, Location: observation.Source.Start, Recoverable: recoverable, Metadata: copyMetadata}
}

func cloneTSMetadata(value map[string]any) map[string]any {
	result := make(map[string]any, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}
