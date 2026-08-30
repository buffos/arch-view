package sourceindex

import (
	"encoding/json"
	"path"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

// CloneSourceIndex returns an independent copy suitable for cache and
// aggregation boundaries. JSON cloning keeps the copy aligned with the
// transport contract and avoids sharing nested slices or extension payloads.
func CloneSourceIndex(value *analysis.SourceIndex) *analysis.SourceIndex {
	if value == nil {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var clone analysis.SourceIndex
	if err := json.Unmarshal(data, &clone); err != nil {
		return nil
	}
	return &clone
}

// NamespaceScopeSnapshot moves one authoritative snapshot into an aggregate
// scope. Its records remain structurally intact; only identity context and
// external evidence IDs are qualified for the aggregate consumer.
func NamespaceScopeSnapshot(value analysis.SourceIndexSnapshot, scopeID, projectRoot string) (analysis.SourceIndexSnapshot, error) {
	clone, ok := cloneSourceSnapshot(value)
	if !ok {
		return analysis.SourceIndexSnapshot{}, analysis.NewHostError(analysis.ErrSourceFactInvalid, "aggregate source snapshot could not be cloned", nil)
	}
	clone.ScopeContext.ScopeID = strings.TrimSpace(scopeID)
	if clone.ScopeContext.ScopeID == "" {
		return analysis.SourceIndexSnapshot{}, analysis.NewHostError(analysis.ErrSourceScopeUnavailable, "aggregate source snapshot requires a scope id", nil)
	}
	if projectRoot != "" {
		clone.ScopeContext.ProjectRoot = normalizeAggregatePath(projectRoot)
		if clone.ScopeContext.ProjectRoot == "" {
			clone.ScopeContext.ProjectRoot = "."
		}
	}
	oldSnapshotID := clone.SnapshotID
	clone.SnapshotID = opaqueID("snapshot", "aggregate", clone.ScopeContext.ScopeID, oldSnapshotID)
	clone.ScopeContext.SourceSnapshotIDs = nil
	canonicalizeSnapshot(&clone)
	clone.SnapshotDigest = analysis.ComputeSourceIndexDigest(clone)
	if err := validateSnapshotAfterNamespace(clone); err != nil {
		return analysis.SourceIndexSnapshot{}, err
	}
	return clone, nil
}

// BuildCombinedProjection creates a derived read model without merging facts
// by path. Every entity reference carries its originating scope and snapshot.
func BuildCombinedProjection(snapshots []analysis.SourceIndexSnapshot) (*analysis.SourceIndexSnapshot, error) {
	if len(snapshots) == 0 {
		return nil, nil
	}
	ordered := append([]analysis.SourceIndexSnapshot(nil), snapshots...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].SnapshotID < ordered[j].SnapshotID })
	ids := make([]string, 0, len(ordered))
	for _, snapshot := range ordered {
		ids = append(ids, snapshot.SnapshotID)
	}
	combinedScopeID := opaqueID("scope", "combined", strings.Join(ids, "\x00"))
	projection := analysis.SourceIndexSnapshot{
		SnapshotID:   opaqueID("projection", strings.Join(ids, "\x00")),
		SnapshotKind: analysis.SourceIndexSnapshotCombined,
		ScopeContext: analysis.ScopeContext{
			ScopeID:                combinedScopeID,
			ProjectRoot:            ".",
			SourceScopeFingerprint: digestStrings(ids),
			Mode:                   analysis.SourceIndexCombinedMode,
			SourceSnapshotIDs:      append([]string(nil), ids...),
		},
		Producer: analysis.ProducerContext{
			AnalyzerID:      "org.archview.aggregate",
			AnalyzerVersion: "1.0.0",
			ProtocolVersion: analysis.AnalyzerAPIVersion,
			Extractors:      []analysis.ExtractorIdentity{},
		},
		Input: analysis.InputContext{
			RequestedCapabilities: []string{},
		},
		Capabilities:  []analysis.CapabilityDescriptor{},
		Coverage:      []analysis.CoverageRecord{},
		Files:         []analysis.FileRecord{},
		Symbols:       []analysis.SymbolRecord{},
		Documentation: []analysis.DocumentationRecord{},
		Occurrences:   []analysis.SymbolOccurrence{},
		Relations:     []analysis.CodeRelation{},
		Metrics:       []analysis.MetricFact{},
		Extensions:    []analysis.ExtensionBlock{},
	}
	capabilitySet := make(map[string]analysis.CapabilityDescriptor)
	requestedSet := make(map[string]struct{})
	extractorSet := make(map[string]analysis.ExtractorIdentity)
	coverageSet := make(map[string]analysis.CoverageRecord)
	for _, snapshot := range ordered {
		for _, capability := range snapshot.Capabilities {
			capabilitySet[capability.ID] = capability
		}
		for _, capability := range snapshot.Input.RequestedCapabilities {
			requestedSet[capability] = struct{}{}
		}
		for _, extractor := range snapshot.Producer.Extractors {
			extractorSet[extractor.ID+"\x00"+extractor.Version] = extractor
		}
		for _, coverage := range snapshot.Coverage {
			key := coverage.Capability + "\x00" + coverage.SubjectKind
			if existing, ok := coverageSet[key]; ok {
				existing.EligibleCount = sumOptionalCounts(existing.EligibleCount, coverage.EligibleCount)
				existing.ObservedCount = sumOptionalCounts(existing.ObservedCount, coverage.ObservedCount)
				existing.Status = combineFactStatus(existing.Status, coverage.Status)
				existing.Provenance.Status = existing.Status
				if existing.Reason == "" {
					existing.Reason = coverage.Reason
				}
				coverageSet[key] = existing
			} else {
				coverageSet[key] = coverage
			}
		}
		for _, file := range snapshot.Files {
			value := file
			value.ID = projectionEntityID("file", snapshot, file.ID)
			value.Path = aggregateSourcePath(snapshot.ScopeContext.ProjectRoot, file.Path)
			value.Provenance.EvidenceIDs = namespaceExternalIDs(snapshot.ScopeContext.ScopeID, value.Provenance.EvidenceIDs)
			projection.Files = append(projection.Files, value)
		}
		fileIDs := make(map[string]string, len(snapshot.Files))
		for _, file := range snapshot.Files {
			fileIDs[file.ID] = projectionEntityID("file", snapshot, file.ID)
		}
		symbolIDs := make(map[string]string, len(snapshot.Symbols))
		for _, symbol := range snapshot.Symbols {
			symbolIDs[symbol.ID] = projectionEntityID("symbol", snapshot, symbol.ID)
		}
		documentationIDs := make(map[string]string, len(snapshot.Documentation))
		for _, documentation := range snapshot.Documentation {
			documentationIDs[documentation.ID] = projectionEntityID("documentation", snapshot, documentation.ID)
		}
		occurrenceIDs := make(map[string]string, len(snapshot.Occurrences))
		for _, occurrence := range snapshot.Occurrences {
			occurrenceIDs[occurrence.ID] = projectionEntityID("occurrence", snapshot, occurrence.ID)
		}
		for _, symbol := range snapshot.Symbols {
			value := symbol
			value.ID = symbolIDs[symbol.ID]
			value.Locations = append([]analysis.SymbolLocation(nil), symbol.Locations...)
			value.DocumentationIDs = mapIDs(symbol.DocumentationIDs, documentationIDs)
			value.Provenance.EvidenceIDs = namespaceExternalIDs(snapshot.ScopeContext.ScopeID, value.Provenance.EvidenceIDs)
			for index := range value.Locations {
				value.Locations[index].Span.FileID = fileIDs[value.Locations[index].Span.FileID]
				value.Locations[index].SourceReferenceIDs = namespaceExternalIDs(snapshot.ScopeContext.ScopeID, value.Locations[index].SourceReferenceIDs)
			}
			projection.Symbols = append(projection.Symbols, value)
		}
		for _, documentation := range snapshot.Documentation {
			value := documentation
			value.ID = documentationIDs[documentation.ID]
			value.Spans = append([]analysis.SourceSpan{}, documentation.Spans...)
			value.SubjectRef = projectionRef(documentation.SubjectRef, snapshot, fileIDs, symbolIDs, documentationIDs, occurrenceIDs)
			value.SourceReferenceIDs = namespaceExternalIDs(snapshot.ScopeContext.ScopeID, value.SourceReferenceIDs)
			value.Provenance.EvidenceIDs = namespaceExternalIDs(snapshot.ScopeContext.ScopeID, value.Provenance.EvidenceIDs)
			for index := range value.Spans {
				value.Spans[index].FileID = fileIDs[value.Spans[index].FileID]
			}
			projection.Documentation = append(projection.Documentation, value)
		}
		for _, occurrence := range snapshot.Occurrences {
			value := occurrence
			value.ID = occurrenceIDs[occurrence.ID]
			if value.SymbolRef != nil {
				ref := projectionRef(*value.SymbolRef, snapshot, fileIDs, symbolIDs, documentationIDs, occurrenceIDs)
				value.SymbolRef = &ref
			}
			value.SourceSpan.FileID = fileIDs[value.SourceSpan.FileID]
			value.Provenance.EvidenceIDs = namespaceExternalIDs(snapshot.ScopeContext.ScopeID, value.Provenance.EvidenceIDs)
			projection.Occurrences = append(projection.Occurrences, value)
		}
		for _, relation := range snapshot.Relations {
			value := relation
			value.ID = projectionEntityID("relation", snapshot, relation.ID)
			value.EvidenceSpans = append([]analysis.SourceSpan{}, relation.EvidenceSpans...)
			value.FromRef = projectionRef(relation.FromRef, snapshot, fileIDs, symbolIDs, documentationIDs, occurrenceIDs)
			if value.ToRef != nil {
				ref := projectionRef(*value.ToRef, snapshot, fileIDs, symbolIDs, documentationIDs, occurrenceIDs)
				value.ToRef = &ref
			}
			for index := range value.EvidenceSpans {
				value.EvidenceSpans[index].FileID = fileIDs[value.EvidenceSpans[index].FileID]
			}
			value.Provenance.EvidenceIDs = namespaceExternalIDs(snapshot.ScopeContext.ScopeID, value.Provenance.EvidenceIDs)
			projection.Relations = append(projection.Relations, value)
		}
		for _, metric := range snapshot.Metrics {
			value := metric
			value.ID = projectionEntityID("metric", snapshot, metric.ID)
			value.SubjectRef = projectionRef(metric.SubjectRef, snapshot, fileIDs, symbolIDs, documentationIDs, occurrenceIDs)
			value.Provenance.EvidenceIDs = namespaceExternalIDs(snapshot.ScopeContext.ScopeID, value.Provenance.EvidenceIDs)
			projection.Metrics = append(projection.Metrics, value)
		}
	}
	for _, capability := range capabilitySet {
		projection.Capabilities = append(projection.Capabilities, capability)
	}
	for capability := range requestedSet {
		projection.Input.RequestedCapabilities = append(projection.Input.RequestedCapabilities, capability)
	}
	for _, extractor := range extractorSet {
		projection.Producer.Extractors = append(projection.Producer.Extractors, extractor)
	}
	for _, coverage := range coverageSet {
		projection.Coverage = append(projection.Coverage, coverage)
	}
	sort.Strings(projection.Input.RequestedCapabilities)
	projection.Input.EligibleFileCount = len(projection.Files)
	projection.Input.SourceSetDigest = computeSourceSetDigest(projection.Files)
	canonicalizeSnapshot(&projection)
	projection.SnapshotDigest = analysis.ComputeSourceIndexDigest(projection)
	index := analysis.SourceIndex{SchemaVersion: analysis.SourceIndexSchemaVersion, Snapshots: ordered, Projection: &projection, Extensions: []analysis.ExtensionBlock{}}
	if err := analysis.ValidateSourceIndex(index); err != nil {
		return nil, err
	}
	return &projection, nil
}

