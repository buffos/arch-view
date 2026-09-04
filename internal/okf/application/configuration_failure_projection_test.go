package application

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/buffo/arch-view/internal/okf/configuration"
	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
	"github.com/buffo/arch-view/internal/okf/profile"
)

type failingProjectionStore struct {
	ports.ConfigurationStore
	fail atomic.Bool
	err  error
}

func (store *failingProjectionStore) Save(ctx context.Context, root string, value domain.ProjectConfiguration, revision, operation string, input []byte) (domain.ProjectConfiguration, error) {
	if store.fail.Load() {
		return domain.ProjectConfiguration{}, store.err
	}
	return store.ConfigurationStore.Save(ctx, root, value, revision, operation, input)
}

// SC-017 x SC-019: failed persistence must not invalidate valid pending work.
func TestFailedProfileSavePreservesInFlightProjectionAndConfiguration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	root := t.TempDir()
	configPath := filepath.Join(root, ".archview.json")
	writeApplicationFile(t, configPath, `{"layout":{"algorithm":"layered"},"analysis":{"custom":"retained"},"unknown":{"value":42}}`)
	writeApplicationFile(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: concept\ntitle: Root\n---\n")
	registry := profile.NewRegistry()
	rule := &pausedNavigationRule{entered: make(chan struct{}), release: make(chan struct{})}
	if err := registry.Register(rule); err != nil {
		t.Fatal(err)
	}
	writeFailure := errors.New("injected write failure")
	store := &failingProjectionStore{ConfigurationStore: configuration.NewStore(), err: writeFailure}
	service := NewWithDependencies(root, nil, nil, store, registry)
	saved, err := service.SaveProfileAs(ctx, domain.Profile{Name: "Working", Rules: []domain.RuleInvocation{{RuleID: rule.ID(), Version: "1", Enabled: true}}}, "", "working", "", "create", nil)
	if err != nil {
		t.Fatal(err)
	}
	before, err := service.SelectProfile(ctx, "", "project:working")
	if err != nil {
		t.Fatal(err)
	}
	beforeBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	rule.armed.Store(true)
	done := make(chan error, 1)
	go func() { _, err := service.Focus(ctx, "", "root"); done <- err }()
	select {
	case <-rule.entered:
	case <-ctx.Done():
		t.Fatal("projection did not pause")
	}
	store.fail.Store(true)
	updated := saved.Profiles[0]
	updated.Name = "Rejected change"
	if _, err := service.SaveProfile(ctx, updated, saved.Revision, "fail", nil); !errors.Is(err, writeFailure) {
		t.Fatalf("missing write error: %v", err)
	}
	current, err := service.Session(ctx, "")
	if err != nil || !reflect.DeepEqual(before, current) {
		t.Fatalf("failed save changed active session: %v", err)
	}
	afterBytes, err := os.ReadFile(configPath)
	if err != nil || string(beforeBytes) != string(afterBytes) {
		t.Fatalf("failed save changed configuration: %v", err)
	}
	close(rule.release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("failed save invalidated pending work: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("pending projection did not finish")
	}
	after, err := service.Session(ctx, "")
	if err != nil || after.Projection == nil || after.ProfileID != "project:working" || after.Navigation.FocusRoot != "root" || !after.Navigation.CanGoBack {
		t.Fatalf("pending focus not published: %+v %v", after, err)
	}
}
