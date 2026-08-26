package scene

import (
	"github.com/buffo/arch-view/internal/analysis"
)

func validDisplayMode(mode string) bool {
	switch mode {
	case "overview", "detail", "list":
		return true
	default:
		return false
	}
}

func validReferenceVisibility(value string) bool {
	switch value {
	case ReferenceVisibilityHidden, ReferenceVisibilityAggregated, ReferenceVisibilityExpanded:
		return true
	default:
		return false
	}
}

func referenceScopeFilter(values []string) (map[string]struct{}, error) {
	if len(values) == 0 {
		return nil, nil
	}
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !validReferenceScope(value) {
			return nil, analysis.NewHostError(analysis.ErrInvalidRequest, "viewer reference scope is unsupported", map[string]any{"reference_scope": value})
		}
		result[value] = struct{}{}
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

func scopeAllowed(scope string, allowed map[string]struct{}) bool {
	if len(allowed) == 0 {
		return true
	}
	_, ok := allowed[scope]
	return ok
}
