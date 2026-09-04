package application

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestProfileRetryMatchesTypedInputWithoutTransportBody(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	service := New(root)
	readConfig := func() []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(root, ".archview.json"))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	value := domain.Profile{Name: "Original"}
	saved, err := service.SaveProfileAs(ctx, value, "", "retry", "", "create", nil)
	if err != nil {
		t.Fatal(err)
	}
	before := readConfig()
	replayed, err := service.SaveProfileAs(ctx, value, "", "retry", "", "create", nil)
	if err != nil || !reflect.DeepEqual(replayed, saved) {
		t.Fatalf("exact Save As retry changed result: %v", err)
	}
	value.Name = "Changed"
	if _, err := service.SaveProfileAs(ctx, value, "", "retry", "", "create", nil); !hasApplicationCode(err, "okf_idempotency_conflict") {
		t.Fatalf("changed typed Save As input replayed: %v", err)
	}
	if !bytes.Equal(before, readConfig()) {
		t.Fatal("Save As retries changed persisted configuration")
	}
	value = saved.Profiles[0]
	value.Name = "Updated"
	updated, err := service.SaveProfile(ctx, value, saved.Revision, "update", nil)
	if err != nil {
		t.Fatal(err)
	}
	before = readConfig()
	replayed, err = service.SaveProfile(ctx, value, saved.Revision, "update", nil)
	if err != nil || !reflect.DeepEqual(replayed, updated) {
		t.Fatalf("exact Save retry changed result: %v", err)
	}
	value.Name = "Different update"
	if _, err := service.SaveProfile(ctx, value, saved.Revision, "update", nil); !hasApplicationCode(err, "okf_idempotency_conflict") {
		t.Fatalf("changed typed Save input replayed: %v", err)
	}
	if !bytes.Equal(before, readConfig()) {
		t.Fatal("Save retries changed persisted configuration")
	}
}
