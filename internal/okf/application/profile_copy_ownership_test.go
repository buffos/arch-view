package application

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestProfileSaveAsPreservesSourceAndCallerOwnership(t *testing.T) {
	ctx := context.Background()
	service := New(t.TempDir())
	var declaration domain.Profile
	if err := json.Unmarshal([]byte(`{"profile_id":"draft","name":"Original","bases":["builtin:neutral"],"custom":{"nested":[1,2,3]}}`), &declaration); err != nil {
		t.Fatal(err)
	}
	before, err := json.Marshal(declaration)
	if err != nil {
		t.Fatal(err)
	}
	created, err := service.SaveProfileAs(ctx, declaration, "", "source", "", "create-source", nil)
	if err != nil {
		t.Fatal(err)
	}
	after, err := json.Marshal(declaration)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("Save As changed caller declaration: %s, %v", after, err)
	}
	original := domain.CloneProfile(created.Profiles[0])
	copied, err := service.SaveProfileAs(ctx, domain.Profile{}, "project:source", "copy", created.Revision, "copy-source", nil)
	if err != nil {
		t.Fatal(err)
	}
	var copy domain.Profile
	for _, candidate := range copied.Profiles {
		if candidate.ProfileID == "project:copy" {
			copy = candidate
		}
	}
	if copy.ProfileID == "" {
		t.Fatal("Save As did not return the copied profile")
	}
	copy.Name = "Updated copy"
	copyBefore, err := json.Marshal(copy)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SaveProfile(ctx, copy, copied.Revision, "save-copy", nil); err != nil {
		t.Fatal(err)
	}
	copyAfter, err := json.Marshal(copy)
	if err != nil || !bytes.Equal(copyBefore, copyAfter) {
		t.Fatalf("Save changed caller declaration: %s, %v", copyAfter, err)
	}
	catalog, err := service.Profiles(ctx)
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, candidate := range catalog.Profiles {
		if candidate.ProfileID != "project:source" && candidate.ProfileID != "project:copy" {
			continue
		}
		seen++
		encoded, err := json.Marshal(candidate)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(encoded, &fields); err != nil || string(fields["custom"]) != `{"nested":[1,2,3]}` {
			t.Fatalf("extension lost: %s, %v", encoded, err)
		}
		if candidate.ProfileID == "project:source" {
			want, err := json.Marshal(original)
			if err != nil || !bytes.Equal(want, encoded) {
				t.Fatalf("editing copy changed source: %s, %v", encoded, err)
			}
		} else if candidate.Name != "Updated copy" {
			t.Fatalf("copy update missing: %#v", candidate)
		}
	}
	if seen != 2 {
		t.Fatalf("expected source and copy, found %d", seen)
	}
}
