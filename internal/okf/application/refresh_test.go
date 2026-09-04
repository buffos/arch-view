package application

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/buffo/arch-view/internal/okf/configuration"
	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
)

func TestRefreshPreservesCancellationAndDeadlineCauses(t *testing.T) {
	service := New(t.TempDir())
	if _, err := service.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	before := service.Catalog()
	for _, timeout := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		wantCause, wantCode, wantStatus := context.Canceled, "okf_operation_cancelled", 409
		if timeout {
			ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			wantCause, wantCode, wantStatus = context.DeadlineExceeded, "okf_operation_timeout", 504
		}
		_, err := service.Refresh(ctx)
		cancel()
		var failure *domain.Error
		if !errors.Is(err, wantCause) || !errors.As(err, &failure) || failure.Code != wantCode || failure.Status != wantStatus {
			t.Fatalf("timeout=%v error=%+v", timeout, err)
		}
		if !reflect.DeepEqual(before, service.Catalog()) {
			t.Fatal("interrupted refresh changed published catalog")
		}
	}
}

type cancelAfterConfigLoad struct {
	ports.ConfigurationStore
	cancel context.CancelFunc
}

func (store *cancelAfterConfigLoad) Load(ctx context.Context, root string) (domain.ProjectConfiguration, error) {
	value, err := store.ConfigurationStore.Load(ctx, root)
	if store.cancel != nil {
		store.cancel()
	}
	return value, err
}

func TestCancelledRefreshPreservesPublishedCatalogAndSession(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, ".okf", "root.md")
	writeApplicationFile(t, file, "---\ntype: concept\ntitle: Before\n---\n")
	store := &cancelAfterConfigLoad{ConfigurationStore: configuration.NewStore()}
	service := NewWithDependencies(root, nil, nil, store, nil)
	if _, err := service.Session(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	beforeCatalog := service.Catalog()
	beforeSession := service.sessionLockedRead("")
	beforeIndex := service.indexes[".okf"]
	writeApplicationFile(t, file, "---\ntype: concept\ntitle: After\n---\n")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store.cancel = cancel
	if _, err := service.Refresh(ctx); !hasApplicationCode(err, "okf_operation_cancelled") {
		t.Fatalf("cancelled refresh published: %v", err)
	}
	if !reflect.DeepEqual(beforeCatalog, service.Catalog()) || !reflect.DeepEqual(beforeSession, service.sessionLockedRead("")) || !reflect.DeepEqual(beforeIndex, service.indexes[".okf"]) {
		t.Fatal("cancelled refresh changed published state")
	}
}

func TestRefreshAllowsRecoveryFromDisappearingFocusAndInvalidBundle(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	primary := filepath.Join(root, ".okf", "root.md")
	child := filepath.Join(root, ".okf", "child.md")
	writeApplicationFile(t, primary, "---\ntype: concept\n---\n")
	writeApplicationFile(t, child, "---\ntype: concept\nparent: root\n---\n")
	writeApplicationFile(t, filepath.Join(root, "other", ".okf", "other.md"), "---\ntype: concept\n---\n")
	service := New(root)
	if _, err := service.Session(ctx, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Focus(ctx, "", "child"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(child); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Session(ctx, ""); !hasApplicationCode(err, "okf_concept_not_found") {
		t.Fatalf("missing focus: %v", err)
	}
	back, err := service.Back(ctx, "")
	if err != nil || back.Projection == nil || back.Navigation.FocusRoot != "" {
		t.Fatalf("Back failed to recover: %#v %v", back, err)
	}
	writeApplicationFile(t, primary, "invalid frontmatter")
	catalog, err := service.Refresh(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Bundles) != 2 || catalog.Bundles[0].Selectable {
		t.Fatalf("invalid bundle selectable: %#v", catalog)
	}
	selected, err := service.SelectBundle(ctx, "", "other/.okf")
	if err != nil || selected.Projection == nil || len(selected.Projection.Nodes) != 1 || selected.Projection.Nodes[0].ConceptID != "other" {
		t.Fatalf("independent bundle recovery failed: %#v %v", selected, err)
	}
}
