package canonical

import (
	"encoding/json"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
)

func normalizationDiagnostics(value model.Model) []model.Diagnostic {
	moduleIDs := make(map[string]struct{}, len(value.Modules))
	for _, module := range value.Modules {
		moduleIDs[module.ID] = struct{}{}
	}
	result := []model.Diagnostic{}
	for _, relationship := range value.Relationships {
		if _, exists := moduleIDs[relationship.FromModuleID]; !exists {
			result = append(result, model.Diagnostic{
				ID:          stableID("diagnostic", "model_missing_source", relationship.ID),
				Code:        "model_missing_source_module",
				Severity:    "error",
				Message:     "Relationship source module is not present in the normalized module set.",
				Recoverable: false,
			})
		}
	}
	return result
}

func normalizationConflictDiagnostics(result analysis.AnalysisResult) []model.Diagnostic {
	diagnostics := []model.Diagnostic{}
	moduleValues := make(map[string]analysis.ModuleObservation)
	for _, value := range result.Modules {
		if existing, exists := moduleValues[value.ID]; exists && moduleObservationConflicts(existing, value) {
			diagnostics = append(diagnostics, conflictDiagnostic("model_conflicting_module", value.ID, append(existing.SourceReferenceIDs, value.SourceReferenceIDs...), map[string]any{
				"first_observation":       existing,
				"conflicting_observation": value,
			}))
		} else {
			moduleValues[value.ID] = value
		}
	}
	referenceValues := make(map[string]analysis.Reference)
	for _, value := range result.References {
		if existing, exists := referenceValues[value.ID]; exists && referenceObservationConflicts(existing, value) {
			diagnostics = append(diagnostics, conflictDiagnostic("model_conflicting_reference", value.ID, nil, map[string]any{
				"first_observation":       existing,
				"conflicting_observation": value,
			}))
		} else {
			referenceValues[value.ID] = value
		}
	}
	relationshipValues := make(map[string]analysis.RelationshipObservation)
	for _, value := range result.Relationships {
		if existing, exists := relationshipValues[value.ID]; exists && relationshipObservationConflicts(existing, value) {
			diagnostics = append(diagnostics, conflictDiagnostic("model_conflicting_relationship", value.ID, append(existing.SourceReferenceIDs, value.SourceReferenceIDs...), map[string]any{
				"first_observation":       existing,
				"conflicting_observation": value,
			}))
		} else {
			relationshipValues[value.ID] = value
		}
	}
	sourceValues := make(map[string]analysis.SourceReference)
	for _, value := range result.SourceReferences {
		if existing, exists := sourceValues[value.ID]; exists && sourceObservationConflicts(existing, value) {
			diagnostics = append(diagnostics, conflictDiagnostic("model_conflicting_source_reference", value.ID, []string{value.ID}, map[string]any{
				"first_observation":       existing,
				"conflicting_observation": value,
			}))
		} else {
			sourceValues[value.ID] = value
		}
	}
	sortModelDiagnostics(diagnostics)
	unique := diagnostics[:0]
	seen := make(map[string]struct{}, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if _, exists := seen[diagnostic.ID]; exists {
			continue
		}
		seen[diagnostic.ID] = struct{}{}
		unique = append(unique, diagnostic)
	}
	return unique
}

func conflictDiagnostic(code, subject string, sourceIDs []string, metadata map[string]any) model.Diagnostic {
	return model.Diagnostic{
		ID:                 stableID("diagnostic", code, subject),
		Code:               code,
		Severity:           "warning",
		Message:            "Duplicate observations with the same stable identity contain conflicting metadata.",
		Subject:            subject,
		SourceReferenceIDs: sortedUnique(sourceIDs),
		Recoverable:        true,
		Metadata:           metadata,
	}
}

func moduleObservationConflicts(left, right analysis.ModuleObservation) bool {
	return left.Language != right.Language || left.Kind != right.Kind || left.Name != right.Name || left.DisplayName != right.DisplayName || !stringSlicesEqual(left.Hierarchy, right.Hierarchy) || metadataConflicts(left.Metadata, right.Metadata)
}

func referenceObservationConflicts(left, right analysis.Reference) bool {
	return left.Name != right.Name || left.Scope != right.Scope || left.Language != right.Language || metadataConflicts(left.Metadata, right.Metadata)
}

func relationshipObservationConflicts(left, right analysis.RelationshipObservation) bool {
	if left.Type != right.Type || left.FromModuleID != right.FromModuleID || left.ToModuleID != right.ToModuleID || left.ToReferenceID != right.ToReferenceID || !confidenceEqual(left.Confidence, right.Confidence) {
		return true
	}
	return metadataConflicts(left.Metadata, right.Metadata)
}

func sourceObservationConflicts(left, right analysis.SourceReference) bool {
	return normalizedPath(left.Path) != normalizedPath(right.Path) || !positionEqual(left.Start, right.Start) || !positionEqual(left.End, right.End) || left.Symbol != right.Symbol || left.Kind != right.Kind
}

func metadataConflicts(left, right map[string]any) bool {
	for key, rightValue := range right {
		leftValue, exists := left[key]
		if exists && !jsonValuesEqual(leftValue, rightValue) {
			return true
		}
	}
	return false
}

func jsonValuesEqual(left, right any) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftJSON) == string(rightJSON)
}

func stringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func confidenceEqual(left, right *analysis.Confidence) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Basis == right.Basis && left.Score == right.Score
}

func positionEqual(left, right *analysis.Position) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Line == right.Line && left.Column == right.Column
}
