package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"path"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

func Normalize(result analysis.AnalysisResult) (Model, error) {
	status, err := modelStatus(result.Status)
	if err != nil {
		return Model{}, err
	}
	model := Model{
		SchemaVersion: SchemaVersion,
		Status:        status,
		Project: Project{
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
		Derived: Derived{
			Cycles:                  []CycleGroup{},
			FeedbackRelationshipIDs: []string{},
			Layers:                  []Layer{},
			AlgorithmProvenance: map[string]any{
				"cycle_algorithm":    "tarjan-v1",
				"feedback_algorithm": "ordered-cycle-break-v1",
				"layer_algorithm":    "dependency-depth-v1",
			},
		},
	}

	model.Diagnostics = append(model.Diagnostics, normalizationConflictDiagnostics(result)...)
	model.Diagnostics = append(model.Diagnostics, normalizationDiagnostics(model)...)
	sortModelDiagnostics(model.Diagnostics)
	if status == StatusComplete {
		for _, diagnostic := range model.Diagnostics {
			if diagnostic.Recoverable {
				model.Status = StatusPartial
				break
			}
		}
	}
	deriveGraph(&model)
	model.ModelID = modelID(model)
	if err := Validate(model); err != nil {
		return Model{}, err
	}
	return model, nil
}

func Validate(model Model) error {
	if model.SchemaVersion != SchemaVersion {
		return analysis.NewHostError(analysis.ErrInvalidModel, "model schema version is unsupported", map[string]any{"schema_version": model.SchemaVersion})
	}
	if strings.TrimSpace(model.ModelID) == "" {
		return analysis.NewHostError(analysis.ErrInvalidModel, "model id is required", nil)
	}
	if model.Status != StatusComplete && model.Status != StatusPartial && model.Status != StatusFailed {
		return analysis.NewHostError(analysis.ErrInvalidModel, "model status is invalid", map[string]any{"status": model.Status})
	}
	if model.Project.RootLabel == "" || model.Project.Boundary == "" || model.Project.Language == "" {
		return analysis.NewHostError(analysis.ErrInvalidModel, "model project metadata is incomplete", nil)
	}
	if model.Analyzer.ID == "" || model.Analyzer.Version == "" || model.Analyzer.Language == "" || model.Analyzer.APIVersion == "" {
		return analysis.NewHostError(analysis.ErrInvalidModel, "model analyzer metadata is incomplete", nil)
	}
	if model.Project.Language != model.Analyzer.Language {
		return analysis.NewHostError(analysis.ErrInvalidModel, "model project and analyzer languages do not match", nil)
	}
	if model.Modules == nil || model.References == nil || model.SourceReferences == nil || model.Relationships == nil || model.Diagnostics == nil {
		return analysis.NewHostError(analysis.ErrInvalidModel, "model collections must be serialized as arrays", nil)
	}
	if model.Derived.Cycles == nil || model.Derived.FeedbackRelationshipIDs == nil || model.Derived.Layers == nil || model.Derived.AlgorithmProvenance == nil {
		return analysis.NewHostError(analysis.ErrInvalidModel, "model derived projections are incomplete", nil)
	}
	if err := validateCanonicalOrdering(model); err != nil {
		return err
	}

	moduleIDs := make(map[string]struct{}, len(model.Modules))
	sourceIDs := make(map[string]struct{}, len(model.SourceReferences))
	for _, source := range model.SourceReferences {
		if source.ID == "" || source.Path == "" || source.Path != normalizedPath(source.Path) || source.Kind == "" || !repositoryRelative(source.Path) || !validPosition(source.Start) || !validPosition(source.End) {
			return analysis.NewHostError(analysis.ErrInvalidModel, "model source reference is invalid", map[string]any{"id": source.ID, "path": source.Path})
		}
		if _, exists := sourceIDs[source.ID]; exists {
			return analysis.NewHostError(analysis.ErrInvalidModel, "model source reference ids must be unique", map[string]any{"id": source.ID})
		}
		sourceIDs[source.ID] = struct{}{}
	}
	for _, module := range model.Modules {
		if module.ID == "" || module.Language == "" || module.Kind == "" || module.Name == "" || module.DisplayName == "" {
			return analysis.NewHostError(analysis.ErrInvalidModel, "model module is incomplete", map[string]any{"id": module.ID})
		}
		if _, exists := moduleIDs[module.ID]; exists {
			return analysis.NewHostError(analysis.ErrInvalidModel, "model module ids must be unique", map[string]any{"id": module.ID})
		}
		moduleIDs[module.ID] = struct{}{}
		for _, segment := range module.Hierarchy {
			if strings.TrimSpace(segment) == "" {
				return analysis.NewHostError(analysis.ErrInvalidModel, "model hierarchy segments cannot be empty", map[string]any{"module_id": module.ID})
			}
		}
		for _, sourceID := range module.SourceReferenceIDs {
			if _, exists := sourceIDs[sourceID]; !exists {
				return analysis.NewHostError(analysis.ErrInvalidModel, "model module references unknown source evidence", map[string]any{"module_id": module.ID, "source_reference_id": sourceID})
			}
		}
	}

	referenceIDs := make(map[string]struct{}, len(model.References))
	for _, reference := range model.References {
		if reference.ID == "" || reference.Name == "" || !validReferenceScope(reference.Scope) {
			return analysis.NewHostError(analysis.ErrInvalidModel, "model reference is incomplete", map[string]any{"id": reference.ID})
		}
		if _, exists := referenceIDs[reference.ID]; exists {
			return analysis.NewHostError(analysis.ErrInvalidModel, "model reference ids must be unique", map[string]any{"id": reference.ID})
		}
		referenceIDs[reference.ID] = struct{}{}
	}

	relationshipIDs := make(map[string]struct{}, len(model.Relationships))
	for _, relationship := range model.Relationships {
		if relationship.ID == "" || relationship.Type == "" || relationship.FromModuleID == "" {
			return analysis.NewHostError(analysis.ErrInvalidModel, "model relationship is incomplete", map[string]any{"id": relationship.ID})
		}
		if _, exists := relationshipIDs[relationship.ID]; exists {
			return analysis.NewHostError(analysis.ErrInvalidModel, "model relationship ids must be unique", map[string]any{"id": relationship.ID})
		}
		relationshipIDs[relationship.ID] = struct{}{}
		if _, exists := moduleIDs[relationship.FromModuleID]; !exists {
			return analysis.NewHostError(analysis.ErrInvalidModel, "model relationship source module is unknown", map[string]any{"id": relationship.ID})
		}
		if (relationship.ToModuleID == "") == (relationship.ToReferenceID == "") {
			return analysis.NewHostError(analysis.ErrInvalidModel, "model relationship must have exactly one target", map[string]any{"id": relationship.ID})
		}
		if relationship.ToModuleID != "" {
			if _, exists := moduleIDs[relationship.ToModuleID]; !exists {
				return analysis.NewHostError(analysis.ErrInvalidModel, "model relationship target module is unknown", map[string]any{"id": relationship.ID})
			}
		}
		if relationship.ToReferenceID != "" {
			if _, exists := referenceIDs[relationship.ToReferenceID]; !exists {
				return analysis.NewHostError(analysis.ErrInvalidModel, "model relationship target reference is unknown", map[string]any{"id": relationship.ID})
			}
		}
		for _, sourceID := range relationship.SourceReferenceIDs {
			if _, exists := sourceIDs[sourceID]; !exists {
				return analysis.NewHostError(analysis.ErrInvalidModel, "model relationship references unknown source evidence", map[string]any{"id": relationship.ID, "source_reference_id": sourceID})
			}
		}
		if relationship.Confidence != nil && (relationship.Confidence.Basis == "" || relationship.Confidence.Score < 0 || relationship.Confidence.Score > 1 || math.IsNaN(relationship.Confidence.Score) || math.IsInf(relationship.Confidence.Score, 0)) {
			return analysis.NewHostError(analysis.ErrInvalidModel, "model relationship confidence is invalid", map[string]any{"id": relationship.ID})
		}
	}
	if err := validateDerivedProjections(model, moduleIDs, relationshipIDs); err != nil {
		return err
	}

	diagnosticIDs := make(map[string]struct{}, len(model.Diagnostics))
	for _, diagnostic := range model.Diagnostics {
		if diagnostic.ID == "" || diagnostic.Code == "" || diagnostic.Message == "" {
			return analysis.NewHostError(analysis.ErrInvalidModel, "model diagnostic is incomplete", map[string]any{"id": diagnostic.ID})
		}
		if diagnostic.Severity != "info" && diagnostic.Severity != "warning" && diagnostic.Severity != "error" {
			return analysis.NewHostError(analysis.ErrInvalidModel, "model diagnostic severity is invalid", map[string]any{"id": diagnostic.ID})
		}
		if _, exists := diagnosticIDs[diagnostic.ID]; exists {
			return analysis.NewHostError(analysis.ErrInvalidModel, "model diagnostic ids must be unique", map[string]any{"id": diagnostic.ID})
		}
		diagnosticIDs[diagnostic.ID] = struct{}{}
		if diagnostic.Path != "" && (diagnostic.Path != normalizedPath(diagnostic.Path) || !repositoryRelative(diagnostic.Path)) {
			return analysis.NewHostError(analysis.ErrInvalidModel, "model diagnostic path is not repository-relative", map[string]any{"id": diagnostic.ID})
		}
		if !validPosition(diagnostic.Location) {
			return analysis.NewHostError(analysis.ErrInvalidModel, "model diagnostic location is invalid", map[string]any{"id": diagnostic.ID})
		}
		for _, sourceID := range diagnostic.SourceReferenceIDs {
			if _, exists := sourceIDs[sourceID]; !exists {
				return analysis.NewHostError(analysis.ErrInvalidModel, "model diagnostic references unknown source evidence", map[string]any{"id": diagnostic.ID, "source_reference_id": sourceID})
			}
		}
	}
	if model.Status == StatusComplete {
		for _, diagnostic := range model.Diagnostics {
			if diagnostic.Severity == "error" || diagnostic.Recoverable {
				return analysis.NewHostError(analysis.ErrInvalidModel, "complete model cannot contain error or recoverable diagnostics", map[string]any{"id": diagnostic.ID})
			}
		}
	}
	if model.Status == StatusPartial {
		recoverable := false
		for _, diagnostic := range model.Diagnostics {
			if diagnostic.Recoverable {
				recoverable = true
				break
			}
		}
		if !recoverable {
			return analysis.NewHostError(analysis.ErrInvalidModel, "partial model must retain a recoverable diagnostic", nil)
		}
	}
	if _, err := json.Marshal(model); err != nil {
		return analysis.WrapHostError(analysis.ErrInvalidModel, "model is not serializable", err, nil)
	}
	if model.ModelID != modelID(model) {
		return analysis.NewHostError(analysis.ErrInvalidModel, "model id does not match canonical model content", map[string]any{"model_id": model.ModelID})
	}
	return nil
}

func modelStatus(status analysis.AnalysisStatus) (Status, error) {
	switch status {
	case analysis.StatusComplete:
		return StatusComplete, nil
	case analysis.StatusPartial:
		return StatusPartial, nil
	case analysis.StatusFailed:
		return StatusFailed, nil
	case analysis.StatusCancelled:
		return "", analysis.NewHostError(analysis.ErrCancelled, "cancelled analysis cannot be normalized into a model", nil)
	default:
		return "", analysis.NewHostError(analysis.ErrInvalidModel, "analysis status cannot be normalized into a model", map[string]any{"status": status})
	}
}

func sortModelDiagnostics(values []Diagnostic) {
	sort.Slice(values, func(i, j int) bool {
		if values[i].ID == values[j].ID {
			return values[i].Code < values[j].Code
		}
		return values[i].ID < values[j].ID
	})
}

func validReferenceScope(scope string) bool {
	switch scope {
	case "external", "standard_library", "unresolved", "dynamic":
		return true
	default:
		return false
	}
}

func validateCanonicalOrdering(model Model) error {
	if !sortedObservationIDs(model.Modules, func(module analysis.ModuleObservation) string { return module.ID }) ||
		!sortedObservationIDs(model.References, func(reference analysis.Reference) string { return reference.ID }) ||
		!sortedObservationIDs(model.SourceReferences, func(source analysis.SourceReference) string { return source.ID }) ||
		!sortedObservationIDs(model.Relationships, func(relationship analysis.RelationshipObservation) string { return relationship.ID }) ||
		!sort.SliceIsSorted(model.Diagnostics, func(i, j int) bool { return model.Diagnostics[i].ID < model.Diagnostics[j].ID }) {
		return analysis.NewHostError(analysis.ErrInvalidModel, "model collections must be sorted by stable identity", nil)
	}
	for _, module := range model.Modules {
		if !sort.StringsAreSorted(module.SourceReferenceIDs) || !sort.StringsAreSorted(module.Tags) {
			return analysis.NewHostError(analysis.ErrInvalidModel, "module evidence and tags must be sorted", map[string]any{"module_id": module.ID})
		}
	}
	for _, relationship := range model.Relationships {
		if !sort.StringsAreSorted(relationship.SourceReferenceIDs) {
			return analysis.NewHostError(analysis.ErrInvalidModel, "relationship evidence must be sorted", map[string]any{"relationship_id": relationship.ID})
		}
	}
	if !sort.StringsAreSorted(model.Derived.FeedbackRelationshipIDs) || !sort.SliceIsSorted(model.Derived.Cycles, func(i, j int) bool { return model.Derived.Cycles[i].ID < model.Derived.Cycles[j].ID }) || !sort.SliceIsSorted(model.Derived.Layers, func(i, j int) bool { return model.Derived.Layers[i].Layer < model.Derived.Layers[j].Layer }) {
		return analysis.NewHostError(analysis.ErrInvalidModel, "derived collections must be sorted", nil)
	}
	for _, cycle := range model.Derived.Cycles {
		if !sort.StringsAreSorted(cycle.ModuleIDs) || !sort.StringsAreSorted(cycle.RelationshipIDs) {
			return analysis.NewHostError(analysis.ErrInvalidModel, "cycle members must be sorted", map[string]any{"cycle_id": cycle.ID})
		}
	}
	for _, layer := range model.Derived.Layers {
		if !sort.StringsAreSorted(layer.ModuleIDs) {
			return analysis.NewHostError(analysis.ErrInvalidModel, "layer members must be sorted", map[string]any{"layer": layer.Layer})
		}
	}
	return nil
}

func validateDerivedProjections(model Model, moduleIDs map[string]struct{}, relationshipIDs map[string]struct{}) error {
	cycleIDs := make(map[string]struct{}, len(model.Derived.Cycles))
	for _, cycle := range model.Derived.Cycles {
		if cycle.ID == "" {
			return analysis.NewHostError(analysis.ErrInvalidModel, "cycle id is required", nil)
		}
		if _, exists := cycleIDs[cycle.ID]; exists {
			return analysis.NewHostError(analysis.ErrInvalidModel, "cycle ids must be unique", map[string]any{"cycle_id": cycle.ID})
		}
		cycleIDs[cycle.ID] = struct{}{}
		for _, moduleID := range cycle.ModuleIDs {
			if _, exists := moduleIDs[moduleID]; !exists {
				return analysis.NewHostError(analysis.ErrInvalidModel, "cycle references unknown module", map[string]any{"cycle_id": cycle.ID, "module_id": moduleID})
			}
		}
		for _, relationshipID := range cycle.RelationshipIDs {
			if _, exists := relationshipIDs[relationshipID]; !exists {
				return analysis.NewHostError(analysis.ErrInvalidModel, "cycle references unknown relationship", map[string]any{"cycle_id": cycle.ID, "relationship_id": relationshipID})
			}
		}
	}
	feedbackIDs := make(map[string]struct{}, len(model.Derived.FeedbackRelationshipIDs))
	for _, relationshipID := range model.Derived.FeedbackRelationshipIDs {
		if _, exists := relationshipIDs[relationshipID]; !exists {
			return analysis.NewHostError(analysis.ErrInvalidModel, "feedback projection references unknown relationship", map[string]any{"relationship_id": relationshipID})
		}
		if _, exists := feedbackIDs[relationshipID]; exists {
			return analysis.NewHostError(analysis.ErrInvalidModel, "feedback relationship ids must be unique", map[string]any{"relationship_id": relationshipID})
		}
		feedbackIDs[relationshipID] = struct{}{}
	}
	layerModules := make(map[string]struct{}, len(moduleIDs))
	for _, layer := range model.Derived.Layers {
		if layer.Layer < 0 {
			return analysis.NewHostError(analysis.ErrInvalidModel, "layer number cannot be negative", map[string]any{"layer": layer.Layer})
		}
		for _, moduleID := range layer.ModuleIDs {
			if _, exists := moduleIDs[moduleID]; !exists {
				return analysis.NewHostError(analysis.ErrInvalidModel, "layer references unknown module", map[string]any{"layer": layer.Layer, "module_id": moduleID})
			}
			if _, exists := layerModules[moduleID]; exists {
				return analysis.NewHostError(analysis.ErrInvalidModel, "layer assigns a module more than once", map[string]any{"module_id": moduleID})
			}
			layerModules[moduleID] = struct{}{}
		}
	}
	if len(layerModules) != len(moduleIDs) {
		return analysis.NewHostError(analysis.ErrInvalidModel, "layer projection must assign every module", nil)
	}
	return nil
}

func sortedObservationIDs[T any](values []T, id func(T) string) bool {
	return sort.SliceIsSorted(values, func(i, j int) bool { return id(values[i]) < id(values[j]) })
}

func normalizeModules(values []analysis.ModuleObservation) []analysis.ModuleObservation {
	byID := make(map[string]analysis.ModuleObservation, len(values))
	for _, value := range values {
		value.SourceReferenceIDs = sortedUnique(value.SourceReferenceIDs)
		value.Tags = sortedUnique(value.Tags)
		value.Hierarchy = append([]string(nil), value.Hierarchy...)
		if value.Metadata == nil {
			value.Metadata = map[string]any{}
		}
		if existing, ok := byID[value.ID]; ok {
			existing.SourceReferenceIDs = sortedUnique(append(existing.SourceReferenceIDs, value.SourceReferenceIDs...))
			existing.Tags = sortedUnique(append(existing.Tags, value.Tags...))
			existing.Metadata = mergeMetadata(existing.Metadata, value.Metadata)
			byID[value.ID] = existing
		} else {
			byID[value.ID] = value
		}
	}
	result := make([]analysis.ModuleObservation, 0, len(byID))
	for _, value := range byID {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func normalizeReferences(values []analysis.Reference) []analysis.Reference {
	byID := make(map[string]analysis.Reference, len(values))
	for _, value := range values {
		if value.Metadata == nil {
			value.Metadata = map[string]any{}
		}
		if existing, ok := byID[value.ID]; ok {
			existing.Metadata = mergeMetadata(existing.Metadata, value.Metadata)
			byID[value.ID] = existing
		} else {
			byID[value.ID] = value
		}
	}
	result := make([]analysis.Reference, 0, len(byID))
	for _, value := range byID {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func normalizeSourceReferences(values []analysis.SourceReference) []analysis.SourceReference {
	byID := make(map[string]analysis.SourceReference, len(values))
	for _, value := range values {
		value.Path = normalizedPath(value.Path)
		if existing, ok := byID[value.ID]; ok {
			if existing.Start == nil {
				existing.Start = value.Start
			}
			if existing.End == nil {
				existing.End = value.End
			}
			if existing.Symbol == "" {
				existing.Symbol = value.Symbol
			}
			byID[value.ID] = existing
		} else {
			byID[value.ID] = value
		}
	}
	result := make([]analysis.SourceReference, 0, len(byID))
	for _, value := range byID {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func normalizeRelationships(values []analysis.RelationshipObservation) []analysis.RelationshipObservation {
	byID := make(map[string]analysis.RelationshipObservation, len(values))
	for _, value := range values {
		value.SourceReferenceIDs = sortedUnique(value.SourceReferenceIDs)
		if value.Metadata == nil {
			value.Metadata = map[string]any{}
		}
		if existing, ok := byID[value.ID]; ok {
			existing.SourceReferenceIDs = sortedUnique(append(existing.SourceReferenceIDs, value.SourceReferenceIDs...))
			existing.Metadata = mergeMetadata(existing.Metadata, value.Metadata)
			byID[value.ID] = existing
		} else {
			byID[value.ID] = value
		}
	}
	result := make([]analysis.RelationshipObservation, 0, len(byID))
	for _, value := range byID {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func normalizeDiagnostics(values []analysis.Diagnostic, sources []analysis.SourceReference) []Diagnostic {
	pathSources := make(map[string][]string)
	for _, source := range sources {
		pathSources[normalizedPath(source.Path)] = append(pathSources[normalizedPath(source.Path)], source.ID)
	}
	result := make([]Diagnostic, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		pathValue := normalizedPath(value.Path)
		sourceIDs := sortedUnique(pathSources[pathValue])
		converted := Diagnostic{
			ID:                 stableID("diagnostic", value.Code, value.Severity, value.Message, value.Subject, pathValue, positionKey(value.Location)),
			Code:               value.Code,
			Severity:           value.Severity,
			Message:            value.Message,
			Subject:            value.Subject,
			Path:               pathValue,
			Location:           clonePosition(value.Location),
			SourceReferenceIDs: sourceIDs,
			Recoverable:        value.Recoverable,
			Metadata:           cloneMetadata(value.Metadata),
		}
		if _, exists := seen[converted.ID]; exists {
			continue
		}
		seen[converted.ID] = struct{}{}
		result = append(result, converted)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func normalizationDiagnostics(model Model) []Diagnostic {
	moduleIDs := make(map[string]struct{}, len(model.Modules))
	for _, module := range model.Modules {
		moduleIDs[module.ID] = struct{}{}
	}
	result := []Diagnostic{}
	for _, relationship := range model.Relationships {
		if _, exists := moduleIDs[relationship.FromModuleID]; !exists {
			result = append(result, Diagnostic{
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

func normalizationConflictDiagnostics(result analysis.AnalysisResult) []Diagnostic {
	diagnostics := []Diagnostic{}
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

func conflictDiagnostic(code, subject string, sourceIDs []string, metadata map[string]any) Diagnostic {
	return Diagnostic{
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

func mergeMetadata(left, right map[string]any) map[string]any {
	result := cloneMetadata(left)
	if result == nil {
		result = map[string]any{}
	}
	for key, value := range right {
		if _, exists := result[key]; !exists {
			result[key] = value
		}
	}
	return result
}

func cloneMetadata(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	result := make(map[string]any, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}

func sortedUnique(values []string) []string {
	result := append([]string{}, values...)
	sort.Strings(result)
	if len(result) < 2 {
		return result
	}
	write := 1
	for _, value := range result[1:] {
		if value != result[write-1] {
			result[write] = value
			write++
		}
	}
	return result[:write]
}

func normalizedPath(value string) string {
	if value == "" {
		return ""
	}
	return path.Clean(strings.ReplaceAll(value, "\\", "/"))
}

func repositoryRelative(value string) bool {
	normalized := normalizedPath(value)
	if normalized == "" || normalized == "." || strings.HasPrefix(normalized, "/") || windowsAbsolutePath(normalized) {
		return false
	}
	return normalized != ".." && !strings.HasPrefix(normalized, "../")
}

func windowsAbsolutePath(value string) bool {
	return len(value) >= 2 && ((value[0] >= 'a' && value[0] <= 'z') || (value[0] >= 'A' && value[0] <= 'Z')) && value[1] == ':'
}

func validPosition(position *analysis.Position) bool {
	return position == nil || (position.Line > 0 && position.Column > 0)
}

func clonePosition(position *analysis.Position) *analysis.Position {
	if position == nil {
		return nil
	}
	copy := *position
	return &copy
}

func positionKey(position *analysis.Position) string {
	if position == nil {
		return ""
	}
	return fmt.Sprintf("%d:%d", position.Line, position.Column)
}

func modelID(model Model) string {
	model.ModelID = ""
	data, _ := json.Marshal(model)
	sum := sha256.Sum256(data)
	return "model-" + hex.EncodeToString(sum[:12])
}

func stableID(kind string, parts ...string) string {
	payload := kind + "\x00" + strings.Join(parts, "\x00")
	sum := sha256.Sum256([]byte(payload))
	return "model:" + kind + ":" + hex.EncodeToString(sum[:8])
}
