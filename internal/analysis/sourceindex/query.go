package sourceindex

import (
	"encoding/base64"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

const (
	DefaultQueryLimit             = 50
	MaxQueryLimit                 = 200
	CompactDocumentationRuneLimit = 512
)

// QueryOptions is shared by all structural read operations. No query returns
// source text unless the caller separately requests bounded evidence context.
type QueryOptions struct {
	SnapshotID string
	ScopeID    string
	// ModuleIDs and FileIDs are explicit containment filters. A module ID is
	// resolved only through module->file contains relations; a file ID is
	// resolved directly. When both are supplied their memberships are unioned.
	ModuleIDs                []string
	FileIDs                  []string
	PathPrefix               string
	PathGlob                 string
	Language                 string
	Role                     string
	Status                   string
	Name                     string
	QualifiedName            string
	Category                 string
	LanguageKind             string
	DocumentationText        string
	SubjectID                string
	SubjectIDs               []string
	CaseSensitive            bool
	Limit                    int
	Cursor                   string
	IncludeDocumentationText bool
}

type Page[T any] struct {
	SnapshotID string `json:"snapshot_id"`
	ScopeID    string `json:"scope_id"`
	Items      []T    `json:"items"`
	Total      int    `json:"total"`
	NextCursor string `json:"next_cursor,omitempty"`
	// Coverage is optional page metadata. It keeps bounded consumers aware of
	// unsupported, partial, and unknown capabilities without returning the
	// complete source-index attachment.
	Coverage []analysis.CoverageRecord `json:"coverage,omitempty"`
}

type ModuleSourceFacts struct {
	SnapshotID    string                         `json:"snapshot_id"`
	ScopeID       string                         `json:"scope_id"`
	ModuleID      string                         `json:"module_id"`
	Files         []analysis.FileRecord          `json:"files"`
	Symbols       []analysis.SymbolRecord        `json:"symbols"`
	Documentation []analysis.DocumentationRecord `json:"documentation"`
	Relations     []analysis.CodeRelation        `json:"relations"`
	Coverage      []analysis.CoverageRecord      `json:"coverage"`
}

type SourceFactEvidence struct {
	SnapshotID         string                         `json:"snapshot_id"`
	ScopeID            string                         `json:"scope_id"`
	EntityID           string                         `json:"entity_id"`
	Spans              []analysis.SourceSpan          `json:"spans"`
	SourceReferenceIDs []string                       `json:"source_reference_ids"`
	Documentation      []analysis.DocumentationRecord `json:"documentation"`
	Relations          []analysis.CodeRelation        `json:"relations"`
}

// QueryService is a deterministic, read-only projection over one source-index
// attachment. It performs no path discovery and never broadens a scope.
type QueryService struct {
	Index    *analysis.SourceIndex
	MaxLimit int
}

func NewQueryService(index *analysis.SourceIndex) *QueryService {
	return &QueryService{Index: index, MaxLimit: MaxQueryLimit}
}

func (service *QueryService) snapshot(options QueryOptions) (*analysis.SourceIndexSnapshot, error) {
	if err := validateQueryOptions(options); err != nil {
		return nil, err
	}
	if service == nil || service.Index == nil {
		return nil, analysis.NewHostError(analysis.ErrSourceScopeUnavailable, "source-index is unavailable", nil)
	}
	requestedSnapshot := strings.TrimSpace(options.SnapshotID)
	requestedScope := strings.TrimSpace(options.ScopeID)
	if requestedSnapshot == "" && (requestedScope == "" || strings.EqualFold(requestedScope, "all")) && service.Index.Projection != nil {
		return service.Index.Projection, nil
	}
	var match *analysis.SourceIndexSnapshot
	for index := range service.Index.Snapshots {
		candidate := &service.Index.Snapshots[index]
		if requestedSnapshot != "" && requestedSnapshot != candidate.SnapshotID {
			continue
		}
		if requestedScope != "" && !strings.EqualFold(requestedScope, "all") && requestedScope != candidate.ScopeContext.ScopeID {
			continue
		}
		if match != nil {
			return nil, analysis.NewHostError(analysis.ErrSourceScopeUnavailable, "source-index selector matches more than one snapshot", map[string]any{"scope_id": requestedScope, "snapshot_id": requestedSnapshot})
		}
		match = candidate
	}
	if match == nil {
		if requestedSnapshot == "" && requestedScope == "" && len(service.Index.Snapshots) == 1 {
			return &service.Index.Snapshots[0], nil
		}
		return nil, analysis.NewHostError(analysis.ErrAnalysisScopeNotFound, "source-index scope or snapshot was not found", map[string]any{"scope_id": requestedScope, "snapshot_id": requestedSnapshot})
	}
	return match, nil
}

func (service *QueryService) pageBounds(options QueryOptions, total int) (int, int, error) {
	limit := options.Limit
	if limit == 0 {
		limit = DefaultQueryLimit
	}
	maxLimit := service.MaxLimit
	if maxLimit <= 0 || maxLimit > MaxQueryLimit {
		maxLimit = MaxQueryLimit
	}
	if limit < 1 || limit > maxLimit {
		return 0, 0, analysis.NewHostError(analysis.ErrInvalidRequest, "source-index query limit is outside the allowed bound", map[string]any{"limit": limit, "max": maxLimit})
	}
	start, err := decodeCursor(options.Cursor)
	if err != nil {
		return 0, 0, err
	}
	if start < 0 || start > total {
		return 0, 0, analysis.NewHostError(analysis.ErrInvalidRequest, "source-index query cursor is outside the result set", map[string]any{"cursor": options.Cursor})
	}
	end := start + limit
	if end > total {
		end = total
	}
	return start, end, nil
}

func (service *QueryService) FindFiles(options QueryOptions) (Page[analysis.FileRecord], error) {
	snapshot, err := service.snapshot(options)
	if err != nil {
		return Page[analysis.FileRecord]{}, err
	}
	allowedFiles := explicitFileIDs(snapshot, options)
	items := make([]analysis.FileRecord, 0)
	for _, file := range snapshot.Files {
		if allowedFiles != nil {
			if _, ok := allowedFiles[file.ID]; !ok {
				continue
			}
		}
		if !matchFile(file, options) {
			continue
		}
		items = append(items, file)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Path == items[j].Path {
			return items[i].ID < items[j].ID
		}
		return items[i].Path < items[j].Path
	})
	start, end, err := service.pageBounds(options, len(items))
	if err != nil {
		return Page[analysis.FileRecord]{}, err
	}
	return Page[analysis.FileRecord]{SnapshotID: snapshot.SnapshotID, ScopeID: snapshot.ScopeContext.ScopeID, Items: items[start:end], Total: len(items), NextCursor: nextCursor(end, len(items)), Coverage: pageCoverage(snapshot)}, nil
}

