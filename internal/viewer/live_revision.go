package viewer

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/live"
	"github.com/buffo/arch-view/internal/model"
)

// snapshotForRequest keeps all live viewer reads pinned to one immutable
// revision when the browser supplies the revision advertised by the live
// status controller. Without this boundary, a graph request and a follow-up
// source or quality request could observe different ready revisions.
func (s *Server) snapshotForRequest(request *http.Request) (model.Model, error) {
	if s == nil {
		return model.Model{}, analysis.NewHostError(analysis.ErrInvalidRequest, "viewer server is unavailable", nil)
	}
	if s.liveSession == nil {
		return s.snapshot(), nil
	}
	revision, supplied, err := requestedLiveRevision(request)
	if err != nil {
		return model.Model{}, err
	}
	if !supplied {
		return s.snapshot(), nil
	}
	record, err := s.liveSession.QuerySpecificRevision(request.Context(), revision)
	if err != nil {
		return model.Model{}, liveViewerHostError(err)
	}
	return record.Model, nil
}

func requestedLiveRevision(request *http.Request) (int, bool, error) {
	if request == nil {
		return 0, false, nil
	}
	raw := strings.TrimSpace(request.URL.Query().Get("revision"))
	if raw == "" {
		return 0, false, nil
	}
	revision, err := strconv.Atoi(raw)
	if err != nil || revision < 1 {
		return 0, true, analysis.NewHostError(analysis.ErrInvalidRequest, "live revision must be a positive integer", map[string]any{"revision": raw})
	}
	return revision, true, nil
}

func liveViewerHostError(err error) error {
	var queryErr *live.QueryError
	if errors.As(err, &queryErr) {
		return analysis.NewHostError(analysis.ErrorCode(queryErr.Code), queryErr.Message, queryErr.Details)
	}
	return err
}

func liveViewerHTTPStatus(err error) int {
	code := analysis.ErrorCodeOf(err)
	switch code {
	case analysis.ErrInvalidRequest:
		return http.StatusBadRequest
	case analysis.ErrSourceScopeUnavailable, analysis.ErrSourceReferenceInvalid, analysis.ErrAnalysisScopeNotFound,
		analysis.ErrInvalidModel:
		return http.StatusNotFound
	case analysis.ErrSourceIndexDigestMismatch:
		return http.StatusConflict
	case analysis.ErrorCode(live.ErrorQueryBudget):
		return http.StatusRequestEntityTooLarge
	case analysis.ErrorCode(live.ErrorNoReadySnapshot), analysis.ErrorCode(live.ErrorRevisionUnavailable),
		analysis.ErrorCode(live.ErrorAnalysisWaitTimeout), analysis.ErrorCode(live.ErrorInputUnstable):
		return http.StatusConflict
	case analysis.ErrorCode("mcp_permission_denied"), analysis.ErrorCode(live.ErrorQualityPolicyPermissionDenied):
		return http.StatusForbidden
	default:
		return http.StatusUnprocessableEntity
	}
}
