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

// CompilerOptions contains the statically relevant subset of effective
// TypeScript compiler options. Paths are absolute and are used only after
// they have been checked to remain inside the selected repository.
type CompilerOptions struct {
	BaseURL          string
	Paths            map[string][]string
	RootDir          string
	RootDirs         []string
	OutDir           string
	AllowJS          bool
	Module           string
	ModuleResolution string
	JSX              string
	ResolveJSON      bool
}

// SourcePattern retains the config file directory that gives an include or
// exclude pattern its meaning under tsconfig inheritance.
type SourcePattern struct {
	Value   string
	BaseDir string
}

// Project is the resolved TypeScript project boundary and static scope.
type Project struct {
	Root               string
	Boundary           string
	ConfigPath         string
	ConfigAbsolutePath string
	ConfigDirectory    string
	ConfigurationFiles []string
	ExtendsChain       []string
	CompilerOptions    CompilerOptions
	Include            []SourcePattern
	HasInclude         bool
	Exclude            []SourcePattern
	Files              []string
	HasFiles           bool
	References         []string
	SourceRoots        []string
	WalkRoots          []string
	IncludeJavaScript  bool
	Runtime            string
	PackageName        string
	PackageType        string
	PackageRoot        string
	PackageJSONPath    string
	ConfigDiagnostics  []analysis.Diagnostic
}

type effectiveConfig struct {
	CompilerOptions    CompilerOptions
	Include            []SourcePattern
	HasInclude         bool
	Exclude            []SourcePattern
	HasExclude         bool
	Files              []string
	HasFiles           bool
	References         []string
	ExtendsChain       []string
	ConfigurationFiles []string
	Diagnostics        []analysis.Diagnostic
}

type rawTSConfig struct {
	Extends         string
	ExtendsInvalid  bool
	CompilerOptions map[string]any
	CompilerInvalid bool
	Include         []string
	HasInclude      bool
	Exclude         []string
	HasExclude      bool
	Files           []string
	HasFiles        bool
	References      []string
	HasReferences   bool
}

