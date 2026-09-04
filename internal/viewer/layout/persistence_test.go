package layout_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	analysisconfig "github.com/buffo/arch-view/internal/analysis/config"
	"github.com/buffo/arch-view/internal/okf/configuration"
	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/viewer/layout"
)

func TestLayoutSavePreservesLaterOKFEditsAndAnalysisCanReload(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".archview.json")
	initial := []byte(`{"schema_version":"arch-view.config/v1","layout":{"algorithm":"layered","options":{}},"extension":{"keep":true}}`)
	if err := os.WriteFile(path, initial, 0o600); err != nil {
		t.Fatal(err)
	}
	session := layout.NewSession(root)
	if session.Response().Status != "valid" {
		t.Fatal(session.Response())
	}
	store := configuration.NewStore()
	config, err := store.Load(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	config.Profiles = []domain.Profile{{ProfileID: "project:review", Name: "New profile"}}
	if _, err := store.Save(context.Background(), root, config, config.Revision, "profile-save", []byte("new-profile")); err != nil {
		t.Fatal(err)
	}
	if err := session.SaveActive(layout.LayoutProfile{Algorithm: "mrtree", Options: map[string]any{}}); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(context.Background(), root)
	if err != nil || len(loaded.Profiles) != 1 || loaded.Profiles[0].Name != "New profile" {
		t.Fatalf("layout save lost OKF profile: %#v, %v", loaded, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := analysisconfig.Decode(data, nil)
	if err != nil || parsed.Layout.Algorithm != "mrtree" {
		t.Fatalf("analysis reload: %#v, %v", parsed, err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if len(raw["extension"]) == 0 {
		t.Fatal("extension was lost")
	}
}

func TestLayoutSaveAsPreservesDestinationSections(t *testing.T) {
	session := layout.NewSession(t.TempDir())
	destination := t.TempDir()
	path := filepath.Join(destination, ".archview.json")
	data := []byte(`{"schema_version":"arch-view.config/v2","layout":{"algorithm":"layered","options":{}},"analysis":{},"okf":{"default_graph":"docs/.okf"},"extension":17}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := session.SaveAs(layout.DefaultProfile(), destination, true); err != nil {
		t.Fatal(err)
	}
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(updated, &raw); err != nil {
		t.Fatal(err)
	}
	if string(raw["extension"]) != "17" || len(raw["analysis"]) == 0 || len(raw["okf"]) == 0 {
		t.Fatalf("destination sections were overwritten: %s", updated)
	}
}
