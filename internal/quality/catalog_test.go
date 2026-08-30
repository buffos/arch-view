package quality_test

import (
	"testing"

	"github.com/buffo/arch-view/internal/quality"
)

func TestDefaultCatalogBindingsAreValidForSessionRuleSelection(t *testing.T) {
	catalog := quality.NewDefaultCatalog()
	bindings := make([]quality.RuleBinding, 0, len(catalog.ListQualityRules()))
	for _, descriptor := range catalog.ListQualityRules() {
		binding, ok := catalog.DefaultRuleBinding(descriptor.ID, descriptor.Version)
		if !ok {
			t.Fatalf("default binding missing for %s@%s", descriptor.ID, descriptor.Version)
		}
		bindings = append(bindings, binding)
	}
	profile := quality.QualityProfile{
		SchemaVersion:  quality.SchemaVersion,
		ProfileID:      "profile:session",
		ProfileVersion: "1.0.0",
		EnabledRules:   bindings,
		Constraints:    []quality.ArchitectureConstraint{},
		Extensions:     []quality.ExtensionBlock{},
	}
	if _, err := quality.ValidateQualityProfile(profile, catalog); err != nil {
		t.Fatalf("default session bindings should validate: %v", err)
	}
}
