package live

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/sourceindex"
)

type ScopeInfo struct {
	ScopeID         string   `json:"scope_id"`
	ProjectRoot     string   `json:"project_root"`
	AnalyzerID      string   `json:"analyzer_id"`
	AnalyzerVersion string   `json:"analyzer_version"`
	Language        string   `json:"language"`
	Status          string   `json:"status"`
	Capabilities    []string `json:"capabilities,omitempty"`
}

type ItemPage[T any] struct {
	Items      []T    `json:"items"`
	Total      int    `json:"total"`
	NextCursor string `json:"next_cursor,omitempty"`
}

type ModuleFactsPage struct {
	Items      []sourceindex.ModuleSourceFacts `json:"items"`
	Total      int                             `json:"total"`
	NextCursor string                          `json:"next_cursor,omitempty"`
}

type CallersCalleesResult struct {
	EntityID  string                  `json:"entity_id"`
	Status    string                  `json:"status"`
	Callers   []analysis.EntityRef    `json:"callers"`
	Callees   []analysis.EntityRef    `json:"callees"`
	Relations []analysis.CodeRelation `json:"relations,omitempty"`
}

type SourceContextResult struct {
	Context SourceContext `json:"context"`
}

type cursorState struct {
	Version string `json:"version"`
	Context string `json:"context"`
	Offset  int    `json:"offset"`
}

const cursorVersion = "arch-view.query-cursor/v1"

func (session *LiveSession) ListScopes(ctx context.Context, request QueryRequest) (QueryEnvelope, error) {
	if err := session.requireOperation(OperationSearch); err != nil {
		return QueryEnvelope{}, err
	}
	record, freshness, returned, request, err := session.prepareQuery(ctx, request)
	if err != nil {
		return QueryEnvelope{}, err
	}
	items := make([]ScopeInfo, 0, len(record.Run.Scopes))
	requestedScopes := make(map[string]struct{}, len(request.Query.ScopeIDs))
	for _, scopeID := range request.Query.ScopeIDs {
		if value := strings.TrimSpace(scopeID); value != "" {
			requestedScopes[value] = struct{}{}
		}
	}
	for _, scope := range record.Run.Scopes {
		if len(requestedScopes) > 0 {
			if _, ok := requestedScopes[scope.ScopeID]; !ok {
				continue
			}
		}
		capabilities := []string{}
		if item := record.ScopeModels[scope.ScopeID]; item.SourceIndex != nil {
			if item.SourceIndex.Projection != nil {
				capabilities = capabilitiesFromSnapshot(item.SourceIndex.Projection)
			} else if len(item.SourceIndex.Snapshots) > 0 {
				capabilities = capabilitiesFromSnapshot(&item.SourceIndex.Snapshots[0])
			}
		}
		items = append(items, ScopeInfo{ScopeID: scope.ScopeID, ProjectRoot: scope.ProjectRoot, AnalyzerID: scope.Analyzer.ID, AnalyzerVersion: scope.Analyzer.Version, Language: scope.Analyzer.Language, Status: string(scope.Status), Capabilities: capabilities})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ScopeID < items[j].ScopeID })
	return session.pageEnvelope(record, freshness, returned, request, "list_scopes", items, []string{"technical_ids"})
}

func (session *LiveSession) FindFiles(ctx context.Context, request QueryRequest) (QueryEnvelope, error) {
	if err := session.requireOperation(OperationSearch); err != nil {
		return QueryEnvelope{}, err
	}
	record, freshness, returned, request, err := session.prepareQuery(ctx, request)
	if err != nil {
		return QueryEnvelope{}, err
	}
	items := make([]analysis.FileRecord, 0)
	coverage := []CapabilityCoverage{}
	for _, snapshot := range session.sourceSnapshots(record, request.Query.ScopeIDs) {
		coverage = mergeCapabilityCoverage(coverage, snapshotCapabilityCoverage(snapshot))
		allowed := explicitFileIDs(snapshot, request.Query.ModuleIDs, request.Query.FileIDs)
		for _, file := range snapshot.Files {
			if allowed != nil {
				if _, ok := allowed[file.ID]; !ok {
					continue
				}
			}
			if !matchFileRecord(file, snapshot, request.Query) {
				continue
			}
			items = append(items, file)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		left := fileSortKey(items[i], record)
		right := fileSortKey(items[j], record)
		return left < right
	})
	if len(session.sourceSnapshots(record, request.Query.ScopeIDs)) == 0 {
		coverage = append(coverage, CapabilityCoverage{Capability: "source:index", Status: "unavailable", Reason: "source index is not attached to this revision"})
	}
	coverage = mergeCapabilityCoverage(coverage, session.requestedSourceCoverage(record, request.Query.ScopeIDs))
	envelope, err := session.pageEnvelope(record, freshness, returned, request, "find_files", items, []string{"provenance", "content_hash"})
	if err == nil {
		envelope.Capabilities = mergeCapabilityCoverage(envelope.Capabilities, coverage)
	}
	return envelope, err
}

func (session *LiveSession) FindSymbols(ctx context.Context, request QueryRequest) (QueryEnvelope, error) {
	if err := session.requireOperation(OperationSearch); err != nil {
		return QueryEnvelope{}, err
	}
	record, freshness, returned, request, err := session.prepareQuery(ctx, request)
	if err != nil {
		return QueryEnvelope{}, err
	}
	items := make([]analysis.SymbolRecord, 0)
	coverage := []CapabilityCoverage{}
	pathByFile := make(map[string]string)
	languageByFile := make(map[string]string)
	for _, snapshot := range session.sourceSnapshots(record, request.Query.ScopeIDs) {
		coverage = mergeCapabilityCoverage(coverage, snapshotCapabilityCoverage(snapshot))
		allowed := explicitSymbolIDs(snapshot, request.Query.ModuleIDs, request.Query.FileIDs)
		for _, file := range snapshot.Files {
			pathByFile[file.ID] = displaySourcePath(snapshot, file.Path)
			languageByFile[file.ID] = file.Language.ID
		}
		for _, symbol := range snapshot.Symbols {
			if allowed != nil {
				if _, ok := allowed[symbol.ID]; !ok {
					continue
				}
			}
			if !matchSymbolRecord(symbol, pathByFile, languageByFile, request.Query) {
				continue
			}
			items = append(items, compactSymbol(symbol, request.Projection))
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return symbolSortKey(items[i], pathByFile) < symbolSortKey(items[j], pathByFile) })
	if len(session.sourceSnapshots(record, request.Query.ScopeIDs)) == 0 {
		coverage = append(coverage, CapabilityCoverage{Capability: "source:index", Status: "unavailable", Reason: "source index is not attached to this revision"})
	}
	coverage = mergeCapabilityCoverage(coverage, session.requestedSourceCoverage(record, request.Query.ScopeIDs))
	envelope, err := session.pageEnvelope(record, freshness, returned, request, "find_symbols", items, []string{"provenance", "structural_facts", "body_span"})
	if err == nil {
		envelope.Capabilities = mergeCapabilityCoverage(envelope.Capabilities, coverage)
	}
	return envelope, err
}

