package viewer

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/orchestration"
	"github.com/buffo/arch-view/internal/analysis/sourceindex"
	"github.com/buffo/arch-view/internal/model"
)

func (s *Server) handleSourceIndex(writer http.ResponseWriter, request *http.Request, parts []string, modelID string, value *model.Model, aggregate *orchestration.AnalysisRun) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}
	index, err := sourceIndexForRequest(value, aggregate, request.URL.Query())
	if err != nil {
		writeHTTPError(writer, sourceIndexHTTPStatus(err), err)
		return
	}
	if len(parts) == 2 {
		writeJSON(writer, http.StatusOK, index)
		return
	}
	options, err := sourceIndexQueryOptions(request.URL.Query())
	if err != nil {
		writeHTTPError(writer, http.StatusBadRequest, err)
		return
	}
	// A concrete aggregate scope has already selected a single cached model
	// above. Its local source snapshot may use an analyzer-local identity, so
	// do not ask the query service to resolve the job scope a second time.
	if aggregate != nil {
		scope := strings.TrimSpace(request.URL.Query().Get("scope"))
		if scope != "" && !scopeEqualAll(scope) {
			options.ScopeID = ""
		}
	}
	service := sourceindex.NewQueryService(index)
	switch parts[2] {
	case "files":
		result, queryErr := service.FindFiles(options)
		if queryErr != nil {
			writeHTTPError(writer, sourceIndexHTTPStatus(queryErr), queryErr)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	case "symbols":
		result, queryErr := service.FindSymbols(options)
		if queryErr != nil {
			writeHTTPError(writer, sourceIndexHTTPStatus(queryErr), queryErr)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	case "documentation":
		result, queryErr := service.FindDocumentation(options)
		if queryErr != nil {
			writeHTTPError(writer, sourceIndexHTTPStatus(queryErr), queryErr)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	case "evidence":
		if len(parts) != 4 || strings.TrimSpace(parts[3]) == "" {
			writeHTTPError(writer, http.StatusBadRequest, analysis.NewHostError(analysis.ErrInvalidRequest, "source-fact evidence requires an entity id", nil))
			return
		}
		entityID, unescapeErr := url.PathUnescape(parts[3])
		if unescapeErr != nil {
			writeHTTPError(writer, http.StatusBadRequest, analysis.NewHostError(analysis.ErrInvalidRequest, "source-fact evidence entity id is malformed", nil))
			return
		}
		result, queryErr := service.GetEvidence(entityID, options)
		if queryErr != nil {
			writeHTTPError(writer, sourceIndexHTTPStatus(queryErr), queryErr)
			return
		}
		response := sourceFactEvidenceResponse{SourceFactEvidence: result}
		if value := request.URL.Query().Get("include_source"); value != "" {
			includeSource, parseErr := strconv.ParseBool(value)
			if parseErr != nil {
				writeHTTPError(writer, http.StatusBadRequest, analysis.NewHostError(analysis.ErrInvalidRequest, "include_source must be a boolean", map[string]any{"include_source": value}))
				return
			}
			if includeSource {
				excerpt, excerptErr := s.sourceFactExcerpt(index, result, modelID)
				if excerptErr != nil {
					writeHTTPError(writer, sourceIndexHTTPStatus(excerptErr), excerptErr)
					return
				}
				response.SourceContext = excerpt
			}
		}
		writeJSON(writer, http.StatusOK, response)
	default:
		http.NotFound(writer, request)
	}
}

func sourceIndexForRequest(value *model.Model, aggregate *orchestration.AnalysisRun, query url.Values) (*analysis.SourceIndex, error) {
	if aggregate == nil {
		if value == nil || value.SourceIndex == nil {
			return nil, analysis.NewHostError(analysis.ErrSourceScopeUnavailable, "source-index is unavailable for this model revision", nil)
		}
		return value.SourceIndex, nil
	}
	scope := strings.TrimSpace(query.Get("scope"))
	if scope != "" && !scopeEqualAll(scope) {
		selected, err := aggregate.SelectAnalysisScope(scope)
		if err != nil {
			return nil, err
		}
		if selected.Model.SourceIndex == nil {
			return nil, analysis.NewHostError(analysis.ErrSourceScopeUnavailable, "source-index is unavailable for the selected analysis scope", map[string]any{"scope_id": scope})
		}
		return selected.Model.SourceIndex, nil
	}
	combined, ok := aggregate.CombinedCanonicalModel()
	if !ok || combined.SourceIndex == nil {
		return nil, analysis.NewHostError(analysis.ErrSourceScopeUnavailable, "the aggregate run has no source-index projection", map[string]any{"run_id": aggregate.RunID})
	}
	return combined.SourceIndex, nil
}

func sourceIndexQueryOptions(query url.Values) (sourceindex.QueryOptions, error) {
	subjectIDs := queryStringValues(query, "subject_id")
	subjectID := ""
	if len(subjectIDs) == 1 {
		subjectID = subjectIDs[0]
	}
	options := sourceindex.QueryOptions{
		SnapshotID:               query.Get("snapshot_id"),
		ScopeID:                  query.Get("scope"),
		ModuleIDs:                queryStringValues(query, "module_id"),
		FileIDs:                  queryStringValues(query, "file_id"),
		PathPrefix:               query.Get("path_prefix"),
		PathGlob:                 query.Get("path_glob"),
		Language:                 query.Get("language"),
		Role:                     query.Get("role"),
		Status:                   query.Get("status"),
		Name:                     query.Get("name"),
		QualifiedName:            query.Get("qualified_name"),
		Category:                 query.Get("category"),
		LanguageKind:             query.Get("language_kind"),
		DocumentationText:        query.Get("documentation_text"),
		SubjectID:                subjectID,
		SubjectIDs:               subjectIDs,
		Cursor:                   query.Get("cursor"),
		IncludeDocumentationText: query.Get("include_documentation_text") == "true",
	}
	if value := query.Get("case_sensitive"); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return sourceindex.QueryOptions{}, analysis.NewHostError(analysis.ErrInvalidRequest, "case_sensitive must be a boolean", map[string]any{"case_sensitive": value})
		}
		options.CaseSensitive = parsed
	}
	if value := query.Get("include_documentation_text"); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return sourceindex.QueryOptions{}, analysis.NewHostError(analysis.ErrInvalidRequest, "include_documentation_text must be a boolean", map[string]any{"include_documentation_text": value})
		}
		options.IncludeDocumentationText = parsed
	}
	if value := query.Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return sourceindex.QueryOptions{}, analysis.NewHostError(analysis.ErrInvalidRequest, "limit must be an integer", map[string]any{"limit": value})
		}
		options.Limit = parsed
	}
	return options, nil
}

