package application

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestObstructedConfigurationPathPreservesSessionAndAllowsRetry(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeApplicationFile(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: concept\ntitle: Root\n---\n")
	service := New(root)
	saved, err := service.SaveProfileAs(ctx, domain.Profile{Name: "Original"}, "", "working", "", "create", nil)
	if err != nil {
		t.Fatal(err)
	}
	before, err := service.SelectProfile(ctx, "", "project:working")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".archview.json")
	backup := filepath.Join(root, "saved-config.json")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// A directory at the document path makes the real filesystem read fail.
	// Preserve the original file so recovery does not depend on test recreation.
	if err := os.Rename(path, backup); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	updated := saved.Profiles[0]
	updated.Name = "Retried edit"
	if _, err := service.SaveProfile(ctx, updated, saved.Revision, "retry-write", nil); err == nil {
		t.Fatal("obstructed configuration save succeeded")
	}
	current, err := service.Session(ctx, "")
	if err != nil || !reflect.DeepEqual(before, current) {
		t.Fatalf("failed save changed session: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		t.Fatalf("save replaced filesystem obstruction: %v", err)
	}
	preserved, err := os.ReadFile(backup)
	if err != nil || string(preserved) != string(original) {
		t.Fatalf("original configuration changed: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(backup, path); err != nil {
		t.Fatal(err)
	}
	retried, err := service.SaveProfile(ctx, updated, saved.Revision, "retry-write", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(retried.Profiles) != 1 || retried.Profiles[0].Name != updated.Name || retried.Revision == saved.Revision {
		t.Fatalf("failed operation poisoned retry: %+v", retried)
	}
}
