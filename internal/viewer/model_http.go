package viewer

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/orchestration"
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/quality"
	"github.com/buffo/arch-view/internal/viewer/scene"
)

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
	if err != nil {
		writeHTTPError(writer, http.StatusNotFound, analysis.NewHostError(analysis.ErrInvalidModel, "requested model was not found", map[string]any{"model_id": modelID}))
		return
	}
	if aggregate := s.aggregateSnapshot(); aggregate != nil {
		s.handleAggregateModel(writer, request, parts, modelID, aggregate)
		return
	}
	value, snapshotErr := s.snapshotForRequest(request)
	if snapshotErr != nil {
		writeHTTPError(writer, liveViewerHTTPStatus(snapshotErr), snapshotErr)
		return
	}
	if s.liveSession == nil && modelID != value.ModelID {
		writeHTTPError(writer, http.StatusNotFound, analysis.NewHostError(analysis.ErrInvalidModel, "requested model was not found", map[string]any{"model_id": modelID}))
		return
	}
	if s.liveSession != nil && value.ModelID == "" {
		writeHTTPError(writer, http.StatusConflict, analysis.NewHostError(analysis.ErrInvalidModel, "the live session has no ready model yet", map[string]any{"session_id": s.liveSession.Config().SessionID}))
		return
	}
	if s.liveSession != nil && modelID != s.liveSession.Config().SessionID && modelID != value.ModelID {
		writeHTTPError(writer, http.StatusNotFound, analysis.NewHostError(analysis.ErrInvalidModel, "requested live model was not found", map[string]any{"model_id": modelID}))
		return
	}
	if len(parts) == 1 {
		response, responseErr := modelForResponse(value, request)
		if responseErr != nil {
			writeHTTPError(writer, http.StatusBadRequest, responseErr)
			return
		}
		writeJSON(writer, http.StatusOK, response)
		return
	}
	if parts[1] == "source-index" {
		s.handleSourceIndex(writer, request, parts, modelID, &value, nil)
		return
	}
	if parts[1] == "quality" {
		s.handleQuality(writer, request, parts, modelID, s.qualityReportForRequest("all", value.QualityReport, request), &value, nil)
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
	sceneSnapshot, err := scene.BuildSceneWithOptions(value, selectedPath, displayMode, scene.SceneOptions{ReferenceVisibility: referenceVisibility, ReferenceScopes: referenceScopes})
	if err != nil {
		writeHTTPError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(writer, http.StatusOK, sceneSnapshot)
}

func (s *Server) handleAggregateModel(writer http.ResponseWriter, request *http.Request, parts []string, modelID string, aggregate *orchestration.AnalysisRun) {
	combined, ok := aggregate.CombinedCanonicalModel()
	if !ok {
		writeHTTPError(writer, http.StatusUnprocessableEntity, analysis.NewHostError(analysis.ErrInvalidModel, "the aggregate run has no usable combined model", map[string]any{"run_id": aggregate.RunID}))
		return
	}
	if aggregate.Model == nil || modelID != aggregate.Model.ModelID {
		writeHTTPError(writer, http.StatusNotFound, analysis.NewHostError(analysis.ErrInvalidModel, "requested model was not found", map[string]any{"model_id": modelID}))
		return
	}
	if len(parts) == 1 {
		response, responseErr := aggregateModelForResponse(*aggregate.Model, request)
		if responseErr != nil {
			writeHTTPError(writer, http.StatusBadRequest, responseErr)
			return
		}
		writeJSON(writer, http.StatusOK, response)
		return
	}
	if parts[1] == "source-index" {
		s.handleSourceIndex(writer, request, parts, modelID, nil, aggregate)
		return
	}
	if parts[1] == "quality" {
		var report *quality.QualityEvaluation
		if aggregate.Model != nil {
			report = aggregate.Model.QualityReport
		}
		if scope := normalizedQualityScope(request.URL.Query().Get("scope")); scope != "" {
			scopeResult, scopeErr := aggregate.ScopeResult(scope)
			if scopeErr != nil {
				writeHTTPError(writer, http.StatusNotFound, scopeErr)
				return
			}
			report = scopeResult.QualityReport
			report = s.qualityReportForRequest(scope, report, request)
		} else {
			report = s.qualityReportForRequest("all", report, request)
		}
		s.handleQuality(writer, request, parts, modelID, report, nil, aggregate)
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
	sceneSnapshot, err := scene.BuildSceneWithOptions(combined, selectedPath, displayMode, scene.SceneOptions{ReferenceVisibility: referenceVisibility, ReferenceScopes: referenceScopes})
	if err != nil {
		writeHTTPError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	sceneSnapshot.ScopeID = "all"
	sceneSnapshot.AggregateStatus = aggregate.Status
	sceneSnapshot.ScopeStatus = "aggregate"
	writeJSON(writer, http.StatusOK, sceneSnapshot)
}

// modelForResponse keeps the historical full model response by default while
// allowing the graph viewer to opt out of the potentially large source-index
// attachment. The copy prevents a transport preference from mutating the
// server's canonical model cached for other clients.
func modelForResponse(value model.Model, request *http.Request) (model.Model, error) {
	includeSourceIndex := true
	if raw := request.URL.Query().Get("include_source_index"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return model.Model{}, analysis.NewHostError(analysis.ErrInvalidRequest, "include_source_index must be a boolean", map[string]any{"include_source_index": raw})
		}
		includeSourceIndex = parsed
	}
	if !includeSourceIndex {
		value.SourceIndex = nil
	}
	return value, nil
}

func aggregateModelForResponse(value orchestration.AggregateModel, request *http.Request) (orchestration.AggregateModel, error) {
	includeSourceIndex := true
	if raw := request.URL.Query().Get("include_source_index"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return orchestration.AggregateModel{}, analysis.NewHostError(analysis.ErrInvalidRequest, "include_source_index must be a boolean", map[string]any{"include_source_index": raw})
		}
		includeSourceIndex = parsed
	}
	if !includeSourceIndex {
		value.SourceIndex = nil
	}
	return value, nil
}