func queryStringValues(query url.Values, key string) []string {
	values := make([]string, 0, len(query[key]))
	seen := make(map[string]struct{}, len(query[key]))
	for _, value := range query[key] {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return values
}

func sourceIndexHTTPStatus(err error) int {
	switch analysis.ErrorCodeOf(err) {
	case analysis.ErrInvalidRequest:
		return http.StatusBadRequest
	case analysis.ErrAnalysisScopeNotFound, analysis.ErrSourceScopeUnavailable, analysis.ErrSourceReferenceInvalid:
		return http.StatusNotFound
	default:
		return http.StatusUnprocessableEntity
	}
}

type sourceFactEvidenceResponse struct {
	sourceindex.SourceFactEvidence
	SourceContext *SourceExcerpt `json:"source_context,omitempty"`
}

func (s *Server) sourceFactExcerpt(index *analysis.SourceIndex, evidence sourceindex.SourceFactEvidence, modelID string) (*SourceExcerpt, error) {
	return s.sourceFactExcerptWithBudget(index, evidence, modelID, maxSourceSpan, maxSourceBytes)
}

func (s *Server) sourceFactExcerptWithBudget(index *analysis.SourceIndex, evidence sourceindex.SourceFactEvidence, modelID string, maxLines, maxBytes int) (*SourceExcerpt, error) {
	if index == nil {
		return nil, analysis.NewHostError(analysis.ErrSourceScopeUnavailable, "source-index is unavailable", nil)
	}
	if s.getSourceRoot() == "" {
		return nil, analysis.NewHostError(analysis.ErrInvalidRequest, "source inspection is unavailable for a model-only session", nil)
	}
	if len(evidence.Spans) == 0 {
		return nil, analysis.NewHostError(analysis.ErrSourceReferenceInvalid, "source-fact evidence has no source span", map[string]any{"entity_id": evidence.EntityID})
	}
	if maxLines <= 0 || maxLines > maxSourceSpan {
		return nil, analysis.NewHostError(analysis.ErrInvalidRequest, "source-fact evidence line budget exceeds the supported bound", map[string]any{"max_lines": maxSourceSpan})
	}
	if maxBytes <= 0 || maxBytes > maxSourceBytes {
		return nil, analysis.NewHostError(analysis.ErrInvalidRequest, "source-fact evidence byte budget exceeds the supported bound", map[string]any{"max_bytes": maxSourceBytes})
	}
	span := evidence.Spans[0]
	var file *analysis.FileRecord
	var snapshot *analysis.SourceIndexSnapshot
	for snapshotIndex := range index.Snapshots {
		candidate := &index.Snapshots[snapshotIndex]
		for fileIndex := range candidate.Files {
			if candidate.Files[fileIndex].ID == span.FileID {
				file = &candidate.Files[fileIndex]
				snapshot = candidate
				break
			}
		}
		if file != nil {
			break
		}
	}
	if file == nil && index.Projection != nil {
		for fileIndex := range index.Projection.Files {
			if index.Projection.Files[fileIndex].ID == span.FileID {
				file = &index.Projection.Files[fileIndex]
				snapshot = index.Projection
				break
			}
		}
	}
	if file == nil {
		return nil, analysis.NewHostError(analysis.ErrSourceReferenceInvalid, "source-fact span file was not found", map[string]any{"file_id": span.FileID})
	}
	relativePath := file.Path
	if snapshot != nil {
		relativePath = repositorySourcePath(snapshot.ScopeContext.ProjectRoot, relativePath)
	}
	relativePath, err := safeRepositoryPath(relativePath)
	if err != nil {
		return nil, err
	}
	filePath, err := containedFilePath(s.getSourceRoot(), relativePath)
	if err != nil {
		return nil, err
	}
	content, err := readSourceFile(filePath)
	if err != nil {
		return nil, err
	}
	if len(content) > maxBytes {
		return nil, analysis.NewHostError(analysis.ErrInvalidRequest, "source-fact evidence source context exceeds the requested byte budget", map[string]any{"max_bytes": maxBytes})
	}
	digest := sha256.Sum256(content)
	actualDigest := hex.EncodeToString(digest[:])
	if file.Size.ContentHash.Algorithm != analysis.SourceHashAlgorithm || file.Size.ContentHash.Value != actualDigest || span.ContentHash.Algorithm != analysis.SourceHashAlgorithm || span.ContentHash.Value != actualDigest {
		return nil, analysis.NewHostError(analysis.ErrSourceIndexDigestMismatch, "source-fact evidence no longer matches the indexed source file", map[string]any{"path": relativePath, "expected": span.ContentHash.Value, "actual": actualDigest})
	}
	lines := splitPhysicalSourceLines(content)
	startLine := span.Start.Line
	endLine := span.End.Line
	if startLine < 1 || endLine < startLine || endLine-startLine+1 > maxLines {
		return nil, analysis.NewHostError(analysis.ErrInvalidRequest, "source-fact evidence span exceeds the requested source inspection bound", map[string]any{"max_lines": maxLines})
	}
	if startLine > len(lines) {
		return nil, analysis.NewHostError(analysis.ErrInvalidRequest, "source-fact evidence span is beyond the source file", map[string]any{"path": relativePath})
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}
	excerptLines := make([]SourceLine, 0, endLine-startLine+1)
	for lineNumber := startLine; lineNumber <= endLine; lineNumber++ {
		excerptLines = append(excerptLines, SourceLine{Number: lineNumber, Text: lines[lineNumber-1]})
	}
	return &SourceExcerpt{
		SchemaVersion:     sourceSchemaVersion,
		ModelID:           modelID,
		ModelRevision:     modelID,
		SourceReferenceID: evidence.EntityID,
		Path:              relativePath,
		Start:             &analysis.Position{Line: span.Start.Line, Column: span.Start.Column},
		End:               &analysis.Position{Line: span.End.Line, Column: span.End.Column},
		Lines:             excerptLines,
		ReadOnly:          true,
	}, nil
}

func splitPhysicalSourceLines(content []byte) []string {
	lines := make([]string, 0, 1)
	start := 0
	for index := 0; index < len(content); index++ {
		if content[index] != '\n' && content[index] != '\r' {
			continue
		}
		lines = append(lines, string(content[start:index]))
		if content[index] == '\r' && index+1 < len(content) && content[index+1] == '\n' {
			index++
		}
		start = index + 1
	}
	lines = append(lines, string(content[start:]))
	return lines
}
