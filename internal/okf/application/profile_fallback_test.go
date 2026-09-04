package application

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/profile"
)

func TestInvalidDiskProfileRemainsRepairableWithNeutralProjection(t *testing.T) {
	root := t.TempDir()
	writeApplicationFile(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: concept\ntitle: Root\n---\n")
	configPath := filepath.Join(root, ".archview.json")
	config := `{"okf":{"schema_version":"arch-view.okf/v1","profiles":[{"profile_id":"project:broken","rules":[{"rule_id":"okf.rule.label_template","enabled":true,"parameters":{"template":false}}]}]}}`
	writeApplicationFile(t, configPath, config)
	service := New(root)
	ctx := context.Background()
	catalog, err := service.Profiles(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range catalog.Profiles {
		if item.ProfileID == "project:broken" {
			found = true
			if item.Status != domain.ProfileInvalid {
				t.Fatalf("invalid profile marked %s", item.Status)
			}
		}
	}
	if !found {
		t.Fatal("invalid declaration lost from repair catalog")
	}
	view, err := service.SelectProfile(ctx, "", "project:broken")
	if err != nil {
		t.Fatal(err)
	}
	if view.ProfileID != profile.DefaultProfileID || view.Projection.Profile.ProfileID != profile.DefaultProfileID {
		t.Fatalf("invalid profile not replaced by neutral: %#v", view)
	}
	diagnosed := false
	for _, diagnostic := range view.Projection.Diagnostics {
		if diagnostic.Code == "okf_rule_invalid" && diagnostic.ProfileID == "project:broken" {
			diagnosed = true
		}
	}
	if !diagnosed {
		t.Fatal("neutral fallback lost invalid-profile diagnostic")
	}
	after, err := os.ReadFile(configPath)
	if err != nil || string(after) != config {
		t.Fatalf("fallback modified persisted declaration: %v", err)
	}
}
