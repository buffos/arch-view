package application

import (
	"context"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/profile"
	"github.com/buffo/arch-view/internal/okf/projection"
)

// projectionBuilder prepares a result without publishing or superseding sessions.
type projectionBuilder struct{ profiles *profile.Registry }

func (builder projectionBuilder) build(ctx context.Context, index domain.BundleIndex, profileID string, navigation domain.NavigationState) (domain.ProjectionSnapshot, string, error) {
	effective, diagnostics := builder.profiles.ResolveProfile(profileID)
	if effective.ProfileID == "" || effective.Status == domain.ProfileInvalid {
		effective, _ = builder.profiles.ResolveProfile(profile.DefaultProfileID)
		profileID = effective.ProfileID
	}
	navigation.Breadcrumbs = breadcrumbIDs(navigation.FocusRoot, index, effective.Hierarchy)
	snapshot, err := projection.Build(ctx, index, effective, navigation, builder.profiles)
	if err != nil {
		return domain.ProjectionSnapshot{}, profileID, err
	}
	snapshot.Diagnostics = append(snapshot.Diagnostics, diagnostics...)
	return snapshot, profileID, nil
}
