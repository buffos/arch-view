package application

import (
	"context"
	"maps"

	"github.com/buffo/arch-view/internal/okf/domain"
)

type DiagnosticReport struct {
	Diagnostics []domain.Diagnostic `json:"diagnostics"`
	Revision    string              `json:"-"`
}

type DiagnosticQuery struct {
	ProjectID, BundleID, ProfileID, ConceptID, RelationshipID, OperationID string
	Severity, Category                                                     string
}

func (query DiagnosticQuery) matches(projectID string, value domain.Diagnostic) bool {
	return (query.ProjectID == "" || query.ProjectID == projectID) &&
		(query.BundleID == "" || value.BundleID == "" || query.BundleID == value.BundleID) &&
		(query.ProfileID == "" || query.ProfileID == value.ProfileID) &&
		(query.ConceptID == "" || query.ConceptID == value.ConceptID) &&
		(query.RelationshipID == "" || query.RelationshipID == value.RelationshipID) &&
		(query.OperationID == "" || query.OperationID == value.OperationID) &&
		(query.Severity == "" || query.Severity == value.Severity) &&
		(query.Category == "" || query.Category == value.Category)
}

// Diagnostics collects published source and configuration explanations without
// refreshing source or navigating a session. Unscoped warnings remain visible
// when the caller filters by bundle.
func (service *diagnosticService) Diagnostics(ctx context.Context, query DiagnosticQuery) (DiagnosticReport, error) {
	if err := service.ensureReady(ctx); err != nil {
		return DiagnosticReport{}, err
	}
	service.mu.RLock()
	catalog := service.catalog
	catalog.Bundles = cloneCandidates(catalog.Bundles)
	catalog.Diagnostics = domain.CloneDiagnostics(catalog.Diagnostics)
	reporter := diagnosticReporter{catalog: catalog, profiles: service.profileCatalogLocked(), indexes: maps.Clone(service.indexes)}
	service.mu.RUnlock()
	return reporter.report(ctx, query)
}

// Published indexes are immutable. Capturing their map with the catalog and
// configuration keeps one report on one publication, even during Refresh.
type diagnosticReporter struct {
	catalog  domain.BundleCatalog
	profiles profileCatalogBuilder
	indexes  map[string]domain.BundleIndex
}

func (reporter diagnosticReporter) report(ctx context.Context, query DiagnosticQuery) (DiagnosticReport, error) {
	profiles, profileErr := reporter.profiles.build()
	catalog := reporter.catalog
	diagnostics := append(domain.CloneDiagnostics(catalog.Diagnostics), profiles.Diagnostics...)
	for _, bundle := range catalog.Bundles {
		if err := ctx.Err(); err != nil {
			return DiagnosticReport{}, domain.ContextOperationError(err, "diagnostics")
		}
		if !bundle.Selectable {
			diagnostics = append(diagnostics, bundle.Diagnostics...)
			continue
		}
		index, exists := reporter.indexes[bundle.BundleID]
		if exists {
			diagnostics = append(diagnostics, index.Diagnostics...)
			provided, err := reporter.profiles.registry.Diagnose(ctx, index)
			if err != nil {
				return DiagnosticReport{}, domain.ContextOperationError(err, "diagnostics")
			}
			diagnostics = append(diagnostics, provided...)
		}
	}
	if err := ctx.Err(); err != nil {
		return DiagnosticReport{}, domain.ContextOperationError(err, "diagnostics")
	}
	if profileErr != nil {
		diagnostics = append(diagnostics, diagnosticsFromError(profileErr)...)
	}
	filtered := make([]domain.Diagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if query.matches(catalog.ProjectID, diagnostic) {
			filtered = append(filtered, diagnostic)
		}
	}
	return DiagnosticReport{Diagnostics: domain.CloneDiagnostics(boundDiagnostics(filtered)), Revision: catalog.Revision}, nil
}