// ResolveProject selects one tsconfig project, resolves its extends chain,
// and reads package metadata as static context. It never executes package
// scripts or a compiler.
func ResolveProject(root string, options analysis.EffectiveOptions) (Project, error) {
	absoluteRoot, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return Project{}, analysis.WrapHostError(analysis.ErrInvalidRequest, "TypeScript project root could not be normalized", err, nil)
	}
	absoluteRoot = filepath.Clean(absoluteRoot)
	info, err := os.Stat(absoluteRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return Project{}, analysis.NewHostError(analysis.ErrUnreadableProject, "TypeScript project root does not exist", map[string]any{"project_root": root})
		}
		return Project{}, analysis.WrapHostError(analysis.ErrUnreadableProject, "TypeScript project root could not be read", err, map[string]any{"project_root": root})
	}
	if !info.IsDir() {
		return Project{}, analysis.NewHostError(analysis.ErrInvalidRequest, "TypeScript project root must be a directory", map[string]any{"project_root": root})
	}

	configSelector := optionString(options, "config")
	configPath := ""
	if configSelector != "" {
		configPath, err = resolveConfigSelector(absoluteRoot, configSelector)
		if err != nil {
			return Project{}, err
		}
	} else {
		candidates, candidateErr := discoverConfigCandidates(absoluteRoot)
		if candidateErr != nil {
			return Project{}, candidateErr
		}
		switch len(candidates) {
		case 0:
			return Project{}, analysis.NewHostError(analysis.ErrUnsupportedProject, "TypeScript project requires a readable tsconfig.json or an explicit TypeScript config path", map[string]any{"project_root": root})
		case 1:
			configPath = candidates[0]
		default:
			relative := make([]string, 0, len(candidates))
			for _, candidate := range candidates {
				relative = append(relative, relativeProjectPath(absoluteRoot, candidate))
			}
			return Project{}, analysis.NewHostError(analysis.ErrModuleSelection, "multiple TypeScript project configs were found; an explicit config selection is required", map[string]any{"configs": relative})
		}
	}

	resolved, loadErr := loadEffectiveConfig(absoluteRoot, configPath, map[string]bool{})
	if loadErr != nil {
		return Project{}, loadErr
	}
	configRelative := relativeProjectPath(absoluteRoot, configPath)
	configDirectory := filepath.Dir(configPath)
	packageInfo, packageDiagnostics := readPackageContext(absoluteRoot, configDirectory)
	diagnostics := append([]analysis.Diagnostic{}, resolved.Diagnostics...)
	diagnostics = append(diagnostics, packageDiagnostics...)

	runtime := optionString(options, "runtime")
	if runtime == "" {
		runtime = "auto"
	}
	if runtime != "auto" && runtime != "esm" && runtime != "cjs" {
		diagnostics = append(diagnostics, analysis.Diagnostic{
			Code:        "typescript_invalid_runtime",
			Severity:    "warning",
			Message:     fmt.Sprintf("TypeScript runtime %q is unsupported; automatic package-export conditions were used.", runtime),
			Subject:     runtime,
			Recoverable: true,
		})
		runtime = "auto"
	}

	include := resolved.Include
	hasInclude := resolved.HasInclude
	if !resolved.HasInclude && !resolved.HasFiles {
		include = []SourcePattern{{Value: "**/*", BaseDir: configDirectory}}
		hasInclude = true
	}
	exclude := append([]SourcePattern{}, resolved.Exclude...)
	for _, pattern := range optionStrings(options, "exclude") {
		if normalized, ok := normalizePatternValue(absoluteRoot, absoluteRoot, pattern); ok {
			exclude = append(exclude, SourcePattern{Value: normalized, BaseDir: absoluteRoot})
		} else {
			diagnostics = append(diagnostics, analysis.Diagnostic{
				Code:        "typescript_exclude_invalid",
				Severity:    "warning",
				Message:     fmt.Sprintf("TypeScript exclusion %q was ignored because it escapes the project root.", pattern),
				Subject:     pattern,
				Recoverable: true,
			})
		}
	}

	sourceRoots := effectiveSourceRoots(absoluteRoot, configDirectory, resolved.CompilerOptions)
	project := Project{
		Root:               absoluteRoot,
		Boundary:           configRelative,
		ConfigPath:         configRelative,
		ConfigAbsolutePath: configPath,
		ConfigDirectory:    configDirectory,
		ConfigurationFiles: append([]string{}, resolved.ConfigurationFiles...),
		ExtendsChain:       append([]string{}, resolved.ExtendsChain...),
		CompilerOptions:    resolved.CompilerOptions,
		Include:            include,
		HasInclude:         hasInclude,
		Exclude:            exclude,
		Files:              relativeFiles(absoluteRoot, resolved.Files),
		HasFiles:           resolved.HasFiles,
		References:         append([]string{}, resolved.References...),
		SourceRoots:        sourceRoots,
		WalkRoots:          []string{"."},
		IncludeJavaScript:  optionBool(options, "include_js") || resolved.CompilerOptions.AllowJS,
		Runtime:            runtime,
		PackageName:        packageInfo.Name,
		PackageType:        packageInfo.Type,
		PackageRoot:        packageInfo.Root,
		PackageJSONPath:    packageInfo.Path,
		ConfigDiagnostics:  diagnostics,
	}
	return project, nil
}

type packageContext struct {
	Name string
	Type string
	Root string
	Path string
}

