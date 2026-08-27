package rustanalyzer

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

// BuildResult converts the Rust discovery aggregate to the shared analyzer
// contract. Rust-specific semantics remain in metadata; the host/model see
// only modules, hierarchy, static dependencies, references, evidence, and
// diagnostics.
func BuildResult(project Project, discovery discoveryResult, request analysis.AnalyzeRequest, manifest analysis.Manifest) analysis.AnalysisResult {
	modules := make([]analysis.ModuleObservation, 0, len(discovery.Modules))
	moduleKeys := make([]string, 0, len(discovery.Modules))
	for key := range discovery.Modules {
		moduleKeys = append(moduleKeys, key)
	}
	sort.Strings(moduleKeys)
	for _, key := range moduleKeys {
		module := discovery.Modules[key]
		modules = append(modules, rustModuleObservation(project, module))
	}

	references := make(map[string]analysis.Reference)
	relationships := make(map[string]analysis.RelationshipObservation)
	sourceReferences := make(map[string]analysis.SourceReference, len(discovery.SourceReferences)+len(project.Dependencies))
	for id, source := range discovery.SourceReferences {
		sourceReferences[id] = rebaseRustSource(project, source)
	}
	diagnostics := make([]analysis.Diagnostic, 0, len(discovery.Diagnostics)+len(discovery.Uses))
	for _, diagnostic := range discovery.Diagnostics {
		diagnostics = append(diagnostics, rebaseRustDiagnostic(project, diagnostic))
	}

	rootModule := discovery.Modules[""]
	if rootModule == nil {
		rootModule = &rustModule{ID: rustModuleID(project.PackageName, nil)}
		modules = append(modules, rustModuleObservation(project, rootModule))
	}
	for _, dependency := range project.Dependencies {
		if dependency.Source.ID != "" {
			source := rebaseRustSource(project, dependency.Source)
			sourceReferences[source.ID] = source
		}
		addRustDependencyObservation(project, rootModule, dependency, references, relationships)
	}

	for _, observation := range discovery.Uses {
		if observation.Source.ID != "" {
			source := rebaseRustSource(project, observation.Source)
			sourceReferences[source.ID] = source
		}
		targetModule, scope, reference := resolveRustUse(project, discovery.Modules, observation)
		if targetModule == nil && scope == "unresolved" {
			if reference.ID == "" {
				reference = rustReference(project, observation.Path, "unresolved", observation.Kind, nil)
			}
			diagnostics = append(diagnostics, analysis.Diagnostic{
				Code:        "rust_unresolved_use",
				Severity:    "warning",
				Message:     fmt.Sprintf("Rust use path %q could not be resolved statically; it was retained as an unresolved reference.", observation.Path),
				Path:        observation.Source.Path,
				Location:    observation.Source.Start,
				Recoverable: true,
				Metadata:    map[string]any{"use_path": observation.Path, "kind": observation.Kind},
			})
		}
		if reference.ID != "" {
			mergeRustReference(references, reference)
		}
		addRustUseRelationship(project, observation, targetModule, scope, reference, relationships)
	}

	result := analysis.AnalysisResult{
		Status:   analysis.StatusComplete,
		Analyzer: analyzerInfo(manifest),
		Project: analysis.ProjectInfo{
			RootLabel:     filepath.Base(project.Root),
			Boundary:      project.Boundary,
			ModulePath:    project.PackageName,
			ModuleRoot:    project.RelativeCrateRoot,
			WorkspacePath: project.RelativeWorkspacePath,
		},
		OptionsFingerprint: request.Options.Fingerprint,
		Modules:            modules,
		Relationships:      rustRelationshipValues(relationships),
		References:         rustReferenceValues(references),
		SourceReferences:   rustSourceValues(sourceReferences),
		Diagnostics:        diagnostics,
	}
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Recoverable || diagnostic.Severity == "error" {
			result.Status = analysis.StatusPartial
			break
		}
	}
	sortRustDiagnostics(result.Diagnostics)
	result.Summary = analysis.ComputeSummary(result)
	return result
}

func analyzerInfo(manifest analysis.Manifest) analysis.AnalyzerInfo {
	return analysis.AnalyzerInfo{ID: manifest.ID, Version: manifest.Version, Language: manifest.Language, APIVersion: manifest.APIVersion}
}

