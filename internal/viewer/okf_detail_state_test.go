package viewer

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestOKFHTTPDetailMatchesProjectionRollupState(t *testing.T) {
	for _, scenario := range []struct{ name, childState, expected string }{
		{"agreeing children", "implemented", "implemented"},
		{"mixed children", "specified", "bounded"},
		{"unknown child", "unmapped", "bounded"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			root := t.TempDir()
			writeOKFHTTPFixture(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: aggregate\ntitle: Root\nstate: bounded\n---\n")
			writeOKFHTTPFixture(t, filepath.Join(root, ".okf", "a.md"), "---\ntype: concept\nparent: root\nstate: implemented\n---\n")
			writeOKFHTTPFixture(t, filepath.Join(root, ".okf", "b.md"), "---\ntype: concept\nparent: root\nstate: "+scenario.childState+"\n---\n")
			server, err := NewServer(fixtureModel(t), ServerOptions{SourceRoot: root})
			if err != nil {
				t.Fatal(err)
			}
			host := httptest.NewServer(server.Handler())
			defer host.Close()
			response := postOKFHTTP(t, host.URL+"/v1/okf/sessions/default/bundle", http.MethodPut, map[string]any{"bundle_id": ".okf"})
			if response.status != http.StatusOK {
				t.Fatalf("select bundle: %d %s", response.status, response.body)
			}
			response = postOKFHTTP(t, host.URL+"/v1/okf/sessions/default/profile", http.MethodPut, map[string]any{"profile_id": "builtin:fog-of-war"})
			if response.status != http.StatusOK {
				t.Fatalf("select profile: %d %s", response.status, response.body)
			}
			var projection struct {
				Data domain.ProjectionSnapshot `json:"data"`
			}
			response = getOKFHTTP(t, host.URL+"/v1/okf/sessions/default/projection")
			decodeOKFHTTP(t, response, &projection)
			if response.status != http.StatusOK || len(projection.Data.Nodes) != 3 {
				t.Fatalf("projection: %d %s", response.status, response.body)
			}
			seenRoot := false
			for _, node := range projection.Data.Nodes {
				var detail struct {
					Data domain.ConceptDetail `json:"data"`
				}
				response = getOKFHTTP(t, host.URL+"/v1/okf/sessions/default/concept-detail?concept_id="+node.ConceptID)
				decodeOKFHTTP(t, response, &detail)
				if response.status != http.StatusOK {
					t.Fatalf("detail: %d %s", response.status, response.body)
				}
				if detail.Data.SourceRevision != projection.Data.Source.SourceRevision || detail.Data.BundleID != projection.Data.Source.BundleID || detail.Data.ConceptID != node.ConceptID {
					t.Fatalf("detail identity does not match projection: %+v", detail.Data)
				}
				if detail.Data.DeclaredState != node.DeclaredState || detail.Data.EffectiveState != node.EffectiveState {
					t.Fatalf("state mismatch for %s: detail=%s/%s projection=%s/%s", node.ConceptID, detail.Data.DeclaredState, detail.Data.EffectiveState, node.DeclaredState, node.EffectiveState)
				}
				if node.ConceptID == "root" && (detail.Data.DeclaredState != "bounded" || detail.Data.EffectiveState != scenario.expected || detail.Data.Frontmatter["state"] != "bounded") {
					t.Fatalf("incorrect roll-up or rewritten source state: %+v", detail.Data)
				}
				seenRoot = seenRoot || node.ConceptID == "root"
			}
			if !seenRoot {
				t.Fatal("roll-up root missing from projection")
			}
		})
	}
}
