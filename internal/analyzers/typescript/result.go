package tsanalyzer

import (
	"context"
	"path/filepath"
	"sort"

	"github.com/buffo/arch-view/internal/analysis"
	tssyntax "github.com/buffo/arch-view/internal/analysis/syntax/typescript"
)

// BuildResult converts project discovery and syntax dependency observations
// into the language-neutral analyzer contract.
func BuildResult(project Project, discovery discoveryResult, request analysis.AnalyzeRequest, manifest analysis.Manifest) analysis.AnalysisResult {
	return buildResult(context.Background(), project, discovery, request, manifest, treeSitterTSImportExtractor{provider: tssyntax.NewProvider()})
}

func buildResult(ctx context.Context, project Project, discovery discoveryResult, request analysis.AnalyzeRequest, manifest analysis.Manifest, extractor tsImportExtractor) analysis.AnalysisResult {
	if ctx == nil {
		ctx = context.Background()
	}
	if extractor == nil {
		extractor = treeSitterTSImportExtractor{provider: tssyntax.NewProvider()}
	}
	diagnostics := append([]analysis.Diagnostic{}, discovery.Diagnostics...)
	index := newTSModuleIndex(discovery)
	relationships := make(map[string]analysis.RelationshipObservation)
	references := make(map[string]analysis.Reference)
	sources := make(map[string]analysis.SourceReference, len(discovery.SourceReferences))
	for _, source := range discovery.SourceReferences {
		sources[source.ID] = source
	}

	paths := make([]string, 0, len(discovery.Files))
	for relative := range discovery.Files {
		paths = append(paths, relative)
	}
	sort.Strings(paths)
	for _, relative := range paths {
		file := discovery.Files[relative]
		extraction := extractor.Extract(ctx, relative, file.Content, file.ModuleID)
		diagnostics = append(diagnostics, extraction.Diagnostics...)
		if extraction.BackendError != nil {
			diagnostics = append(diagnostics, tsImportBackendDiagnostic(relative, extraction.BackendError))
		}
		for _, observation := range extraction.Observations {
			sources[observation.Source.ID] = observation.Source
			resolution := resolveTSImport(project, discovery, observation, index)
			if resolution.Diagnostic != nil {
				diagnostics = append(diagnostics, *resolution.Diagnostic)
			}
			addTSRelationship(observation, resolution, relationships, references)
		}
	}
	if len(discovery.Modules) == 0 {
		diagnostics = append(diagnostics, analysis.Diagnostic{
			Code:        "typescript_no_modules",
			Severity:    "warning",
			Message:     "No eligible TypeScript modules were found in the selected project scope.",
			Recoverable: true,
		})
	}

	result := analysis.AnalysisResult{
		RunID:              "",
		Status:             analysis.StatusComplete,
		Analyzer:           analyzerInfo(manifest),
		Project:            typescriptProjectInfo(project),
		OptionsFingerprint: request.Options.Fingerprint,
		Modules:            append([]analysis.ModuleObservation{}, discovery.Modules...),
		Relationships:      sortedTSRelationships(relationships),
		References:         sortedTSReferences(references),
		SourceReferences:   sortedTSSources(sources),
		Diagnostics:        diagnostics,
	}
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Recoverable {
			result.Status = analysis.StatusPartial
			break
		}
	}
	sortTSDiagnostics(result.Diagnostics)
	result.Summary = analysis.ComputeSummary(result)
	return result
}

func analyzerInfo(manifest analysis.Manifest) analysis.AnalyzerInfo {
	return analysis.AnalyzerInfo{ID: manifest.ID, Version: manifest.Version, Language: manifest.Language, APIVersion: manifest.APIVersion}
}

func typescriptProjectInfo(project Project) analysis.ProjectInfo {
	moduleRoot := "."
	if len(project.SourceRoots) > 0 {
		moduleRoot = project.SourceRoots[0]
	}
	info := analysis.ProjectInfo{
		RootLabel:  filepath.Base(project.Root),
		Boundary:   project.Boundary,
		ModuleRoot: moduleRoot,
		ModulePath: project.PackageName,
	}
	if project.PackageJSONPath != "" {
		info.WorkspacePath = project.PackageJSONPath
	}
	return info
}

