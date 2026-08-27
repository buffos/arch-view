package clojureanalyzer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

type clojureFormContext struct {
	Form        *cljForm
	Platforms   []string
	Conditional bool
}

type clojureReaderIssue struct {
	Location *analysis.Position
	Message  string
}

type clojureReaderBranch struct {
	Selector string
	Form     *cljForm
}

// extractClojureDependencies reads only namespace declaration clauses. The
// returned source references point at the static spelling that produced each
// observation; no require, macro, or runtime loader is invoked.
func extractClojureDependencies(project Project, files []clojureFileObservation) ([]clojureDependencyObservation, []analysis.SourceReference, []analysis.Diagnostic) {
	dependencies := make([]clojureDependencyObservation, 0)
	sources := make(map[string]analysis.SourceReference)
	diagnostics := make([]analysis.Diagnostic, 0)
	for _, file := range files {
		if file.Namespace == "" || file.NamespaceForm == nil || len(file.NamespaceForm.Items) < 3 {
			continue
		}
		for _, form := range file.NamespaceForm.Items[2:] {
			contexts, issues := expandClojureForm(form, project.Platform, nil, false)
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
				if context.Form == nil || context.Form.Kind != formList || len(context.Form.Items) == 0 {
					continue
				}
				clause := formKeyword(context.Form.Items[0])
				if !isClojureDependencyClause(clause) {
					continue
				}
				parsed, parsedSources, parsedDiagnostics := parseClojureDependencyClause(file, context, clause)
				dependencies = append(dependencies, parsed...)
				for _, source := range parsedSources {
					sources[source.ID] = source
				}
				diagnostics = append(diagnostics, parsedDiagnostics...)
			}
		}
	}
	sort.Slice(dependencies, func(i, j int) bool {
		if dependencies[i].FromModuleID != dependencies[j].FromModuleID {
			return dependencies[i].FromModuleID < dependencies[j].FromModuleID
		}
		if dependencies[i].TargetNamespace != dependencies[j].TargetNamespace {
			return dependencies[i].TargetNamespace < dependencies[j].TargetNamespace
		}
		return dependencies[i].Source.ID < dependencies[j].Source.ID
	})
	return dependencies, sortedClojureSources(sources), diagnostics
}

func expandClojureForm(form *cljForm, platform string, inheritedPlatforms []string, inheritedConditional bool) ([]clojureFormContext, []clojureReaderIssue) {
	if form == nil {
		return nil, nil
	}
	if form.Kind != formReaderConditional {
		return []clojureFormContext{{Form: form, Platforms: append([]string(nil), inheritedPlatforms...), Conditional: inheritedConditional}}, nil
	}
	branches, issues := clojureReaderBranches(form)
	selected := selectClojureReaderBranches(branches, platform)
	contexts := make([]clojureFormContext, 0)
	for _, branch := range selected {
		platforms := append([]string(nil), inheritedPlatforms...)
		if branch.Selector != "default" {
			platforms = appendUniqueString(platforms, branch.Selector)
		} else {
			platforms = appendUniqueString(platforms, "default")
		}
		if form.ReaderSplice && (branch.Form.Kind == formVector || branch.Form.Kind == formList || branch.Form.Kind == formSet) {
			for _, item := range branch.Form.Items {
				children, childIssues := expandClojureForm(item, platform, platforms, true)
				contexts = append(contexts, children...)
				issues = append(issues, childIssues...)
			}
			continue
		}
		children, childIssues := expandClojureForm(branch.Form, platform, platforms, true)
		contexts = append(contexts, children...)
		issues = append(issues, childIssues...)
	}
	return contexts, issues
}

