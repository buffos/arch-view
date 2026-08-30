// Package canonical converts analyzer output into the versioned architecture
// model and validates the resulting canonical representation.
package canonical

import (
	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
)

// Normalize converts an analyzer result into a deterministic canonical model.
// Duplicate observations are merged, recoverable conflicts become diagnostics,
// and graph projections are derived before the model is returned.
func Normalize(result analysis.AnalysisResult) (model.Model, error) {
	status, err := modelStatus(result.Status)
	if err != nil {
		return model.Model{}, err
	}
	value := model.Model{
		SchemaVersion: model.SchemaVersion,
		Status:        status,
		Project: model.Project{
			RootLabel: result.Project.RootLabel,
			Boundary:  result.Project.Boundary,
			Language:  result.Analyzer.Language,
		},
		Analyzer:         result.Analyzer,
		Modules:          normalizeModules(result.Modules),
		References:       normalizeReferences(result.References),
		SourceReferences: normalizeSourceReferences(result.SourceReferences),
		Relationships:    normalizeRelationships(result.Relationships),
		Diagnostics:      normalizeDiagnostics(result.Diagnostics, result.SourceReferences),
		Derived: model.Derived{
			Cycles:                  []model.CycleGroup{},
			FeedbackRelationshipIDs: []string{},
			Layers:                  []model.Layer{},
			AlgorithmProvenance: map[string]any{
				"cycle_algorithm":    "tarjan-v1",
				"feedback_algorithm": "ordered-cycle-break-v1",
				"layer_algorithm":    "dependency-depth-v1",
			},
		},
		SourceIndex: result.SourceIndex,
	}

	value.Diagnostics = append(value.Diagnostics, normalizationConflictDiagnostics(result)...)
	value.Diagnostics = append(value.Diagnostics, normalizationDiagnostics(value)...)
	sortModelDiagnostics(value.Diagnostics)
	if status == model.StatusComplete {
		for _, diagnostic := range value.Diagnostics {
			if diagnostic.Recoverable {
				value.Status = model.StatusPartial
				break
			}
		}
	}
	model.DeriveGraph(&value)
	value.ModelID = modelID(value)
	if err := Validate(value); err != nil {
		return model.Model{}, err
	}
	return value, nil
}