func (session *LiveSession) GetDocumentation(ctx context.Context, request QueryRequest) (QueryEnvelope, error) {
	if err := session.requireOperation(OperationSearch); err != nil {
		return QueryEnvelope{}, err
	}
	record, freshness, returned, request, err := session.prepareQuery(ctx, request)
	if err != nil {
		return QueryEnvelope{}, err
	}
	items := make([]analysis.DocumentationRecord, 0)
	coverage := []CapabilityCoverage{}
	for _, snapshot := range session.sourceSnapshots(record, request.Query.ScopeIDs) {
		coverage = mergeCapabilityCoverage(coverage, snapshotCapabilityCoverage(snapshot))
		allowedFiles := explicitFileIDs(snapshot, request.Query.ModuleIDs, request.Query.FileIDs)
		allowedSymbols := explicitSymbolIDs(snapshot, request.Query.ModuleIDs, request.Query.FileIDs)
		filesByID := make(map[string]analysis.FileRecord, len(snapshot.Files))
		pathsByFile := make(map[string]string, len(snapshot.Files))
		languagesByFile := make(map[string]string, len(snapshot.Files))
		symbolsByID := make(map[string]analysis.SymbolRecord, len(snapshot.Symbols))
		for _, file := range snapshot.Files {
			filesByID[file.ID] = file
			pathsByFile[file.ID] = displaySourcePath(snapshot, file.Path)
			languagesByFile[file.ID] = file.Language.ID
		}
		for _, symbol := range snapshot.Symbols {
			symbolsByID[symbol.ID] = symbol
		}
		for _, documentation := range snapshot.Documentation {
			if request.Query.SubjectID != "" && documentation.SubjectRef.ID != request.Query.SubjectID {
				continue
			}
			if allowedFiles != nil {
				if documentation.SubjectRef.Kind == "file" {
					if _, ok := allowedFiles[documentation.SubjectRef.ID]; !ok {
						continue
					}
				} else if documentation.SubjectRef.Kind == "symbol" {
					if _, ok := allowedSymbols[documentation.SubjectRef.ID]; !ok {
						continue
					}
				} else {
					continue
				}
			}
			if !matchDocumentationRecord(documentation, request.Query) {
				continue
			}
			if !matchDocumentationSubject(documentation, snapshot, filesByID, symbolsByID, pathsByFile, languagesByFile, request.Query) {
				continue
			}
			items = append(items, compactDocumentation(documentation, request.Projection))
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return documentationSortKey(items[i]) < documentationSortKey(items[j]) })
	if len(session.sourceSnapshots(record, request.Query.ScopeIDs)) == 0 {
		coverage = append(coverage, CapabilityCoverage{Capability: "source:documentation", Status: "unavailable", Reason: "source index is not attached to this revision"})
	}
	coverage = mergeCapabilityCoverage(coverage, session.requestedSourceCoverage(record, request.Query.ScopeIDs))
	envelope, err := session.pageEnvelope(record, freshness, returned, request, "get_documentation", items, []string{"raw_text"})
	if err == nil {
		envelope.Capabilities = mergeCapabilityCoverage(envelope.Capabilities, coverage)
	}
	return envelope, err
}

func (session *LiveSession) GetModuleFacts(ctx context.Context, request QueryRequest, moduleID string) (QueryEnvelope, error) {
	if err := session.requireOperation(OperationSearch); err != nil {
		return QueryEnvelope{}, err
	}
	record, freshness, returned, request, err := session.prepareQuery(ctx, request)
	if err != nil {
		return QueryEnvelope{}, err
	}
	moduleID = strings.TrimSpace(moduleID)
	if moduleID == "" {
		return QueryEnvelope{}, newLiveError("QueryInvalid", "module_id is required", nil)
	}
	items := make([]sourceindex.ModuleSourceFacts, 0)
	coverage := []CapabilityCoverage{}
	for _, snapshot := range session.sourceSnapshots(record, request.Query.ScopeIDs) {
		coverage = mergeCapabilityCoverage(coverage, snapshotCapabilityCoverage(snapshot))
		index := analysis.SourceIndex{SchemaVersion: analysis.SourceIndexSchemaVersion, Snapshots: []analysis.SourceIndexSnapshot{*snapshot}}
		facts, factsErr := sourceindex.NewQueryService(&index).InspectModule(moduleID, sourceindex.QueryOptions{ScopeID: snapshot.ScopeContext.ScopeID, SnapshotID: snapshot.SnapshotID, IncludeDocumentationText: hasProjection(request.Projection, "documentation_text")})
		if factsErr == nil {
			items = append(items, facts)
		}
	}
	if len(items) == 0 && len(coverage) == 0 {
		coverage = []CapabilityCoverage{{Capability: "source:index", Status: "unavailable", Reason: "source index is not attached to this revision"}}
	}
	coverage = mergeCapabilityCoverage(coverage, session.requestedSourceCoverage(record, request.Query.ScopeIDs))
	return session.pageEnvelope(record, freshness, returned, request, "get_module_facts", ModuleFactsPage{Items: items, Total: len(items)}, nil, coverage)
}

func (session *LiveSession) GetCallersCallees(ctx context.Context, request QueryRequest, entityID string) (QueryEnvelope, error) {
	if err := session.requireOperation(OperationSearch); err != nil {
		return QueryEnvelope{}, err
	}
	record, freshness, returned, request, err := session.prepareQuery(ctx, request)
	if err != nil {
		return QueryEnvelope{}, err
	}
	entityID = strings.TrimSpace(entityID)
	if entityID == "" {
		return QueryEnvelope{}, newLiveError("QueryInvalid", "entity_id is required", nil)
	}
	result := CallersCalleesResult{EntityID: entityID, Status: "unsupported", Callers: []analysis.EntityRef{}, Callees: []analysis.EntityRef{}, Relations: []analysis.CodeRelation{}}
	coverage := []CapabilityCoverage{}
	for _, snapshot := range session.sourceSnapshots(record, request.Query.ScopeIDs) {
		coverage = mergeCapabilityCoverage(coverage, snapshotCapabilityCoverage(snapshot))
		if !snapshotHasCapability(snapshot, "source:callers-callees") {
			continue
		}
		result.Status = "observed"
		for _, relation := range snapshot.Relations {
			if relation.Category != "call" && relation.Category != "calls" && relation.Category != "source:call" {
				continue
			}
			if relation.FromRef.ID == entityID && relation.ToRef != nil {
				result.Callees = append(result.Callees, *relation.ToRef)
				result.Relations = append(result.Relations, relation)
			} else if relation.ToRef != nil && relation.ToRef.ID == entityID {
				result.Callers = append(result.Callers, relation.FromRef)
				result.Relations = append(result.Relations, relation)
			}
		}
	}
	if result.Status == "observed" {
		coverage = mergeCapabilityCoverage(coverage, []CapabilityCoverage{{Capability: "source:callers-callees", Status: "observed"}})
	} else {
		status := "unsupported"
		reason := "no registered analyzer reported caller/callee extraction"
		if len(session.sourceSnapshots(record, request.Query.ScopeIDs)) == 0 {
			status = "unavailable"
			reason = "source index is not attached to this revision"
		}
		coverage = mergeCapabilityCoverage(coverage, []CapabilityCoverage{{Capability: "source:callers-callees", Status: status, Reason: reason}})
	}
	coverage = mergeCapabilityCoverage(coverage, session.requestedSourceCoverage(record, request.Query.ScopeIDs))
	return session.pageEnvelope(record, freshness, returned, request, "get_callers_callees", result, nil, coverage)
}