func validateSnapshotAfterNamespace(snapshot analysis.SourceIndexSnapshot) error {
	index := analysis.SourceIndex{SchemaVersion: analysis.SourceIndexSchemaVersion, Snapshots: []analysis.SourceIndexSnapshot{snapshot}, Extensions: []analysis.ExtensionBlock{}}
	return analysis.ValidateSourceIndex(index)
}

func cloneSourceSnapshot(value analysis.SourceIndexSnapshot) (analysis.SourceIndexSnapshot, bool) {
	data, err := json.Marshal(value)
	if err != nil {
		return analysis.SourceIndexSnapshot{}, false
	}
	var clone analysis.SourceIndexSnapshot
	if err := json.Unmarshal(data, &clone); err != nil {
		return analysis.SourceIndexSnapshot{}, false
	}
	return clone, true
}

func projectionEntityID(kind string, snapshot analysis.SourceIndexSnapshot, value string) string {
	return opaqueID("projection-"+kind, snapshot.ScopeContext.ScopeID, snapshot.SnapshotID, value)
}

func projectionRef(value analysis.EntityRef, snapshot analysis.SourceIndexSnapshot, fileIDs, symbolIDs, documentationIDs, occurrenceIDs map[string]string) analysis.EntityRef {
	value.ScopeID = snapshot.ScopeContext.ScopeID
	value.SnapshotID = snapshot.SnapshotID
	switch value.Kind {
	case "file":
		value.ID = fileIDs[value.ID]
	case "symbol":
		value.ID = symbolIDs[value.ID]
	case "documentation":
		value.ID = documentationIDs[value.ID]
	case "occurrence":
		value.ID = occurrenceIDs[value.ID]
	case "module":
		value.ID = qualifiedModuleID(snapshot.ScopeContext.ScopeID, value.ID)
	}
	return value
}

