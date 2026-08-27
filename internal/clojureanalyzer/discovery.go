package clojureanalyzer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

type discoveryResult struct {
	Modules          []analysis.ModuleObservation
	Dependencies     []clojureDependencyObservation
	DynamicLoads     []clojureDynamicObservation
	Polymorphic      []clojurePolymorphicObservation
	SourceReferences []analysis.SourceReference
	Diagnostics      []analysis.Diagnostic
	Files            []clojureFileObservation
}

type clojureFileObservation struct {
	Path              string
	Flavor            string
	SourceID          string
	NamespaceSourceID string
	Namespace         string
	ModuleID          string
	Forms             []*cljForm
	NamespaceForm     *cljForm
}

type clojureModuleAccumulator struct {
	Namespace   string
	SourceIDs   map[string]struct{}
	Paths       map[string]struct{}
	SourceRoots map[string]struct{}
	Flavors     map[string]struct{}
	Metadata    map[string]any
}

type clojureDependencyObservation struct {
	FromModuleID    string
	Source          analysis.SourceReference
	TargetNamespace string
	Kind            string
	Alias           string
	ReferredSymbols []string
	Platforms       []string
	Conditional     bool
	Resolution      string
}

type clojureDynamicObservation struct {
	FromModuleID string
	Source       analysis.SourceReference
	Function     string
	Target       string
}

