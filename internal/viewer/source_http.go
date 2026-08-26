package viewer

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
)

func (s *Server) handleSource(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}
	value := s.snapshot()
	query := request.URL.Query()
	requestedModelID := query.Get("model_id")
	if requestedModelID == "" || requestedModelID != value.ModelID {
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

	source, err := sourceForQuery(value, query)
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
		ModelID:           value.ModelID,
		ModelRevision:     value.ModelID,
		SourceReferenceID: source.ID,
		Path:              relativePath,
		Start:             start,
		End:               end,
		Lines:             excerptLines,
		ReadOnly:          true,
	})
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