func (session *LiveSession) FindText(ctx context.Context, request TextSearchQuery) (QueryEnvelope, error) {
	if err := session.requireOperation(OperationSearch); err != nil {
		return QueryEnvelope{}, err
	}
	consistency := request.Consistency
	if consistency == "" {
		consistency = session.validated.Config.FreshnessPolicy.DefaultConsistency
	}
	record, freshness, returned, err := session.prepareConsistency(ctx, consistency, request.Revision)
	if err != nil {
		return QueryEnvelope{}, err
	}
	if strings.TrimSpace(request.Pattern) == "" {
		return QueryEnvelope{}, newLiveError("QueryInvalid", "text pattern is required", nil)
	}
	if err := validateRecordScopeIDs(record, request.ScopeIDs); err != nil {
		return QueryEnvelope{}, err
	}
	if request.Mode == "" {
		request.Mode = "literal"
	}
	request.Consistency = consistency
	if request.Mode != "literal" && request.Mode != "regex" {
		return QueryEnvelope{}, newLiveError("QueryInvalid", "text search mode must be literal or regex", nil)
	}
	if request.Mode == "regex" && len(request.Pattern) > 512 {
		return QueryEnvelope{}, newLiveError("QueryInvalid", "regular expression exceeds the supported length", nil)
	}
	if err := validateStructuralQuery(StructuralQuery{PathGlob: request.PathGlob}); err != nil {
		return QueryEnvelope{}, err
	}
	var expression *regexp.Regexp
	if request.Mode == "regex" {
		pattern := request.Pattern
		if !request.CaseSensitive {
			pattern = "(?i)" + pattern
		}
		compiled, compileErr := regexp.Compile(pattern)
		if compileErr != nil {
			return QueryEnvelope{}, newLiveError("QueryInvalid", "text search regular expression is invalid", map[string]any{"error": compileErr.Error()})
		}
		expression = compiled
	}
	maxBytes, maxItems, err := session.normalizeBudget(request.MaxBytes, request.MaxItems)
	if err != nil {
		return QueryEnvelope{}, err
	}
	if request.MaxLineBytes == 0 {
		request.MaxLineBytes = maxBytes
	}
	if request.MaxLineBytes < 1 || request.MaxLineBytes > session.validated.Config.QueryPolicy.HardMaxBytes {
		return QueryEnvelope{}, newLiveError(ErrorQueryBudget, "max_line_bytes is outside the configured hard bound", nil)
	}
	contextKey := queryContext(record, request, "find_text", maxBytes, maxItems)
	offset, err := decodeQueryCursor(request.Cursor, contextKey)
	if err != nil {
		return QueryEnvelope{}, err
	}
	items := make([]TextMatch, 0)
	coverage := []CapabilityCoverage{}
	for _, snapshot := range session.sourceSnapshots(record, request.ScopeIDs) {
		coverage = mergeCapabilityCoverage(coverage, snapshotCapabilityCoverage(snapshot))
		for _, file := range snapshot.Files {
			if request.Language != "" && !textMatches(file.Language.ID, request.Language, request.CaseSensitive) {
				continue
			}
			displayPath := displaySourcePath(snapshot, file.Path)
			if request.PathGlob != "" {
				matched, matchErr := path.Match(request.PathGlob, displayPath)
				if matchErr != nil {
					return QueryEnvelope{}, newLiveError("QueryInvalid", "text search path glob is invalid", map[string]any{"error": matchErr.Error()})
				}
				if !matched {
					continue
				}
			}
			content, readErr := readSourceFile(session.validated.RepositoryRoot, snapshot, file.Path)
			if readErr != nil {
				return QueryEnvelope{}, readErr
			}
			expectedHash := file.Size.ContentHash
			if expectedHash.Value != "" && !strings.EqualFold(expectedHash.Value, digestBytes(content).Value) {
				return QueryEnvelope{}, newLiveError("SourceContextContentChanged", "source file no longer matches the selected revision", map[string]any{"path": displayPath, "revision": record.Snapshot.Revision})
			}
			for lineNumber, line := range splitLines(string(content)) {
				lineValue := truncateUTF8(line, request.MaxLineBytes)
				matches := textMatchIndices(lineValue, request.Pattern, request.CaseSensitive, expression)
				for _, match := range matches {
					items = append(items, TextMatch{ScopeID: snapshot.ScopeContext.ScopeID, Path: displayPath, Line: lineNumber + 1, Column: match[0] + 1, EndColumn: match[1] + 1, Text: lineValue, MatchLength: match[1] - match[0]})
				}
			}
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		left := fmt.Sprintf("%s\x00%s\x00%012d\x00%012d", items[i].ScopeID, items[i].Path, items[i].Line, items[i].Column)
		right := fmt.Sprintf("%s\x00%s\x00%012d\x00%012d", items[j].ScopeID, items[j].Path, items[j].Line, items[j].Column)
		return left < right
	})
	if len(session.sourceSnapshots(record, request.ScopeIDs)) == 0 {
		coverage = append(coverage, CapabilityCoverage{Capability: "source:text", Status: "unavailable", Reason: "source index is not attached to this revision"})
	}
	coverage = mergeCapabilityCoverage(coverage, session.requestedSourceCoverage(record, request.ScopeIDs))
	page, total, next, budget, err := pageSlice(items, offset, maxItems, maxBytes)
	if err != nil {
		return QueryEnvelope{}, err
	}
	if next != "" {
		nextOffset, _ := strconv.Atoi(strings.TrimPrefix(next, "offset:"))
		next = encodeQueryCursor(contextKey, nextOffset)
	}
	return QueryEnvelope{SchemaVersion: QuerySchemaVersion, SessionID: session.validated.Config.SessionID, SnapshotID: record.Snapshot.SnapshotID, Revision: record.Snapshot.Revision, RequestedConsistency: consistency, ReturnedConsistency: returned, Freshness: freshness, ScopeIDs: append([]string(nil), record.Snapshot.ScopeIDs...), Result: ItemPage[TextMatch]{Items: page, Total: total, NextCursor: next}, ResultCount: total, OmittedFields: []string{"source_content"}, Capabilities: coverage, Budget: budget, NextCursor: next, Diagnostics: []QueryDiagnostic{}}, nil
}

