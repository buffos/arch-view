package application

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/profile"
)

// SC-019 stateful cross: persisted configuration changes while evaluation is
// paused. Neither its stale snapshot nor its pending focus may be published.
func TestConfigurationMutationSupersedesInFlightProjection(t *testing.T) {
	for _, action := range []string{"save", "delete"} {
		t.Run(action, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			root := t.TempDir()
			writeApplicationFile(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: concept\ntitle: Root\n---\n")
			registry := profile.NewRegistry()
			rule := &pausedNavigationRule{entered: make(chan struct{}), release: make(chan struct{})}
			if err := registry.Register(rule); err != nil {
				t.Fatal(err)
			}
			service := NewWithDependencies(root, nil, nil, nil, registry)
			declaration := domain.Profile{Name: "Working", Rules: []domain.RuleInvocation{{RuleID: rule.ID(), Version: "1", Enabled: true}}}
			saved, err := service.SaveProfileAs(ctx, declaration, "", "working", "", "create", nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := service.SelectProfile(ctx, "", "project:working"); err != nil {
				t.Fatal(err)
			}
			rule.armed.Store(true)
			done := make(chan error, 1)
			go func() { _, err := service.Focus(ctx, "", "root"); done <- err }()
			select {
			case <-rule.entered:
			case <-ctx.Done():
				t.Fatal("evaluation did not pause")
			}
			wantProfile, wantLabel := "project:working", "Saved label"
			if action == "save" {
				updated := saved.Profiles[0]
				updated.Rules = []domain.RuleInvocation{{RuleID: "okf.rule.label_template", Version: "1", Enabled: true, Parameters: map[string]any{"template": wantLabel}}}
				_, err = service.SaveProfile(ctx, updated, saved.Revision, "save", nil)
			} else {
				wantProfile, wantLabel = profile.DefaultProfileID, "Root"
				_, err = service.DeleteProfile(ctx, "project:working", "", true, saved.Revision, "delete", nil)
			}
			if err != nil {
				t.Fatal(err)
			}
			current, err := service.Session(ctx, "")
			if err != nil || current.Projection == nil || len(current.Projection.Nodes) != 1 {
				t.Fatalf("current projection unavailable: %+v %v", current, err)
			}
			close(rule.release)
			select {
			case err := <-done:
				if !hasApplicationCode(err, "okf_operation_superseded") {
					t.Fatalf("old request result: %v", err)
				}
			case <-ctx.Done():
				t.Fatal("old request did not finish")
			}
			after, err := service.Session(ctx, "")
			if err != nil || after.Projection == nil || len(after.Projection.Nodes) != 1 {
				t.Fatalf("session lost: %+v %v", after, err)
			}
			if after.ProfileID != wantProfile || after.Projection.Nodes[0].Label != wantLabel || after.Navigation.FocusRoot != "" || after.Navigation.CanGoBack {
				t.Fatalf("stale work changed current profile/focus: %+v", after)
			}
			if current.Projection.ProjectionRevision != after.Projection.ProjectionRevision {
				t.Fatal("old result replaced current snapshot")
			}
		})
	}
}