// Discover walks effective Clojure source roots and builds namespace
// observations. It parses source as data and never loads, evaluates, or
// expands a target namespace.
func Discover(ctx context.Context, project Project, options analysis.EffectiveOptions) (discoveryResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return discoveryResult{}, err
	}
	includeTests := optionBool(options, "include_tests")
	excludePatterns := optionStrings(options, "exclude")
	modules := make(map[string]*clojureModuleAccumulator)
	sources := make(map[string]analysis.SourceReference)
	files := make(map[string]clojureFileObservation)
	result := discoveryResult{Diagnostics: append([]analysis.Diagnostic{}, project.ConfigDiagnostics...)}

	for _, sourceRoot := range project.SourceRoots {
		if err := ctx.Err(); err != nil {
			return discoveryResult{}, err
		}
		walkErr := filepath.WalkDir(sourceRoot.Absolute, func(filePath string, entry fs.DirEntry, walkErr error) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if walkErr != nil {
				result.Diagnostics = append(result.Diagnostics, unreadableClojureDiagnostic(project.Root, filePath, walkErr))
				return nil
			}
			if entry == nil {
				return nil
			}
			relativeProject := relativeProjectPath(project.Root, filePath)
			if entry.IsDir() {
				if excludedClojureDirectory(relativeProject, excludePatterns) {
					return fs.SkipDir
				}
				return nil
			}
			if !isClojureSourceFile(entry.Name()) || isExcludedClojureFile(relativeProject, entry.Name(), includeTests, excludePatterns) {
				return nil
			}
			if !safeFileWithinRoot(project.Root, filePath) {
				result.Diagnostics = append(result.Diagnostics, analysis.Diagnostic{
					Code:        "clojure_path_outside_root",
					Severity:    "warning",
					Message:     "Clojure source file resolves outside the selected project root and was ignored.",
					Path:        relativeProject,
					Recoverable: true,
				})
				return nil
			}
			if _, alreadySeen := files[relativeProject]; alreadySeen {
				return nil
			}
			content, err := os.ReadFile(filePath)
			if err != nil {
				result.Diagnostics = append(result.Diagnostics, unreadableClojureDiagnostic(project.Root, filePath, err))
				return nil
			}
			flavor := clojureFlavor(entry.Name())
			if !clojureFlavorSelected(flavor, project.Platform) {
				return nil
			}
			parsed := parseClojureNamespaceFile(string(content))
			for _, issue := range parsed.SyntaxIssues {
				result.Diagnostics = append(result.Diagnostics, analysis.Diagnostic{
					Code:        "clojure_syntax_error",
					Severity:    "warning",
					Message:     "Clojure source contains malformed data-shaped syntax; parseable observations were retained.",
					Path:        relativeProject,
					Location:    &analysis.Position{Line: issue.Line, Column: issue.Column},
					Recoverable: true,
				})
			}
			fileSourceID := clojureStableID("file", relativeProject)
			fileSource := analysis.SourceReference{ID: fileSourceID, Path: relativeProject, Symbol: parsed.Namespace, Kind: "file"}
			sources[fileSourceID] = fileSource
			file := clojureFileObservation{
				Path:          relativeProject,
				Flavor:        flavor,
				SourceID:      fileSourceID,
				Namespace:     parsed.Namespace,
				NamespaceForm: parsed.NamespaceForm,
				Forms:         parsed.Forms,
			}
			if parsed.Namespace == "" {
				result.Diagnostics = append(result.Diagnostics, analysis.Diagnostic{
					Code:        parsed.NamespaceDiagnosticCode,
					Severity:    "warning",
					Message:     parsed.NamespaceDiagnosticMessage,
					Path:        relativeProject,
					Location:    parsed.NamespaceDiagnosticLocation,
					Recoverable: true,
				})
				files[relativeProject] = file
				return nil
			}

			moduleID := clojureModuleID(parsed.Namespace)
			file.ModuleID = moduleID
			file.NamespaceSourceID = clojureStableID("namespace", relativeProject, parsed.Namespace)
			namespaceSource := analysis.SourceReference{
				ID:     file.NamespaceSourceID,
				Path:   relativeProject,
				Start:  clonePosition(parsed.NamespaceForm.Start),
				End:    clonePosition(parsed.NamespaceForm.End),
				Symbol: parsed.Namespace,
				Kind:   "namespace",
			}
			sources[file.NamespaceSourceID] = namespaceSource
			fileSource.Symbol = parsed.Namespace
			sources[fileSourceID] = fileSource
			files[relativeProject] = file

			module := modules[parsed.Namespace]
			if module == nil {
				module = &clojureModuleAccumulator{
					Namespace:   parsed.Namespace,
					SourceIDs:   make(map[string]struct{}),
					Paths:       make(map[string]struct{}),
					SourceRoots: make(map[string]struct{}),
					Flavors:     make(map[string]struct{}),
					Metadata:    map[string]any{},
				}
				modules[parsed.Namespace] = module
			}
			module.SourceIDs[fileSourceID] = struct{}{}
			module.SourceIDs[file.NamespaceSourceID] = struct{}{}
			module.Paths[relativeProject] = struct{}{}
			module.SourceRoots[sourceRoot.Relative] = struct{}{}
			module.Flavors[flavor] = struct{}{}
			result.Files = append(result.Files, file)
			return nil
		})
		if walkErr != nil {
			if ctx.Err() != nil {
				return discoveryResult{}, ctx.Err()
			}
			result.Diagnostics = append(result.Diagnostics, unreadableClojureDiagnostic(project.Root, sourceRoot.Absolute, walkErr))
		}
	}

	dependencies, dependencySources, dependencyDiagnostics := extractClojureDependencies(project, result.Files)
	result.Dependencies = dependencies
	result.Diagnostics = append(result.Diagnostics, dependencyDiagnostics...)
	for _, source := range dependencySources {
		sources[source.ID] = source
	}
	polymorphic, dynamic, safetySources, safetyDiagnostics := extractClojureSafetyObservations(project.Platform, result.Files)
	result.Polymorphic = polymorphic
	result.DynamicLoads = dynamic
	result.Diagnostics = append(result.Diagnostics, safetyDiagnostics...)
	for _, source := range safetySources {
		sources[source.ID] = source
	}
	result.Modules = buildClojureModuleObservations(project, modules)
	buildClojurePolymorphicMetadata(result.Modules, result.Polymorphic)
	result.SourceReferences = sortedClojureSources(sources)
	sort.Slice(result.Files, func(i, j int) bool { return result.Files[i].Path < result.Files[j].Path })
	sortClojureDiagnostics(result.Diagnostics)
	return result, nil
}

