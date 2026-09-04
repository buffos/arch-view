package application

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/profile"
)

func TestProfileInheritanceRenameAndDeleteAreAtomic(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeApplicationFile(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: concept\n---\n")
	service := New(root)
	create := func(id string, bases ...string) domain.ProjectConfiguration {
		t.Helper()
		value, err := service.SaveProfileAs(ctx, domain.Profile{ProfileID: id, Name: id, Bases: bases}, "", id, "", id, []byte(id))
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	create("base")
	create("child", "project:base")
	saved, err := service.RenameProfile(ctx, "project:base", "renamed", "Renamed", "", "rename", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range saved.Profiles {
		if item.ProfileID == "project:child" && (len(item.Bases) != 1 || item.Bases[0] != "project:renamed") {
			t.Fatalf("stale inheritance: %#v", item)
		}
	}
	configPath := filepath.Join(root, ".archview.json")
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ replacement, code string }{
		{"", "okf_delete_binding_required"},
		{"project:child", "okf_profile_invalid"},
	} {
		_, err := service.DeleteProfile(ctx, "project:renamed", tc.replacement, false, saved.Revision, "reject-"+tc.replacement, nil)
		if !hasApplicationCode(err, tc.code) {
			t.Fatalf("replacement %q: %v", tc.replacement, err)
		}
		after, err := os.ReadFile(configPath)
		if err != nil || string(before) != string(after) {
			t.Fatalf("rejected delete mutated disk: %v", err)
		}
	}
	deleted, err := service.DeleteProfile(ctx, "project:renamed", "", true, saved.Revision, "delete", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(deleted.Profiles) != 1 || deleted.Profiles[0].Bases[0] != profile.DefaultProfileID {
		t.Fatalf("invalid fallback: %#v", deleted)
	}
}

func TestProfileRetryIdentityIncludesCommandAndTarget(t *testing.T) {
	service := New(t.TempDir())
	ctx := context.Background()
	value := domain.Profile{ProfileID: "draft", Name: "Draft"}
	saved, err := service.SaveProfileAs(ctx, value, "", "first", "", "same-operation", nil)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := service.SaveProfileAs(ctx, value, "", "first", "", "same-operation", nil)
	if err != nil || replay.Revision != saved.Revision {
		t.Fatalf("retry failed: %v", err)
	}
	_, err = service.SaveProfileAs(ctx, value, "", "second", "", "same-operation", nil)
	if !hasApplicationCode(err, "okf_idempotency_conflict") {
		t.Fatalf("cross-target retry: %v", err)
	}
	_, err = service.DeleteProfile(ctx, "project:first", "", false, "", "same-operation", nil)
	if !hasApplicationCode(err, "okf_idempotency_conflict") {
		t.Fatalf("cross-command retry: %v", err)
	}
}