func readPackageContext(root, startDirectory string) (packageContext, []analysis.Diagnostic) {
	diagnostics := make([]analysis.Diagnostic, 0)
	for directory := filepath.Clean(startDirectory); pathWithin(root, directory); directory = filepath.Dir(directory) {
		packagePath := filepath.Join(directory, "package.json")
		if existsAsFile(packagePath) {
			if !safeFileWithinRoot(root, packagePath) {
				diagnostics = append(diagnostics, analysis.Diagnostic{
					Code:        "typescript_package_json_outside_root",
					Severity:    "warning",
					Message:     "package.json resolves outside the selected project root and was ignored.",
					Path:        relativeProjectPath(root, packagePath),
					Recoverable: true,
				})
				continue
			}
			data, err := os.ReadFile(packagePath)
			if err != nil {
				return packageContext{Root: directory, Path: relativeProjectPath(root, packagePath)}, append(diagnostics, analysis.Diagnostic{
					Code:        "typescript_package_json_unreadable",
					Severity:    "warning",
					Message:     fmt.Sprintf("package.json could not be read: %v", err),
					Path:        relativeProjectPath(root, packagePath),
					Recoverable: true,
				})
			}
			var value map[string]any
			if err := json.Unmarshal(data, &value); err != nil {
				return packageContext{Root: directory, Path: relativeProjectPath(root, packagePath)}, append(diagnostics, analysis.Diagnostic{
					Code:        "typescript_package_json_invalid",
					Severity:    "warning",
					Message:     fmt.Sprintf("package.json could not be parsed as JSON: %v", err),
					Path:        relativeProjectPath(root, packagePath),
					Recoverable: true,
				})
			}
			if value == nil {
				return packageContext{Root: directory, Path: relativeProjectPath(root, packagePath)}, append(diagnostics, analysis.Diagnostic{
					Code:        "typescript_package_json_invalid",
					Severity:    "warning",
					Message:     "package.json must contain a JSON object; package context was ignored.",
					Path:        relativeProjectPath(root, packagePath),
					Recoverable: true,
				})
			}
			name, _ := value["name"].(string)
			typeValue, _ := value["type"].(string)
			return packageContext{Name: strings.TrimSpace(name), Type: strings.TrimSpace(typeValue), Root: directory, Path: relativeProjectPath(root, packagePath)}, diagnostics
		}
		if directory == root {
			break
		}
	}
	return packageContext{}, diagnostics
}

func resolveConfigSelector(root, selector string) (string, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "TypeScript config selection cannot be empty", nil)
	}
	path := selector
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, filepath.FromSlash(path))
	}
	path = filepath.Clean(path)
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		path = filepath.Join(path, "tsconfig.json")
	}
	if !pathWithin(root, path) || !resolvedPathWithin(root, path) {
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "selected TypeScript config must remain inside the project root", map[string]any{"config": selector})
	}
	if !existsAsFile(path) {
		if filepath.Ext(path) == "" {
			candidate := path + ".json"
			if existsAsFile(candidate) {
				path = candidate
			}
		}
		if !existsAsFile(path) {
			return "", analysis.NewHostError(analysis.ErrUnreadableProject, "selected TypeScript config could not be read", map[string]any{"config": selector})
		}
	}
	return path, nil
}

func discoverConfigCandidates(root string) ([]string, error) {
	result := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			if path == root {
				return walkErr
			}
			return nil
		}
		relative := relativeProjectPath(root, path)
		if entry.IsDir() {
			if relative != "." && defaultExcludedDirectory(relative) {
				return filepath.SkipDir
			}
			return nil
		}
		name := strings.ToLower(entry.Name())
		if name == "tsconfig.json" || (strings.HasPrefix(name, "tsconfig.") && strings.HasSuffix(name, ".json")) {
			if safeFileWithinRoot(root, path) {
				result = append(result, filepath.Clean(path))
			}
		}
		return nil
	})
	if err != nil {
		return nil, analysis.WrapHostError(analysis.ErrUnreadableProject, "TypeScript project configs could not be scanned", err, map[string]any{"project_root": root})
	}
	sort.Strings(result)
	// A config referenced only through extends is an inherited fragment, not a
	// competing project boundary. Keep it available to the selected config but
	// do not make the common tsconfig.base.json pattern ambiguous.
	referenced := make(map[string]struct{})
	for _, candidate := range result {
		data, readErr := os.ReadFile(candidate)
		if readErr != nil {
			continue
		}
		raw, parseErr := parseTSConfig(data)
		if parseErr != nil || raw.Extends == "" {
			continue
		}
		if parent, resolved := resolveExtendsPath(root, filepath.Dir(candidate), raw.Extends); resolved {
			referenced[tsPathKey(parent)] = struct{}{}
		}
	}
	filtered := make([]string, 0, len(result))
	for _, candidate := range result {
		if _, isFragment := referenced[tsPathKey(candidate)]; !isFragment {
			filtered = append(filtered, candidate)
		}
	}
	if len(filtered) > 0 {
		return filtered, nil
	}
	return result, nil
}