func (session *LiveSession) GetSourceContext(ctx context.Context, request SourceContextRequest) (QueryEnvelope, error) {
	if err := session.requireOperation(OperationSourceContext); err != nil {
		return QueryEnvelope{}, err
	}
	if request.MaxLines <= 0 || request.MaxBytes <= 0 {
		return QueryEnvelope{}, newLiveError(ErrorQueryBudget, "source context requires explicit positive max_lines and max_bytes", nil)
	}
	if request.MaxLines > session.validated.Config.QueryPolicy.HardContextLines || request.MaxBytes > session.validated.Config.QueryPolicy.HardMaxBytes {
		return QueryEnvelope{}, newLiveError(ErrorQueryBudget, "source context budget exceeds the configured hard bound", nil)
	}
	if request.Consistency == "" {
		request.Consistency = session.validated.Config.FreshnessPolicy.DefaultConsistency
	}
	if request.Consistency != ConsistencySpecific || request.Revision < 1 {
		return QueryEnvelope{}, newLiveError("QueryInvalid", "source context requires an explicit revision", nil)
	}
	record, freshness, returned, err := session.prepareConsistency(ctx, request.Consistency, request.Revision)
	if err != nil {
		return QueryEnvelope{}, err
	}
	if request.SessionID != "" && request.SessionID != session.validated.Config.SessionID {
		return QueryEnvelope{}, newLiveError("QueryInvalid", "source context session_id does not match the live session", nil)
	}
	var snapshot *analysis.SourceIndexSnapshot
	if request.EntityID != "" && request.ScopeID == "" && request.Span == nil {
		snapshot, err = session.sourceSnapshotForEntity(record, request.EntityID)
	} else {
		snapshot, err = session.sourceSnapshot(record, request.ScopeID, request.Span)
	}
	if err != nil {
		return QueryEnvelope{}, err
	}
	var evidence sourceindex.SourceFactEvidence
	if request.EntityID != "" {
		evidence, err = sourceindex.NewQueryService(&analysis.SourceIndex{SchemaVersion: analysis.SourceIndexSchemaVersion, Snapshots: []analysis.SourceIndexSnapshot{*snapshot}}).GetEvidence(request.EntityID, sourceindex.QueryOptions{ScopeID: snapshot.ScopeContext.ScopeID, SnapshotID: snapshot.SnapshotID})
		if err != nil {
			return QueryEnvelope{}, newLiveError(ErrorSourceContextOutOfScope, "source context entity is not present in the selected revision", map[string]any{"entity_id": request.EntityID})
		}
	}
	span := request.Span
	if span == nil {
		if len(evidence.Spans) == 0 {
			return QueryEnvelope{}, newLiveError(ErrorSourceContextOutOfScope, "source context requires an indexed entity with a source span", nil)
		}
		value := evidence.Spans[0]
		span = &value
	} else if request.EntityID != "" {
		if !containsAnalysisSpan(evidence.Spans, *span) {
			return QueryEnvelope{}, newLiveError(ErrorSourceContextOutOfScope, "source context span is not attached to the requested indexed entity", map[string]any{"entity_id": request.EntityID})
		}
	} else if !containsIndexedSpan(snapshot, *span) {
		return QueryEnvelope{}, newLiveError(ErrorSourceContextOutOfScope, "source context requires a span reported by the selected source index", nil)
	}
	file, ok := sourceFileByID(snapshot, span.FileID)
	if !ok {
		return QueryEnvelope{}, newLiveError(ErrorSourceContextOutOfScope, "source context span refers to a file outside the selected source index", nil)
	}
	content, err := readSourceFile(session.validated.RepositoryRoot, snapshot, file.Path)
	if err != nil {
		return QueryEnvelope{}, err
	}
	actualHash := digestBytes(content)
	expectedHash := span.ContentHash
	if expectedHash.Value == "" {
		expectedHash = file.Size.ContentHash
	}
	if expectedHash.Value != "" && !strings.EqualFold(expectedHash.Value, actualHash.Value) {
		return QueryEnvelope{}, newLiveError("SourceContextContentChanged", "source file no longer matches the indexed content hash", map[string]any{"path": displaySourcePath(snapshot, file.Path)})
	}
	lines := splitLines(string(content))
	start := span.Start.Line
	end := span.End.Line
	if start < 1 || end < start || start > len(lines) {
		return QueryEnvelope{}, newLiveError(ErrorSourceContextOutOfScope, "source context span has invalid line bounds", nil)
	}
	if end > len(lines) {
		end = len(lines)
	}
	if end-start+1 > request.MaxLines {
		end = start + request.MaxLines - 1
	}
	excerpt := strings.Join(lines[start-1:end], "\n")
	truncated := end < span.End.Line
	if len([]byte(excerpt)) > request.MaxBytes {
		excerpt = truncateUTF8(excerpt, request.MaxBytes)
		end = start + strings.Count(excerpt, "\n")
		truncated = true
	}
	value := SourceContext{ScopeID: snapshot.ScopeContext.ScopeID, SnapshotID: snapshot.SnapshotID, Path: displaySourcePath(snapshot, file.Path), StartLine: start, EndLine: end, Content: excerpt, ContentHash: actualHash, Truncated: truncated}
	return QueryEnvelope{SchemaVersion: QuerySchemaVersion, SessionID: session.validated.Config.SessionID, SnapshotID: record.Snapshot.SnapshotID, Revision: record.Snapshot.Revision, RequestedConsistency: request.Consistency, ReturnedConsistency: returned, Freshness: freshness, ScopeIDs: append([]string(nil), record.Snapshot.ScopeIDs...), Result: SourceContextResult{Context: value}, ResultCount: 1, OmittedFields: []string{}, Capabilities: []CapabilityCoverage{{Capability: "source:context", Status: "observed"}}, Budget: BudgetUsage{MaxBytes: request.MaxBytes, MaxItems: 1, EmittedBytes: len([]byte(excerpt)), EmittedItems: 1}, Diagnostics: []QueryDiagnostic{}}, nil
}

func (session *LiveSession) prepareQuery(ctx context.Context, request QueryRequest) (*RevisionRecord, Freshness, string, QueryRequest, error) {
	if request.SessionID != "" && request.SessionID != session.validated.Config.SessionID {
		return nil, Freshness{}, "", request, newLiveError("QueryInvalid", "query session_id does not match the live session", nil)
	}
	record, freshness, returned, err := session.prepareConsistency(ctx, request.Consistency, request.Revision)
	if err != nil {
		return nil, Freshness{}, "", request, err
	}
	if request.Consistency == "" {
		request.Consistency = session.validated.Config.FreshnessPolicy.DefaultConsistency
	}
	if err := validateRecordScopeIDs(record, request.Query.ScopeIDs); err != nil {
		return nil, Freshness{}, "", request, err
	}
	if err := validateStructuralQuery(request.Query); err != nil {
		return nil, Freshness{}, "", request, err
	}
	return record, freshness, returned, request, nil
}

func (session *LiveSession) prepareConsistency(ctx context.Context, consistency Consistency, revision int) (*RevisionRecord, Freshness, string, error) {
	if consistency == "" {
		consistency = session.validated.Config.FreshnessPolicy.DefaultConsistency
	}
	if consistency != ConsistencyLatestReady && consistency != ConsistencyRequireCurrent && consistency != ConsistencySpecific {
		return nil, Freshness{}, "", newLiveError("QueryInvalid", "consistency selector is unsupported", map[string]any{"consistency": consistency})
	}
	var record *RevisionRecord
	var err error
	returned := ""
	switch consistency {
	case ConsistencyRequireCurrent:
		record, err = session.EnsureCurrentSnapshot(ctx)
		returned = "current"
	case ConsistencySpecific:
		record, err = session.CurrentRecord(ConsistencySpecific, revision)
		returned = "specific_revision"
	default:
		record, err = session.CurrentRecord(ConsistencyLatestReady, 0)
		returned = "latest_ready"
	}
	if err != nil {
		return nil, Freshness{}, "", err
	}
	session.mu.RLock()
	building := session.building
	stale := session.stale
	unstable := session.inputUnstable
	changed := append([]string(nil), session.lastChanged...)
	groupID := session.lastGroupID
	reconciliation := session.lastReconcile
	state := session.state
	session.mu.RUnlock()
	freshness := session.freshnessFor(consistency, building, stale, unstable, changed, groupID, reconciliation, record)
	if record == nil && state == SessionFailed {
		freshness.Status = FreshnessFailed
	}
	if consistency == ConsistencySpecific {
		freshness.Reconciliation = ReconciliationNotRequested
		if latest, ok := session.store.ReadLatestReady(session.validated.Config.SessionID); ok && latest.Snapshot.Revision != record.Snapshot.Revision {
			freshness.Status = FreshnessStale
			freshness.LastReadyRevision = latest.Snapshot.Revision
		}
	}
	return record, freshness, returned, nil
}

func (session *LiveSession) normalizeBudget(maxBytes, maxItems int) (int, int, error) {
	policy := session.validated.Config.QueryPolicy
	if maxBytes == 0 {
		maxBytes = policy.DefaultMaxBytes
	}
	if maxItems == 0 {
		maxItems = policy.DefaultMaxItems
	}
	if maxBytes < 1 || maxBytes > policy.HardMaxBytes || maxItems < 1 || maxItems > policy.HardMaxItems {
		return 0, 0, newLiveError(ErrorQueryBudget, "query budget exceeds the configured hard bound", map[string]any{"max_bytes": maxBytes, "max_items": maxItems})
	}
	return maxBytes, maxItems, nil
}

