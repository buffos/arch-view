package observations

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analyzers/go/imports"
	"github.com/buffo/arch-view/internal/analyzers/go/scanner"
)

func Build(scan scanner.ScanResult, request analysis.AnalyzeRequest, project scanner.Project, manifest analysis.Manifest) analysis.AnalysisResult {
	options := request.Options.Values
	includeExternal := optionBool(options, "include_external")
	packagesByImportPath := make(map[string]*scanner.Package, len(scan.Packages))
	modules := make([]analysis.ModuleObservation, 0, len(scan.Packages))
	for _, pkg := range scan.Packages {
		packagesByImportPath[pkg.ImportPath] = pkg
		moduleID := scanner.PackageID(project.ModulePath, pkg.RelativeDir)
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
			Hierarchy:          scanner.Hierarchy(pkg.RelativeDir),
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
		fromModuleID := scanner.PackageID(project.ModulePath, packagesByImportPath[record.FromImportPath].RelativeDir)
		target := imports.ResolveImportTarget(project, packagesByImportPath, record.Import.ImportPath, includeExternal)
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
				ID:                 scanner.StableID("relationship", fromModuleID, target.Scope, target.ModuleID, target.ReferenceID),
				Type:               "depends_on",
				FromModuleID:       fromModuleID,
				ToModuleID:         target.ModuleID,
				ToReferenceID:      target.ReferenceID,
				SourceReferenceIDs: []string{},
				Confidence:         imports.ConfidenceFor(target.TargetScope),
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
				Code:        imports.DiagnosticCodeFor(target.TargetScope),
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
	return result
}

func analyzerInfo(manifest analysis.Manifest) analysis.AnalyzerInfo {
	return analysis.AnalyzerInfo{
		ID:         manifest.ID,
		Version:    manifest.Version,
		Language:   manifest.Language,
		APIVersion: manifest.APIVersion,
	}
}

func optionBool(values map[string]any, name string) bool {
	value, _ := values[name].(bool)
	return value
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
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
