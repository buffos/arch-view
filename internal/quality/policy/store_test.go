package policy

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/buffo/arch-view/internal/quality"
)

func TestFileStoreRestrictsDestinationsAndReportsConflicts(t *testing.T) {
	root := t.TempDir()
	store, err := NewFileStore(root)
	if err != nil {
		t.Fatal(err)
	}
	profile := testPolicyProfile("profile:test", "1.0.0")

	if _, err := store.SaveProfile(context.Background(), profile, "../escape.json", false); !errors.Is(err, ErrInvalidDestination) {
		t.Fatalf("path traversal error = %v, want ErrInvalidDestination", err)
	}
	if _, err := os.Stat(filepath.Join(root, ProfileDirectoryName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid destination created policy directory: %v", err)
	}

	first, err := store.SaveProfile(context.Background(), profile, "test.json", false)
	if err != nil {
		t.Fatalf("first profile write: %v", err)
	}
	if first.RelativePath != "quality-profiles/test.json" || first.Digest.Value == "" || first.Overwritten {
		t.Fatalf("write result = %#v", first)
	}
	if _, err := store.SaveProfile(context.Background(), profile, "test.json", false); !errors.Is(err, ErrDestinationConflict) {
		t.Fatalf("conflicting profile write = %v, want ErrDestinationConflict", err)
	}
	updated := profile
	updated.ProfileVersion = "1.0.1"
	second, err := store.SaveProfile(context.Background(), updated, "test.json", true)
	if err != nil {
		t.Fatalf("overwrite profile: %v", err)
	}
	if !second.Overwritten || second.Digest.Value == first.Digest.Value {
		t.Fatalf("overwrite result = %#v", second)
	}

	resolved, info, err := store.ProfileDocument(context.Background(), "profile:test", "1.0.1")
	if err != nil {
		t.Fatalf("resolve overwritten profile: %v", err)
	}
	if resolved.ProfileVersion != "1.0.1" || info.FileName != "test.json" || info.Status != "available" {
		t.Fatalf("resolved profile = %#v, info = %#v", resolved, info)
	}
}

func TestServiceValidatesBeforeWritingAndBaselineStoreIsBounded(t *testing.T) {
	root := t.TempDir()
	store, err := NewFileStore(root)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(store, quality.NewDefaultCatalog())
	invalid := testPolicyProfile("not-namespaced", "1.0.0")
	if _, err := service.SaveProfile(context.Background(), invalid, "invalid.json", false); err == nil {
		t.Fatal("invalid profile was saved")
	}
	if _, err := os.Stat(filepath.Join(root, ProfileDirectoryName, "invalid.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid profile exists after failed validation: %v", err)
	}

	baseline := quality.Baseline{SchemaVersion: quality.BaselineSchemaVersion, BaselineID: "baseline:test", Entries: []quality.BaselineEntry{}, Extensions: []quality.ExtensionBlock{}}
	if _, err := service.SaveBaseline(context.Background(), baseline, "../baseline.json", false); !errors.Is(err, ErrInvalidDestination) {
		t.Fatalf("baseline traversal error = %v", err)
	}
	if _, err := service.SaveBaseline(context.Background(), baseline, "test.json", false); err != nil {
		t.Fatalf("baseline write: %v", err)
	}
	if _, err := store.ReadBaseline(context.Background(), "quality-baselines/test.json"); !errors.Is(err, ErrInvalidDestination) {
		t.Fatalf("nested baseline read = %v, want ErrInvalidDestination", err)
	}
}

func testPolicyProfile(id, version string) quality.QualityProfile {
	return quality.QualityProfile{
		SchemaVersion:  quality.SchemaVersion,
		ProfileID:      id,
		ProfileVersion: version,
		EnabledRules:   []quality.RuleBinding{},
		SeverityPolicy: quality.TypedConfigBlock{},
		Constraints:    []quality.ArchitectureConstraint{},
		Extensions:     []quality.ExtensionBlock{},
	}
}
