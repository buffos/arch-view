package clojureanalyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

// SourceRoot is an effective, repository-contained Clojure source root.
type SourceRoot struct {
	Absolute string
	Relative string
}

// Project contains the resolved Clojure-family boundary and static discovery
// policy. It deliberately contains no classpath, runtime, or loaded namespace
// state.
type Project struct {
	Root               string
	Boundary           string
	ConfigurationFiles []string
	SourceRoots        []SourceRoot
	Platform           string
	ConfigDiagnostics  []analysis.Diagnostic
}

type clojureConfiguration struct {
	SourceRoots []string
	TestRoots   []string
	Diagnostics []analysis.Diagnostic
}

// ResolveProject selects the preferred Clojure-family marker, reads only the
// marker's data-shaped configuration, and resolves safe source roots.
func ResolveProject(root string, options analysis.EffectiveOptions) (Project, error) {
	absoluteRoot, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return Project{}, analysis.WrapHostError(analysis.ErrInvalidRequest, "Clojure project root could not be normalized", err, nil)
	}
	absoluteRoot = filepath.Clean(absoluteRoot)
	info, err := os.Stat(absoluteRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return Project{}, analysis.NewHostError(analysis.ErrUnreadableProject, "Clojure project root does not exist", map[string]any{"project_root": root})
		}
		return Project{}, analysis.WrapHostError(analysis.ErrUnreadableProject, "Clojure project root could not be read", err, map[string]any{"project_root": root})
	}
	if !info.IsDir() {
		return Project{}, analysis.NewHostError(analysis.ErrInvalidRequest, "Clojure project root must be a directory", map[string]any{"project_root": root})
	}

	platform := optionString(options, "platform")
	if platform == "" {
		platform = "both"
	}
	if platform != "clj" && platform != "cljs" && platform != "both" {
		return Project{}, analysis.NewHostError(analysis.ErrInvalidOptions, "Clojure platform must be clj, cljs, or both", map[string]any{"platform": platform})
	}

	boundary := preferredBoundary(absoluteRoot)
	if boundary == "" {
		return Project{}, analysis.NewHostError(analysis.ErrUnsupportedProject, "Clojure project requires deps.edn, project.clj, or shadow-cljs.edn at the selected root", map[string]any{"project_root": root})
	}
	config := readConfiguration(absoluteRoot, boundary)
	diagnostics := append([]analysis.Diagnostic{}, config.Diagnostics...)

	explicitRoots := optionStrings(options, "source_roots")
	rootValues := append([]string(nil), explicitRoots...)
	rootSource := "explicit analyzer options"
	if len(rootValues) == 0 {
		rootValues = append([]string(nil), config.SourceRoots...)
		rootSource = "project configuration"
	}
	if len(rootValues) == 0 {
		if directoryExists(filepath.Join(absoluteRoot, "src")) {
			rootValues = []string{"src"}
			rootSource = "src layout default"
		} else {
			rootValues = []string{"."}
			rootSource = "project root default"
		}
	}
	if len(explicitRoots) == 0 && len(config.TestRoots) > 0 && optionBool(options, "include_tests") {
		rootValues = append(rootValues, config.TestRoots...)
	}

	roots := make([]SourceRoot, 0, len(rootValues))
	for _, value := range rootValues {
		sourceRoot, rootErr := normalizeSourceRoot(absoluteRoot, value)
		if rootErr != nil {
			diagnostics = append(diagnostics, sourceRootDiagnostic(value, rootErr))
			continue
		}
		if !containsSourceRoot(roots, sourceRoot.Absolute) {
			roots = append(roots, sourceRoot)
		}
	}
	if len(roots) == 0 {
		fallback, fallbackErr := normalizeSourceRoot(absoluteRoot, ".")
		if fallbackErr == nil {
			roots = append(roots, fallback)
			diagnostics = append(diagnostics, analysis.Diagnostic{
				Code:        "clojure_source_root_fallback",
				Severity:    "warning",
				Message:     fmt.Sprintf("No usable Clojure source roots were found from %s; the project root was used as a safe fallback.", rootSource),
				Recoverable: true,
			})
		}
	}
	sort.Slice(roots, func(i, j int) bool { return roots[i].Relative < roots[j].Relative })

	return Project{
		Root:               absoluteRoot,
		Boundary:           boundary,
		ConfigurationFiles: []string{boundary},
		SourceRoots:        roots,
		Platform:           platform,
		ConfigDiagnostics:  diagnostics,
	}, nil
}

func preferredBoundary(root string) string {
	for _, marker := range []string{"deps.edn", "project.clj", "shadow-cljs.edn"} {
		if safeProjectFile(root, marker) {
			return marker
		}
	}
	return ""
}

func safeProjectFile(root, relative string) bool {
	filePath := filepath.Join(root, filepath.FromSlash(relative))
	return existsAsFile(filePath) && resolvedPathWithin(root, filePath)
}

