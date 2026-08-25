package analysis

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

var (
	manifestIDPattern      = regexp.MustCompile(`^[a-z0-9]+(?:[.-][a-z0-9]+)*$`)
	semanticVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$`)
	apiVersionPattern      = regexp.MustCompile(`^arch-view\.analyzer/v([0-9]+)(?:\.([0-9]+))?$`)
)

func ValidateManifest(manifest Manifest) error {
	if !manifestIDPattern.MatchString(manifest.ID) {
		return NewHostError(ErrInvalidManifest, "analyzer id must be lowercase stable identifier text", map[string]any{"id": manifest.ID})
	}
	if !semanticVersionPattern.MatchString(manifest.Version) {
		return NewHostError(ErrInvalidManifest, "analyzer version must be semantic-version text", map[string]any{"id": manifest.ID, "version": manifest.Version})
	}
	if strings.ToLower(manifest.Language) != manifest.Language || manifest.Language == "" {
		return NewHostError(ErrInvalidManifest, "analyzer language must be lowercase and non-empty", map[string]any{"id": manifest.ID, "language": manifest.Language})
	}
	if !apiVersionCompatible(manifest.APIVersion) {
		if apiVersionPattern.MatchString(manifest.APIVersion) {
			return NewHostError(ErrAPIIncompatible, "analyzer API major version is not supported", map[string]any{"id": manifest.ID, "api_version": manifest.APIVersion, "host_api_version": AnalyzerAPIVersion})
		}
		return NewHostError(ErrInvalidManifest, "analyzer API version is malformed", map[string]any{"id": manifest.ID, "api_version": manifest.APIVersion})
	}
	if len(manifest.DetectionMarkers) == 0 {
		return NewHostError(ErrInvalidManifest, "analyzer must declare at least one detection marker", map[string]any{"id": manifest.ID})
	}
	seenCapabilities := make(map[string]struct{}, len(manifest.Capabilities))
	for _, capability := range manifest.Capabilities {
		if capability == "" {
			return NewHostError(ErrInvalidManifest, "analyzer capabilities cannot contain empty values", map[string]any{"id": manifest.ID})
		}
		if _, exists := seenCapabilities[capability]; exists {
			return NewHostError(ErrInvalidManifest, "analyzer capabilities must be unique", map[string]any{"id": manifest.ID, "capability": capability})
		}
		seenCapabilities[capability] = struct{}{}
	}
	seenOptions := make(map[string]struct{}, len(manifest.Options))
	for _, option := range manifest.Options {
		if option.Name == "" {
			return NewHostError(ErrInvalidManifest, "analyzer option name cannot be empty", map[string]any{"id": manifest.ID})
		}
		if _, exists := seenOptions[option.Name]; exists {
			return NewHostError(ErrInvalidManifest, "analyzer option names must be unique", map[string]any{"id": manifest.ID, "option": option.Name})
		}
		seenOptions[option.Name] = struct{}{}
		if _, err := normalizeOptionValue(option, option.Default); err != nil {
			return WrapHostError(ErrInvalidManifest, "analyzer option default is invalid", err, map[string]any{"id": manifest.ID, "option": option.Name})
		}
	}
	for _, marker := range manifest.DetectionMarkers {
		if marker.Kind == "" || marker.Value == "" || marker.Weight <= 0 || math.IsNaN(marker.Weight) || math.IsInf(marker.Weight, 0) {
			return NewHostError(ErrInvalidManifest, "analyzer detection markers require kind, value, and positive weight", map[string]any{"id": manifest.ID})
		}
	}
	return nil
}

func apiVersionCompatible(version string) bool {
	matches := apiVersionPattern.FindStringSubmatch(version)
	return len(matches) > 1 && matches[1] == "1"
}

func analyzerInfo(manifest Manifest) AnalyzerInfo {
	return AnalyzerInfo{
		ID:         manifest.ID,
		Version:    manifest.Version,
		Language:   manifest.Language,
		APIVersion: manifest.APIVersion,
	}
}

func ensureCandidate(candidate DetectionCandidate, manifest Manifest) (DetectionCandidate, error) {
	if candidate.AnalyzerID == "" {
		return DetectionCandidate{}, NewHostError(ErrResultInvalid, "analyzer detection candidate omitted analyzer id", map[string]any{"expected": manifest.ID})
	}
	if candidate.AnalyzerID != manifest.ID {
		return DetectionCandidate{}, NewHostError(ErrResultInvalid, "analyzer detection candidate returned the wrong analyzer id", map[string]any{"expected": manifest.ID, "actual": candidate.AnalyzerID})
	}
	if candidate.Confidence < 0 || candidate.Confidence > 1 || math.IsNaN(candidate.Confidence) || math.IsInf(candidate.Confidence, 0) {
		return DetectionCandidate{}, NewHostError(ErrResultInvalid, "analyzer detection confidence must be between zero and one", map[string]any{"id": manifest.ID, "confidence": candidate.Confidence})
	}
	if strings.TrimSpace(candidate.Reason) == "" {
		candidate.Reason = fmt.Sprintf("detected by %s", manifest.ID)
	}
	return candidate, nil
}
