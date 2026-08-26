package canonical

import (
	"sort"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
)

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

func normalizeDiagnostics(values []analysis.Diagnostic, sources []analysis.SourceReference) []model.Diagnostic {
	pathSources := make(map[string][]string)
	for _, source := range sources {
		pathSources[normalizedPath(source.Path)] = append(pathSources[normalizedPath(source.Path)], source.ID)
	}
	result := make([]model.Diagnostic, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		pathValue := normalizedPath(value.Path)
		sourceIDs := sortedUnique(pathSources[pathValue])
		converted := model.Diagnostic{
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
