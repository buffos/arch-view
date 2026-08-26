package viewer

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/viewer/scene"
)

func queryReferenceVisibility(value string) (string, error) {
	if value == "" {
		return scene.ReferenceVisibilityHidden, nil
	}
	switch value {
	case scene.ReferenceVisibilityHidden, scene.ReferenceVisibilityAggregated, scene.ReferenceVisibilityExpanded:
		return value, nil
	default:
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "viewer reference visibility is unsupported", map[string]any{"reference_visibility": value})
	}
}

func queryReferenceScopes(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	result := []string{}
	for _, value := range values {
		if value == "" || !validReferenceScope(value) {
			return nil, analysis.NewHostError(analysis.ErrInvalidRequest, "viewer reference scope is unsupported", map[string]any{"reference_scope": value})
		}
		result = appendUniqueString(result, value)
	}
	return result, nil
}

func validReferenceScope(value string) bool {
	switch value {
	case "standard_library", "external", "unresolved", "dynamic":
		return true
	default:
		return false
	}
}

func appendUniqueString(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func queryHierarchyPath(values []string) ([]string, error) {
	pathValues := []string{}
	for _, value := range values {
		if value == "" {
			return nil, analysis.NewHostError(analysis.ErrInvalidRequest, "viewer hierarchy path contains an empty segment", nil)
		}
		// The contract uses one query value per segment. Accepting a slash-
		// separated value as well keeps the endpoint convenient for direct use.
		for _, segment := range strings.Split(value, "/") {
			if segment == "" {
				return nil, analysis.NewHostError(analysis.ErrInvalidRequest, "viewer hierarchy path contains an empty segment", nil)
			}
			pathValues = append(pathValues, segment)
		}
	}
	return pathValues, nil
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	writer.WriteHeader(status)
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(value)
}

func writeHTTPError(writer http.ResponseWriter, status int, err error) {
	data, marshalErr := analysis.MarshalError(err)
	if marshalErr != nil {
		data = []byte(`{"error":{"code":"host_failure","message":"viewer request failed"}}`)
	}
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	writer.WriteHeader(status)
	_, _ = writer.Write(data)
	_, _ = writer.Write([]byte("\n"))
}

func writeMethodNotAllowed(writer http.ResponseWriter, allowed string) {
	writer.Header().Set("Allow", allowed)
	writeHTTPError(writer, http.StatusMethodNotAllowed, analysis.NewHostError(analysis.ErrInvalidRequest, "viewer endpoint is read-only", map[string]any{"allowed": allowed}))
}
