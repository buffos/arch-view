package canonical

import (
	"encoding/json"
	"math"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
)

// Validate checks the invariants of a canonical architecture model.
func Validate(value model.Model) error {
	if value.SchemaVersion != model.SchemaVersion {
		return analysis.NewHostError(analysis.ErrInvalidModel, "model schema version is unsupported", map[string]any{"schema_version": value.SchemaVersion})
	}
	if strings.TrimSpace(value.ModelID) == "" {
		return analysis.NewHostError(analysis.ErrInvalidModel, "model id is required", nil)
	}
	if value.Status != model.StatusComplete && value.Status != model.StatusPartial && value.Status != model.StatusFailed {
		return analysis.NewHostError(analysis.ErrInvalidModel, "model status is invalid", map[string]any{"status": value.Status})
	}
	if value.Project.RootLabel == "" || value.Project.Boundary == "" || value.Project.Language == "" {
		return analysis.NewHostError(analysis.ErrInvalidModel, "model project metadata is incomplete", nil)
	}
	if value.Analyzer.ID == "" || value.Analyzer.Version == "" || value.Analyzer.Language == "" || value.Analyzer.APIVersion == "" {
		return analysis.NewHostError(analysis.ErrInvalidModel, "model analyzer metadata is incomplete", nil)
	}
	if value.Project.Language != value.Analyzer.Language {
		return analysis.NewHostError(analysis.ErrInvalidModel, "model project and analyzer languages do not match", nil)
	}
	if value.Modules == nil || value.References == nil || value.SourceReferences == nil || value.Relationships == nil || value.Diagnostics == nil {
		return analysis.NewHostError(analysis.ErrInvalidModel, "model collections must be serialized as arrays", nil)
	}
	if value.SourceIndex != nil {
		if err := analysis.ValidateSourceIndex(*value.SourceIndex); err != nil {
			return analysis.WrapHostError(analysis.ErrInvalidModel, "model source index is invalid", err, nil)
		}
	}
	if value.Derived.Cycles == nil || value.Derived.FeedbackRelationshipIDs == nil || value.Derived.Layers == nil || value.Derived.AlgorithmProvenance == nil {
		return analysis.NewHostError(analysis.ErrInvalidModel, "model derived projections are incomplete", nil)
	}
	if err := validateCanonicalOrdering(value); err != nil {
		return err
	}

	moduleIDs := make(map[string]struct{}, len(value.Modules))
	sourceIDs := make(map[string]struct{}, len(value.SourceReferences))
	for _, source := range value.SourceReferences {
		if source.ID == "" || source.Path == "" || source.Path != normalizedPath(source.Path) || source.Kind == "" || !repositoryRelative(source.Path) || !validPosition(source.Start) || !validPosition(source.End) {
			return analysis.NewHostError(analysis.ErrInvalidModel, "model source reference is invalid", map[string]any{"id": source.ID, "path": source.Path})
		}
		if _, exists := sourceIDs[source.ID]; exists {
			return analysis.NewHostError(analysis.ErrInvalidModel, "model source reference ids must be unique", map[string]any{"id": source.ID})
		}
		sourceIDs[source.ID] = struct{}{}
	}
	for _, module := range value.Modules {
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

	referenceIDs := make(map[string]struct{}, len(value.References))
	for _, reference := range value.References {
		if reference.ID == "" || reference.Name == "" || !validReferenceScope(reference.Scope) {
			return analysis.NewHostError(analysis.ErrInvalidModel, "model reference is incomplete", map[string]any{"id": reference.ID})
		}
		if _, exists := referenceIDs[reference.ID]; exists {
			return analysis.NewHostError(analysis.ErrInvalidModel, "model reference ids must be unique", map[string]any{"id": reference.ID})
		}
		referenceIDs[reference.ID] = struct{}{}
	}

	relationshipIDs := make(map[string]struct{}, len(value.Relationships))
	for _, relationship := range value.Relationships {
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
	if err := validateDerivedProjections(value, moduleIDs, relationshipIDs); err != nil {
		return err
	}

	diagnosticIDs := make(map[string]struct{}, len(value.Diagnostics))
	for _, diagnostic := range value.Diagnostics {
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
	if value.Status == model.StatusComplete {
		for _, diagnostic := range value.Diagnostics {
			if diagnostic.Severity == "error" || diagnostic.Recoverable {
				return analysis.NewHostError(analysis.ErrInvalidModel, "complete model cannot contain error or recoverable diagnostics", map[string]any{"id": diagnostic.ID})
			}
		}
	}
	if value.Status == model.StatusPartial {
		recoverable := false
		for _, diagnostic := range value.Diagnostics {
			if diagnostic.Recoverable {
				recoverable = true
				break
			}
		}
		if !recoverable {
			return analysis.NewHostError(analysis.ErrInvalidModel, "partial model must retain a recoverable diagnostic", nil)
		}
	}
	if _, err := json.Marshal(value); err != nil {
		return analysis.WrapHostError(analysis.ErrInvalidModel, "model is not serializable", err, nil)
	}
	if value.ModelID != modelID(value) {
		return analysis.NewHostError(analysis.ErrInvalidModel, "model id does not match canonical model content", map[string]any{"model_id": value.ModelID})
	}
	return nil
}

func modelStatus(status analysis.AnalysisStatus) (model.Status, error) {
	switch status {
	case analysis.StatusComplete:
		return model.StatusComplete, nil
	case analysis.StatusPartial:
		return model.StatusPartial, nil
	case analysis.StatusFailed:
		return model.StatusFailed, nil
	case analysis.StatusCancelled:
		return "", analysis.NewHostError(analysis.ErrCancelled, "cancelled analysis cannot be normalized into a model", nil)
	default:
		return "", analysis.NewHostError(analysis.ErrInvalidModel, "analysis status cannot be normalized into a model", map[string]any{"status": status})
	}
}

func sortModelDiagnostics(values []model.Diagnostic) {
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

func validateCanonicalOrdering(value model.Model) error {
	if !sortedObservationIDs(value.Modules, func(module analysis.ModuleObservation) string { return module.ID }) ||
		!sortedObservationIDs(value.References, func(reference analysis.Reference) string { return reference.ID }) ||
		!sortedObservationIDs(value.SourceReferences, func(source analysis.SourceReference) string { return source.ID }) ||
		!sortedObservationIDs(value.Relationships, func(relationship analysis.RelationshipObservation) string { return relationship.ID }) ||
		!sort.SliceIsSorted(value.Diagnostics, func(i, j int) bool { return value.Diagnostics[i].ID < value.Diagnostics[j].ID }) {
		return analysis.NewHostError(analysis.ErrInvalidModel, "model collections must be sorted by stable identity", nil)
	}
	for _, module := range value.Modules {
		if !sort.StringsAreSorted(module.SourceReferenceIDs) || !sort.StringsAreSorted(module.Tags) {
			return analysis.NewHostError(analysis.ErrInvalidModel, "module evidence and tags must be sorted", map[string]any{"module_id": module.ID})
		}
	}
	for _, relationship := range value.Relationships {
		if !sort.StringsAreSorted(relationship.SourceReferenceIDs) {
			return analysis.NewHostError(analysis.ErrInvalidModel, "relationship evidence must be sorted", map[string]any{"relationship_id": relationship.ID})
		}
	}
	if !sort.StringsAreSorted(value.Derived.FeedbackRelationshipIDs) || !sort.SliceIsSorted(value.Derived.Cycles, func(i, j int) bool { return value.Derived.Cycles[i].ID < value.Derived.Cycles[j].ID }) || !sort.SliceIsSorted(value.Derived.Layers, func(i, j int) bool { return value.Derived.Layers[i].Layer < value.Derived.Layers[j].Layer }) {
		return analysis.NewHostError(analysis.ErrInvalidModel, "derived collections must be sorted", nil)
	}
	for _, cycle := range value.Derived.Cycles {
		if !sort.StringsAreSorted(cycle.ModuleIDs) || !sort.StringsAreSorted(cycle.RelationshipIDs) {
			return analysis.NewHostError(analysis.ErrInvalidModel, "cycle members must be sorted", map[string]any{"cycle_id": cycle.ID})
		}
	}
	for _, layer := range value.Derived.Layers {
		if !sort.StringsAreSorted(layer.ModuleIDs) {
			return analysis.NewHostError(analysis.ErrInvalidModel, "layer members must be sorted", map[string]any{"layer": layer.Layer})
		}
	}
	return nil
}

func validateDerivedProjections(value model.Model, moduleIDs map[string]struct{}, relationshipIDs map[string]struct{}) error {
	cycleIDs := make(map[string]struct{}, len(value.Derived.Cycles))
	for _, cycle := range value.Derived.Cycles {
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
	feedbackIDs := make(map[string]struct{}, len(value.Derived.FeedbackRelationshipIDs))
	for _, relationshipID := range value.Derived.FeedbackRelationshipIDs {
		if _, exists := relationshipIDs[relationshipID]; !exists {
			return analysis.NewHostError(analysis.ErrInvalidModel, "feedback projection references unknown relationship", map[string]any{"relationship_id": relationshipID})
		}
		if _, exists := feedbackIDs[relationshipID]; exists {
			return analysis.NewHostError(analysis.ErrInvalidModel, "feedback relationship ids must be unique", map[string]any{"relationship_id": relationshipID})
		}
		feedbackIDs[relationshipID] = struct{}{}
	}
	layerModules := make(map[string]struct{}, len(moduleIDs))
	for _, layer := range value.Derived.Layers {
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
