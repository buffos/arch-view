package viewer

import (
	"context"
	"embed"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
)

//go:embed web/index.html web/styles.css web/app.js web/app/*.js web/layout_request.js web/graph_route.js web/vendor/elk.bundled.js web/vendor/elk-worker.min.js
var webFiles embed.FS

const (
	sourceSchemaVersion = "arch-view.source/v1"
	maxSourceBytes      = 4 * 1024 * 1024
	maxSourceSpan       = 120
)

// ReanalysisRequest is the small request envelope accepted by the local
// viewer. The callback is owned by the CLI/orchestration layer so the viewer
// does not know how a language analyzer is selected or executed.
type ReanalysisRequest struct {
	ProjectRoot string         `json:"project_root"`
	Language    string         `json:"language"`
	Options     map[string]any `json:"options"`
}

// ReanalyzeFunc lets the CLI connect the viewer's read-only reanalysis action
// to the analyzer host without introducing a viewer-to-language dependency.
type ReanalyzeFunc func(context.Context, ReanalysisRequest) (model.Model, error)

type ServerOptions struct {
	SourceRoot string
	Reanalyze  ReanalyzeFunc
}

type Server struct {
	mu         sync.RWMutex
	model      model.Model
	sourceRoot string
	reanalyze  ReanalyzeFunc
	layout     layoutSession
	handler    http.Handler
}

func NewServer(value model.Model, options ...ServerOptions) (*Server, error) {
	if err := model.Validate(value); err != nil {
		return nil, err
	}
	var option ServerOptions
	if len(options) > 0 {
		option = options[0]
	}
	sourceRoot, err := normalizeSourceRoot(option.SourceRoot)
	if err != nil {
		return nil, err
	}
	server := &Server{model: value, sourceRoot: sourceRoot, reanalyze: option.Reanalyze, layout: discoverLayoutSession(sourceRoot)}
	mux := http.NewServeMux()
	mux.HandleFunc("/", server.handleRoot)
	mux.HandleFunc("/assets/", server.handleAsset)
	mux.HandleFunc("/v1/layout/options", server.handleLayoutOptions)
	mux.HandleFunc("/v1/layout/config/save-as", server.handleLayoutConfigSaveAs)
	mux.HandleFunc("/v1/layout/config", server.handleLayoutConfig)
	mux.HandleFunc("/v1/layout/apply", server.handleLayoutApply)
	mux.HandleFunc("/v1/layout/reset", server.handleLayoutReset)
	mux.HandleFunc("/v1/source", server.handleSource)
	mux.HandleFunc("/v1/reanalysis", server.handleReanalysis)
	mux.HandleFunc("/v1/models/", server.handleModel)
	server.handler = mux
	return server, nil
}

func (s *Server) Handler() http.Handler {
	return s.handler
}

func (s *Server) snapshot() model.Model {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.model
}

func (s *Server) sourceEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sourceRoot != ""
}

func (s *Server) reanalysisEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.reanalyze != nil && s.sourceRoot != ""
}

func (s *Server) handleRoot(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path != "/" {
		http.NotFound(writer, request)
		return
	}
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}
	data, err := webFiles.ReadFile("web/index.html")
	if err != nil {
		writeHTTPError(writer, http.StatusInternalServerError, analysis.NewHostError(analysis.ErrHostFailure, "viewer application could not be loaded", nil))
		return
	}
	value := s.snapshot()
	content := strings.ReplaceAll(string(data), "__ARCH_VIEW_MODEL_ID__", html.EscapeString(value.ModelID))
	content = strings.ReplaceAll(content, "__ARCH_VIEW_SOURCE_ENABLED__", strconv.FormatBool(s.sourceEnabled()))
	content = strings.ReplaceAll(content, "__ARCH_VIEW_REANALYSIS_ENABLED__", strconv.FormatBool(s.reanalysisEnabled()))
	content = strings.ReplaceAll(content, "__ARCH_VIEW_WORKER_URL__", "/assets/vendor/elk-worker.min.js")
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	_, _ = io.WriteString(writer, content)
}

