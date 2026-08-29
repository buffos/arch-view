package orchestration

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

// filterResultBySourceScope is the host-side safety net for adapters that do
// not yet consume SourceScope internally. The request still carries the full
// scope to every analyzer, while this boundary guarantees that an adapter
// cannot publish observations for a path removed by the immutable plan.
func filterResultBySourceScope(result analysis.AnalysisResult, scope analysis.SourceScope) analysis.AnalysisResult {
	if scope.MatchedLocalPaths == nil {
		return result
	}
	allowed := make(map[string]struct{}, len(scope.MatchedLocalPaths))
	for _, value := range scope.MatchedLocalPaths {
		allowed[normalizeLocalSourcePath(value)] = struct{}{}
	}

	allowedSourceIDs := make(map[string]struct{}, len(result.SourceReferences))
	filteredSourceCount := 0
	filteredSources := make([]analysis.SourceReference, 0, len(result.SourceReferences))
	for _, source := range result.SourceReferences {
		if _, ok := allowed[normalizeLocalSourcePath(source.Path)]; !ok {
			filteredSourceCount++
			continue
		}
		allowedSourceIDs[source.ID] = struct{}{}
		filteredSources = append(filteredSources, source)
	}
	result.SourceReferences = filteredSources

	moduleIDs := make(map[string]struct{}, len(result.Modules))
	originalModuleCount := len(result.Modules)
	filteredModules := make([]analysis.ModuleObservation, 0, len(result.Modules))
	for _, module := range result.Modules {
		originalEvidence := len(module.SourceReferenceIDs)
		module.SourceReferenceIDs = filterIDs(module.SourceReferenceIDs, allowedSourceIDs)
		if originalEvidence > 0 && len(module.SourceReferenceIDs) == 0 {
			continue
		}
		moduleIDs[module.ID] = struct{}{}
		filteredModules = append(filteredModules, module)
	}
	result.Modules = filteredModules
	filteredSourceCount += originalModuleCount - len(result.Modules)

	originalRelationshipCount := len(result.Relationships)
	filteredRelationships := make([]analysis.RelationshipObservation, 0, len(result.Relationships))
	for _, relationship := range result.Relationships {
		if _, ok := moduleIDs[relationship.FromModuleID]; !ok {
			continue
		}
		if relationship.ToModuleID != "" {
			if _, ok := moduleIDs[relationship.ToModuleID]; !ok {
				continue
			}
		}
		originalEvidence := len(relationship.SourceReferenceIDs)
		relationship.SourceReferenceIDs = filterIDs(relationship.SourceReferenceIDs, allowedSourceIDs)
		if originalEvidence > 0 && len(relationship.SourceReferenceIDs) == 0 {
			continue
		}
		filteredRelationships = append(filteredRelationships, relationship)
	}
	result.Relationships = filteredRelationships
	filteredSourceCount += originalRelationshipCount - len(result.Relationships)

	usedReferenceIDs := make(map[string]struct{}, len(result.Relationships))
	for _, relationship := range result.Relationships {
		if relationship.ToReferenceID != "" {
			usedReferenceIDs[relationship.ToReferenceID] = struct{}{}
		}
	}
	originalReferenceCount := len(result.References)
	filteredReferences := make([]analysis.Reference, 0, len(result.References))
	for _, reference := range result.References {
		if _, ok := usedReferenceIDs[reference.ID]; ok {
			filteredReferences = append(filteredReferences, reference)
		}
	}
	result.References = filteredReferences
	filteredSourceCount += originalReferenceCount - len(result.References)

	filteredDiagnostics := make([]analysis.Diagnostic, 0, len(result.Diagnostics))
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Path != "" {
			if _, ok := allowed[normalizeLocalSourcePath(diagnostic.Path)]; !ok {
				filteredSourceCount++
				continue
			}
		}
		filteredDiagnostics = append(filteredDiagnostics, diagnostic)
	}
	result.Diagnostics = filteredDiagnostics
	if result.Status == analysis.StatusComplete && filteredSourceCount > 0 {
		result.Status = analysis.StatusPartial
	}
	if filteredSourceCount > 0 && !hasDiagnosticCode(result.Diagnostics, "analysis_scope_filtered") {
		result.Diagnostics = append(result.Diagnostics, analysis.Diagnostic{
			Code:        "analysis_scope_filtered",
			Severity:    "warning",
			Message:     "The analyzer returned only observations outside the effective source scope; excluded observations were not published.",
			Recoverable: true,
			Metadata:    map[string]any{"filtered_observation_count": filteredSourceCount},
		})
	}
	sort.Slice(result.SourceReferences, func(i, j int) bool { return result.SourceReferences[i].ID < result.SourceReferences[j].ID })
	result.Summary = analysis.ComputeSummary(result)
	return result
}

func hasDiagnosticCode(values []analysis.Diagnostic, code string) bool {
	for _, value := range values {
		if value.Code == code {
			return true
		}
	}
	return false
}

func normalizeLocalSourcePath(value string) string {
	value = filepath.ToSlash(filepath.Clean(value))
	return strings.TrimPrefix(value, "./")
}

func filterIDs(values []string, allowed map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := allowed[value]; ok {
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}