func loadEffectiveConfig(root, path string, stack map[string]bool) (effectiveConfig, error) {
	path = filepath.Clean(path)
	stackKey := tsPathKey(path)
	if stack[stackKey] {
		return effectiveConfig{
			Diagnostics: []analysis.Diagnostic{{
				Code:        "typescript_extends_cycle",
				Severity:    "warning",
				Message:     "TypeScript config extends chain contains a cycle; the repeated config was ignored.",
				Path:        relativeProjectPath(root, path),
				Recoverable: true,
			}},
		}, nil
	}
	stack[stackKey] = true
	defer delete(stack, stackKey)

	current := effectiveConfig{}
	data, err := os.ReadFile(path)
	if err != nil {
		current.Diagnostics = append(current.Diagnostics, analysis.Diagnostic{
			Code:        "typescript_configuration_unreadable",
			Severity:    "warning",
			Message:     fmt.Sprintf("TypeScript configuration could not be read: %v", err),
			Path:        relativeProjectPath(root, path),
			Recoverable: true,
		})
		return current, nil
	}
	raw, parseErr := parseTSConfig(data)
	if parseErr != nil {
		current.Diagnostics = append(current.Diagnostics, analysis.Diagnostic{
			Code:        "typescript_configuration_invalid",
			Severity:    "warning",
			Message:     fmt.Sprintf("TypeScript configuration could not be parsed: %v", parseErr),
			Path:        relativeProjectPath(root, path),
			Recoverable: true,
		})
		raw = rawTSConfig{}
	}
	if raw.ExtendsInvalid {
		current.Diagnostics = append(current.Diagnostics, analysis.Diagnostic{
			Code:        "typescript_extends_invalid",
			Severity:    "warning",
			Message:     "TypeScript config extends must be a string path or package selector; the invalid value was ignored.",
			Path:        relativeProjectPath(root, path),
			Recoverable: true,
		})
	}
	if raw.CompilerInvalid {
		current.Diagnostics = append(current.Diagnostics, analysis.Diagnostic{
			Code:        "typescript_compiler_options_invalid",
			Severity:    "warning",
			Message:     "TypeScript compilerOptions must be an object; the invalid value was ignored.",
			Path:        relativeProjectPath(root, path),
			Recoverable: true,
		})
	}

	configDir := filepath.Dir(path)
	if raw.Extends != "" {
		parentPath, resolved := resolveExtendsPath(root, configDir, raw.Extends)
		if !resolved {
			current.Diagnostics = append(current.Diagnostics, analysis.Diagnostic{
				Code:        "typescript_extends_unresolved",
				Severity:    "warning",
				Message:     fmt.Sprintf("TypeScript config extends %q, but that config could not be resolved safely.", raw.Extends),
				Path:        relativeProjectPath(root, path),
				Subject:     raw.Extends,
				Recoverable: true,
			})
		} else {
			parent, parentErr := loadEffectiveConfig(root, parentPath, stack)
			if parentErr != nil {
				return effectiveConfig{}, parentErr
			}
			current = parent
		}
	}

	compiler, compilerDiagnostics := applyCompilerOptions(root, configDir, current.CompilerOptions, raw.CompilerOptions)
	current.CompilerOptions = compiler
	current.Diagnostics = append(current.Diagnostics, compilerDiagnostics...)
	if raw.HasInclude {
		current.Include = makePatterns(root, configDir, raw.Include, &current.Diagnostics)
		current.HasInclude = true
		current.HasFiles = false
		current.Files = nil
	}
	if raw.HasExclude {
		current.Exclude = makePatterns(root, configDir, raw.Exclude, &current.Diagnostics)
		current.HasExclude = true
	}
	if raw.HasFiles {
		current.Files = resolveConfigFiles(root, configDir, raw.Files, &current.Diagnostics)
		current.HasFiles = true
		current.HasInclude = false
		current.Include = nil
	}
	if raw.HasReferences {
		current.References = resolveProjectReferences(root, configDir, raw.References, &current.Diagnostics)
	}
	relative := relativeProjectPath(root, path)
	current.ExtendsChain = append(current.ExtendsChain, relative)
	current.ConfigurationFiles = append(current.ConfigurationFiles, relative)
	return current, nil
}

