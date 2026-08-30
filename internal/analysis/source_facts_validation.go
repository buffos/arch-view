package analysis

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
)

// ValidateSourceIndex validates the source-facts attachment without making
// assumptions about a language-specific extractor. The architecture result
// remains backward compatible because callers only invoke this when the
// optional attachment is present.
func ValidateSourceIndex(index SourceIndex) error {
	if index.SchemaVersion != SourceIndexSchemaVersion {
		return sourceFactError(ErrSourceFactInvalid, "source-index schema version is unsupported", map[string]any{"schema_version": index.SchemaVersion})
	}
	if index.Snapshots == nil {
		return sourceFactError(ErrSourceFactInvalid, "source-index snapshots must be serialized as an array", nil)
	}
	if err := validateExtensions(index.Extensions); err != nil {
		return err
	}
	if !sort.SliceIsSorted(index.Snapshots, func(i, j int) bool { return index.Snapshots[i].SnapshotID < index.Snapshots[j].SnapshotID }) {
		return sourceFactError(ErrSourceFactInvalid, "source-index snapshots must be sorted by stable identity", nil)
	}
	snapshotIDs := make(map[string]struct{}, len(index.Snapshots))
	for _, snapshot := range index.Snapshots {
		if _, exists := snapshotIDs[snapshot.SnapshotID]; exists {
			return sourceFactError(ErrSourceFactInvalid, "source-index snapshot ids must be unique", map[string]any{"snapshot_id": snapshot.SnapshotID})
		}
		snapshotIDs[snapshot.SnapshotID] = struct{}{}
		if err := validateSourceIndexSnapshot(snapshot); err != nil {
			return err
		}
	}
	if index.Projection != nil {
		if index.Projection.SnapshotKind != SourceIndexSnapshotCombined || index.Projection.ScopeContext.Mode != SourceIndexCombinedMode {
			return sourceFactError(ErrSourceFactInvalid, "source-index projection must be a combined projection snapshot", nil)
		}
		if err := validateSourceIndexSnapshot(*index.Projection); err != nil {
			return err
		}
	}
	if _, err := json.Marshal(index); err != nil {
		return WrapHostError(ErrSourceFactInvalid, "source-index is not serializable", err, nil)
	}
	return nil
}

// ComputeSourceIndexDigest returns the deterministic semantic digest for a
// snapshot. The digest field itself is intentionally excluded from the
// payload, so callers can calculate it before publishing the snapshot.
func ComputeSourceIndexDigest(snapshot SourceIndexSnapshot) ContentDigest {
	payload := sourceIndexDigestPayload{
		SnapshotID:    snapshot.SnapshotID,
		SnapshotKind:  snapshot.SnapshotKind,
		ScopeContext:  snapshot.ScopeContext,
		Producer:      snapshot.Producer,
		Input:         snapshot.Input,
		Capabilities:  snapshot.Capabilities,
		Coverage:      snapshot.Coverage,
		Files:         snapshot.Files,
		Symbols:       snapshot.Symbols,
		Documentation: snapshot.Documentation,
		Occurrences:   snapshot.Occurrences,
		Relations:     snapshot.Relations,
		Metrics:       snapshot.Metrics,
		Extensions:    snapshot.Extensions,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		// All source-index fields are JSON values by contract. Returning a
		// deterministic empty digest is preferable to panicking in a helper
		// that is also used by diagnostics and tests.
		return ContentDigest{Algorithm: SourceHashAlgorithm, Value: ""}
	}
	digest := sha256.Sum256(data)
	return ContentDigest{Algorithm: SourceHashAlgorithm, Value: hex.EncodeToString(digest[:])}
}

// ValidateSourceFactCollections validates normalized fact collections before
// an extractor batch is committed to a snapshot. This lets the builder reject
// one malformed batch without discarding valid facts from other files or
// extractors.
func ValidateSourceFactCollections(files []FileRecord, symbols []SymbolRecord, documentation []DocumentationRecord, occurrences []SymbolOccurrence, relations []CodeRelation, metrics []MetricFact) error {
	fileIDs := make(map[string]struct{}, len(files))
	fileBoundaries := make(map[string]sourceFileBoundary, len(files))
	for _, file := range files {
		fileIDs[file.ID] = struct{}{}
		fileBoundaries[file.ID] = sourceFileBoundary{hash: file.Size.ContentHash, byteCount: file.Size.ByteCount}
		if err := validateFileRecord(file); err != nil {
			return err
		}
	}
	symbolIDs := make(map[string]struct{}, len(symbols))
	for _, symbol := range symbols {
		if _, exists := symbolIDs[symbol.ID]; exists {
			return sourceFactError(ErrSourceFactInvalid, "source-index symbol ids must be unique", map[string]any{"symbol_id": symbol.ID})
		}
		symbolIDs[symbol.ID] = struct{}{}
		if err := validateSymbolRecord(symbol, fileIDs, fileBoundaries); err != nil {
			return err
		}
	}
	documentationIDs := make(map[string]struct{}, len(documentation))
	for _, value := range documentation {
		if _, exists := documentationIDs[value.ID]; exists {
			return sourceFactError(ErrSourceFactInvalid, "source-index documentation ids must be unique", map[string]any{"documentation_id": value.ID})
		}
		documentationIDs[value.ID] = struct{}{}
		if err := validateDocumentationRecord(value, fileIDs, symbolIDs, fileBoundaries); err != nil {
			return err
		}
	}
	occurrenceIDs := make(map[string]struct{}, len(occurrences))
	for _, value := range occurrences {
		if _, exists := occurrenceIDs[value.ID]; exists {
			return sourceFactError(ErrSourceFactInvalid, "source-index occurrence ids must be unique", map[string]any{"occurrence_id": value.ID})
		}
		occurrenceIDs[value.ID] = struct{}{}
		if err := validateOccurrence(value, fileIDs, symbolIDs, fileBoundaries); err != nil {
			return err
		}
	}
	for _, value := range relations {
		if err := validateRelation(value, fileIDs, symbolIDs, documentationIDs, occurrenceIDs, fileBoundaries); err != nil {
			return err
		}
	}
	for _, value := range metrics {
		if err := validateMetric(value, fileIDs, symbolIDs, documentationIDs, occurrenceIDs); err != nil {
			return err
		}
	}
	return nil
}

