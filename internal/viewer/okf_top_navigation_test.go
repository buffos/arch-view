package viewer

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/okf/application"
)

func TestOKFHTTPTopLevelPreservesContextAndHistory(t *testing.T) {
	for _, profileID := range []string{"builtin:neutral", "builtin:fog-of-war"} {
		for _, full := range []bool{false, true} {
			t.Run(profileID+"/full="+map[bool]string{false: "false", true: "true"}[full], func(t *testing.T) {
				root := t.TempDir()
				writeOKFHTTPFixture(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: topic\n---\n")
				writeOKFHTTPFixture(t, filepath.Join(root, ".okf", "child.md"), "---\ntype: topic\nparent: root\n---\n")
				server, err := NewServer(fixtureModel(t), ServerOptions{SourceRoot: root})
				if err != nil {
					t.Fatal(err)
				}
				host := httptest.NewServer(server.Handler())
				defer host.Close()
				base := host.URL + "/v1/okf/sessions/default"
				command := func(path, method string, body any) application.SessionView {
					t.Helper()
					response := postOKFHTTP(t, base+path, method, body)
					if response.status != http.StatusOK {
						t.Fatalf("%s: %d %s", path, response.status, response.body)
					}
					var envelope struct {
						Data application.SessionView `json:"data"`
					}
					decodeOKFHTTP(t, response, &envelope)
					return envelope.Data
				}
				command("/bundle", http.MethodPut, map[string]any{"bundle_id": ".okf"})
				command("/profile", http.MethodPut, map[string]any{"profile_id": profileID})
				command("/navigation/depth", http.MethodPut, map[string]any{"depth": 1, "full": full})
				command("/navigation/focus", http.MethodPost, map[string]any{"concept_id": "child"})
				for _, step := range []struct {
					path, focus string
					back        bool
				}{
					{"top", "", true}, {"top", "", true},
					{"back", "child", true}, {"back", "", false},
				} {
					view := command("/navigation/"+step.path, http.MethodPost, map[string]any{})
					if view.BundleID != ".okf" || view.ProfileID != profileID || view.Navigation.FocusRoot != step.focus || view.Navigation.Depth != 1 || view.Navigation.Full != full || view.Navigation.CanGoBack != step.back || view.Projection == nil {
						t.Fatalf("%s changed context or history: %+v", step.path, view)
					}
				}
				response := postOKFHTTP(t, base+"/navigation/back", http.MethodPost, map[string]any{})
				if response.status != http.StatusConflict || !strings.Contains(string(response.body), "okf_navigation_history_empty") {
					t.Fatalf("empty Back: %d %s", response.status, response.body)
				}
			})
		}
	}
}
