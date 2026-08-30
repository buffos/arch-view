package scanner

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/syntax"
	gosyntax "github.com/buffo/arch-view/internal/analysis/syntax/go"
)

func ScanProject(ctx context.Context, request analysis.AnalyzeRequest, project Project) (ScanResult, error) {
	return ScanProjectWithSyntaxProvider(ctx, request, project, gosyntax.NewProvider())
}

// ScanProjectWithSyntaxProvider scans Go source with the supplied syntax
// provider. A provider failure is reported as a recoverable diagnostic; source
// parsing never falls back to a second implementation.
func ScanProjectWithSyntaxProvider(ctx context.Context, request analysis.AnalyzeRequest, project Project, syntaxProvider syntax.Provider) (ScanResult, error) {
	options := request.Options.Values
	includeTests := optionBool(options, "include_tests")
	includeGenerated := optionBool(options, "include_generated")
	excludePatterns := optionStrings(options, "exclude")
	packages := make(map[string]*Package)
	result := ScanResult{SyntaxProvider: syntaxProvider}

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
		relativeModulePath, relErr := filepath.Rel(project.ModuleRoot, filepath.Dir(filePath))
		if relErr != nil {
			return relErr
		}
		if relativeModulePath == "." {
			relativeModulePath = "."
		}
		roles := []string{"role:source"}
		if strings.HasSuffix(entry.Name(), "_test.go") {
			roles = append(roles, "role:test")
		}
		if generated {
			roles = append(roles, "role:generated")
		}
		sourceFileIndex := len(result.SourceFiles)
		result.SourceFiles = append(result.SourceFiles, SourceFile{
			RelativePath:   relativeProject,
			Content:        append([]byte(nil), content...),
			Language:       analysis.LanguageRef{ID: "language:go"},
			Roles:          roles,
			AnalysisStatus: analysis.FileAnalysisComplete,
			ModuleID:       PackageID(project.ModulePath, relativeModulePath),
		})
		if syntaxProvider == nil {
			result.SourceFiles[sourceFileIndex].AnalysisStatus = analysis.FileAnalysisUnknown
			result.Diagnostics = append(result.Diagnostics, goSyntaxBackendDiagnostic(relativeProject, errors.New("go tree-sitter syntax provider is not configured")))
			return nil
		}
		parsed, parseErr := syntaxProvider.Parse(ctx, syntax.Source{Path: relativeProject, Content: content})
		if parseErr != nil {
			parsed.Close()
			result.SourceFiles[sourceFileIndex].AnalysisStatus = analysis.FileAnalysisUnparsed
			if ctx.Err() != nil {
				return ctx.Err()
			}
			result.Diagnostics = append(result.Diagnostics, goSyntaxBackendDiagnostic(relativeProject, parseErr))
			return nil
		}
		defer parsed.Close()
		for _, issue := range parsed.Issues {
			result.SourceFiles[sourceFileIndex].AnalysisStatus = analysis.FileAnalysisPartial
			result.Diagnostics = append(result.Diagnostics, goSyntaxIssueDiagnostic(relativeProject, issue))
		}
		if parsed.Tree == nil || parsed.Tree.Root() == nil {
			result.SourceFiles[sourceFileIndex].AnalysisStatus = analysis.FileAnalysisUnparsed
			result.Diagnostics = append(result.Diagnostics, goSyntaxBackendDiagnostic(relativeProject, errors.New("tree-sitter returned no Go syntax tree")))
			return nil
		}
		root := parsed.Tree.Root()
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
		result.SourceFiles[sourceFileIndex].SourceReferenceIDs = []string{fileSource.ID}
		fileRecord := File{
			RelativePath:    relativeProject,
			PackageName:     goSyntaxPackageName(root),
			SourceReference: fileSource,
			IsTest:          strings.HasSuffix(entry.Name(), "_test.go"),
			IsGenerated:     generated,
			Constraints:     constraints,
		}
		for _, importSpec := range goSyntaxImportSpecs(root) {
			pathNode := importSpec.ChildByFieldName("path")
			if pathNode == nil {
				continue
			}
			importPath, unquoteErr := strconv.Unquote(pathNode.Text())
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
			rangeValue := pathNode.Range()
			startLine := int(rangeValue.Start.Row) + 1
			startColumn := int(rangeValue.Start.Column) + 1
			endLine := int(rangeValue.End.Row) + 1
			endColumn := int(rangeValue.End.Column) + 1
			importSource := analysis.SourceReference{
				ID:     StableID("import", relativeProject, strconv.Itoa(startLine), strconv.Itoa(startColumn), importPath),
				Path:   relativeProject,
				Start:  &analysis.Position{Line: startLine, Column: startColumn},
				End:    &analysis.Position{Line: endLine, Column: endColumn},
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
	sort.Slice(result.SourceFiles, func(i, j int) bool { return result.SourceFiles[i].RelativePath < result.SourceFiles[j].RelativePath })
	return result, nil
}
