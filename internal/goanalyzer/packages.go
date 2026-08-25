package goanalyzer

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/build"
	"go/parser"
	"go/scanner"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

var defaultExcludedRootDirectories = map[string]struct{}{
	".cache":   {},
	"bin":      {},
	"build":    {},
	"cache":    {},
	"dist":     {},
	"external": {},
	"out":      {},
	"target":   {},
	"tmp":      {},
}

var defaultExcludedNestedDirectories = map[string]struct{}{
	".git":   {},
	"vendor": {},
}

type scannedPackage struct {
	Directory   string
	RelativeDir string
	ImportPath  string
	PackageName string
	Files       []scannedFile
}

type scannedFile struct {
	RelativePath    string
	PackageName     string
	SourceReference analysis.SourceReference
	Imports         []scannedImport
	IsTest          bool
	IsGenerated     bool
	Constraints     []string
}

type scannedImport struct {
	ImportPath  string
	Source      analysis.SourceReference
	Constraints []string
}

type scanResult struct {
	Packages         []*scannedPackage
	SourceReferences []analysis.SourceReference
	Imports          []scannedImportRecord
	Diagnostics      []analysis.Diagnostic
}

type scannedImportRecord struct {
	FromImportPath string
	Import         scannedImport
}

type importTarget struct {
	Scope        string
	TargetScope  string
	ModuleID     string
	ReferenceID  string
	Reference    analysis.Reference
	HasReference bool
}

func analyzePackages(ctx context.Context, request analysis.AnalyzeRequest, project Project, manifest analysis.Manifest) (analysis.AnalysisResult, error) {
	scan, err := scanProject(ctx, request, project)
	if err != nil {
		return analysis.AnalysisResult{}, err
	}

	options := request.Options.Values
	includeExternal := optionBool(options, "include_external")
	packagesByImportPath := make(map[string]*scannedPackage, len(scan.Packages))
	modules := make([]analysis.ModuleObservation, 0, len(scan.Packages))
	for _, pkg := range scan.Packages {
		packagesByImportPath[pkg.ImportPath] = pkg
		moduleID := goPackageID(project.ModulePath, pkg.RelativeDir)
		tags := make([]string, 0, 2)
		metadata := map[string]any{
			"import_path":        pkg.ImportPath,
			"package_name":       pkg.PackageName,
			"relative_directory": pkg.RelativeDir,
			"file_count":         len(pkg.Files),
		}
		if project.GoVersion != "" {
			metadata["go_version"] = project.GoVersion
		}
		sourceIDs := make([]string, 0, len(pkg.Files))
		for _, file := range pkg.Files {
			sourceIDs = append(sourceIDs, file.SourceReference.ID)
			if file.IsTest {
				tags = appendUnique(tags, "test")
			}
			if file.IsGenerated {
				tags = appendUnique(tags, "generated")
			}
		}
		sort.Strings(sourceIDs)
		sort.Strings(tags)
		modules = append(modules, analysis.ModuleObservation{
			ID:                 moduleID,
			Language:           "go",
			Kind:               "package",
			Name:               pkg.PackageName,
			DisplayName:        pkg.ImportPath,
			Hierarchy:          hierarchyFor(pkg.RelativeDir),
			SourceReferenceIDs: sourceIDs,
			Tags:               tags,
			Metadata:           metadata,
		})
	}
	sort.Slice(modules, func(i, j int) bool { return modules[i].ID < modules[j].ID })

	references := make(map[string]analysis.Reference)
	relationships := make(map[string]analysis.RelationshipObservation)
	diagnostics := append([]analysis.Diagnostic(nil), scan.Diagnostics...)
	for _, record := range scan.Imports {
		fromModuleID := goPackageID(project.ModulePath, packagesByImportPath[record.FromImportPath].RelativeDir)
		target := resolveImportTarget(project, packagesByImportPath, record.Import.ImportPath, includeExternal)
		if target.ModuleID == "" && target.ReferenceID == "" {
			continue
		}
		if target.HasReference {
			references[target.Reference.ID] = target.Reference
		}
		relationshipKey := fromModuleID + "\x00" + target.Scope + "\x00" + target.ModuleID + target.ReferenceID
		relationship, exists := relationships[relationshipKey]
		if !exists {
			relationship = analysis.RelationshipObservation{
				ID:                 stableID("relationship", fromModuleID, target.Scope, target.ModuleID, target.ReferenceID),
				Type:               "depends_on",
				FromModuleID:       fromModuleID,
				ToModuleID:         target.ModuleID,
				ToReferenceID:      target.ReferenceID,
				SourceReferenceIDs: []string{},
				Confidence:         confidenceFor(target.TargetScope),
				Metadata: map[string]any{
					"target_scope": target.TargetScope,
					"import_paths": []string{},
				},
			}
		}
		relationship.SourceReferenceIDs = appendUnique(relationship.SourceReferenceIDs, record.Import.Source.ID)
		importPaths, _ := relationship.Metadata["import_paths"].([]string)
		importPaths = appendUnique(importPaths, record.Import.ImportPath)
		relationship.Metadata["import_paths"] = importPaths
		if len(record.Import.Constraints) > 0 {
			constraints, _ := relationship.Metadata["build_constraints"].([]string)
			for _, constraint := range record.Import.Constraints {
				constraints = appendUnique(constraints, constraint)
			}
			sort.Strings(constraints)
			relationship.Metadata["build_constraints"] = constraints
		}
		sort.Strings(relationship.SourceReferenceIDs)
		sort.Strings(importPaths)
		relationships[relationshipKey] = relationship

		if target.TargetScope == "unresolved" || target.TargetScope == "cgo" {
			diagnostics = append(diagnostics, analysis.Diagnostic{
				Code:        diagnosticCodeFor(target.TargetScope),
				Severity:    "warning",
				Message:     fmt.Sprintf("Go import %q was classified as %s.", record.Import.ImportPath, target.TargetScope),
				Path:        record.Import.Source.Path,
				Location:    record.Import.Source.Start,
				Recoverable: true,
				Metadata: map[string]any{
					"import_path":  record.Import.ImportPath,
					"target_scope": target.TargetScope,
				},
			})
		}
	}

	if len(modules) == 0 {
		diagnostics = append(diagnostics, analysis.Diagnostic{
			Code:        "go_no_packages",
			Severity:    "warning",
			Message:     "No eligible Go packages were found in the selected module.",
			Recoverable: true,
		})
	}

	result := analysis.AnalysisResult{
		Status:   analysis.StatusComplete,
		Analyzer: analyzerInfo(manifest),
		Project: analysis.ProjectInfo{
			RootLabel:     filepath.Base(project.Root),
			Boundary:      project.Boundary,
			ModulePath:    project.ModulePath,
			ModuleRoot:    project.RelativeModuleRoot,
			WorkspacePath: project.RelativeWorkspacePath,
		},
		Modules:          modules,
		Relationships:    relationshipValues(relationships),
		References:       referenceValues(references),
		SourceReferences: scan.SourceReferences,
		Diagnostics:      diagnostics,
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Recoverable {
			result.Status = analysis.StatusPartial
			break
		}
	}
	result.Summary = analysis.ComputeSummary(result)
	return result, nil
}

