package policy

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/buffo/arch-view/internal/quality"
)

func TestAppendBaselineCreatesCanonicalRevisionAndMergesIdempotently(t *testing.T) {
	root := t.TempDir()
	store, err := NewFileStore(root)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(store, quality.NewDefaultCatalog())
	profile := testPolicyProfile("profile:append", "1.0.0")
	if _, err := service.SaveProfile(context.Background(), profile, "main.json", false); err != nil {
		t.Fatalf("save profile: %v", err)
	}
	entry := quality.BaselineEntry{FindingKey: "finding:one", RuleID: "rule:one", RuleVersion: "1.0.0", ProfileID: profile.ProfileID, ProfileVersion: profile.ProfileVersion, FormulaVersions: []quality.FormulaVersion{}, Reason: "accepted after review"}
	first, err := service.AppendBaseline(context.Background(), BaselineAppendRequest{Profile: profile, ProfileFileName: "main.json", BaselineFileName: "main.json", BaselineID: "baseline:main", Entries: []quality.BaselineEntry{entry}})
	if err != nil {
		t.Fatalf("first append: %v", err)
	}
	if !first.Changed || first.Baseline.Revision != "1.0.0" || len(first.Added) != 1 || first.Profile.Baseline == nil || first.Profile.Baseline.Revision != "1.0.0" {
		t.Fatalf("first append = %#v", first)
	}
	second, err := service.AppendBaseline(context.Background(), BaselineAppendRequest{Profile: first.Profile, ProfileFileName: "main.json", BaselineFileName: "main.json", Entries: []quality.BaselineEntry{entry}})
	if err != nil {
		t.Fatalf("idempotent append: %v", err)
	}
	if second.Changed || len(second.Added) != 0 || len(second.Existing) != 1 || second.Baseline.Revision != "1.0.0" {
		t.Fatalf("idempotent append = %#v", second)
	}
	entryTwo := entry
	entryTwo.FindingKey = "finding:two"
	third, err := service.AppendBaseline(context.Background(), BaselineAppendRequest{Profile: first.Profile, ProfileFileName: "main.json", BaselineFileName: "main.json", Entries: []quality.BaselineEntry{entryTwo}, ExpectedRevision: "1.0.0"})
	if err != nil {
		t.Fatalf("second entry append: %v", err)
	}
	if !third.Changed || third.Baseline.Revision != "1.0.1" || len(third.Baseline.Entries) != 2 {
		t.Fatalf("second entry append = %#v", third)
	}
	if _, err := service.AppendBaseline(context.Background(), BaselineAppendRequest{Profile: third.Profile, ProfileFileName: "main.json", BaselineFileName: "main.json", Entries: []quality.BaselineEntry{entryTwo}, ExpectedRevision: "1.0.0"}); !errors.Is(err, ErrBaselineConflict) {
		t.Fatalf("stale expected revision error = %v, want ErrBaselineConflict", err)
	}
	if _, err := service.AppendBaseline(context.Background(), BaselineAppendRequest{Profile: profile, ProfileFileName: "main.json", BaselineFileName: "other.json", Entries: []quality.BaselineEntry{entry}, ExpectedRevision: "1.0.0"}); !errors.Is(err, ErrBaselineConflict) {
		t.Fatalf("expected revision on missing baseline error = %v, want ErrBaselineConflict", err)
	}
	resolved, info, err := store.ResolveBaseline(context.Background(), "baseline:main", "1.0.1")
	if err != nil || info.FileName != "main.json" || len(resolved.Entries) != 2 {
		t.Fatalf("resolved appended baseline = %#v, info = %#v, err = %v", resolved, info, err)
	}
}

func TestWriteManagedPairRollsBackBaselineWhenProfilePublicationFails(t *testing.T) {
	root := t.TempDir()
	store, err := NewFileStore(root)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(store, quality.NewDefaultCatalog())
	previous := quality.Baseline{
		SchemaVersion: quality.BaselineSchemaVersion,
		BaselineID:    "baseline:rollback",
		Revision:      "1.0.0",
		Entries:       []quality.BaselineEntry{},
		Extensions:    []quality.ExtensionBlock{},
	}
	if _, err := store.SaveBaseline(context.Background(), previous, "rollback.json", false); err != nil {
		t.Fatalf("save previous baseline: %v", err)
	}
	path := filepath.Join(root, BaselineDirectoryName, "rollback.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read previous baseline: %v", err)
	}
	next := previous
	next.Revision = "1.0.1"
	profile := testPolicyProfile("profile:rollback", "1.0.0")
	if _, _, err := service.writeManagedPair(context.Background(), next, profile, "rollback.json", "../profile.json", true, previous); err == nil {
		t.Fatal("managed pair publication unexpectedly succeeded with an unsafe profile path")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read rolled-back baseline: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("baseline changed after failed profile publication: before=%s after=%s", before, after)
	}
}