type sourceIndexDigestPayload struct {
	SnapshotID    string                 `json:"snapshot_id"`
	SnapshotKind  string                 `json:"snapshot_kind"`
	ScopeContext  ScopeContext           `json:"scope_context"`
	Producer      ProducerContext        `json:"producer"`
	Input         InputContext           `json:"input"`
	Capabilities  []CapabilityDescriptor `json:"capabilities"`
	Coverage      []CoverageRecord       `json:"coverage"`
	Files         []FileRecord           `json:"files"`
	Symbols       []SymbolRecord         `json:"symbols"`
	Documentation []DocumentationRecord  `json:"documentation"`
	Occurrences   []SymbolOccurrence     `json:"occurrences"`
	Relations     []CodeRelation         `json:"relations"`
	Metrics       []MetricFact           `json:"metrics"`
	Extensions    []ExtensionBlock       `json:"extensions"`
}

func validateSourceIndexSnapshot(snapshot SourceIndexSnapshot) error {
	if strings.TrimSpace(snapshot.SnapshotID) == "" {
		return sourceFactError(ErrSourceFactInvalid, "source-index snapshot id is required", nil)
	}
	if snapshot.SnapshotKind != SourceIndexSnapshotScope && snapshot.SnapshotKind != SourceIndexSnapshotCombined {
		return sourceFactError(ErrSourceFactInvalid, "source-index snapshot kind is invalid", map[string]any{"snapshot_id": snapshot.SnapshotID})
	}
	if err := validateScopeContext(snapshot); err != nil {
		return err
	}
	if err := validateProducer(snapshot.Producer); err != nil {
		return err
	}
	if snapshot.Input.EligibleFileCount < 0 || snapshot.Input.EligibleFileCount != len(snapshot.Files) {
		return sourceFactError(ErrSourceFactInvalid, "source-index input file count does not match file facts", map[string]any{"snapshot_id": snapshot.SnapshotID})
	}
	if err := validateDigest(snapshot.Input.SourceSetDigest, "source set digest"); err != nil {
		return err
	}
	if !sortedUniqueStrings(snapshot.Input.RequestedCapabilities) {
		return sourceFactError(ErrSourceFactInvalid, "requested source-index capabilities must be sorted and unique", map[string]any{"snapshot_id": snapshot.SnapshotID})
	}
	if err := validateCapabilities(snapshot.Capabilities); err != nil {
		return err
	}
	if err := validateCoverage(snapshot.Coverage); err != nil {
		return err
	}
	if err := validateExtensions(snapshot.Extensions); err != nil {
		return err
	}

	fileIDs := make(map[string]struct{}, len(snapshot.Files))
	fileBoundaries := make(map[string]sourceFileBoundary, len(snapshot.Files))
	filePaths := make(map[string]struct{}, len(snapshot.Files))
	for _, file := range snapshot.Files {
		if _, exists := fileIDs[file.ID]; exists {
			return sourceFactError(ErrSourceFactInvalid, "source-index file ids must be unique", map[string]any{"file_id": file.ID})
		}
		if _, exists := filePaths[file.Path]; exists {
			return sourceFactError(ErrSourceFactInvalid, "source-index file paths must be unique within a snapshot", map[string]any{"path": file.Path})
		}
		fileIDs[file.ID] = struct{}{}
		fileBoundaries[file.ID] = sourceFileBoundary{hash: file.Size.ContentHash, byteCount: file.Size.ByteCount}
		filePaths[file.Path] = struct{}{}
		if err := validateFileRecord(file); err != nil {
			return err
		}
	}
	paths := sourceIndexPaths(snapshot.Files)
	if !sort.SliceIsSorted(snapshot.Files, func(i, j int) bool { return fileFactKey(snapshot.Files[i]) < fileFactKey(snapshot.Files[j]) }) {
		return sourceFactError(ErrSourceFactInvalid, "source-index files must be sorted by canonical path", map[string]any{"snapshot_id": snapshot.SnapshotID})
	}

	symbolIDs := make(map[string]struct{}, len(snapshot.Symbols))
	for _, symbol := range snapshot.Symbols {
		if _, exists := symbolIDs[symbol.ID]; exists {
			return sourceFactError(ErrSourceFactInvalid, "source-index symbol ids must be unique", map[string]any{"symbol_id": symbol.ID})
		}
		symbolIDs[symbol.ID] = struct{}{}
		if err := validateSymbolRecord(symbol, fileIDs, fileBoundaries); err != nil {
			return err
		}
	}
	if !sort.SliceIsSorted(snapshot.Symbols, func(i, j int) bool {
		return symbolFactKey(snapshot.Symbols[i], paths) < symbolFactKey(snapshot.Symbols[j], paths)
	}) {
		return sourceFactError(ErrSourceFactInvalid, "source-index symbols must use canonical structural ordering", map[string]any{"snapshot_id": snapshot.SnapshotID})
	}

	documentationIDs := make(map[string]struct{}, len(snapshot.Documentation))
	primaryBySubject := make(map[string]string)
	for _, documentation := range snapshot.Documentation {
		if _, exists := documentationIDs[documentation.ID]; exists {
			return sourceFactError(ErrSourceFactInvalid, "source-index documentation ids must be unique", map[string]any{"documentation_id": documentation.ID})
		}
		documentationIDs[documentation.ID] = struct{}{}
		if err := validateDocumentationRecord(documentation, fileIDs, symbolIDs, fileBoundaries); err != nil {
			return err
		}
		if documentation.IsPrimary {
			key := entityKey(documentation.SubjectRef) + "\x00" + documentation.SelectionGroup
			if previous, exists := primaryBySubject[key]; exists {
				return sourceFactError(ErrDocumentationSelectionInvalid, "source-index documentation has multiple primary candidates", map[string]any{"first": previous, "second": documentation.ID})
			}
			primaryBySubject[key] = documentation.ID
		}
	}
	if !sort.SliceIsSorted(snapshot.Documentation, func(i, j int) bool {
		return documentationFactKey(snapshot.Documentation[i], paths) < documentationFactKey(snapshot.Documentation[j], paths)
	}) {
		return sourceFactError(ErrSourceFactInvalid, "source-index documentation must use canonical structural ordering", map[string]any{"snapshot_id": snapshot.SnapshotID})
	}

	occurrenceIDs := make(map[string]struct{}, len(snapshot.Occurrences))
	for _, occurrence := range snapshot.Occurrences {
		if _, exists := occurrenceIDs[occurrence.ID]; exists {
			return sourceFactError(ErrSourceFactInvalid, "source-index occurrence ids must be unique", map[string]any{"occurrence_id": occurrence.ID})
		}
		occurrenceIDs[occurrence.ID] = struct{}{}
		if err := validateOccurrence(occurrence, fileIDs, symbolIDs, fileBoundaries); err != nil {
			return err
		}
	}
	if !sort.SliceIsSorted(snapshot.Occurrences, func(i, j int) bool {
		return occurrenceFactKey(snapshot.Occurrences[i], paths) < occurrenceFactKey(snapshot.Occurrences[j], paths)
	}) {
		return sourceFactError(ErrSourceFactInvalid, "source-index occurrences must use canonical structural ordering", map[string]any{"snapshot_id": snapshot.SnapshotID})
	}

	relationIDs := make(map[string]struct{}, len(snapshot.Relations))
	for _, relation := range snapshot.Relations {
		if _, exists := relationIDs[relation.ID]; exists {
			return sourceFactError(ErrSourceRelationInvalid, "source-index relation ids must be unique", map[string]any{"relation_id": relation.ID})
		}
		relationIDs[relation.ID] = struct{}{}
		if err := validateRelation(relation, fileIDs, symbolIDs, documentationIDs, occurrenceIDs, fileBoundaries); err != nil {
			return err
		}
	}
	if !sort.SliceIsSorted(snapshot.Relations, func(i, j int) bool {
		return relationFactKey(snapshot.Relations[i], paths) < relationFactKey(snapshot.Relations[j], paths)
	}) {
		return sourceFactError(ErrSourceFactInvalid, "source-index relations must use canonical structural ordering", map[string]any{"snapshot_id": snapshot.SnapshotID})
	}

	metricIDs := make(map[string]struct{}, len(snapshot.Metrics))
	for _, metric := range snapshot.Metrics {
		if _, exists := metricIDs[metric.ID]; exists {
			return sourceFactError(ErrMetricFactInvalid, "source-index metric ids must be unique", map[string]any{"metric_id": metric.ID})
		}
		metricIDs[metric.ID] = struct{}{}
		if err := validateMetric(metric, fileIDs, symbolIDs, documentationIDs, occurrenceIDs); err != nil {
			return err
		}
	}
	if !sort.SliceIsSorted(snapshot.Metrics, func(i, j int) bool { return metricFactKey(snapshot.Metrics[i]) < metricFactKey(snapshot.Metrics[j]) }) {
		return sourceFactError(ErrSourceFactInvalid, "source-index metrics must use canonical structural ordering", map[string]any{"snapshot_id": snapshot.SnapshotID})
	}

	expected := ComputeSourceIndexDigest(snapshot)
	if snapshot.SnapshotDigest != expected {
		return sourceFactError(ErrSourceIndexDigestMismatch, "source-index snapshot digest does not match its canonical payload", map[string]any{"snapshot_id": snapshot.SnapshotID, "expected": expected.Value, "actual": snapshot.SnapshotDigest.Value})
	}
	return nil
}

