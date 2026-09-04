package viewer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/buffo/arch-view/internal/okf/application"
	"github.com/buffo/arch-view/internal/okf/contract"
	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
	"github.com/buffo/arch-view/internal/okf/profile"
)

type httpPausedProjection struct {
	armed   atomic.Bool
	entered chan struct{}
	release chan struct{}
}

func (*httpPausedProjection) ID() string          { return "test.http_pause" }
func (*httpPausedProjection) Version() string     { return "1" }
func (*httpPausedProjection) Description() string { return "Pause one HTTP projection" }
func (rule *httpPausedProjection) Evaluate(ctx context.Context, _ domain.ConceptDocument, _ domain.RuleInvocation) (ports.RuleResult, error) {
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

func TestOKFHTTPReportsCancelledAndSupersededProjections(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		name := "superseded"
		if cancelled {
			name = "cancelled"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeOKFHTTPFixture(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: topic\n---\n")
			rule := &httpPausedProjection{entered: make(chan struct{}), release: make(chan struct{}, 1)}
			defer close(rule.release)
			registry := profile.NewRegistry()
			if err := registry.Register(rule); err != nil {
				t.Fatal(err)
			}
			api := application.NewWithDependencies(root, nil, nil, nil, registry)
			_, err := api.SaveProfileAs(context.Background(), domain.Profile{Rules: []domain.RuleInvocation{{RuleID: rule.ID(), Version: "1", Enabled: true}}}, "", "pause", "", "setup", nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := api.SelectProfile(context.Background(), "", "project:pause"); err != nil {
				t.Fatal(err)
			}
			server, err := NewServer(fixtureModel(t), ServerOptions{SourceRoot: root, OKFApplication: api})
			if err != nil {
				t.Fatal(err)
			}
			handler := server.Handler()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			old := httptest.NewRecorder()
			done := make(chan struct{})
			rule.armed.Store(true)
			go func() {
				defer close(done)
				handler.ServeHTTP(old, httptest.NewRequest(http.MethodPost, "/v1/okf/sessions/default/navigation/focus", strings.NewReader(`{"concept_id":"root"}`)).WithContext(ctx))
			}()
			select {
			case <-rule.entered:
			case <-time.After(5 * time.Second):
				t.Fatal("older HTTP projection did not start")
			}
			wantCode, wantDepth := "okf_operation_cancelled", 2
			if !cancelled {
				newer := httptest.NewRecorder()
				handler.ServeHTTP(newer, httptest.NewRequest(http.MethodPut, "/v1/okf/sessions/default/navigation/depth", strings.NewReader(`{"depth":3,"full":false}`)))
				if newer.Code != http.StatusOK {
					t.Fatalf("newer request: %d %s", newer.Code, newer.Body.String())
				}
				wantCode, wantDepth = "okf_operation_superseded", 3
				rule.release <- struct{}{}
			} else {
				cancel()
			}
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("older HTTP projection did not finish")
			}
			var failure contract.Failure
			if err := json.Unmarshal(old.Body.Bytes(), &failure); err != nil || old.Code != http.StatusConflict || failure.Error.Code != wantCode || failure.Meta.RequestID == "" {
				t.Fatalf("failure envelope: %d %s (%v)", old.Code, old.Body.String(), err)
			}
			current := httptest.NewRecorder()
			handler.ServeHTTP(current, httptest.NewRequest(http.MethodGet, "/v1/okf/sessions/default/projection", nil))
			var projection struct{ Data domain.ProjectionSnapshot }
			if err := json.Unmarshal(current.Body.Bytes(), &projection); err != nil || current.Code != http.StatusOK || projection.Data.Navigation.Depth != wantDepth || projection.Data.Navigation.FocusRoot != "" || projection.Data.Navigation.CanGoBack {
				t.Fatalf("stale projection/history published: %d %s (%v)", current.Code, current.Body.String(), err)
			}
			if projection.Data.Profile.ProfileID != "project:pause" || projection.Data.Source.BundleID != ".okf" || len(projection.Data.Nodes) != 1 || projection.Data.Nodes[0].ConceptID != "root" {
				t.Fatalf("surviving projection lost source/profile: %+v", projection.Data)
			}
		})
	}
}
