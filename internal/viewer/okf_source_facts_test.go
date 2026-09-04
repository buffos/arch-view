package viewer

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

// SC-003: inspect generic source facts through the real HTTP boundary.
func TestOKFHTTPPreservesArbitrarySourceFactsAndLocalLinkOutcome(t *testing.T) {
	root := t.TempDir()
	body := "# Heading\n\n**Bold** and [Target](target.md#part).\n\n<script>alert(1)</script>\n"
	source := "---\ntype: recipe\ntitle: Source facts\ndescription: Arbitrary vocabulary\nextra:\n  count: 0\n  enabled: false\n  empty: null\n  tags: [alpha, β]\n  markup: '<script>not code</script>'\n---\n" + body
	path := filepath.Join(root, ".okf", "root.md")
	writeOKFHTTPFixture(t, path, source)
	writeOKFHTTPFixture(t, filepath.Join(root, ".okf", "target.md"), "---\ntype: note\ntitle: Target\n---\n# Part\n")
	server, err := NewServer(fixtureModel(t), ServerOptions{SourceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	host := httptest.NewServer(server.Handler())
	defer host.Close()
	selection := postOKFHTTP(t, host.URL+"/v1/okf/sessions/default/bundle", http.MethodPut, map[string]any{"bundle_id": ".okf"})
	if selection.status != http.StatusOK {
		t.Fatalf("selection: %d", selection.status)
	}
	response := getOKFHTTP(t, host.URL+"/v1/okf/sessions/default/concept-detail?concept_id=root")
	var envelope struct {
		Data domain.ConceptDetail `json:"data"`
	}
	decodeOKFHTTP(t, response, &envelope)
	detail := envelope.Data
	if response.status != http.StatusOK || detail.BundleID != ".okf" || detail.SourceRevision == "" || detail.ConceptID != "root" {
		t.Fatalf("detail identity: %d %+v", response.status, detail)
	}
	if detail.Overview["type"] != "recipe" || detail.Overview["title"] != "Source facts" || detail.Overview["description"] != "Arbitrary vocabulary" {
		t.Fatalf("generic overview lost: %+v", detail.Overview)
	}
	want := map[string]any{"count": float64(0), "enabled": false, "empty": nil, "tags": []any{"alpha", "β"}, "markup": "<script>not code</script>"}
	if !reflect.DeepEqual(detail.MappedMetadata["extra"], want) {
		t.Fatalf("unknown values changed: %#v", detail.MappedMetadata)
	}
	if detail.RawMarkdown != body || strings.Contains(detail.RenderedMarkdown.Content, "<script") || !strings.Contains(detail.RenderedMarkdown.Content, "<strong>Bold</strong>") {
		t.Fatalf("raw/rendered Markdown mismatch: %+v", detail)
	}
	if len(detail.SemanticLinks) != 1 || detail.SemanticLinks[0].TargetID != "target" || detail.SemanticLinks[0].Fragment != "part" || !detail.SemanticLinks[0].Resolved || !detail.SemanticLinks[0].Safe {
		t.Fatalf("local link outcome: %+v", detail.SemanticLinks)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != source {
		t.Fatalf("inspection modified source: %v", err)
	}
}
