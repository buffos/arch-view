package application

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestConfigurationRepairRefreshesCatalogWithoutRescanning(t *testing.T) {
	root := t.TempDir()
	writeApplicationFile(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: topic\n---\n")
	writeApplicationFile(t, filepath.Join(root, "second", ".okf", "root.md"), "---\ntype: topic\n---\n")
	writeApplicationFile(t, filepath.Join(root, "invalid", ".okf", "bad.md"), "missing frontmatter\n")
	writeApplicationFile(t, filepath.Join(root, ".archview.json"), `{"okf":{"schema_version":"arch-view.okf/v1","bindings":[{"bundle_id":".okf","profile_id":"project:missing"}]}}`)
	service := New(root)
	ctx := context.Background()
	initial, err := service.Refresh(ctx)
	if err != nil {
		t.Fatal(err)
	}
	hasMissing := func(values []domain.Diagnostic) bool {
		for _, diagnostic := range values {
			if diagnostic.Code == "okf_profile_not_found" {
				return true
			}
		}
		return false
	}
	if !hasMissing(initial.Diagnostics) {
		t.Fatal("fixture did not report stale binding")
	}
	saved, err := service.SaveProfileAs(ctx, domain.Profile{Name: "Repaired"}, "", "missing", "", "repair", nil)
	if err != nil {
		t.Fatal(err)
	}
	profiles, err := service.Profiles(ctx)
	if err != nil || hasMissing(profiles.Diagnostics) || hasMissing(service.Catalog().Diagnostics) {
		t.Fatalf("repaired binding still diagnosed: %+v, %v", profiles.Diagnostics, err)
	}
	foundInvalid := false
	for _, candidate := range service.Catalog().Bundles {
		if candidate.BundleID == "invalid/.okf" {
			foundInvalid = !candidate.Selectable && len(candidate.Diagnostics) > 0
		}
	}
	if !foundInvalid {
		t.Fatal("configuration repair discarded unrelated source diagnostics")
	}
	if _, err := service.Bind(ctx, "second/.okf", "project:missing", saved.Revision, "bind-second", nil); err != nil {
		t.Fatal(err)
	}
	if got := service.Catalog().DefaultBundleID; got != "second/.okf" {
		t.Fatalf("saved default bundle not published: %q", got)
	}
}
