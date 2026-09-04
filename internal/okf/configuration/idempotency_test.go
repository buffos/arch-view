package configuration

import (
	"context"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestStoreScopesOperationIdentityByProject(t *testing.T) {
	store := NewStore()
	ctx := context.Background()
	first, second := t.TempDir(), t.TempDir()
	for _, item := range []struct{ root, graph string }{{first, "first/.okf"}, {second, "second/.okf"}} {
		value := domain.ProjectConfiguration{DefaultGraph: item.graph}
		if _, err := store.Save(ctx, item.root, value, "", "same-operation", []byte("same-input")); err != nil {
			t.Fatal(err)
		}
		loaded, err := store.Load(ctx, item.root)
		if err != nil || loaded.DefaultGraph != item.graph {
			t.Fatalf("project isolation failed: %#v %v", loaded, err)
		}
	}
}
