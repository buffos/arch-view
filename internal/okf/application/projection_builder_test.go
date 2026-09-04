package application

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/profile"
)

func TestProjectionBuilderResolvesFallbackWithoutSessionState(t *testing.T) {
	builder := projectionBuilder{profiles: profile.NewRegistry()}
	index := domain.BundleIndex{
		BundleID: "selected", SourceRevision: "r1", ConceptOrder: []string{"root"},
		Documents: map[string]domain.ConceptDocument{"root": {ConceptID: "root", Type: "topic", Title: "Root"}},
	}
	navigation := domain.NavigationState{Depth: 2, Full: true, FocusRoot: "root", CanGoBack: true}
	snapshot, selected, err := builder.build(context.Background(), index, "project:missing", navigation)
	if err != nil {
		t.Fatal(err)
	}
	if selected != profile.DefaultProfileID || snapshot.Profile.ProfileID != selected || len(snapshot.Nodes) != 1 {
		t.Fatalf("unexpected fallback: selected=%s snapshot=%+v", selected, snapshot)
	}
	if !snapshot.Navigation.Full || !snapshot.Navigation.CanGoBack || len(snapshot.Navigation.Breadcrumbs) != 1 {
		t.Fatalf("navigation not preserved: %+v", snapshot.Navigation)
	}
	if len(navigation.Breadcrumbs) != 0 {
		t.Fatal("builder mutated input navigation")
	}
	for _, diagnostic := range snapshot.Diagnostics {
		if diagnostic.Code == "okf_profile_not_found" {
			return
		}
	}
	t.Fatal("fallback diagnostic missing")
}

func TestProjectionBuilderReturnsNoSnapshotOnFailure(t *testing.T) {
	builder := projectionBuilder{profiles: profile.NewRegistry()}
	index := domain.BundleIndex{BundleID: "selected", ConceptOrder: []string{"root"}, Documents: map[string]domain.ConceptDocument{"root": {ConceptID: "root", Type: "topic"}}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, scenario := range []struct {
		name  string
		ctx   context.Context
		focus string
	}{
		{"cancelled", ctx, "root"},
		{"foreign focus", context.Background(), "foreign"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			snapshot, _, err := builder.build(scenario.ctx, index, profile.DefaultProfileID, domain.NavigationState{Depth: 2, FocusRoot: scenario.focus})
			if scenario.name == "cancelled" && !errors.Is(err, context.Canceled) {
				t.Fatalf("cancellation lost: %v", err)
			}
			if scenario.name == "foreign focus" && !hasApplicationCode(err, "okf_concept_not_found") {
				t.Fatalf("focus error lost: %v", err)
			}
			if !reflect.DeepEqual(snapshot, domain.ProjectionSnapshot{}) {
				t.Fatal("failed build returned a publishable snapshot")
			}
		})
	}
}
