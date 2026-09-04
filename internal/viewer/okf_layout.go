package viewer

import (
	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/viewer/layout"
)

type okfLayoutValidator struct{}

func (okfLayoutValidator) Validate(value domain.LayoutSettings) error {
	_, err := layout.ValidateProfile(layout.LayoutProfile{Algorithm: value.Algorithm, Options: value.Options, Features: value.Features})
	return err
}