func (service *QueryService) FindSymbols(options QueryOptions) (Page[analysis.SymbolRecord], error) {
	snapshot, err := service.snapshot(options)
	if err != nil {
		return Page[analysis.SymbolRecord]{}, err
	}
	pathByFile := make(map[string]string, len(snapshot.Files))
	for _, file := range snapshot.Files {
		pathByFile[file.ID] = file.Path
	}
	allowedSymbols := explicitSymbolIDs(snapshot, options)
	items := make([]analysis.SymbolRecord, 0)
	for _, symbol := range snapshot.Symbols {
		if allowedSymbols != nil {
			if _, ok := allowedSymbols[symbol.ID]; !ok {
				continue
			}
		}
		if !matchSymbol(symbol, options) {
			continue
		}
		if options.PathPrefix != "" || options.PathGlob != "" {
			if !symbolMatchesPath(symbol, pathByFile, options) {
				continue
			}
		}
		items = append(items, symbol)
	}
	sort.Slice(items, func(i, j int) bool { return symbolSortKey(items[i], pathByFile) < symbolSortKey(items[j], pathByFile) })
	start, end, err := service.pageBounds(options, len(items))
	if err != nil {
		return Page[analysis.SymbolRecord]{}, err
	}
	return Page[analysis.SymbolRecord]{SnapshotID: snapshot.SnapshotID, ScopeID: snapshot.ScopeContext.ScopeID, Items: items[start:end], Total: len(items), NextCursor: nextCursor(end, len(items)), Coverage: pageCoverage(snapshot)}, nil
}

