package tsanalyzer

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
	Files            map[string]tsFileObservation
	SourceReferences []analysis.SourceReference
	Diagnostics      []analysis.Diagnostic
}

type tsFileObservation struct {
	RelativePath  string
	AbsolutePath  string
	ModuleID      string
	Extension     string
	SourceRoot    string
	IsTest        bool
	IsJavaScript  bool
	IsDeclaration bool
	Content       string
}

// Discover walks the selected project scope and records module/file evidence.
// It does not invoke TypeScript, package scripts, bundlers, or target code.
func Discover(ctx context.Context, project Project, options analysis.EffectiveOptions) (discoveryResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return discoveryResult{}, err
	}
	includeTests := optionBool(options, "include_tests")
	includeJavaScript := project.IncludeJavaScript
	files := make(map[string]tsFileObservation)
	diagnostics := append([]analysis.Diagnostic{}, project.ConfigDiagnostics...)

	walkRoots := project.WalkRoots
	if len(walkRoots) == 0 {
		walkRoots = []string{"."}
	}
	for _, walkRoot := range walkRoots {
		if err := ctx.Err(); err != nil {
			return discoveryResult{}, err
		}
		absoluteRoot := filepath.Join(project.Root, filepath.FromSlash(walkRoot))
		walkErr := filepath.WalkDir(absoluteRoot, func(filePath string, entry fs.DirEntry, walkErr error) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if walkErr != nil {
				diagnostics = append(diagnostics, unreadableTSDiagnostic(project.Root, filePath, walkErr))
				return nil
			}
			relative := relativeProjectPath(project.Root, filePath)
			if entry.IsDir() {
				if relative != "." && (defaultExcludedDirectory(relative) || excludedByPatterns(project.Root, relative, project.Exclude)) {
					return fs.SkipDir
				}
				return nil
			}
			if !safeFileWithinRoot(project.Root, filePath) {
				diagnostics = append(diagnostics, analysis.Diagnostic{
					Code:        "typescript_path_outside_root",
					Severity:    "warning",
					Message:     "TypeScript source path resolves outside the selected project root and was ignored.",
					Path:        relative,
					Recoverable: true,
				})
				return nil
			}
			if !eligibleTSSource(relative, entry.Name(), project, includeTests, includeJavaScript) {
				return nil
			}
			if project.HasFiles && !containsString(project.Files, relative) {
				return nil
			}
			if !project.HasFiles && project.HasInclude && !includedByPatterns(project.Root, relative, project.Include) {
				return nil
			}
			content, err := os.ReadFile(filePath)
			if err != nil {
				diagnostics = append(diagnostics, unreadableTSDiagnostic(project.Root, filePath, err))
				return nil
			}
			if generatedTSSource(string(content)) {
				return nil
			}
			relative = filepath.ToSlash(filepath.Clean(relative))
			if _, exists := files[relative]; exists {
				return nil
			}
			extension := strings.ToLower(filepath.Ext(entry.Name()))
			observation := tsFileObservation{
				RelativePath:  relative,
				AbsolutePath:  filepath.Clean(filePath),
				ModuleID:      moduleIDForPath(relative),
				Extension:     extension,
				SourceRoot:    sourceRootForPath(project, filePath),
				IsTest:        isTestPath(relative, entry.Name()),
				IsJavaScript:  extension == ".js" || extension == ".jsx",
				IsDeclaration: strings.HasSuffix(strings.ToLower(entry.Name()), ".d.ts"),
				Content:       string(content),
			}
			files[relative] = observation
			return nil
		})
		if walkErr != nil {
			if ctx.Err() != nil {
				return discoveryResult{}, ctx.Err()
			}
			diagnostics = append(diagnostics, unreadableTSDiagnostic(project.Root, absoluteRoot, walkErr))
		}
	}
	assignUniqueTSModuleIDs(files)
	if project.HasFiles {
		for _, configured := range project.Files {
			if !containsTSFile(files, configured) {
				diagnostics = append(diagnostics, analysis.Diagnostic{
					Code:        "typescript_configured_file_unavailable",
					Severity:    "warning",
					Message:     "A file listed by the selected TypeScript configuration was not available in the eligible source scope.",
					Path:        configured,
					Recoverable: true,
				})
			}
		}
	}

	paths := make([]string, 0, len(files))
	for relative := range files {
		paths = append(paths, relative)
	}
	sort.Strings(paths)
	modules := make([]analysis.ModuleObservation, 0, len(paths))
	sources := make([]analysis.SourceReference, 0, len(paths))
	for _, relative := range paths {
		file := files[relative]
		sourceID := stableTSID("source", relative)
		tags := []string{}
		if file.IsTest {
			tags = append(tags, "test")
		}
		if file.IsJavaScript {
			tags = append(tags, "javascript")
		}
		if file.IsDeclaration {
			tags = append(tags, "declaration")
		}
		if file.Extension == ".tsx" || file.Extension == ".jsx" {
			tags = append(tags, "jsx")
		}
		sort.Strings(tags)
		metadata := map[string]any{
			"path":        relative,
			"extension":   file.Extension,
			"source_root": file.SourceRoot,
		}
		if file.IsTest {
			metadata["test"] = true
		}
		if file.IsJavaScript {
			metadata["javascript"] = true
		}
		if file.IsDeclaration {
			metadata["declaration"] = true
		}
		modules = append(modules, analysis.ModuleObservation{
			ID:                 file.ModuleID,
			Language:           "typescript",
			Kind:               "module",
			Name:               strings.TrimPrefix(file.ModuleID, "ts:module:"),
			DisplayName:        moduleDisplayName(relative),
			Hierarchy:          moduleHierarchy(relative),
			SourceReferenceIDs: []string{sourceID},
			Tags:               tags,
			Metadata:           metadata,
		})
		sources = append(sources, analysis.SourceReference{
			ID:     sourceID,
			Path:   relative,
			Symbol: moduleDisplayName(relative),
			Kind:   "file",
		})
	}
	return discoveryResult{Modules: modules, Files: files, SourceReferences: sources, Diagnostics: diagnostics}, nil
}