func validateScopeContext(snapshot SourceIndexSnapshot) error {
	scope := snapshot.ScopeContext
	if strings.TrimSpace(scope.ScopeID) == "" {
		return sourceFactError(ErrSourceScopeUnavailable, "source-index scope id is required", map[string]any{"snapshot_id": snapshot.SnapshotID})
	}
	if scope.ProjectRoot == "" {
		return sourceFactError(ErrSourceScopeUnavailable, "source-index project root is required", map[string]any{"snapshot_id": snapshot.SnapshotID})
	}
	if scope.ProjectRoot != "." && (scope.ProjectRoot != normalizeRepositoryPath(scope.ProjectRoot) || !isRepositoryRelativePath(scope.ProjectRoot)) {
		return sourceFactError(ErrSourceScopeUnavailable, "source-index project root must be a normalized relative path", map[string]any{"project_root": scope.ProjectRoot})
	}
	if err := validateDigest(scope.SourceScopeFingerprint, "source scope fingerprint"); err != nil {
		return err
	}
	if scope.SourcePolicyFingerprint != nil {
		if err := validateDigest(*scope.SourcePolicyFingerprint, "source policy fingerprint"); err != nil {
			return err
		}
	}
	if snapshot.SnapshotKind == SourceIndexSnapshotScope {
		if scope.Mode != SourceIndexScopeMode || len(scope.SourceSnapshotIDs) != 0 {
			return sourceFactError(ErrSourceScopeUnavailable, "scope snapshots cannot declare combined projection sources", map[string]any{"snapshot_id": snapshot.SnapshotID})
		}
	} else if scope.Mode != SourceIndexCombinedMode || len(scope.SourceSnapshotIDs) == 0 {
		return sourceFactError(ErrSourceScopeUnavailable, "combined projections require source snapshot ids", map[string]any{"snapshot_id": snapshot.SnapshotID})
	} else if !sortedUniqueStrings(scope.SourceSnapshotIDs) {
		return sourceFactError(ErrSourceScopeUnavailable, "combined projection source snapshot ids must be sorted and unique", map[string]any{"snapshot_id": snapshot.SnapshotID})
	}
	return nil
}