func mapIDs(values []string, mapping map[string]string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if mapped, ok := mapping[value]; ok {
			result = append(result, mapped)
		}
	}
	sort.Strings(result)
	return result
}

func namespaceExternalIDs(scopeID string, values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			result = append(result, scopeID+"::"+value)
		}
	}
	sort.Strings(result)
	return result
}

func sumOptionalCounts(left, right *int) *int {
	if left == nil && right == nil {
		return nil
	}
	value := 0
	if left != nil {
		value += *left
	}
	if right != nil {
		value += *right
	}
	return &value
}

func combineFactStatus(left, right string) string {
	if left == right {
		return left
	}
	if left == analysis.FactStatusUnsupported && right == analysis.FactStatusUnsupported {
		return analysis.FactStatusUnsupported
	}
	if left == analysis.FactStatusObserved && right == analysis.FactStatusObserved {
		return analysis.FactStatusObserved
	}
	return analysis.FactStatusPartial
}

func aggregateSourcePath(root, value string) string {
	root = normalizeAggregatePath(root)
	value = normalizeAggregatePath(value)
	if root == "" || root == "." {
		return value
	}
	if value == "" || value == "." {
		return root
	}
	return normalizeAggregatePath(path.Join(root, value))
}

func normalizeAggregatePath(value string) string {
	value = strings.TrimPrefix(strings.ReplaceAll(strings.TrimSpace(value), "\\", "/"), "./")
	if value == "" {
		return "."
	}
	return path.Clean(value)
}

func qualifiedModuleID(scopeID, moduleID string) string {
	const hexDigits = "0123456789ABCDEF"
	var builder strings.Builder
	for _, character := range []byte(moduleID) {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') || strings.ContainsRune("-._~", rune(character)) {
			builder.WriteByte(character)
			continue
		}
		builder.WriteByte('%')
		builder.WriteByte(hexDigits[character>>4])
		builder.WriteByte(hexDigits[character&0x0f])
	}
	return scopeID + "::" + builder.String()
}
