package application

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestProfilePreviewSeparatesDeclarationFromEffectiveDraft(t *testing.T) {
	root := t.TempDir()
	writeApplicationFile(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: concept\n---\n")
	service := New(root)
	ctx := context.Background()
	base := domain.Profile{Navigation: domain.NavigationSettings{DefaultDepth: 7}}
	if _, err := service.SaveProfileAs(ctx, base, "", "base", "", "create-base", nil); err != nil {
		t.Fatal(err)
	}
	before, err := service.Session(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".archview.json")
	disk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.PreviewProfile(ctx, domain.Profile{ProfileID: "project:draft", Bases: []string{"project:base"}})
	if err != nil || !result.Valid {
		t.Fatalf("preview failed: %#v %v", result, err)
	}
	if result.Profile.Navigation.DefaultDepth != 0 || result.EffectiveProfile.Navigation.DefaultDepth != 7 {
		t.Fatalf("declaration/effective conflated: %#v", result)
	}
	after, err := service.Session(ctx, "")
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("preview changed session")
	}
	current, err := os.ReadFile(path)
	if err != nil || string(current) != string(disk) {
		t.Fatal("preview changed configuration")
	}
}