func validateProducer(producer ProducerContext) error {
	if strings.TrimSpace(producer.AnalyzerID) == "" || strings.TrimSpace(producer.AnalyzerVersion) == "" {
		return sourceFactError(ErrSourceFactInvalid, "source-index producer requires analyzer identity and version", nil)
	}
	if producer.Extractors == nil {
		return sourceFactError(ErrSourceFactInvalid, "source-index producer extractors must be serialized as an array", nil)
	}
	seen := make(map[string]struct{}, len(producer.Extractors))
	for _, extractor := range producer.Extractors {
		if strings.TrimSpace(extractor.ID) == "" || strings.TrimSpace(extractor.Version) == "" {
			return sourceFactError(ErrSourceFactInvalid, "source-index extractor identity is incomplete", nil)
		}
		key := extractor.ID + "\x00" + extractor.Version
		if _, exists := seen[key]; exists {
			return sourceFactError(ErrSourceFactInvalid, "source-index extractor identities must be unique", map[string]any{"id": extractor.ID})
		}
		seen[key] = struct{}{}
	}
	if !sort.SliceIsSorted(producer.Extractors, func(i, j int) bool {
		if producer.Extractors[i].ID == producer.Extractors[j].ID {
			return producer.Extractors[i].Version < producer.Extractors[j].Version
		}
		return producer.Extractors[i].ID < producer.Extractors[j].ID
	}) {
		return sourceFactError(ErrSourceFactInvalid, "source-index extractor identities must be sorted", nil)
	}
	return nil
}

func validateCapabilities(values []CapabilityDescriptor) error {
	if values == nil {
		return sourceFactError(ErrSourceFactInvalid, "source-index capabilities must be serialized as an array", nil)
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if strings.TrimSpace(value.ID) == "" {
			return sourceFactError(ErrSourceFactInvalid, "source-index capability id is required", nil)
		}
		if _, exists := seen[value.ID]; exists {
			return sourceFactError(ErrSourceFactInvalid, "source-index capability ids must be unique", map[string]any{"capability": value.ID})
		}
		seen[value.ID] = struct{}{}
		if !sortedUniqueStrings(value.SupportedLanguages) {
			return sourceFactError(ErrSourceFactInvalid, "source-index capability languages must be sorted and unique", map[string]any{"capability": value.ID})
		}
	}
	if !sort.SliceIsSorted(values, func(i, j int) bool {
		if values[i].ID == values[j].ID {
			return values[i].Version < values[j].Version
		}
		return values[i].ID < values[j].ID
	}) {
		return sourceFactError(ErrSourceFactInvalid, "source-index capabilities must be sorted", nil)
	}
	return nil
}

