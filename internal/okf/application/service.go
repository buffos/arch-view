package application

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/okf/configuration"
	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/hierarchy"
	"github.com/buffo/arch-view/internal/okf/ports"
	"github.com/buffo/arch-view/internal/okf/profile"
	"github.com/buffo/arch-view/internal/okf/source"
)

const DefaultSessionID = "default"

func New(root string) *Service {
	scanner := source.NewFilesystemScanner()
	return NewWithDependencies(root, scanner, scanner, configuration.NewStore(), profile.NewRegistry())
}

// NewWithDependencies builds the OKF application boundary from focused ports.
// New supplies the local filesystem implementations used by the viewer.
func NewWithDependencies(root string, scanner ports.BundleScanner, indexer ports.BundleIndexer, configStore ports.ConfigurationStore, registry *profile.Registry, relationshipAdapters ...ports.RelationshipAdapter) *Service {
	absolute, _ := filepath.Abs(root)
	if scanner == nil {
		scanner = source.NewFilesystemScanner()
	}
	if indexer == nil {
		if value, ok := scanner.(ports.BundleIndexer); ok {
			indexer = value
		} else {
			indexer = source.NewFilesystemScanner()
		}
	}
	if configStore == nil {
		configStore = configuration.NewStore()
	}
	if registry == nil {
		registry = profile.NewRegistry()
	}
	cleanRoot := filepath.Clean(absolute)
	adapters := append([]ports.RelationshipAdapter(nil), relationshipAdapters...)
	state := &applicationState{root: cleanRoot, projectID: filepath.Base(cleanRoot), catalogBuilder: catalogBuilder{scanner: scanner, indexer: indexer, adapters: adapters}, relationshipAdapters: adapters, registry: registry, configStore: configStore, indexes: make(map[string]domain.BundleIndex), sessions: make(map[string]*domain.Session)}
	return composeService(state)
}

func (service *Service) Root() string { return service.root }

// SetLayoutValidator injects the host renderer's pinned layout catalog. The
// OKF domain remains independent from the viewer implementation.
func (service *Service) SetLayoutValidator(value ports.LayoutValidator) {
	service.mu.Lock()
	service.layoutValidator = value
	service.mu.Unlock()
}

func sessionViewLocked(value *domain.Session) SessionView {
	result := SessionView{SessionID: value.SessionID, BundleID: value.BundleID, ProfileID: value.ProfileID, Navigation: domain.NavigationState{FocusRoot: value.FocusRoot, Depth: value.Depth, Full: value.Full, Breadcrumbs: breadcrumbIDs(value.FocusRoot, domain.BundleIndex{}, domain.HierarchySettings{})}, Diagnostics: []domain.Diagnostic{}}
	if value.LastSnapshot != nil {
		projection := domain.CloneSnapshot(*value.LastSnapshot)
		result.Projection = &projection
	}
	if value.LastSnapshot != nil {
		result.Navigation = value.LastSnapshot.Navigation
		result.Navigation.Breadcrumbs = append([]string(nil), value.LastSnapshot.Navigation.Breadcrumbs...)
	}
	result.Navigation.CanGoBack = len(value.History) > 0
	if result.Projection != nil {
		result.Projection.Navigation.CanGoBack = result.Navigation.CanGoBack
	}
	return result
}

func chooseDefault(candidates []domain.BundleCandidate, configured string) string {
	for _, candidate := range candidates {
		if candidate.Selectable && candidate.BundleID == configured {
			return configured
		}
	}
	for _, candidate := range candidates {
		if candidate.Selectable {
			return candidate.BundleID
		}
	}
	return ""
}

func cloneCandidates(values []domain.BundleCandidate) []domain.BundleCandidate {
	result := make([]domain.BundleCandidate, len(values))
	for index, value := range values {
		result[index] = value
		result[index].Diagnostics = domain.CloneDiagnostics(value.Diagnostics)
	}
	return result
}

