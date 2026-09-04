package application

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/profile"
)

func (service *profileService) Bind(ctx context.Context, bundleID, profileID, expectedRevision, operationID string, input []byte) (domain.ProjectConfiguration, error) {
	input = commandInput("bind", input, bundleID, profileID, expectedRevision)
	if err := service.ensureReady(ctx); err != nil {
		return domain.ProjectConfiguration{}, err
	}
	service.configurationMu.Lock()
	defer service.configurationMu.Unlock()
	if value, err, replayed := service.operations.replay(operationID, input); replayed {
		return value, err
	}
	if profileID == "" {
		profileID = profile.DefaultProfileID
	}
	service.mu.Lock()
	if _, exists := service.indexes[bundleID]; !exists {
		service.mu.Unlock()
		return domain.ProjectConfiguration{}, domain.NewError("okf_bundle_not_found", 404, "the requested bundle is not selectable", nil)
	}
	if _, exists := service.registry.Profile(profileID); !exists {
		service.mu.Unlock()
		return domain.ProjectConfiguration{}, domain.NewError("okf_profile_not_found", 404, "the requested profile is not available", nil)
	}
	value := cloneConfiguration(service.config)
	value.DefaultGraph = bundleID
	value.Bindings = setBinding(value.Bindings, domain.ProfileBinding{BundleID: bundleID, ProfileID: profileID})
	service.mu.Unlock()
	return service.saveConfiguration(ctx, value, expectedRevision, operationID, input)
}

func (service *profileService) SaveProfile(ctx context.Context, value domain.Profile, expectedRevision, operationID string, input []byte) (domain.ProjectConfiguration, error) {
	input, err := profileCommandInput("save", input, value, value.ProfileID, expectedRevision)
	if err != nil {
		return domain.ProjectConfiguration{}, err
	}
	if err := service.ensureReady(ctx); err != nil {
		return domain.ProjectConfiguration{}, err
	}
	service.configurationMu.Lock()
	defer service.configurationMu.Unlock()
	if saved, err, replayed := service.operations.replay(operationID, input); replayed {
		return saved, err
	}
	configValue, lifecycle := service.profileEditContext()
	configValue, err = lifecycle.save(ctx, configValue, value)
	if err != nil {
		return domain.ProjectConfiguration{}, err
	}
	return service.saveConfiguration(ctx, configValue, expectedRevision, operationID, input)
}

func (service *profileService) SaveProfileAs(ctx context.Context, value domain.Profile, sourceID, newID, expectedRevision, operationID string, input []byte) (domain.ProjectConfiguration, error) {
	input, err := profileCommandInput("save-as", input, value, sourceID, newID, expectedRevision)
	if err != nil {
		return domain.ProjectConfiguration{}, err
	}
	if err := service.ensureReady(ctx); err != nil {
		return domain.ProjectConfiguration{}, err
	}
	service.configurationMu.Lock()
	defer service.configurationMu.Unlock()
	if saved, err, replayed := service.operations.replay(operationID, input); replayed {
		return saved, err
	}
	configValue, lifecycle := service.profileEditContext()
	configValue, err = lifecycle.saveAs(ctx, configValue, value, sourceID, newID)
	if err != nil {
		return domain.ProjectConfiguration{}, err
	}
	return service.saveConfiguration(ctx, configValue, expectedRevision, operationID, input)
}

func (service *profileService) RenameProfile(ctx context.Context, oldID, newID, newName, expectedRevision, operationID string, input []byte) (domain.ProjectConfiguration, error) {
	input = commandInput("rename", input, oldID, newID, newName, expectedRevision)
	if err := service.ensureReady(ctx); err != nil {
		return domain.ProjectConfiguration{}, err
	}
	service.configurationMu.Lock()
	defer service.configurationMu.Unlock()
	if saved, err, replayed := service.operations.replay(operationID, input); replayed {
		return saved, err
	}
	if strings.HasPrefix(oldID, "builtin:") {
		return domain.ProjectConfiguration{}, domain.NewError("okf_builtin_immutable", 403, "built-in profiles cannot be renamed", nil)
	}
	newID = projectProfileID(newID)
	if newID == "" {
		return domain.ProjectConfiguration{}, domain.NewError("okf_profile_invalid", 400, "the new profile ID is required", nil)
	}
	service.mu.Lock()
	configValue, err := renameConfigurationProfile(service.config, oldID, newID, newName)
	if err != nil {
		service.mu.Unlock()
		return domain.ProjectConfiguration{}, err
	}
	renameSessions := make(map[string]uint64)
	for sessionID, session := range service.sessions {
		if session.ProfileID == oldID {
			renameSessions[sessionID] = session.Request
		}
	}
	service.mu.Unlock()
	saved, err := service.saveConfiguration(ctx, configValue, expectedRevision, operationID, input)
	if err != nil {
		return domain.ProjectConfiguration{}, err
	}
	service.mu.Lock()
	for sessionID, previousRequest := range renameSessions {
		if session, exists := service.sessions[sessionID]; exists && session.Request == previousRequest+1 && session.ProfileID == profile.DefaultProfileID {
			session.ProfileID = newID
			session.LastSnapshot = nil
			session.Request++
		}
	}
	service.mu.Unlock()
	return saved, nil
}