func validateCoverage(values []CoverageRecord) error {
	if values == nil {
		return sourceFactError(ErrSourceFactInvalid, "source-index coverage must be serialized as an array", nil)
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if strings.TrimSpace(value.Capability) == "" || strings.TrimSpace(value.SubjectKind) == "" {
			return sourceFactError(ErrSourceFactInvalid, "source-index coverage requires capability and subject kind", nil)
		}
		if !validFactStatus(value.Status) {
			return sourceFactError(ErrSourceFactInvalid, "source-index coverage status is invalid", map[string]any{"status": value.Status})
		}
		key := value.Capability + "\x00" + value.SubjectKind
		if _, exists := seen[key]; exists {
			return sourceFactError(ErrSourceFactInvalid, "source-index coverage keys must be unique", map[string]any{"capability": value.Capability, "subject_kind": value.SubjectKind})
		}
		seen[key] = struct{}{}
		if value.EligibleCount != nil && *value.EligibleCount < 0 {
			return sourceFactError(ErrSourceFactInvalid, "source-index eligible coverage count cannot be negative", nil)
		}
		if value.ObservedCount != nil && *value.ObservedCount < 0 {
			return sourceFactError(ErrSourceFactInvalid, "source-index observed coverage count cannot be negative", nil)
		}
		if value.EligibleCount != nil && value.ObservedCount != nil && *value.ObservedCount > *value.EligibleCount {
			return sourceFactError(ErrSourceFactInvalid, "source-index observed coverage count cannot exceed eligible count", nil)
		}
		if err := validateProvenance(value.Provenance); err != nil {
			return err
		}
	}
	if !sort.SliceIsSorted(values, func(i, j int) bool {
		if values[i].Capability == values[j].Capability {
			return values[i].SubjectKind < values[j].SubjectKind
		}
		return values[i].Capability < values[j].Capability
	}) {
		return sourceFactError(ErrSourceFactInvalid, "source-index coverage must be sorted", nil)
	}
	return nil
}

func validateFileRecord(file FileRecord) error {
	if strings.TrimSpace(file.ID) == "" || file.Path == "" || file.Path != normalizeRepositoryPath(file.Path) || !isRepositoryRelativePath(file.Path) {
		return sourceFactError(ErrSourceFactInvalid, "source-index file identity or path is invalid", map[string]any{"file_id": file.ID, "path": file.Path})
	}
	if file.Language.ID == "" {
		return sourceFactError(ErrSourceFactInvalid, "source-index file language is required", map[string]any{"file_id": file.ID})
	}
	if file.Roles == nil || !sortedUniqueStrings(file.Roles) {
		return sourceFactError(ErrSourceFactInvalid, "source-index file roles must be sorted and unique", map[string]any{"file_id": file.ID})
	}
	switch file.AnalysisStatus {
	case FileAnalysisComplete, FileAnalysisPartial, FileAnalysisUnparsed, FileAnalysisUnknown:
	default:
		return sourceFactError(ErrSourceFactInvalid, "source-index file analysis status is invalid", map[string]any{"file_id": file.ID, "status": file.AnalysisStatus})
	}
	if file.Size.ByteCount < 0 || file.Size.LineCount < 0 {
		return sourceFactError(ErrSourceFactInvalid, "source-index file size cannot be negative", map[string]any{"file_id": file.ID})
	}
	if err := validateDigest(file.Size.ContentHash, "file content hash"); err != nil {
		return err
	}
	if err := validateProvenance(file.Provenance); err != nil {
		return err
	}
	return validateExtensions(file.Extensions)
}

func validateSymbolRecord(symbol SymbolRecord, fileIDs map[string]struct{}, fileBoundaries map[string]sourceFileBoundary) error {
	if strings.TrimSpace(symbol.ID) == "" || strings.TrimSpace(symbol.Name) == "" || !validSymbolCategory(symbol.Category) {
		return sourceFactError(ErrSourceFactInvalid, "source-index symbol identity or category is invalid", map[string]any{"symbol_id": symbol.ID})
	}
	if len(symbol.Locations) == 0 {
		return sourceFactError(ErrSourceFactInvalid, "source-index symbol must have at least one location", map[string]any{"symbol_id": symbol.ID})
	}
	for _, location := range symbol.Locations {
		if location.Kind == "" {
			return sourceFactError(ErrSourceSpanInvalid, "source-index symbol location kind is required", map[string]any{"symbol_id": symbol.ID})
		}
		if err := validateSpan(location.Span, fileBoundaries); err != nil {
			return err
		}
		if !sortedUniqueStrings(location.SourceReferenceIDs) {
			return sourceFactError(ErrSourceReferenceInvalid, "source-index symbol source references must be sorted and unique", map[string]any{"symbol_id": symbol.ID})
		}
	}
	if symbol.BodySpan != nil {
		if err := validateSpan(*symbol.BodySpan, fileBoundaries); err != nil {
			return err
		}
	}
	if !sortedUniqueStrings(symbol.DocumentationIDs) {
		return sourceFactError(ErrSourceFactInvalid, "source-index symbol documentation ids must be sorted and unique", map[string]any{"symbol_id": symbol.ID})
	}
	if symbol.Visibility.Classification == "" || !validVisibility(symbol.Visibility.Classification) {
		return sourceFactError(ErrSourceFactInvalid, "source-index symbol visibility is invalid", map[string]any{"symbol_id": symbol.ID})
	}
	if err := validateProvenance(symbol.Visibility.Provenance); err != nil {
		return err
	}
	if err := validateProvenance(symbol.Provenance); err != nil {
		return err
	}
	return validateExtensions(symbol.Extensions)
}

