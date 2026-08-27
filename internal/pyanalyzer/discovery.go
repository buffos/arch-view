package pyanalyzer

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

type discoveryResult struct {
	Modules          []analysis.ModuleObservation
	SourceReferences []analysis.SourceReference
	Diagnostics      []analysis.Diagnostic
}

type fileObservation struct {
	Path       string
	Qualified  string
	Kind       string
	SourceID   string
	IsStub     bool
	IsTest     bool
	SourceRoot string
}

type packageObservation struct {
	Qualified string
	SourceIDs map[string]struct{}
	Paths     map[string]struct{}
	HasInit   bool
	Tags      map[string]struct{}
}

type moduleObservation struct {
	Qualified string
	SourceIDs map[string]struct{}
	Paths     map[string]struct{}
	Tags      map[string]struct{}
}

// Discover walks effective source roots and builds package/module observations.
// It never follows Python imports, invokes a Python process, or treats a file
// as a graph node independently of its package/module observation.
func Discover(ctx context.Context, project Project, options analysis.EffectiveOptions) (discoveryResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return discoveryResult{}, err
	}
	includeStubs := optionBool(options, "include_stubs")
	includeTests := optionBool(options, "include_tests")
	excludePatterns := optionStrings(options, "exclude")
	packages := make(map[string]*packageObservation)
	modules := make(map[string]*moduleObservation)
	sources := make(map[string]analysis.SourceReference)
	files := make(map[string]fileObservation)
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
				result.Diagnostics = append(result.Diagnostics, unreadableDiagnostic(project.Root, filePath, walkErr))
				return nil
			}
			relativeProject := relativeProjectPath(project.Root, filePath)
			if entry.IsDir() {
				if excludedDirectory(relativeProject, excludePatterns) {
					return fs.SkipDir
				}
				return nil
			}
			if !isPythonFile(entry.Name(), includeStubs) || isExcludedPythonFile(relativeProject, entry.Name(), includeTests, excludePatterns) {
				return nil
			}
			if !safeFileWithinRoot(project.Root, filePath) {
				result.Diagnostics = append(result.Diagnostics, analysis.Diagnostic{
					Code:        "python_path_outside_root",
					Severity:    "warning",
					Message:     "Python source file resolves outside the selected project root and was ignored.",
					Path:        relativeProject,
					Recoverable: true,
				})
				return nil
			}
			content, err := os.ReadFile(filePath)
			if err != nil {
				result.Diagnostics = append(result.Diagnostics, unreadableDiagnostic(project.Root, filePath, err))
				return nil
			}
			if issue := validatePythonSyntax(string(content)); issue != nil {
				result.Diagnostics = append(result.Diagnostics, analysis.Diagnostic{
					Code:        "python_syntax_error",
					Severity:    "error",
					Message:     "Python source has an unsupported or malformed syntax construct; its module evidence was retained.",
					Path:        relativeProject,
					Location:    &analysis.Position{Line: issue.Line, Column: issue.Column},
					Recoverable: true,
				})
			}
			qualified, kind := qualifiedName(sourceRoot.Absolute, filePath, entry.Name())
			if qualified == "" {
				return nil
			}
			isStub := strings.EqualFold(filepath.Ext(entry.Name()), ".pyi")
			file := fileObservation{
				Path:       relativeProject,
				Qualified:  qualified,
				Kind:       kind,
				SourceID:   stableID("file", relativeProject),
				IsStub:     isStub,
				IsTest:     isTestPath(relativeProject, entry.Name()),
				SourceRoot: sourceRoot.Relative,
			}
			if existing, ok := files[relativeProject]; ok {
				if existing.Qualified != file.Qualified || existing.Kind != file.Kind {
					result.Diagnostics = append(result.Diagnostics, conflictingLayoutDiagnostic(relativeProject, existing, file))
				}
				return nil
			}
			files[relativeProject] = file
			sources[file.SourceID] = analysis.SourceReference{
				ID:     file.SourceID,
				Path:   relativeProject,
				Symbol: qualified,
				Kind:   "file",
			}
			if kind == "package" {
				packageValue := ensurePackage(packages, qualified)
				if existingPath := firstConflictingPath(packageValue.Paths, relativeProject); existingPath != "" {
					result.Diagnostics = append(result.Diagnostics, conflictingObservationDiagnostic("package", qualified, existingPath, relativeProject))
				}
				packageValue.HasInit = true
				packageValue.SourceIDs[file.SourceID] = struct{}{}
				packageValue.Paths[relativeProject] = struct{}{}
				addFileTags(packageValue.Tags, file)
			} else {
				moduleValue := ensureModule(modules, qualified)
				if existingPath := firstConflictingPath(moduleValue.Paths, relativeProject); existingPath != "" {
					result.Diagnostics = append(result.Diagnostics, conflictingObservationDiagnostic("module", qualified, existingPath, relativeProject))
				}
				moduleValue.SourceIDs[file.SourceID] = struct{}{}
				moduleValue.Paths[relativeProject] = struct{}{}
				addFileTags(moduleValue.Tags, file)
			}
			for _, packageName := range parentPackages(qualified, kind) {
				packageValue := ensurePackage(packages, packageName)
				if !packageValue.HasInit {
					packageValue.SourceIDs[file.SourceID] = struct{}{}
					packageValue.Paths[relativeProject] = struct{}{}
				}
				if kind == "module" {
					addFileTags(packageValue.Tags, file)
				}
			}
			return nil
		})
		if walkErr != nil {
			if ctx.Err() != nil {
				return discoveryResult{}, ctx.Err()
			}
			result.Diagnostics = append(result.Diagnostics, unreadableDiagnostic(project.Root, sourceRoot.Absolute, walkErr))
		}
	}
	for _, qualified := range sortedModuleNames(modules) {
		if _, packageExists := packages[qualified]; packageExists {
			result.Diagnostics = append(result.Diagnostics, analysis.Diagnostic{
				Code:        "python_conflicting_layout",
				Severity:    "warning",
				Message:     "A package and module share the same qualified name; both observations were retained with kind-qualified IDs.",
				Subject:     qualified,
				Recoverable: true,
				Metadata:    map[string]any{"qualified_name": qualified, "package_id": moduleID("package", qualified), "module_id": moduleID("module", qualified)},
			})
		}
	}

	result.Modules = buildModuleObservations(project, packages, modules)
	result.SourceReferences = sortedSources(sources)
	sortDiagnostics(result.Diagnostics)
	return result, nil
}