func (service *profileService) DeleteProfile(ctx context.Context, profileID, replacementID string, neutralFallback bool, expectedRevision, operationID string, input []byte) (domain.ProjectConfiguration, error) {
	fallback, _ := json.Marshal(neutralFallback)
	input = commandInput("delete", input, profileID, replacementID, string(fallback), expectedRevision)
	if err := service.ensureReady(ctx); err != nil {
		return domain.ProjectConfiguration{}, err
	}
	service.configurationMu.Lock()
	defer service.configurationMu.Unlock()
	if saved, err, replayed := service.operations.replay(operationID, input); replayed {
		return saved, err
	}
	if strings.HasPrefix(profileID, "builtin:") {
		return domain.ProjectConfiguration{}, domain.NewError("okf_builtin_immutable", 403, "built-in profiles cannot be deleted", nil)
	}
	service.mu.Lock()
	configValue, err := deleteConfigurationProfile(service.config, profileID, replacementID, neutralFallback, service.registry)
	service.mu.Unlock()
	if err != nil {
		return domain.ProjectConfiguration{}, err
	}
	return service.saveConfiguration(ctx, configValue, expectedRevision, operationID, input)
}

func (service *profileService) profileEditContext() (domain.ProjectConfiguration, profileLifecycle) {
	service.mu.RLock()
	defer service.mu.RUnlock()
	return cloneConfiguration(service.config), profileLifecycle{
		validator: configurationValidator{profiles: service.registry, layout: service.layoutValidator},
	}
}

func (service *profileService) saveConfiguration(ctx context.Context, value domain.ProjectConfiguration, expectedRevision, operationID string, input []byte) (domain.ProjectConfiguration, error) {
	service.mu.RLock()
	writer := configurationWriter{
		root: service.root, store: service.configStore,
		validator: configurationValidator{profiles: service.registry, layout: service.layoutValidator},
	}
	service.mu.RUnlock()
	operationID = strings.TrimSpace(operationID)
	saved, err := writer.save(ctx, value, expectedRevision, operationID, input)
	if err != nil {
		return domain.ProjectConfiguration{}, err
	}
	service.mu.Lock()
	service.config = saved
	service.registry.SetProjectProfiles(saved.Profiles)
	service.catalog.Diagnostics = append(domain.CloneDiagnostics(service.sourceDiagnostics), staleConfigurationDiagnostics(saved, service.indexes, service.registry)...)
	service.catalog.DefaultBundleID = chooseDefault(service.catalog.Bundles, saved.DefaultGraph)
	for _, session := range service.sessions {
		session.LastSnapshot = nil
		session.Request++
		if session.ProfileID != "" {
			if _, exists := service.registry.Profile(session.ProfileID); !exists {
				session.ProfileID = profile.DefaultProfileID
			}
		}
	}
	service.operations.record(operationID, input, saved)
	service.mu.Unlock()
	return saved, nil
}

func cloneConfiguration(value domain.ProjectConfiguration) domain.ProjectConfiguration {
	return domain.CloneConfiguration(value)
}

func setBinding(values []domain.ProfileBinding, binding domain.ProfileBinding) []domain.ProfileBinding {
	for index := range values {
		if values[index].BundleID == binding.BundleID {
			values[index] = binding
			return values
		}
	}
	return append(values, binding)
}

func upsertProfile(values []domain.Profile, value domain.Profile) []domain.Profile {
	for index := range values {
		if values[index].ProfileID == value.ProfileID {
			values[index] = domain.CloneProfile(value)
			return values
		}
	}
	return append(values, domain.CloneProfile(value))
}

func containsProfile(values []domain.Profile, profileID string) bool {
	for _, value := range values {
		if value.ProfileID == profileID {
			return true
		}
	}
	return false
}

func projectProfileID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "project:") {
		return value
	}
	return "project:" + value
}

func diagnosticsFromErrorValue(err error) []domain.Diagnostic { return diagnosticsFromError(err) }

func commandInput(command string, body []byte, arguments ...string) []byte {
	encoded, _ := json.Marshal(struct {
		Command   string
		Arguments []string
		Body      []byte
	}{command, arguments, body})
	return encoded
}

func profileCommandInput(command string, body []byte, value domain.Profile, arguments ...string) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, domain.NewError("okf_profile_invalid", 400, "profile cannot be encoded", map[string]any{"profile_id": value.ProfileID})
	}
	return commandInput(command, body, append(arguments, string(encoded))...), nil
}
