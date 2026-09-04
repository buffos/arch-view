package application

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestFocusedServicesSharePublicationWithoutUnrelatedOperations(t *testing.T) {
	root := t.TempDir()
	writeApplicationFile(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: topic\ntitle: Original\n---\n")
	service := New(root)
	ctx := context.Background()
	// Exercise components directly, including lazy initialization, rather than
	// relying only on the facade's promoted methods.
	initial, err := service.sessionService.Session(ctx, "")
	if err != nil || initial.Projection == nil {
		t.Fatalf("initial session: %+v %v", initial, err)
	}
	_, err = service.profileService.SaveProfileAs(ctx, domain.Profile{Name: "Independent profile"}, "", "independent", "", "create", nil)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := service.sessionService.SelectProfile(ctx, "", "project:independent")
	if err != nil || selected.ProfileID != "project:independent" {
		t.Fatalf("profile publication: %+v %v", selected, err)
	}
	detail, err := service.inspectionService.Detail(ctx, "", "root")
	if err != nil || detail.Overview["title"] != "Original" {
		t.Fatalf("inspection: %+v %v", detail, err)
	}
	writeApplicationFile(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: topic\ntitle: Refreshed\n---\n")
	if _, err := service.catalogService.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	after, err := service.sessionService.Session(ctx, "")
	if err != nil || after.Projection == nil || after.Projection.Source.SourceRevision == initial.Projection.Source.SourceRevision {
		t.Fatalf("refresh did not invalidate projection: %+v %v", after, err)
	}
	if _, err := service.diagnosticService.Diagnostics(ctx, DiagnosticQuery{}); err != nil {
		t.Fatal(err)
	}
	if _, ok := any(service.sessionService).(ProfileOperations); ok {
		t.Fatal("navigation exposes persistence")
	}
	if _, ok := any(service.profileService).(SessionOperations); ok {
		t.Fatal("profile lifecycle exposes navigation")
	}
	if _, ok := any(service.inspectionService).(ProfileOperations); ok {
		t.Fatal("inspection exposes persistence")
	}
}