func rustModuleObservation(project Project, module *rustModule) analysis.ModuleObservation {
	sourceIDs := sortedRustSet(module.SourceReferenceIDs)
	tags := sortedRustSet(module.Tags)
	paths := sortedRustSet(module.Paths)
	declared := sortedRustSet(module.DeclaredModules)
	conditions := sortedRustSet(module.CfgConditions)
	targetKinds := sortedRustSet(module.TargetKinds)
	macros := sortedRustSet(module.MacroAttributes)
	metadata := map[string]any{
		"crate_name":        project.PackageName,
		"module_path":       strings.Join(module.Path, "::"),
		"relative_paths":    paths,
		"file_count":        len(paths),
		"manifest_path":     project.RelativeManifestPath,
		"declared_modules":  declared,
		"target_kinds":      targetKinds,
		"features":          append([]string{}, project.Features...),
		"declared_features": append([]string{}, project.DeclaredFeatures...),
	}
	if project.Edition != "" {
		metadata["edition"] = project.Edition
	}
	if project.Target != "" {
		metadata["target"] = project.Target
	}
	if project.RelativeWorkspacePath != "" {
		metadata["workspace_path"] = project.RelativeWorkspacePath
	}
	if len(conditions) > 0 {
		metadata["cfg_conditions"] = conditions
		metadata["cfg_uncertain"] = !rustCfgKnown(conditions)
	}
	if len(macros) > 0 {
		metadata["macro_attributes"] = macros
		metadata["generated_uncertainty"] = true
	}
	if module.Generated {
		metadata["generated"] = true
	}
	displayName := module.DisplayName(project.PackageName)
	return analysis.ModuleObservation{
		ID:                 module.ID,
		Language:           "rust",
		Kind:               module.Kind,
		Name:               module.Name,
		DisplayName:        displayName,
		Hierarchy:          append([]string{}, module.Path...),
		SourceReferenceIDs: sourceIDs,
		Tags:               tags,
		Metadata:           metadata,
	}
}

func addRustDependencyObservation(project Project, from *rustModule, dependency CargoDependency, references map[string]analysis.Reference, relationships map[string]analysis.RelationshipObservation) {
	if from == nil || dependency.Name == "" {
		return
	}
	scope := rustDependencyScope(dependency.Name)
	metadata := map[string]any{
		"kind":            "dependency",
		"dependency_kind": dependency.Kind,
		"target_scope":    scope,
		"declared_name":   dependency.Name,
	}
	if dependency.Version != "" {
		metadata["version"] = dependency.Version
	}
	if dependency.Path != "" {
		metadata["path"] = dependency.Path
	}
	if dependency.Registry != "" {
		metadata["registry"] = dependency.Registry
	}
	if dependency.Optional {
		metadata["optional"] = true
	}
	if dependency.Workspace {
		metadata["workspace"] = true
	}
	if dependency.DefaultFeatures != nil {
		metadata["default_features"] = *dependency.DefaultFeatures
	}
	if len(dependency.Features) > 0 {
		metadata["features"] = append([]string(nil), dependency.Features...)
	}
	if dependency.Target != "" {
		metadata["cfg_conditions"] = []string{dependency.Target}
		metadata["cfg_uncertain"] = true
	}
	reference := rustReference(project, dependency.Name, scope, "dependency", metadata)
	mergeRustReference(references, reference)
	relationID := stableID("relationship", from.ID, "dependency", reference.ID, dependency.Name)
	relation := relationships[relationID]
	if relation.ID == "" {
		relation = analysis.RelationshipObservation{ID: relationID, Type: "depends_on", FromModuleID: from.ID, ToReferenceID: reference.ID, SourceReferenceIDs: []string{}, Confidence: rustConfidence(scope, dependency.Target != ""), Metadata: map[string]any{}}
	}
	mergeRustMetadata(relation.Metadata, metadata)
	if dependency.Source.ID != "" {
		relation.SourceReferenceIDs = appendUniqueString(relation.SourceReferenceIDs, dependency.Source.ID)
	}
	sort.Strings(relation.SourceReferenceIDs)
	relationships[relationID] = relation
}