func (session *LiveSession) pageEnvelope(record *RevisionRecord, freshness Freshness, returned string, request QueryRequest, operation string, values any, omitted []string, extra ...[]CapabilityCoverage) (QueryEnvelope, error) {
	maxBytes, maxItems, err := session.normalizeBudget(request.MaxBytes, request.MaxItems)
	if err != nil {
		return QueryEnvelope{}, err
	}
	switch values.(type) {
	case CallersCalleesResult, SourceContextResult:
		if request.Cursor != "" {
			return QueryEnvelope{}, newLiveError(ErrorQueryCursor, "this query result does not support pagination", nil)
		}
	}
	contextKey := queryContext(record, request, operation, maxBytes, maxItems)
	offset, err := decodeQueryCursor(request.Cursor, contextKey)
	if err != nil {
		return QueryEnvelope{}, err
	}
	items, total, next, budget, err := pageAny(values, offset, maxItems, maxBytes)
	if err != nil {
		return QueryEnvelope{}, err
	}
	if next != "" {
		nextOffset, parseErr := strconv.Atoi(strings.TrimPrefix(next, "offset:"))
		if parseErr != nil {
			return QueryEnvelope{}, newLiveError(ErrorQueryCursor, "query pagination produced an invalid continuation", nil)
		}
		next = encodeQueryCursor(contextKey, nextOffset)
	}
	result := values
	switch values.(type) {
	case []ScopeInfo:
		result = ItemPage[ScopeInfo]{Items: items.([]ScopeInfo), Total: total, NextCursor: next}
	case []analysis.FileRecord:
		result = ItemPage[analysis.FileRecord]{Items: items.([]analysis.FileRecord), Total: total, NextCursor: next}
	case []analysis.SymbolRecord:
		result = ItemPage[analysis.SymbolRecord]{Items: items.([]analysis.SymbolRecord), Total: total, NextCursor: next}
	case []analysis.DocumentationRecord:
		result = ItemPage[analysis.DocumentationRecord]{Items: items.([]analysis.DocumentationRecord), Total: total, NextCursor: next}
	case ModuleFactsPage:
		modulePage := items.(ModuleFactsPage)
		modulePage.NextCursor = next
		result = modulePage
	case CallersCalleesResult:
		result = values
	case SourceContextResult:
		result = values
	}
	capabilities := []CapabilityCoverage{}
	for _, group := range extra {
		capabilities = mergeCapabilityCoverage(capabilities, group)
	}
	return QueryEnvelope{SchemaVersion: QuerySchemaVersion, SessionID: session.validated.Config.SessionID, SnapshotID: record.Snapshot.SnapshotID, Revision: record.Snapshot.Revision, RequestedConsistency: request.Consistency, ReturnedConsistency: returned, Freshness: freshness, ScopeIDs: append([]string(nil), record.Snapshot.ScopeIDs...), Result: result, ResultCount: total, OmittedFields: uniqueStrings(omitted), Capabilities: capabilities, Budget: budget, NextCursor: next, Diagnostics: []QueryDiagnostic{}}, nil
}

func queryContext(record *RevisionRecord, request any, kind string, maxBytes, maxItems int) string {
	switch value := request.(type) {
	case QueryRequest:
		value.Cursor = ""
		request = value
	case TextSearchQuery:
		value.Cursor = ""
		request = value
	case QualityFindingsRequest:
		value.Cursor = ""
		request = value
	case QualityCatalogRequest:
		value.Cursor = ""
		request = value
	case QualityEvidenceRequest:
		value.Cursor = ""
		request = value
	case QualityBaselinesRequest:
		value.QueryRequest.Cursor = ""
		request = value
	}
	value := struct {
		Version  string
		Kind     string
		Revision int
		Snapshot string
		Request  any
		MaxBytes int
		MaxItems int
	}{cursorVersion, kind, record.Snapshot.Revision, record.Snapshot.SnapshotID, request, maxBytes, maxItems}
	data, _ := json.Marshal(value)
	return digestJSON(data).Value
}

func encodeQueryCursor(context string, offset int) string {
	data, _ := json.Marshal(cursorState{Version: cursorVersion, Context: context, Offset: offset})
	return base64.RawURLEncoding.EncodeToString(data)
}

func decodeQueryCursor(value, expectedContext string) (int, error) {
	if value == "" {
		return 0, nil
	}
	data, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return 0, newLiveError(ErrorQueryCursor, "query cursor is malformed", nil)
	}
	var cursor cursorState
	if err := json.Unmarshal(data, &cursor); err != nil || cursor.Version != cursorVersion || cursor.Context != expectedContext || cursor.Offset < 0 {
		return 0, newLiveError(ErrorQueryCursor, "query cursor does not belong to this session, revision, query, projection, or budget", nil)
	}
	return cursor.Offset, nil
}

func pageAny(values any, offset, maxItems, maxBytes int) (any, int, string, BudgetUsage, error) {
	switch typed := values.(type) {
	case []ScopeInfo:
		return pageSlice(typed, offset, maxItems, maxBytes)
	case []analysis.FileRecord:
		return pageSlice(typed, offset, maxItems, maxBytes)
	case []analysis.SymbolRecord:
		return pageSlice(typed, offset, maxItems, maxBytes)
	case []analysis.DocumentationRecord:
		return pageSlice(typed, offset, maxItems, maxBytes)
	case ModuleFactsPage:
		page, total, next, budget, err := pageSlice(typed.Items, offset, maxItems, maxBytes)
		if err != nil {
			return nil, 0, "", BudgetUsage{}, err
		}
		return ModuleFactsPage{Items: page, Total: total, NextCursor: next}, total, next, budget, nil
	case CallersCalleesResult:
		emittedBytes := jsonSize(typed)
		if emittedBytes > maxBytes {
			return nil, 0, "", BudgetUsage{}, newLiveError(ErrorQueryBudget, "the callers/callees result exceeds the query byte budget", map[string]any{"max_bytes": maxBytes, "emitted_bytes": emittedBytes})
		}
		return typed, 1, "", BudgetUsage{MaxBytes: maxBytes, MaxItems: maxItems, EmittedBytes: emittedBytes, EmittedItems: 1}, nil
	default:
		return values, 1, "", BudgetUsage{MaxBytes: maxBytes, MaxItems: maxItems}, nil
	}
}

func pageSlice[T any](values []T, offset, maxItems, maxBytes int) ([]T, int, string, BudgetUsage, error) {
	if offset < 0 || offset > len(values) {
		return nil, 0, "", BudgetUsage{}, newLiveError(ErrorQueryCursor, "query cursor is outside the result set", nil)
	}
	end := offset
	emittedBytes := 0
	for end < len(values) && end-offset < maxItems {
		size := jsonSize(values[end])
		if emittedBytes+size > maxBytes {
			if end == offset {
				return nil, 0, "", BudgetUsage{}, newLiveError(ErrorQueryBudget, "the first result exceeds the query byte budget", map[string]any{"max_bytes": maxBytes})
			}
			break
		}
		emittedBytes += size
		end++
	}
	truncated := end < len(values)
	next := ""
	if truncated {
		// The caller binds this cursor to the complete query context after the
		// typed page has been assembled.
		next = fmt.Sprintf("offset:%d", end)
	}
	return append([]T(nil), values[offset:end]...), len(values), next, BudgetUsage{MaxBytes: maxBytes, MaxItems: maxItems, EmittedBytes: emittedBytes, EmittedItems: end - offset, Truncated: truncated}, nil
}

