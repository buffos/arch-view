package viewer

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/buffo/arch-view/internal/okf/application"
	"github.com/buffo/arch-view/internal/okf/domain"
)

// SC-005: change presentation through HTTP and return to the original costume.
func TestOKFHTTPProfileSwitchPreservesSourceAndRestoresNeutral(t *testing.T) {
	root := t.TempDir()
	sources := map[string]string{
		"root.md":  "---\ntype: area\ntitle: Root\n---\nRoot body\n",
		"child.md": "---\ntype: item\ntitle: Child\nparent: root\nstate: implemented\ncustom: retained\n---\nChild body\n",
	}
	for path, content := range sources {
		writeOKFHTTPFixture(t, filepath.Join(root, ".okf", path), content)
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
		t.Fatalf("select bundle: %d", selection.status)
	}
	selectProfile := func(id string) domain.ProjectionSnapshot {
		t.Helper()
		response := postOKFHTTP(t, base+"/profile", http.MethodPut, map[string]any{"profile_id": id})
		var envelope struct {
			Data application.SessionView `json:"data"`
		}
		decodeOKFHTTP(t, response, &envelope)
		if response.status != http.StatusOK || envelope.Data.Projection == nil || envelope.Data.ProfileID != id {
			t.Fatalf("profile selection: %d %+v", response.status, envelope.Data)
		}
		return *envelope.Data.Projection
	}
	neutral := selectProfile("builtin:neutral")
	fog := selectProfile("builtin:fog-of-war")
	if len(neutral.Nodes) != 2 || len(fog.Nodes) != 2 || neutral.Source != fog.Source || neutral.Source.SourceRevision == "" {
		t.Fatal("profile switch changed source identity")
	}
	changed := false
	for i, node := range neutral.Nodes {
		other := fog.Nodes[i]
		if node.ID != other.ID || node.ConceptID != other.ConceptID || node.SourcePath != other.SourcePath {
			t.Fatal("profile switch changed concept identity")
		}
		if len(node.PresentationFields) != 0 {
			t.Fatal("Neutral contains profile-specific fields")
		}
		if !reflect.DeepEqual(node.PresentationStyle, other.PresentationStyle) {
			changed = true
		}
		if other.ConceptID == "child" && (other.EffectiveState != "implemented" || len(other.PresentationFields) != 1 || other.PresentationFields[0].Value != "implemented") {
			t.Fatalf("Fog failed to interpret state: %+v", other)
		}
	}
	if !changed {
		t.Fatal("different profiles produced identical styling")
	}
	restored := selectProfile("builtin:neutral")
	if !reflect.DeepEqual(neutral.Nodes, restored.Nodes) || neutral.Source != restored.Source || neutral.ProjectionRevision != restored.ProjectionRevision {
		t.Fatal("switching back retained Fog presentation")
	}
	for path, expected := range sources {
		actual, err := os.ReadFile(filepath.Join(root, ".okf", path))
		if err != nil || string(actual) != expected {
			t.Fatalf("source modified: %s %v", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".archview.json")); !os.IsNotExist(err) {
		t.Fatalf("session-only profile switch persisted configuration: %v", err)
	}
}
