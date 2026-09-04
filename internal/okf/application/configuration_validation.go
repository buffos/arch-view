package application

import (
	"context"
	"strings"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
	"github.com/buffo/arch-view/internal/okf/profile"
)

// configurationValidator checks proposed documents without accessing sessions or storage.
type configurationValidator struct {
	profiles *profile.Registry
	layout   ports.LayoutValidator
}

func (validator configurationValidator) validateProfile(ctx context.Context, value domain.Profile) (domain.Profile, []domain.Diagnostic) {
	if err := contextErr(ctx); err != nil {
		return value, []domain.Diagnostic{{Code: "okf_operation_cancelled", Severity: "error", Category: "cancellation", Message: err.Error()}}
	}
	validated, diagnostics := profile.Validate(value, validator.profiles)
	if validator.layout != nil {
		effective, _ := validator.profiles.ResolveCandidate(validated)
		if err := validator.layout.Validate(effective.Layout); err != nil {
			diagnostics = append(diagnostics, domain.Diagnostic{Code: "okf_layout_invalid", Severity: "error", Category: "profile", ProfileID: validated.ProfileID, Message: err.Error(), Recovery: "Choose an algorithm and options from the pinned ELK catalog."})
			validated.Status = domain.ProfileInvalid
		}
	}
	return validated, diagnostics
}

func (validator configurationValidator) validate(ctx context.Context, value domain.ProjectConfiguration) error {
	if err := contextErr(ctx); err != nil {
		return contextOperationError(err, "configuration validation")
	}
	registry := validator.profiles.WithProjectProfiles(value.Profiles)
	seen := make(map[string]bool)
	for _, item := range value.Profiles {
		if err := contextErr(ctx); err != nil {
			return contextOperationError(err, "configuration validation")
		}
		if seen[item.ProfileID] || item.ProfileID == "" || strings.HasPrefix(item.ProfileID, "builtin:") {
			return domain.NewError("okf_profile_invalid", 400, "project profiles require unique non-builtin identities", map[string]any{"profile_id": item.ProfileID})
		}
		seen[item.ProfileID] = true
		_, diagnostics := profile.Validate(item, registry)
		if len(diagnostics) != 0 {
			return domain.NewError("okf_profile_invalid", 400, "the proposed configuration contains an invalid profile", nil).WithDiagnostics(diagnostics...)
		}
		effective, _ := registry.ResolveProfile(item.ProfileID)
		if validator.layout != nil {
			if err := validator.layout.Validate(effective.Layout); err != nil {
				return domain.WrapError("okf_layout_invalid", 400, "the proposed profile layout is invalid", err)
			}
		}
	}
	for _, binding := range value.Bindings {
		if err := contextErr(ctx); err != nil {
			return contextOperationError(err, "configuration validation")
		}
		if binding.ProfileID == "" {
			continue
		}
		if _, exists := registry.Profile(binding.ProfileID); !exists {
			return domain.NewError("okf_profile_not_found", 400, "a binding refers to an unavailable profile", map[string]any{"profile_id": binding.ProfileID})
		}
	}
	return nil
}