func (service *QueryService) FindDocumentation(options QueryOptions) (Page[analysis.DocumentationRecord], error) {
	snapshot, err := service.snapshot(options)
	if err != nil {
		return Page[analysis.DocumentationRecord]{}, err
	}
	allowedFiles, allowedSymbols := explicitSubjectIDs(snapshot, options)
	items := make([]analysis.DocumentationRecord, 0)
	for _, documentation := range snapshot.Documentation {
		if allowedFiles != nil || allowedSymbols != nil {
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
		if !matchDocumentation(documentation, options) {
			continue
		}
		value := queryDocumentation(documentation, options)
		items = append(items, value)
	}
	start, end, err := service.pageBounds(options, len(items))
	if err != nil {
		return Page[analysis.DocumentationRecord]{}, err
	}
	return Page[analysis.DocumentationRecord]{SnapshotID: snapshot.SnapshotID, ScopeID: snapshot.ScopeContext.ScopeID, Items: items[start:end], Total: len(items), NextCursor: nextCursor(end, len(items)), Coverage: pageCoverage(snapshot)}, nil
}

func pageCoverage(snapshot *analysis.SourceIndexSnapshot) []analysis.CoverageRecord {
	if snapshot == nil || len(snapshot.Coverage) == 0 {
		return nil
	}
	return append([]analysis.CoverageRecord(nil), snapshot.Coverage...)
}

// explicitFileIDs resolves the request's membership boundary without using
// paths, names, or symbol locations as a substitute for containment.
// A non-nil empty map is intentional: an explicit unknown ID must return an
// empty page rather than broadening to every source fact.
func explicitFileIDs(snapshot *analysis.SourceIndexSnapshot, options QueryOptions) map[string]struct{} {
	if len(options.ModuleIDs) == 0 && len(options.FileIDs) == 0 {
		return nil
	}
	fileIDs := make(map[string]struct{})
	for _, fileID := range options.FileIDs {
		if value := strings.TrimSpace(fileID); value != "" {
			fileIDs[value] = struct{}{}
		}
	}
	requestedModules := uniqueSet(options.ModuleIDs)
	if len(requestedModules) > 0 {
		for _, relation := range snapshot.Relations {
			if relation.Category != analysis.RelationContains || relation.FromRef.Kind != "module" || relation.ToRef == nil || relation.ToRef.Kind != "file" {
				continue
			}
			if _, ok := requestedModules[relation.FromRef.ID]; ok {
				fileIDs[relation.ToRef.ID] = struct{}{}
			}
		}
	}
	return fileIDs
}

func explicitSymbolIDs(snapshot *analysis.SourceIndexSnapshot, options QueryOptions) map[string]struct{} {
	if len(options.ModuleIDs) == 0 && len(options.FileIDs) == 0 {
		return nil
	}
	allowedFiles := explicitFileIDs(snapshot, options)
	symbolIDs := make(map[string]struct{})
	for _, relation := range snapshot.Relations {
		if relation.Category != analysis.RelationContains && relation.Category != analysis.RelationDeclares {
			continue
		}
		if relation.FromRef.Kind != "file" || relation.ToRef == nil || relation.ToRef.Kind != "symbol" {
			continue
		}
		if _, ok := allowedFiles[relation.FromRef.ID]; ok {
			symbolIDs[relation.ToRef.ID] = struct{}{}
		}
	}
	return symbolIDs
}

func explicitSubjectIDs(snapshot *analysis.SourceIndexSnapshot, options QueryOptions) (map[string]struct{}, map[string]struct{}) {
	if len(options.ModuleIDs) == 0 && len(options.FileIDs) == 0 {
		return nil, nil
	}
	files := explicitFileIDs(snapshot, options)
	symbols := make(map[string]struct{})
	for _, relation := range snapshot.Relations {
		if relation.Category != analysis.RelationContains && relation.Category != analysis.RelationDeclares {
			continue
		}
		if relation.FromRef.Kind != "file" || relation.ToRef == nil || relation.ToRef.Kind != "symbol" {
			continue
		}
		if _, ok := files[relation.FromRef.ID]; ok {
			symbols[relation.ToRef.ID] = struct{}{}
		}
	}
	return files, symbols
}

func uniqueSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result[value] = struct{}{}
		}
	}
	return result
}

