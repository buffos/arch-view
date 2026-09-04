package viewer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/okf/application"
	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
	"github.com/buffo/arch-view/internal/okf/profile"
)

type httpDiagnosticProvider struct{}

func (httpDiagnosticProvider) Metadata() ports.Extension {
	return ports.Extension{ID: "test.diagnostic.http", Version: "1", Description: "HTTP diagnostic check", Capabilities: []string{"source-check"}, DefinitionSchema: map[string]any{"type": "array"}}
}
func (httpDiagnosticProvider) Diagnose(_ context.Context, index domain.BundleIndex) ([]domain.Diagnostic, error) {
	index.Documents["root"].Frontmatter["custom"] = "changed"
	return []domain.Diagnostic{{Code: "test.diagnostic", Severity: "info", Category: "custom", Message: "Provider explanation", ConceptID: "root", Recovery: "Review the concept", Details: map[string]any{"source_revision": index.SourceRevision}}}, nil
}

func TestOKFHTTPDiagnosticProviderSurvivesProfileSave(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".okf", "root.md")
	source := "---\ntype: topic\ncustom: retained\n---\n[escape](../outside.md)\n"
	writeOKFHTTPFixture(t, path, source)
	registry := profile.NewRegistry()
	if err := registry.RegisterDiagnosticProvider(httpDiagnosticProvider{}); err != nil {
		t.Fatal(err)
	}
	service := application.NewWithDependencies(root, nil, nil, nil, registry)
	server, err := NewServer(fixtureModel(t), ServerOptions{SourceRoot: root, OKFApplication: service})
	if err != nil {
		t.Fatal(err)
	}
	host := httptest.NewServer(server.Handler())
	defer host.Close()
	created := postOKFHTTP(t, host.URL+"/v1/okf/profiles/save-as", http.MethodPost, map[string]any{"profile": domain.Profile{Name: "Saved"}, "new_profile_id": "saved"})
	if created.status != http.StatusCreated {
		t.Fatalf("save: %d %s", created.status, created.body)
	}
	before, err := service.Session(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	response := getOKFHTTP(t, host.URL+"/v1/okf/diagnostics?bundle_id=.okf&concept_id=root&category=custom&severity=info")
	var envelope struct {
		Data application.DiagnosticReport `json:"data"`
	}
	decodeOKFHTTP(t, response, &envelope)
	if response.status != http.StatusOK || len(envelope.Data.Diagnostics) != 1 {
		t.Fatalf("provider lost or filter ignored: %d %+v", response.status, envelope)
	}
	value := envelope.Data.Diagnostics[0]
	if value.Code != "test.diagnostic" || value.BundleID != ".okf" || value.Recovery != "Review the concept" || value.Details["source_revision"] == "" {
		t.Fatalf("provider output lost: %+v", value)
	}
	all := getOKFHTTP(t, host.URL+"/v1/okf/diagnostics")
	if !strings.Contains(string(all.body), "okf_bundle_boundary_violation") {
		t.Fatal("provider replaced built-in diagnostics")
	}
	catalog := getOKFHTTP(t, host.URL+"/v1/okf/extensions")
	if catalog.status != http.StatusOK || !strings.Contains(string(catalog.body), "test.diagnostic.http") || !strings.Contains(string(catalog.body), "diagnostic_provider") {
		t.Fatal("provider metadata missing")
	}
	after, err := service.Session(context.Background(), "")
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("query changed session: %v", err)
	}
	detail, err := service.Detail(context.Background(), "", "root")
	if err != nil || detail.Frontmatter["custom"] != "retained" {
		t.Fatalf("provider changed indexed metadata: %+v %v", detail, err)
	}
	bytes, err := os.ReadFile(path)
	if err != nil || string(bytes) != source {
		t.Fatalf("provider changed source: %v", err)
	}
}