type parsedClojureNamespaceFile struct {
	Forms                       []*cljForm
	Namespace                   string
	NamespaceForm               *cljForm
	SyntaxIssues                []syntaxIssue
	NamespaceDiagnosticCode     string
	NamespaceDiagnosticMessage  string
	NamespaceDiagnosticLocation *analysis.Position
}

func parseClojureNamespaceFile(content string) parsedClojureNamespaceFile {
	forms, issues := parseClojureForms(content)
	result := parsedClojureNamespaceFile{Forms: forms, SyntaxIssues: issues}
	for _, form := range forms {
		if form == nil || form.Quoted || form.Kind != formList || len(form.Items) == 0 || formSymbol(form.Items[0]) != "ns" {
			continue
		}
		result.NamespaceForm = form
		if len(form.Items) < 2 {
			result.NamespaceDiagnosticCode = "clojure_malformed_ns"
			result.NamespaceDiagnosticMessage = "Clojure ns form has no namespace symbol; no module was fabricated from the file path."
			result.NamespaceDiagnosticLocation = clonePosition(form.Start)
			return result
		}
		namespace := formSymbol(form.Items[1])
		if !validClojureNamespace(namespace) {
			result.NamespaceDiagnosticCode = "clojure_malformed_ns"
			result.NamespaceDiagnosticMessage = "Clojure ns form has an invalid namespace symbol; no module was fabricated from the file path."
			result.NamespaceDiagnosticLocation = clonePosition(form.Items[1].Start)
			return result
		}
		result.Namespace = namespace
		return result
	}
	result.NamespaceDiagnosticCode = "clojure_missing_ns"
	result.NamespaceDiagnosticMessage = "Clojure source file has no top-level ns form; no module was fabricated from the file path."
	if len(forms) > 0 {
		result.NamespaceDiagnosticLocation = clonePosition(forms[0].Start)
	}
	return result
}

func buildClojureModuleObservations(project Project, modules map[string]*clojureModuleAccumulator) []analysis.ModuleObservation {
	namespaces := make([]string, 0, len(modules))
	for namespace := range modules {
		namespaces = append(namespaces, namespace)
	}
	sort.Strings(namespaces)
	result := make([]analysis.ModuleObservation, 0, len(namespaces))
	for _, namespace := range namespaces {
		value := modules[namespace]
		paths := sortedClojureSet(value.Paths)
		roots := sortedClojureSet(value.SourceRoots)
		flavors := sortedClojureSet(value.Flavors)
		sourceIDs := sortedClojureSet(value.SourceIDs)
		metadata := map[string]any{
			"qualified_name":      namespace,
			"namespace":           namespace,
			"flavors":             flavors,
			"paths":               paths,
			"relative_paths":      append([]string(nil), paths...),
			"file_count":          len(paths),
			"source_roots":        roots,
			"configuration_files": append([]string(nil), project.ConfigurationFiles...),
			"platform":            project.Platform,
		}
		for key, value := range value.Metadata {
			metadata[key] = value
		}
		tags := []string{"namespace"}
		result = append(result, analysis.ModuleObservation{
			ID:                 clojureModuleID(namespace),
			Language:           "clojure",
			Kind:               "namespace",
			Name:               clojureDisplayName(namespace),
			DisplayName:        namespace,
			Hierarchy:          strings.Split(namespace, "."),
			SourceReferenceIDs: sourceIDs,
			Tags:               tags,
			Metadata:           metadata,
		})
	}
	return result
}

func isClojureSourceFile(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".clj", ".cljs", ".cljc":
		return true
	default:
		return false
	}
}

func clojureFlavor(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".cljs":
		return "cljs"
	case ".cljc":
		return "cljc"
	default:
		return "clj"
	}
}

func clojureFlavorSelected(flavor, platform string) bool {
	if platform == "both" || platform == "" {
		return true
	}
	return flavor == platform || flavor == "cljc"
}

func excludedClojureDirectory(relative string, patterns []string) bool {
	if isDefaultClojureExcluded(relative) || matchesClojureExclusion(relative, patterns) {
		return true
	}
	return false
}

