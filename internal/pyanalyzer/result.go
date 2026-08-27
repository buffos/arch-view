package pyanalyzer

import (
	"path/filepath"

	"github.com/buffo/arch-view/internal/analysis"
)

// BuildResult converts discovery output into the common analyzer contract.
func BuildResult(project Project, discovery discoveryResult, _ analysis.AnalyzeRequest, manifest analysis.Manifest) analysis.AnalysisResult {
	diagnostics := append([]analysis.Diagnostic{}, discovery.Diagnostics...)
	relationships, references, importDiagnostics := buildImportObservations(project, discovery)
	diagnostics = append(diagnostics, importDiagnostics...)
	if len(discovery.Modules) == 0 {
		diagnostics = append(diagnostics, analysis.Diagnostic{
			Code:        "python_no_modules",
			Severity:    "warning",
			Message:     "No eligible Python packages or modules were found in the selected source roots.",
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
	sortDiagnostics(result.Diagnostics)
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
