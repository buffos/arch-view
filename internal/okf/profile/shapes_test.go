package profile

import (
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

type profileShape struct{}

func (profileShape) Definition() domain.ShapeDefinition {
	return domain.ShapeDefinition{ID: "custom.shape", Version: "2", Description: "Custom rectangle", Geometry: "rectangle", Content: domain.ShapeBox{Width: 1, Height: 1}}
}

func TestProfileShapeValidationUsesEffectiveRegisteredReferences(t *testing.T) {
	registry := NewRegistry()
	if err := registry.RegisterShape(profileShape{}); err != nil {
		t.Fatal(err)
	}
	for _, reference := range []string{"rectangle", "okf.shape.diamond@1", "custom.shape@2", "custom.shape@1"} {
		base := domain.Profile{ProfileID: "project:base", Style: domain.StyleSettings{Tokens: map[string]domain.StyleToken{"custom": {Shape: reference}}}}
		registry.SetProjectProfiles([]domain.Profile{base})
		child := domain.Profile{ProfileID: "project:child", Bases: []string{base.ProfileID}}
		checked, diagnostics := Validate(child, registry)
		if reference == "custom.shape@1" {
			if checked.Status != domain.ProfileInvalid || len(diagnostics) != 1 || diagnostics[0].Code != "okf_shape_unsupported" {
				t.Fatalf("missing version accepted: %+v %+v", checked, diagnostics)
			}
		} else if checked.Status != domain.ProfileValid || len(diagnostics) != 0 {
			t.Fatalf("registered shape rejected: %s %+v", reference, diagnostics)
		}
	}
}
