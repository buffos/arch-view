package application

import (
	"context"
	"errors"

	"github.com/buffo/arch-view/internal/okf/configuration"
	"github.com/buffo/arch-view/internal/okf/domain"
)

func (service *catalogService) Refresh(ctx context.Context) (domain.BundleCatalog, error) {
	service.configurationMu.Lock()
	defer service.configurationMu.Unlock()
	built, err := service.catalogBuilder.build(ctx, service.root)
	if err != nil {
		return domain.BundleCatalog{}, err
	}
	candidates, indexes, diagnostics := built.candidates, built.indexes, built.diagnostics
	configValue, configErr := service.configStore.Load(ctx, service.root)
	if configErr != nil {
		if errors.Is(configErr, context.Canceled) || errors.Is(configErr, context.DeadlineExceeded) {
			return domain.BundleCatalog{}, contextOperationError(configErr, "catalog refresh")
		}
		diagnostics = append(diagnostics, diagnosticsFromError(configErr)...)
		configValue = domain.ProjectConfiguration{SchemaVersion: configuration.SectionSchemaVersion}
	}
	service.mu.Lock()
	if err := contextErr(ctx); err != nil {
		service.mu.Unlock()
		return domain.BundleCatalog{}, contextOperationError(err, "catalog refresh publication")
	}
	service.indexes = indexes
	service.sourceDiagnostics = domain.CloneDiagnostics(built.diagnostics)
	service.config = configValue
	service.registry.SetProjectProfiles(configValue.Profiles)
	diagnostics = append(diagnostics, staleConfigurationDiagnostics(configValue, indexes, service.registry)...)
	defaultBundle := chooseDefault(candidates, configValue.DefaultGraph)
	service.catalog = domain.BundleCatalog{ProjectID: service.projectID, DefaultBundleID: defaultBundle, Bundles: cloneCandidates(candidates), Diagnostics: append([]domain.Diagnostic(nil), diagnostics...), Revision: catalogRevision(candidates)}
	service.ready = true
	for _, session := range service.sessions {
		session.LastSnapshot = nil
		session.Request++
	}
	service.mu.Unlock()
	return service.Catalog(), nil
}

func (service *catalogService) Catalog() domain.BundleCatalog {
	service.mu.RLock()
	defer service.mu.RUnlock()
	result := service.catalog
	result.Bundles = cloneCandidates(service.catalog.Bundles)
	result.Diagnostics = domain.CloneDiagnostics(service.catalog.Diagnostics)
	return result
}

func (service *catalogService) Summary(ctx context.Context, bundleID string) (domain.BundleSummary, error) {
	if err := service.ensureReady(ctx); err != nil {
		return domain.BundleSummary{}, err
	}
	service.mu.RLock()
	index, exists := service.indexes[bundleID]
	service.mu.RUnlock()
	if !exists {
		return domain.BundleSummary{}, domain.NewError("okf_bundle_not_found", 404, "the requested bundle is not valid or was not discovered", map[string]any{"bundle_id": bundleID})
	}
	files := append([]string(nil), index.ConceptOrder...)
	return domain.BundleSummary{BundleID: bundleID, SourceRevision: index.SourceRevision, ConceptCount: len(index.ConceptOrder), LinkCount: len(index.Relationships), Files: files, Diagnostics: domain.CloneDiagnostics(index.Diagnostics)}, nil
}
