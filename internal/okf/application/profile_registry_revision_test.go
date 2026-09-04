package application

import (
	"context"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/profile"
)

func TestProfileCatalogRevisionTracksRegistrationsWithoutMetadataCallbacks(t *testing.T) {
	ctx := context.Background()
	registry := profile.NewRegistry()
	service := NewWithDependencies(t.TempDir(), nil, nil, nil, registry)
	revision := func() string {
		t.Helper()
		catalog, err := service.Profiles(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return catalog.RegistryRevision
	}
	initial := revision()
	if initial == "" || revision() != initial {
		t.Fatal("unstable registry revision")
	}
	// Description and schema both panic; neither is needed to list profiles.
	if err := registry.Register(brokenCatalogRule{}); err != nil {
		t.Fatal(err)
	}
	withRule := revision()
	if withRule == initial {
		t.Fatal("rule registration absent from revision")
	}
	shape := domain.ShapeDefinition{ID: "test.revision", Version: "1", Description: "Revision rectangle", Geometry: "rectangle", Content: domain.ShapeBox{Width: 1, Height: 1}}
	if err := registry.RegisterShape(catalogShape{shape}); err != nil {
		t.Fatal(err)
	}
	withShape := revision()
	if withShape == withRule || revision() != withShape {
		t.Fatal("shape registration absent or unstable")
	}
	other := profile.NewRegistry()
	if err := other.RegisterShape(catalogShape{shape}); err != nil {
		t.Fatal(err)
	}
	if err := other.Register(brokenCatalogRule{}); err != nil {
		t.Fatal(err)
	}
	otherRevision, err := other.Revision()
	if err != nil || otherRevision != withShape {
		t.Fatalf("registration order changed revision: %v", err)
	}
}
