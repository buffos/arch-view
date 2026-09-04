package application

import (
	"context"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
	"github.com/buffo/arch-view/internal/okf/profile"
)

type pausedCatalogValidator struct {
	armed   atomic.Bool
	entered chan struct{}
	release chan struct{}
}

func (*pausedCatalogValidator) ID() string          { return "review.catalog_validation" }
func (*pausedCatalogValidator) Version() string     { return "1" }
func (*pausedCatalogValidator) Description() string { return "Catalog consistency test" }
func (*pausedCatalogValidator) Evaluate(context.Context, domain.ConceptDocument, domain.RuleInvocation) (ports.RuleResult, error) {
	return ports.RuleResult{}, nil
}
func (rule *pausedCatalogValidator) ValidateParameters(map[string]any) error {
	if rule.armed.CompareAndSwap(true, false) {
		close(rule.entered)
		<-rule.release
	}
	return nil
}

func TestProfileCatalogResolvesOneConfigurationSnapshot(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rule := &pausedCatalogValidator{entered: make(chan struct{}), release: make(chan struct{})}
	var once sync.Once
	release := func() { once.Do(func() { close(rule.release) }) }
	defer release()
	registry := profile.NewRegistry()
	if err := registry.Register(rule); err != nil {
		t.Fatal(err)
	}
	service := NewWithDependencies(t.TempDir(), nil, nil, nil, registry)
	if _, err := service.SaveProfileAs(ctx, domain.Profile{Name: "A", Rules: []domain.RuleInvocation{{RuleID: rule.ID(), Version: "1", Enabled: true}}}, "", "a", "", "create-a", nil); err != nil {
		t.Fatal(err)
	}
	saved, err := service.SaveProfileAs(ctx, domain.Profile{Name: "B"}, "", "b", "", "create-b", nil)
	if err != nil {
		t.Fatal(err)
	}
	before, err := service.Profiles(ctx)
	if err != nil {
		t.Fatal(err)
	}
	type result struct {
		catalog domain.ProfileCatalog
		err     error
	}
	done := make(chan result, 1)
	rule.armed.Store(true)
	go func() { value, err := service.Profiles(ctx); done <- result{value, err} }()
	select {
	case <-rule.entered:
	case <-ctx.Done():
		t.Fatal("catalog validation did not pause")
	}
	if _, err := service.DeleteProfile(ctx, "project:b", "", false, saved.Revision, "delete-b", nil); err != nil {
		t.Fatal(err)
	}
	release()
	select {
	case value := <-done:
		if value.err != nil {
			t.Fatal(value.err)
		}
		if !reflect.DeepEqual(value.catalog, before) {
			t.Fatalf("catalog mixed old declarations/revision with new resolution: before=%+v after=%+v", before, value.catalog)
		}
	case <-ctx.Done():
		t.Fatal("catalog request did not finish")
	}
	current, err := service.Profiles(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if current.ConfigurationRevision == before.ConfigurationRevision {
		t.Fatal("new request did not see the deletion revision")
	}
	for _, value := range current.Profiles {
		if value.ProfileID == "project:b" {
			t.Fatal("new request retained deleted profile")
		}
	}
}
