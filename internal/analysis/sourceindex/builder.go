package sourceindex

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/syntax"
)

const (
	CapabilityFiles         = "source:files"
	CapabilitySize          = "source:size"
	CapabilityDeclarations  = "source:declarations"
	CapabilityDocumentation = "source:documentation"
	CapabilityVisibility    = "source:visibility"
)

// BuildSourceIndex builds one authoritative scope snapshot. File facts are
// assembled before parsing so a parser or extractor failure cannot erase an
// eligible file from the attachment.
func BuildSourceIndex(ctx context.Context, input BuildInput) (analysis.SourceIndex, []analysis.Diagnostic, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	files, err := normalizeInputFiles(input.Files)
	if err != nil {
		return analysis.SourceIndex{}, nil, err
	}
	requested := normalizeRequestedCapabilities(input.RequestedCapabilities)
	if len(requested) == 0 {
		requested = []string{CapabilityFiles, CapabilitySize}
	}
	scope := normalizeScope(input.Scope, files)
	producer := normalizeProducer(input.Producer)

	fileRecords := make([]analysis.FileRecord, 0, len(files))
	fileByPath := make(map[string]analysis.FileRecord, len(files))
	for _, inputFile := range files {
		if _, exists := fileByPath[inputFile.Path]; exists {
			return analysis.SourceIndex{}, nil, analysis.NewHostError(analysis.ErrSourceFileEnumerationFailed, "source-index input contains duplicate file paths", map[string]any{"path": inputFile.Path})
		}
		file := buildFileRecord(scope, producer, inputFile)
		fileRecords = append(fileRecords, file)
		fileByPath[inputFile.Path] = file
	}
	sourceSetDigest := computeSourceSetDigest(fileRecords)
	if input.Scope.SourceScopeFingerprint.Value == "" || input.Scope.SourceScopeFingerprint.Algorithm == "" {
		scope.SourceScopeFingerprint = sourceSetDigest
	}
	if scope.SourceScopeFingerprint.Value == "" {
		scope.SourceScopeFingerprint = sourceSetDigest
	}

	capabilities, extractors, extractorCapabilities, err := resolveCapabilities(input.Extractors, requested, files)
	if err != nil {
		return analysis.SourceIndex{}, nil, err
	}
	producer.Extractors = mergeExtractorIdentities(producer.Extractors, extractors)
	stats := make(map[string]*capabilityStats, len(requested))
	for _, capability := range requested {
		stats[capability] = &capabilityStats{eligible: len(files)}
	}

	allSymbols := make([]analysis.SymbolRecord, 0)
	allDocumentation := make([]analysis.DocumentationRecord, 0)
	allOccurrences := make([]analysis.SymbolOccurrence, 0)
	allRelations := make([]analysis.CodeRelation, 0)
	allMetrics := make([]analysis.MetricFact, 0)
	diagnostics := make([]analysis.Diagnostic, 0)

	for fileIndex := range fileRecords {
		if err := ctx.Err(); err != nil {
			return analysis.SourceIndex{}, diagnostics, err
		}
		file := &fileRecords[fileIndex]
		inputFile := files[fileIndex]
		language := languageToken(file.Language.ID)
		fileExtractors := extractorsForLanguage(extractors, extractorCapabilities, language, requested)
		if len(fileExtractors) == 0 {
			markUnsupportedCapabilities(stats, requested, language, input.Extractors)
			continue
		}
		if input.SyntaxProvider == nil {
			markExtractionUnavailable(file, stats, fileExtractors, requested)
			diagnostics = append(diagnostics, sourceDiagnostic("source_index_syntax_unavailable", "source syntax provider is unavailable; file facts were retained without extracted declarations", inputFile.Path, "source-index", "warning"))
			continue
		}
		parsed, parseErr := input.SyntaxProvider.Parse(ctx, syntax.Source{Path: file.Path, Content: inputFile.Content})
		if parseErr != nil {
			parsed.Close()
			file.AnalysisStatus = analysis.FileAnalysisUnparsed
			for _, extractor := range fileExtractors {
				markExtractorFailure(stats, extractor, requested, "syntax provider failed")
			}
			diagnostics = append(diagnostics, sourceDiagnostic("source_index_parse_failed", fmt.Sprintf("source syntax parsing failed: %v", parseErr), inputFile.Path, "source-index", "warning"))
			continue
		}
		for _, issue := range parsed.Issues {
			file.AnalysisStatus = analysis.FileAnalysisPartial
			diagnostics = append(diagnostics, syntaxIssueDiagnostic(inputFile.Path, issue))
		}
		if parsed.Tree == nil || parsed.Tree.Root() == nil {
			parsed.Close()
			file.AnalysisStatus = analysis.FileAnalysisUnparsed
			for _, extractor := range fileExtractors {
				markExtractorFailure(stats, extractor, requested, "syntax provider returned no tree")
			}
			diagnostics = append(diagnostics, sourceDiagnostic("source_index_parse_failed", "source syntax provider returned no syntax tree", inputFile.Path, "source-index", "warning"))
			continue
		}
		root := parsed.Tree.Root()
		for _, extractor := range fileExtractors {
			batch, extractErr := extractor.Extract(SourceFactInput{
				File:                  *file,
				Content:               inputFile.Content,
				Tree:                  root,
				Scope:                 scope,
				RequestedCapabilities: requested,
			})
			if extractErr != nil {
				file.AnalysisStatus = analysis.FileAnalysisPartial
				markExtractorFailure(stats, extractor, requested, extractErr.Error())
				diagnostics = append(diagnostics, sourceDiagnostic("source_index_extraction_failed", fmt.Sprintf("source-fact extraction failed: %v", extractErr), inputFile.Path, extractor.ID(), "warning"))
				continue
			}
			lengths := [5]int{len(allSymbols), len(allDocumentation), len(allOccurrences), len(allRelations), len(allMetrics)}
			appendErr := appendBatch(&allSymbols, &allDocumentation, &allOccurrences, &allRelations, &allMetrics, batch, *file, extractor)
			if appendErr == nil {
				appendErr = analysis.ValidateSourceFactCollections(fileRecords, allSymbols, allDocumentation, allOccurrences, allRelations, allMetrics)
			}
			if appendErr != nil {
				allSymbols = allSymbols[:lengths[0]]
				allDocumentation = allDocumentation[:lengths[1]]
				allOccurrences = allOccurrences[:lengths[2]]
				allRelations = allRelations[:lengths[3]]
				allMetrics = allMetrics[:lengths[4]]
				file.AnalysisStatus = analysis.FileAnalysisPartial
				markExtractorFailure(stats, extractor, requested, appendErr.Error())
				diagnostics = append(diagnostics, sourceDiagnostic("source_index_fact_invalid", appendErr.Error(), inputFile.Path, extractor.ID(), "warning"))
				continue
			}
			markExtractorObserved(stats, extractor, requested)
		}
		parsed.Close()
	}

	allRelations = append(allRelations, moduleFileRelations(files, fileRecords, scope, producer)...)
	normalizeDocumentationLinks(allSymbols, allDocumentation)
	coverage := buildCoverage(requested, capabilities, stats, fileRecords, producer)
	snapshot := analysis.SourceIndexSnapshot{
		SnapshotKind:  analysis.SourceIndexSnapshotScope,
		ScopeContext:  scope,
		Producer:      producer,
		Input:         analysis.InputContext{EligibleFileCount: len(fileRecords), SourceSetDigest: sourceSetDigest, RequestedCapabilities: requested},
		Capabilities:  capabilities,
		Coverage:      coverage,
		Files:         fileRecords,
		Symbols:       allSymbols,
		Documentation: allDocumentation,
		Occurrences:   allOccurrences,
		Relations:     allRelations,
		Metrics:       allMetrics,
		Extensions:    []analysis.ExtensionBlock{},
	}
	extractorIdentityJSON, _ := json.Marshal(producer.Extractors)
	scopeJSON, _ := json.Marshal(scope)
	snapshot.SnapshotID = opaqueID("snapshot", scope.ScopeID, producer.AnalyzerID, producer.AnalyzerVersion, producer.ProtocolVersion, sourceSetDigest.Value, string(scopeJSON), string(extractorIdentityJSON), strings.Join(requested, "\x00"))
	canonicalizeSnapshot(&snapshot)
	snapshot.SnapshotDigest = analysis.ComputeSourceIndexDigest(snapshot)
	index := analysis.SourceIndex{
		SchemaVersion: analysis.SourceIndexSchemaVersion,
		Snapshots:     []analysis.SourceIndexSnapshot{snapshot},
		Extensions:    []analysis.ExtensionBlock{},
	}
	if err := analysis.ValidateSourceIndex(index); err != nil {
		return analysis.SourceIndex{}, diagnostics, err
	}
	return index, diagnostics, nil
}

