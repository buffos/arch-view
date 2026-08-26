package viewer

import (
	"context"
	"embed"
	"html"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/model/canonical"
	"github.com/buffo/arch-view/internal/viewer/layout"
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
	layout     layout.Session
	handler    http.Handler
}

func NewServer(value model.Model, options ...ServerOptions) (*Server, error) {
	if err := canonical.Validate(value); err != nil {
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
	server := &Server{model: value, sourceRoot: sourceRoot, reanalyze: option.Reanalyze, layout: layout.NewSession(sourceRoot)}
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

// Handler returns the local viewer's HTTP handler.
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

func (s *Server) getSourceRoot() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sourceRoot
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
	contentType := "text/javascript; charset=utf-8"
	if name == "styles.css" {
		contentType = "text/css; charset=utf-8"
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
