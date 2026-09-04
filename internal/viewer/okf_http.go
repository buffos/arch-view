package viewer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	okfapplication "github.com/buffo/arch-view/internal/okf/application"
	"github.com/buffo/arch-view/internal/okf/contract"
	"github.com/buffo/arch-view/internal/okf/domain"
)

const maxOKFRequestBytes = 2 << 20

func (s *Server) handleOKF(writer http.ResponseWriter, request *http.Request) {
	if s.okf == nil {
		writeOKFError(writer, domain.NewError("okf_project_not_found", 503, "OKF views require a project-backed viewer session", nil), request)
		return
	}
	pathValue := strings.TrimPrefix(request.URL.EscapedPath(), "/v1/okf")
	pathValue = strings.Trim(pathValue, "/")
	segments := splitOKFPath(pathValue)
	for index, segment := range segments {
		decoded, err := url.PathUnescape(segment)
		if err != nil {
			writeOKFError(writer, domain.NewError("okf_invalid_request", 400, "invalid path encoding", nil), request)
			return
		}
		segments[index] = decoded
	}
	ctx := request.Context()
	switch {
	case len(segments) == 2 && segments[0] == "catalog" && segments[1] == "refresh":
		s.handleOKFRefresh(writer, request, ctx)
	case len(segments) == 1 && segments[0] == "catalog":
		s.handleOKFCatalog(writer, request, ctx)
	case len(segments) == 2 && segments[0] == "bundles" && segments[1] == "summary":
		s.handleOKFSummary(writer, request, ctx)
	case len(segments) == 1 && segments[0] == "profiles":
		s.handleOKFProfiles(writer, request, ctx)
	case len(segments) == 2 && segments[0] == "profiles" && segments[1] == "validate":
		s.handleOKFValidateProfile(writer, request, ctx)
	case len(segments) == 1 && segments[0] == "extensions":
		s.handleOKFExtensions(writer, request, ctx)
	case len(segments) == 1 && segments[0] == "diagnostics":
		s.handleOKFDiagnostics(writer, request, ctx)
	case len(segments) == 1 && segments[0] == "bindings":
		s.handleOKFBinding(writer, request, ctx)
	case len(segments) == 2 && segments[0] == "profiles" && segments[1] == "save-as":
		if request.Method != http.MethodPost {
			writeOKFMethodNotAllowed(writer, request, http.MethodPost)
			return
		}
		s.handleOKFSaveAs(writer, request, ctx)
	case len(segments) >= 2 && segments[0] == "sessions":
		s.handleOKFSession(writer, request, ctx, segments[1], segments[2:])
	case len(segments) >= 2 && segments[0] == "profiles":
		s.handleOKFProfileCommand(writer, request, ctx, segments[1], segments[2:])
	default:
		writeOKFError(writer, domain.NewError("okf_invalid_request", 400, "unknown OKF endpoint", nil), request)
	}
}

func (s *Server) handleOKFRefresh(writer http.ResponseWriter, request *http.Request, ctx context.Context) {
	if request.Method != http.MethodPost {
		writeOKFMethodNotAllowed(writer, request, http.MethodPost)
		return
	}
	value, err := s.okf.Refresh(ctx)
	if err != nil {
		writeOKFError(writer, err, request)
		return
	}
	writeOKFSuccess(writer, http.StatusOK, value, value.Diagnostics, request.Header.Get("Idempotency-Key"), value.Revision, request)
}

func (s *Server) handleOKFCatalog(writer http.ResponseWriter, request *http.Request, ctx context.Context) {
	if request.Method != http.MethodGet {
		writeOKFMethodNotAllowed(writer, request, http.MethodGet)
		return
	}
	value := s.okf.Catalog()
	if value.ProjectID == "" {
		var err error
		value, err = s.okf.Refresh(ctx)
		if err != nil {
			writeOKFError(writer, err, request)
			return
		}
	}
	writeOKFSuccess(writer, http.StatusOK, value, value.Diagnostics, "", value.Revision, request)
}

