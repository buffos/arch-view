package clojureanalyzer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

type clojurePolymorphicObservation struct {
	FromModuleID string
	Source       analysis.SourceReference
	Kind         string
	Name         string
}

// extractClojureSafetyObservations scans parsed forms for statically
// recognizable polymorphic and dynamic-loading constructs. It does not
// expand macros or invoke any Clojure runtime behavior.
func extractClojureSafetyObservations(platform string, files []clojureFileObservation) ([]clojurePolymorphicObservation, []clojureDynamicObservation, []analysis.SourceReference, []analysis.Diagnostic) {
	polymorphic := make([]clojurePolymorphicObservation, 0)
	dynamic := make([]clojureDynamicObservation, 0)
	sources := make(map[string]analysis.SourceReference)
	diagnostics := make([]analysis.Diagnostic, 0)
	for _, file := range files {
		for _, form := range file.Forms {
			if form == nil || form == file.NamespaceForm || file.ModuleID == "" {
				continue
			}
			var visit func(*cljForm)
			visit = func(current *cljForm) {
				if current == nil {
					return
				}
				if current.Quoted {
					return
				}
				if current.Kind == formReaderConditional {
					contexts, issues := expandClojureForm(current, platform, nil, false)
					for _, issue := range issues {
						diagnostics = append(diagnostics, analysis.Diagnostic{
							Code:        "clojure_reader_conditional",
							Severity:    "warning",
							Message:     issue.Message,
							Path:        file.Path,
							Location:    issue.Location,
							Recoverable: true,
						})
					}
					for _, context := range contexts {
						visit(context.Form)
					}
					return
				}
				if current.Kind == formList && len(current.Items) > 0 {
					operator := formSymbol(current.Items[0])
					if operator == "quote" || operator == "var" {
						return
					}
					switch operator {
					case "defprotocol", "defmulti":
						if len(current.Items) >= 2 {
							name := clojureFormName(current.Items[1])
							sourceID := clojureStableID("polymorphic", file.Path, operator, positionKey(current.Start))
							source := analysis.SourceReference{ID: sourceID, Path: file.Path, Start: clonePosition(current.Start), End: clonePosition(current.End), Symbol: operator, Kind: "polymorphic"}
							sources[sourceID] = source
							polymorphic = append(polymorphic, clojurePolymorphicObservation{FromModuleID: file.ModuleID, Source: source, Kind: operator, Name: name})
						}
					case "require", "use", "load", "load-file", "load-string", "eval", "resolve", "refer":
						target := ""
						if len(current.Items) >= 2 {
							target = clojureDynamicTarget(operator, current.Items[1])
						}
						sourceID := clojureStableID("dynamic", file.Path, operator, positionKey(current.Start))
						source := analysis.SourceReference{ID: sourceID, Path: file.Path, Start: clonePosition(current.Start), End: clonePosition(current.End), Symbol: operator, Kind: "dynamic"}
						sources[sourceID] = source
						dynamic = append(dynamic, clojureDynamicObservation{FromModuleID: file.ModuleID, Source: source, Function: operator, Target: target})
						diagnostics = append(diagnostics, analysis.Diagnostic{
							Code:        "clojure_dynamic_loading",
							Severity:    "warning",
							Message:     fmt.Sprintf("Clojure form %s was retained as a dynamic reference and was not executed.", operator),
							Subject:     target,
							Path:        file.Path,
							Location:    clonePosition(current.Start),
							Recoverable: true,
							Metadata: map[string]any{
								"dynamic_function":    operator,
								"dynamic_target":      target,
								"source_reference_id": sourceID,
							},
						})
					}
				}
				for _, child := range current.Items {
					visit(child)
				}
			}
			visit(form)
		}
	}
	sort.Slice(polymorphic, func(i, j int) bool {
		if polymorphic[i].FromModuleID != polymorphic[j].FromModuleID {
			return polymorphic[i].FromModuleID < polymorphic[j].FromModuleID
		}
		return polymorphic[i].Source.ID < polymorphic[j].Source.ID
	})
	sort.Slice(dynamic, func(i, j int) bool {
		if dynamic[i].FromModuleID != dynamic[j].FromModuleID {
			return dynamic[i].FromModuleID < dynamic[j].FromModuleID
		}
		return dynamic[i].Source.ID < dynamic[j].Source.ID
	})
	sortClojureDiagnostics(diagnostics)
	return polymorphic, dynamic, sortedClojureSources(sources), diagnostics
}

func clojureFormName(form *cljForm) string {
	if form == nil {
		return ""
	}
	if symbol := formSymbol(form); symbol != "" {
		return symbol
	}
	if value, ok := formString(form); ok {
		return value
	}
	if form.Kind == formVector || form.Kind == formList || form.Kind == formSet {
		parts := make([]string, 0, len(form.Items))
		for _, child := range form.Items {
			if value := clojureFormName(child); value != "" {
				parts = append(parts, value)
			}
		}
		return "[" + strings.Join(parts, " ") + "]"
	}
	return form.Atom
}

