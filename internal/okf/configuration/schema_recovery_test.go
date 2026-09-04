package configuration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestUnsupportedSchemaIsDiagnosedWithoutRewritingAndCanBeRepaired(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".archview.json")
	invalid := []byte(`{"keep":{"value":42},"okf":{"schema_version":"arch-view.okf/v999","profiles":[]}}`)
	if err := os.WriteFile(path, invalid, 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewStore()
	_, err := store.Load(context.Background(), root)
	if !hasCode(err, "okf_configuration_invalid") || !strings.Contains(err.Error(), "unsupported OKF configuration schema") {
		t.Fatalf("wrong schema diagnostic: %v", err)
	}
	if strings.Contains(err.Error(), "not valid JSON") {
		t.Fatalf("schema incompatibility was mislabeled as JSON syntax: %v", err)
	}
	_, err = store.Save(context.Background(), root, domain.ProjectConfiguration{}, revision(invalid), "unsupported-save", nil)
	if !hasCode(err, "okf_configuration_invalid") {
		t.Fatalf("unsupported schema was overwritten: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(invalid) {
		t.Fatalf("failed load/save modified configuration: %s, %v", after, err)
	}
	repaired := strings.Replace(string(invalid), "arch-view.okf/v999", SectionSchemaVersion, 1)
	if err := os.WriteFile(path, []byte(repaired), 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(context.Background(), root)
	if err != nil || loaded.SchemaVersion != SectionSchemaVersion || loaded.Revision == "" {
		t.Fatalf("schema repair did not recover: %+v %v", loaded, err)
	}
}
