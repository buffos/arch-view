package viewer

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/model/canonical"
)

func (s *Server) handleReanalysis(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	s.mu.RLock()
	reanalyze := s.reanalyze
	reanalyzeCombined := s.reanalyzeCombined
	sourceRoot := s.sourceRoot
	current := s.model
	s.mu.RUnlock()
	if aggregate := s.aggregateSnapshot(); aggregate != nil {
		if reanalyzeCombined == nil || sourceRoot == "" {
			writeHTTPError(writer, http.StatusForbidden, analysis.NewHostError(analysis.ErrInvalidRequest, "combined reanalysis is unavailable for this viewer session", nil))
			return
		}
		var input CombinedAnalysisRequest
		decoder := json.NewDecoder(io.LimitReader(request.Body, 1<<20))
		if err := decoder.Decode(&input); err != nil {
			writeHTTPError(writer, http.StatusBadRequest, analysis.WrapHostError(analysis.ErrInvalidRequest, "reanalysis request is invalid JSON", err, nil))
			return
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
		if input.CLIOptions == nil {
			input.CLIOptions = map[string]any{}
		}
		next, err := reanalyzeCombined(request.Context(), input)
		if err != nil {
			writeHTTPError(writer, aggregateHTTPStatus(err), err)
			return
		}
		status := aggregateRunHTTPStatus(next)
		if status != http.StatusOK {
			writeJSON(writer, status, next)
			return
		}
		s.storeAggregateRun(&next)
		writeJSON(writer, status, next)
		return
	}
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
	if err := canonical.Validate(next); err != nil {
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
