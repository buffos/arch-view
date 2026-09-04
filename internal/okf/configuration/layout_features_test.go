package configuration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestFeaturePersistenceUpgradesV2WithoutChangingArchitecture(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, configFileName)
	original := []byte(`{"schema_version":"arch-view.config/v1","layout":{"algorithm":"layered","options":{"elk.direction":"DOWN"}},"vendor":{"keep":true}}`)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	store := NewStore()
	loaded, err := store.Load(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	loaded.Profiles = []domain.Profile{{ProfileID: "project:base", Name: "Base", Layout: domain.LayoutSettings{Features: []string{"ports"}}}, {ProfileID: "project:clear", Name: "Clear", Bases: []string{"project:base"}, Layout: domain.LayoutSettings{Features: []string{}}}}
	saved, err := store.Save(context.Background(), root, loaded, loaded.Revision, "features", []byte("features"))
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if string(raw["schema_version"]) != `"arch-view.config/v2"` || string(raw["analysis"]) != "{}" || len(raw["vendor"]) == 0 {
		t.Fatal(string(data))
	}
	var before map[string]json.RawMessage
	_ = json.Unmarshal(original, &before)
	var oldLayout, newLayout any
	_ = json.Unmarshal(before["layout"], &oldLayout)
	_ = json.Unmarshal(raw["layout"], &newLayout)
	oldJSON, _ := json.Marshal(oldLayout)
	newJSON, _ := json.Marshal(newLayout)
	if string(oldJSON) != string(newJSON) {
		t.Fatal("OKF save changed architecture layout")
	}
	reloaded, err := store.Load(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Profiles[0].Layout.Features) != 1 || reloaded.Profiles[1].Layout.Features == nil {
		t.Fatal("features round trip lost intent")
	}
	repeated, err := store.Save(context.Background(), root, loaded, loaded.Revision, "features", []byte("features"))
	if err != nil || repeated.Revision != saved.Revision {
		t.Fatal("idempotency lost", err)
	}
}