func (s *Server) handleOKFSummary(writer http.ResponseWriter, request *http.Request, ctx context.Context) {
	if request.Method != http.MethodGet {
		writeOKFMethodNotAllowed(writer, request, http.MethodGet)
		return
	}
	bundleID := request.URL.Query().Get("bundle_id")
	value, err := s.okf.Summary(ctx, bundleID)
	if err != nil {
		writeOKFError(writer, err, request)
		return
	}
	writeOKFSuccess(writer, http.StatusOK, value, value.Diagnostics, "", value.SourceRevision, request)
}

func (s *Server) handleOKFProfiles(writer http.ResponseWriter, request *http.Request, ctx context.Context) {
	if request.Method != http.MethodGet {
		writeOKFMethodNotAllowed(writer, request, http.MethodGet)
		return
	}
	value, err := s.okf.Profiles(ctx)
	if err != nil {
		writeOKFError(writer, err, request)
		return
	}
	writeOKFSuccess(writer, http.StatusOK, value, value.Diagnostics, "", value.RegistryRevision, request)
}

func (s *Server) handleOKFExtensions(writer http.ResponseWriter, request *http.Request, ctx context.Context) {
	if request.Method != http.MethodGet {
		writeOKFMethodNotAllowed(writer, request, http.MethodGet)
		return
	}
	value, err := s.okf.Extensions(ctx)
	if err != nil {
		writeOKFError(writer, err, request)
		return
	}
	revision, err := contract.ExtensionCatalogRevision(value)
	if err != nil {
		writeOKFError(writer, err, request)
		return
	}
	writeOKFSuccess(writer, http.StatusOK, map[string]any{"extensions": value}, nil, "", revision, request)
}

func (s *Server) handleOKFDiagnostics(writer http.ResponseWriter, request *http.Request, ctx context.Context) {
	if request.Method != http.MethodGet {
		writeOKFMethodNotAllowed(writer, request, http.MethodGet)
		return
	}
	query := request.URL.Query()
	value, err := s.okf.Diagnostics(ctx, okfapplication.DiagnosticQuery{
		ProjectID: query.Get("project_id"), BundleID: query.Get("bundle_id"), ProfileID: query.Get("profile_id"), ConceptID: query.Get("concept_id"),
		RelationshipID: query.Get("relationship_id"), OperationID: query.Get("operation_id"), Severity: query.Get("severity"), Category: query.Get("category"),
	})
	if err != nil {
		writeOKFError(writer, err, request)
		return
	}
	writeOKFSuccess(writer, http.StatusOK, value, value.Diagnostics, "", value.Revision, request)
}

func (s *Server) handleOKFBinding(writer http.ResponseWriter, request *http.Request, ctx context.Context) {
	if request.Method != http.MethodPut {
		writeOKFMethodNotAllowed(writer, request, http.MethodPut)
		return
	}
	bundleID := request.URL.Query().Get("bundle_id")
	if bundleID == "" {
		writeOKFError(writer, domain.NewError("okf_invalid_request", 400, "bundle_id is required", nil), request)
		return
	}
	data, err := readOKFBody(request.Body)
	if err != nil {
		writeOKFError(writer, err, request)
		return
	}
	var input contract.BindingRequest
	if err := decodeOKF(data, &input); err != nil {
		writeOKFError(writer, err, request)
		return
	}
	operationID := operationID(request, input.OperationID)
	value, err := s.okf.Bind(ctx, bundleID, input.ProfileID, expectedRevision(request, input.ExpectedRevision), operationID, data)
	if err != nil {
		writeOKFError(writer, err, request)
		return
	}
	writeOKFSuccess(writer, http.StatusOK, value, nil, operationID, value.Revision, request)
}