func normalizeSourceRoot(root, value string) (SourceRoot, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return SourceRoot{}, fmt.Errorf("source root is empty")
	}
	absolute := trimmed
	if !filepath.IsAbs(absolute) {
		absolute = filepath.Join(root, absolute)
	}
	absolute = filepath.Clean(absolute)
	if !pathWithin(root, absolute) {
		return SourceRoot{}, fmt.Errorf("source root escapes the selected project root")
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return SourceRoot{}, fmt.Errorf("source root could not be read: %w", err)
	}
	if !info.IsDir() {
		return SourceRoot{}, fmt.Errorf("source root is not a directory")
	}
	if !resolvedPathWithin(root, absolute) {
		return SourceRoot{}, fmt.Errorf("source root resolves outside the selected project root")
	}
	relative, err := filepath.Rel(root, absolute)
	if err != nil {
		return SourceRoot{}, fmt.Errorf("source root could not be made repository-relative: %w", err)
	}
	if relative == "." {
		return SourceRoot{Absolute: absolute, Relative: "."}, nil
	}
	return SourceRoot{Absolute: absolute, Relative: filepath.ToSlash(relative)}, nil
}

func resolvedPathWithin(root, candidate string) bool {
	resolvedRoot, rootErr := filepath.EvalSymlinks(root)
	resolvedCandidate, candidateErr := filepath.EvalSymlinks(candidate)
	if rootErr != nil || candidateErr != nil {
		return false
	}
	return pathWithin(filepath.Clean(resolvedRoot), filepath.Clean(resolvedCandidate))
}

func pathWithin(root, candidate string) bool {
	relative, err := filepath.Rel(filepath.Clean(root), filepath.Clean(candidate))
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

func sourceRootDiagnostic(value string, err error) analysis.Diagnostic {
	return analysis.Diagnostic{
		Code:        "clojure_source_root_invalid",
		Severity:    "warning",
		Message:     fmt.Sprintf("Clojure source root %q was ignored: %v", value, err),
		Subject:     value,
		Recoverable: true,
	}
}

func containsSourceRoot(roots []SourceRoot, absolute string) bool {
	for _, root := range roots {
		if root.Absolute == absolute {
			return true
		}
	}
	return false
}

func optionString(options analysis.EffectiveOptions, name string) string {
	value, ok := options.Values[name]
	if !ok || value == nil {
		return ""
	}
	typed, _ := value.(string)
	return strings.TrimSpace(typed)
}

func optionStrings(options analysis.EffectiveOptions, name string) []string {
	value, _ := options.Values[name].([]string)
	return append([]string(nil), value...)
}

func optionBool(options analysis.EffectiveOptions, name string) bool {
	value, _ := options.Values[name].(bool)
	return value
}

func directoryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func readConfiguration(root, boundary string) clojureConfiguration {
	path := filepath.Join(root, filepath.FromSlash(boundary))
	content, err := os.ReadFile(path)
	if err != nil {
		return clojureConfiguration{Diagnostics: []analysis.Diagnostic{unreadableConfigurationDiagnostic(boundary, err)}}
	}
	forms, issues := parseClojureForms(string(content))
	diagnostics := make([]analysis.Diagnostic, 0, len(issues))
	for _, issue := range issues {
		diagnostics = append(diagnostics, analysis.Diagnostic{
			Code:        "clojure_configuration_syntax",
			Severity:    "warning",
			Message:     "Clojure project configuration contains malformed data-shaped syntax; usable paths were retained.",
			Path:        filepath.ToSlash(boundary),
			Location:    &analysis.Position{Line: issue.Line, Column: issue.Column},
			Recoverable: true,
		})
	}
	var sourceKeys []string
	switch boundary {
	case "deps.edn":
		sourceKeys = []string{"paths", "extra-paths"}
	case "project.clj", "shadow-cljs.edn":
		sourceKeys = []string{"source-paths", "extra-paths"}
	}
	sources := collectConfigPaths(forms, sourceKeys)
	tests := collectConfigPaths(forms, []string{"test-paths", "test-path", "test_paths"})
	return clojureConfiguration{SourceRoots: dedupeStrings(sources), TestRoots: dedupeStrings(tests), Diagnostics: diagnostics}
}

func unreadableConfigurationDiagnostic(path string, err error) analysis.Diagnostic {
	return analysis.Diagnostic{
		Code:        "clojure_configuration_unreadable",
		Severity:    "warning",
		Message:     fmt.Sprintf("Clojure project configuration %q could not be read: %v", path, err),
		Path:        filepath.ToSlash(path),
		Recoverable: true,
	}
}

func collectConfigPaths(forms []*cljForm, keys []string) []string {
	keySet := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		keySet[":"+strings.TrimPrefix(key, ":")] = struct{}{}
	}
	result := make([]string, 0)
	var visit func(*cljForm)
	visit = func(form *cljForm) {
		if form == nil {
			return
		}
		if form.Kind == formMap || form.Kind == formList {
			for index := 0; index+1 < len(form.Items); index++ {
				key := formKeyword(form.Items[index])
				if _, ok := keySet[key]; ok {
					result = append(result, configStringValues(form.Items[index+1])...)
				}
			}
		}
		for _, child := range form.Items {
			visit(child)
		}
	}
	for _, form := range forms {
		visit(form)
	}
	return result
}

func configStringValues(form *cljForm) []string {
	if form == nil {
		return nil
	}
	if value, ok := formString(form); ok {
		return []string{value}
	}
	if form.Kind != formVector && form.Kind != formList && form.Kind != formSet {
		return nil
	}
	result := make([]string, 0, len(form.Items))
	for _, child := range form.Items {
		if value, ok := formString(child); ok {
			result = append(result, value)
		}
	}
	return result
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func relativeProjectPath(root, filePath string) string {
	relative, err := filepath.Rel(root, filePath)
	if err != nil {
		return filepath.ToSlash(filePath)
	}
	if relative == "." {
		return "."
	}
	return filepath.ToSlash(relative)
}