func scanProject(ctx context.Context, request analysis.AnalyzeRequest, project Project) (scanResult, error) {
	options := request.Options.Values
	includeTests := optionBool(options, "include_tests")
	includeGenerated := optionBool(options, "include_generated")
	excludePatterns := optionStrings(options, "exclude")
	packages := make(map[string]*scannedPackage)
	result := scanResult{}

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
			pkg = &scannedPackage{
				Directory:   filepath.Clean(filepath.Dir(filePath)),
				RelativeDir: filepath.Clean(relativeModulePath),
				ImportPath:  goImportPath(project.ModulePath, relativeModulePath),
			}
			packages[pkg.Directory] = pkg
		}
		fileSource := analysis.SourceReference{
			ID:   stableID("file", relativeProject),
			Path: relativeProject,
			Kind: "file",
		}
		fileRecord := scannedFile{
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
				ID:     stableID("import", relativeProject, strconv.Itoa(start.Line), strconv.Itoa(start.Column), importPath),
				Path:   relativeProject,
				Start:  &analysis.Position{Line: start.Line, Column: start.Column},
				End:    &analysis.Position{Line: end.Line, Column: end.Column},
				Symbol: importPath,
				Kind:   "import",
			}
			fileRecord.Imports = append(fileRecord.Imports, scannedImport{ImportPath: importPath, Source: importSource, Constraints: constraints})
			result.SourceReferences = append(result.SourceReferences, importSource)
		}
		pkg.Files = append(pkg.Files, fileRecord)
		result.SourceReferences = append(result.SourceReferences, fileSource)
		return nil
	})
	if walkErr != nil {
		if ctx.Err() != nil {
			return scanResult{}, ctx.Err()
		}
		return scanResult{}, analysis.WrapHostError(analysis.ErrUnreadableProject, "Go module could not be scanned", walkErr, map[string]any{"module_root": project.ModuleRoot})
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
				result.Imports = append(result.Imports, scannedImportRecord{
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

func resolveImportTarget(project Project, packages map[string]*scannedPackage, importPath string, includeExternal bool) importTarget {
	if pkg, ok := packages[importPath]; ok {
		return importTarget{
			Scope:       "local",
			TargetScope: "local",
			ModuleID:    goPackageID(project.ModulePath, pkg.RelativeDir),
		}
	}
	if importPath == "C" {
		return referenceTarget(importPath, "external", "cgo", includeExternal)
	}
	if strings.HasPrefix(importPath, project.ModulePath+"/") || importPath == project.ModulePath {
		relative := strings.TrimPrefix(importPath, project.ModulePath)
		relative = strings.TrimPrefix(relative, "/")
		directory := project.ModuleRoot
		if relative != "" {
			directory = filepath.Join(project.ModuleRoot, filepath.FromSlash(relative))
		}
		if hasGoFile(directory) {
			return referenceTarget(importPath, "unresolved", "conditional", includeExternal)
		}
		return referenceTarget(importPath, "unresolved", "unresolved", includeExternal)
	}
	if isStandardLibraryImport(importPath) {
		return referenceTarget(importPath, "standard_library", "standard_library", includeExternal)
	}
	return referenceTarget(importPath, "external", "external", includeExternal)
}

func referenceTarget(importPath, scope, targetScope string, includeExternal bool) importTarget {
	metadata := map[string]any{
		"import_path":  importPath,
		"target_scope": targetScope,
	}
	if scope == "external" {
		if includeExternal {
			metadata["detail"] = "full"
		} else {
			metadata["detail"] = "summary"
		}
	}
	reference := analysis.Reference{
		ID:       stableID("reference", scope, importPath),
		Name:     importPath,
		Scope:    scope,
		Language: "go",
		Metadata: metadata,
	}
	return importTarget{
		Scope:        scope,
		TargetScope:  targetScope,
		ReferenceID:  reference.ID,
		Reference:    reference,
		HasReference: true,
	}
}

func confidenceFor(targetScope string) *analysis.Confidence {
	switch targetScope {
	case "local", "standard_library", "external":
		return &analysis.Confidence{Basis: "resolved", Score: 1}
	case "conditional":
		return &analysis.Confidence{Basis: "inferred", Score: 0.5}
	case "cgo":
		return &analysis.Confidence{Basis: "dynamic", Score: 0.4}
	default:
		return &analysis.Confidence{Basis: "unresolved", Score: 0.2}
	}
}

func diagnosticCodeFor(targetScope string) string {
	if targetScope == "cgo" {
		return "go_cgo_import"
	}
	return "go_unresolved_import"
}

func matchesBuildContext(filePath string, tags []string) (bool, []string, error) {
	buildContext := build.Default
	buildContext.BuildTags = append([]string(nil), tags...)
	matched, err := buildContext.MatchFile(filepath.Dir(filePath), filepath.Base(filePath))
	return matched, buildConstraintLines(filePath), err
}

func buildConstraintLines(filePath string) []string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	var constraints []string
	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineCount := 0
	for scanner.Scan() {
		lineCount++
		if lineCount > 100 {
			break
		}
		trimmed := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(trimmed, "//go:build "):
			constraints = append(constraints, strings.TrimSpace(strings.TrimPrefix(trimmed, "//go:build ")))
		case strings.HasPrefix(trimmed, "// +build "):
			constraints = append(constraints, strings.TrimSpace(strings.TrimPrefix(trimmed, "// +build ")))
		case trimmed == "", strings.HasPrefix(trimmed, "//"):
			continue
		default:
			sort.Strings(constraints)
			return constraints
		}
	}
	sort.Strings(constraints)
	return constraints
}

func parseDiagnostic(err error, relativePath string) analysis.Diagnostic {
	location := &analysis.Position{Line: 1, Column: 1}
	switch errors := err.(type) {
	case scanner.ErrorList:
		if len(errors) > 0 {
			location = parseErrorPosition(errors[0].Pos, location)
		}
	case *scanner.ErrorList:
		if errors != nil && len(*errors) > 0 {
			location = parseErrorPosition((*errors)[0].Pos, location)
		}
	}
	return analysis.Diagnostic{
		Code:        "go_parse_error",
		Severity:    "error",
		Message:     fmt.Sprintf("Go source could not be parsed: %v", err),
		Path:        relativePath,
		Location:    location,
		Recoverable: true,
	}
}

func parseErrorPosition(position token.Position, fallback *analysis.Position) *analysis.Position {
	if position.Line <= 0 || position.Column <= 0 {
		return fallback
	}
	return &analysis.Position{Line: position.Line, Column: position.Column}
}

func isGeneratedSource(content []byte) bool {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for index := 0; scanner.Scan() && index < 20; index++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			if strings.Contains(line, "Code generated") && strings.Contains(line, "DO NOT EDIT") {
				return true
			}
			continue
		}
		break
	}
	return false
}