func validateDocumentationRecord(documentation DocumentationRecord, fileIDs, symbolIDs map[string]struct{}, fileBoundaries map[string]sourceFileBoundary) error {
	if strings.TrimSpace(documentation.ID) == "" || documentation.SelectionGroup == "" || documentation.Format == "" {
		return sourceFactError(ErrSourceFactInvalid, "source-index documentation identity is incomplete", map[string]any{"documentation_id": documentation.ID})
	}
	if err := validateEntityRef(documentation.SubjectRef, fileIDs, symbolIDs, nil, nil); err != nil {
		return err
	}
	if documentation.SubjectRef.Kind != "file" && documentation.SubjectRef.Kind != "symbol" && documentation.SubjectRef.Kind != "module" {
		return sourceFactError(ErrSourceFactInvalid, "source-index documentation subject kind is unsupported", map[string]any{"documentation_id": documentation.ID})
	}
	for _, span := range documentation.Spans {
		if err := validateSpan(span, fileBoundaries); err != nil {
			return err
		}
	}
	if documentation.Spans == nil {
		return sourceFactError(ErrSourceFactInvalid, "source-index documentation spans must be serialized as an array", map[string]any{"documentation_id": documentation.ID})
	}
	if !sortedUniqueStrings(documentation.SourceReferenceIDs) {
		return sourceFactError(ErrSourceReferenceInvalid, "source-index documentation source references must be sorted and unique", map[string]any{"documentation_id": documentation.ID})
	}
	if documentation.AttachmentBasis == "" || documentation.Status == "" || documentation.Completeness == "" {
		return sourceFactError(ErrDocumentationSelectionInvalid, "source-index documentation selection metadata is incomplete", map[string]any{"documentation_id": documentation.ID})
	}
	if !validDocumentationStatus(documentation.Status) || !validDocumentationCompleteness(documentation.Completeness) {
		return sourceFactError(ErrDocumentationSelectionInvalid, "source-index documentation status or completeness is invalid", map[string]any{"documentation_id": documentation.ID})
	}
	if documentation.PrecedenceRank != nil && *documentation.PrecedenceRank < 0 {
		return sourceFactError(ErrDocumentationSelectionInvalid, "source-index documentation precedence cannot be negative", map[string]any{"documentation_id": documentation.ID})
	}
	if documentation.Status == DocumentationPresent && documentation.RawText == "" && documentation.NormalizedText == "" {
		return sourceFactError(ErrDocumentationSelectionInvalid, "present source-index documentation must retain text", map[string]any{"documentation_id": documentation.ID})
	}
	if err := validateProvenance(documentation.Provenance); err != nil {
		return err
	}
	return validateExtensions(documentation.Extensions)
}

func validateOccurrence(occurrence SymbolOccurrence, fileIDs, symbolIDs map[string]struct{}, fileBoundaries map[string]sourceFileBoundary) error {
	if strings.TrimSpace(occurrence.ID) == "" || occurrence.TargetName == "" || occurrence.OccurrenceKind == "" || occurrence.ResolutionStatus == "" {
		return sourceFactError(ErrSourceFactInvalid, "source-index occurrence is incomplete", map[string]any{"occurrence_id": occurrence.ID})
	}
	if occurrence.SymbolRef != nil {
		if err := validateEntityRef(*occurrence.SymbolRef, fileIDs, symbolIDs, nil, nil); err != nil {
			return err
		}
		if occurrence.SymbolRef.Kind != "symbol" {
			return sourceFactError(ErrSourceFactInvalid, "source-index occurrence symbol reference must target a symbol", map[string]any{"occurrence_id": occurrence.ID})
		}
	}
	if err := validateSpan(occurrence.SourceSpan, fileBoundaries); err != nil {
		return err
	}
	if err := validateProvenance(occurrence.Provenance); err != nil {
		return err
	}
	return validateExtensions(occurrence.Extensions)
}