func jsonSize(value any) int {
	data, _ := json.Marshal(value)
	return len(data)
}

func (session *LiveSession) sourceSnapshots(record *RevisionRecord, scopeIDs []string) []*analysis.SourceIndexSnapshot {
	if record == nil || record.Model.SourceIndex == nil {
		return nil
	}
	allowed := make(map[string]struct{}, len(scopeIDs))
	for _, scopeID := range scopeIDs {
		allowed[scopeID] = struct{}{}
	}
	result := make([]*analysis.SourceIndexSnapshot, 0)
	for index := range record.Model.SourceIndex.Snapshots {
		snapshot := &record.Model.SourceIndex.Snapshots[index]
		if len(allowed) > 0 {
			if _, ok := allowed[snapshot.ScopeContext.ScopeID]; !ok {
				continue
			}
		}
		result = append(result, snapshot)
	}
	if len(result) == 0 && len(allowed) == 0 && record.Model.SourceIndex.Projection != nil {
		result = append(result, record.Model.SourceIndex.Projection)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ScopeContext.ScopeID < result[j].ScopeContext.ScopeID })
	return result
}

func (session *LiveSession) sourceSnapshot(record *RevisionRecord, scopeID string, span *analysis.SourceSpan) (*analysis.SourceIndexSnapshot, error) {
	if record == nil || record.Model.SourceIndex == nil {
		return nil, newLiveError(ErrorSourceContextOutOfScope, "source index is unavailable for this revision", nil)
	}
	values := session.sourceSnapshots(record, []string{scopeID})
	if scopeID == "" {
		values = session.sourceSnapshots(record, nil)
	}
	if len(values) == 0 {
		return nil, newLiveError(ErrorSourceContextOutOfScope, "requested source scope is unavailable", map[string]any{"scope_id": scopeID})
	}
	if span != nil {
		var matched *analysis.SourceIndexSnapshot
		for _, value := range values {
			if _, ok := sourceFileByID(value, span.FileID); ok {
				if matched != nil {
					return nil, newLiveError(ErrorSourceContextOutOfScope, "source context span matches more than one source scope; provide scope_id", nil)
				}
				matched = value
			}
		}
		if matched != nil {
			return matched, nil
		}
	}
	if len(values) == 1 {
		return values[0], nil
	}
	return nil, newLiveError(ErrorSourceContextOutOfScope, "source context scope is ambiguous; provide scope_id", nil)
}

func (session *LiveSession) sourceSnapshotForEntity(record *RevisionRecord, entityID string) (*analysis.SourceIndexSnapshot, error) {
	values := session.sourceSnapshots(record, nil)
	if len(values) == 0 {
		return nil, newLiveError(ErrorSourceContextOutOfScope, "source index is unavailable for this revision", nil)
	}
	var matched *analysis.SourceIndexSnapshot
	for _, snapshot := range values {
		index := analysis.SourceIndex{SchemaVersion: analysis.SourceIndexSchemaVersion, Snapshots: []analysis.SourceIndexSnapshot{*snapshot}}
		if _, err := sourceindex.NewQueryService(&index).GetEvidence(entityID, sourceindex.QueryOptions{ScopeID: snapshot.ScopeContext.ScopeID, SnapshotID: snapshot.SnapshotID}); err != nil {
			continue
		}
		if matched != nil {
			return nil, newLiveError(ErrorSourceContextOutOfScope, "source context entity matches more than one source scope; provide scope_id", map[string]any{"entity_id": entityID})
		}
		matched = snapshot
	}
	if matched == nil {
		return nil, newLiveError(ErrorSourceContextOutOfScope, "source context entity is not present in the selected revision", map[string]any{"entity_id": entityID})
	}
	return matched, nil
}

func sourceFileByID(snapshot *analysis.SourceIndexSnapshot, id string) (analysis.FileRecord, bool) {
	for _, file := range snapshot.Files {
		if file.ID == id {
			return file, true
		}
	}
	return analysis.FileRecord{}, false
}

func containsAnalysisSpan(values []analysis.SourceSpan, target analysis.SourceSpan) bool {
	for _, value := range values {
		if value.FileID == target.FileID && value.Start == target.Start && value.End == target.End && value.CoordinateSystem == target.CoordinateSystem {
			return true
		}
	}
	return false
}

func containsIndexedSpan(snapshot *analysis.SourceIndexSnapshot, target analysis.SourceSpan) bool {
	if snapshot == nil {
		return false
	}
	for _, symbol := range snapshot.Symbols {
		for _, location := range symbol.Locations {
			if sameAnalysisSpan(location.Span, target) {
				return true
			}
		}
		if symbol.BodySpan != nil && sameAnalysisSpan(*symbol.BodySpan, target) {
			return true
		}
	}
	for _, documentation := range snapshot.Documentation {
		for _, span := range documentation.Spans {
			if sameAnalysisSpan(span, target) {
				return true
			}
		}
	}
	for _, occurrence := range snapshot.Occurrences {
		if sameAnalysisSpan(occurrence.SourceSpan, target) {
			return true
		}
	}
	for _, relation := range snapshot.Relations {
		for _, span := range relation.EvidenceSpans {
			if sameAnalysisSpan(span, target) {
				return true
			}
		}
	}
	return false
}

func sameAnalysisSpan(left, right analysis.SourceSpan) bool {
	return left.FileID == right.FileID && left.Start == right.Start && left.End == right.End && left.CoordinateSystem == right.CoordinateSystem
}

func readSourceFile(repositoryRoot string, snapshot *analysis.SourceIndexSnapshot, relative string) ([]byte, error) {
	if snapshot == nil {
		return nil, newLiveError(ErrorSourceContextOutOfScope, "source snapshot is unavailable", nil)
	}
	projectRoot, err := normalizeRelativePath(snapshot.ScopeContext.ProjectRoot, true)
	if err != nil {
		return nil, newLiveError(ErrorSourceContextOutOfScope, "source snapshot project root is invalid", nil)
	}
	filePath, err := normalizeRelativePath(relative, false)
	if err != nil {
		return nil, newLiveError(ErrorSourceContextOutOfScope, "source context file path is invalid", nil)
	}
	combined := filePath
	if projectRoot != "." && projectRoot != "" {
		combined = filepath.ToSlash(filepath.Join(projectRoot, filepath.FromSlash(filePath)))
	}
	root, err := filepath.Abs(repositoryRoot)
	if err != nil {
		return nil, newLiveError(ErrorSourceContextOutOfScope, "source repository root is invalid", nil)
	}
	candidate := filepath.Clean(filepath.Join(root, filepath.FromSlash(combined)))
	if !pathWithin(root, candidate) {
		return nil, newLiveError(ErrorSourceContextOutOfScope, "source context path escapes the repository root", nil)
	}
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil || !pathWithin(root, resolved) {
		return nil, newLiveError(ErrorSourceContextOutOfScope, "source context path is missing or escapes the repository root", nil)
	}
	return osReadFile(resolved)
}

func displaySourcePath(snapshot *analysis.SourceIndexSnapshot, relative string) string {
	if snapshot == nil || snapshot.ScopeContext.ProjectRoot == "" || snapshot.ScopeContext.ProjectRoot == "." {
		return filepath.ToSlash(filepath.Clean(relative))
	}
	return filepath.ToSlash(filepath.Join(snapshot.ScopeContext.ProjectRoot, filepath.FromSlash(relative)))
}

