package scanner

import (
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

func ScanProject(ctx context.Context, request analysis.AnalyzeRequest, project Project) (ScanResult, error) {
	options := request.Options.Values
	includeTests := optionBool(options, "include_tests")
	includeGenerated := optionBool(options, "include_generated")
	excludePatterns := optionStrings(options, "exclude")
	packages := make(map[string]*Package)
	result := ScanResult{}

	walkErr := filepath.WalkDir(project.ModuleRoot, func(filePath string, entry fs.DirEntry, walkErr error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if walkErr != nil {
			if filePath == project.ModuleRoot {
				return walkErr
			}
			result.Diagnostics = append(result.Diagnostics, analysis.Diagnostic{
				Code:        "go_unreadable_file",
				Severity:    "error",
				Message:     fmt.Sprintf("Go source path could not be read: %v", walkErr),
				Path:        relativeProjectPath(project.Root, filePath),
				Recoverable: true,
			})
			return nil
		}
		if entry.IsDir() {
			relativeModulePath, relErr := filepath.Rel(project.ModuleRoot, filePath)
			if relErr != nil {
				return relErr
			}
			if relativeModulePath != "." && excludedDirectory(relativeModulePath, excludePatterns) {
				return fs.SkipDir
			}
			if relativeModulePath != "." && existsAsFile(filepath.Join(filePath, "go.mod")) {
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Ext(entry.Name()) != ".go" {
			return nil
		}
		if !includeTests && strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		relativeProject := relativeProjectPath(project.Root, filePath)
		if matchesAnyExclude(relativeProject, excludePatterns) {
			return nil
		}
		content, readErr := os.ReadFile(filePath)
		if readErr != nil {
			result.Diagnostics = append(result.Diagnostics, analysis.Diagnostic{
				Code:        "go_unreadable_file",
				Severity:    "error",
				Message:     fmt.Sprintf("Go source file could not be read: %v", readErr),
				Path:        relativeProject,
				Recoverable: true,
			})
			return nil
		}
		matched, constraints, matchErr := matchesBuildContext(filePath, optionStrings(options, "build_tags"))
		if matchErr != nil {
			result.Diagnostics = append(result.Diagnostics, analysis.Diagnostic{
				Code:        "go_build_constraint_error",
				Severity:    "error",
				Message:     fmt.Sprintf("Go build constraints could not be evaluated: %v", matchErr),
				Path:        relativeProject,
				Recoverable: true,
			})
			return nil
		}
		if !matched {
			return nil
		}
		generated := isGeneratedSource(content)
		if generated && !includeGenerated {
			return nil
		}
		fileSet := token.NewFileSet()
		parsed, parseErr := parser.ParseFile(fileSet, filePath, content, parser.ParseComments)
		if parseErr != nil {
			result.Diagnostics = append(result.Diagnostics, parseDiagnostic(parseErr, relativeProject))
			if parsed == nil {
				return nil
			}
		}
		if parsed == nil {
			return nil
		}
		relativeModulePath, relErr := filepath.Rel(project.ModuleRoot, filepath.Dir(filePath))
		if relErr != nil {
			return relErr
		}
		if relativeModulePath == "." {
			relativeModulePath = "."
		}
		pkg := packages[filepath.Clean(filepath.Dir(filePath))]
		if pkg == nil {
			pkg = &Package{
				Directory:   filepath.Clean(filepath.Dir(filePath)),
				RelativeDir: filepath.Clean(relativeModulePath),
				ImportPath:  goImportPath(project.ModulePath, relativeModulePath),
			}
			packages[pkg.Directory] = pkg
		}
		fileSource := analysis.SourceReference{
			ID:   StableID("file", relativeProject),
			Path: relativeProject,
			Kind: "file",
		}
		fileRecord := File{
			RelativePath:    relativeProject,
			PackageName:     parsed.Name.Name,
			SourceReference: fileSource,
			IsTest:          strings.HasSuffix(entry.Name(), "_test.go"),
			IsGenerated:     generated,
			Constraints:     constraints,
		}
		for _, importSpec := range parsed.Imports {
			importPath, unquoteErr := strconv.Unquote(importSpec.Path.Value)
			if unquoteErr != nil {
				result.Diagnostics = append(result.Diagnostics, analysis.Diagnostic{
					Code:        "go_import_literal_error",
					Severity:    "error",
					Message:     fmt.Sprintf("Go import path could not be decoded: %v", unquoteErr),
					Path:        relativeProject,
					Recoverable: true,
				})
				continue
			}
			start := fileSet.Position(importSpec.Path.Pos())
			end := fileSet.Position(importSpec.Path.End())
			importSource := analysis.SourceReference{
				ID:     StableID("import", relativeProject, strconv.Itoa(start.Line), strconv.Itoa(start.Column), importPath),
				Path:   relativeProject,
				Start:  &analysis.Position{Line: start.Line, Column: start.Column},
				End:    &analysis.Position{Line: end.Line, Column: end.Column},
				Symbol: importPath,
				Kind:   "import",
			}
			fileRecord.Imports = append(fileRecord.Imports, Import{ImportPath: importPath, Source: importSource, Constraints: constraints})
			result.SourceReferences = append(result.SourceReferences, importSource)
		}
		pkg.Files = append(pkg.Files, fileRecord)
		result.SourceReferences = append(result.SourceReferences, fileSource)
		return nil
	})
	if walkErr != nil {
		if ctx.Err() != nil {
			return ScanResult{}, ctx.Err()
		}
		return ScanResult{}, analysis.WrapHostError(analysis.ErrUnreadableProject, "Go module could not be scanned", walkErr, map[string]any{"module_root": project.ModuleRoot})
	}

	for _, pkg := range packages {
		sort.Slice(pkg.Files, func(i, j int) bool { return pkg.Files[i].RelativePath < pkg.Files[j].RelativePath })
		if len(pkg.Files) == 0 {
			continue
		}
		for _, file := range pkg.Files {
			if !file.IsTest {
				pkg.PackageName = file.PackageName
				break
			}
		}
		if pkg.PackageName == "" {
			pkg.PackageName = pkg.Files[0].PackageName
		}
		for _, file := range pkg.Files {
			validTestPackage := file.IsTest && file.PackageName == pkg.PackageName+"_test"
			if file.PackageName != pkg.PackageName && !validTestPackage {
				result.Diagnostics = append(result.Diagnostics, analysis.Diagnostic{
					Code:        "go_package_name_mismatch",
					Severity:    "warning",
					Message:     fmt.Sprintf("Go package directory contains package names %q and %q.", pkg.PackageName, file.PackageName),
					Path:        file.RelativePath,
					Recoverable: true,
				})
			}
		}
		for _, file := range pkg.Files {
			for _, importRecord := range file.Imports {
				result.Imports = append(result.Imports, ImportRecord{
					FromImportPath: pkg.ImportPath,
					Import:         importRecord,
				})
			}
		}
		result.Packages = append(result.Packages, pkg)
	}
	sort.Slice(result.Packages, func(i, j int) bool { return result.Packages[i].ImportPath < result.Packages[j].ImportPath })
	sort.Slice(result.SourceReferences, func(i, j int) bool { return result.SourceReferences[i].ID < result.SourceReferences[j].ID })
	sort.Slice(result.Imports, func(i, j int) bool {
		if result.Imports[i].FromImportPath == result.Imports[j].FromImportPath {
			return result.Imports[i].Import.Source.ID < result.Imports[j].Import.Source.ID
		}
		return result.Imports[i].FromImportPath < result.Imports[j].FromImportPath
	})
	return result, nil
}
