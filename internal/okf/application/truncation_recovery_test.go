package application

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/profile"
)

func TestTruncatedSessionRecoversHiddenSubtreeAndBack(t *testing.T) {
	root := t.TempDir()
	for _, id := range []string{"root", "root/a", "root/z", "root/z/leaf"} {
		writeApplicationFile(t, filepath.Join(root, ".okf", filepath.FromSlash(id)+".md"), "---\ntype: concept\n---\nContent")
	}
	ctx := context.Background()
	service := New(root)
	if _, err := service.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	declaration := domain.Profile{ProfileID: "project:limited", Bases: []string{profile.DefaultProfileID}, Navigation: domain.NavigationSettings{MaxNodes: 2}}
	if _, err := service.SaveProfileAs(ctx, declaration, "", "limited", "", "create-limited", []byte("limited")); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SelectProfile(ctx, "", "project:limited"); err != nil {
		t.Fatal(err)
	}
	initial, err := service.SetNavigation(ctx, "", 2, true)
	if err != nil {
		t.Fatal(err)
	}
	if initial.Projection.Status != domain.ProjectionTruncated || initial.Projection.Counts.HiddenNodes != 2 {
		t.Fatalf("initial projection=%+v", initial.Projection)
	}
	for _, node := range initial.Projection.Nodes {
		if node.ConceptID == "root/z" {
			t.Fatal("recovery target must actually be hidden in initial projection")
		}
	}
	focused, err := service.Focus(ctx, "", "root/z")
	if err != nil {
		t.Fatal(err)
	}
	if focused.ProfileID != initial.ProfileID || focused.BundleID != initial.BundleID || !focused.Navigation.Full || focused.Projection.Status != domain.ProjectionReady {
		t.Fatalf("focused session=%+v", focused)
	}
	var ids []string
	for _, node := range focused.Projection.Nodes {
		ids = append(ids, node.ConceptID)
	}
	if !reflect.DeepEqual(ids, []string{"root/z", "root/z/leaf"}) {
		t.Fatalf("focused nodes=%v", ids)
	}
	back, err := service.Back(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if back.Projection.ProjectionRevision != initial.Projection.ProjectionRevision || !reflect.DeepEqual(back.Projection.Nodes, initial.Projection.Nodes) {
		t.Fatal("Back did not restore the prior truncated projection")
	}
}
