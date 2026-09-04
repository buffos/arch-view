package application

import (
	"context"
	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/profile"
)

// profileCatalogBuilder resolves a captured configuration outside the application
// lock, since registered parameter validators may execute host code.
type profileCatalogBuilder struct {
	registry    *profile.Registry
	bindings    []domain.ProfileBinding
	revision    string
	diagnostics []domain.Diagnostic
}

func (service *applicationState) profileCatalogLocked() profileCatalogBuilder {
	return profileCatalogBuilder{
		registry:    service.registry.WithProjectProfiles(service.config.Profiles),
		bindings:    append([]domain.ProfileBinding(nil), service.config.Bindings...),
		revision:    service.config.Revision,
		diagnostics: domain.CloneDiagnostics(service.catalog.Diagnostics),
	}
}

func (builder profileCatalogBuilder) build() (domain.ProfileCatalog, error) {
	profiles := builder.registry.Profiles()
	diagnostics := domain.CloneDiagnostics(builder.diagnostics)
	for i := range profiles {
		resolved, explanations := builder.registry.ResolveProfile(profiles[i].ProfileID)
		profiles[i].Status = resolved.Status
		diagnostics = append(diagnostics, explanations...)
	}
	revision, err := builder.registry.Revision()
	if err != nil {
		return domain.ProfileCatalog{}, err
	}
	return domain.ProfileCatalog{Profiles: profiles, Bindings: builder.bindings, RegistryRevision: revision, ConfigurationRevision: builder.revision, Diagnostics: boundDiagnostics(diagnostics)}, nil
}

func (service *profileService) Profiles(ctx context.Context) (domain.ProfileCatalog, error) {
	if err := service.ensureReady(ctx); err != nil {
		return domain.ProfileCatalog{}, err
	}
	service.mu.RLock()
	builder := service.profileCatalogLocked()
	service.mu.RUnlock()
	return builder.build()
}

func (service *profileService) ValidateProfile(ctx context.Context, value domain.Profile) (domain.Profile, []domain.Diagnostic) {
	service.mu.RLock()
	validator := configurationValidator{profiles: service.registry, layout: service.layoutValidator}
	service.mu.RUnlock()
	return validator.validateProfile(ctx, value)
}