func (s *Server) handleOKFSession(writer http.ResponseWriter, request *http.Request, ctx context.Context, sessionID string, action []string) {
	if len(action) == 0 {
		if request.Method != http.MethodGet {
			writeOKFMethodNotAllowed(writer, request, http.MethodGet)
			return
		}
		value, err := s.okf.Session(ctx, sessionID)
		if err != nil {
			writeOKFError(writer, err, request)
			return
		}
		writeOKFSuccess(writer, http.StatusOK, value, value.Diagnostics, "", sessionRevision(value), request)
		return
	}
	switch action[0] {
	case "bundle":
		if request.Method != http.MethodPut {
			writeOKFMethodNotAllowed(writer, request, http.MethodPut)
			return
		}
		data, err := readOKFBody(request.Body)
		if err != nil {
			writeOKFError(writer, err, request)
			return
		}
		var input contract.SessionBundleRequest
		if err := decodeOKF(data, &input); err != nil {
			writeOKFError(writer, err, request)
			return
		}
		value, err := s.okf.SelectBundle(ctx, sessionID, input.BundleID)
		if err != nil {
			writeOKFError(writer, err, request)
			return
		}
		writeOKFSuccess(writer, http.StatusOK, value, value.Diagnostics, "", sessionRevision(value), request)
	case "profile":
		if request.Method != http.MethodPut {
			writeOKFMethodNotAllowed(writer, request, http.MethodPut)
			return
		}
		data, err := readOKFBody(request.Body)
		if err != nil {
			writeOKFError(writer, err, request)
			return
		}
		var input contract.SessionProfileRequest
		if err := decodeOKF(data, &input); err != nil {
			writeOKFError(writer, err, request)
			return
		}
		value, err := s.okf.SelectProfile(ctx, sessionID, input.ProfileID)
		if err != nil {
			writeOKFError(writer, err, request)
			return
		}
		writeOKFSuccess(writer, http.StatusOK, value, value.Diagnostics, "", sessionRevision(value), request)
	case "projection":
		if request.Method != http.MethodGet {
			writeOKFMethodNotAllowed(writer, request, http.MethodGet)
			return
		}
		value, err := s.okf.Projection(ctx, sessionID)
		if err != nil {
			writeOKFError(writer, err, request)
			return
		}
		var data any = value.Projection
		if data == nil {
			data = value
		}
		writeOKFSuccess(writer, http.StatusOK, data, value.Diagnostics, "", sessionRevision(value), request)
	case "navigation":
		s.handleOKFNavigation(writer, request, ctx, sessionID, action[1:])
	case "concept-detail":
		if request.Method != http.MethodGet {
			writeOKFMethodNotAllowed(writer, request, http.MethodGet)
			return
		}
		conceptID := request.URL.Query().Get("concept_id")
		if conceptID == "" {
			writeOKFError(writer, domain.NewError("okf_invalid_request", 400, "concept_id is required", nil), request)
			return
		}
		value, err := s.okf.Detail(ctx, sessionID, conceptID)
		if err != nil {
			writeOKFError(writer, err, request)
			return
		}
		writeOKFSuccess(writer, http.StatusOK, value, value.Diagnostics, "", value.SourceRevision, request)
	default:
		writeOKFError(writer, domain.NewError("okf_invalid_request", 400, "unknown session endpoint", nil), request)
	}
}

func (s *Server) handleOKFNavigation(writer http.ResponseWriter, request *http.Request, ctx context.Context, sessionID string, action []string) {
	if len(action) == 0 {
		if request.Method != http.MethodGet {
			writeOKFMethodNotAllowed(writer, request, http.MethodGet)
			return
		}
		value, err := s.okf.Session(ctx, sessionID)
		if err != nil {
			writeOKFError(writer, err, request)
			return
		}
		writeOKFSuccess(writer, http.StatusOK, value.Navigation, value.Diagnostics, "", sessionRevision(value), request)
		return
	}
	switch action[0] {
	case "depth":
		if request.Method != http.MethodPut {
			writeOKFMethodNotAllowed(writer, request, http.MethodPut)
			return
		}
		data, err := readOKFBody(request.Body)
		if err != nil {
			writeOKFError(writer, err, request)
			return
		}
		var input contract.NavigationDepthRequest
		if err := decodeOKF(data, &input); err != nil {
			writeOKFError(writer, err, request)
			return
		}
		value, err := s.okf.SetNavigation(ctx, sessionID, input.Depth, input.Full)
		if err != nil {
			writeOKFError(writer, err, request)
			return
		}
		writeOKFSuccess(writer, http.StatusOK, value, value.Diagnostics, "", sessionRevision(value), request)
	case "focus":
		if request.Method != http.MethodPost {
			writeOKFMethodNotAllowed(writer, request, http.MethodPost)
			return
		}
		data, err := readOKFBody(request.Body)
		if err != nil {
			writeOKFError(writer, err, request)
			return
		}
		var input contract.NavigationFocusRequest
		if err := decodeOKF(data, &input); err != nil {
			writeOKFError(writer, err, request)
			return
		}
		value, err := s.okf.Focus(ctx, sessionID, input.ConceptID)
		if err != nil {
			writeOKFError(writer, err, request)
			return
		}
		writeOKFSuccess(writer, http.StatusOK, value, value.Diagnostics, "", sessionRevision(value), request)
	case "back", "top":
		if request.Method != http.MethodPost {
			writeOKFMethodNotAllowed(writer, request, http.MethodPost)
			return
		}
		operation := s.okf.Back
		if action[0] == "top" {
			operation = s.okf.TopLevel
		}
		value, err := operation(ctx, sessionID)
		if err != nil {
			writeOKFError(writer, err, request)
			return
		}
		writeOKFSuccess(writer, http.StatusOK, value, value.Diagnostics, "", sessionRevision(value), request)
	default:
		writeOKFError(writer, domain.NewError("okf_invalid_request", 400, "unknown navigation endpoint", nil), request)
	}
}