func (s *Server) handleAsset(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}
	name := strings.TrimPrefix(request.URL.Path, "/assets/")
	if !isViewerAsset(name) {
		http.NotFound(writer, request)
		return
	}
	data, err := webFiles.ReadFile("web/" + name)
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	contentType := "text/plain; charset=utf-8"
	switch name {
	case "styles.css":
		contentType = "text/css; charset=utf-8"
	default:
		contentType = "text/javascript; charset=utf-8"
	}
	writer.Header().Set("Content-Type", contentType)
	if name == "styles.css" || name == "app.js" || name == "layout_request.js" || name == "graph_route.js" || strings.HasPrefix(name, "app/") {
		writer.Header().Set("Cache-Control", "no-store")
	} else {
		writer.Header().Set("Cache-Control", "public, max-age=3600, immutable")
	}
	_, _ = writer.Write(data)
}

func isViewerAsset(name string) bool {
	switch name {
	case "styles.css", "app.js", "layout_request.js", "graph_route.js", "vendor/elk.bundled.js", "vendor/elk-worker.min.js":
		return true
	}
	return strings.HasPrefix(name, "app/") && path.Clean(name) == name && strings.HasSuffix(name, ".js") && !strings.Contains(name, "\\")
}

func (s *Server) handleModel(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}
	value := s.snapshot()
	trimmed := strings.TrimPrefix(request.URL.Path, "/v1/models/")
	parts := strings.Split(trimmed, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(writer, request)
		return
	}
	modelID, err := url.PathUnescape(parts[0])
	if err != nil || modelID != value.ModelID {
		writeHTTPError(writer, http.StatusNotFound, analysis.NewHostError(analysis.ErrInvalidModel, "requested model was not found", map[string]any{"model_id": modelID}))
		return
	}
	if len(parts) == 1 {
		writeJSON(writer, http.StatusOK, value)
		return
	}
	if len(parts) != 2 || parts[1] != "projection" {
		http.NotFound(writer, request)
		return
	}
	selectedPath, err := queryHierarchyPath(request.URL.Query()["path"])
	if err != nil {
		writeHTTPError(writer, http.StatusBadRequest, err)
		return
	}
	displayMode := request.URL.Query().Get("mode")
	if displayMode == "" {
		displayMode = "overview"
	}
	referenceVisibility, err := queryReferenceVisibility(request.URL.Query().Get("reference_visibility"))
	if err != nil {
		writeHTTPError(writer, http.StatusBadRequest, err)
		return
	}
	referenceScopes, err := queryReferenceScopes(request.URL.Query()["reference_scope"])
	if err != nil {
		writeHTTPError(writer, http.StatusBadRequest, err)
		return
	}
	scene, err := BuildSceneWithOptions(value, selectedPath, displayMode, SceneOptions{ReferenceVisibility: referenceVisibility, ReferenceScopes: referenceScopes})
	if err != nil {
		writeHTTPError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(writer, http.StatusOK, scene)
}

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
	startLine, endLine, err := sourceLineRange(query, source, len(strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")))
	if err != nil {
		writeHTTPError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
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

func (s *Server) handleReanalysis(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	s.mu.RLock()
	reanalyze := s.reanalyze
	sourceRoot := s.sourceRoot
	current := s.model
	s.mu.RUnlock()
	if reanalyze == nil || sourceRoot == "" {
		writeHTTPError(writer, http.StatusForbidden, analysis.NewHostError(analysis.ErrInvalidRequest, "reanalysis is unavailable for this viewer session", nil))
		return
	}
	var input ReanalysisRequest
	decoder := json.NewDecoder(io.LimitReader(request.Body, 1<<20))
	if err := decoder.Decode(&input); err != nil {
		writeHTTPError(writer, http.StatusBadRequest, analysis.WrapHostError(analysis.ErrInvalidRequest, "reanalysis request is invalid JSON", err, nil))
		return
	}
	if input.Options == nil {
		input.Options = map[string]any{}
	}
	if input.ProjectRoot == "" {
		input.ProjectRoot = sourceRoot
	}
	requestedRoot, err := normalizeSourceRoot(input.ProjectRoot)
	if err != nil {
		writeHTTPError(writer, http.StatusForbidden, err)
		return
	}
	if !samePath(requestedRoot, sourceRoot) {
		writeHTTPError(writer, http.StatusForbidden, analysis.NewHostError(analysis.ErrInvalidRequest, "reanalysis project root must remain within the opened project", map[string]any{"project_root": input.ProjectRoot}))
		return
	}
	if input.Language == "" {
		input.Language = current.Project.Language
	}
	next, err := reanalyze(request.Context(), input)
	if err != nil {
		writeHTTPError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	if err := model.Validate(next); err != nil {
		writeHTTPError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	if next.Status != model.StatusComplete && next.Status != model.StatusPartial {
		writeHTTPError(writer, http.StatusUnprocessableEntity, analysis.NewHostError(analysis.ErrInvalidModel, "reanalysis must produce a complete or partial model", map[string]any{"status": next.Status}))
		return
	}
	s.mu.Lock()
	s.model = next
	s.mu.Unlock()
	writeJSON(writer, http.StatusOK, struct {
		ModelID       string       `json:"model_id"`
		ModelRevision string       `json:"model_revision"`
		Status        model.Status `json:"status"`
		Model         model.Model  `json:"model"`
	}{ModelID: next.ModelID, ModelRevision: next.ModelID, Status: next.Status, Model: next})
}

func (s *Server) getSourceRoot() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sourceRoot
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

func normalizeSourceRoot(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	absolute, err := filepath.Abs(value)
	if err != nil {
		return "", analysis.WrapHostError(analysis.ErrInvalidRequest, "source root could not be normalized", err, nil)
	}
	absolute = filepath.Clean(absolute)
	info, err := os.Stat(absolute)
	if err != nil {
		return "", analysis.WrapHostError(analysis.ErrUnreadableProject, "source root could not be read", err, map[string]any{"project_root": value})
	}
	if !info.IsDir() {
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "source root must be a directory", map[string]any{"project_root": value})
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", analysis.WrapHostError(analysis.ErrUnreadableProject, "source root symlinks could not be resolved", err, map[string]any{"project_root": value})
	}
	return filepath.Clean(resolved), nil
}

func safeRepositoryPath(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "source path is required", nil)
	}
	normalized := strings.ReplaceAll(value, "\\", "/")
	cleaned := path.Clean(normalized)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.HasPrefix(cleaned, "/") || windowsAbsolutePath(cleaned) {
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "source path must remain repository-relative", map[string]any{"path": value})
	}
	return cleaned, nil
}

func containedFilePath(root, relative string) (string, error) {
	candidate, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		return "", analysis.WrapHostError(analysis.ErrInvalidRequest, "source path could not be normalized", err, nil)
	}
	if !samePathRoot(root, candidate) {
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "source path escapes the project root", map[string]any{"path": relative})
	}
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		if os.IsNotExist(err) {
			return "", analysis.NewHostError(analysis.ErrInvalidModel, "source file was not found", map[string]any{"path": relative})
		}
		return "", analysis.WrapHostError(analysis.ErrUnreadableProject, "source file could not be resolved", err, map[string]any{"path": relative})
	}
	if !samePathRoot(root, resolved) {
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "source symlink escapes the project root", map[string]any{"path": relative})
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", analysis.WrapHostError(analysis.ErrUnreadableProject, "source file could not be inspected", err, map[string]any{"path": relative})
	}
	if info.IsDir() {
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "source path is a directory", map[string]any{"path": relative})
	}
	return resolved, nil
}

