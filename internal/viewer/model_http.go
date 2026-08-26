package viewer

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/viewer/scene"
)

func (s *Server) handleModel(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}
	value := s.snapshot()
	trimmed := strings.TrimPrefix(request.URL.Path, "/v1/models/")
	parts := strings.Split(trimmed, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(writer, request)
		return
	}
	modelID, err := url.PathUnescape(parts[0])
	if err != nil || modelID != value.ModelID {
		writeHTTPError(writer, http.StatusNotFound, analysis.NewHostError(analysis.ErrInvalidModel, "requested model was not found", map[string]any{"model_id": modelID}))
		return
	}
	if len(parts) == 1 {
		writeJSON(writer, http.StatusOK, value)
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
