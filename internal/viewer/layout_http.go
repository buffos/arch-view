package viewer

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

const maxLayoutRequestBytes = 1 << 20

func (s *Server) handleLayoutOptions(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}
	writeJSON(writer, http.StatusOK, layoutCatalog())
}

func (s *Server) handleLayoutConfig(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet:
		s.mu.RLock()
		response := s.layout.response()
		s.mu.RUnlock()
		writeJSON(writer, http.StatusOK, response)
	case http.MethodPut:
		profile, err := decodeLayoutProfileRequest(request.Body)
		if err != nil {
			writeHTTPError(writer, layoutErrorStatus(err), err)
			return
		}
		if err := s.saveActiveLayout(profile); err != nil {
			writeHTTPError(writer, layoutErrorStatus(err), err)
			return
		}
		s.mu.RLock()
		response := s.layout.response()
		s.mu.RUnlock()
		writeJSON(writer, http.StatusOK, response)
	default:
		writeMethodNotAllowed(writer, http.MethodGet+", "+http.MethodPut)
	}
}

func (s *Server) handleLayoutConfigSaveAs(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPut {
		writeMethodNotAllowed(writer, http.MethodPut)
		return
	}
	input, err := decodeLayoutSaveAsRequest(request.Body)
	if err != nil {
		writeHTTPError(writer, layoutErrorStatus(err), err)
		return
	}
	if err := s.saveLayoutAs(input.Layout, input.DestinationDir, input.Confirm); err != nil {
		writeHTTPError(writer, layoutErrorStatus(err), err)
		return
	}
	s.mu.RLock()
	response := s.layout.response()
	s.mu.RUnlock()
	writeJSON(writer, http.StatusOK, response)
}

func (s *Server) handleLayoutApply(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	profile, err := decodeLayoutProfileRequest(request.Body)
	if err != nil {
		writeHTTPError(writer, layoutErrorStatus(err), err)
		return
	}
	s.mu.Lock()
	s.layout.profile = profile
	s.layout.status = "valid"
	s.layout.origin = "session"
	s.layout.diagnostics = sessionLayoutDiagnostics(s.sourceRoot)
	s.layout.canSave = s.layout.activePath != "" && s.layout.activeOrigin != "" && s.layout.status == "valid"
	s.mu.Unlock()
	s.mu.RLock()
	response := s.layout.response()
	s.mu.RUnlock()
	writeJSON(writer, http.StatusOK, response)
}

func (s *Server) handleLayoutReset(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	s.mu.Lock()
	s.layout.profile = defaultLayoutProfile()
	s.layout.status = "valid"
	s.layout.origin = "session"
	s.layout.diagnostics = sessionLayoutDiagnostics(s.sourceRoot)
	s.layout.canSave = s.layout.activePath != "" && s.layout.activeOrigin != "" && s.layout.status == "valid"
	s.mu.Unlock()
	s.mu.RLock()
	response := s.layout.response()
	s.mu.RUnlock()
	writeJSON(writer, http.StatusOK, response)
}

func (s *Server) saveActiveLayout(profile LayoutProfile) error {
	data, err := encodeLayoutConfig(profile)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	activePath := s.layout.activePath
	canSave := s.layout.canSave
	if activePath == "" || !canSave {
		if activePath == "" {
			return analysis.NewHostError(analysis.ErrSaveAsRequired, "there is no active .archview.json file; use Save As to choose a destination", nil)
		}
		return analysis.NewHostError(analysis.ErrInvalidOptions, "the nearest .archview.json is invalid; use Save As after correcting the profile", map[string]any{"path": activePath})
	}
	if err := writeLayoutConfigAtomically(activePath, data); err != nil {
		return err
	}
	s.layout.profile = profile
	s.layout.origin = s.layout.activeOrigin
	s.layout.status = "valid"
	s.layout.canSave = true
	s.layout.diagnostics = sessionLayoutDiagnostics(s.sourceRoot)
	return nil
}

func (s *Server) saveLayoutAs(profile LayoutProfile, destinationDir string, confirm bool) error {
	if !confirm {
		return analysis.NewHostError(analysis.ErrInvalidOptions, "Save As requires explicit confirmation", nil)
	}
	directory, err := normalizeLayoutDestination(destinationDir)
	if err != nil {
		return err
	}
	profile, err = validateLayoutProfile(profile)
	if err != nil {
		return err
	}
	configPath := filepath.Join(directory, layoutConfigFileName)
	data, err := encodeLayoutConfig(profile)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sourceRoot := s.sourceRoot
	if !s.layout.canSaveAs || sourceRoot == "" {
		return analysis.NewHostError(analysis.ErrPersistenceUnavailable, "Save As is unavailable for a model-only session", nil)
	}
	origin := activeLayoutOrigin(sourceRoot, configPath)
	if err := writeLayoutConfigAtomically(configPath, data); err != nil {
		return err
	}
	s.layout.profile = profile
	s.layout.activePath = configPath
	s.layout.activeOrigin = origin
	s.layout.origin = origin
	s.layout.status = "valid"
	s.layout.canSave = true
	s.layout.canSaveAs = true
	s.layout.diagnostics = nil
	return nil
}