func explicitFileIDs(snapshot *analysis.SourceIndexSnapshot, moduleIDs, fileIDs []string) map[string]struct{} {
	if len(moduleIDs) == 0 && len(fileIDs) == 0 {
		return nil
	}
	result := make(map[string]struct{})
	for _, id := range fileIDs {
		result[strings.TrimSpace(id)] = struct{}{}
	}
	modules := make(map[string]struct{}, len(moduleIDs))
	for _, id := range moduleIDs {
		modules[strings.TrimSpace(id)] = struct{}{}
	}
	for _, relation := range snapshot.Relations {
		if relation.Category == analysis.RelationContains && relation.FromRef.Kind == "module" && relation.ToRef != nil && relation.ToRef.Kind == "file" {
			if _, ok := modules[relation.FromRef.ID]; ok {
				result[relation.ToRef.ID] = struct{}{}
			}
		}
	}
	return result
}

func explicitSymbolIDs(snapshot *analysis.SourceIndexSnapshot, moduleIDs, fileIDs []string) map[string]struct{} {
	if len(moduleIDs) == 0 && len(fileIDs) == 0 {
		return nil
	}
	files := explicitFileIDs(snapshot, moduleIDs, fileIDs)
	result := make(map[string]struct{})
	for _, relation := range snapshot.Relations {
		if (relation.Category != analysis.RelationContains && relation.Category != analysis.RelationDeclares) || relation.FromRef.Kind != "file" || relation.ToRef == nil || relation.ToRef.Kind != "symbol" {
			continue
		}
		if _, ok := files[relation.FromRef.ID]; ok {
			result[relation.ToRef.ID] = struct{}{}
		}
	}
	return result
}

func matchFileRecord(file analysis.FileRecord, snapshot *analysis.SourceIndexSnapshot, query StructuralQuery) bool {
	pathValue := displaySourcePath(snapshot, file.Path)
	return (query.PathPrefix == "" || textHasPrefix(pathValue, query.PathPrefix, query.CaseSensitive)) &&
		(query.PathGlob == "" || pathMatches(query.PathGlob, pathValue)) &&
		(query.Language == "" || textMatches(file.Language.ID, query.Language, query.CaseSensitive)) &&
		(len(query.Roles) == 0 || anyTextMatch(file.Roles, query.Roles, query.CaseSensitive)) &&
		(len(query.Statuses) == 0 || anyTextMatch([]string{file.AnalysisStatus}, query.Statuses, query.CaseSensitive))
}

