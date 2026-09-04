//go:build windows

package application

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestUnreadableSourceRecoversWithoutAffectingIndependentBundle(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	path := filepath.Join(root, ".okf", "root.md")
	const original = "---\ntype: topic\ntitle: Original source\n---\nUnchanged body.\n"
	writeApplicationFile(t, path, original)
	writeApplicationFile(t, filepath.Join(root, "other", ".okf", "other.md"), "---\ntype: topic\ntitle: Independent\n---\n")
	service := New(root)
	before, err := service.SelectBundle(ctx, "", ".okf")
	if err != nil || before.Projection == nil {
		t.Fatalf("initial selection: %+v %v", before, err)
	}
	widePath, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	// Deny all sharing so the real scanner cannot open this source for reading.
	handle, err := syscall.CreateFile(widePath, syscall.GENERIC_READ, 0, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if handle != syscall.InvalidHandle {
			_ = syscall.CloseHandle(handle)
		}
	}()
	catalog, err := service.Refresh(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, candidate := range catalog.Bundles {
		if candidate.BundleID != ".okf" {
			continue
		}
		if candidate.Selectable || candidate.Status != domain.BundleUnreadable {
			t.Fatalf("locked source still selectable: %+v", candidate)
		}
		for _, diagnostic := range candidate.Diagnostics {
			found = found || diagnostic.Code == "okf_bundle_unreadable"
		}
	}
	if !found {
		t.Fatal("missing unreadable-source diagnostic")
	}
	if _, err := service.Session(ctx, ""); !hasApplicationCode(err, "okf_bundle_not_found") {
		t.Fatalf("session reused inaccessible source: %v", err)
	}
	other, err := service.SelectBundle(ctx, "", "other/.okf")
	if err != nil || other.Projection == nil || len(other.Projection.Nodes) != 1 || other.Projection.Nodes[0].ConceptID != "other" {
		t.Fatalf("independent bundle unavailable: %+v %v", other, err)
	}
	if err := syscall.CloseHandle(handle); err != nil {
		t.Fatal(err)
	}
	handle = syscall.InvalidHandle
	if _, err := service.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	recovered, err := service.SelectBundle(ctx, "", ".okf")
	if err != nil || recovered.Projection == nil || len(recovered.Projection.Nodes) != 1 || recovered.Projection.Nodes[0].Label != "Original source" {
		t.Fatalf("source did not recover: %+v %v", recovered, err)
	}
	if recovered.Projection.Source != before.Projection.Source {
		t.Fatal("recovered source identity or revision changed")
	}
	bytes, err := os.ReadFile(path)
	if err != nil || string(bytes) != original {
		t.Fatalf("source changed during recovery: %v", err)
	}
}