// InspectModule follows explicit module->file and file->symbol relations. It
// never treats a matching path or package name as implicit containment.
func (service *QueryService) InspectModule(moduleID string, options QueryOptions) (ModuleSourceFacts, error) {
	snapshot, err := service.snapshot(options)
	if err != nil {
		return ModuleSourceFacts{}, err
	}
	moduleID = strings.TrimSpace(moduleID)
	if moduleID == "" {
		return ModuleSourceFacts{}, analysis.NewHostError(analysis.ErrInvalidRequest, "module id is required for source-fact inspection", nil)
	}
	fileIDs := make(map[string]struct{})
	for _, relation := range snapshot.Relations {
		if relation.Category != analysis.RelationContains || relation.FromRef.Kind != "module" || relation.FromRef.ID != moduleID || relation.ToRef == nil || relation.ToRef.Kind != "file" {
			continue
		}
		fileIDs[relation.ToRef.ID] = struct{}{}
	}
	files := make([]analysis.FileRecord, 0, len(fileIDs))
	for _, file := range snapshot.Files {
		if _, ok := fileIDs[file.ID]; ok {
			files = append(files, file)
		}
	}
	symbolIDs := make(map[string]struct{})
	for _, relation := range snapshot.Relations {
		if relation.Category != analysis.RelationContains && relation.Category != analysis.RelationDeclares {
			continue
		}
		if relation.FromRef.Kind != "file" || relation.ToRef == nil || relation.ToRef.Kind != "symbol" {
			continue
		}
		if _, ok := fileIDs[relation.FromRef.ID]; ok {
			symbolIDs[relation.ToRef.ID] = struct{}{}
		}
	}
	symbols := make([]analysis.SymbolRecord, 0, len(symbolIDs))
	for _, symbol := range snapshot.Symbols {
		if _, ok := symbolIDs[symbol.ID]; ok {
			symbols = append(symbols, symbol)
		}
	}
	documentation := make([]analysis.DocumentationRecord, 0)
	for _, value := range snapshot.Documentation {
		switch value.SubjectRef.Kind {
		case "file":
			if _, ok := fileIDs[value.SubjectRef.ID]; ok {
				documentation = append(documentation, queryDocumentation(value, options))
			}
		case "symbol":
			if _, ok := symbolIDs[value.SubjectRef.ID]; ok {
				documentation = append(documentation, queryDocumentation(value, options))
			}
		}
	}
	relations := make([]analysis.CodeRelation, 0)
	for _, relation := range snapshot.Relations {
		if relation.FromRef.Kind == "module" && relation.FromRef.ID == moduleID {
			relations = append(relations, relation)
			continue
		}
		if relation.FromRef.Kind == "file" {
			if _, ok := fileIDs[relation.FromRef.ID]; ok {
				relations = append(relations, relation)
			}
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return ModuleSourceFacts{SnapshotID: snapshot.SnapshotID, ScopeID: snapshot.ScopeContext.ScopeID, ModuleID: moduleID, Files: files, Symbols: symbols, Documentation: documentation, Relations: relations, Coverage: append([]analysis.CoverageRecord(nil), snapshot.Coverage...)}, nil
}

func (service *QueryService) GetEvidence(entityID string, options QueryOptions) (SourceFactEvidence, error) {
	snapshot, err := service.snapshot(options)
	if err != nil {
		return SourceFactEvidence{}, err
	}
	entityID = strings.TrimSpace(entityID)
	if entityID == "" {
		return SourceFactEvidence{}, analysis.NewHostError(analysis.ErrInvalidRequest, "source-fact evidence entity id is required", nil)
	}
	result := SourceFactEvidence{SnapshotID: snapshot.SnapshotID, ScopeID: snapshot.ScopeContext.ScopeID, EntityID: entityID, Spans: []analysis.SourceSpan{}, SourceReferenceIDs: []string{}, Documentation: []analysis.DocumentationRecord{}, Relations: []analysis.CodeRelation{}}
	for _, symbol := range snapshot.Symbols {
		if symbol.ID != entityID {
			continue
		}
		for _, location := range symbol.Locations {
			result.Spans = append(result.Spans, location.Span)
			result.SourceReferenceIDs = append(result.SourceReferenceIDs, location.SourceReferenceIDs...)
		}
	}
	for _, documentation := range snapshot.Documentation {
		if documentation.ID == entityID {
			result.Spans = append(result.Spans, documentation.Spans...)
			result.SourceReferenceIDs = append(result.SourceReferenceIDs, documentation.SourceReferenceIDs...)
			result.Documentation = append(result.Documentation, queryDocumentation(documentation, options))
		}
		if documentation.SubjectRef.ID == entityID {
			result.Documentation = append(result.Documentation, queryDocumentation(documentation, options))
		}
	}
	for _, occurrence := range snapshot.Occurrences {
		if occurrence.ID == entityID {
			result.Spans = append(result.Spans, occurrence.SourceSpan)
		}
	}
	for _, relation := range snapshot.Relations {
		if relation.ID == entityID || relation.FromRef.ID == entityID || (relation.ToRef != nil && relation.ToRef.ID == entityID) {
			result.Relations = append(result.Relations, relation)
			result.Spans = append(result.Spans, relation.EvidenceSpans...)
		}
	}
	if len(result.Spans) == 0 && len(result.Documentation) == 0 && len(result.Relations) == 0 {
		return SourceFactEvidence{}, analysis.NewHostError(analysis.ErrSourceReferenceInvalid, "source-fact evidence entity was not found", map[string]any{"entity_id": entityID})
	}
	result.SourceReferenceIDs = uniqueSorted(result.SourceReferenceIDs)
	sort.Slice(result.Spans, func(i, j int) bool { return spanSortKey(result.Spans[i]) < spanSortKey(result.Spans[j]) })
	return result, nil
}

func queryDocumentation(value analysis.DocumentationRecord, options QueryOptions) analysis.DocumentationRecord {
	if options.IncludeDocumentationText {
		return value
	}
	value.RawText = ""
	runes := []rune(value.NormalizedText)
	if len(runes) > CompactDocumentationRuneLimit {
		value.NormalizedText = string(runes[:CompactDocumentationRuneLimit])
		value.Completeness = analysis.DocumentationTruncated
	}
	return value
}

func validateQueryOptions(options QueryOptions) error {
	if options.PathGlob == "" {
		return nil
	}
	if _, err := path.Match(options.PathGlob, "source-index-validation-probe"); err != nil {
		return analysis.NewHostError(analysis.ErrInvalidRequest, "source-index path glob is malformed", map[string]any{"path_glob": options.PathGlob})
	}
	return nil
}

func matchFile(file analysis.FileRecord, options QueryOptions) bool {
	if options.PathPrefix != "" && !matchesPrefix(file.Path, options.PathPrefix, options.CaseSensitive) {
		return false
	}
	if options.PathGlob != "" {
		matched, err := path.Match(options.PathGlob, file.Path)
		if err != nil || !matched {
			return false
		}
	}
	if options.Language != "" && !matchesText(file.Language.ID, options.Language, options.CaseSensitive) {
		return false
	}
	if options.Role != "" && !containsMatch(file.Roles, options.Role, options.CaseSensitive) {
		return false
	}
	return options.Status == "" || matchesText(file.AnalysisStatus, options.Status, options.CaseSensitive)
}

func matchSymbol(symbol analysis.SymbolRecord, options QueryOptions) bool {
	return (options.Name == "" || matchesText(symbol.Name, options.Name, options.CaseSensitive)) &&
		(options.QualifiedName == "" || matchesText(symbol.QualifiedName, options.QualifiedName, options.CaseSensitive)) &&
		(options.Category == "" || matchesText(symbol.Category, options.Category, options.CaseSensitive)) &&
		(options.LanguageKind == "" || matchesText(symbol.LanguageKind, options.LanguageKind, options.CaseSensitive)) &&
		(options.Status == "" || matchesText(symbol.Provenance.Status, options.Status, options.CaseSensitive))
}

func matchDocumentation(value analysis.DocumentationRecord, options QueryOptions) bool {
	if options.SubjectID != "" && value.SubjectRef.ID != options.SubjectID {
		return false
	}
	if len(options.SubjectIDs) > 0 {
		allowed := uniqueSet(options.SubjectIDs)
		if _, ok := allowed[value.SubjectRef.ID]; !ok {
			return false
		}
	}
	if options.Status != "" && !matchesText(value.Status, options.Status, options.CaseSensitive) {
		return false
	}
	if options.DocumentationText != "" {
		text := value.NormalizedText
		if text == "" {
			text = value.RawText
		}
		if !matchesText(text, options.DocumentationText, options.CaseSensitive) {
			return false
		}
	}
	return true
}

func symbolMatchesPath(symbol analysis.SymbolRecord, paths map[string]string, options QueryOptions) bool {
	for _, location := range symbol.Locations {
		filePath := paths[location.Span.FileID]
		if options.PathPrefix != "" && !matchesPrefix(filePath, options.PathPrefix, options.CaseSensitive) {
			continue
		}
		if options.PathGlob != "" {
			matched, err := path.Match(options.PathGlob, filePath)
			if err != nil || !matched {
				continue
			}
		}
		return true
	}
	return false
}

func symbolSortKey(symbol analysis.SymbolRecord, paths map[string]string) string {
	pathValue := ""
	start := 0
	if len(symbol.Locations) > 0 {
		pathValue = paths[symbol.Locations[0].Span.FileID]
		start = symbol.Locations[0].Span.Start.ByteOffset
		for _, location := range symbol.Locations[1:] {
			candidatePath := paths[location.Span.FileID]
			if candidatePath < pathValue || candidatePath == pathValue && location.Span.Start.ByteOffset < start {
				pathValue = candidatePath
				start = location.Span.Start.ByteOffset
			}
		}
	}
	return fmt.Sprintf("%s\x00%012d\x00%s\x00%s\x00%s\x00%s", pathValue, start, symbol.Category, symbol.LanguageKind, symbol.Name, symbol.ID)
}

func spanSortKey(span analysis.SourceSpan) string {
	return fmt.Sprintf("%s\x00%012d\x00%012d", span.FileID, span.Start.ByteOffset, span.End.ByteOffset)
}

func matchesText(value, expected string, caseSensitive bool) bool {
	if caseSensitive {
		return strings.Contains(value, expected)
	}
	return strings.Contains(strings.ToLower(value), strings.ToLower(expected))
}

func matchesPrefix(value, expected string, caseSensitive bool) bool {
	if caseSensitive {
		return strings.HasPrefix(value, expected)
	}
	return strings.HasPrefix(strings.ToLower(value), strings.ToLower(expected))
}

func containsMatch(values []string, expected string, caseSensitive bool) bool {
	for _, value := range values {
		if (caseSensitive && value == expected) || (!caseSensitive && strings.EqualFold(value, expected)) {
			return true
		}
	}
	return false
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
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

func decodeCursor(value string) (int, error) {
	if value == "" {
		return 0, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return 0, analysis.NewHostError(analysis.ErrInvalidRequest, "source-index query cursor is malformed", map[string]any{"cursor": value})
	}
	if !strings.HasPrefix(string(decoded), "v1:") {
		return 0, analysis.NewHostError(analysis.ErrInvalidRequest, "source-index query cursor version is unsupported", nil)
	}
	position, err := strconv.Atoi(strings.TrimPrefix(string(decoded), "v1:"))
	if err != nil || position < 0 {
		return 0, analysis.NewHostError(analysis.ErrInvalidRequest, "source-index query cursor position is invalid", nil)
	}
	return position, nil
}

func nextCursor(end, total int) string {
	if end >= total {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString([]byte("v1:" + strconv.Itoa(end)))
}
