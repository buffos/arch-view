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

// SC-001 backend boundary: both sources are valid and selection is reversible.
func TestOKFHTTPSelectsIndependentValidBundlesWithoutSourceMutation(t *testing.T) {
	root := t.TempDir()
	sources := map[string]string{
		"a/.okf/alpha.md": "---\ntype: concept\ntitle: Alpha\n---\nAlpha source\n",
		"z/.okf/zeta.md":  "---\ntype: concept\ntitle: Zeta\n---\nZeta source\n",
	}
	for path, content := range sources {
		writeOKFHTTPFixture(t, filepath.Join(root, filepath.FromSlash(path)), content)
	}
	server, err := NewServer(fixtureModel(t), ServerOptions{SourceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	host := httptest.NewServer(server.Handler())
	defer host.Close()
	response := getOKFHTTP(t, host.URL+"/v1/okf/catalog")
	var catalog struct {
		Data domain.BundleCatalog `json:"data"`
	}
	decodeOKFHTTP(t, response, &catalog)
	if response.status != http.StatusOK {
		t.Fatalf("catalog: %d", response.status)
	}
	var ids []string
	for _, bundle := range catalog.Data.Bundles {
		if !bundle.Selectable {
			t.Fatalf("valid bundle not selectable: %+v", bundle)
		}
		ids = append(ids, bundle.BundleID)
	}
	if !reflect.DeepEqual(ids, []string{"a/.okf", "z/.okf"}) {
		t.Fatalf("catalog order: %v", ids)
	}
	for _, selection := range []struct{ bundle, concept string }{
		{"a/.okf", "alpha"}, {"z/.okf", "zeta"}, {"a/.okf", "alpha"},
	} {
		response := postOKFHTTP(t, host.URL+"/v1/okf/sessions/default/bundle", http.MethodPut, map[string]any{"bundle_id": selection.bundle})
		var envelope struct {
			Data application.SessionView `json:"data"`
		}
		decodeOKFHTTP(t, response, &envelope)
		view := envelope.Data
		if response.status != http.StatusOK || view.BundleID != selection.bundle || view.Projection == nil {
			t.Fatalf("selection: %d %+v", response.status, view)
		}
		if view.ProfileID != "builtin:neutral" || view.Projection.Source.BundleID != selection.bundle || len(view.Projection.Nodes) != 1 || view.Projection.Nodes[0].ConceptID != selection.concept {
			t.Fatalf("bundle isolation/default profile: %+v", view.Projection)
		}
		for _, other := range []string{"alpha", "zeta"} {
			if other == selection.concept {
				continue
			}
			response := getOKFHTTP(t, host.URL+"/v1/okf/sessions/default/concept-detail?concept_id="+other)
			if response.status != http.StatusNotFound {
				t.Fatalf("foreign concept detail escaped bundle: %d", response.status)
			}
		}
	}
	for path, expected := range sources {
		actual, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil || string(actual) != expected {
			t.Fatalf("source modified: %s %v", path, err)
		}
	}
}
