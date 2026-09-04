package application

import (
	"strings"

	"github.com/buffo/arch-view/internal/okf/domain"
)

// renameConfigurationProfile rewrites profile identity and references in an
// isolated declaration. Its caller owns ID normalization, persistence, and
// session transitions; rejected edits never change the input configuration.
func renameConfigurationProfile(current domain.ProjectConfiguration, oldID, newID, newName string) (domain.ProjectConfiguration, error) {
	found := false
	for _, item := range current.Profiles {
		if item.ProfileID == oldID {
			found = true
		}
		if item.ProfileID == newID && item.ProfileID != oldID {
			return domain.ProjectConfiguration{}, domain.NewError("okf_profile_id_conflict", 409, "the new profile ID is already in use", nil)
		}
	}
	if !found {
		return domain.ProjectConfiguration{}, domain.NewError("okf_profile_not_found", 404, "the project profile is not available", nil)
	}
	value := domain.CloneConfiguration(current)
	for index := range value.Profiles {
		for baseIndex, base := range value.Profiles[index].Bases {
			if base == oldID {
				value.Profiles[index].Bases[baseIndex] = newID
			}
		}
		if value.Profiles[index].ProfileID == oldID {
			value.Profiles[index].ProfileID = newID
			value.Profiles[index].Name = strings.TrimSpace(newName)
			if value.Profiles[index].Name == "" {
				value.Profiles[index].Name = newID
			}
		}
	}
	for index := range value.Bindings {
		if value.Bindings[index].ProfileID == oldID {
			value.Bindings[index].ProfileID = newID
		}
	}
	return value, nil
}
