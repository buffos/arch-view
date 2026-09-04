package profile

import (
	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
	"maps"
)

// ResolveCandidate composes an unsaved declaration without publishing it to
// active sessions. Defaults are applied after the bases have been merged.
func (registry *Registry) ResolveCandidate(value domain.Profile) (domain.Profile, []domain.Diagnostic) {
	registry.mu.RLock()
	candidate := &Registry{profiles: make(map[string]domain.Profile, len(registry.profiles)+1), strategies: make(map[string]ports.RuleStrategy, len(registry.strategies))}
	candidate.shapes = registry.shapes
	candidate.detailRenderers = registry.cloneDetailRenderersLocked()
	candidate.diagnosticProviders = maps.Clone(registry.diagnosticProviders)
	candidate.extensionMetadata = maps.Clone(registry.extensionMetadata)
	for id, item := range registry.profiles {
		candidate.profiles[id] = item
	}
	for key, strategy := range registry.strategies {
		candidate.strategies[key] = strategy
	}
	registry.mu.RUnlock()
	candidate.profiles[value.ProfileID] = declaration(value)
	return candidate.ResolveProfile(value.ProfileID)
}

func declaration(value domain.Profile) domain.Profile {
	value = domain.CloneProfile(value)
	if value.Name == "" {
		value.Name = value.ProfileID
	}
	if value.Origin == "" {
		value.Origin = "project_local"
	}
	return value
}

// WithProjectProfiles returns an isolated registry for validating a complete
// proposed configuration using the same registered strategies.
func (registry *Registry) WithProjectProfiles(values []domain.Profile) *Registry {
	candidate := NewRegistry()
	candidate.shapes = registry.shapes
	registry.mu.RLock()
	candidate.detailRenderers = registry.cloneDetailRenderersLocked()
	candidate.diagnosticProviders = maps.Clone(registry.diagnosticProviders)
	candidate.extensionMetadata = maps.Clone(registry.extensionMetadata)
	for key, strategy := range registry.strategies {
		candidate.strategies[key] = strategy
	}
	registry.mu.RUnlock()
	candidate.SetProjectProfiles(values)
	return candidate
}
