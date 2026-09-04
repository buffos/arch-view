package application

import (
	"reflect"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestRenameConfigurationProfileRewritesAllReferencesWithoutMutatingInput(t *testing.T) {
	current := domain.ProjectConfiguration{
		Profiles: []domain.Profile{
			{ProfileID: "project:base", Name: "Base", Layout: domain.LayoutSettings{Options: map[string]any{"custom": "retained"}}},
			{ProfileID: "project:child", Bases: []string{"builtin:neutral", "project:base"}},
		},
		Bindings: []domain.ProfileBinding{
			{BundleID: "a/.okf", ProfileID: "project:base"},
			{BundleID: "b/.okf", ProfileID: "project:base"},
			{BundleID: "c/.okf", ProfileID: "project:child"},
		},
	}
	before := domain.CloneConfiguration(current)
	renamed, err := renameConfigurationProfile(current, "project:base", "project:renamed", "  Renamed  ")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(current, before) {
		t.Fatal("rename mutated input")
	}
	if renamed.Profiles[0].ProfileID != "project:renamed" || renamed.Profiles[0].Name != "Renamed" || renamed.Profiles[1].Bases[1] != "project:renamed" {
		t.Fatalf("identity or inheritance not rewritten: %+v", renamed.Profiles)
	}
	for index, binding := range renamed.Bindings {
		want := "project:renamed"
		if index == 2 {
			want = "project:child"
		}
		if binding.ProfileID != want {
			t.Fatalf("binding %d: %+v", index, binding)
		}
	}
	renamed.Profiles[0].Layout.Options["custom"] = "changed"
	if !reflect.DeepEqual(current, before) {
		t.Fatal("result aliases input")
	}
	for _, scenario := range []struct{ oldID, newID, code string }{
		{"project:base", "project:child", "okf_profile_id_conflict"},
		{"project:missing", "project:new", "okf_profile_not_found"},
	} {
		if _, err := renameConfigurationProfile(current, scenario.oldID, scenario.newID, ""); !hasApplicationCode(err, scenario.code) {
			t.Fatalf("rejected edit: %v", err)
		}
		if !reflect.DeepEqual(current, before) {
			t.Fatal("rejected rename mutated input")
		}
	}
}
