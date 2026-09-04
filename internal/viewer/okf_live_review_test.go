package viewer

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/buffo/arch-view/internal/okf/application"
	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
	"github.com/buffo/arch-view/internal/okf/profile"
)

type liveFocusContext struct{}
type livePausedProjection struct{ *httpPausedProjection }

func (rule livePausedProjection) Evaluate(ctx context.Context, document domain.ConceptDocument, invocation domain.RuleInvocation) (ports.RuleResult, error) {
	if ctx.Value(liveFocusContext{}) != true {
		return ports.RuleResult{}, nil
	}
	return rule.httpPausedProjection.Evaluate(ctx, document, invocation)
}

// Opt-in fixture for exercising the real browser against registered providers
// and worker-load failures. No review controls are present in production builds.
func TestOKFLiveReviewFixture(t *testing.T) {
	if os.Getenv("ARCHVIEW_OKF_LIVE_REVIEW") != "1" {
		t.Skip("interactive browser review fixture")
	}
	root := t.TempDir()
	writeOKFHTTPFixture(t, filepath.Join(root, ".okf/root.md"), "---\ntype: topic\ntitle: Source root\ncustom: retained\n---\nOriginal **content**\n")
	writeOKFHTTPFixture(t, filepath.Join(root, ".okf/target.md"), "---\ntype: topic\ntitle: Source target\nparent: root\n---\nTarget content\n")
	registry := profile.NewRegistry()
	paused := &httpPausedProjection{entered: make(chan struct{}), release: make(chan struct{}, 1)}
	defer close(paused.release)
	if err := registry.Register(livePausedProjection{paused}); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterPresentationPropertyProvider(httpPresentationProvider{}); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterDetailRenderer(httpDetailRenderer{}); err != nil {
		t.Fatal(err)
	}
	service := application.NewWithDependencies(root, nil, nil, nil, registry)
	if _, err := service.SaveProfileAs(context.Background(), domain.Profile{Name: "Review paused projection", Rules: []domain.RuleInvocation{{RuleID: paused.ID(), Version: "1", Enabled: true}}}, "", "pause", "", "pause", nil); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"custom", "panic"} {
		value := domain.Profile{Name: "Review " + mode, Rules: []domain.RuleInvocation{{RuleID: "test.presentation.http", Version: "1", Enabled: true}}, Style: domain.StyleSettings{Tokens: map[string]domain.StyleToken{"accent": {Fill: "#112233", Text: "#ffffff"}}}, Details: domain.DetailSettings{Renderer: &domain.DetailRendererSelection{ID: "test.detail.http", Version: "1", Parameters: map[string]any{"mode": mode}}}}
		if _, err := service.SaveProfileAs(context.Background(), value, "", mode, "", mode, nil); err != nil {
			t.Fatal(err)
		}
	}
	server, err := NewServer(fixtureModel(t), ServerOptions{SourceRoot: root, OKFApplication: service})
	if err != nil {
		t.Fatal(err)
	}
	var failWorker atomic.Bool
	var focusStatus atomic.Int32
	var stopOnce sync.Once
	stop := make(chan struct{})
	handler := server.Handler()
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		switch r.URL.Path {
		case "/__review/projection":
			if r.Method == http.MethodGet {
				entered := false
				select {
				case <-paused.entered:
					entered = true
				default:
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"entered":%t,"status":%d}`, entered, focusStatus.Load())
				return
			}
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			switch r.URL.Query().Get("action") {
			case "arm":
				select {
				case <-paused.entered:
					w.WriteHeader(http.StatusConflict)
					return
				default:
				}
				paused.armed.Store(true)
			case "release":
				select {
				case paused.release <- struct{}{}:
				default:
				}
			default:
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		case "/v1/okf/sessions/default/navigation/focus":
			recorded := httptest.NewRecorder()
			handler.ServeHTTP(recorded, r.WithContext(context.WithValue(r.Context(), liveFocusContext{}, true)))
			focusStatus.Store(int32(recorded.Code))
			fmt.Printf("Live focus response: %d %s\n", recorded.Code, recorded.Body.String())
			for key, values := range recorded.Header() {
				w.Header()[key] = values
			}
			w.WriteHeader(recorded.Code)
			w.Write(recorded.Body.Bytes())
			return
		case "/__review/worker":
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			failWorker.Store(r.URL.Query().Get("fail") == "1")
			w.WriteHeader(http.StatusNoContent)
			return
		case "/__review/stop":
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			stopOnce.Do(func() { close(stop) })
			w.WriteHeader(http.StatusNoContent)
			return
		case "/assets/vendor/elk-worker.min.js":
			w.Header().Set("Content-Type", "application/javascript")
			if failWorker.Load() {
				fmt.Fprint(w, `throw new Error("Controlled review worker failure");`)
				return
			}
			data, err := Asset("vendor/elk-worker.min.js")
			if err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Write(data)
			return
		}
		handler.ServeHTTP(w, r)
	}))
	defer host.Close()
	defer func() {
		select {
		case paused.release <- struct{}{}:
		default:
		}
	}()
	fmt.Println("OKF live review:", host.URL+"/?view=okf")
	select {
	case <-stop:
	case <-time.After(10 * time.Minute):
		t.Fatal("live review fixture timed out")
	}
}