func diagnosticsFromError(err error) []domain.Diagnostic {
	if errors.Is(err, context.Canceled) {
		return []domain.Diagnostic{{Code: "okf_operation_cancelled", Severity: "error", Category: "cancellation", Message: "the OKF operation was cancelled", Recovery: "Retry the operation."}}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return []domain.Diagnostic{{Code: "okf_operation_timeout", Severity: "error", Category: "cancellation", Message: "the OKF operation exceeded its processing deadline", Recovery: "Retry with a larger processing budget."}}
	}
	var value *domain.Error
	if errors.As(err, &value) {
		if len(value.Diagnostics) > 0 {
			return domain.CloneDiagnostics(value.Diagnostics)
		}
		return []domain.Diagnostic{{Code: value.Code, Severity: "error", Category: "infrastructure", Message: value.Message, Details: value.Details}}
	}
	return []domain.Diagnostic{{Code: "okf_projection_failed", Severity: "error", Category: "infrastructure", Message: err.Error()}}
}

func statusFromError(err error) string {
	var value *domain.Error
	if errors.As(err, &value) {
		for _, diagnostic := range value.Diagnostics {
			switch diagnostic.Code {
			case "okf_bundle_unavailable":
				return domain.BundleUnavailable
			case "okf_bundle_unreadable":
				return domain.BundleUnreadable
			}
		}
	}
	return domain.BundleInvalid
}

func catalogRevision(values []domain.BundleCandidate) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, value.BundleID+"\x00"+value.Status+"\x00"+value.SourceRevision)
	}
	sort.Strings(parts)
	return strings.Join(parts, "\x00")
}

func breadcrumbIDs(focus string, index domain.BundleIndex, settings domain.HierarchySettings) []string {
	if focus == "" {
		return []string{}
	}
	if _, exists := index.Documents[focus]; !exists {
		return []string{focus}
	}
	containment, _ := hierarchy.Normalize(index, settings)
	parents := make(map[string]string, len(containment))
	for _, relationship := range containment {
		parents[relationship.To] = relationship.From
	}
	result := []string{focus}
	seen := map[string]bool{focus: true}
	for current := focus; ; {
		parent, exists := parents[current]
		if !exists || seen[parent] {
			break
		}
		seen[parent] = true
		result = append([]string{parent}, result...)
		current = parent
	}
	return result
}

func staleConfigurationDiagnostics(value domain.ProjectConfiguration, indexes map[string]domain.BundleIndex, registry *profile.Registry) []domain.Diagnostic {
	diagnostics := make([]domain.Diagnostic, 0)
	if value.DefaultGraph != "" {
		if _, exists := indexes[value.DefaultGraph]; !exists {
			diagnostics = append(diagnostics, domain.Diagnostic{Code: "okf_bundle_not_found", Severity: "warning", Category: "configuration", BundleID: value.DefaultGraph, Message: "The configured default OKF bundle is not currently selectable.", Recovery: "Select an available bundle or update the project binding."})
		}
	}
	for _, binding := range value.Bindings {
		if binding.BundleID != "" {
			if _, exists := indexes[binding.BundleID]; !exists {
				diagnostics = append(diagnostics, domain.Diagnostic{Code: "okf_bundle_not_found", Severity: "warning", Category: "configuration", BundleID: binding.BundleID, ProfileID: binding.ProfileID, Message: "A configured OKF binding points to an unavailable bundle.", Recovery: "Refresh the catalog or update the binding."})
			}
		}
		if binding.ProfileID != "" {
			if _, exists := registry.Profile(binding.ProfileID); !exists {
				diagnostics = append(diagnostics, domain.Diagnostic{Code: "okf_profile_not_found", Severity: "warning", Category: "configuration", BundleID: binding.BundleID, ProfileID: binding.ProfileID, Message: "A configured OKF binding points to an unavailable profile.", Recovery: "Choose another profile or repair the project configuration."})
			}
		}
	}
	return diagnostics
}

func boundDiagnostics(values []domain.Diagnostic) []domain.Diagnostic {
	if len(values) <= 200 {
		return values
	}
	return append(append([]domain.Diagnostic(nil), values[:199]...), domain.Diagnostic{Code: "okf_diagnostics_truncated", Severity: "warning", Category: "scale", Message: "Additional diagnostics were omitted."})
}

func contextErr(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func contextOperationError(err error, operation string) *domain.Error {
	return domain.ContextOperationError(err, operation)
}
