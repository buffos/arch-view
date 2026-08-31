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
	"github.com/buffo/arch-view/internal/analysis/orchestration"
	"github.com/buffo/arch-view/internal/live"
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/model/canonical"
	"github.com/buffo/arch-view/internal/quality"
	"github.com/buffo/arch-view/internal/viewer/layout"
)

//go:embed web/index.html web/styles.css web/styles/*.css web/app.js web/app/*.js web/layout_request.js web/graph_route.js web/vendor/elk.bundled.js web/vendor/elk-worker.min.js
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

// CombinedAnalysisRequest is the transport-neutral request envelope used by
// the aggregate HTTP surface and the project-backed viewer. The CLI owns
// conversion from flags/configuration into these validated planning inputs.
type CombinedAnalysisRequest struct {
	ProjectRoot       string                          `json:"project_root"`
	SourceScopePolicy orchestration.SourceScopePolicy `json:"source_scope_policy"`
	CLIOptions        map[string]any                  `json:"cli_options,omitempty"`
}

// CombinedAnalyzeFunc connects the viewer/HTTP transport to the orchestration
// layer without making the viewer responsible for analyzer selection.
type CombinedAnalyzeFunc func(context.Context, CombinedAnalysisRequest) (orchestration.AnalysisRun, error)

type ServerOptions struct {
	SourceRoot        string
	Reanalyze         ReanalyzeFunc
	AnalyzeCombined   CombinedAnalyzeFunc
	ReanalyzeCombined CombinedAnalyzeFunc
	LiveSession       *live.LiveSession
}

type Server struct {
	mu                     sync.RWMutex
	model                  model.Model
	aggregate              *orchestration.AnalysisRun
	runs                   map[string]*orchestration.AnalysisRun
	sourceRoot             string
	reanalyze              ReanalyzeFunc
	analyzeCombined        CombinedAnalyzeFunc
	reanalyzeCombined      CombinedAnalyzeFunc
	liveSession            *live.LiveSession
	liveHTTP               *live.HTTPServer
	qualityReports         map[string]quality.QualityEvaluation
	qualityReportRevisions map[string]int
	layout                 layout.Session
	handler                http.Handler
}

func NewServer(value model.Model, options ...ServerOptions) (*Server, error) {
	if err := canonical.Validate(value); err != nil {
		return nil, err
	}
	return newServer(value, nil, options...)
}

// NewAggregateServer creates a viewer over one cached combined analysis run.
// A run with no usable scope is allowed so its diagnostics can be inspected;
// it simply has no model-backed graph projection.
func NewAggregateServer(run orchestration.AnalysisRun, options ...ServerOptions) (*Server, error) {
	var value model.Model
	if combined, ok := run.CombinedCanonicalModel(); ok {
		value = combined
		if err := canonical.Validate(value); err != nil {
			return nil, err
		}
	}
	return newServer(value, &run, options...)
}

// NewLiveServer creates a viewer over the latest ready revision of a live
// session. One-shot viewer constructors retain their historical behavior.
func NewLiveServer(session *live.LiveSession, options ...ServerOptions) (*Server, error) {
	if session == nil {
		return nil, analysis.NewHostError(analysis.ErrInvalidRequest, "live viewer requires a live session", nil)
	}
	option := ServerOptions{SourceRoot: session.RepositoryRoot(), LiveSession: session}
	if len(options) > 0 {
		option = options[0]
		option.LiveSession = session
		if option.SourceRoot == "" {
			option.SourceRoot = session.RepositoryRoot()
		}
	}
	return newServer(model.Model{}, nil, option)
}

