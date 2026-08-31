package viewer

import (
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/orchestration"
	"github.com/buffo/arch-view/internal/model"
)

func (s *Server) handleSource(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}
	query := request.URL.Query()
	requestedModelID := query.Get("model_id")
	aggregate := s.aggregateSnapshot()
	value, snapshotErr := s.snapshotForRequest(request)
	if snapshotErr != nil {
		writeHTTPError(writer, liveViewerHTTPStatus(snapshotErr), snapshotErr)
		return
	}
	responseModelID := value.ModelID
	if aggregate != nil {
		responseModelID = s.modelID()
		if requestedModelID == "" || requestedModelID != responseModelID {
			writeHTTPError(writer, http.StatusNotFound, analysis.NewHostError(analysis.ErrInvalidModel, "source evidence belongs to a different aggregate revision", map[string]any{"model_id": requestedModelID}))
			return
		}
		var err error
		value, err = aggregateSourceModel(*aggregate, query)
		if err != nil {
			writeHTTPError(writer, sourceErrorStatus(err), err)
			return
		}
	} else if requestedModelID == "" || requestedModelID != value.ModelID && (s.liveSession == nil || requestedModelID != s.liveSession.Config().SessionID) {
		writeHTTPError(writer, http.StatusNotFound, analysis.NewHostError(analysis.ErrInvalidModel, "source evidence belongs to a different model revision", map[string]any{"model_id": requestedModelID}))
		return
	}
	sourceRoot := s.getSourceRoot()
	if sourceRoot == "" {
		writeHTTPError(writer, http.StatusForbidden, analysis.NewHostError(analysis.ErrInvalidRequest, "source inspection is unavailable for a model-only session", nil))
		return
	}
	if requestedPath := query.Get("path"); requestedPath != "" {
		if _, err := safeRepositoryPath(requestedPath); err != nil {
			writeHTTPError(writer, http.StatusForbidden, err)
			return
		}
	}

	sourceQuery := query
	if aggregate != nil && query.Get("scope") != "" && !scopeEqualAll(query.Get("scope")) {
		sourceQuery = cloneURLValues(query)
		if query.Get("evidence_id") != "" {
			// Individual scene evidence keeps its local analyzer ID and path;
			// source lookup uses the cached scope model, whose path is promoted
			// to the repository root below.
			sourceQuery.Del("path")
		} else if requestedPath := query.Get("path"); requestedPath != "" {
			if summary, ok := aggregateScopeSummary(*aggregate, query.Get("scope")); ok {
				sourceQuery.Set("path", repositorySourcePath(summary.ProjectRoot, requestedPath))
			}
		}
	}
	source, err := sourceForQuery(value, sourceQuery)
	if err != nil {
		writeHTTPError(writer, sourceErrorStatus(err), err)
		return
	}
	relativePath, err := safeRepositoryPath(source.Path)
	if err != nil {
		writeHTTPError(writer, http.StatusForbidden, err)
		return
	}
	filePath, err := containedFilePath(sourceRoot, relativePath)
	if err != nil {
		writeHTTPError(writer, sourceErrorStatus(err), err)
		return
	}
	content, err := readSourceFile(filePath)
	if err != nil {
		writeHTTPError(writer, sourceErrorStatus(err), err)
		return
	}
	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	startLine, endLine, err := sourceLineRange(query, source, len(lines))
	if err != nil {
		writeHTTPError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	if startLine > len(lines) {
		writeHTTPError(writer, http.StatusNotFound, analysis.NewHostError(analysis.ErrInvalidRequest, "source location is beyond the end of the file", map[string]any{"path": relativePath, "start_line": startLine}))
		return
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}
	excerptLines := make([]SourceLine, 0, endLine-startLine+1)
	for lineNumber := startLine; lineNumber <= endLine; lineNumber++ {
		excerptLines = append(excerptLines, SourceLine{Number: lineNumber, Text: strings.TrimSuffix(lines[lineNumber-1], "\r")})
	}
	start := &analysis.Position{Line: startLine, Column: 1}
	end := &analysis.Position{Line: endLine, Column: 1}
	if source.Start != nil {
		start.Column = source.Start.Column
	}
	if source.End != nil {
		end.Column = source.End.Column
	}
	writeJSON(writer, http.StatusOK, SourceExcerpt{
		SchemaVersion:     sourceSchemaVersion,
		ModelID:           responseModelID,
		ModelRevision:     responseModelID,
		SourceReferenceID: source.ID,
		Path:              relativePath,
		Start:             start,
		End:               end,
		Lines:             excerptLines,
		ReadOnly:          true,
	})
}

