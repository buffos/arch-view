package analysis

import (
	"encoding/json"
	"math"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

func ValidateAnalysisResult(result AnalysisResult, manifest Manifest, projectRoot string) error {
	if result.Status != StatusComplete && result.Status != StatusPartial && result.Status != StatusFailed && result.Status != StatusCancelled {
		return NewHostError(ErrResultInvalid, "analysis result has an invalid status", map[string]any{"status": result.Status})
	}
	expected := analyzerInfo(manifest)
	if result.Analyzer.ID != expected.ID ||
		result.Analyzer.Version != expected.Version ||
		result.Analyzer.Language != expected.Language ||
		result.Analyzer.APIVersion != expected.APIVersion {
		return NewHostError(ErrResultInvalid, "analysis result analyzer provenance does not match the selected manifest", map[string]any{"expected": expected.ID, "actual": result.Analyzer.ID})
	}
	if result.Analyzer.RuntimeMode == "" {
		if result.Analyzer.RuntimeSource != "" || result.Analyzer.RuntimePlatform != "" {
			return NewHostError(ErrResultInvalid, "analysis result runtime mode is required when provenance is set", nil)
		}
	} else {
		switch result.Analyzer.RuntimeMode {
		case RuntimeModePackaged, RuntimeModeInProcess, RuntimeModeExplicit:
		default:
			return NewHostError(ErrResultInvalid, "analysis result runtime mode is invalid", map[string]any{"runtime_mode": result.Analyzer.RuntimeMode})
		}
		if strings.TrimSpace(result.Analyzer.RuntimeSource) == "" || strings.TrimSpace(result.Analyzer.RuntimeSource) != result.Analyzer.RuntimeSource ||
			strings.TrimSpace(result.Analyzer.RuntimePlatform) == "" || strings.TrimSpace(result.Analyzer.RuntimePlatform) != result.Analyzer.RuntimePlatform ||
			strings.ContainsAny(result.Analyzer.RuntimeSource+result.Analyzer.RuntimePlatform, "\x00\r\n") {
			return NewHostError(ErrResultInvalid, "analysis result runtime provenance is invalid", map[string]any{"runtime_mode": result.Analyzer.RuntimeMode})
		}
	}
	if strings.TrimSpace(result.RunID) == "" {
		return NewHostError(ErrResultInvalid, "analysis result must include a run id", nil)
	}
	if strings.TrimSpace(result.Project.RootLabel) == "" || strings.TrimSpace(result.Project.Boundary) == "" {
		return NewHostError(ErrResultInvalid, "analysis result must include project boundary metadata", nil)
	}
	if projectRoot == "" {
		return NewHostError(ErrResultInvalid, "host project root cannot be empty", nil)
	}

	moduleIDs := make(map[string]struct{}, len(result.Modules))
	for _, module := range result.Modules {
		if module.ID == "" {
			return NewHostError(ErrResultInvalid, "module observation id cannot be empty", nil)
		}
		if _, exists := moduleIDs[module.ID]; exists {
			return NewHostError(ErrResultInvalid, "module observation ids must be unique", map[string]any{"id": module.ID})
		}
		moduleIDs[module.ID] = struct{}{}
	}
	referenceIDs := make(map[string]struct{}, len(result.References))
	for _, reference := range result.References {
		if reference.ID == "" {
			return NewHostError(ErrResultInvalid, "reference id cannot be empty", nil)
		}
		if _, exists := referenceIDs[reference.ID]; exists {
			return NewHostError(ErrResultInvalid, "reference ids must be unique", map[string]any{"id": reference.ID})
		}
		referenceIDs[reference.ID] = struct{}{}
	}
	sourceIDs := make(map[string]struct{}, len(result.SourceReferences))
	for _, source := range result.SourceReferences {
		if source.ID == "" || source.Path == "" {
			return NewHostError(ErrResultInvalid, "source references require id and repository-relative path", nil)
		}
		if !isRepositoryRelativePath(source.Path) {
			return NewHostError(ErrResultInvalid, "source reference paths cannot escape the project root", map[string]any{"path": source.Path})
		}
		if !validPosition(source.Start) || !validPosition(source.End) {
			return NewHostError(ErrResultInvalid, "source reference positions must use positive one-based line and column values", map[string]any{"source_reference_id": source.ID})
		}
		if _, exists := sourceIDs[source.ID]; exists {
			return NewHostError(ErrResultInvalid, "source reference ids must be unique", map[string]any{"id": source.ID})
		}
		sourceIDs[source.ID] = struct{}{}
	}
	for _, module := range result.Modules {
		for _, sourceID := range module.SourceReferenceIDs {
			if _, exists := sourceIDs[sourceID]; !exists {
				return NewHostError(ErrResultInvalid, "module references unknown source evidence", map[string]any{"module_id": module.ID, "source_reference_id": sourceID})
			}
		}
	}
	relationshipIDs := make(map[string]struct{}, len(result.Relationships))
	for _, relationship := range result.Relationships {
		if relationship.ID == "" || relationship.Type == "" || relationship.FromModuleID == "" {
			return NewHostError(ErrResultInvalid, "relationship requires id, type, and source module", nil)
		}
		if _, exists := relationshipIDs[relationship.ID]; exists {
			return NewHostError(ErrResultInvalid, "relationship ids must be unique", map[string]any{"id": relationship.ID})
		}
		relationshipIDs[relationship.ID] = struct{}{}
		if _, exists := moduleIDs[relationship.FromModuleID]; !exists {
			return NewHostError(ErrResultInvalid, "relationship source module is unknown", map[string]any{"relationship_id": relationship.ID, "module_id": relationship.FromModuleID})
		}
		if (relationship.ToModuleID == "") == (relationship.ToReferenceID == "") {
			return NewHostError(ErrResultInvalid, "relationship must have exactly one target", map[string]any{"relationship_id": relationship.ID})
		}
		if relationship.ToModuleID != "" {
			if _, exists := moduleIDs[relationship.ToModuleID]; !exists {
				return NewHostError(ErrResultInvalid, "relationship target module is unknown", map[string]any{"relationship_id": relationship.ID, "module_id": relationship.ToModuleID})
			}
		}
		if relationship.ToReferenceID != "" {
			if _, exists := referenceIDs[relationship.ToReferenceID]; !exists {
				return NewHostError(ErrResultInvalid, "relationship target reference is unknown", map[string]any{"relationship_id": relationship.ID, "reference_id": relationship.ToReferenceID})
			}
		}
		if relationship.Confidence != nil && !validConfidence(relationship.Confidence) {
			return NewHostError(ErrResultInvalid, "relationship confidence must have a basis and a score between zero and one", map[string]any{"relationship_id": relationship.ID})
		}
		for _, sourceID := range relationship.SourceReferenceIDs {
			if _, exists := sourceIDs[sourceID]; !exists {
				return NewHostError(ErrResultInvalid, "relationship references unknown source evidence", map[string]any{"relationship_id": relationship.ID, "source_reference_id": sourceID})
			}
		}
	}
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == "" || diagnostic.Message == "" {
			return NewHostError(ErrResultInvalid, "diagnostics require code and message", nil)
		}
		switch diagnostic.Severity {
		case "info", "warning", "error":
		default:
			return NewHostError(ErrResultInvalid, "diagnostic severity is invalid", map[string]any{"code": diagnostic.Code, "severity": diagnostic.Severity})
		}
		if diagnostic.Path != "" && !isRepositoryRelativePath(diagnostic.Path) {
			return NewHostError(ErrResultInvalid, "diagnostic paths must be repository-relative", map[string]any{"code": diagnostic.Code, "path": diagnostic.Path})
		}
		if !validPosition(diagnostic.Location) {
			return NewHostError(ErrResultInvalid, "diagnostic locations must use positive one-based line and column values", map[string]any{"code": diagnostic.Code})
		}
	}
	if result.Status == StatusComplete {
		for _, diagnostic := range result.Diagnostics {
			if diagnostic.Severity == "error" {
				return NewHostError(ErrResultInvalid, "complete result cannot contain error diagnostics", map[string]any{"code": diagnostic.Code})
			}
		}
	}
	if result.Status == StatusPartial {
		recoverable := false
		for _, diagnostic := range result.Diagnostics {
			if diagnostic.Recoverable {
				recoverable = true
				break
			}
		}
		if !recoverable {
			return NewHostError(ErrResultInvalid, "partial result must retain a recoverable diagnostic", nil)
		}
	}
	if _, err := json.Marshal(result); err != nil {
		return WrapHostError(ErrResultInvalid, "analysis result is not serializable", err, nil)
	}
	return nil
}