func addRustUseRelationship(project Project, observation rustUseObservation, target *rustModule, scope string, reference analysis.Reference, relationships map[string]analysis.RelationshipObservation) {
	targetKey := reference.ID
	if target != nil {
		targetKey = target.ID
	}
	if targetKey == "" {
		return
	}
	relationID := stableID("relationship", observation.FromModuleID, observation.Kind, targetKey)
	relation := relationships[relationID]
	if relation.ID == "" {
		relation = analysis.RelationshipObservation{ID: relationID, Type: "depends_on", FromModuleID: observation.FromModuleID, ToModuleID: "", ToReferenceID: "", SourceReferenceIDs: []string{}, Confidence: rustConfidence(scope, len(observation.CfgConditions) > 0), Metadata: map[string]any{}}
		if target != nil {
			relation.ToModuleID = target.ID
		} else {
			relation.ToReferenceID = reference.ID
		}
	}
	metadata := map[string]any{
		"kind":         observation.Kind,
		"target_scope": scope,
		"import_paths": []string{observation.Path},
		"import_names": []string{observation.ImportedName},
	}
	if observation.Alias != "" {
		metadata["aliases"] = []string{observation.Alias}
	}
	if observation.Kind == "pub_use" {
		metadata["reexport"] = true
	}
	if len(observation.CfgConditions) > 0 {
		metadata["cfg_conditions"] = append([]string(nil), observation.CfgConditions...)
		metadata["cfg_uncertain"] = !rustCfgKnown(observation.CfgConditions)
	}
	mergeRustMetadata(relation.Metadata, metadata)
	if observation.Source.ID != "" {
		relation.SourceReferenceIDs = appendUniqueString(relation.SourceReferenceIDs, observation.Source.ID)
	}
	sort.Strings(relation.SourceReferenceIDs)
	relationships[relationID] = relation
	_ = project
}

func resolveRustUse(project Project, modules map[string]*rustModule, observation rustUseObservation) (*rustModule, string, analysis.Reference) {
	parts := rustUseParts(observation.Path)
	if len(parts) == 0 {
		return nil, "unresolved", analysis.Reference{}
	}
	currentPath := rustModulePathForID(modules, observation.FromModuleID)
	type moduleCandidate struct {
		path          []string
		minimumLength int
	}
	candidates := make([]moduleCandidate, 0, 4)
	relativePath := parts[0] == "crate" || parts[0] == "self" || parts[0] == "super"
	switch parts[0] {
	case "crate":
		path := parts[1:]
		minimumLength := 1
		if len(path) == 0 {
			minimumLength = 0
		}
		candidates = append(candidates, moduleCandidate{path: path, minimumLength: minimumLength})
	case "self":
		path := append(append([]string(nil), currentPath...), parts[1:]...)
		minimumLength := len(currentPath)
		if len(parts) > 1 {
			minimumLength++
		}
		candidates = append(candidates, moduleCandidate{path: path, minimumLength: minimumLength})
	case "super":
		base := append([]string(nil), currentPath...)
		for len(parts) > 0 && parts[0] == "super" {
			if len(base) > 0 {
				base = base[:len(base)-1]
			}
			parts = parts[1:]
		}
		minimumLength := len(base)
		if len(parts) > 0 {
			minimumLength++
		}
		candidates = append(candidates, moduleCandidate{path: append(base, parts...), minimumLength: minimumLength})
	default:
		if normalizeCargoName(parts[0]) == normalizeCargoName(project.PackageName) {
			path := parts[1:]
			minimumLength := 1
			if len(path) == 0 {
				minimumLength = 0
			}
			candidates = append(candidates, moduleCandidate{path: path, minimumLength: minimumLength})
		} else {
			candidates = append(candidates, moduleCandidate{path: parts, minimumLength: 1})
		}
		for index := len(currentPath); index >= 0; index-- {
			candidates = append(candidates, moduleCandidate{path: append(append([]string(nil), currentPath[:index]...), parts...), minimumLength: 1})
		}
	}
	if len(parts) == 0 {
		return nil, "unresolved", rustReference(project, observation.Path, "unresolved", observation.Kind, map[string]any{"kind": observation.Kind, "target_scope": "unresolved"})
	}
	for _, candidate := range candidates {
		for length := len(candidate.path); length >= candidate.minimumLength; length-- {
			key := strings.Join(candidate.path[:length], "::")
			if module := modules[key]; module != nil {
				if module.ID == observation.FromModuleID && length == 0 {
					continue
				}
				return module, "local", analysis.Reference{}
			}
		}
	}
	name := parts[0]
	if relativePath || name == "crate" || name == "self" || name == "super" {
		return nil, "unresolved", rustReference(project, observation.Path, "unresolved", observation.Kind, map[string]any{"kind": observation.Kind, "target_scope": "unresolved"})
	}
	scope := rustDependencyScope(name)
	if scope == "standard_library" || (scope == "external" && rustProjectDeclaresDependency(project, name)) {
		return nil, scope, rustReference(project, name, scope, observation.Kind, map[string]any{"kind": observation.Kind, "target_scope": scope})
	}
	return nil, "unresolved", rustReference(project, observation.Path, "unresolved", observation.Kind, map[string]any{"kind": observation.Kind, "target_scope": "unresolved"})
}