func aggregateSourceModel(run orchestration.AnalysisRun, query url.Values) (model.Model, error) {
	if scope := strings.TrimSpace(query.Get("scope")); scope != "" && !scopeEqualAll(scope) {
		selected, err := run.SelectAnalysisScope(scope)
		if err != nil {
			return model.Model{}, err
		}
		if selected.Model.ModelID == "" {
			return model.Model{}, analysis.NewHostError(analysis.ErrInvalidModel, "source evidence is unavailable for the selected analysis scope", map[string]any{"scope_id": scope})
		}
		value := selected.Model
		value.SourceReferences = append([]analysis.SourceReference(nil), value.SourceReferences...)
		for index := range value.SourceReferences {
			value.SourceReferences[index].Path = repositorySourcePath(selected.Summary.ProjectRoot, value.SourceReferences[index].Path)
		}
		return value, nil
	}
	combined, combinedOK := run.CombinedCanonicalModel()
	evidenceID := query.Get("evidence_id")
	requestedPath := query.Get("path")
	if combinedOK && sourceModelContains(combined, evidenceID, requestedPath) {
		return combined, nil
	}
	for _, summary := range run.Scopes {
		value, err := run.ScopeResult(summary.ScopeID)
		if err != nil {
			continue
		}
		if sourceModelContains(valueToModel(value), evidenceID, requestedPath) {
			return valueToModel(value), nil
		}
	}
	if combinedOK {
		return combined, nil
	}
	return model.Model{}, analysis.NewHostError(analysis.ErrInvalidModel, "source evidence was not found in the aggregate run", map[string]any{"evidence_id": evidenceID, "path": requestedPath})
}

func aggregateScopeSummary(run orchestration.AnalysisRun, scope string) (orchestration.ScopeSummary, bool) {
	for _, summary := range run.Scopes {
		if summary.ScopeID == scope {
			return summary, true
		}
	}
	return orchestration.ScopeSummary{}, false
}

func repositorySourcePath(root, local string) string {
	root = strings.TrimPrefix(strings.ReplaceAll(root, "\\", "/"), "./")
	local = strings.TrimPrefix(strings.ReplaceAll(local, "\\", "/"), "./")
	if root == "" || root == "." {
		return path.Clean(local)
	}
	if local == "" || local == "." {
		return path.Clean(root)
	}
	if local == root || strings.HasPrefix(local, root+"/") {
		return path.Clean(local)
	}
	return path.Clean(root + "/" + local)
}

func cloneURLValues(values url.Values) url.Values {
	result := make(url.Values, len(values))
	for key, items := range values {
		result[key] = append([]string(nil), items...)
	}
	return result
}

func valueToModel(value analysis.AnalysisResult) model.Model {
	// Aggregate source lookup only needs the source-reference collection. This
	// lightweight model avoids exposing analyzer results as a public viewer
	// response while keeping the existing sourceForQuery matching semantics.
	return model.Model{SourceReferences: value.SourceReferences}
}

func sourceModelContains(value model.Model, evidenceID, requestedPath string) bool {
	for _, source := range value.SourceReferences {
		if evidenceID != "" && source.ID == evidenceID {
			return true
		}
		if evidenceID == "" && requestedPath != "" && source.Path == requestedPath {
			return true
		}
	}
	return false
}

