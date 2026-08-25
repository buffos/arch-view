package viewer

import (
	"embed"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
)

//go:embed web/index.html web/styles.css web/app.js web/vendor/elk.bundled.js web/vendor/elk-worker.min.js
var webFiles embed.FS

type Server struct {
	model   model.Model
	handler http.Handler
}

func NewServer(value model.Model) (*Server, error) {
	if err := model.Validate(value); err != nil {
		return nil, err
	}
	server := &Server{model: value}
	mux := http.NewServeMux()
	mux.HandleFunc("/", server.handleRoot)
	mux.HandleFunc("/assets/", server.handleAsset)
	mux.HandleFunc("/v1/models/", server.handleModel)
	server.handler = mux
	return server, nil
}

func (s *Server) Handler() http.Handler {
	return s.handler
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
	content := strings.ReplaceAll(string(data), "__ARCH_VIEW_MODEL_ID__", html.EscapeString(s.model.ModelID))
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
	if name != "styles.css" && name != "app.js" && name != "vendor/elk.bundled.js" && name != "vendor/elk-worker.min.js" {
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
	case "app.js":
		contentType = "text/javascript; charset=utf-8"
	case "vendor/elk.bundled.js":
		contentType = "text/javascript; charset=utf-8"
	case "vendor/elk-worker.min.js":
		contentType = "text/javascript; charset=utf-8"
	}
	writer.Header().Set("Content-Type", contentType)
	writer.Header().Set("Cache-Control", "public, max-age=3600, immutable")
	_, _ = writer.Write(data)
}

func (s *Server) handleModel(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}
	trimmed := strings.TrimPrefix(request.URL.Path, "/v1/models/")
	parts := strings.Split(trimmed, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(writer, request)
		return
	}
	modelID, err := url.PathUnescape(parts[0])
	if err != nil || modelID != s.model.ModelID {
		writeHTTPError(writer, http.StatusNotFound, analysis.NewHostError(analysis.ErrInvalidModel, "requested model was not found", map[string]any{"model_id": modelID}))
		return
	}
	if len(parts) == 1 {
		writeJSON(writer, http.StatusOK, s.model)
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
	scene, err := BuildSceneWithOptions(s.model, selectedPath, displayMode, SceneOptions{ReferenceVisibility: referenceVisibility, ReferenceScopes: referenceScopes})
	if err != nil {
		writeHTTPError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(writer, http.StatusOK, scene)
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
