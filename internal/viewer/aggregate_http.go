package viewer

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/orchestration"
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/viewer/scene"
)

const scopeListSchemaVersion = "arch-view.scopes/v1"

func (s *Server) handleAnalyses(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	s.mu.RLock()
	analyze := s.analyzeCombined
	defaultRoot := s.sourceRoot
	s.mu.RUnlock()
	if analyze == nil {
		writeHTTPError(writer, http.StatusForbidden, analysis.NewHostError(analysis.ErrInvalidRequest, "combined analysis is unavailable for this viewer session", nil))
		return
	}
	var input CombinedAnalysisRequest
	decoder := json.NewDecoder(io.LimitReader(request.Body, 1<<20))
	if err := decoder.Decode(&input); err != nil {
		writeHTTPError(writer, http.StatusBadRequest, analysis.WrapHostError(analysis.ErrInvalidRequest, "analysis request is invalid JSON", err, nil))
		return
	}
	if strings.TrimSpace(input.ProjectRoot) == "" {
		input.ProjectRoot = defaultRoot
	}
	requestedRoot, err := normalizeSourceRoot(input.ProjectRoot)
	if err != nil {
		writeHTTPError(writer, aggregateHTTPStatus(err), err)
		return
	}
	if defaultRoot == "" || !samePath(requestedRoot, defaultRoot) {
		writeHTTPError(writer, http.StatusForbidden, analysis.NewHostError(analysis.ErrInvalidRequest, "analysis project root must match the opened project", map[string]any{"project_root": input.ProjectRoot}))
		return
	}
	input.ProjectRoot = requestedRoot
	if input.CLIOptions == nil {
		input.CLIOptions = map[string]any{}
	}
	run, err := analyze(request.Context(), input)
	if err != nil {
		writeHTTPError(writer, aggregateHTTPStatus(err), err)
		return
	}
	s.storeAggregateRun(&run)
	writeJSON(writer, aggregateRunHTTPStatus(run), run)
}

func (s *Server) handleAnalysis(writer http.ResponseWriter, request *http.Request) {
	trimmed := strings.TrimPrefix(request.URL.Path, "/v1/analyses/")
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(writer, request)
		return
	}
	runID, err := url.PathUnescape(parts[0])
	if err != nil {
		writeHTTPError(writer, http.StatusNotFound, analysis.NewHostError(analysis.ErrInvalidRequest, "analysis run was not found", nil))
		return
	}
	run := s.lookupAggregateRun(runID)
	if run == nil {
		writeHTTPError(writer, http.StatusNotFound, analysis.NewHostError(analysis.ErrInvalidRequest, "analysis run was not found", map[string]any{"run_id": runID}))
		return
	}
	if len(parts) == 1 {
		if request.Method != http.MethodGet {
			writeMethodNotAllowed(writer, http.MethodGet)
			return
		}
		writeJSON(writer, http.StatusOK, run)
		return
	}
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}
	switch parts[1] {
	case "scopes":
		if len(parts) != 2 {
			http.NotFound(writer, request)
			return
		}
		writeJSON(writer, http.StatusOK, struct {
			SchemaVersion string                           `json:"schema_version"`
			RunID         string                           `json:"run_id"`
			ActiveScope   string                           `json:"active_scope"`
			Status        analysis.AnalysisStatus          `json:"status"`
			Scopes        []orchestration.ScopeSummary     `json:"scopes"`
			Diagnostics   []orchestration.ScopedDiagnostic `json:"diagnostics"`
		}{SchemaVersion: scopeListSchemaVersion, RunID: run.RunID, ActiveScope: "all", Status: run.Status, Scopes: run.Scopes, Diagnostics: run.Diagnostics})
	case "events":
		if len(parts) != 2 {
			http.NotFound(writer, request)
			return
		}
		writeJSON(writer, http.StatusOK, struct {
			RunID  string                            `json:"run_id"`
			Events []orchestration.JobLifecycleEvent `json:"events"`
		}{RunID: run.RunID, Events: orchestration.PublishJobLifecycle(orchestration.ExecutionSnapshot{Events: run.Events})})
	case "projection":
		if len(parts) != 2 {
			http.NotFound(writer, request)
			return
		}
		s.handleAggregateProjection(writer, request, run)
	default:
		http.NotFound(writer, request)
	}
}

