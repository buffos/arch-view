// Package adapter translates existing analyzer/model contracts into the
// neutral quality input boundary. The quality package itself remains free of
// those dependencies so optional report fields do not create import cycles.
package adapter

import (
	"fmt"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/quality"
)

func EvaluationInputFromSourceIndex(index analysis.SourceIndex) (quality.EvaluationInput, error) {
	if err := analysis.ValidateSourceIndex(index); err != nil {
		return quality.EvaluationInput{}, fmt.Errorf("source-index cannot be adapted for quality evaluation: %w", err)
	}
	if index.Projection != nil {
		return quality.EvaluationInput{SourceSnapshots: []quality.SourceSnapshot{SourceSnapshotFromAnalysis(*index.Projection)}, Options: map[string]any{}}, nil
	}
	result := quality.EvaluationInput{SourceSnapshots: make([]quality.SourceSnapshot, 0, len(index.Snapshots)), Options: map[string]any{}}
	for _, snapshot := range index.Snapshots {
		result.SourceSnapshots = append(result.SourceSnapshots, SourceSnapshotFromAnalysis(snapshot))
	}
	return result, nil
}

func SourceSnapshotFromAnalysis(snapshot analysis.SourceIndexSnapshot) quality.SourceSnapshot {
	result := quality.SourceSnapshot{
		SnapshotID:    snapshot.SnapshotID,
		ScopeID:       snapshot.ScopeContext.ScopeID,
		Capabilities:  make([]quality.CapabilityDescriptor, 0, len(snapshot.Capabilities)),
		Coverage:      make([]quality.SourceCoverage, 0, len(snapshot.Coverage)),
		Files:         make([]quality.SourceFile, 0, len(snapshot.Files)),
		Symbols:       make([]quality.SourceSymbol, 0, len(snapshot.Symbols)),
		Documentation: make([]quality.SourceDocumentation, 0, len(snapshot.Documentation)),
		Relations:     make([]quality.SourceRelation, 0, len(snapshot.Relations)),
		Metrics:       make([]quality.MetricFact, 0, len(snapshot.Metrics)),
	}
	for _, value := range snapshot.Capabilities {
		result.Capabilities = append(result.Capabilities, quality.CapabilityDescriptor{ID: value.ID, Version: value.Version, SupportedLanguages: append([]string(nil), value.SupportedLanguages...), Description: value.Description})
	}
	for _, value := range snapshot.Coverage {
		result.Coverage = append(result.Coverage, quality.SourceCoverage{Capability: value.Capability, SubjectKind: value.SubjectKind, Status: value.Status, EligibleCount: cloneInt(value.EligibleCount), ObservedCount: cloneInt(value.ObservedCount), Reason: value.Reason, Provenance: provenanceFromAnalysis(value.Provenance)})
	}
	for _, value := range snapshot.Files {
		result.Files = append(result.Files, quality.SourceFile{ID: value.ID, StableKey: value.Path, Path: value.Path, Language: value.Language.ID, LineCount: value.Size.LineCount, ByteCount: value.Size.ByteCount, ContentHash: digestFromAnalysis(value.Size.ContentHash), AnalysisStatus: value.AnalysisStatus, Provenance: provenanceFromAnalysis(value.Provenance)})
	}
	for _, value := range snapshot.Symbols {
		locations := make([]quality.SourceLocation, 0, len(value.Locations))
		for _, location := range value.Locations {
			locations = append(locations, quality.SourceLocation{Kind: location.Kind, Span: spanFromAnalysis(location.Span), SourceReferenceIDs: append([]string(nil), location.SourceReferenceIDs...)})
		}
		var bodySpan *quality.SourceSpan
		if value.BodySpan != nil {
			converted := spanFromAnalysis(*value.BodySpan)
			bodySpan = &converted
		}
		visibilityStatus := value.Visibility.Provenance.Status
		if value.Visibility.Classification == "unknown" {
			visibilityStatus = "unknown"
		}
		result.Symbols = append(result.Symbols, quality.SourceSymbol{ID: value.ID, StableKey: value.StableKey, Name: value.Name, QualifiedName: value.QualifiedName, Category: value.Category, LanguageKind: value.LanguageKind, Visibility: value.Visibility.Classification, VisibilityStatus: visibilityStatus, Locations: locations, BodySpan: bodySpan, DocumentationIDs: append([]string(nil), value.DocumentationIDs...), MemberCount: cloneInt(value.MemberCount), MethodCount: cloneInt(value.MethodCount), DependencyCount: cloneInt(value.DependencyCount), ConcreteDependencyCount: cloneInt(value.ConcreteDependencyCount), InterfaceMethodCount: cloneInt(value.InterfaceMethodCount), TypeSwitchCount: cloneInt(value.TypeSwitchCount), HierarchyDepth: cloneInt(value.HierarchyDepth), DerivedTypeCount: cloneInt(value.DerivedTypeCount), AbstractionCount: cloneInt(value.AbstractionCount), StructuralFacts: cloneIntMap(value.StructuralFacts), Provenance: provenanceFromAnalysis(value.Provenance)})
	}
	for _, value := range snapshot.Documentation {
		spans := make([]quality.SourceSpan, 0, len(value.Spans))
		for _, span := range value.Spans {
			spans = append(spans, spanFromAnalysis(span))
		}
		result.Documentation = append(result.Documentation, quality.SourceDocumentation{ID: value.ID, SubjectRef: entityFromAnalysis(value.SubjectRef, snapshot), Status: value.Status, Spans: spans, Provenance: provenanceFromAnalysis(value.Provenance)})
	}
	for _, value := range snapshot.Relations {
		spans := make([]quality.SourceSpan, 0, len(value.EvidenceSpans))
		for _, span := range value.EvidenceSpans {
			spans = append(spans, spanFromAnalysis(span))
		}
		var toRef *quality.EntityRef
		if value.ToRef != nil {
			converted := entityFromAnalysis(*value.ToRef, snapshot)
			toRef = &converted
		}
		result.Relations = append(result.Relations, quality.SourceRelation{ID: value.ID, Category: value.Category, FromRef: entityFromAnalysis(value.FromRef, snapshot), ToRef: toRef, EvidenceSpans: spans, Provenance: provenanceFromAnalysis(value.Provenance)})
	}
	for _, value := range snapshot.Metrics {
		result.Metrics = append(result.Metrics, metricFromAnalysis(value, snapshot))
	}
	return result
}

