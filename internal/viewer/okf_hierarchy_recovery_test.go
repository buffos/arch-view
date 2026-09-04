package viewer

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/buffo/arch-view/internal/okf/application"
	"github.com/buffo/arch-view/internal/okf/domain"
)

// SC-006: conflict exclusion preserves indexed concepts and explicit precedence.
func TestOKFHTTPHierarchyConflictRepairRetainsIndexedConcepts(t *testing.T) {
	root := t.TempDir()
	sources := map[string]string{
		"area.md":              "---\ntype: topic\n---\nArea\n",
		"other.md":             "---\ntype: topic\n---\nOther\n",
		"area/topic.md":        "---\ntype: topic\n---\nFallback child\n",
		"area/explicit.md":     "---\ntype: topic\nparent: /other.md\n---\nExplicit override\n",
		"area/topic/detail.md": "---\ntype: topic\nparent: [/area.md, /other.md]\n---\nInspectable conflict\n",
	}
	for path, source := range sources {
		writeOKFHTTPFixture(t, filepath.Join(root, ".okf", filepath.FromSlash(path)), source)
	}
	server, err := NewServer(fixtureModel(t), ServerOptions{SourceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	host := httptest.NewServer(server.Handler())
	defer host.Close()
	for _, repaired := range []bool{false, true} {
		if repaired {
			sources["area/topic/detail.md"] = "---\ntype: topic\nparent: /other.md\n---\nInspectable conflict\n"
			writeOKFHTTPFixture(t, filepath.Join(root, ".okf", "area", "topic", "detail.md"), sources["area/topic/detail.md"])
			response := postOKFHTTP(t, host.URL+"/v1/okf/catalog/refresh", http.MethodPost, map[string]any{})
			if response.status != http.StatusOK {
				t.Fatalf("refresh: %d", response.status)
			}
		}
		response := postOKFHTTP(t, host.URL+"/v1/okf/sessions/default/bundle", http.MethodPut, map[string]any{"bundle_id": ".okf"})
		var envelope struct {
			Data application.SessionView `json:"data"`
		}
		decodeOKFHTTP(t, response, &envelope)
		if response.status != http.StatusOK || envelope.Data.Projection == nil {
			t.Fatalf("selection: %d %+v", response.status, envelope.Data)
		}
		projection := envelope.Data.Projection
		var ids []string
		for _, node := range projection.Nodes {
			ids = append(ids, node.ConceptID)
		}
		sort.Strings(ids)
		wantIDs := []string{"area", "area/explicit", "area/topic", "area/topic/detail", "other"}
		if !reflect.DeepEqual(ids, wantIDs) {
			t.Fatalf("indexed concept lost, repaired=%v: %v", repaired, ids)
		}
		wantEdges := map[string]string{"area>area/topic": "filesystem_fallback", "other>area/explicit": "explicit_parent"}
		if repaired {
			wantEdges["other>area/topic/detail"] = "explicit_parent"
		}
		if len(projection.Relationships) != len(wantEdges) {
			t.Fatalf("unexpected containment: %+v", projection.Relationships)
		}
		for _, edge := range projection.Relationships {
			provenance, exists := wantEdges[edge.From+">"+edge.To]
			if !exists || edge.Kind != "containment" || len(edge.Provenance) != 1 || edge.Provenance[0].Source != provenance {
				t.Fatalf("wrong precedence/provenance: %+v", edge)
			}
		}
		conflict := false
		for _, diagnostic := range projection.Diagnostics {
			if diagnostic.Code == "okf_hierarchy_conflict" {
				conflict = true
				if diagnostic.ConceptID != "area/topic/detail" || diagnostic.Recovery == "" {
					t.Fatalf("unscoped conflict: %+v", diagnostic)
				}
			}
		}
		if conflict == repaired {
			t.Fatalf("conflict diagnostic after repaired=%v: %+v", repaired, projection.Diagnostics)
		}
		var summary struct {
			Data domain.BundleSummary `json:"data"`
		}
		decodeOKFHTTP(t, getOKFHTTP(t, host.URL+"/v1/okf/bundles/summary?bundle_id=.okf"), &summary)
		if !reflect.DeepEqual(summary.Data.Files, wantIDs) {
			t.Fatalf("summary dropped indexed concepts: %+v", summary.Data)
		}
		for path, source := range sources {
			actual, err := os.ReadFile(filepath.Join(root, ".okf", filepath.FromSlash(path)))
			if err != nil || string(actual) != source {
				t.Fatalf("viewer changed source %s: %v", path, err)
			}
		}
	}
}
