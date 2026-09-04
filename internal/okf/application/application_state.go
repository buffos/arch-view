package application

import (
	"context"
	"sync"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
	"github.com/buffo/arch-view/internal/okf/profile"
)

// applicationState is the shared publication boundary. Use-case behavior belongs
// to the focused services composed by Service, not to this synchronized data.
type applicationState struct {
	mu                   sync.RWMutex
	configurationMu      sync.Mutex
	root                 string
	projectID            string
	catalogBuilder       catalogBuilder
	relationshipAdapters []ports.RelationshipAdapter
	registry             *profile.Registry
	configStore          ports.ConfigurationStore
	layoutValidator      ports.LayoutValidator
	config               domain.ProjectConfiguration
	sourceDiagnostics    []domain.Diagnostic
	catalog              domain.BundleCatalog
	indexes              map[string]domain.BundleIndex
	sessions             map[string]*domain.Session
	operations           operationJournal
	ready                bool
	initialize           func(context.Context) (domain.BundleCatalog, error)
}

func (service *applicationState) ensureReady(ctx context.Context) error {
	if err := contextErr(ctx); err != nil {
		return contextOperationError(err, "application request")
	}
	service.mu.RLock()
	ready := service.ready
	service.mu.RUnlock()
	if ready {
		return nil
	}
	_, err := service.initialize(ctx)
	return err
}

func (service *applicationState) sessionLocked(sessionID string) *domain.Session {
	if sessionID == "" {
		sessionID = DefaultSessionID
	}
	if value, exists := service.sessions[sessionID]; exists {
		return value
	}
	value := &domain.Session{SessionID: sessionID, Depth: 2}
	service.sessions[sessionID] = value
	return value
}

func (service *applicationState) sessionLockedRead(sessionID string) domain.Session {
	if sessionID == "" {
		sessionID = DefaultSessionID
	}
	if value, exists := service.sessions[sessionID]; exists {
		copyValue := *value
		copyValue.History = append([]domain.NavigationState(nil), value.History...)
		return copyValue
	}
	return domain.Session{SessionID: sessionID, Depth: 2}
}

func (service *applicationState) boundProfileLocked(bundleID string) string {
	for _, binding := range service.config.Bindings {
		if binding.BundleID == bundleID && binding.ProfileID != "" {
			if _, ok := service.registry.Profile(binding.ProfileID); ok {
				return binding.ProfileID
			}
		}
	}
	return profile.DefaultProfileID
}

func (service *applicationState) defaultDepthLocked(profileID string) int {
	value, diagnostics := service.registry.ResolveProfile(profileID)
	if len(diagnostics) == 0 && value.Navigation.DefaultDepth > 0 {
		return value.Navigation.DefaultDepth
	}
	return 2
}