func isDefaultClojureExcluded(relative string) bool {
	for _, part := range strings.Split(filepath.ToSlash(relative), "/") {
		switch strings.ToLower(part) {
		case ".git", ".cpcache", ".shadow-cljs", "target", "build", "dist", "generated", "out", "tmp", "vendor", "external", "node_modules":
			return true
		}
	}
	return false
}

func isExcludedClojureFile(relative, name string, includeTests bool, patterns []string) bool {
	if matchesClojureExclusion(relative, patterns) || matchesClojureExclusion(name, patterns) {
		return true
	}
	if includeTests {
		return false
	}
	return isClojureTestPath(relative, name)
}

func isClojureTestPath(relative, name string) bool {
	parts := strings.Split(strings.ToLower(filepath.ToSlash(relative)), "/")
	for _, part := range parts {
		if part == "test" || part == "tests" || part == "test-resources" {
			return true
		}
	}
	base := strings.ToLower(strings.TrimSuffix(name, filepath.Ext(name)))
	return strings.HasSuffix(base, "-test") || strings.HasSuffix(base, "_test") || strings.HasSuffix(base, "-tests") || strings.HasSuffix(base, "_tests")
}

func matchesClojureExclusion(value string, patterns []string) bool {
	clean := filepath.ToSlash(filepath.Clean(value))
	for _, raw := range patterns {
		pattern := strings.TrimPrefix(filepath.ToSlash(filepath.Clean(raw)), "./")
		if pattern == "" || pattern == "." {
			continue
		}
		if clean == strings.TrimSuffix(pattern, "/**") {
			return true
		}
		if matched, _ := path.Match(pattern, clean); matched {
			return true
		}
		if strings.HasSuffix(pattern, "/**") && strings.HasPrefix(clean, strings.TrimSuffix(pattern, "/**")+"/") {
			return true
		}
		if strings.HasPrefix(pattern, "**/") {
			short := strings.TrimPrefix(pattern, "**/")
			if matched, _ := path.Match(short, path.Base(clean)); matched {
				return true
			}
		}
	}
	return false
}

func safeFileWithinRoot(root, candidate string) bool {
	info, err := os.Stat(candidate)
	if err != nil || info.IsDir() {
		return false
	}
	return resolvedPathWithin(root, candidate)
}

func unreadableClojureDiagnostic(root, filePath string, err error) analysis.Diagnostic {
	return analysis.Diagnostic{
		Code:        "clojure_source_unreadable",
		Severity:    "warning",
		Message:     fmt.Sprintf("Clojure source %q could not be read: %v", relativeProjectPath(root, filePath), err),
		Path:        relativeProjectPath(root, filePath),
		Recoverable: true,
	}
}

func clojureStableID(parts ...string) string {
	payload := strings.Join(parts, "\x00")
	sum := sha256.Sum256([]byte(payload))
	return "clj:" + hex.EncodeToString(sum[:12])
}

func clojureModuleID(namespace string) string {
	return "clj:" + namespace
}

func clojureDisplayName(namespace string) string {
	if index := strings.LastIndexByte(namespace, '.'); index >= 0 && index+1 < len(namespace) {
		return namespace[index+1:]
	}
	return namespace
}

func clonePosition(position *analysis.Position) *analysis.Position {
	if position == nil {
		return nil
	}
	copy := *position
	return &copy
}

func sortedClojureSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func sortedClojureSources(values map[string]analysis.SourceReference) []analysis.SourceReference {
	result := make([]analysis.SourceReference, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func sortClojureDiagnostics(values []analysis.Diagnostic) {
	sort.Slice(values, func(i, j int) bool {
		left := values[i].Code + "\x00" + values[i].Path + "\x00" + values[i].Message
		right := values[j].Code + "\x00" + values[j].Path + "\x00" + values[j].Message
		if left == right && values[i].Location != nil && values[j].Location != nil {
			if values[i].Location.Line != values[j].Location.Line {
				return values[i].Location.Line < values[j].Location.Line
			}
			return values[i].Location.Column < values[j].Location.Column
		}
		return left < right
	})
}
