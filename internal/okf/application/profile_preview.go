package application

import (
	"context"

	"github.com/buffo/arch-view/internal/okf/domain"
)

// PreviewProfile returns separate editable and effective values without publishing
// the draft to sessions or persisting it. Composition remains server-owned.
func (service *profileService) PreviewProfile(ctx context.Context, value domain.Profile) (ProfilePreview, error) {
	if err := service.ensureReady(ctx); err != nil {
		return ProfilePreview{}, err
	}
	service.configurationMu.Lock()
	defer service.configurationMu.Unlock()
	validated, diagnostics := service.ValidateProfile(ctx, value)
	effective, _ := service.registry.ResolveCandidate(validated)
	if err := ctx.Err(); err != nil {
		return ProfilePreview{}, contextOperationError(err, "profile preview")
	}
	return ProfilePreview{Profile: validated, EffectiveProfile: effective, Valid: len(diagnostics) == 0, Diagnostics: diagnostics}, nil
}
