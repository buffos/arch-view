package application

import (
	"context"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/interaction"
	"github.com/buffo/arch-view/internal/okf/ports"
	"github.com/buffo/arch-view/internal/okf/profile"
)

type inspectionProfiles interface {
	ports.RuleEvaluator
	ports.DetailRendererResolver
	Profile(string) (domain.Profile, bool)
	ResolveProfile(string) (domain.Profile, []domain.Diagnostic)
}

// conceptInspector owns detail assembly, not session selection or persistence.
type conceptInspector struct {
	profiles inspectionProfiles
}

func (inspector conceptInspector) inspect(ctx context.Context, index domain.BundleIndex, profileID, conceptID string) (domain.ConceptDetail, error) {
	profileValue, exists := inspector.profiles.Profile(profileID)
	if !exists {
		profileValue, _ = inspector.profiles.Profile(profile.DefaultProfileID)
	}
	effective, diagnostics := inspector.profiles.ResolveProfile(profileValue.ProfileID)
	if effective.ProfileID == "" || effective.Status == domain.ProfileInvalid {
		effective, _ = inspector.profiles.ResolveProfile(profile.DefaultProfileID)
	}
	var renderer ports.DetailRenderer
	if selection := effective.Details.Renderer; selection != nil {
		renderer, _ = inspector.profiles.ResolveDetailRenderer(selection.ID, selection.Version)
	}
	value, err := interaction.DetailWithRenderer(ctx, index, effective, conceptID, inspector.profiles, renderer)
	if err != nil {
		return value, err
	}
	value.Diagnostics = append(value.Diagnostics, diagnostics...)
	return value, nil
}