func validateRelation(relation CodeRelation, fileIDs, symbolIDs, documentationIDs, occurrenceIDs map[string]struct{}, fileBoundaries map[string]sourceFileBoundary) error {
	if strings.TrimSpace(relation.ID) == "" || relation.Category == "" {
		return sourceFactError(ErrSourceRelationInvalid, "source-index relation identity is incomplete", map[string]any{"relation_id": relation.ID})
	}
	if err := validateEntityRef(relation.FromRef, fileIDs, symbolIDs, documentationIDs, occurrenceIDs); err != nil {
		return err
	}
	if (relation.ToRef == nil) == (relation.UnresolvedTarget == nil) {
		return sourceFactError(ErrSourceRelationInvalid, "source-index relation must have exactly one target", map[string]any{"relation_id": relation.ID})
	}
	if relation.ToRef != nil {
		if err := validateEntityRef(*relation.ToRef, fileIDs, symbolIDs, documentationIDs, occurrenceIDs); err != nil {
			return err
		}
	}
	if relation.UnresolvedTarget != nil && relation.UnresolvedTarget.DisplayName == "" {
		return sourceFactError(ErrSourceRelationInvalid, "unresolved source-index relation target requires a display name", map[string]any{"relation_id": relation.ID})
	}
	for _, span := range relation.EvidenceSpans {
		if err := validateSpan(span, fileBoundaries); err != nil {
			return err
		}
	}
	if relation.EvidenceSpans == nil {
		return sourceFactError(ErrSourceRelationInvalid, "source-index relation evidence spans must be serialized as an array", map[string]any{"relation_id": relation.ID})
	}
	if err := validateProvenance(relation.Provenance); err != nil {
		return err
	}
	return validateExtensions(relation.Extensions)
}

func validateMetric(metric MetricFact, fileIDs, symbolIDs, documentationIDs, occurrenceIDs map[string]struct{}) error {
	if strings.TrimSpace(metric.ID) == "" || metric.MetricID == "" || metric.FormulaID == "" || metric.FormulaVersion == "" {
		return sourceFactError(ErrMetricFactInvalid, "source-index metric metadata is incomplete", map[string]any{"metric_id": metric.ID})
	}
	if err := validateEntityRef(metric.SubjectRef, fileIDs, symbolIDs, documentationIDs, occurrenceIDs); err != nil {
		return err
	}
	if metric.Value.Kind == "" || metric.Value.Value == nil || !validMetricValue(metric.Value.Kind, metric.Value.Value) {
		return sourceFactError(ErrMetricFactInvalid, "source-index metric value is not a typed finite value", map[string]any{"metric_id": metric.ID})
	}
	if err := validateProvenance(metric.Provenance); err != nil {
		return err
	}
	return validateExtensions(metric.Extensions)
}

func validateEntityRef(ref EntityRef, fileIDs, symbolIDs, documentationIDs, occurrenceIDs map[string]struct{}) error {
	if strings.TrimSpace(ref.Kind) == "" || strings.TrimSpace(ref.ID) == "" {
		return sourceFactError(ErrSourceReferenceInvalid, "source-index entity reference is incomplete", nil)
	}
	var known map[string]struct{}
	switch ref.Kind {
	case "file":
		known = fileIDs
	case "symbol":
		known = symbolIDs
	case "documentation":
		known = documentationIDs
	case "occurrence":
		known = occurrenceIDs
	case "module", "external", "metric":
		return nil
	default:
		// Open entity kinds may be supplied by extensions. Their local
		// endpoint cannot be validated without a corresponding v1 record.
		return nil
	}
	if known == nil {
		return nil
	}
	if _, exists := known[ref.ID]; !exists {
		return sourceFactError(ErrSourceReferenceInvalid, "source-index entity reference points to an unknown record", map[string]any{"kind": ref.Kind, "id": ref.ID})
	}
	return nil
}

type sourceFileBoundary struct {
	hash      ContentDigest
	byteCount int
}

func validateSpan(span SourceSpan, fileBoundaries map[string]sourceFileBoundary) error {
	if span.FileID == "" {
		return sourceFactError(ErrSourceSpanInvalid, "source-index span requires a file id", nil)
	}
	fileBoundary, exists := fileBoundaries[span.FileID]
	if !exists {
		return sourceFactError(ErrSourceSpanInvalid, "source-index span references an unknown file", map[string]any{"file_id": span.FileID})
	}
	if span.CoordinateSystem != SourceSpanCoordinateSystem || span.Start.ByteOffset < 0 || span.End.ByteOffset < span.Start.ByteOffset || span.Start.Line < 1 || span.End.Line < 1 || span.Start.Column < 1 || span.End.Column < 1 {
		return sourceFactError(ErrSourceSpanInvalid, "source-index span coordinates are invalid", map[string]any{"file_id": span.FileID})
	}
	if span.End.ByteOffset > fileBoundary.byteCount {
		return sourceFactError(ErrSourceSpanInvalid, "source-index span exceeds its file byte boundary", map[string]any{"file_id": span.FileID, "end_byte": span.End.ByteOffset, "byte_count": fileBoundary.byteCount})
	}
	if span.End.Line < span.Start.Line || (span.End.Line == span.Start.Line && span.End.Column < span.Start.Column) {
		return sourceFactError(ErrSourceSpanInvalid, "source-index span end must be after its start", map[string]any{"file_id": span.FileID})
	}
	if err := validateDigest(span.ContentHash, "source span content hash"); err != nil {
		return err
	}
	if span.ContentHash != fileBoundary.hash {
		return sourceFactError(ErrSourceSpanInvalid, "source-index span content hash does not match its file", map[string]any{"file_id": span.FileID})
	}
	return nil
}

