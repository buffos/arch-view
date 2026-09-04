package viewer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/buffo/arch-view/internal/okf/application"
	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
	"github.com/buffo/arch-view/internal/okf/profile"
)

type httpPresentationProvider struct{}

func (httpPresentationProvider) Metadata() ports.Extension {
	return ports.Extension{ID: "test.presentation.http", Version: "1", Description: "Custom display", Capabilities: []string{"label", "token", "shape"}, ParameterSchema: map[string]any{"type": "object"}}
}
func (httpPresentationProvider) ValidateParameters(map[string]any) error { return nil }
func (httpPresentationProvider) Properties(context.Context, domain.ConceptDocument, map[string]any) (ports.PresentationProperties, error) {
	return ports.PresentationProperties{Label: "Provider title", Token: "accent", Shape: "ellipse", Annotations: map[string]any{"note": "Derived presentation"}}, nil
}

func TestOKFHTTPPresentationProviderReachesProjection(t *testing.T) {
	root := t.TempDir()
	writeOKFHTTPFixture(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: topic\ntitle: Source title\n---\nOriginal content\n")
	registry := profile.NewRegistry()
	if err := registry.RegisterPresentationPropertyProvider(httpPresentationProvider{}); err != nil {
		t.Fatal(err)
	}
	service := application.NewWithDependencies(root, nil, nil, nil, registry)
	server, err := NewServer(fixtureModel(t), ServerOptions{SourceRoot: root, OKFApplication: service})
	if err != nil {
		t.Fatal(err)
	}
	host := httptest.NewServer(server.Handler())
	defer host.Close()
	value := domain.Profile{Name: "Properties", Rules: []domain.RuleInvocation{{RuleID: "test.presentation.http", Version: "1", Enabled: true}}, Style: domain.StyleSettings{Tokens: map[string]domain.StyleToken{"accent": {Fill: "#112233", Text: "#ffffff"}}}}
	created := postOKFHTTP(t, host.URL+"/v1/okf/profiles/save-as", http.MethodPost, map[string]any{"profile": value, "new_profile_id": "properties"})
	if created.status != http.StatusCreated {
		t.Fatalf("save: %d %s", created.status, created.body)
	}
	for _, id := range []string{"project:properties", "builtin:neutral"} {
		response := postOKFHTTP(t, host.URL+"/v1/okf/sessions/default/profile", http.MethodPut, map[string]any{"profile_id": id})
		var envelope struct {
			Data application.SessionView `json:"data"`
		}
		decodeOKFHTTP(t, response, &envelope)
		if response.status != http.StatusOK || envelope.Data.Projection == nil || len(envelope.Data.Projection.Nodes) != 1 {
			t.Fatalf("projection: %d %+v", response.status, envelope)
		}
		node := envelope.Data.Projection.Nodes[0]
		if node.ConceptID != "root" {
			t.Fatal("presentation changed source identity")
		}
		if id == "project:properties" {
			if node.Label != "Provider title" || node.PresentationStyle.Fill != "#112233" || node.ShapeDefinition == nil || node.ShapeDefinition.Geometry != "ellipse" || node.Annotations["note"] != "Derived presentation" {
				t.Fatalf("properties lost: %+v", node)
			}
		} else if node.Label != "Source title" || node.Annotations["note"] != nil {
			t.Fatalf("Neutral retained custom presentation: %+v", node)
		}
	}
	detail, err := service.Detail(context.Background(), "", "root")
	if err != nil || detail.Overview["title"] != "Source title" {
		t.Fatalf("provider changed source detail: %+v %v", detail, err)
	}
}