func readSourceFile(filePath string) ([]byte, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, analysis.WrapHostError(analysis.ErrUnreadableProject, "source file could not be inspected", err, map[string]any{"path": filePath})
	}
	if info.Size() > maxSourceBytes {
		return nil, analysis.NewHostError(analysis.ErrInvalidRequest, "source file is too large to inspect", map[string]any{"max_bytes": maxSourceBytes})
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil, analysis.WrapHostError(analysis.ErrUnreadableProject, "source file could not be read", err, map[string]any{"path": filePath})
	}
	defer func() { _ = file.Close() }()
	content, err := io.ReadAll(io.LimitReader(file, maxSourceBytes+1))
	if err != nil {
		return nil, analysis.WrapHostError(analysis.ErrUnreadableProject, "source file could not be read", err, map[string]any{"path": filePath})
	}
	if len(content) > maxSourceBytes {
		return nil, analysis.NewHostError(analysis.ErrInvalidRequest, "source file is too large to inspect", map[string]any{"max_bytes": maxSourceBytes})
	}
	return content, nil
}

func samePathRoot(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

func samePath(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func windowsAbsolutePath(value string) bool {
	return len(value) >= 2 && ((value[0] >= 'a' && value[0] <= 'z') || (value[0] >= 'A' && value[0] <= 'Z')) && value[1] == ':'
}

func sourceErrorStatus(err error) int {
	switch analysis.ErrorCodeOf(err) {
	case analysis.ErrInvalidRequest:
		return http.StatusForbidden
	case analysis.ErrInvalidModel:
		return http.StatusNotFound
	default:
		return http.StatusUnprocessableEntity
	}
}

func queryReferenceVisibility(value string) (string, error) {
	if value == "" {
		return ReferenceVisibilityHidden, nil
	}
	if !validReferenceVisibility(value) {
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "viewer reference visibility is unsupported", map[string]any{"reference_visibility": value})
	}
	return value, nil
}

func queryReferenceScopes(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	result := []string{}
	for _, value := range values {
		if value == "" || !validReferenceScope(value) {
			return nil, analysis.NewHostError(analysis.ErrInvalidRequest, "viewer reference scope is unsupported", map[string]any{"reference_scope": value})
		}
		result = appendUniqueString(result, value)
	}
	return result, nil
}

func queryHierarchyPath(values []string) ([]string, error) {
	pathValues := []string{}
	for _, value := range values {
		if value == "" {
			return nil, analysis.NewHostError(analysis.ErrInvalidRequest, "viewer hierarchy path contains an empty segment", nil)
		}
		// The contract uses one query value per segment. Accepting a slash-
		// separated value as well keeps the endpoint convenient for direct use.
		for _, segment := range strings.Split(value, "/") {
			if segment == "" {
				return nil, analysis.NewHostError(analysis.ErrInvalidRequest, "viewer hierarchy path contains an empty segment", nil)
			}
			pathValues = append(pathValues, segment)
		}
	}
	return pathValues, nil
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	writer.WriteHeader(status)
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(value)
}

func writeHTTPError(writer http.ResponseWriter, status int, err error) {
	data, marshalErr := analysis.MarshalError(err)
	if marshalErr != nil {
		data = []byte(`{"error":{"code":"host_failure","message":"viewer request failed"}}`)
	}
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	writer.WriteHeader(status)
	_, _ = writer.Write(data)
	_, _ = writer.Write([]byte("\n"))
}

func writeMethodNotAllowed(writer http.ResponseWriter, allowed string) {
	writer.Header().Set("Allow", allowed)
	writeHTTPError(writer, http.StatusMethodNotAllowed, analysis.NewHostError(analysis.ErrInvalidRequest, "viewer endpoint is read-only", map[string]any{"allowed": allowed}))
}