func clojureReaderBranches(form *cljForm) ([]clojureReaderBranch, []clojureReaderIssue) {
	issues := make([]clojureReaderIssue, 0)
	if len(form.Items)%2 != 0 {
		issues = append(issues, clojureReaderIssue{Location: clonePosition(form.Start), Message: "Clojure reader conditional has an unmatched selector or branch; no guessed branch was added."})
		return nil, issues
	}
	branches := make([]clojureReaderBranch, 0, len(form.Items)/2)
	for index := 0; index+1 < len(form.Items); index += 2 {
		selector := formKeyword(form.Items[index])
		if selector == "" {
			issues = append(issues, clojureReaderIssue{Location: clonePosition(form.Items[index].Start), Message: "Clojure reader conditional selector must be a keyword; the malformed branch was ignored."})
			continue
		}
		selector = strings.TrimPrefix(selector, ":")
		switch selector {
		case "clj", "cljs", "default":
			branches = append(branches, clojureReaderBranch{Selector: selector, Form: form.Items[index+1]})
		default:
			issues = append(issues, clojureReaderIssue{Location: clonePosition(form.Items[index].Start), Message: fmt.Sprintf("Clojure reader conditional selector %q is unsupported; the branch was ignored.", selector)})
		}
	}
	return branches, issues
}

func selectClojureReaderBranches(branches []clojureReaderBranch, platform string) []clojureReaderBranch {
	selected := make([]clojureReaderBranch, 0, len(branches))
	hasClj, hasCLJS := false, false
	for _, branch := range branches {
		switch branch.Selector {
		case "clj":
			hasClj = true
		case "cljs":
			hasCLJS = true
		}
	}
	switch platform {
	case "clj":
		for _, branch := range branches {
			if branch.Selector == "clj" || (branch.Selector == "default" && !hasClj) {
				selected = append(selected, branch)
			}
		}
	case "cljs":
		for _, branch := range branches {
			if branch.Selector == "cljs" || (branch.Selector == "default" && !hasCLJS) {
				selected = append(selected, branch)
			}
		}
	default:
		for _, branch := range branches {
			if branch.Selector == "clj" || branch.Selector == "cljs" || branch.Selector == "default" {
				selected = append(selected, branch)
			}
		}
	}
	return selected
}

func isClojureDependencyClause(clause string) bool {
	switch clause {
	case ":require", ":use", ":require-macros", ":use-macros":
		return true
	default:
		return false
	}
}

func parseClojureDependencyClause(file clojureFileObservation, context clojureFormContext, clause string) ([]clojureDependencyObservation, []analysis.SourceReference, []analysis.Diagnostic) {
	kind := "require"
	if clause == ":use" {
		kind = "use"
	}
	if clause == ":require-macros" || clause == ":use-macros" {
		kind = "macro"
	}
	dependencies := make([]clojureDependencyObservation, 0)
	sources := make([]analysis.SourceReference, 0)
	diagnostics := make([]analysis.Diagnostic, 0)
	for _, spec := range context.Form.Items[1:] {
		target, modifiers, ok := clojureDependencySpec(spec)
		if !ok || !validClojureNamespace(target) {
			diagnostics = append(diagnostics, analysis.Diagnostic{
				Code:        "clojure_dependency_syntax",
				Severity:    "warning",
				Message:     "Clojure namespace dependency could not be interpreted statically and was ignored.",
				Path:        file.Path,
				Location:    clonePosition(spec.Start),
				Recoverable: true,
			})
			continue
		}
		alias, referredSymbols, includeMacros, referMacros := parseClojureDependencyModifiers(modifiers)
		dependencyKind := kind
		if includeMacros || referMacros {
			dependencyKind = "macro"
		}
		sourceID := clojureStableID("dependency", file.Path, target, dependencyKind, positionKey(spec.Start))
		source := analysis.SourceReference{
			ID:     sourceID,
			Path:   file.Path,
			Start:  clonePosition(spec.Start),
			End:    clonePosition(spec.End),
			Symbol: target,
			Kind:   "dependency",
		}
		sources = append(sources, source)
		dependencies = append(dependencies, clojureDependencyObservation{
			FromModuleID:    file.ModuleID,
			Source:          source,
			TargetNamespace: target,
			Kind:            dependencyKind,
			Alias:           alias,
			ReferredSymbols: referredSymbols,
			Platforms:       append([]string(nil), context.Platforms...),
			Conditional:     context.Conditional,
		})
	}
	return dependencies, sources, diagnostics
}

func clojureDependencySpec(form *cljForm) (string, []*cljForm, bool) {
	if symbol := formSymbol(form); symbol != "" {
		return symbol, nil, true
	}
	if form == nil || (form.Kind != formVector && form.Kind != formList) || len(form.Items) == 0 {
		return "", nil, false
	}
	target := formSymbol(form.Items[0])
	if target == "" {
		return "", nil, false
	}
	return target, form.Items[1:], true
}

