package orchestration

import (
	"encoding/json"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/sourceindex"
	"github.com/buffo/arch-view/internal/model"
)

// CloneAnalysisRun returns an independent run, including the non-serialized
// caches used by scope selection and local consumers. A JSON round trip alone
// is insufficient here because those caches intentionally have json:"-" tags.
func CloneAnalysisRun(value AnalysisRun) AnalysisRun {
	clone := value
	clone.JobPlan = cloneJobPlan(value.JobPlan)
	clone.Scopes = make([]ScopeSummary, len(value.Scopes))
	for index, item := range value.Scopes {
		clone.Scopes[index] = cloneScopeSummary(item)
	}
	clone.Diagnostics = make([]ScopedDiagnostic, len(value.Diagnostics))
	for index, item := range value.Diagnostics {
		clone.Diagnostics[index] = cloneScopedDiagnostic(item)
	}
	clone.Events = make([]JobLifecycleEvent, len(value.Events))
	for index, item := range value.Events {
		clone.Events[index] = cloneJobLifecycleEvent(item)
	}

	if value.Model != nil {
		modelClone := cloneAggregateModel(*value.Model)
		clone.Model = &modelClone
	} else {
		clone.Model = nil
	}
	if value.combinedCanonical.SchemaVersion != "" || value.combinedCanonical.ModelID != "" {
		clone.combinedCanonical = cloneModel(value.combinedCanonical)
	}
	if value.scopeModels != nil {
		clone.scopeModels = make(map[string]model.Model, len(value.scopeModels))
		for key, item := range value.scopeModels {
			clone.scopeModels[key] = cloneModel(item)
		}
	} else {
		clone.scopeModels = nil
	}
	clone.scopeResults = cloneScopeResults(value.scopeResults)
	return clone
}

func cloneScopeResults(values map[string]analysis.AnalysisResult) map[string]analysis.AnalysisResult {
	if values == nil {
		return nil
	}
	result := make(map[string]analysis.AnalysisResult, len(values))
	for key, value := range values {
		result[key] = cloneAnalysisResult(value)
	}
	return result
}

func cloneScopeSummary(value ScopeSummary) ScopeSummary {
	data, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var clone ScopeSummary
	if err := json.Unmarshal(data, &clone); err != nil {
		return value
	}
	return clone
}

func cloneScopedDiagnostic(value ScopedDiagnostic) ScopedDiagnostic {
	clone := value
	clone.Location = clonePosition(value.Location)
	clone.Metadata = cloneAnyMap(value.Metadata)
	return clone
}

func cloneJobLifecycleEvent(value JobLifecycleEvent) JobLifecycleEvent {
	clone := value
	clone.Diagnostic = cloneDiagnosticPointer(value.Diagnostic)
	return clone
}

func cloneDiagnosticPointer(value *analysis.Diagnostic) *analysis.Diagnostic {
	if value == nil {
		return nil
	}
	clone := cloneDiagnostic(*value)
	return &clone
}

func cloneAggregateModel(value AggregateModel) AggregateModel {
	data, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var clone AggregateModel
	if err := json.Unmarshal(data, &clone); err != nil {
		return value
	}
	clone.SourceIndex = sourceindex.CloneSourceIndex(value.SourceIndex)
	return clone
}

func cloneModel(value model.Model) model.Model {
	data, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var clone model.Model
	if err := json.Unmarshal(data, &clone); err != nil {
		return value
	}
	clone.SourceIndex = sourceindex.CloneSourceIndex(value.SourceIndex)
	return clone
}
