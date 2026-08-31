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

func TestFileStoreListsAndResolvesBaselinesWithoutGuessing(t *testing.T) {
	root := t.TempDir()
	store, err := NewFileStore(root)
	if err != nil {
		t.Fatal(err)
	}
	base := func(id, revision string) quality.Baseline {
		return quality.Baseline{SchemaVersion: quality.BaselineSchemaVersion, BaselineID: id, Revision: revision, Entries: []quality.BaselineEntry{}, Extensions: []quality.ExtensionBlock{}}
	}
	if _, err := store.SaveBaseline(context.Background(), base("baseline:main", "1.0.1"), "z.json", false); err != nil {
		t.Fatalf("save z baseline: %v", err)
	}
	if _, err := store.SaveBaseline(context.Background(), base("baseline:main", "1.0.0"), "a.json", false); err != nil {
		t.Fatalf("save a baseline: %v", err)
	}
	if _, err := store.SaveBaseline(context.Background(), base("baseline:other", "1.0.0"), "other.json", false); err != nil {
		t.Fatalf("save other baseline: %v", err)
	}
	invalid := base("baseline:broken", "1.0.0")
	invalid.SchemaVersion = "arch-view.quality-baseline/v0"
	if _, err := store.SaveBaseline(context.Background(), invalid, "broken.json", false); err != nil {
		t.Fatalf("save invalid baseline fixture: %v", err)
	}
	infos, err := store.ListBaselines(context.Background())
	if err != nil {
		t.Fatalf("list baselines: %v", err)
	}
	if len(infos) != 4 || infos[0].FileName != "broken.json" || infos[1].FileName != "a.json" || infos[2].FileName != "z.json" {
		t.Fatalf("baseline list = %#v", infos)
	}
	resolved, info, err := store.ResolveBaseline(context.Background(), "baseline:main", "1.0.0")
	if err != nil || resolved.Revision != "1.0.0" || info.FileName != "a.json" {
		t.Fatalf("resolved baseline = %#v, info = %#v, err = %v", resolved, info, err)
	}
	if _, _, err := store.ResolveBaseline(context.Background(), "baseline:main", ""); !errors.Is(err, ErrBaselineConflict) {
		t.Fatalf("ambiguous baseline error = %v, want ErrBaselineConflict", err)
	}
	if _, _, err := store.ResolveBaseline(context.Background(), "baseline:missing", "1.0.0"); !errors.Is(err, ErrBaselineNotFound) {
		t.Fatalf("missing baseline error = %v, want ErrBaselineNotFound", err)
	}
	if _, _, err := store.ResolveBaseline(context.Background(), "baseline:broken", "1.0.0"); !errors.Is(err, ErrBaselineInvalid) {
		t.Fatalf("invalid baseline error = %v, want ErrBaselineInvalid", err)
	}
}

func TestFileStoreTreatsMalformedManagedDocumentAsInvalid(t *testing.T) {
	root := t.TempDir()
	store, err := NewFileStore(root)
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, BaselineDirectoryName)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "main.json"), []byte("{not-json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.ResolveBaseline(context.Background(), "baseline:main", "1.0.0"); !errors.Is(err, ErrBaselineInvalid) {
		t.Fatalf("malformed managed baseline error = %v, want ErrBaselineInvalid", err)
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