func excludedDirectory(relativePath string, patterns []string) bool {
	clean := filepath.ToSlash(filepath.Clean(relativePath))
	segments := strings.Split(clean, "/")
	if len(segments) > 0 {
		if _, excluded := defaultExcludedRootDirectories[segments[0]]; excluded {
			return true
		}
	}
	for _, segment := range segments {
		if _, excluded := defaultExcludedNestedDirectories[segment]; excluded {
			return true
		}
	}
	return matchesAnyExclude(clean, patterns)
}

func matchesAnyExclude(relativePath string, patterns []string) bool {
	clean := filepath.ToSlash(filepath.Clean(relativePath))
	for _, pattern := range patterns {
		pattern = strings.TrimPrefix(filepath.ToSlash(filepath.Clean(pattern)), "./")
		if pattern == "" || pattern == "." {
			continue
		}
		if matched, _ := path.Match(pattern, clean); matched {
			return true
		}
		if strings.HasSuffix(pattern, "/**") {
			prefix := strings.TrimSuffix(pattern, "/**")
			if clean == prefix || strings.HasPrefix(clean, prefix+"/") {
				return true
			}
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

func hasGoFile(directory string) bool {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".go" {
			return true
		}
	}
	return false
}

func isStandardLibraryImport(importPath string) bool {
	first := importPath
	if slash := strings.IndexByte(first, '/'); slash >= 0 {
		first = first[:slash]
	}
	if first == "" || strings.Contains(first, ".") || first == "internal" || first == "vendor" {
		return false
	}
	standardRoot := filepath.Clean(build.Default.GOROOT)
	if standardRoot == "." || standardRoot == "" {
		return false
	}
	imported, err := build.Default.Import(importPath, "", build.FindOnly)
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(standardRoot, imported.Dir)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

func goImportPath(modulePath, relativeDirectory string) string {
	if relativeDirectory == "." || relativeDirectory == "" {
		return modulePath
	}
	return strings.TrimSuffix(modulePath, "/") + "/" + filepath.ToSlash(relativeDirectory)
}

func goPackageID(modulePath, relativeDirectory string) string {
	return "go:" + goImportPath(modulePath, relativeDirectory)
}

func hierarchyFor(relativeDirectory string) []string {
	if relativeDirectory == "." || relativeDirectory == "" {
		return []string{}
	}
	segments := strings.Split(filepath.ToSlash(relativeDirectory), "/")
	result := make([]string, 0, len(segments))
	for _, segment := range segments {
		if segment != "" && segment != "." {
			result = append(result, segment)
		}
	}
	return result
}

func relativeProjectPath(root, filePath string) string {
	relative, err := filepath.Rel(root, filePath)
	if err != nil {
		return filepath.ToSlash(filepath.Clean(filePath))
	}
	return filepath.ToSlash(filepath.Clean(relative))
}

func stableID(kind string, parts ...string) string {
	payload := kind + "\x00" + strings.Join(parts, "\x00")
	sum := sha256.Sum256([]byte(payload))
	return "go:" + kind + ":" + hex.EncodeToString(sum[:8])
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func optionBool(values map[string]any, name string) bool {
	value, _ := values[name].(bool)
	return value
}

func optionStrings(values map[string]any, name string) []string {
	value, _ := values[name].([]string)
	return append([]string(nil), value...)
}

func relationshipValues(values map[string]analysis.RelationshipObservation) []analysis.RelationshipObservation {
	result := make([]analysis.RelationshipObservation, 0, len(values))
	for _, value := range values {
		sort.Strings(value.SourceReferenceIDs)
		if importPaths, ok := value.Metadata["import_paths"].([]string); ok {
			sort.Strings(importPaths)
			value.Metadata["import_paths"] = importPaths
		}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func referenceValues(values map[string]analysis.Reference) []analysis.Reference {
	result := make([]analysis.Reference, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}
