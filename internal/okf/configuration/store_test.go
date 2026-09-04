package configuration

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestStorePreservesUnrelatedAndUnknownConfigurationFields(t *testing.T) {
	root := t.TempDir()
	pathValue := filepath.Join(root, ".archview.json")
	original := []byte(`{
  "schema_version": "arch-view.config/v1",
  "layout": {"algorithm": "layered", "options": {"spacing": 24}},
  "analysis": {"language": "go", "custom": true},
  "unknown": {"owner": "team"},
  "okf": {
    "schema_version": "arch-view.okf/v1",
    "custom_okf": {"retained": true},
    "default_graph": "first/.okf",
    "bindings": [],
    "profiles": [{"profile_id": "project:demo", "name": "Demo", "bases": ["builtin:neutral"], "rules": []}]
  }
}`)
	if err := os.WriteFile(pathValue, original, 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewStore()
	value, err := store.Load(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	value.Profiles[0].Name = "Updated"
	saved, err := store.Save(context.Background(), root, value, value.Revision, "update-demo", []byte(`{"name":"Updated"}`))
	if err != nil {
		t.Fatal(err)
	}
	if saved.Revision == value.Revision || saved.Revision == "" {
		t.Fatalf("revisions = %q -> %q", value.Revision, saved.Revision)
	}

	data, err := os.ReadFile(pathValue)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"layout", "analysis", "unknown"} {
		if string(document[key]) == "" {
			t.Fatalf("missing preserved field %q", key)
		}
	}
	var section map[string]json.RawMessage
	if err := json.Unmarshal(document["okf"], &section); err != nil {
		t.Fatal(err)
	}
	var custom map[string]bool
	if err := json.Unmarshal(section["custom_okf"], &custom); err != nil || !custom["retained"] {
		t.Fatalf("unknown OKF field was not preserved: %s err=%v", section["custom_okf"], err)
	}
	var profiles []domain.Profile
	if err := json.Unmarshal(section["profiles"], &profiles); err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 || profiles[0].Name != "Updated" {
		t.Fatalf("saved profiles = %#v", profiles)
	}
}

func TestStoreRejectsStaleRevisionAndReplaysIdempotentWrites(t *testing.T) {
	root := t.TempDir()
	store := NewStore()
	initial, err := store.Load(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.Save(context.Background(), root, domain.ProjectConfiguration{Profiles: []domain.Profile{{ProfileID: "project:one", Name: "One"}}}, initial.Revision, "op-1", []byte(`one`))
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(root, ".archview.json"))
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := store.Save(context.Background(), root, domain.ProjectConfiguration{Profiles: []domain.Profile{{ProfileID: "project:one", Name: "Changed"}}}, initial.Revision, "op-1", []byte(`one`))
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Revision != first.Revision || replayed.Profiles[0].Name != "One" {
		t.Fatalf("replayed = %#v", replayed)
	}
	if _, err := store.Save(context.Background(), root, domain.ProjectConfiguration{}, initial.Revision, "op-1", []byte(`different`)); !hasCode(err, "okf_idempotency_conflict") {
		t.Fatalf("different idempotent input error = %v", err)
	}
	if _, err := store.Save(context.Background(), root, domain.ProjectConfiguration{}, "sha256:stale", "op-2", []byte(`two`)); !hasCode(err, "okf_revision_conflict") {
		t.Fatalf("stale revision error = %v", err)
	}
	after, err := os.ReadFile(filepath.Join(root, ".archview.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("stale write changed the configuration")
	}
}

func hasCode(err error, code string) bool {
	var value *domain.Error
	return errors.As(err, &value) && value.Code == code
}

func TestStoreRejectsNullDocumentWithoutMutatingIt(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".archview.json")
	if err := os.WriteFile(path, []byte("null"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewStore()
	if _, err := store.Load(context.Background(), root); err == nil {
		t.Fatal("null document accepted")
	}
	if _, err := store.Save(context.Background(), root, domain.ProjectConfiguration{}, "", "null", nil); err == nil {
		t.Fatal("null document overwritten")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "null" {
		t.Fatalf("null document changed: %s, %v", data, err)
	}
}