func isRepositoryRelativePath(path string) bool {
	normalized := normalizeRepositoryPath(path)
	if normalized == "" || normalized == "." || strings.HasPrefix(normalized, "/") || windowsAbsolutePath(normalized) {
		return false
	}
	return normalized != ".." && !strings.HasPrefix(normalized, "../")
}

func normalizeRepositoryPath(value string) string {
	if value == "" {
		return ""
	}
	return path.Clean(strings.ReplaceAll(filepath.ToSlash(value), "\\", "/"))
}

func windowsAbsolutePath(value string) bool {
	return len(value) >= 2 && ((value[0] >= 'a' && value[0] <= 'z') || (value[0] >= 'A' && value[0] <= 'Z')) && value[1] == ':'
}

func validPosition(position *Position) bool {
	return position == nil || (position.Line > 0 && position.Column > 0)
}

func validConfidence(confidence *Confidence) bool {
	return confidence != nil && strings.TrimSpace(confidence.Basis) != "" && confidence.Score >= 0 && confidence.Score <= 1 && !math.IsNaN(confidence.Score) && !math.IsInf(confidence.Score, 0)
}

func ComputeSummary(result AnalysisResult) AnalysisSummary {
	return AnalysisSummary{
		ModuleCount:          len(result.Modules),
		RelationshipCount:    len(result.Relationships),
		ReferenceCount:       len(result.References),
		SourceReferenceCount: len(result.SourceReferences),
		DiagnosticCount:      len(result.Diagnostics),
	}
}