func assignUniqueTSModuleIDs(files map[string]tsFileObservation) {
	pathsByID := make(map[string][]string)
	paths := make([]string, 0, len(files))
	for relative, file := range files {
		pathsByID[file.ModuleID] = append(pathsByID[file.ModuleID], relative)
		paths = append(paths, relative)
	}
	sort.Strings(paths)
	for _, relative := range paths {
		file := files[relative]
		if len(pathsByID[file.ModuleID]) < 2 {
			continue
		}
		extension := strings.TrimPrefix(filepath.Ext(relative), ".")
		if extension == "" {
			extension = "file"
		}
		file.ModuleID = file.ModuleID + ":" + extension
		files[relative] = file
	}
}

func sourceRootForPath(project Project, filePath string) string {
	best := "."
	bestDepth := -1
	for _, relative := range project.SourceRoots {
		candidate := filepath.Join(project.Root, filepath.FromSlash(relative))
		if !pathWithin(candidate, filePath) {
			continue
		}
		depth := len(splitPathPattern(normalizedRelative(relative)))
		if depth > bestDepth {
			best = relative
			bestDepth = depth
		}
	}
	return best
}

func eligibleTSSource(relative, name string, project Project, includeTests, includeJavaScript bool) bool {
	if excludedTSFile(relative, name, project, includeTests) {
		return false
	}
	extension := strings.ToLower(filepath.Ext(name))
	switch extension {
	case ".ts", ".tsx":
		return true
	case ".js", ".jsx":
		return includeJavaScript
	default:
		return false
	}
}

func containsString(values []string, value string) bool {
	for _, current := range values {
		if tsPathKey(current) == tsPathKey(value) {
			return true
		}
	}
	return false
}

func containsTSFile(files map[string]tsFileObservation, value string) bool {
	for relative := range files {
		if tsPathKey(relative) == tsPathKey(value) {
			return true
		}
	}
	return false
}
