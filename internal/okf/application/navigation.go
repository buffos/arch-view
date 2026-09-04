package application

import (
	"context"
	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/profile"
)

func (service *sessionService) Session(ctx context.Context, sessionID string) (SessionView, error) {
	if err := service.ensureReady(ctx); err != nil {
		return SessionView{}, err
	}
	if sessionID == "" {
		sessionID = DefaultSessionID
	}
	service.mu.Lock()
	session := service.sessionLocked(sessionID)
	if session.BundleID == "" {
		session.BundleID = service.catalog.DefaultBundleID
	}
	if session.ProfileID == "" {
		session.ProfileID = service.boundProfileLocked(session.BundleID)
	}
	if session.ProfileID == "" {
		session.ProfileID = profile.DefaultProfileID
	}
	if session.Depth == 0 || (session.LastSnapshot == nil && session.Request == 0) {
		session.Depth = service.defaultDepthLocked(session.ProfileID)
	}
	value := sessionViewLocked(session)
	service.mu.Unlock()
	if value.Projection == nil && value.BundleID != "" {
		return service.project(ctx, sessionID)
	}
	return value, nil
}

func (service *sessionService) SelectBundle(ctx context.Context, sessionID, bundleID string) (SessionView, error) {
	if err := service.ensureReady(ctx); err != nil {
		return SessionView{}, err
	}
	service.mu.Lock()
	if _, exists := service.indexes[bundleID]; !exists {
		service.mu.Unlock()
		return SessionView{}, domain.NewError("okf_bundle_not_found", 404, "the requested bundle is not selectable", map[string]any{"bundle_id": bundleID})
	}
	session := service.navigationCandidateLocked(sessionID)
	session.BundleID, session.ProfileID, session.FocusRoot, session.Full = bundleID, service.boundProfileLocked(bundleID), "", false
	session.Depth, session.History, session.LastSnapshot = service.defaultDepthLocked(session.ProfileID), nil, nil
	session.Request++
	service.sessionLocked(sessionID).Request = session.Request
	service.mu.Unlock()
	return service.projectCandidate(ctx, *session)
}

func (service *sessionService) SelectProfile(ctx context.Context, sessionID, profileID string) (SessionView, error) {
	if err := service.ensureReady(ctx); err != nil {
		return SessionView{}, err
	}
	if profileID == "" {
		profileID = profile.DefaultProfileID
	}
	if _, exists := service.registry.Profile(profileID); !exists {
		return SessionView{}, domain.NewError("okf_profile_not_found", 404, "the requested profile is not available", map[string]any{"profile_id": profileID})
	}
	service.mu.Lock()
	session := service.navigationCandidateLocked(sessionID)
	if session.BundleID == "" {
		session.BundleID = service.catalog.DefaultBundleID
	}
	session.ProfileID = profileID
	session.Request++
	service.sessionLocked(sessionID).Request = session.Request
	service.mu.Unlock()
	return service.projectCandidate(ctx, *session)
}

func (service *sessionService) SetNavigation(ctx context.Context, sessionID string, depth int, full bool) (SessionView, error) {
	if depth < 1 && !full {
		return SessionView{}, domain.NewError("okf_depth_invalid", 400, "navigation depth must be at least one", map[string]any{"depth": depth})
	}
	if err := service.ensureReady(ctx); err != nil {
		return SessionView{}, err
	}
	service.mu.Lock()
	session := service.navigationCandidateLocked(sessionID)
	if session.BundleID == "" {
		session.BundleID = service.catalog.DefaultBundleID
	}
	session.Depth, session.Full, session.Request = depth, full, session.Request+1
	service.sessionLocked(sessionID).Request = session.Request
	service.mu.Unlock()
	return service.projectCandidate(ctx, *session)
}