func normalizeResultCollections(result *AnalysisResult) {
	if result.Modules == nil {
		result.Modules = []ModuleObservation{}
	}
	if result.Relationships == nil {
		result.Relationships = []RelationshipObservation{}
	}
	if result.References == nil {
		result.References = []Reference{}
	}
	if result.SourceReferences == nil {
		result.SourceReferences = []SourceReference{}
	}
	if result.Diagnostics == nil {
		result.Diagnostics = []Diagnostic{}
	}
	for index := range result.SourceReferences {
		if result.SourceReferences[index].Path != "" {
			result.SourceReferences[index].Path = normalizeRepositoryPath(result.SourceReferences[index].Path)
		}
	}
	for index := range result.Diagnostics {
		if result.Diagnostics[index].Path != "" {
			result.Diagnostics[index].Path = normalizeRepositoryPath(result.Diagnostics[index].Path)
		}
	}
	sort.Slice(result.Modules, func(i, j int) bool {
		return result.Modules[i].ID < result.Modules[j].ID
	})
	sort.Slice(result.Relationships, func(i, j int) bool {
		return result.Relationships[i].ID < result.Relationships[j].ID
	})
	sort.Slice(result.References, func(i, j int) bool {
		return result.References[i].ID < result.References[j].ID
	})
	sort.Slice(result.SourceReferences, func(i, j int) bool {
		return result.SourceReferences[i].ID < result.SourceReferences[j].ID
	})
	sort.Slice(result.Diagnostics, func(i, j int) bool {
		left, _ := json.Marshal(result.Diagnostics[i])
		right, _ := json.Marshal(result.Diagnostics[j])
		return string(left) < string(right)
	})
}