func validateProvenance(provenance FactProvenance) error {
	if !validFactStatus(provenance.Status) || strings.TrimSpace(provenance.Basis) == "" || strings.TrimSpace(provenance.Provider) == "" || strings.TrimSpace(provenance.ProviderVersion) == "" {
		return sourceFactError(ErrSourceFactInvalid, "source-index provenance is incomplete or invalid", nil)
	}
	if provenance.EvidenceIDs == nil || !sortedUniqueStrings(provenance.EvidenceIDs) {
		return sourceFactError(ErrSourceFactInvalid, "source-index provenance evidence ids must be sorted and unique", nil)
	}
	if provenance.Score != nil && (math.IsNaN(provenance.Score.Value) || math.IsInf(provenance.Score.Value, 0) || provenance.Score.Value < 0 || provenance.Score.Value > 1 || provenance.Score.ScaleID == "" || provenance.Score.ScaleVersion == "") {
		return sourceFactError(ErrSourceFactInvalid, "source-index provenance score is invalid", nil)
	}
	return nil
}

func validateDigest(digest ContentDigest, label string) error {
	if digest.Algorithm != SourceHashAlgorithm || len(digest.Value) != sha256.Size*2 || strings.ToLower(digest.Value) != digest.Value {
		return sourceFactError(ErrSourceFactInvalid, fmt.Sprintf("%s must be a lowercase SHA-256 digest", label), map[string]any{"algorithm": digest.Algorithm, "value": digest.Value})
	}
	if _, err := hex.DecodeString(digest.Value); err != nil {
		return sourceFactError(ErrSourceFactInvalid, fmt.Sprintf("%s is not valid hexadecimal", label), map[string]any{"value": digest.Value})
	}
	return nil
}

func validateExtensions(values []ExtensionBlock) error {
	if values == nil {
		return sourceFactError(ErrSourceFactInvalid, "source-index extensions must be serialized as an array", nil)
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if strings.TrimSpace(value.Namespace) == "" || strings.TrimSpace(value.SchemaVersion) == "" || strings.TrimSpace(value.Capability) == "" {
			return sourceFactError(ErrSourceFactInvalid, "source-index extension metadata is incomplete", nil)
		}
		key := value.Namespace + "\x00" + value.SchemaVersion + "\x00" + value.Capability
		if _, exists := seen[key]; exists {
			return sourceFactError(ErrSourceFactInvalid, "source-index extension keys must be unique", map[string]any{"namespace": value.Namespace, "capability": value.Capability})
		}
		seen[key] = struct{}{}
	}
	if !sort.SliceIsSorted(values, func(i, j int) bool {
		left := values[i].Namespace + "\x00" + values[i].SchemaVersion + "\x00" + values[i].Capability
		right := values[j].Namespace + "\x00" + values[j].SchemaVersion + "\x00" + values[j].Capability
		return left < right
	}) {
		return sourceFactError(ErrSourceFactInvalid, "source-index extensions must be sorted", nil)
	}
	return nil
}

func validFactStatus(value string) bool {
	switch value {
	case FactStatusObserved, FactStatusAbsent, FactStatusUnknown, FactStatusUnsupported, FactStatusPartial:
		return true
	default:
		return false
	}
}

func validSymbolCategory(value string) bool {
	switch value {
	case SymbolCategoryCallable, SymbolCategoryType, SymbolCategoryNamespace, SymbolCategoryValue, SymbolCategoryMember, SymbolCategoryMacro, SymbolCategoryUnknown:
		return true
	default:
		return false
	}
}

func validVisibility(value string) bool {
	switch value {
	case "public", "protected", "private", "internal", "package", "module", "default", "unknown":
		return true
	default:
		return false
	}
}

func validDocumentationStatus(value string) bool {
	switch value {
	case DocumentationPresent, DocumentationAbsent, DocumentationUnknown, DocumentationUnsupported, DocumentationPartial:
		return true
	default:
		return false
	}
}

func validDocumentationCompleteness(value string) bool {
	switch value {
	case DocumentationComplete, DocumentationSummaryOnly, DocumentationTruncated, DocumentationUnknownSize:
		return true
	default:
		return false
	}
}

func validMetricValue(kind string, value any) bool {
	switch kind {
	case "integer":
		switch typed := value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, json.Number:
			return true
		case float32:
			return !math.IsNaN(float64(typed)) && !math.IsInf(float64(typed), 0) && math.Trunc(float64(typed)) == float64(typed)
		case float64:
			return !math.IsNaN(typed) && !math.IsInf(typed, 0) && math.Trunc(typed) == typed
		default:
			return false
		}
	case "number":
		switch typed := value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return true
		case float32:
			return !math.IsNaN(float64(typed)) && !math.IsInf(float64(typed), 0)
		case float64:
			return !math.IsNaN(typed) && !math.IsInf(typed, 0)
		case json.Number:
			_, err := typed.Float64()
			return err == nil
		default:
			return false
		}
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	default:
		return false
	}
}

func sortedUniqueStrings(values []string) bool {
	if values == nil {
		return true
	}
	for index := 1; index < len(values); index++ {
		if values[index-1] >= values[index] {
			return false
		}
	}
	return true
}

func entityKey(ref EntityRef) string {
	return ref.Kind + "\x00" + ref.ID + "\x00" + ref.ScopeID + "\x00" + ref.SnapshotID
}

func sourceFactError(code ErrorCode, message string, details map[string]any) error {
	return NewHostError(code, message, details)
}