func resolveExtendsPath(root, configDir, selector string) (string, bool) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return "", false
	}
	if strings.HasPrefix(selector, ".") || filepath.IsAbs(selector) {
		candidate := selector
		if !filepath.IsAbs(candidate) {
			candidate = filepath.Join(configDir, filepath.FromSlash(candidate))
		}
		candidate = filepath.Clean(candidate)
		candidates := []string{candidate}
		if filepath.Ext(candidate) == "" {
			candidates = append(candidates, candidate+".json")
		}
		for _, value := range candidates {
			if pathWithin(root, value) && resolvedPathWithin(root, value) && existsAsFile(value) {
				return value, true
			}
		}
		return "", false
	}
	for directory := configDir; pathWithin(root, directory); directory = filepath.Dir(directory) {
		packagePath := filepath.Join(directory, "node_modules", filepath.FromSlash(selector))
		candidates := []string{packagePath}
		if filepath.Ext(packagePath) == "" {
			candidates = append(candidates, packagePath+".json")
		}
		for _, value := range candidates {
			if pathWithin(root, value) && resolvedPathWithin(root, value) && existsAsFile(value) {
				return value, true
			}
		}
		if directory == root {
			break
		}
	}
	return "", false
}

func parseTSConfig(data []byte) (rawTSConfig, error) {
	clean := stripJSONC(data)
	if isJSONNull(clean) {
		return rawTSConfig{}, fmt.Errorf("configuration root cannot be null")
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(clean, &object); err != nil {
		return rawTSConfig{}, err
	}
	result := rawTSConfig{}
	if value, ok := object["extends"]; ok {
		if isJSONNull(value) || json.Unmarshal(value, &result.Extends) != nil {
			result.ExtendsInvalid = true
		}
	}
	if value, ok := object["compilerOptions"]; ok {
		if isJSONNull(value) || json.Unmarshal(value, &result.CompilerOptions) != nil {
			result.CompilerInvalid = true
		}
	}
	if value, ok := object["include"]; ok {
		result.HasInclude = true
		if isJSONNull(value) {
			return rawTSConfig{}, fmt.Errorf("include: value cannot be null")
		}
		if err := json.Unmarshal(value, &result.Include); err != nil {
			return rawTSConfig{}, fmt.Errorf("include: %w", err)
		}
	}
	if value, ok := object["exclude"]; ok {
		result.HasExclude = true
		if isJSONNull(value) {
			return rawTSConfig{}, fmt.Errorf("exclude: value cannot be null")
		}
		if err := json.Unmarshal(value, &result.Exclude); err != nil {
			return rawTSConfig{}, fmt.Errorf("exclude: %w", err)
		}
	}
	if value, ok := object["files"]; ok {
		result.HasFiles = true
		if isJSONNull(value) {
			return rawTSConfig{}, fmt.Errorf("files: value cannot be null")
		}
		if err := json.Unmarshal(value, &result.Files); err != nil {
			return rawTSConfig{}, fmt.Errorf("files: %w", err)
		}
	}
	if value, ok := object["references"]; ok {
		result.HasReferences = true
		var references []struct {
			Path string `json:"path"`
		}
		if isJSONNull(value) {
			return rawTSConfig{}, fmt.Errorf("references: value cannot be null")
		}
		if err := json.Unmarshal(value, &references); err != nil {
			return rawTSConfig{}, fmt.Errorf("references: %w", err)
		}
		for _, reference := range references {
			if strings.TrimSpace(reference.Path) != "" {
				result.References = append(result.References, reference.Path)
			}
		}
	}
	return result, nil
}

func applyCompilerOptions(root, configDir string, inherited CompilerOptions, raw map[string]any) (CompilerOptions, []analysis.Diagnostic) {
	result := cloneCompilerOptions(inherited)
	diagnostics := make([]analysis.Diagnostic, 0)
	if result.Paths == nil {
		result.Paths = map[string][]string{}
	}
	get := func(name string) (any, bool) {
		if value, ok := raw[name]; ok {
			return value, true
		}
		keys := make([]string, 0)
		for key := range raw {
			if strings.EqualFold(key, name) {
				keys = append(keys, key)
			}
		}
		sort.Strings(keys)
		if len(keys) > 0 {
			return raw[keys[0]], true
		}
		return nil, false
	}
	invalidOption := func(name string, value any) {
		path := relativeProjectPath(root, configDir)
		if path == "." {
			path = ""
		}
		diagnostics = append(diagnostics, analysis.Diagnostic{
			Code:        "typescript_compiler_option_invalid",
			Severity:    "warning",
			Message:     fmt.Sprintf("TypeScript compiler option %q has unsupported value type %T and was ignored.", name, value),
			Path:        path,
			Subject:     name,
			Recoverable: true,
		})
	}
	if rawValue, present := get("baseUrl"); present {
		value, ok := rawValue.(string)
		if !ok {
			invalidOption("baseUrl", rawValue)
		} else if resolved, valid := safeConfigPath(root, configDir, value); valid {
			result.BaseURL = resolved
		} else {
			diagnostics = append(diagnostics, configPathDiagnostic(root, configDir, "typescript_base_url_invalid", value, "baseUrl escapes the selected project root and was ignored."))
		}
	}
	pathBase := result.BaseURL
	if pathBase == "" {
		pathBase = configDir
	}
	if rawValue, present := get("paths"); present {
		value, ok := rawValue.(map[string]any)
		if !ok {
			invalidOption("paths", rawValue)
		} else {
			result.Paths = map[string][]string{}
			for alias, rawTargets := range value {
				targets, ok := stringValues(rawTargets)
				if !ok {
					diagnostics = append(diagnostics, configPathDiagnostic(root, configDir, "typescript_paths_invalid", alias, "paths entries must contain string target arrays."))
					continue
				}
				for _, target := range targets {
					resolved, valid := safeConfigPathWithBase(root, pathBase, target)
					if !valid {
						diagnostics = append(diagnostics, configPathDiagnostic(root, configDir, "typescript_paths_invalid", target, "a paths target escapes the selected project root and was ignored."))
						continue
					}
					result.Paths[alias] = append(result.Paths[alias], resolved)
				}
			}
		}
	}
	if rawValue, present := get("rootDir"); present {
		value, ok := rawValue.(string)
		if !ok {
			invalidOption("rootDir", rawValue)
		} else if resolved, valid := safeConfigPath(root, configDir, value); valid {
			result.RootDir = resolved
		} else {
			diagnostics = append(diagnostics, configPathDiagnostic(root, configDir, "typescript_root_dir_invalid", value, "rootDir escapes the selected project root and was ignored."))
		}
	}
	if value, present := get("rootDirs"); present {
		if values, valid := stringValues(value); valid {
			result.RootDirs = nil
			for _, item := range values {
				if resolved, inside := safeConfigPath(root, configDir, item); inside {
					result.RootDirs = append(result.RootDirs, resolved)
				} else {
					diagnostics = append(diagnostics, configPathDiagnostic(root, configDir, "typescript_root_dirs_invalid", item, "a rootDirs entry escapes the selected project root and was ignored."))
				}
			}
		} else {
			invalidOption("rootDirs", value)
		}
	}
	if rawValue, present := get("outDir"); present {
		value, ok := rawValue.(string)
		if !ok {
			invalidOption("outDir", rawValue)
		} else if resolved, valid := safeConfigPath(root, configDir, value); valid {
			result.OutDir = resolved
		} else {
			diagnostics = append(diagnostics, configPathDiagnostic(root, configDir, "typescript_out_dir_invalid", value, "outDir escapes the selected project root and was ignored."))
		}
	}
	if value, present := get("allowJs"); present {
		if typed, ok := value.(bool); ok {
			result.AllowJS = typed
		} else {
			invalidOption("allowJs", value)
		}
	}
	if value, present := get("module"); present {
		if typed, ok := value.(string); ok {
			result.Module = strings.TrimSpace(typed)
		} else {
			invalidOption("module", value)
		}
	}
	if value, present := get("moduleResolution"); present {
		if typed, ok := value.(string); ok {
			result.ModuleResolution = strings.TrimSpace(typed)
		} else {
			invalidOption("moduleResolution", value)
		}
	}
	if value, present := get("jsx"); present {
		if typed, ok := value.(string); ok {
			result.JSX = strings.TrimSpace(typed)
		} else {
			invalidOption("jsx", value)
		}
	}
	if value, present := get("resolveJsonModule"); present {
		if typed, ok := value.(bool); ok {
			result.ResolveJSON = typed
		} else {
			invalidOption("resolveJsonModule", value)
		}
	}
	return result, diagnostics
}

func cloneCompilerOptions(value CompilerOptions) CompilerOptions {
	result := value
	result.RootDirs = append([]string{}, value.RootDirs...)
	result.Paths = make(map[string][]string, len(value.Paths))
	for key, values := range value.Paths {
		result.Paths[key] = append([]string{}, values...)
	}
	return result
}

func effectiveSourceRoots(root, configDir string, compiler CompilerOptions) []string {
	paths := append([]string{}, compiler.RootDirs...)
	if len(paths) == 0 && compiler.RootDir != "" {
		paths = append(paths, compiler.RootDir)
	}
	if len(paths) == 0 {
		paths = append(paths, root)
	}
	result := make([]string, 0, len(paths))
	seen := map[string]struct{}{}
	for _, value := range paths {
		relative := relativeProjectPath(root, value)
		if _, ok := seen[tsPathKey(relative)]; ok {
			continue
		}
		seen[tsPathKey(relative)] = struct{}{}
		result = append(result, relative)
	}
	sort.Strings(result)
	return result
}

func resolveConfigFiles(root, baseDir string, values []string, diagnostics *[]analysis.Diagnostic) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
		candidate := filepath.Clean(filepath.Join(baseDir, filepath.FromSlash(normalized)))
		if normalized == "" || filepath.IsAbs(normalized) || (len(normalized) >= 2 && normalized[1] == ':') || !safeConfigReferencePath(root, candidate) {
			*diagnostics = append(*diagnostics, configPathDiagnostic(root, baseDir, "typescript_file_invalid", value, "files entry escapes the selected project root and was ignored."))
			continue
		}
		result = append(result, candidate)
	}
	sort.Strings(result)
	return uniqueStrings(result)
}