func (s *Server) handleOKFProfileCommand(writer http.ResponseWriter, request *http.Request, ctx context.Context, profileID string, action []string) {
	decodedID := profileID
	if len(action) == 1 && action[0] == "rename" {
		if request.Method != http.MethodPost {
			writeOKFMethodNotAllowed(writer, request, http.MethodPost)
			return
		}
		data, readErr := readOKFBody(request.Body)
		if readErr != nil {
			writeOKFError(writer, readErr, request)
			return
		}
		var input contract.ProfileRenameRequest
		if readErr := decodeOKF(data, &input); readErr != nil {
			writeOKFError(writer, readErr, request)
			return
		}
		op := operationID(request, input.OperationID)
		value, callErr := s.okf.RenameProfile(ctx, decodedID, input.NewProfileID, input.NewName, expectedRevision(request, input.ExpectedRevision), op, data)
		if callErr != nil {
			writeOKFError(writer, callErr, request)
			return
		}
		writeOKFSuccess(writer, http.StatusOK, value, nil, op, value.Revision, request)
		return
	}
	if len(action) != 0 {
		writeOKFError(writer, domain.NewError("okf_invalid_request", 400, "unknown profile endpoint", nil), request)
		return
	}
	switch request.Method {
	case http.MethodPut:
		data, readErr := readOKFBody(request.Body)
		if readErr != nil {
			writeOKFError(writer, readErr, request)
			return
		}
		value, requestExpectedRevision, requestOperationID, readErr := decodeProfileSave(data)
		if readErr != nil {
			writeOKFError(writer, readErr, request)
			return
		}
		value.ProfileID = decodedID
		op := operationID(request, requestOperationID)
		configValue, callErr := s.okf.SaveProfile(ctx, value, expectedRevision(request, requestExpectedRevision), op, data)
		if callErr != nil {
			writeOKFError(writer, callErr, request)
			return
		}
		writeOKFSuccess(writer, http.StatusOK, configValue, nil, op, configValue.Revision, request)
	case http.MethodDelete:
		data, readErr := readOKFBody(request.Body)
		if readErr != nil {
			writeOKFError(writer, readErr, request)
			return
		}
		var input contract.ProfileDeleteRequest
		if len(bytes.TrimSpace(data)) > 0 {
			if readErr := decodeOKF(data, &input); readErr != nil {
				writeOKFError(writer, readErr, request)
				return
			}
		}
		op := operationID(request, input.OperationID)
		configValue, callErr := s.okf.DeleteProfile(ctx, decodedID, input.ReplacementProfileID, input.NeutralFallback, expectedRevision(request, input.ExpectedRevision), op, data)
		if callErr != nil {
			writeOKFError(writer, callErr, request)
			return
		}
		writeOKFSuccess(writer, http.StatusOK, configValue, nil, op, configValue.Revision, request)
	default:
		writeOKFMethodNotAllowed(writer, request, http.MethodPut+", "+http.MethodDelete)
	}
}

