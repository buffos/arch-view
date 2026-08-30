package adapter

import (
	"fmt"

	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/model/canonical"
	"github.com/buffo/arch-view/internal/quality"
)

func EvaluationInputFromModel(value model.Model) (quality.EvaluationInput, error) {
	if err := canonical.Validate(value); err != nil {
		return quality.EvaluationInput{}, fmt.Errorf("canonical model cannot be adapted for quality evaluation: %w", err)
	}
	result := quality.EvaluationInput{Architecture: ArchitectureModelFromModel(value), Options: map[string]any{"model_id": value.ModelID}}
	if value.SourceIndex != nil {
		source, err := EvaluationInputFromSourceIndex(*value.SourceIndex)
		if err != nil {
			return quality.EvaluationInput{}, err
		}
		result.SourceSnapshots = source.SourceSnapshots
	}
	return result, nil
}

func ArchitectureModelFromModel(value model.Model) *quality.ArchitectureModel {
	result := &quality.ArchitectureModel{
		ScopeID:           value.ModelID,
		Aggregate:         value.SourceIndex != nil && value.SourceIndex.Projection != nil,
		SourceSnapshotIDs: []string{},
		Modules:           make([]quality.ArchitectureModule, 0, len(value.Modules)),
		Relationships:     make([]quality.ArchitectureRelationship, 0, len(value.Relationships)),
		Cycles:            make([]quality.ArchitectureCycle, 0, len(value.Derived.Cycles)),
		Layers:            make([]quality.ArchitectureLayer, 0, len(value.Derived.Layers)),
	}
	if value.SourceIndex != nil {
		for _, snapshot := range value.SourceIndex.Snapshots {
			result.SourceSnapshotIDs = append(result.SourceSnapshotIDs, snapshot.SnapshotID)
		}
	}
	for _, module := range value.Modules {
		result.Modules = append(result.Modules, quality.ArchitectureModule{ID: module.ID, StableKey: module.ID, Name: module.Name, Hierarchy: append([]string(nil), module.Hierarchy...), Tags: append([]string(nil), module.Tags...)})
	}
	for _, relationship := range value.Relationships {
		result.Relationships = append(result.Relationships, quality.ArchitectureRelationship{ID: relationship.ID, Type: relationship.Type, FromModuleID: relationship.FromModuleID, ToModuleID: relationship.ToModuleID, ToReferenceID: relationship.ToReferenceID, SourceReferenceIDs: append([]string(nil), relationship.SourceReferenceIDs...)})
	}
	for _, cycle := range value.Derived.Cycles {
		result.Cycles = append(result.Cycles, quality.ArchitectureCycle{ID: cycle.ID, ModuleIDs: append([]string(nil), cycle.ModuleIDs...), RelationshipIDs: append([]string(nil), cycle.RelationshipIDs...)})
	}
	for _, layer := range value.Derived.Layers {
		result.Layers = append(result.Layers, quality.ArchitectureLayer{Layer: layer.Layer, ModuleIDs: append([]string(nil), layer.ModuleIDs...)})
	}
	return result
}
