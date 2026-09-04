package application

import (
	"reflect"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/profile"
)

type availableProfiles map[string]domain.Profile

func (values availableProfiles) Profile(id string) (domain.Profile, bool) {
	value, exists := values[id]
	return value, exists
}

func TestDeleteConfigurationProfilePreservesInputAndReassignsEveryReference(t *testing.T) {
	current := domain.ProjectConfiguration{
		Profiles: []domain.Profile{{ProfileID: "project:base"}, {ProfileID: "project:child", Bases: []string{"project:base"}}},
		Bindings: []domain.ProfileBinding{{BundleID: "a/.okf", ProfileID: "project:base"}, {BundleID: "b/.okf", ProfileID: "project:base"}},
	}
	before := domain.CloneConfiguration(current)
	lookup := availableProfiles{"project:replacement": {ProfileID: "project:replacement"}}
	for _, replacement := range []string{"project:replacement", ""} {
		deleted, err := deleteConfigurationProfile(current, "project:base", replacement, replacement == "", lookup)
		if err != nil {
			t.Fatal(err)
		}
		want := replacement
		if want == "" {
			want = profile.DefaultProfileID
		}
		if len(deleted.Profiles) != 1 || deleted.Profiles[0].Bases[0] != want {
			t.Fatalf("inheritance: %+v", deleted.Profiles)
		}
		for _, binding := range deleted.Bindings {
			if binding.ProfileID != want {
				t.Fatalf("binding: %+v", binding)
			}
		}
		if !reflect.DeepEqual(current, before) {
			t.Fatal("deletion mutated input")
		}
		deleted.Profiles[0].Bases[0] = "changed"
		if !reflect.DeepEqual(current, before) {
			t.Fatal("result aliases input")
		}
	}
	for _, scenario := range []struct{ replacement, code string }{
		{"", "okf_delete_binding_required"},
		{"project:base", "okf_delete_binding_required"},
		{"project:missing", "okf_profile_not_found"},
	} {
		_, err := deleteConfigurationProfile(current, "project:base", scenario.replacement, false, lookup)
		if !hasApplicationCode(err, scenario.code) {
			t.Fatalf("rejection: %v", err)
		}
		if !reflect.DeepEqual(current, before) {
			t.Fatal("rejected deletion mutated input")
		}
	}
}