func buildModuleObservations(project Project, packages map[string]*packageObservation, modules map[string]*moduleObservation) []analysis.ModuleObservation {
	rootValues := make([]string, 0, len(project.SourceRoots))
	for _, root := range project.SourceRoots {
		rootValues = append(rootValues, root.Relative)
	}
	sort.Strings(rootValues)
	result := make([]analysis.ModuleObservation, 0, len(packages)+len(modules))
	packageNames := sortedPackageNames(packages)
	for _, qualified := range packageNames {
		value := packages[qualified]
		sourceIDs := sortedKeys(value.SourceIDs)
		paths := sortedKeys(value.Paths)
		packageKind := "namespace"
		if value.HasInit {
			packageKind = "regular"
		} else {
			value.Tags["namespace"] = struct{}{}
		}
		tags := sortedKeys(value.Tags)
		metadata := map[string]any{
			"qualified_name":      qualified,
			"package_kind":        packageKind,
			"paths":               paths,
			"file_count":          len(paths),
			"source_roots":        rootValues,
			"configuration_files": append([]string(nil), project.ConfigurationFiles...),
		}
		addProjectVersion(metadata, project.PythonVersion)
		result = append(result, analysis.ModuleObservation{
			ID:                 moduleID("package", qualified),
			Language:           "python",
			Kind:               "package",
			Name:               displayName(qualified),
			DisplayName:        qualified,
			Hierarchy:          hierarchy(qualified),
			SourceReferenceIDs: sourceIDs,
			Tags:               tags,
			Metadata:           metadata,
		})
	}
	moduleNames := sortedModuleNames(modules)
	for _, qualified := range moduleNames {
		value := modules[qualified]
		sourceIDs := sortedKeys(value.SourceIDs)
		paths := sortedKeys(value.Paths)
		tags := sortedKeys(value.Tags)
		metadata := map[string]any{
			"qualified_name":      qualified,
			"module_kind":         "module",
			"relative_paths":      paths,
			"relative_path":       paths[0],
			"file_count":          len(paths),
			"source_roots":        rootValues,
			"configuration_files": append([]string(nil), project.ConfigurationFiles...),
			"stub":                containsString(tags, "stub"),
		}
		addProjectVersion(metadata, project.PythonVersion)
		result = append(result, analysis.ModuleObservation{
			ID:                 moduleID("module", qualified),
			Language:           "python",
			Kind:               "module",
			Name:               displayName(qualified),
			DisplayName:        qualified,
			Hierarchy:          hierarchy(qualified),
			SourceReferenceIDs: sourceIDs,
			Tags:               tags,
			Metadata:           metadata,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}