func resolveProjectReferences(root, baseDir string, values []string, diagnostics *[]analysis.Diagnostic) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
		candidate := filepath.Clean(filepath.Join(baseDir, filepath.FromSlash(normalized)))
		if normalized == "" || filepath.IsAbs(normalized) || (len(normalized) >= 2 && normalized[1] == ':') || !safeConfigReferencePath(root, candidate) {
			*diagnostics = append(*diagnostics, configPathDiagnostic(root, baseDir, "typescript_project_reference_invalid", value, "project reference escapes the selected project root and was ignored."))
			continue
		}
		if _, err := os.Stat(candidate); err != nil && os.IsNotExist(err) {
			*diagnostics = append(*diagnostics, configPathDiagnostic(root, baseDir, "typescript_project_reference_unavailable", value, "project reference was retained but its configured path is unavailable."))
		}
		result = append(result, relativeProjectPath(root, candidate))
	}
	sort.Strings(result)
	return uniqueStrings(result)
}

func safeConfigReferencePath(root, candidate string) bool {
	if !pathWithin(root, candidate) {
		return false
	}
	if _, err := os.Stat(candidate); err == nil {
		return resolvedPathWithin(root, candidate)
	} else if !os.IsNotExist(err) {
		return false
	}
	if info, err := os.Lstat(candidate); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return false
	}
	for current := filepath.Clean(filepath.Dir(candidate)); ; current = filepath.Dir(current) {
		if _, err := os.Stat(current); err == nil {
			return safeFileWithinRoot(root, current)
		} else if !os.IsNotExist(err) {
			return false
		}
		next := filepath.Dir(current)
		if next == current {
			return false
		}
	}
}

