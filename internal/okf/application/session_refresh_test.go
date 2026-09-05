package application

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/buffo/arch-view/internal/okf/profile"
)

func TestSessionRebuildsAfterSourceRefreshAndProfileSave(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeApplicationFile(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: area\ntitle: Before\n---\n")
	service := New(root)
	before, err := service.Session(ctx, DefaultSessionID)
	if err != nil {
		t.Fatal(err)
	}
	writeApplicationFile(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: area\ntitle: After\n---\n")
	if _, err := service.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	after, err := service.Session(ctx, DefaultSessionID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Projection.Source.SourceRevision == before.Projection.Source.SourceRevision || after.Projection.Nodes[0].Label != "After" {
		t.Fatalf("refresh returned stale projection: %#v", after.Projection)
	}
	neutral, _ := profile.NewRegistry().Profile(profile.DefaultProfileID)
	saved, err := service.SaveProfileAs(ctx, neutral, "", "review", "", "create", []byte("create"))
	if err != nil {
		t.Fatal(err)
	}
	before, err = service.SelectProfile(ctx, DefaultSessionID, "project:review")
	if err != nil {
		t.Fatal(err)
	}
	edited := saved.Profiles[0]
	if edited.Layout.Options == nil {
		edited.Layout.Options = map[string]any{}
	}
	edited.Layout.Options["org.eclipse.elk.direction"] = "DOWN"
	if _, err := service.SaveProfile(ctx, edited, saved.Revision, "update", []byte("update")); err != nil {
		t.Fatal(err)
	}
	after, err = service.Session(ctx, DefaultSessionID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Projection.Profile.Layout.Algorithm != "layered" || after.Projection.Profile.Layout.Options["org.eclipse.elk.direction"] != "DOWN" || after.Projection.Profile.ProfileRevision == before.Projection.Profile.ProfileRevision {
		t.Fatalf("profile save returned stale projection: %#v", after.Projection.Profile)
	}
}
