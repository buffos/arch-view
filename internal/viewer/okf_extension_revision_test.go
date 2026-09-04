package viewer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/buffo/arch-view/internal/okf/application"
	"github.com/buffo/arch-view/internal/okf/contract"
	"github.com/buffo/arch-view/internal/okf/domain"
)

type revisionHTTPAdapter struct{ version, schemaType string }

func (revisionHTTPAdapter) ID() string              { return "test.revision" }
func (adapter revisionHTTPAdapter) Version() string { return adapter.version }
func (adapter revisionHTTPAdapter) ParameterSchema() map[string]any {
	return map[string]any{"type": adapter.schemaType}
}
func (revisionHTTPAdapter) Relationships(context.Context, domain.BundleIndex) ([]domain.Relationship, []domain.Diagnostic) {
	return nil, nil
}

func TestOKFExtensionHTTPRevisionTracksInjectedMetadata(t *testing.T) {
	revision := func(adapter revisionHTTPAdapter) string {
		root := t.TempDir()
		api := application.NewWithDependencies(root, nil, nil, nil, nil, adapter)
		server, err := NewServer(fixtureModel(t), ServerOptions{SourceRoot: root, OKFApplication: api})
		if err != nil {
			t.Fatal(err)
		}
		host := httptest.NewServer(server.Handler())
		defer host.Close()
		response := getOKFHTTP(t, host.URL+"/v1/okf/extensions")
		var envelope struct {
			Meta contract.Meta `json:"meta"`
		}
		decodeOKFHTTP(t, response, &envelope)
		if response.status != http.StatusOK || envelope.Meta.Revision == "" {
			t.Fatalf("catalog=%d %s", response.status, response.body)
		}
		return envelope.Meta.Revision
	}
	first := revision(revisionHTTPAdapter{"1", "object"})
	if first != revision(revisionHTTPAdapter{"1", "object"}) {
		t.Fatal("identical catalogs have different revisions")
	}
	if first == revision(revisionHTTPAdapter{"2", "object"}) {
		t.Fatal("provider version change did not change revision")
	}
	if first == revision(revisionHTTPAdapter{"1", "string"}) {
		t.Fatal("provider schema change did not change revision")
	}
}