func parseClojureDependencyModifiers(modifiers []*cljForm) (string, []string, bool, bool) {
	alias := ""
	referred := make([]string, 0)
	includeMacros := false
	referMacros := false
	for index := 0; index < len(modifiers); index++ {
		key := formKeyword(modifiers[index])
		if key == "" || index+1 >= len(modifiers) {
			continue
		}
		value := modifiers[index+1]
		switch key {
		case ":as":
			alias = formSymbol(value)
		case ":refer", ":only":
			referred = appendUniqueStrings(referred, clojureSymbolsFromCollection(value)...)
		case ":refer-macros":
			referred = appendUniqueStrings(referred, clojureSymbolsFromCollection(value)...)
			referMacros = true
		case ":include-macros":
			includeMacros = value != nil && value.Kind == formAtom && value.Atom == "true"
		}
		index++
	}
	sort.Strings(referred)
	return alias, referred, includeMacros, referMacros
}

func clojureSymbolsFromCollection(form *cljForm) []string {
	if form == nil {
		return nil
	}
	if symbol := formSymbol(form); symbol != "" {
		return []string{symbol}
	}
	if form.Kind != formVector && form.Kind != formList && form.Kind != formSet {
		return nil
	}
	result := make([]string, 0, len(form.Items))
	for _, item := range form.Items {
		if symbol := formSymbol(item); symbol != "" {
			result = append(result, symbol)
		}
	}
	return result
}

func positionKey(position *analysis.Position) string {
	if position == nil {
		return "0:0"
	}
	return fmt.Sprintf("%d:%d", position.Line, position.Column)
}

func appendUniqueString(values []string, value string) []string {
	for _, current := range values {
		if current == value {
			return values
		}
	}
	return append(values, value)
}

func appendUniqueStrings(values []string, additions ...string) []string {
	if values == nil {
		values = []string{}
	}
	for _, addition := range additions {
		if addition != "" {
			values = appendUniqueString(values, addition)
		}
	}
	return values
}

