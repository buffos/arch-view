package viewer

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDiagnosticReadPreservesPublishedSession(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, ".okf", "root.md")
	writeOKFHTTPFixture(t, file, "---\ntype: concept\ntitle: Before\n---\n[outside](../outside.md)")
	writeOKFHTTPFixture(t, filepath.Join(root, "invalid", ".okf", "bad.md"), "missing frontmatter")
	server, err := NewServer(fixtureModel(t), ServerOptions{SourceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	before, err := server.okf.Session(context.Background(), "default")
	if err != nil {
		t.Fatal(err)
	}
	writeOKFHTTPFixture(t, file, "---\ntype: concept\ntitle: After\n---\n")
	response := getOKFHTTP(t, httpServer.URL+"/v1/okf/diagnostics")
	if response.status != 200 || !strings.Contains(string(response.body), "okf_bundle_boundary_violation") || !strings.Contains(string(response.body), "invalid/.okf") {
		t.Fatalf("missing published diagnostics: %d %s", response.status, response.body)
	}
	after, err := server.okf.Session(context.Background(), "default")
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("diagnostic read changed session: %v", err)
	}
}