func makePatterns(root, baseDir string, values []string, diagnostics *[]analysis.Diagnostic) []SourcePattern {
	result := make([]SourcePattern, 0, len(values))
	for _, value := range values {
		normalized, ok := normalizePatternValue(root, baseDir, value)
		if !ok {
			*diagnostics = append(*diagnostics, configPathDiagnostic(root, baseDir, "typescript_pattern_invalid", value, "a source pattern escapes the selected project root and was ignored."))
			continue
		}
		result = append(result, SourcePattern{Value: normalized, BaseDir: baseDir})
	}
	return result
}

func normalizePatternValue(root, baseDir, value string) (string, bool) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" {
		return "", false
	}
	if filepath.IsAbs(value) || (len(value) >= 2 && value[1] == ':') {
		return "", false
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.Join(baseDir, filepath.FromSlash(value))))
	if !pathWithin(root, filepath.FromSlash(clean)) {
		return "", false
	}
	// Store the pattern relative to its config directory. The matcher resolves
	// the base directory separately, so preserving the original value is more
	// faithful for wildcard patterns than converting it to an absolute path.
	return filepath.ToSlash(filepath.Clean(value)), true
}

func normalizePatternForPath(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	value = strings.TrimPrefix(value, "./")
	if value == "." {
		return "**/*"
	}
	return value
}