func rustProjectDeclaresDependency(project Project, name string) bool {
	for _, dependency := range project.Dependencies {
		if normalizeCargoName(dependency.Name) == normalizeCargoName(name) {
			return true
		}
	}
	return false
}

func rustUseParts(value string) []string {
	parts := strings.Split(strings.TrimSpace(value), "::")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = normalizeRustIdent(strings.TrimSpace(part))
		if part == "" || part == "*" {
			continue
		}
		result = append(result, part)
	}
	return result
}

func rustModulePathForID(modules map[string]*rustModule, id string) []string {
	for _, module := range modules {
		if module.ID == id {
			return append([]string(nil), module.Path...)
		}
	}
	return nil
}

func rustReference(project Project, name, scope, kind string, metadata map[string]any) analysis.Reference {
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["kind"] = kind
	metadata["target_scope"] = scope
	return analysis.Reference{ID: stableID("reference", scope, name), Name: name, Scope: scope, Language: "rust", Metadata: metadata}
}

func mergeRustReference(references map[string]analysis.Reference, value analysis.Reference) {
	if value.ID == "" {
		return
	}
	existing := references[value.ID]
	if existing.ID == "" {
		references[value.ID] = value
		return
	}
	mergeRustMetadata(existing.Metadata, value.Metadata)
	references[value.ID] = existing
}

func mergeRustMetadata(target, values map[string]any) {
	if target == nil {
		return
	}
	for key, value := range values {
		switch typed := value.(type) {
		case []string:
			existing, _ := target[key].([]string)
			for _, item := range typed {
				if item != "" {
					existing = appendUniqueString(existing, item)
				}
			}
			sort.Strings(existing)
			target[key] = existing
		case bool:
			if current, ok := target[key].(bool); !ok || typed {
				target[key] = typed
			} else if !current {
				target[key] = false
			}
		default:
			if _, exists := target[key]; !exists {
				target[key] = value
			}
		}
	}
}

func rustConfidence(scope string, uncertain bool) *analysis.Confidence {
	if uncertain {
		return &analysis.Confidence{Basis: "inferred", Score: 0.5}
	}
	switch scope {
	case "local", "standard_library":
		return &analysis.Confidence{Basis: "resolved", Score: 1}
	case "unresolved":
		return &analysis.Confidence{Basis: "unresolved", Score: 0.2}
	default:
		return &analysis.Confidence{Basis: "inferred", Score: 0.8}
	}
}

func rustDependencyScope(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "std", "core", "alloc", "proc_macro", "test":
		return "standard_library"
	default:
		return "external"
	}
}

func rebaseRustSource(project Project, source analysis.SourceReference) analysis.SourceReference {
	if source.Path != "" && filepath.IsAbs(source.Path) {
		source.Path = projectRelative(project.Root, source.Path)
	}
	if source.ID == "" {
		source.ID = stableID("source", source.Kind, source.Path, source.Symbol, positionKey(source.Start), positionKey(source.End))
	}
	return source
}

func rebaseRustDiagnostic(project Project, diagnostic analysis.Diagnostic) analysis.Diagnostic {
	if diagnostic.Path != "" && filepath.IsAbs(diagnostic.Path) {
		diagnostic.Path = projectRelative(project.Root, diagnostic.Path)
	}
	return diagnostic
}

func sortedRustSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func rustRelationshipValues(values map[string]analysis.RelationshipObservation) []analysis.RelationshipObservation {
	result := make([]analysis.RelationshipObservation, 0, len(values))
	for _, value := range values {
		sort.Strings(value.SourceReferenceIDs)
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func rustReferenceValues(values map[string]analysis.Reference) []analysis.Reference {
	result := make([]analysis.Reference, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func rustSourceValues(values map[string]analysis.SourceReference) []analysis.SourceReference {
	result := make([]analysis.SourceReference, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return valueSourceSortKey(result[i]) < valueSourceSortKey(result[j]) })
	return result
}

func valueSourceSortKey(value analysis.SourceReference) string {
	return value.ID + "\x00" + value.Path
}

func sortRustDiagnostics(values []analysis.Diagnostic) {
	sort.Slice(values, func(i, j int) bool {
		left, _ := json.Marshal(values[i])
		right, _ := json.Marshal(values[j])
		return string(left) < string(right)
	})
}