type SourceLine struct {
	Number int    `json:"number"`
	Text   string `json:"text"`
}

type SourceExcerpt struct {
	SchemaVersion     string             `json:"schema_version"`
	ModelID           string             `json:"model_id"`
	ModelRevision     string             `json:"model_revision"`
	SourceReferenceID string             `json:"source_reference_id"`
	Path              string             `json:"path"`
	Start             *analysis.Position `json:"start"`
	End               *analysis.Position `json:"end"`
	Lines             []SourceLine       `json:"lines"`
	ReadOnly          bool               `json:"read_only"`
}

func sourceForQuery(value model.Model, query url.Values) (analysis.SourceReference, error) {
	evidenceID := query.Get("evidence_id")
	requestedPath := query.Get("path")
	var match *analysis.SourceReference
	for index := range value.SourceReferences {
		source := &value.SourceReferences[index]
		if evidenceID != "" && source.ID == evidenceID {
			match = source
			break
		}
		if evidenceID == "" && requestedPath != "" && source.Path == requestedPath && source.Kind != "file" {
			if match == nil || source.Start != nil {
				match = source
			}
		}
	}
	if evidenceID != "" && match == nil {
		return analysis.SourceReference{}, analysis.NewHostError(analysis.ErrInvalidModel, "source evidence was not found in the active model revision", map[string]any{"evidence_id": evidenceID})
	}
	if match == nil {
		for index := range value.SourceReferences {
			source := &value.SourceReferences[index]
			if requestedPath == source.Path {
				match = source
				break
			}
		}
	}
	if match == nil {
		return analysis.SourceReference{}, analysis.NewHostError(analysis.ErrInvalidModel, "source path is not present in the active model evidence", map[string]any{"path": requestedPath})
	}
	if requestedPath != "" && requestedPath != match.Path {
		return analysis.SourceReference{}, analysis.NewHostError(analysis.ErrInvalidRequest, "source path does not match the requested evidence", map[string]any{"path": requestedPath, "evidence_id": evidenceID})
	}
	return *match, nil
}

func sourceLineRange(query url.Values, source analysis.SourceReference, lineCount int) (int, int, error) {
	startLine := 0
	endLine := 0
	var err error
	if value := query.Get("start_line"); value != "" {
		startLine, err = strconv.Atoi(value)
		if err != nil || startLine < 1 {
			return 0, 0, analysis.NewHostError(analysis.ErrInvalidRequest, "start_line must be a positive integer", map[string]any{"start_line": value})
		}
	}
	if value := query.Get("end_line"); value != "" {
		endLine, err = strconv.Atoi(value)
		if err != nil || endLine < 1 {
			return 0, 0, analysis.NewHostError(analysis.ErrInvalidRequest, "end_line must be a positive integer", map[string]any{"end_line": value})
		}
	}
	if startLine == 0 && source.Start != nil {
		startLine = source.Start.Line
	}
	if endLine == 0 && source.End != nil {
		endLine = source.End.Line
	}
	if startLine == 0 {
		startLine = 1
	}
	if endLine == 0 {
		endLine = startLine + 18
	}
	if startLine < 1 || endLine < 1 || endLine < startLine {
		return 0, 0, analysis.NewHostError(analysis.ErrInvalidRequest, "source line range is invalid", map[string]any{"start_line": startLine, "end_line": endLine})
	}
	if endLine-startLine+1 > maxSourceSpan {
		return 0, 0, analysis.NewHostError(analysis.ErrInvalidRequest, "source excerpt is too large", map[string]any{"max_lines": maxSourceSpan})
	}
	if lineCount == 0 {
		return 0, 0, analysis.NewHostError(analysis.ErrUnreadableProject, "source file is empty", nil)
	}
	return startLine, endLine, nil
}
