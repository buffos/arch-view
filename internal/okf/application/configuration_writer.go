package application

import (
	"context"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
)

// configurationWriter prepares and persists a valid candidate. It has no access
// to sessions or published application state; the caller publishes only success.
type configurationWriter struct {
	root      string
	store     ports.ConfigurationStore
	validator configurationValidator
}

func (writer configurationWriter) save(ctx context.Context, candidate domain.ProjectConfiguration, expectedRevision, operationID string, input []byte) (domain.ProjectConfiguration, error) {
	if err := writer.validator.validate(ctx, candidate); err != nil {
		return domain.ProjectConfiguration{}, err
	}
	value := domain.CloneConfiguration(candidate)
	if expectedRevision == "" {
		expectedRevision = value.Revision
	}
	for index := range value.Profiles {
		value.Profiles[index].Revision = domain.ProfileRevision(value.Profiles[index])
	}
	return writer.store.Save(ctx, writer.root, value, expectedRevision, operationID, input)
}