func (s *Server) handleAggregateProjection(writer http.ResponseWriter, request *http.Request, run *orchestration.AnalysisRun) {
	query := request.URL.Query()
	scope := strings.TrimSpace(query.Get("scope"))
	if scope == "" {
		scope = "all"
	}
	selected, err := run.SelectAnalysisScope(scope)
	if err != nil {
		if scopeEqualAll(scope) && run.Model == nil {
			sceneSnapshot := failedAggregateScene(*run, "all", nil, query)
			writeJSON(writer, http.StatusOK, sceneSnapshot)
			return
		}
		writeHTTPError(writer, aggregateHTTPStatus(err), err)
		return
	}
	displayMode := query.Get("mode")
	if displayMode == "" {
		displayMode = "overview"
	}
	referenceVisibility, err := queryReferenceVisibility(query.Get("reference_visibility"))
	if err != nil {
		writeHTTPError(writer, http.StatusBadRequest, err)
		return
	}
	referenceScopes, err := queryReferenceScopes(query["reference_scope"])
	if err != nil {
		writeHTTPError(writer, http.StatusBadRequest, err)
		return
	}
	hierarchy, err := queryHierarchyPath(query["path"])
	if err != nil {
		writeHTTPError(writer, http.StatusBadRequest, err)
		return
	}
	if selected.Model.ModelID == "" {
		sceneSnapshot := failedAggregateScene(*run, scope, &selected.Summary, query)
		sceneSnapshot.HierarchyPath = hierarchy
		writeJSON(writer, http.StatusOK, sceneSnapshot)
		return
	}
	sceneSnapshot, err := scene.BuildSceneWithOptions(selected.Model, hierarchy, displayMode, scene.SceneOptions{ReferenceVisibility: referenceVisibility, ReferenceScopes: referenceScopes})
	if err != nil {
		writeHTTPError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	sceneSnapshot.ModelID = aggregateModelID(*run, selected.Model.ModelID)
	sceneSnapshot.ModelRevision = sceneSnapshot.ModelID
	sceneSnapshot.ScopeID = selected.Scope
	sceneSnapshot.AggregateStatus = run.Status
	if selected.Scope == "all" {
		sceneSnapshot.ScopeStatus = "aggregate"
	} else {
		sceneSnapshot.ScopeStatus = string(selected.Summary.Status)
	}
	writeJSON(writer, http.StatusOK, sceneSnapshot)
}

func failedAggregateScene(run orchestration.AnalysisRun, scope string, summary *orchestration.ScopeSummary, query url.Values) scene.SceneSnapshot {
	status := model.StatusFailed
	scopeStatus := string(orchestration.JobFailed)
	project := scene.SceneProject{RootLabel: run.Repository.RootLabel, Boundary: "aggregate", Language: "mixed"}
	diagnostics := append([]orchestration.ScopedDiagnostic(nil), run.Diagnostics...)
	if summary != nil {
		project.RootLabel = summary.ProjectRoot
		project.Language = summary.Analyzer.Language
		if project.Language == "" {
			project.Language = "mixed"
		}
		scopeStatus = string(summary.Status)
		diagnostics = diagnostics[:0]
		for _, diagnostic := range run.Diagnostics {
			if diagnostic.ScopeID == scope {
				diagnostics = append(diagnostics, diagnostic)
			}
		}
	}
	if scopeStatus == string(orchestration.JobCancelled) {
		status = model.StatusFailed
	}
	indicators := make([]scene.DiagnosticIndicator, 0, len(diagnostics))
	readingOrder := make([]string, 0, len(diagnostics))
	descriptions := make(map[string]string, len(diagnostics))
	for index, diagnostic := range diagnostics {
		id := diagnostic.ID
		if id == "" {
			id = scope + "::diagnostic-" + string(rune('0'+index))
		}
		indicators = append(indicators, scene.DiagnosticIndicator{ID: id, Code: diagnostic.Code, Severity: diagnostic.Severity, Message: diagnostic.Message, Subject: diagnostic.Subject, Path: diagnostic.Path, Recoverable: diagnostic.Recoverable, Label: diagnostic.Code + " · " + diagnostic.Message})
		readingOrder = append(readingOrder, id)
		descriptions[id] = diagnostic.Message
	}
	hierarchy, _ := queryHierarchyPath(query["path"])
	return scene.SceneSnapshot{
		SchemaVersion:        scene.SceneSchemaVersion,
		ModelID:              aggregateModelID(run, run.RunID),
		ModelRevision:        aggregateModelID(run, run.RunID),
		Status:               status,
		ScopeID:              scope,
		AggregateStatus:      run.Status,
		ScopeStatus:          scopeStatus,
		Project:              project,
		HierarchyPath:        hierarchy,
		DisplayMode:          "overview",
		ReferenceVisibility:  scene.ReferenceVisibilityHidden,
		VisibleNodes:         []scene.VisibleNode{},
		VisibleRelationships: []scene.VisibleRelationship{},
		CycleIndicators:      []scene.CycleIndicator{},
		DiagnosticIndicators: indicators,
		LayerLabels:          []scene.LayerLabel{},
		ReferenceSummary:     scene.ReferenceSummary{ByScope: []scene.ReferenceScopeSummary{}},
		ReferenceDetails:     []scene.ReferenceDetail{},
		EvidenceLinks:        []scene.EvidenceLink{},
		Accessibility:        scene.Accessibility{ReadingOrder: readingOrder, Descriptions: descriptions},
		Summary:              scene.SceneSummary{DiagnosticCount: len(indicators)},
	}
}

func (s *Server) lookupAggregateRun(runID string) *orchestration.AnalysisRun {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.runs[runID]
}

func (s *Server) storeAggregateRun(run *orchestration.AnalysisRun) {
	if run == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[run.RunID] = run
	s.aggregate = run
	if combined, ok := run.CombinedCanonicalModel(); ok {
		s.model = combined
	} else {
		s.model = model.Model{}
	}
}

func aggregateModelID(run orchestration.AnalysisRun, fallback string) string {
	if run.Model != nil && run.Model.ModelID != "" {
		return run.Model.ModelID
	}
	if fallback != "" {
		return fallback
	}
	return run.RunID
}

func scopeEqualAll(value string) bool {
	return value == "" || strings.EqualFold(value, "all")
}

func aggregateRunHTTPStatus(run orchestration.AnalysisRun) int {
	if run.Status == analysis.StatusComplete || run.Status == analysis.StatusPartial {
		return http.StatusOK
	}
	if run.Status == analysis.StatusCancelled {
		return http.StatusUnprocessableEntity
	}
	return http.StatusInternalServerError
}

func aggregateHTTPStatus(err error) int {
	switch analysis.ErrorCodeOf(err) {
	case analysis.ErrInvalidRequest, analysis.ErrInvalidOptions, analysis.ErrAnalysisScopeFilterInvalid:
		return http.StatusBadRequest
	case analysis.ErrAnalysisScopeNotFound:
		return http.StatusNotFound
	case analysis.ErrAmbiguousAnalyzer, analysis.ErrDuplicateAnalyzer:
		return http.StatusConflict
	case analysis.ErrNoAnalyzer, analysis.ErrUnsupportedProject, analysis.ErrUnreadableProject,
		analysis.ErrAnalyzerPackageNotFound, analysis.ErrAnalyzerPlatformUnsupported,
		analysis.ErrAnalyzerRuntimeOverrideRequired, analysis.ErrCancelled:
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}
