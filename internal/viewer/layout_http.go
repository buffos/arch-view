package viewer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/viewer/layout"
)

const maxLayoutRequestBytes = 1 << 20

type layoutApplyRequest struct {
	SchemaVersion string               `json:"schema_version"`
	Layout        layout.LayoutProfile `json:"layout"`
}

type layoutSaveAsRequest struct {
	SchemaVersion  string               `json:"schema_version"`
	Layout         layout.LayoutProfile `json:"layout"`
	DestinationDir string               `json:"destination_dir"`
	Confirm        bool                 `json:"confirm"`
}

func (s *Server) handleLayoutOptions(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}
	writeJSON(writer, http.StatusOK, layout.Catalog())
}

func (s *Server) handleLayoutConfig(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet:
		s.mu.RLock()
		response := s.layout.Response()
		s.mu.RUnlock()
		writeJSON(writer, http.StatusOK, response)
	case http.MethodPut:
		profile, err := decodeLayoutProfileRequest(request.Body)
		if err != nil {
			writeHTTPError(writer, layoutErrorStatus(err), err)
			return
		}
		s.mu.Lock()
		err = s.layout.SaveActive(profile)
		response := s.layout.Response()
		s.mu.Unlock()
		if err != nil {
			writeHTTPError(writer, layoutErrorStatus(err), err)
			return
		}
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
	s.mu.Lock()
	err = s.layout.SaveAs(input.Layout, input.DestinationDir, input.Confirm)
	response := s.layout.Response()
	s.mu.Unlock()
	if err != nil {
		writeHTTPError(writer, layoutErrorStatus(err), err)
		return
	}
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
	err = s.layout.Apply(profile)
	response := s.layout.Response()
	s.mu.Unlock()
	if err != nil {
		writeHTTPError(writer, layoutErrorStatus(err), err)
		return
	}
	writeJSON(writer, http.StatusOK, response)
}

func (s *Server) handleLayoutReset(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	s.mu.Lock()
	s.layout.Reset()
	response := s.layout.Response()
	s.mu.Unlock()
	writeJSON(writer, http.StatusOK, response)
}

func decodeLayoutProfileRequest(body io.Reader) (layout.LayoutProfile, error) {
	data, err := readLayoutRequest(body)
	if err != nil {
		return layout.LayoutProfile{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var input layoutApplyRequest
	if err := decoder.Decode(&input); err != nil {
		return layout.LayoutProfile{}, analysis.WrapHostError(analysis.ErrInvalidRequest, "layout request is invalid JSON", err, nil)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return layout.LayoutProfile{}, analysis.WrapHostError(analysis.ErrInvalidRequest, "layout request contains trailing data", err, nil)
	}
	if input.SchemaVersion != layout.ConfigSchemaVersion && input.SchemaVersion != "arch-view.config/v2" {
		return layout.LayoutProfile{}, analysis.NewHostError(analysis.ErrInvalidOptions, "layout configuration schema is unsupported", map[string]any{"schema_version": input.SchemaVersion, "expected": layout.ConfigSchemaVersion})
	}
	return layout.ValidateProfile(input.Layout)
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
	if input.SchemaVersion != layout.ConfigSchemaVersion && input.SchemaVersion != "arch-view.config/v2" {
		return layoutSaveAsRequest{}, analysis.NewHostError(analysis.ErrInvalidOptions, "layout configuration schema is unsupported", map[string]any{"schema_version": input.SchemaVersion, "expected": layout.ConfigSchemaVersion})
	}
	profile, err := layout.ValidateProfile(input.Layout)
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

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("more than one JSON value")
		}
		return err
	}
	return nil
}

func layoutErrorStatus(err error) int {
	if strings.HasPrefix(string(analysis.ErrorCodeOf(err)), "renderer_feature_") {
		return http.StatusUnprocessableEntity
	}
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
