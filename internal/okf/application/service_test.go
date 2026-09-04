package application

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/profile"
)

func TestServiceIsolatesInvalidBundlesAndSupportsSessionNavigation(t *testing.T) {
	root := t.TempDir()
	writeApplicationFile(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: area\ntitle: Root\n---\n")
	writeApplicationFile(t, filepath.Join(root, ".okf", "root", "child.md"), "---\ntype: concept\ntitle: Child\nparent: root\nstate: specified\n---\nSee [root](/root.md).\n")
	writeApplicationFile(t, filepath.Join(root, "broken", ".okf", "invalid.md"), "not frontmatter\n")

	service := New(root)
	catalog, err := service.Refresh(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Bundles) != 2 || catalog.Bundles[0].BundleID != ".okf" || catalog.Bundles[1].Status != domain.BundleInvalid {
		t.Fatalf("catalog = %#v", catalog)
	}
	if catalog.DefaultBundleID != ".okf" {
		t.Fatalf("default bundle = %q", catalog.DefaultBundleID)
	}
	summary, err := service.Summary(context.Background(), ".okf")
	if err != nil || summary.ConceptCount != 2 || summary.LinkCount != 1 {
		t.Fatalf("summary = %#v err=%v", summary, err)
	}
	if _, err := service.SelectBundle(context.Background(), "default", "broken/.okf"); !hasApplicationCode(err, "okf_bundle_not_found") {
		t.Fatalf("invalid bundle selection error = %v", err)
	}

	initial, err := service.Session(context.Background(), DefaultSessionID)
	if err != nil || initial.Projection == nil || len(initial.Projection.Nodes) != 2 {
		t.Fatalf("initial session = %#v err=%v", initial, err)
	}
	focused, err := service.Focus(context.Background(), DefaultSessionID, "root/child")
	if err != nil || focused.Projection == nil || focused.Projection.Navigation.FocusRoot != "root/child" {
		t.Fatalf("focused session = %#v err=%v", focused, err)
	}
	back, err := service.Back(context.Background(), DefaultSessionID)
	if err != nil || back.Projection == nil || back.Projection.Navigation.FocusRoot != "" {
		t.Fatalf("back session = %#v err=%v", back, err)
	}
	if _, err := service.SetNavigation(context.Background(), DefaultSessionID, 0, false); !hasApplicationCode(err, "okf_depth_invalid") {
		t.Fatalf("invalid depth error = %v", err)
	}
}

func TestServicePersistsProfileLifecycleWithBindingSafetyAndIdempotency(t *testing.T) {
	root := t.TempDir()
	writeApplicationFile(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: area\n---\n")
	writeApplicationFile(t, filepath.Join(root, "second", ".okf", "root.md"), "---\ntype: area\n---\n")
	service := New(root)
	if _, err := service.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	neutral, exists := profile.NewRegistry().Profile(profile.DefaultProfileID)
	if !exists {
		t.Fatal("neutral profile missing")
	}
	neutral.Bases = []string{profile.DefaultProfileID}
	neutral.Name = "Working copy"
	saved, err := service.SaveProfileAs(context.Background(), neutral, "", "working", "", "profile-create", []byte("create-working"))
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := service.SaveProfileAs(context.Background(), neutral, "", "working", "", "profile-create", []byte("create-working"))
	if err != nil || replayed.Revision != saved.Revision || len(replayed.Profiles) != 1 {
		t.Fatalf("replayed Save As = %#v err=%v", replayed, err)
	}
	if _, err := service.SaveProfileAs(context.Background(), neutral, "", "working", "", "profile-create", []byte("different")); !hasApplicationCode(err, "okf_idempotency_conflict") {
		t.Fatalf("idempotency conflict = %v", err)
	}
	if _, err := service.SaveProfile(context.Background(), domain.Profile{ProfileID: "new-profile", Name: "New"}, "", "save-new", []byte("save-new")); !hasApplicationCode(err, "okf_profile_not_found") {
		t.Fatalf("Save should not create a profile = %v", err)
	}
	bound, err := service.Bind(context.Background(), ".okf", "project:working", saved.Revision, "bind-working", []byte("bind-working"))
	if err != nil {
		t.Fatal(err)
	}
	if bound.DefaultGraph != ".okf" || len(bound.Bindings) != 1 {
		t.Fatalf("binding = %#v", bound)
	}
	bound, err = service.Bind(context.Background(), "second/.okf", "project:working", bound.Revision, "bind-second", nil)
	if err != nil || len(bound.Bindings) != 2 {
		t.Fatalf("second binding = %#v err=%v", bound, err)
	}
	if _, err := service.SelectProfile(context.Background(), DefaultSessionID, "project:working"); err != nil {
		t.Fatal(err)
	}
	rename, err := service.RenameProfile(context.Background(), "project:working", "renamed", "Renamed", bound.Revision, "rename-working", []byte("rename-working"))
	if err != nil {
		t.Fatal(err)
	}
	if rename.Profiles[0].ProfileID != "project:renamed" || rename.Profiles[0].Name != "Renamed" || rename.Bindings[0].ProfileID != "project:renamed" {
		t.Fatalf("rename = %#v", rename)
	}
	if len(rename.Bindings) != 2 || rename.Bindings[1].ProfileID != "project:renamed" {
		t.Fatalf("rename left a stale binding: %#v", rename.Bindings)
	}
	session, err := service.Session(context.Background(), DefaultSessionID)
	if err != nil || session.ProfileID != "project:renamed" {
		t.Fatalf("renamed session = %#v err=%v", session, err)
	}
	if _, err := service.DeleteProfile(context.Background(), "project:renamed", "", false, rename.Revision, "delete-rejected", []byte("delete-rejected")); !hasApplicationCode(err, "okf_delete_binding_required") {
		t.Fatalf("delete without fallback error = %v", err)
	}
	deleted, err := service.DeleteProfile(context.Background(), "project:renamed", "", true, rename.Revision, "delete-renamed", []byte("delete-renamed"))
	if err != nil {
		t.Fatal(err)
	}
	if len(deleted.Profiles) != 0 || len(deleted.Bindings) != 2 || deleted.Bindings[0].ProfileID != profile.DefaultProfileID || deleted.Bindings[1].ProfileID != profile.DefaultProfileID {
		t.Fatalf("deleted configuration = %#v", deleted)
	}
	loaded, err := os.ReadFile(filepath.Join(root, ".archview.json"))
	if err != nil || len(loaded) == 0 {
		t.Fatalf("persisted configuration = %q err=%v", loaded, err)
	}
}

func writeApplicationFile(t *testing.T, pathValue, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(pathValue), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pathValue, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func hasApplicationCode(err error, code string) bool {
	var value *domain.Error
	return errors.As(err, &value) && value.Code == code
}
