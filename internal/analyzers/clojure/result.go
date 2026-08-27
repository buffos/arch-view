package clojureanalyzer

import (
	"path/filepath"
	"sort"

	"github.com/buffo/arch-view/internal/analysis"
)

// BuildResult converts discovery output into the common analyzer contract.
// Dependency, reference, and dynamic observations are composed into the
// language-neutral result without changing its public shape.
func BuildResult(project Project, discovery discoveryResult, manifest analysis.Manifest) analysis.AnalysisResult {
	diagnostics := append([]analysis.Diagnostic{}, discovery.Diagnostics...)
	relationships, references, dependencyDiagnostics := buildClojureDependencyObservations(discovery.Modules, discovery.Dependencies)
	diagnostics = append(diagnostics, dependencyDiagnostics...)
	dynamicRelationships, dynamicReferences, dynamicDiagnostics := buildClojureDynamicObservations(discovery.DynamicLoads)
	relationships = append(relationships, dynamicRelationships...)
	references = append(references, dynamicReferences...)
	diagnostics = append(diagnostics, dynamicDiagnostics...)
	if len(discovery.Modules) == 0 {
		diagnostics = append(diagnostics, analysis.Diagnostic{
			Code:        "clojure_no_modules",
			Severity:    "warning",
			Message:     "No eligible Clojure namespaces were found in the selected source roots.",
			Recoverable: true,
		})
	}
	result := analysis.AnalysisResult{
		Status:   analysis.StatusComplete,
		Analyzer: analyzerInfo(manifest),
		Project: analysis.ProjectInfo{
			RootLabel:  filepath.Base(project.Root),
			Boundary:   project.Boundary,
			ModuleRoot: firstSourceRoot(project.SourceRoots),
		},
		Modules:          discovery.Modules,
		Relationships:    relationships,
		References:       references,
		SourceReferences: discovery.SourceReferences,
		Diagnostics:      diagnostics,
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Recoverable {
			result.Status = analysis.StatusPartial
			break
		}
	}
	sortClojureRelationshipCollections(result.Relationships, result.References)
	sortClojureDiagnostics(result.Diagnostics)
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

func firstSourceRoot(roots []SourceRoot) string {
	if len(roots) == 0 {
		return "."
	}
	return roots[0].Relative
}

func sortClojureRelationshipCollections(relationships []analysis.RelationshipObservation, references []analysis.Reference) {
	for index := range relationships {
		sort.Strings(relationships[index].SourceReferenceIDs)
	}
	sort.Slice(relationships, func(i, j int) bool { return relationships[i].ID < relationships[j].ID })
	sort.Slice(references, func(i, j int) bool { return references[i].ID < references[j].ID })
}