func (s *Server) handleOKFSaveAs(writer http.ResponseWriter, request *http.Request, ctx context.Context) {
	data, err := readOKFBody(request.Body)
	if err != nil {
		writeOKFError(writer, err, request)
		return
	}
	var input contract.ProfileSaveAsRequest
	if err := decodeOKF(data, &input); err != nil {
		writeOKFError(writer, err, request)
		return
	}
	operation := operationID(request, input.OperationID)
	value, err := s.okf.SaveProfileAs(ctx, input.Profile, input.SourceProfileID, input.NewProfileID, expectedRevision(request, input.ExpectedRevision), operation, data)
	if err != nil {
		writeOKFError(writer, err, request)
		return
	}
	writeOKFSuccess(writer, http.StatusCreated, value, nil, operation, value.Revision, request)
}

func splitOKFPath(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, "/")
	result := make([]string, 0, len(parts))
	for _, item := range parts {
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func readOKFBody(body io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(body, maxOKFRequestBytes+1))
	if err != nil {
		return nil, domain.WrapError("okf_invalid_request", 400, "OKF request could not be read", err)
	}
	if len(data) > maxOKFRequestBytes {
		return nil, domain.NewError("okf_invalid_request", 400, "OKF request is too large", map[string]any{"max_bytes": maxOKFRequestBytes})
	}
	return data, nil
}

func decodeOKF(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return domain.WrapError("okf_invalid_request", 400, "OKF request is invalid JSON", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return domain.NewError("okf_invalid_request", 400, "OKF request contains trailing data", nil)
	}
	return nil
}

func decodeProfile(data []byte) (domain.Profile, error) {
	var fields map[string]json.RawMessage
	if err := decodeOKF(data, &fields); err != nil {
		return domain.Profile{}, err
	}
	if _, wrapped := fields["profile"]; !wrapped || fields["profile_id"] != nil {
		var value domain.Profile
		err := decodeOKF(data, &value)
		return value, err
	}
	var wrapped struct {
		Profile domain.Profile `json:"profile"`
	}
	if err := decodeOKF(data, &wrapped); err != nil {
		return domain.Profile{}, err
	}
	return wrapped.Profile, nil
}

func decodeProfileSave(data []byte) (domain.Profile, string, string, error) {
	var input contract.ProfileSaveRequest
	if err := decodeOKF(data, &input); err == nil && (input.Profile.ProfileID != "" || input.ExpectedRevision != "" || input.OperationID != "") {
		return input.Profile, input.ExpectedRevision, input.OperationID, nil
	}
	value, err := decodeProfile(data)
	return value, "", "", err
}

func writeOKFSuccess(writer http.ResponseWriter, status int, data any, diagnostics []domain.Diagnostic, operationID, revision string, request *http.Request) {
	writeJSON(writer, status, contract.Success{Data: data, Diagnostics: diagnostics, Meta: contract.Meta{RequestID: contract.RequestID(request), OperationID: operationID, Revision: revision}})
}

func writeOKFError(writer http.ResponseWriter, err error, request *http.Request) {
	value, status := contract.ErrorResponse(err, contract.RequestID(request))
	writeJSON(writer, status, value)
}

func writeOKFMethodNotAllowed(writer http.ResponseWriter, request *http.Request, allowed string) {
	writer.Header().Set("Allow", allowed)
	writeOKFError(writer, domain.NewError("okf_method_not_allowed", http.StatusMethodNotAllowed, "the requested HTTP method is not supported for this OKF endpoint", map[string]any{"allowed": allowed}), request)
}

func operationID(request *http.Request, bodyValue string) string {
	if value := request.Header.Get("Idempotency-Key"); value != "" {
		return value
	}
	return bodyValue
}
func expectedRevision(request *http.Request, bodyValue string) string {
	if value := request.Header.Get("If-Match"); value != "" {
		return strings.Trim(value, "\"")
	}
	return bodyValue
}

func diagnosticsFromOKFError(err error) []domain.Diagnostic {
	var value *domain.Error
	if errors.As(err, &value) && len(value.Diagnostics) > 0 {
		return value.Diagnostics
	}
	if err == nil {
		return nil
	}
	return []domain.Diagnostic{{Code: "okf_projection_failed", Severity: "error", Category: "infrastructure", Message: err.Error()}}
}
func sessionRevision(value okfapplication.SessionView) string {
	if value.Projection != nil {
		return value.Projection.ProjectionRevision
	}
	return ""
}