func addTSRelationship(observation tsImportObservation, resolution tsResolution, relationships map[string]analysis.RelationshipObservation, references map[string]analysis.Reference) {
	targetScope := resolution.Scope
	if targetScope == "" {
		targetScope = "unresolved"
	}
	metadata := tsObservationMetadata(observation, resolution)
	var relationshipID string
	var targetModuleID string
	var targetReferenceID string
	if resolution.Scope == "local" && resolution.ModuleID != "" {
		targetModuleID = resolution.ModuleID
		relationshipID = stableTSID("relationship", observation.FromModuleID, "module", targetModuleID)
	} else {
		name := observation.Specifier
		if observation.Computed && observation.Expression != "" {
			name = observation.Expression
		}
		targetReferenceID = stableTSID("reference", targetScope, name)
		relationshipID = stableTSID("relationship", observation.FromModuleID, "reference", targetReferenceID)
		reference := references[targetReferenceID]
		if reference.ID == "" {
			reference = analysis.Reference{ID: targetReferenceID, Name: name, Scope: targetScope, Language: "typescript", Metadata: map[string]any{}}
		}
		mergeTSMetadata(reference.Metadata, metadata)
		references[targetReferenceID] = reference
	}

	relationship := relationships[relationshipID]
	if relationship.ID == "" {
		relationship = analysis.RelationshipObservation{
			ID:                 relationshipID,
			Type:               "depends_on",
			FromModuleID:       observation.FromModuleID,
			ToModuleID:         targetModuleID,
			ToReferenceID:      targetReferenceID,
			SourceReferenceIDs: []string{},
			Confidence:         resolution.Confidence,
			Metadata:           map[string]any{},
		}
	}
	relationship.SourceReferenceIDs = appendUniqueString(relationship.SourceReferenceIDs, observation.Source.ID)
	mergeTSMetadata(relationship.Metadata, metadata)
	relationship.Confidence = mergeTSConfidence(relationship.Confidence, resolution.Confidence)
	relationships[relationshipID] = relationship
}

func tsObservationMetadata(observation tsImportObservation, resolution tsResolution) map[string]any {
	metadata := cloneTSMetadata(resolution.Metadata)
	metadata["target_scope"] = resolution.Scope
	metadata["kind"] = observation.Kind
	metadata["import_paths"] = []string{observation.Specifier}
	metadata["import_kinds"] = []string{observation.Kind}
	metadata["resolution_kinds"] = []string{resolution.ResolutionKind}
	if observation.TypeOnly {
		metadata["type_only"] = true
	}
	if observation.Reexport {
		metadata["reexport"] = true
	}
	if observation.Dynamic {
		metadata["dynamic"] = true
	}
	if observation.Computed {
		metadata["computed"] = true
	}
	if len(observation.ImportNames) > 0 {
		metadata["import_names"] = append([]string{}, observation.ImportNames...)
	}
	if len(observation.Aliases) > 0 {
		metadata["aliases"] = append([]string{}, observation.Aliases...)
	}
	if resolution.Alias != "" {
		metadata["aliases"] = appendUniqueString(asStringSlice(metadata["aliases"]), resolution.Alias)
		metadata["alias"] = resolution.Alias
	}
	if resolution.Runtime != "" {
		metadata["runtime"] = []string{resolution.Runtime}
		metadata["runtime_context"] = resolution.Runtime
	}
	return metadata
}

func mergeTSMetadata(target, values map[string]any) {
	if target == nil {
		return
	}
	for key, value := range values {
		switch typed := value.(type) {
		case []string:
			existing := asStringSlice(target[key])
			for _, item := range typed {
				existing = appendUniqueString(existing, item)
			}
			sort.Strings(existing)
			target[key] = existing
		case bool:
			if current, ok := target[key].(bool); !ok || typed {
				target[key] = typed || current
			}
		case string:
			if existing, exists := target[key].(string); !exists {
				target[key] = typed
			} else if (key == "kind" || key == "runtime_context") && existing != typed {
				target[key] = "mixed"
			}
		default:
			if _, exists := target[key]; !exists {
				target[key] = value
			}
		}
	}
}

func asStringSlice(value any) []string {
	if values, ok := value.([]string); ok {
		return append([]string{}, values...)
	}
	return []string{}
}

func mergeTSConfidence(left, right *analysis.Confidence) *analysis.Confidence {
	if left == nil {
		return right
	}
	if right == nil || right.Score >= left.Score {
		return left
	}
	return right
}

func sortedTSRelationships(values map[string]analysis.RelationshipObservation) []analysis.RelationshipObservation {
	result := make([]analysis.RelationshipObservation, 0, len(values))
	for _, value := range values {
		value.SourceReferenceIDs = sortStringSet(value.SourceReferenceIDs)
		if value.Metadata == nil {
			value.Metadata = map[string]any{}
		}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func sortedTSReferences(values map[string]analysis.Reference) []analysis.Reference {
	result := make([]analysis.Reference, 0, len(values))
	for _, value := range values {
		if value.Metadata == nil {
			value.Metadata = map[string]any{}
		}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func sortedTSSources(values map[string]analysis.SourceReference) []analysis.SourceReference {
	result := make([]analysis.SourceReference, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}
