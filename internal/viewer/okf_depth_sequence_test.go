package viewer

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/buffo/arch-view/internal/okf/application"
)

// SC-009 HTTP sequence, with a cross-depth semantic link that must not extend
// the containment frontier (SC-007). The traversal origin is depth zero.
func TestOKFHTTPDepthSequencePreservesBundleAndProfile(t *testing.T) {
	root := t.TempDir()
	for path, content := range map[string]string{
		"root.md":                  "---\ntype: concept\n---\n[deep](/root/child/grand/leaf.md)\n",
		"root/child.md":            "---\ntype: concept\nparent: root\n---\n",
		"root/child/grand.md":      "---\ntype: concept\nparent: root/child\n---\n",
		"root/child/grand/leaf.md": "---\ntype: concept\nparent: root/child/grand\n---\n",
	} {
		writeOKFHTTPFixture(t, filepath.Join(root, ".okf", filepath.FromSlash(path)), content)
	}
	server, err := NewServer(fixtureModel(t), ServerOptions{SourceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	host := httptest.NewServer(server.Handler())
	defer host.Close()
	base := host.URL + "/v1/okf/sessions/default"
	selection := postOKFHTTP(t, base+"/bundle", http.MethodPut, map[string]any{"bundle_id": ".okf"})
	if selection.status != http.StatusOK {
		t.Fatalf("bundle selection: %d %s", selection.status, selection.body)
	}
	selection = postOKFHTTP(t, base+"/profile", http.MethodPut, map[string]any{"profile_id": "builtin:fog-of-war"})
	if selection.status != http.StatusOK {
		t.Fatalf("profile selection: %d %s", selection.status, selection.body)
	}
	for _, step := range []struct {
		depth int
		full  bool
		ids   []string
	}{
		{1, false, []string{"root", "root/child"}},
		{2, false, []string{"root", "root/child", "root/child/grand"}},
		{99, false, []string{"root", "root/child", "root/child/grand", "root/child/grand/leaf"}},
		{1, true, []string{"root", "root/child", "root/child/grand", "root/child/grand/leaf"}},
	} {
		response := postOKFHTTP(t, base+"/navigation/depth", http.MethodPut, map[string]any{"depth": step.depth, "full": step.full})
		var envelope struct {
			Data application.SessionView `json:"data"`
		}
		decodeOKFHTTP(t, response, &envelope)
		view := envelope.Data
		if response.status != http.StatusOK || view.Projection == nil {
			t.Fatalf("depth request: %d %s", response.status, response.body)
		}
		if view.BundleID != ".okf" || view.ProfileID != "builtin:fog-of-war" || view.Navigation.Depth != step.depth || view.Navigation.Full != step.full {
			t.Fatalf("context changed: %+v", view)
		}
		var ids []string
		for _, node := range view.Projection.Nodes {
			ids = append(ids, node.ConceptID)
			if !step.full && node.Depth > step.depth {
				t.Fatalf("depth frontier exceeded: %+v", node)
			}
		}
		if !reflect.DeepEqual(ids, step.ids) || view.Projection.Counts.HiddenNodes != 4-len(step.ids) {
			t.Fatalf("depth %d full %v: ids=%v counts=%+v", step.depth, step.full, ids, view.Projection.Counts)
		}
	}
}
