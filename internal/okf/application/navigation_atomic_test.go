package application

import (
	"context"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
)

type pausedNavigationRule struct {
	armed   atomic.Bool
	entered chan struct{}
	release chan struct{}
}

func (*pausedNavigationRule) ID() string          { return "test.pause" }
func (*pausedNavigationRule) Version() string     { return "1" }
func (*pausedNavigationRule) Description() string { return "Controlled projection pause" }
func (rule *pausedNavigationRule) Evaluate(ctx context.Context, _ domain.ConceptDocument, _ domain.RuleInvocation) (ports.RuleResult, error) {
	if rule.armed.CompareAndSwap(true, false) {
		close(rule.entered)
		select {
		case <-rule.release:
		case <-ctx.Done():
			return ports.RuleResult{}, ctx.Err()
		}
	}
	return ports.RuleResult{}, nil
}

func TestNavigationPublishesOnlySuccessfulCurrentProjection(t *testing.T) {
	for _, cancelRequest := range []bool{true, false} {
		t.Run(map[bool]string{true: "cancelled", false: "superseded"}[cancelRequest], func(t *testing.T) {
			root := t.TempDir()
			writeApplicationFile(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: concept\n---\n")
			service := New(root)
			if _, err := service.Session(context.Background(), ""); err != nil {
				t.Fatal(err)
			}
			rule := &pausedNavigationRule{entered: make(chan struct{}), release: make(chan struct{})}
			if err := service.registry.Register(rule); err != nil {
				t.Fatal(err)
			}
			service.registry.SetProjectProfiles([]domain.Profile{{ProfileID: "project:pause", Rules: []domain.RuleInvocation{{RuleID: rule.ID(), Version: "1", Enabled: true}}}})
			before, err := service.SelectProfile(context.Background(), "", "project:pause")
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			rule.armed.Store(true)
			done := make(chan error, 1)
			go func() { _, err := service.Focus(ctx, "", "root"); done <- err }()
			select {
			case <-rule.entered:
			case <-time.After(5 * time.Second):
				t.Fatal("projection did not start")
			}
			during, err := service.Session(context.Background(), "")
			if err != nil || !reflect.DeepEqual(before, during) {
				t.Fatal("uncommitted navigation became visible")
			}
			expected := before
			if cancelRequest {
				cancel()
			} else {
				expected, err = service.SetNavigation(context.Background(), "", 3, false)
				if err != nil {
					t.Fatal(err)
				}
				close(rule.release)
			}
			select {
			case err := <-done:
				if err == nil {
					t.Fatal("cancelled or superseded projection succeeded")
				}
			case <-time.After(5 * time.Second):
				t.Fatal("projection did not finish")
			}
			after, err := service.Session(context.Background(), "")
			if err != nil || !reflect.DeepEqual(expected, after) {
				t.Fatal("failed projection overwrote session")
			}
			service.mu.RLock()
			history := len(service.sessionLockedRead("").History)
			service.mu.RUnlock()
			if history != 0 {
				t.Fatal("failed focus changed history")
			}
		})
	}
}
