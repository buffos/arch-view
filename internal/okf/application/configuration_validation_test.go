package application

import (
	"context"
	"errors"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/profile"
)

type rejectingConfigurationLayout struct{}

func (rejectingConfigurationLayout) Validate(domain.LayoutSettings) error {
	return errors.New("unsupported layout")
}

func TestConfigurationValidatorIsolatesProposedProfiles(t *testing.T) {
	registry := profile.NewRegistry()
	validator := configurationValidator{profiles: registry}
	valid := domain.ProjectConfiguration{
		Profiles: []domain.Profile{{ProfileID: "project:new", Bases: []string{profile.DefaultProfileID}}},
		Bindings: []domain.ProfileBinding{{BundleID: ".okf", ProfileID: "project:new"}},
	}
	if err := validator.validate(context.Background(), valid); err != nil {
		t.Fatal(err)
	}
	if _, exists := registry.Profile("project:new"); exists {
		t.Fatal("validation published proposed profile")
	}
	duplicate := domain.CloneConfiguration(valid)
	duplicate.Profiles = append(duplicate.Profiles, duplicate.Profiles[0])
	if err := validator.validate(context.Background(), duplicate); !hasApplicationCode(err, "okf_profile_invalid") {
		t.Fatalf("duplicate accepted: %v", err)
	}
	missing := domain.CloneConfiguration(valid)
	missing.Bindings[0].ProfileID = "project:missing"
	if err := validator.validate(context.Background(), missing); !hasApplicationCode(err, "okf_profile_not_found") {
		t.Fatalf("missing binding accepted: %v", err)
	}
	validator.layout = rejectingConfigurationLayout{}
	if err := validator.validate(context.Background(), valid); !hasApplicationCode(err, "okf_layout_invalid") {
		t.Fatalf("layout rejection lost: %v", err)
	}
}

func TestConfigurationValidatorRejectsCancelledEmptyDocument(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	validator := configurationValidator{profiles: profile.NewRegistry()}
	for _, value := range []domain.ProjectConfiguration{
		{},
		{Bindings: []domain.ProfileBinding{{BundleID: ".okf", ProfileID: profile.DefaultProfileID}}},
	} {
		if err := validator.validate(ctx, value); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled validation returned %v", err)
		}
	}
}