func decodeLayoutProfileRequest(body io.Reader) (LayoutProfile, error) {
	data, err := readLayoutRequest(body)
	if err != nil {
		return LayoutProfile{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var input layoutApplyRequest
	if err := decoder.Decode(&input); err != nil {
		return LayoutProfile{}, analysis.WrapHostError(analysis.ErrInvalidRequest, "layout request is invalid JSON", err, nil)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return LayoutProfile{}, analysis.WrapHostError(analysis.ErrInvalidRequest, "layout request contains trailing data", err, nil)
	}
	if input.SchemaVersion != layoutConfigSchemaVersion {
		return LayoutProfile{}, analysis.NewHostError(analysis.ErrInvalidOptions, "layout configuration schema is unsupported", map[string]any{"schema_version": input.SchemaVersion, "expected": layoutConfigSchemaVersion})
	}
	return validateLayoutProfile(input.Layout)
}

func decodeLayoutSaveAsRequest(body io.Reader) (layoutSaveAsRequest, error) {
	data, err := readLayoutRequest(body)
	if err != nil {
		return layoutSaveAsRequest{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var input layoutSaveAsRequest
	if err := decoder.Decode(&input); err != nil {
		return layoutSaveAsRequest{}, analysis.WrapHostError(analysis.ErrInvalidRequest, "Save As request is invalid JSON", err, nil)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return layoutSaveAsRequest{}, analysis.WrapHostError(analysis.ErrInvalidRequest, "Save As request contains trailing data", err, nil)
	}
	if input.SchemaVersion != layoutConfigSchemaVersion {
		return layoutSaveAsRequest{}, analysis.NewHostError(analysis.ErrInvalidOptions, "layout configuration schema is unsupported", map[string]any{"schema_version": input.SchemaVersion, "expected": layoutConfigSchemaVersion})
	}
	profile, err := validateLayoutProfile(input.Layout)
	if err != nil {
		return layoutSaveAsRequest{}, err
	}
	input.Layout = profile
	input.DestinationDir = strings.TrimSpace(input.DestinationDir)
	return input, nil
}

func readLayoutRequest(body io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(body, maxLayoutRequestBytes+1))
	if err != nil {
		return nil, analysis.WrapHostError(analysis.ErrInvalidRequest, "layout request could not be read", err, nil)
	}
	if len(data) > maxLayoutRequestBytes {
		return nil, analysis.NewHostError(analysis.ErrInvalidRequest, "layout request is too large", map[string]any{"max_bytes": maxLayoutRequestBytes})
	}
	return data, nil
}

func normalizeLayoutDestination(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "Save As destination directory is required", nil)
	}
	absolute, err := filepath.Abs(value)
	if err != nil {
		return "", analysis.WrapHostError(analysis.ErrInvalidRequest, "Save As destination directory could not be normalized", err, map[string]any{"directory": value})
	}
	absolute = filepath.Clean(absolute)
	info, err := os.Stat(absolute)
	if err != nil {
		return "", analysis.WrapHostError(analysis.ErrUnreadableProject, "Save As destination directory could not be read", err, map[string]any{"directory": absolute})
	}
	if !info.IsDir() {
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "Save As destination must be a directory", map[string]any{"directory": absolute})
	}
	return absolute, nil
}

func sessionLayoutDiagnostics(sourceRoot string) []LayoutDiagnostic {
	if sourceRoot == "" {
		return []LayoutDiagnostic{{Code: "persistence_unavailable", Severity: "info", Message: "This model-only session can apply layout settings for the current session, but has no project directory for persistence."}}
	}
	return nil
}

func layoutErrorStatus(err error) int {
	switch analysis.ErrorCodeOf(err) {
	case analysis.ErrInvalidRequest:
		return http.StatusBadRequest
	case analysis.ErrSaveAsRequired:
		return http.StatusConflict
	case analysis.ErrPersistenceUnavailable:
		return http.StatusForbidden
	case analysis.ErrInvalidOptions, analysis.ErrUnsupportedOption:
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}