func metricFromAnalysis(value analysis.MetricFact, snapshot analysis.SourceIndexSnapshot) quality.MetricFact {
	return quality.MetricFact{ID: value.ID, SubjectRef: entityFromAnalysis(value.SubjectRef, snapshot), MetricID: value.MetricID, Value: quality.MetricValue{Kind: value.Value.Kind, Value: value.Value.Value}, Unit: value.Unit, FormulaID: value.FormulaID, FormulaVersion: value.FormulaVersion, Provenance: provenanceFromAnalysis(value.Provenance), Extensions: extensionsFromAnalysis(value.Extensions)}
}

func entityFromAnalysis(value analysis.EntityRef, snapshot analysis.SourceIndexSnapshot) quality.EntityRef {
	if value.ScopeID == "" {
		value.ScopeID = snapshot.ScopeContext.ScopeID
	}
	if value.SnapshotID == "" {
		value.SnapshotID = snapshot.SnapshotID
	}
	stableKey := ""
	switch value.Kind {
	case "file":
		for _, file := range snapshot.Files {
			if file.ID == value.ID {
				stableKey = file.Path
				break
			}
		}
	case "symbol":
		for _, symbol := range snapshot.Symbols {
			if symbol.ID == value.ID {
				stableKey = symbol.StableKey
				if stableKey == "" {
					stableKey = symbol.QualifiedName
				}
				if stableKey == "" {
					stableKey = symbol.Name
				}
				break
			}
		}
	}
	return quality.EntityRef{Kind: value.Kind, ID: value.ID, StableKey: stableKey, SnapshotID: value.SnapshotID, ScopeID: value.ScopeID}
}

func spanFromAnalysis(value analysis.SourceSpan) quality.SourceSpan {
	return quality.SourceSpan{FileID: value.FileID, Start: quality.SpanPosition{ByteOffset: value.Start.ByteOffset, Line: value.Start.Line, Column: value.Start.Column}, End: quality.SpanPosition{ByteOffset: value.End.ByteOffset, Line: value.End.Line, Column: value.End.Column}, CoordinateSystem: value.CoordinateSystem, ContentHash: digestFromAnalysis(value.ContentHash)}
}

func digestFromAnalysis(value analysis.ContentDigest) quality.ContentDigest {
	return quality.ContentDigest{Algorithm: value.Algorithm, Value: value.Value}
}
func provenanceFromAnalysis(value analysis.FactProvenance) quality.FactProvenance {
	provider := strings.TrimSpace(value.Provider)
	if provider != "" && !strings.Contains(provider, ":") {
		// Analyzer manifests historically use IDs such as org.archview.go.
		// Quality provenance is a namespaced contract, so retain that identity
		// under an explicit analyzer namespace at the adapter boundary.
		provider = "analyzer:" + provider
	}
	return quality.FactProvenance{Status: value.Status, Basis: value.Basis, EvidenceIDs: append([]string(nil), value.EvidenceIDs...), Provider: provider, ProviderVersion: value.ProviderVersion}
}
func extensionsFromAnalysis(values []analysis.ExtensionBlock) []quality.ExtensionBlock {
	result := make([]quality.ExtensionBlock, 0, len(values))
	for _, value := range values {
		result = append(result, quality.ExtensionBlock{Namespace: value.Namespace, SchemaVersion: value.SchemaVersion, Capability: value.Capability, Payload: value.Payload})
	}
	return result
}
func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneIntMap(value map[string]int) map[string]int {
	if value == nil {
		return nil
	}
	result := make(map[string]int, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}