type capabilityStats struct {
	eligible    int
	observed    int
	failures    int
	unsupported bool
	reason      string
}

type normalizedInputFile struct {
	Path               string
	Content            []byte
	Language           analysis.LanguageRef
	Roles              []string
	AnalysisStatus     string
	Provenance         analysis.FactProvenance
	ModuleID           string
	SourceReferenceIDs []string
}

func normalizeInputFiles(values []SourceFileInput) ([]normalizedInputFile, error) {
	result := make([]normalizedInputFile, 0, len(values))
	for _, value := range values {
		filePath := normalizeRelativePath(value.Path)
		if filePath == "" || filePath == "." || !analysisPathSafe(filePath) {
			return nil, analysis.NewHostError(analysis.ErrSourceFileEnumerationFailed, "source-index file path must be repository-relative", map[string]any{"path": value.Path})
		}
		language := value.Language
		language.ID = strings.TrimSpace(language.ID)
		if language.ID == "" {
			return nil, analysis.NewHostError(analysis.ErrSourceFileEnumerationFailed, "source-index file language is required", map[string]any{"path": filePath})
		}
		roles := append([]string(nil), value.Roles...)
		if len(roles) == 0 {
			roles = []string{"role:source"}
		}
		roles = normalizeStrings(roles)
		status := value.AnalysisStatus
		if status == "" {
			status = analysis.FileAnalysisComplete
		}
		if status != analysis.FileAnalysisComplete && status != analysis.FileAnalysisPartial && status != analysis.FileAnalysisUnparsed && status != analysis.FileAnalysisUnknown {
			return nil, analysis.NewHostError(analysis.ErrSourceFileEnumerationFailed, "source-index file analysis status is invalid", map[string]any{"path": filePath, "status": status})
		}
		content := append([]byte(nil), value.Content...)
		result = append(result, normalizedInputFile{
			Path:               filePath,
			Content:            content,
			Language:           language,
			Roles:              roles,
			AnalysisStatus:     status,
			Provenance:         value.Provenance,
			ModuleID:           value.ModuleID,
			SourceReferenceIDs: normalizeStrings(value.SourceReferenceIDs),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result, nil
}

func normalizeScope(scope analysis.ScopeContext, files []normalizedInputFile) analysis.ScopeContext {
	if scope.ProjectRoot == "" {
		scope.ProjectRoot = "."
	} else {
		scope.ProjectRoot = normalizeRelativePath(scope.ProjectRoot)
	}
	if scope.ScopeID == "" {
		scope.ScopeID = opaqueID("scope", scope.ProjectRoot)
	}
	if scope.Mode == "" {
		scope.Mode = analysis.SourceIndexScopeMode
	}
	if scope.SourceScopeFingerprint.Algorithm == "" || scope.SourceScopeFingerprint.Value == "" {
		paths := make([]string, 0, len(files))
		for _, file := range files {
			paths = append(paths, file.Path)
		}
		scope.SourceScopeFingerprint = digestStrings(paths)
	}
	return scope
}

func normalizeProducer(producer analysis.ProducerContext) analysis.ProducerContext {
	if producer.Extractors == nil {
		producer.Extractors = []analysis.ExtractorIdentity{}
	}
	return producer
}

func buildFileRecord(scope analysis.ScopeContext, producer analysis.ProducerContext, input normalizedInputFile) analysis.FileRecord {
	hash := sha256.Sum256(input.Content)
	digest := analysis.ContentDigest{Algorithm: analysis.SourceHashAlgorithm, Value: hex.EncodeToString(hash[:])}
	status := input.AnalysisStatus
	provenance := normalizeProvenance(input.Provenance, fileFactStatus(status), producer.AnalyzerID, producer.AnalyzerVersion)
	return analysis.FileRecord{
		ID:             opaqueID("file", scope.ScopeID, input.Path, input.Language.ID, digest.Value),
		Path:           input.Path,
		Language:       input.Language,
		Roles:          input.Roles,
		Size:           analysis.FileSize{LineCount: physicalLineCount(input.Content), ByteCount: len(input.Content), ContentHash: digest},
		AnalysisStatus: status,
		Provenance:     provenance,
		Extensions:     []analysis.ExtensionBlock{},
	}
}

func resolveCapabilities(registry *Registry, requested []string, files []normalizedInputFile) ([]analysis.CapabilityDescriptor, []SourceFactExtractor, map[string][]analysis.CapabilityDescriptor, error) {
	capabilityMap := map[string]analysis.CapabilityDescriptor{
		CapabilityFiles: {ID: CapabilityFiles, Version: "v1", Description: "Eligible source file facts."},
		CapabilitySize:  {ID: CapabilitySize, Version: "v1", Description: "Raw byte, physical line, and content hash facts."},
	}
	allExtractors := []SourceFactExtractor{}
	byExtractor := make(map[string][]analysis.CapabilityDescriptor)
	if registry != nil {
		owners := make(map[string]string)
		for _, extractor := range registry.List() {
			languageMatch := false
			for _, file := range files {
				for _, descriptor := range extractor.Capabilities() {
					if containsString(requested, descriptor.ID) && languageSupported(descriptor, languageToken(file.Language.ID)) {
						ownerKey := languageToken(file.Language.ID) + "\x00" + descriptor.ID
						identity := extractorKey(extractor.ID(), extractor.Version())
						if previous, exists := owners[ownerKey]; exists && previous != identity {
							return nil, nil, nil, analysis.NewHostError(analysis.ErrSourceExtractorFailed, "multiple source-fact extractors claim the same language capability", map[string]any{"language": languageToken(file.Language.ID), "capability": descriptor.ID, "first": previous, "second": identity})
						}
						owners[ownerKey] = identity
						languageMatch = true
						capabilityMap[descriptor.ID] = descriptor
						key := extractorKey(extractor.ID(), extractor.Version())
						byExtractor[key] = append(byExtractor[key], descriptor)
					}
				}
			}
			if languageMatch {
				allExtractors = append(allExtractors, extractor)
			}
		}
	}
	for _, capability := range requested {
		if _, exists := capabilityMap[capability]; !exists {
			capabilityMap[capability] = analysis.CapabilityDescriptor{ID: capability, Version: "v1"}
		}
	}
	capabilities := make([]analysis.CapabilityDescriptor, 0, len(capabilityMap))
	for _, capability := range capabilityMap {
		capabilities = append(capabilities, capability)
	}
	sort.Slice(capabilities, func(i, j int) bool {
		if capabilities[i].ID == capabilities[j].ID {
			return capabilities[i].Version < capabilities[j].Version
		}
		return capabilities[i].ID < capabilities[j].ID
	})
	for key := range byExtractor {
		byExtractor[key] = uniqueCapabilities(byExtractor[key])
	}
	sort.Slice(allExtractors, func(i, j int) bool {
		if allExtractors[i].ID() == allExtractors[j].ID() {
			return allExtractors[i].Version() < allExtractors[j].Version()
		}
		return allExtractors[i].ID() < allExtractors[j].ID()
	})
	return capabilities, allExtractors, byExtractor, nil
}

func extractorsForLanguage(extractors []SourceFactExtractor, extractorCapabilities map[string][]analysis.CapabilityDescriptor, language string, requested []string) []SourceFactExtractor {
	result := make([]SourceFactExtractor, 0)
	for _, extractor := range extractors {
		for _, descriptor := range extractorCapabilities[extractorKey(extractor.ID(), extractor.Version())] {
			if containsString(requested, descriptor.ID) && languageSupported(descriptor, language) {
				result = append(result, extractor)
				break
			}
		}
	}
	return result
}

func appendBatch(symbols *[]analysis.SymbolRecord, documentation *[]analysis.DocumentationRecord, occurrences *[]analysis.SymbolOccurrence, relations *[]analysis.CodeRelation, metrics *[]analysis.MetricFact, batch FactBatch, file analysis.FileRecord, extractor SourceFactExtractor) error {
	symbolsByID := make(map[string]string)
	for index, symbol := range batch.Symbols {
		if symbol.Name == "" || symbol.Category == "" {
			return fmt.Errorf("source-fact extractor %q returned an incomplete symbol at index %d", extractor.ID(), index)
		}
		if symbol.StableKey == "" {
			symbol.StableKey = fmt.Sprintf("%s:%s:%d", symbol.Category, symbol.Name, index)
		}
		oldID := symbol.ID
		if oldID == "" {
			oldID = symbol.StableKey
		}
		symbol.ID = opaqueID("symbol", file.ID, extractor.ID(), symbol.StableKey)
		symbol.Locations = normalizeLocations(symbol.Locations, file)
		if len(symbol.Locations) == 0 {
			return fmt.Errorf("source-fact extractor %q returned symbol %q without a valid location", extractor.ID(), symbol.Name)
		}
		symbol.Visibility = normalizeVisibility(symbol.Visibility, symbol.Provenance, extractor)
		symbol.Provenance = normalizeProvenance(symbol.Provenance, analysis.FactStatusObserved, extractor.ID(), extractor.Version())
		symbol.DocumentationIDs = nil
		symbol.Extensions = normalizeExtensions(symbol.Extensions)
		if oldID != "" {
			symbolsByID[oldID] = symbol.ID
		}
		if symbol.StableKey != "" {
			symbolsByID[symbol.StableKey] = symbol.ID
		}
		*symbols = append(*symbols, symbol)
	}
	for index, documentationValue := range batch.Documentation {
		documentationValue.SubjectRef = resolveBatchRef(documentationValue.SubjectRef, file, symbolsByID)
		if documentationValue.SubjectRef.ID == "" {
			return fmt.Errorf("source-fact extractor %q returned documentation with an unresolved subject at index %d", extractor.ID(), index)
		}
		oldID := documentationValue.ID
		if oldID == "" {
			oldID = fmt.Sprintf("documentation:%d", index)
		}
		documentationValue.ID = opaqueID("documentation", file.ID, extractor.ID(), oldID, documentationValue.SelectionGroup, documentationValue.Format, documentationValue.RawText, documentationValue.NormalizedText)
		documentationValue.Spans = normalizeSpans(documentationValue.Spans, file)
		documentationValue.SourceReferenceIDs = normalizeStrings(documentationValue.SourceReferenceIDs)
		documentationValue.Provenance = normalizeProvenance(documentationValue.Provenance, documentationFactStatus(documentationValue.Status), extractor.ID(), extractor.Version())
		documentationValue.Extensions = normalizeExtensions(documentationValue.Extensions)
		*documentation = append(*documentation, documentationValue)
	}
	for index, occurrence := range batch.Occurrences {
		if occurrence.SymbolRef != nil {
			resolved := resolveBatchRef(*occurrence.SymbolRef, file, symbolsByID)
			occurrence.SymbolRef = &resolved
		}
		occurrence.ID = opaqueID("occurrence", file.ID, extractor.ID(), occurrence.ID, fmt.Sprint(index))
		occurrence.SourceSpan = normalizeSpan(occurrence.SourceSpan, file)
		occurrence.Provenance = normalizeProvenance(occurrence.Provenance, analysis.FactStatusObserved, extractor.ID(), extractor.Version())
		occurrence.Extensions = normalizeExtensions(occurrence.Extensions)
		*occurrences = append(*occurrences, occurrence)
	}
	for index, relation := range batch.Relations {
		relation.FromRef = resolveBatchRef(relation.FromRef, file, symbolsByID)
		if relation.ToRef != nil {
			resolved := resolveBatchRef(*relation.ToRef, file, symbolsByID)
			relation.ToRef = &resolved
		}
		relation.ID = opaqueID("relation", file.ID, extractor.ID(), relation.ID, fmt.Sprint(index))
		relation.EvidenceSpans = normalizeSpans(relation.EvidenceSpans, file)
		relation.Provenance = normalizeProvenance(relation.Provenance, analysis.FactStatusObserved, extractor.ID(), extractor.Version())
		relation.Extensions = normalizeExtensions(relation.Extensions)
		*relations = append(*relations, relation)
	}
	for index, metric := range batch.Metrics {
		metric.SubjectRef = resolveBatchRef(metric.SubjectRef, file, symbolsByID)
		metric.ID = opaqueID("metric", file.ID, extractor.ID(), metric.ID, fmt.Sprint(index))
		metric.Provenance = normalizeProvenance(metric.Provenance, analysis.FactStatusObserved, extractor.ID(), extractor.Version())
		metric.Extensions = normalizeExtensions(metric.Extensions)
		*metrics = append(*metrics, metric)
	}
	return nil
}

func moduleFileRelations(inputs []normalizedInputFile, files []analysis.FileRecord, scope analysis.ScopeContext, producer analysis.ProducerContext) []analysis.CodeRelation {
	result := make([]analysis.CodeRelation, 0)
	for index, input := range inputs {
		if input.ModuleID == "" {
			continue
		}
		span := fullFileSpan(files[index], input.Content)
		result = append(result, analysis.CodeRelation{
			ID:            opaqueID("relation", scope.ScopeID, "module-file", input.ModuleID, files[index].ID),
			Category:      analysis.RelationContains,
			LanguageKind:  "relation:module-file",
			FromRef:       analysis.EntityRef{Kind: "module", ID: input.ModuleID},
			ToRef:         entityRefPointer(analysis.EntityRef{Kind: "file", ID: files[index].ID}),
			EvidenceSpans: []analysis.SourceSpan{span},
			Provenance:    normalizeProvenance(analysis.FactProvenance{}, analysis.FactStatusObserved, producer.AnalyzerID, producer.AnalyzerVersion),
			Extensions:    []analysis.ExtensionBlock{},
		})
	}
	return result
}

func buildCoverage(requested []string, capabilities []analysis.CapabilityDescriptor, stats map[string]*capabilityStats, files []analysis.FileRecord, producer analysis.ProducerContext) []analysis.CoverageRecord {
	descriptorByID := make(map[string]analysis.CapabilityDescriptor, len(capabilities))
	for _, descriptor := range capabilities {
		descriptorByID[descriptor.ID] = descriptor
	}
	result := make([]analysis.CoverageRecord, 0, len(requested))
	for _, capability := range requested {
		state := stats[capability]
		if state == nil {
			state = &capabilityStats{eligible: len(files)}
		}
		status := analysis.FactStatusObserved
		reason := ""
		switch {
		case state.failures > 0:
			status = analysis.FactStatusPartial
			reason = state.reason
		case state.observed > 0 && state.observed < state.eligible:
			status = analysis.FactStatusPartial
			reason = "capability was observed for only part of the eligible source set"
		case state.observed == 0 && state.unsupported:
			status = analysis.FactStatusUnsupported
			reason = state.reason
		}
		observed := state.observed
		eligible := state.eligible
		if capability == CapabilityFiles || capability == CapabilitySize {
			eligible = len(files)
			observed = len(files)
			status = analysis.FactStatusObserved
		}
		provider := producer.AnalyzerID
		version := producer.AnalyzerVersion
		if descriptor := descriptorByID[capability]; descriptor.ID != "" {
			if descriptor.Version != "" {
				version = descriptor.Version
			}
		}
		result = append(result, analysis.CoverageRecord{
			Capability:    capability,
			SubjectKind:   capabilitySubject(capability),
			Status:        status,
			EligibleCount: intPointer(eligible),
			ObservedCount: intPointer(observed),
			Reason:        reason,
			Provenance:    normalizeProvenance(analysis.FactProvenance{}, status, provider, version),
		})
	}
	return result
}

func canonicalizeSnapshot(snapshot *analysis.SourceIndexSnapshot) {
	for index := range snapshot.Files {
		snapshot.Files[index].Roles = normalizeStrings(snapshot.Files[index].Roles)
		snapshot.Files[index].Extensions = normalizeExtensions(snapshot.Files[index].Extensions)
	}
	for index := range snapshot.Symbols {
		snapshot.Symbols[index].DocumentationIDs = normalizeStrings(snapshot.Symbols[index].DocumentationIDs)
		snapshot.Symbols[index].Extensions = normalizeExtensions(snapshot.Symbols[index].Extensions)
		sort.Slice(snapshot.Symbols[index].Locations, func(left, right int) bool {
			return spanKey(snapshot.Symbols[index].Locations[left].Span) < spanKey(snapshot.Symbols[index].Locations[right].Span)
		})
	}
	for index := range snapshot.Documentation {
		snapshot.Documentation[index].SourceReferenceIDs = normalizeStrings(snapshot.Documentation[index].SourceReferenceIDs)
		snapshot.Documentation[index].Extensions = normalizeExtensions(snapshot.Documentation[index].Extensions)
	}
	for index := range snapshot.Occurrences {
		snapshot.Occurrences[index].Extensions = normalizeExtensions(snapshot.Occurrences[index].Extensions)
	}
	for index := range snapshot.Relations {
		snapshot.Relations[index].Extensions = normalizeExtensions(snapshot.Relations[index].Extensions)
	}
	for index := range snapshot.Metrics {
		snapshot.Metrics[index].Extensions = normalizeExtensions(snapshot.Metrics[index].Extensions)
	}
	sort.Slice(snapshot.Capabilities, func(i, j int) bool { return snapshot.Capabilities[i].ID < snapshot.Capabilities[j].ID })
	sort.Slice(snapshot.Coverage, func(i, j int) bool {
		if snapshot.Coverage[i].Capability == snapshot.Coverage[j].Capability {
			return snapshot.Coverage[i].SubjectKind < snapshot.Coverage[j].SubjectKind
		}
		return snapshot.Coverage[i].Capability < snapshot.Coverage[j].Capability
	})
	analysis.CanonicalizeSourceIndexFacts(snapshot)
	for index := range snapshot.Symbols {
		snapshot.Symbols[index].DocumentationIDs = documentationIDsForSymbol(snapshot.Symbols[index], snapshot.Documentation)
	}
}

func normalizeDocumentationLinks(symbols []analysis.SymbolRecord, documentation []analysis.DocumentationRecord) {
	bySubject := make(map[string][]string)
	for _, record := range documentation {
		key := sourceEntityKey(record.SubjectRef)
		bySubject[key] = append(bySubject[key], record.ID)
	}
	for index := range symbols {
		key := sourceEntityKey(analysis.EntityRef{Kind: "symbol", ID: symbols[index].ID})
		symbols[index].DocumentationIDs = normalizeStrings(bySubject[key])
	}
}

func documentationIDsForSymbol(symbol analysis.SymbolRecord, documentation []analysis.DocumentationRecord) []string {
	ids := make([]string, 0)
	for _, record := range documentation {
		if record.SubjectRef.Kind == "symbol" && record.SubjectRef.ID == symbol.ID {
			ids = append(ids, record.ID)
		}
	}
	return normalizeStrings(ids)
}

func normalizeLocations(values []analysis.SymbolLocation, file analysis.FileRecord) []analysis.SymbolLocation {
	result := make([]analysis.SymbolLocation, 0, len(values))
	for _, value := range values {
		value.Span = normalizeSpan(value.Span, file)
		value.SourceReferenceIDs = normalizeStrings(value.SourceReferenceIDs)
		result = append(result, value)
	}
	return result
}

func normalizeSpans(values []analysis.SourceSpan, file analysis.FileRecord) []analysis.SourceSpan {
	result := make([]analysis.SourceSpan, 0, len(values))
	for _, value := range values {
		result = append(result, normalizeSpan(value, file))
	}
	return result
}

func normalizeSpan(value analysis.SourceSpan, file analysis.FileRecord) analysis.SourceSpan {
	if value.FileID == "" {
		value.FileID = file.ID
	}
	if value.CoordinateSystem == "" {
		value.CoordinateSystem = analysis.SourceSpanCoordinateSystem
	}
	if value.ContentHash.Value == "" {
		value.ContentHash = file.Size.ContentHash
	}
	return value
}

func normalizeVisibility(value analysis.VisibilityFact, symbolProvenance analysis.FactProvenance, extractor SourceFactExtractor) analysis.VisibilityFact {
	if value.Classification == "" {
		value.Classification = "unknown"
	}
	if value.Provenance.Status == "" {
		value.Provenance = normalizeProvenance(symbolProvenance, analysis.FactStatusObserved, extractor.ID(), extractor.Version())
	}
	return value
}

func normalizeProvenance(value analysis.FactProvenance, fallbackStatus, provider, version string) analysis.FactProvenance {
	if value.Status == "" {
		value.Status = fallbackStatus
	}
	if value.Basis == "" {
		value.Basis = "syntax"
	}
	if value.EvidenceIDs == nil {
		value.EvidenceIDs = []string{}
	}
	value.EvidenceIDs = normalizeStrings(value.EvidenceIDs)
	if value.Provider == "" {
		value.Provider = provider
	}
	if value.ProviderVersion == "" {
		value.ProviderVersion = version
	}
	return value
}

func normalizeExtensions(values []analysis.ExtensionBlock) []analysis.ExtensionBlock {
	if values == nil {
		return []analysis.ExtensionBlock{}
	}
	result := make([]analysis.ExtensionBlock, len(values))
	copy(result, values)
	sort.Slice(result, func(i, j int) bool {
		left := result[i].Namespace + "\x00" + result[i].SchemaVersion + "\x00" + result[i].Capability
		right := result[j].Namespace + "\x00" + result[j].SchemaVersion + "\x00" + result[j].Capability
		return left < right
	})
	return result
}

func resolveBatchRef(value analysis.EntityRef, file analysis.FileRecord, symbols map[string]string) analysis.EntityRef {
	if value.Kind == "file" && (value.ID == "" || value.ID == file.Path || value.ID == file.ID) {
		value.ID = file.ID
	}
	if value.Kind == "symbol" {
		if resolved, ok := symbols[value.ID]; ok {
			value.ID = resolved
		}
	}
	return value
}

func computeSourceSetDigest(files []analysis.FileRecord) analysis.ContentDigest {
	entries := make([]struct {
		Path string `json:"path"`
		Hash string `json:"hash"`
	}, 0, len(files))
	for _, file := range files {
		entries = append(entries, struct {
			Path string `json:"path"`
			Hash string `json:"hash"`
		}{Path: file.Path, Hash: file.Size.ContentHash.Value})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return digestJSON(entries)
}

func digestStrings(values []string) analysis.ContentDigest {
	copyValues := append([]string(nil), values...)
	sort.Strings(copyValues)
	return digestJSON(copyValues)
}

func digestJSON(value any) analysis.ContentDigest {
	data, err := json.Marshal(value)
	if err != nil {
		return analysis.ContentDigest{Algorithm: analysis.SourceHashAlgorithm}
	}
	hash := sha256.Sum256(data)
	return analysis.ContentDigest{Algorithm: analysis.SourceHashAlgorithm, Value: hex.EncodeToString(hash[:])}
}

func physicalLineCount(content []byte) int {
	if len(content) == 0 {
		return 0
	}
	count := 0
	for index := 0; index < len(content); index++ {
		switch content[index] {
		case '\n':
			count++
		case '\r':
			if index+1 < len(content) && content[index+1] == '\n' {
				index++
			}
			count++
		}
	}
	if last := content[len(content)-1]; last != '\n' && last != '\r' {
		count++
	}
	return count
}

func fullFileSpan(file analysis.FileRecord, content []byte) analysis.SourceSpan {
	return analysis.SourceSpan{
		FileID:           file.ID,
		Start:            analysis.SpanPosition{ByteOffset: 0, Line: 1, Column: 1},
		End:              sourceEndPosition(content),
		CoordinateSystem: analysis.SourceSpanCoordinateSystem,
		ContentHash:      file.Size.ContentHash,
	}
}

func sourceEndPosition(content []byte) analysis.SpanPosition {
	position := analysis.SpanPosition{ByteOffset: len(content), Line: 1, Column: 1}
	for index := 0; index < len(content); index++ {
		switch content[index] {
		case '\r':
			if index+1 < len(content) && content[index+1] == '\n' {
				index++
			}
			position.Line++
			position.Column = 1
		case '\n':
			position.Line++
			position.Column = 1
		default:
			position.Column++
		}
	}
	return position
}

func spanKey(span analysis.SourceSpan) string {
	return fmt.Sprintf("%s:%d:%d:%d", span.FileID, span.Start.ByteOffset, span.End.ByteOffset, span.Start.Line)
}

func syntaxIssueDiagnostic(filePath string, issue syntax.Issue) analysis.Diagnostic {
	return analysis.Diagnostic{
		Code:        "source_index_parse_issue",
		Severity:    "warning",
		Message:     issue.Message,
		Path:        filePath,
		Location:    &analysis.Position{Line: int(issue.Range.Start.Row) + 1, Column: int(issue.Range.Start.Column) + 1},
		Recoverable: true,
	}
}

func sourceDiagnostic(code, message, filePath, subject, severity string) analysis.Diagnostic {
	return analysis.Diagnostic{Code: code, Severity: severity, Message: message, Path: filePath, Subject: subject, Location: &analysis.Position{Line: 1, Column: 1}, Recoverable: true}
}

func markUnsupportedCapabilities(stats map[string]*capabilityStats, requested []string, language string, registry *Registry) {
	for _, capability := range requested {
		if capability == CapabilityFiles || capability == CapabilitySize {
			continue
		}
		if stats[capability] == nil {
			stats[capability] = &capabilityStats{}
		}
		stats[capability].unsupported = true
		if registry == nil {
			stats[capability].reason = "no source-fact extractor registry is configured"
		} else {
			stats[capability].reason = "no registered extractor supports " + capability + " for " + language
		}
	}
}

func markExtractionUnavailable(file *analysis.FileRecord, stats map[string]*capabilityStats, extractors []SourceFactExtractor, requested []string) {
	file.AnalysisStatus = analysis.FileAnalysisUnknown
	for _, extractor := range extractors {
		markExtractorFailure(stats, extractor, requested, "source syntax provider is unavailable")
	}
}

func markExtractorObserved(stats map[string]*capabilityStats, extractor SourceFactExtractor, requested []string) {
	for _, descriptor := range extractor.Capabilities() {
		if !containsString(requested, descriptor.ID) {
			continue
		}
		if stats[descriptor.ID] == nil {
			stats[descriptor.ID] = &capabilityStats{}
		}
		stats[descriptor.ID].observed++
	}
}

func markExtractorFailure(stats map[string]*capabilityStats, extractor SourceFactExtractor, requested []string, reason string) {
	for _, descriptor := range extractor.Capabilities() {
		if !containsString(requested, descriptor.ID) {
			continue
		}
		if stats[descriptor.ID] == nil {
			stats[descriptor.ID] = &capabilityStats{}
		}
		stats[descriptor.ID].failures++
		stats[descriptor.ID].reason = reason
	}
}

func mergeExtractorIdentities(existing []analysis.ExtractorIdentity, extractors []SourceFactExtractor) []analysis.ExtractorIdentity {
	result := make([]analysis.ExtractorIdentity, len(existing), len(existing)+len(extractors))
	copy(result, existing)
	for _, extractor := range extractors {
		result = append(result, analysis.ExtractorIdentity{ID: extractor.ID(), Version: extractor.Version()})
	}
	seen := make(map[string]struct{})
	unique := result[:0]
	for _, identity := range result {
		key := identity.ID + "\x00" + identity.Version
		if identity.ID == "" || identity.Version == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, identity)
	}
	sort.Slice(unique, func(i, j int) bool {
		if unique[i].ID == unique[j].ID {
			return unique[i].Version < unique[j].Version
		}
		return unique[i].ID < unique[j].ID
	})
	return unique
}

func uniqueCapabilities(values []analysis.CapabilityDescriptor) []analysis.CapabilityDescriptor {
	seen := make(map[string]struct{})
	result := make([]analysis.CapabilityDescriptor, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value.ID]; exists {
			continue
		}
		seen[value.ID] = struct{}{}
		value.SupportedLanguages = normalizeStrings(value.SupportedLanguages)
		result = append(result, value)
	}
	return result
}

func entityRefPointer(value analysis.EntityRef) *analysis.EntityRef { return &value }

func intPointer(value int) *int { return &value }

func documentationFactStatus(value string) string {
	switch value {
	case analysis.DocumentationPresent:
		return analysis.FactStatusObserved
	case analysis.DocumentationAbsent:
		return analysis.FactStatusAbsent
	case analysis.DocumentationUnknown:
		return analysis.FactStatusUnknown
	case analysis.DocumentationUnsupported:
		return analysis.FactStatusUnsupported
	case analysis.DocumentationPartial:
		return analysis.FactStatusPartial
	default:
		return analysis.FactStatusObserved
	}
}

func fileFactStatus(value string) string {
	switch value {
	case analysis.FileAnalysisPartial:
		return analysis.FactStatusPartial
	case analysis.FileAnalysisUnparsed, analysis.FileAnalysisUnknown:
		return analysis.FactStatusUnknown
	default:
		return analysis.FactStatusObserved
	}
}

func capabilitySubject(value string) string {
	switch value {
	case CapabilityFiles, CapabilitySize:
		return "file"
	case CapabilityDocumentation:
		return "documentation"
	default:
		return "symbol"
	}
}

func languageToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "language:")
	return value
}

func normalizeRelativePath(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	if value == "" {
		return ""
	}
	return path.Clean(value)
}

func analysisPathSafe(value string) bool {
	return value != "" && value != "." && !strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "../") && value != ".." && !strings.Contains(value, ":")
}

func normalizeRequestedCapabilities(values []string) []string {
	return normalizeStrings(values)
}

func normalizeStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func sourceEntityKey(ref analysis.EntityRef) string {
	return ref.Kind + "\x00" + ref.ID + "\x00" + ref.ScopeID + "\x00" + ref.SnapshotID
}

func opaqueID(kind string, parts ...string) string {
	data := []byte(kind + "\x00" + strings.Join(parts, "\x00"))
	hash := sha256.Sum256(data)
	return kind + ":" + hex.EncodeToString(hash[:16])
}
