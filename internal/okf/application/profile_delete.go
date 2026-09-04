package application

import (
	"strings"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/profile"
)

type profileLookup interface {
	Profile(string) (domain.Profile, bool)
}

// deleteConfigurationProfile creates an isolated candidate with safe references.
// The caller owns full composition validation, persistence, and session updates.
func deleteConfigurationProfile(current domain.ProjectConfiguration, profileID, replacementID string, neutralFallback bool, profiles profileLookup) (domain.ProjectConfiguration, error) {
	value := domain.CloneConfiguration(current)
	found := false
	for index := range value.Profiles {
		if value.Profiles[index].ProfileID == profileID {
			found = true
			value.Profiles = append(value.Profiles[:index], value.Profiles[index+1:]...)
			break
		}
	}
	if !found {
		return domain.ProjectConfiguration{}, domain.NewError("okf_profile_not_found", 404, "the project profile is not available", nil)
	}
	replacementID = strings.TrimSpace(replacementID)
	if replacementID == profileID {
		return domain.ProjectConfiguration{}, domain.NewError("okf_delete_binding_required", 409, "a deleted profile cannot replace its own bindings", nil)
	}
	if replacementID != "" {
		if _, exists := profiles.Profile(replacementID); !exists {
			return domain.ProjectConfiguration{}, domain.NewError("okf_profile_not_found", 404, "the replacement profile is not available", nil)
		}
	}
	for index := range value.Bindings {
		if value.Bindings[index].ProfileID != profileID {
			continue
		}
		if replacementID != "" {
			value.Bindings[index].ProfileID = replacementID
		} else if neutralFallback {
			value.Bindings[index].ProfileID = profile.DefaultProfileID
		} else {
			return domain.ProjectConfiguration{}, domain.NewError("okf_delete_binding_required", 409, "profile deletion requires a replacement or neutral fallback", nil)
		}
	}
	for index := range value.Profiles {
		for baseIndex, base := range value.Profiles[index].Bases {
			if base != profileID {
				continue
			}
			if replacementID == "" && !neutralFallback {
				return domain.ProjectConfiguration{}, domain.NewError("okf_delete_binding_required", 409, "profile deletion requires a replacement for inherited references", nil)
			}
			replacement := replacementID
			if replacement == "" {
				replacement = profile.DefaultProfileID
			}
			value.Profiles[index].Bases[baseIndex] = replacement
		}
	}
	return value, nil
}
