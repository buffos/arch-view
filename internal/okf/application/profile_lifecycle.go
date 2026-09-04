package application

import (
	"context"
	"strings"

	"github.com/buffo/arch-view/internal/okf/domain"
)

// profileLifecycle prepares profile edits without owning persistence or sessions.
type profileLifecycle struct {
	validator configurationValidator
}

func (lifecycle profileLifecycle) save(ctx context.Context, config domain.ProjectConfiguration, value domain.Profile) (domain.ProjectConfiguration, error) {
	if strings.HasPrefix(value.ProfileID, "builtin:") || value.Origin == "builtin" {
		return domain.ProjectConfiguration{}, domain.NewError("okf_builtin_immutable", 403, "built-in profiles cannot be overwritten", nil)
	}
	if strings.TrimSpace(value.ProfileID) == "" {
		return domain.ProjectConfiguration{}, domain.NewError("okf_profile_invalid", 400, "a project-local profile ID is required", nil)
	}
	value.ProfileID = projectProfileID(value.ProfileID)
	value.Origin, value.Immutable = "project_local", false
	validated, diagnostics := lifecycle.validator.validateProfile(ctx, value)
	if len(diagnostics) > 0 {
		return domain.ProjectConfiguration{}, domain.NewError("okf_profile_invalid", 400, "profile validation failed", nil).WithDiagnostics(diagnostics...)
	}
	if !containsProfile(config.Profiles, validated.ProfileID) {
		return domain.ProjectConfiguration{}, domain.NewError("okf_profile_not_found", 404, "Save updates an existing project-local profile; use Save As to create one", map[string]any{"profile_id": validated.ProfileID})
	}
	config = cloneConfiguration(config)
	config.Profiles = upsertProfile(config.Profiles, validated)
	return config, nil
}

func (lifecycle profileLifecycle) saveAs(ctx context.Context, config domain.ProjectConfiguration, value domain.Profile, sourceID, newID string) (domain.ProjectConfiguration, error) {
	if newID == "" {
		return domain.ProjectConfiguration{}, domain.NewError("okf_profile_invalid", 400, "Save As requires a new profile ID", nil)
	}
	if value.ProfileID == "" && sourceID != "" {
		source, exists := lifecycle.validator.profiles.Profile(sourceID)
		if !exists {
			return domain.ProjectConfiguration{}, domain.NewError("okf_profile_not_found", 404, "the source profile is not available", nil)
		}
		value = source
	}
	value.ProfileID = projectProfileID(newID)
	value.Origin, value.Immutable = "project_local", false
	if _, exists := lifecycle.validator.profiles.Profile(value.ProfileID); exists {
		return domain.ProjectConfiguration{}, domain.NewError("okf_profile_id_conflict", 409, "a profile with this ID already exists", map[string]any{"profile_id": value.ProfileID})
	}
	validated, diagnostics := lifecycle.validator.validateProfile(ctx, value)
	if len(diagnostics) > 0 {
		return domain.ProjectConfiguration{}, domain.NewError("okf_profile_invalid", 400, "profile validation failed", nil).WithDiagnostics(diagnostics...)
	}
	config = cloneConfiguration(config)
	config.Profiles = upsertProfile(config.Profiles, validated)
	return config, nil
}