func matchSymbolRecord(symbol analysis.SymbolRecord, paths, languages map[string]string, query StructuralQuery) bool {
	if query.Name != "" && !textMatches(symbol.Name, query.Name, query.CaseSensitive) || query.QualifiedName != "" && !textMatches(symbol.QualifiedName, query.QualifiedName, query.CaseSensitive) || len(query.SymbolCategories) > 0 && !anyTextMatch([]string{symbol.Category}, query.SymbolCategories, query.CaseSensitive) {
		return false
	}
	if query.Language != "" {
		matched := textMatches(symbol.LanguageKind, query.Language, query.CaseSensitive)
		if !matched {
			for _, location := range symbol.Locations {
				if textMatches(languages[location.Span.FileID], query.Language, query.CaseSensitive) {
					matched = true
					break
				}
			}
		}
		if !matched {
			return false
		}
	}
	if query.PathPrefix != "" || query.PathGlob != "" {
		matched := false
		for _, location := range symbol.Locations {
			filePath := paths[location.Span.FileID]
			if (query.PathPrefix == "" || textHasPrefix(filePath, query.PathPrefix, query.CaseSensitive)) && (query.PathGlob == "" || pathMatches(query.PathGlob, filePath)) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func matchDocumentationRecord(value analysis.DocumentationRecord, query StructuralQuery) bool {
	if query.DocumentationText == "" {
		return true
	}
	textValue := value.NormalizedText
	if textValue == "" {
		textValue = value.RawText
	}
	return textMatches(textValue, query.DocumentationText, query.CaseSensitive)
}

func matchDocumentationSubject(value analysis.DocumentationRecord, snapshot *analysis.SourceIndexSnapshot, files map[string]analysis.FileRecord, symbols map[string]analysis.SymbolRecord, paths, languages map[string]string, query StructuralQuery) bool {
	switch value.SubjectRef.Kind {
	case "file":
		file, ok := files[value.SubjectRef.ID]
		if !ok {
			return false
		}
		return matchFileRecord(file, snapshot, query)
	case "symbol":
		symbol, ok := symbols[value.SubjectRef.ID]
		if !ok {
			return false
		}
		return matchSymbolRecord(symbol, paths, languages, query)
	default:
		return query.PathPrefix == "" && query.PathGlob == "" && query.Language == "" && len(query.Roles) == 0 && query.Name == "" && query.QualifiedName == "" && len(query.SymbolCategories) == 0 && len(query.Statuses) == 0
	}
}

func compactSymbol(value analysis.SymbolRecord, projection []string) analysis.SymbolRecord {
	if !hasProjection(projection, "structural_facts") {
		value.StructuralFacts = nil
		value.MemberCount = nil
		value.MethodCount = nil
		value.DependencyCount = nil
		value.ConcreteDependencyCount = nil
		value.InterfaceMethodCount = nil
		value.TypeSwitchCount = nil
		value.HierarchyDepth = nil
		value.DerivedTypeCount = nil
		value.AbstractionCount = nil
	}
	return value
}

func compactDocumentation(value analysis.DocumentationRecord, projection []string) analysis.DocumentationRecord {
	if !hasProjection(projection, "documentation_text") {
		value.RawText = ""
		runes := []rune(value.NormalizedText)
		if len(runes) > 512 {
			value.NormalizedText = string(runes[:512])
			value.Completeness = analysis.DocumentationTruncated
		}
	}
	return value
}

func symbolSortKey(value analysis.SymbolRecord, paths map[string]string) string {
	pathValue := ""
	start := 0
	if len(value.Locations) > 0 {
		pathValue = paths[value.Locations[0].Span.FileID]
		if pathValue == "" {
			pathValue = value.Locations[0].Span.FileID
		}
		start = value.Locations[0].Span.Start.ByteOffset
	}
	return fmt.Sprintf("%s\x00%012d\x00%s\x00%s\x00%s\x00%s", pathValue, start, value.Category, value.LanguageKind, value.Name, value.ID)
}

func documentationSortKey(value analysis.DocumentationRecord) string {
	start := 0
	if len(value.Spans) > 0 {
		start = value.Spans[0].Start.ByteOffset
	}
	return fmt.Sprintf("%s\x00%012d\x00%s", value.SubjectRef.ID, start, value.ID)
}

func fileSortKey(value analysis.FileRecord, record *RevisionRecord) string {
	return value.Path + "\x00" + value.ID
}

func snapshotCapabilityCoverage(snapshot *analysis.SourceIndexSnapshot) []CapabilityCoverage {
	if snapshot == nil {
		return nil
	}
	result := make([]CapabilityCoverage, 0, len(snapshot.Coverage))
	for _, coverage := range snapshot.Coverage {
		providers := []string{}
		if coverage.Provenance.Provider != "" {
			providers = []string{coverage.Provenance.Provider}
		}
		result = append(result, CapabilityCoverage{Capability: coverage.Capability, Status: coverage.Status, Reason: coverage.Reason, ProviderIDs: providers})
	}
	for _, capability := range snapshot.Capabilities {
		if !containsCapability(result, capability.ID) {
			providers := []string{}
			if snapshot.Producer.AnalyzerID != "" {
				providers = []string{snapshot.Producer.AnalyzerID}
			}
			result = append(result, CapabilityCoverage{Capability: capability.ID, Status: "observed", ProviderIDs: providers})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Capability < result[j].Capability })
	return result
}

func mergeCapabilityCoverage(left, right []CapabilityCoverage) []CapabilityCoverage {
	byID := make(map[string]CapabilityCoverage, len(left)+len(right))
	for _, value := range append(append([]CapabilityCoverage(nil), left...), right...) {
		if value.Capability == "" {
			continue
		}
		if existing, ok := byID[value.Capability]; ok {
			existing.Status = mergeCoverageStatus(existing.Status, value.Status)
			if existing.Reason == "" {
				existing.Reason = value.Reason
			}
			existing.ProviderIDs = uniqueStrings(append(existing.ProviderIDs, value.ProviderIDs...))
			byID[value.Capability] = existing
		} else {
			byID[value.Capability] = value
		}
	}
	result := make([]CapabilityCoverage, 0, len(byID))
	for _, value := range byID {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Capability < result[j].Capability })
	return result
}

func mergeCoverageStatus(left, right string) string {
	if left == "" {
		return right
	}
	if right == "" || left == right {
		return left
	}
	// Different providers/scopes reporting different states is aggregate
	// partial coverage. Treating the least capable provider as authoritative
	// would incorrectly erase facts observed by another analyzer.
	return "partial"
}

func containsCapability(values []CapabilityCoverage, target string) bool {
	for _, value := range values {
		if value.Capability == target {
			return true
		}
	}
	return false
}

func capabilitiesFromSnapshot(snapshot *analysis.SourceIndexSnapshot) []string {
	values := []string{}
	if snapshot != nil {
		for _, value := range snapshot.Capabilities {
			values = append(values, value.ID)
		}
	}
	return uniqueStrings(values)
}

func snapshotHasCapability(snapshot *analysis.SourceIndexSnapshot, capability string) bool {
	for _, value := range snapshot.Capabilities {
		if value.ID == capability {
			return true
		}
	}
	return false
}

func (session *LiveSession) requestedSourceCoverage(record *RevisionRecord, scopeIDs []string) []CapabilityCoverage {
	requested := session.validated.Config.SourceIndexRequest.Capabilities
	if len(requested) == 0 {
		return nil
	}
	snapshots := session.sourceSnapshots(record, scopeIDs)
	if len(snapshots) == 0 {
		result := make([]CapabilityCoverage, 0, len(requested))
		for _, capability := range requested {
			result = append(result, CapabilityCoverage{Capability: capability, Status: "unavailable", Reason: "source index is not attached to this revision"})
		}
		return result
	}
	observed := make(map[string]string)
	for _, snapshot := range snapshots {
		for _, capability := range snapshot.Capabilities {
			observed[capability.ID] = "observed"
		}
		for _, coverage := range snapshot.Coverage {
			observed[coverage.Capability] = mergeCoverageStatus(observed[coverage.Capability], coverage.Status)
		}
	}
	result := make([]CapabilityCoverage, 0, len(requested))
	for _, capability := range requested {
		status, ok := observed[capability]
		if !ok {
			result = append(result, CapabilityCoverage{Capability: capability, Status: "unsupported", Reason: "no selected analyzer reported this requested source capability"})
			continue
		}
		result = append(result, CapabilityCoverage{Capability: capability, Status: status})
	}
	return result
}

func anyTextMatch(values, expected []string, caseSensitive bool) bool {
	for _, value := range values {
		for _, candidate := range expected {
			if textMatches(value, candidate, caseSensitive) {
				return true
			}
		}
	}
	return false
}

func textMatches(value, expected string, caseSensitive bool) bool {
	if caseSensitive {
		return strings.Contains(value, expected)
	}
	return strings.Contains(strings.ToLower(value), strings.ToLower(expected))
}

func textHasPrefix(value, expected string, caseSensitive bool) bool {
	if caseSensitive {
		return strings.HasPrefix(value, expected)
	}
	return strings.HasPrefix(strings.ToLower(value), strings.ToLower(expected))
}

func pathMatches(pattern, value string) bool {
	matched, err := path.Match(pattern, value)
	return err == nil && matched
}

func validateStructuralQuery(query StructuralQuery) error {
	for name, value := range map[string]string{"path_prefix": query.PathPrefix, "path_glob": query.PathGlob} {
		if value == "" {
			continue
		}
		normalized := strings.ReplaceAll(value, "\\", "/")
		if strings.ContainsAny(value, "\x00\r\n") || path.IsAbs(normalized) || normalized == ".." || strings.HasPrefix(normalized, "../") || strings.Contains(normalized, "/../") {
			return newLiveError("QueryInvalid", name+" must remain repository-relative", map[string]any{name: value})
		}
	}
	if query.PathGlob == "" {
		return nil
	}
	if _, err := path.Match(query.PathGlob, ""); err != nil {
		return newLiveError("QueryInvalid", "query path glob is invalid", map[string]any{"error": err.Error()})
	}
	return nil
}

func validateRecordScopeIDs(record *RevisionRecord, requested []string) error {
	if len(requested) == 0 {
		return nil
	}
	available := make(map[string]struct{}, len(record.Snapshot.ScopeIDs))
	for _, scopeID := range record.Snapshot.ScopeIDs {
		available[scopeID] = struct{}{}
	}
	for _, scopeID := range requested {
		if _, ok := available[scopeID]; !ok {
			return newLiveError(ErrorRevisionUnavailable, "requested scope is unavailable in the selected revision", map[string]any{"scope_id": scopeID, "revision": record.Snapshot.Revision})
		}
	}
	return nil
}

func hasProjection(values []string, target string) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func splitLines(value string) []string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	return strings.Split(value, "\n")
}

func truncateUTF8(value string, maxBytes int) string {
	if maxBytes < 1 || len(value) <= maxBytes {
		return value
	}
	value = value[:maxBytes]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value
}

func textMatchIndices(line, pattern string, caseSensitive bool, expression *regexp.Regexp) [][2]int {
	if expression != nil {
		matches := expression.FindAllStringIndex(line, -1)
		result := make([][2]int, 0, len(matches))
		for _, match := range matches {
			result = append(result, [2]int{match[0], match[1]})
		}
		return result
	}
	if !caseSensitive {
		lineLower := strings.ToLower(line)
		pattern = strings.ToLower(pattern)
		result := make([][2]int, 0)
		for offset := 0; ; {
			index := strings.Index(lineLower[offset:], pattern)
			if index < 0 {
				break
			}
			start := offset + index
			result = append(result, [2]int{start, start + len(pattern)})
			offset = start + maxInt(1, len(pattern))
		}
		return result
	}
	result := make([][2]int, 0)
	for offset := 0; ; {
		index := strings.Index(line[offset:], pattern)
		if index < 0 {
			break
		}
		start := offset + index
		result = append(result, [2]int{start, start + len(pattern)})
		offset = start + maxInt(1, len(pattern))
	}
	return result
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func digestBytes(value []byte) ContentDigest {
	hash := sha256.Sum256(value)
	return ContentDigest{Algorithm: "hash:sha-256", Value: hex.EncodeToString(hash[:])}
}

func osReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, newLiveError(ErrorSourceContextOutOfScope, "source context file could not be read", map[string]any{"error": err.Error()})
	}
	return data, nil
}