func buildClojureDependencyObservations(modules []analysis.ModuleObservation, dependencies []clojureDependencyObservation) ([]analysis.RelationshipObservation, []analysis.Reference, []analysis.Diagnostic) {
	moduleIDs := make(map[string]struct{}, len(modules))
	topLevels := make(map[string]struct{})
	for _, module := range modules {
		moduleIDs[module.ID] = struct{}{}
		if len(module.Hierarchy) > 0 {
			topLevels[module.Hierarchy[0]] = struct{}{}
		}
	}
	relationships := make(map[string]analysis.RelationshipObservation)
	references := make(map[string]analysis.Reference)
	diagnostics := make([]analysis.Diagnostic, 0)
	for _, dependency := range dependencies {
		scope, resolution, moduleID := classifyClojureTarget(dependency.TargetNamespace, moduleIDs, topLevels)
		metadata := clojureDependencyMetadata(dependency, scope, resolution)
		relationshipID := clojureStableID("relationship", dependency.FromModuleID, dependency.TargetNamespace, dependency.Kind)
		relationship := relationships[relationshipID]
		if relationship.ID == "" {
			relationship = analysis.RelationshipObservation{
				ID:                 relationshipID,
				Type:               "depends_on",
				FromModuleID:       dependency.FromModuleID,
				SourceReferenceIDs: []string{},
				Metadata:           map[string]any{},
			}
		}
		relationship.SourceReferenceIDs = appendUniqueStrings(relationship.SourceReferenceIDs, dependency.Source.ID)
		mergeClojureMetadata(relationship.Metadata, metadata)
		confidence := clojureDependencyConfidence(scope, dependency.Conditional)
		relationship.Confidence = mergeClojureConfidence(relationship.Confidence, confidence)
		if moduleID != "" {
			relationship.ToModuleID = moduleID
			relationship.ToReferenceID = ""
		} else {
			referenceID := clojureStableID("reference", scope, dependency.TargetNamespace)
			relationship.ToReferenceID = referenceID
			relationship.ToModuleID = ""
			reference := references[referenceID]
			if reference.ID == "" {
				reference = analysis.Reference{ID: referenceID, Name: dependency.TargetNamespace, Scope: scope, Language: "clojure", Metadata: map[string]any{}}
			}
			mergeClojureMetadata(reference.Metadata, metadata)
			references[referenceID] = reference
			if scope == "unresolved" {
				diagnostics = append(diagnostics, analysis.Diagnostic{
					Code:        "clojure_unresolved_dependency",
					Severity:    "warning",
					Message:     fmt.Sprintf("Clojure namespace %q could not be resolved to a proven source-root namespace; it was retained as an unresolved reference.", dependency.TargetNamespace),
					Subject:     dependency.TargetNamespace,
					Path:        dependency.Source.Path,
					Location:    clonePosition(dependency.Source.Start),
					Recoverable: true,
					Metadata: map[string]any{
						"source_reference_id": dependency.Source.ID,
						"reference_id":        referenceID,
					},
				})
			}
		}
		relationships[relationshipID] = relationship
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

func classifyClojureTarget(target string, moduleIDs, topLevels map[string]struct{}) (string, string, string) {
	moduleID := clojureModuleID(target)
	if _, ok := moduleIDs[moduleID]; ok {
		return "project_local", "resolved", moduleID
	}
	if isClojureStandardNamespace(target) {
		return "standard_library", "standard_library", ""
	}
	root := target
	if dot := strings.IndexByte(root, '.'); dot >= 0 {
		root = root[:dot]
	}
	if _, ok := topLevels[root]; ok {
		return "unresolved", "unresolved", ""
	}
	return "external", "external", ""
}

func isClojureStandardNamespace(target string) bool {
	return target == "js" || strings.HasPrefix(target, "clojure.") || strings.HasPrefix(target, "cljs.") || strings.HasPrefix(target, "java.") || strings.HasPrefix(target, "goog.")
}

func clojureDependencyMetadata(dependency clojureDependencyObservation, scope, resolution string) map[string]any {
	referredSymbols := append([]string{}, dependency.ReferredSymbols...)
	platforms := append([]string{}, dependency.Platforms...)
	metadata := map[string]any{
		"target_namespace": dependency.TargetNamespace,
		"target_scope":     scope,
		"dependency_kinds": []string{dependency.Kind},
		"resolution_kinds": []string{resolution},
		"refered_symbols":  referredSymbols,
		"referred_symbols": append([]string{}, referredSymbols...),
		"aliases":          []string{},
		"platforms":        platforms,
		"conditional":      dependency.Conditional,
		"resolution":       resolution,
	}
	if dependency.Alias != "" {
		metadata["aliases"] = []string{dependency.Alias}
	}
	return metadata
}

func clojureDependencyConfidence(scope string, conditional bool) *analysis.Confidence {
	if conditional {
		return &analysis.Confidence{Basis: "inferred", Score: 0.5}
	}
	switch scope {
	case "project_local":
		return &analysis.Confidence{Basis: "resolved", Score: 1}
	case "standard_library":
		return &analysis.Confidence{Basis: "standard_library", Score: 0.8}
	case "unresolved":
		return &analysis.Confidence{Basis: "unresolved", Score: 0.2}
	default:
		return &analysis.Confidence{Basis: "external", Score: 0.4}
	}
}

func mergeClojureConfidence(left, right *analysis.Confidence) *analysis.Confidence {
	if left == nil {
		return right
	}
	if right == nil {
		return left
	}
	if right.Score < left.Score {
		return right
	}
	return left
}

func mergeClojureMetadata(target, values map[string]any) {
	if target == nil {
		return
	}
	for key, value := range values {
		switch typed := value.(type) {
		case []string:
			existing, _ := target[key].([]string)
			existing = appendUniqueStrings(existing, typed...)
			sort.Strings(existing)
			target[key] = existing
		case bool:
			if current, ok := target[key].(bool); !ok || typed || !current {
				target[key] = current || typed
			}
		case string:
			if _, ok := target[key].(string); !ok {
				target[key] = typed
			} else if key == "resolution" {
				if current, _ := target[key].(string); current != typed {
					target[key] = "mixed"
				}
			}
		default:
			if _, ok := target[key]; !ok {
				target[key] = value
			}
		}
	}
}