func configPathDiagnostic(root, baseDir, code, subject, message string) analysis.Diagnostic {
	path := relativeProjectPath(root, baseDir)
	if path == "." {
		path = ""
	}
	return analysis.Diagnostic{Code: code, Severity: "warning", Message: message, Subject: subject, Path: path, Recoverable: true}
}

func safeConfigPath(root, baseDir, value string) (string, bool) {
	if strings.ContainsAny(value, "*?[") {
		return "", false
	}
	return safeConfigPathWithBase(root, baseDir, value)
}

func safeConfigPathWithBase(root, baseDir, value string) (string, bool) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" || strings.Contains(value, "*") {
		// A wildcard target is safe to resolve from the base directory. The
		// caller will substitute its wildcard before touching the filesystem.
		if value == "" || filepath.IsAbs(value) || (len(value) >= 2 && value[1] == ':') {
			return "", false
		}
		candidate := filepath.Clean(filepath.Join(baseDir, filepath.FromSlash(value)))
		return candidate, pathWithin(root, candidate)
	}
	if filepath.IsAbs(value) || (len(value) >= 2 && value[1] == ':') {
		return "", false
	}
	candidate := filepath.Clean(filepath.Join(baseDir, filepath.FromSlash(value)))
	return candidate, pathWithin(root, candidate)
}

func stringValues(value any) ([]string, bool) {
	values, ok := value.([]any)
	if !ok {
		if typed, typedOK := value.([]string); typedOK {
			return append([]string{}, typed...), true
		}
		return nil, false
	}
	result := make([]string, 0, len(values))
	for _, item := range values {
		text, ok := item.(string)
		if !ok {
			return nil, false
		}
		result = append(result, text)
	}
	return result, true
}

func optionString(options analysis.EffectiveOptions, name string) string {
	value, ok := options.Values[name]
	if !ok || value == nil {
		return ""
	}
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func optionBool(options analysis.EffectiveOptions, name string) bool {
	value, _ := options.Values[name].(bool)
	return value
}

func optionStrings(options analysis.EffectiveOptions, name string) []string {
	value, _ := options.Values[name].([]string)
	return append([]string{}, value...)
}

func relativeFiles(root string, values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, relativeProjectPath(root, value))
	}
	sort.Strings(result)
	return uniqueStrings(result)
}

func uniqueStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	result := append([]string{}, values...)
	sort.Strings(result)
	write := 1
	for _, value := range result[1:] {
		if value != result[write-1] {
			result[write] = value
			write++
		}
	}
	return result[:write]
}
