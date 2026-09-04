package viewer

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/buffo/arch-view/internal/okf/application"
	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/profile"
)

type manyDiagnosticProvider struct{ httpDiagnosticProvider }

func (manyDiagnosticProvider) Diagnose(context.Context, domain.BundleIndex) ([]domain.Diagnostic, error) {
	values := make([]domain.Diagnostic, 251)
	for i := range values {
		values[i] = domain.Diagnostic{Code: "test.many", Severity: "info", Category: "many", Message: "Provider explanation"}
	}
	values[250].Category = "needle"
	values[250].Recovery = "Retained after filtering"
	return values, nil
}

func TestOKFHTTPBoundsDiagnosticsAfterFiltering(t *testing.T) {
	root := t.TempDir()
	writeOKFHTTPFixture(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: topic\n---\n")
	registry := profile.NewRegistry()
	if err := registry.RegisterDiagnosticProvider(manyDiagnosticProvider{}); err != nil {
		t.Fatal(err)
	}
	service := application.NewWithDependencies(root, nil, nil, nil, registry)
	server, err := NewServer(fixtureModel(t), ServerOptions{SourceRoot: root, OKFApplication: service})
	if err != nil {
		t.Fatal(err)
	}
	host := httptest.NewServer(server.Handler())
	defer host.Close()
	for _, filtered := range []bool{false, true} {
		url := host.URL + "/v1/okf/diagnostics"
		if filtered {
			url += "?category=needle"
		}
		response := getOKFHTTP(t, url)
		var envelope struct {
			Data        application.DiagnosticReport `json:"data"`
			Diagnostics []domain.Diagnostic          `json:"diagnostics"`
		}
		decodeOKFHTTP(t, response, &envelope)
		if response.status != 200 {
			t.Fatalf("query: %d %s", response.status, response.body)
		}
		values := envelope.Data.Diagnostics
		if filtered {
			if len(values) != 1 || values[0].Recovery != "Retained after filtering" {
				t.Fatalf("filter applied after truncation: %+v", values)
			}
		} else if len(values) != 200 || values[199].Code != "okf_diagnostics_truncated" || len(envelope.Diagnostics) != 200 {
			t.Fatalf("unbounded report: data=%d envelope=%d", len(values), len(envelope.Diagnostics))
		}
	}
}
