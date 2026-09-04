package viewer

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/okf/application"
	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/profile"
)

type httpDetailRenderer struct{}

func (httpDetailRenderer) ID() string                      { return "test.detail.http" }
func (httpDetailRenderer) Version() string                 { return "1" }
func (httpDetailRenderer) Description() string             { return "HTTP detail test" }
func (httpDetailRenderer) ParameterSchema() map[string]any { return map[string]any{"type": "object"} }
func (httpDetailRenderer) ValidateParameters(parameters map[string]any) error {
	if _, ok := parameters["mode"].(string); !ok {
		return fmt.Errorf("mode is required")
	}
	return nil
}
func (httpDetailRenderer) Render(_ context.Context, document domain.ConceptDocument, parameters map[string]any) (string, error) {
	document.Frontmatter["custom"] = "changed"
	if parameters["mode"] == "panic" {
		panic("test provider failure")
	}
	return "# Custom view\n\n<script>alert(1)</script>\n\n[target](/target.md) [unsafe](javascript:alert) [escape](../outside.md)\n", nil
}

func TestOKFHTTPDetailRendererSafetyAndFailureFallback(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, ".okf", "root.md")
	source := "---\ntype: topic\ncustom: retained\n---\nOriginal **content**\n"
	writeOKFHTTPFixture(t, sourcePath, source)
	writeOKFHTTPFixture(t, filepath.Join(root, ".okf", "target.md"), "---\ntype: topic\n---\n")
	registry := profile.NewRegistry()
	if err := registry.RegisterDetailRenderer(httpDetailRenderer{}); err != nil {
		t.Fatal(err)
	}
	service := application.NewWithDependencies(root, nil, nil, nil, registry)
	server, err := NewServer(fixtureModel(t), ServerOptions{SourceRoot: root, OKFApplication: service})
	if err != nil {
		t.Fatal(err)
	}
	host := httptest.NewServer(server.Handler())
	defer host.Close()
	for _, mode := range []string{"custom", "panic"} {
		value := domain.Profile{Name: mode, Details: domain.DetailSettings{Renderer: &domain.DetailRendererSelection{ID: "test.detail.http", Version: "1", Parameters: map[string]any{"mode": mode}}}}
		created := postOKFHTTP(t, host.URL+"/v1/okf/profiles/save-as", http.MethodPost, map[string]any{"profile": value, "new_profile_id": mode})
		if created.status != http.StatusCreated {
			t.Fatalf("create: %d %s", created.status, created.body)
		}
		selected := postOKFHTTP(t, host.URL+"/v1/okf/sessions/default/profile", http.MethodPut, map[string]any{"profile_id": "project:" + mode})
		if selected.status != http.StatusOK {
			t.Fatalf("select: %d %s", selected.status, selected.body)
		}
		response := getOKFHTTP(t, host.URL+"/v1/okf/sessions/default/concept-detail?concept_id=root")
		var envelope struct {
			Data domain.ConceptDetail `json:"data"`
			Meta map[string]any       `json:"meta"`
		}
		decodeOKFHTTP(t, response, &envelope)
		detail := envelope.Data
		if response.status != http.StatusOK || envelope.Meta["request_id"] == nil || detail.Frontmatter["custom"] != "retained" || !strings.Contains(detail.RawMarkdown, "Original **content**") {
			t.Fatalf("detail: %d %+v", response.status, envelope)
		}
		content := detail.RenderedMarkdown.Content
		if strings.Contains(content, "<script") || strings.Contains(content, "href=\"javascript:") || strings.Contains(content, "href=\"../outside.md") {
			t.Fatalf("unsafe custom output: %s", content)
		}
		if mode == "custom" {
			if !strings.Contains(content, "<h1>Custom view</h1>") {
				t.Fatalf("renderer not applied: %s", content)
			}
		} else {
			found := false
			for _, diagnostic := range detail.Diagnostics {
				if diagnostic.Code == "okf_detail_renderer_failed" && (diagnostic.BundleID != ".okf" || diagnostic.ConceptID != "root" || diagnostic.ProfileID != "project:panic") {
					t.Fatalf("unscoped renderer failure: %+v", diagnostic)
				}
				found = found || diagnostic.Code == "okf_detail_renderer_failed"
			}
			if !found || !strings.Contains(content, "Original <strong>content</strong>") {
				t.Fatalf("missing safe fallback: %+v", detail)
			}
		}
	}
	configPath := filepath.Join(root, ".archview.json")
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	invalid := domain.Profile{Name: "Invalid", Details: domain.DetailSettings{Renderer: &domain.DetailRendererSelection{ID: "test.detail.http", Version: "missing"}}}
	rejected := postOKFHTTP(t, host.URL+"/v1/okf/profiles/save-as", http.MethodPost, map[string]any{"profile": invalid, "new_profile_id": "invalid"})
	if rejected.status < 400 || !strings.Contains(string(rejected.body), "okf_detail_renderer_invalid") {
		t.Fatalf("unsupported renderer accepted: %d %s", rejected.status, rejected.body)
	}
	after, err := os.ReadFile(configPath)
	if err != nil || string(before) != string(after) {
		t.Fatalf("rejected save changed configuration: %v", err)
	}
	bytes, err := os.ReadFile(sourcePath)
	if err != nil || string(bytes) != source {
		t.Fatalf("source changed: %v", err)
	}
}
