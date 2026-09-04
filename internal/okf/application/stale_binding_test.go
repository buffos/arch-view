package application

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/profile"
)

func TestSC014StaleBoundProfilePreservesDeclarationAndProducesFallback(t *testing.T) {
	for _, scenario := range []struct{ name, base, rule, version, code, active string }{
		{"missing-base", "project:absent", "", "", "okf_profile_not_found", profile.DefaultProfileID},
		{"missing-rule", "", "test.absent", "1", "okf_rule_not_found", "project:stale"},
		{"missing-version", "", "okf.rule.visibility", "999", "okf_rule_not_found", "project:stale"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			root := t.TempDir()
			writeApplicationFile(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: concept\ntitle: Root\n---\n")
			declaration := domain.Profile{ProfileID: "project:stale", Name: "Repair me"}
			if scenario.base != "" {
				declaration.Bases = []string{scenario.base}
			}
			if scenario.rule != "" {
				declaration.Rules = []domain.RuleInvocation{{RuleID: scenario.rule, Version: scenario.version, Enabled: true}}
			}
			data, err := json.Marshal(map[string]any{"okf": domain.ProjectConfiguration{SchemaVersion: "arch-view.okf/v1", Profiles: []domain.Profile{declaration}, Bindings: []domain.ProfileBinding{{BundleID: ".okf", ProfileID: declaration.ProfileID}}}})
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, ".archview.json")
			writeApplicationFile(t, path, string(data))
			service := New(root)
			view, err := service.SelectBundle(context.Background(), "", ".okf")
			if err != nil || view.Projection == nil || view.ProfileID != scenario.active || len(view.Projection.Nodes) != 1 || view.Projection.Nodes[0].Label != "Root" {
				t.Fatalf("fallback: %+v %v", view, err)
			}
			found := false
			for _, diagnostic := range view.Projection.Diagnostics {
				if diagnostic.Code == scenario.code {
					found = true
				}
			}
			if !found {
				t.Fatalf("missing stale-layer diagnosis: %+v", view.Projection.Diagnostics)
			}
			catalog, err := service.Profiles(context.Background())
			if err != nil || len(catalog.Bindings) != 1 || catalog.Bindings[0].ProfileID != declaration.ProfileID {
				t.Fatalf("stale binding lost: %+v %v", catalog, err)
			}
			found = false
			for _, candidate := range catalog.Profiles {
				if candidate.ProfileID == declaration.ProfileID {
					found = true
					if candidate.Status == domain.ProfileValid {
						t.Fatal("stale layer marked fully valid")
					}
				}
			}
			if !found {
				t.Fatal("repairable declaration missing")
			}
			after, err := os.ReadFile(path)
			if err != nil || string(after) != string(data) {
				t.Fatalf("fallback rewrote configuration: %v", err)
			}
		})
	}
}