func (service *sessionService) Focus(ctx context.Context, sessionID, conceptID string) (SessionView, error) {
	if err := service.ensureReady(ctx); err != nil {
		return SessionView{}, err
	}
	service.mu.Lock()
	session := service.navigationCandidateLocked(sessionID)
	if session.BundleID == "" {
		session.BundleID = service.catalog.DefaultBundleID
	}
	index, exists := service.indexes[session.BundleID]
	if !exists {
		service.mu.Unlock()
		return SessionView{}, domain.NewError("okf_bundle_not_found", 404, "the session bundle is not available", nil)
	}
	if _, exists := index.Documents[conceptID]; !exists {
		service.mu.Unlock()
		return SessionView{}, domain.NewError("okf_concept_not_found", 404, "the requested focus concept is not in the selected bundle", map[string]any{"concept_id": conceptID})
	}
	session.History = append(session.History, domain.NavigationState{FocusRoot: session.FocusRoot, Depth: session.Depth, Full: session.Full})
	session.FocusRoot, session.Request = conceptID, session.Request+1
	service.sessionLocked(sessionID).Request = session.Request
	service.mu.Unlock()
	return service.projectCandidate(ctx, *session)
}

func (service *sessionService) TopLevel(ctx context.Context, sessionID string) (SessionView, error) {
	if err := service.ensureReady(ctx); err != nil {
		return SessionView{}, err
	}
	service.mu.Lock()
	session := service.navigationCandidateLocked(sessionID)
	if session.FocusRoot == "" {
		service.mu.Unlock()
		return service.Session(ctx, sessionID)
	}
	session.History = append(session.History, domain.NavigationState{FocusRoot: session.FocusRoot, Depth: session.Depth, Full: session.Full})
	session.FocusRoot, session.Request = "", session.Request+1
	service.sessionLocked(sessionID).Request = session.Request
	service.mu.Unlock()
	return service.projectCandidate(ctx, *session)
}

func (service *sessionService) Back(ctx context.Context, sessionID string) (SessionView, error) {
	if err := service.ensureReady(ctx); err != nil {
		return SessionView{}, err
	}
	service.mu.Lock()
	session := service.navigationCandidateLocked(sessionID)
	if len(session.History) == 0 {
		service.mu.Unlock()
		return SessionView{}, domain.NewError("okf_navigation_history_empty", 409, "there is no previous view to return to", nil)
	}
	last := session.History[len(session.History)-1]
	session.History = session.History[:len(session.History)-1]
	session.FocusRoot, session.Depth, session.Full, session.Request = last.FocusRoot, last.Depth, last.Full, session.Request+1
	service.sessionLocked(sessionID).Request = session.Request
	service.mu.Unlock()
	return service.projectCandidate(ctx, *session)
}

func (service *sessionService) Projection(ctx context.Context, sessionID string) (SessionView, error) {
	return service.Session(ctx, sessionID)
}

func (service *sessionService) project(ctx context.Context, sessionID string) (SessionView, error) {
	if sessionID == "" {
		sessionID = DefaultSessionID
	}
	service.mu.Lock()
	session := service.navigationCandidateLocked(sessionID)
	session.Request++
	service.sessionLocked(sessionID).Request = session.Request
	service.mu.Unlock()
	return service.projectCandidate(ctx, *session)
}

func (service *sessionService) navigationCandidateLocked(sessionID string) *domain.Session {
	service.sessionLocked(sessionID)
	candidate := service.sessionLockedRead(sessionID)
	return &candidate
}

func (service *sessionService) projectCandidate(ctx context.Context, session domain.Session) (SessionView, error) {
	service.mu.RLock()
	index, exists := service.indexes[session.BundleID]
	service.mu.RUnlock()
	if !exists {
		return SessionView{}, domain.NewError("okf_bundle_not_found", 404, "the session has no valid selected bundle", nil)
	}
	navigation := domain.NavigationState{CanGoBack: len(session.History) > 0, FocusRoot: session.FocusRoot, Depth: session.Depth, Full: session.Full}
	snapshot, selectedProfile, err := (projectionBuilder{profiles: service.registry}).build(ctx, index, session.ProfileID, navigation)
	if err != nil {
		return SessionView{}, err
	}
	session.ProfileID = selectedProfile
	service.mu.Lock()
	current := service.sessionLocked(session.SessionID)
	if current.Request != session.Request {
		service.mu.Unlock()
		return SessionView{}, domain.NewError("okf_operation_superseded", 409, "a newer session request superseded this projection", nil)
	}
	if err := contextErr(ctx); err != nil {
		service.mu.Unlock()
		return SessionView{}, contextOperationError(err, "projection publication")
	}
	*current = session
	current.LastSnapshot = &snapshot
	value := sessionViewLocked(current)
	service.mu.Unlock()
	return value, nil
}