func newServer(value model.Model, aggregate *orchestration.AnalysisRun, options ...ServerOptions) (*Server, error) {
	var option ServerOptions
	if len(options) > 0 {
		option = options[0]
	}
	if option.LiveSession != nil && option.SourceRoot == "" {
		option.SourceRoot = option.LiveSession.RepositoryRoot()
	}
	sourceRoot, err := normalizeSourceRoot(option.SourceRoot)
	if err != nil {
		return nil, err
	}
	server := &Server{
		model:                  value,
		aggregate:              aggregate,
		runs:                   make(map[string]*orchestration.AnalysisRun),
		sourceRoot:             sourceRoot,
		reanalyze:              option.Reanalyze,
		analyzeCombined:        option.AnalyzeCombined,
		reanalyzeCombined:      option.ReanalyzeCombined,
		liveSession:            option.LiveSession,
		qualityReports:         make(map[string]quality.QualityEvaluation),
		qualityReportRevisions: make(map[string]int),
		layout:                 layout.NewSession(sourceRoot),
	}
	if aggregate != nil {
		server.runs[aggregate.RunID] = aggregate
	}
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
	mux.HandleFunc("/v1/analyses", server.handleAnalyses)
	mux.HandleFunc("/v1/analyses/", server.handleAnalysis)
	mux.HandleFunc("/v1/quality/profiles", server.handleQualityProfiles)
	mux.HandleFunc("/v1/quality/profiles/save", server.handleQualityProfileSave)
	mux.HandleFunc("/v1/quality/profiles/save-as", server.handleQualityProfileSaveAs)
	mux.HandleFunc("/v1/quality/baselines/create", server.handleQualityBaselineCreate)
	mux.HandleFunc("/v1/quality/rules", server.handleQualityRules)
	mux.HandleFunc("/v1/quality/evaluate", server.handleQualityEvaluation)
	mux.HandleFunc("/v1/models/", server.handleModel)
	if server.liveSession != nil {
		server.liveHTTP = live.NewHTTPServer(live.NewLocalQueryAdapter(server.liveSession), live.HTTPServerOptions{Transport: live.TransportLocalHTTP, SessionID: server.liveSession.Config().SessionID})
		mux.Handle("/v1/live/", server.liveHTTP.Handler())
	}
	server.handler = mux
	return server, nil
}

// Handler returns the local viewer's HTTP handler.
func (s *Server) Handler() http.Handler {
	return s.handler
}

func (s *Server) snapshot() model.Model {
	if s != nil && s.liveSession != nil {
		if record, err := s.liveSession.QueryLatestReady(context.Background()); err == nil && record != nil {
			return record.Model
		}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.model
}

func (s *Server) aggregateSnapshot() *orchestration.AnalysisRun {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.aggregate
}

func (s *Server) modelID() string {
	if s != nil && s.liveSession != nil {
		if record, err := s.liveSession.QueryLatestReady(context.Background()); err == nil && record != nil && record.Model.ModelID != "" {
			return record.Model.ModelID
		}
		return s.liveSession.Config().SessionID
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.aggregate != nil && s.aggregate.Model != nil {
		return s.aggregate.Model.ModelID
	}
	if s.aggregate != nil {
		return s.aggregate.RunID
	}
	return s.model.ModelID
}

func (s *Server) sourceEnabled() bool {
	if s != nil && s.liveSession != nil {
		return s.liveSession.Config().SourceIndexRequest.Enabled && s.liveSession.RepositoryRoot() != ""
	}
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
	if s != nil && s.liveSession != nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return (s.reanalyze != nil || s.reanalyzeCombined != nil) && s.sourceRoot != ""
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
	modelID := s.modelID()
	content := strings.ReplaceAll(string(data), "__ARCH_VIEW_MODEL_ID__", html.EscapeString(modelID))
	content = strings.ReplaceAll(content, "__ARCH_VIEW_SOURCE_ENABLED__", strconv.FormatBool(s.sourceEnabled()))
	content = strings.ReplaceAll(content, "__ARCH_VIEW_REANALYSIS_ENABLED__", strconv.FormatBool(s.reanalysisEnabled()))
	content = strings.ReplaceAll(content, "__ARCH_VIEW_WORKER_URL__", "/assets/vendor/elk-worker.min.js")
	content = strings.ReplaceAll(content, "__ARCH_VIEW_ANALYSIS_RUN_ID__", html.EscapeString(s.analysisRunID()))
	content = strings.ReplaceAll(content, "__ARCH_VIEW_AGGREGATE__", strconv.FormatBool(s.isAggregate()))
	content = strings.ReplaceAll(content, "__ARCH_VIEW_LIVE_ENABLED__", strconv.FormatBool(s.liveSession != nil))
	liveSessionID := ""
	if s.liveSession != nil {
		liveSessionID = s.liveSession.Config().SessionID
	}
	content = strings.ReplaceAll(content, "__ARCH_VIEW_LIVE_SESSION_ID__", html.EscapeString(liveSessionID))
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	_, _ = io.WriteString(writer, content)
}

func (s *Server) analysisRunID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.aggregate == nil {
		return ""
	}
	return s.aggregate.RunID
}

func (s *Server) isAggregate() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.aggregate != nil
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
	data, err := Asset(name)
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