func clojureDynamicTarget(operator string, form *cljForm) string {
	if form == nil {
		return ""
	}
	if value, ok := formString(form); ok {
		switch operator {
		case "load-file", "load":
			return value
		case "require", "use", "resolve", "refer":
			if validClojureNamespace(value) {
				return value
			}
			return "<dynamic-string>"
		default:
			return "<dynamic-string>"
		}
	}
	if symbol := formSymbol(form); symbol != "" {
		return symbol
	}
	return "<dynamic-form>"
}

func buildClojurePolymorphicMetadata(modules []analysis.ModuleObservation, observations []clojurePolymorphicObservation) {
	byID := make(map[string]*analysis.ModuleObservation, len(modules))
	for index := range modules {
		byID[modules[index].ID] = &modules[index]
	}
	for _, observation := range observations {
		module := byID[observation.FromModuleID]
		if module == nil {
			continue
		}
		module.Tags = appendUniqueStrings(module.Tags, "polymorphic")
		sort.Strings(module.Tags)
		module.SourceReferenceIDs = appendUniqueStrings(module.SourceReferenceIDs, observation.Source.ID)
		sort.Strings(module.SourceReferenceIDs)
		values, _ := module.Metadata["polymorphic_forms"].([]string)
		values = appendUniqueStrings(values, observation.Kind+":"+observation.Name)
		sort.Strings(values)
		module.Metadata["polymorphic_forms"] = values
		module.Metadata["polymorphic"] = true
	}
}

func buildClojureDynamicObservations(dynamic []clojureDynamicObservation) ([]analysis.RelationshipObservation, []analysis.Reference, []analysis.Diagnostic) {
	relationships := make(map[string]analysis.RelationshipObservation)
	references := make(map[string]analysis.Reference)
	diagnostics := make([]analysis.Diagnostic, 0)
	for _, observation := range dynamic {
		target := observation.Target
		if target == "" {
			target = "<dynamic>"
		}
		referenceID := clojureStableID("reference", "dynamic", target)
		reference := references[referenceID]
		if reference.ID == "" {
			reference = analysis.Reference{ID: referenceID, Name: target, Scope: "dynamic", Language: "clojure", Metadata: map[string]any{}}
		}
		metadata := map[string]any{
			"target_scope":      "dynamic",
			"dynamic_functions": []string{observation.Function},
			"dynamic_targets":   []string{target},
			"resolution":        "dynamic",
		}
		mergeClojureMetadata(reference.Metadata, metadata)
		references[referenceID] = reference

		relationshipID := clojureStableID("relationship", observation.FromModuleID, "dynamic", target)
		relationship := relationships[relationshipID]
		if relationship.ID == "" {
			relationship = analysis.RelationshipObservation{ID: relationshipID, Type: "depends_on", FromModuleID: observation.FromModuleID, ToReferenceID: referenceID, SourceReferenceIDs: []string{}, Confidence: &analysis.Confidence{Basis: "dynamic", Score: 0.2}, Metadata: map[string]any{}}
		}
		relationship.SourceReferenceIDs = appendUniqueStrings(relationship.SourceReferenceIDs, observation.Source.ID)
		mergeClojureMetadata(relationship.Metadata, metadata)
		relationships[relationshipID] = relationship
		diagnostics = append(diagnostics, analysis.Diagnostic{
			Code:        "clojure_dynamic_reference",
			Severity:    "warning",
			Message:     fmt.Sprintf("Clojure dynamic form %s targeting %q could not be resolved statically; it was retained as a dynamic reference.", observation.Function, target),
			Subject:     target,
			Path:        observation.Source.Path,
			Location:    clonePosition(observation.Source.Start),
			Recoverable: true,
			Metadata: map[string]any{
				"source_reference_id": observation.Source.ID,
				"reference_id":        referenceID,
				"dynamic_function":    observation.Function,
			},
		})
	}
	resultRelationships := make([]analysis.RelationshipObservation, 0, len(relationships))
	for _, relationship := range relationships {
		sort.Strings(relationship.SourceReferenceIDs)
		resultRelationships = append(resultRelationships, relationship)
	}
	sort.Slice(resultRelationships, func(i, j int) bool { return resultRelationships[i].ID < resultRelationships[j].ID })
	resultReferences := make([]analysis.Reference, 0, len(references))
	for _, reference := range references {
		resultReferences = append(resultReferences, reference)
	}
	sort.Slice(resultReferences, func(i, j int) bool { return resultReferences[i].ID < resultReferences[j].ID })
	sortClojureDiagnostics(diagnostics)
	return resultRelationships, resultReferences, diagnostics
}
