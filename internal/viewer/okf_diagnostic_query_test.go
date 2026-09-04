package viewer

import (
	"context"
	"github.com/buffo/arch-view/internal/okf/application"
	"github.com/buffo/arch-view/internal/okf/domain"
	"net/http/httptest"
	"reflect"
	"testing"
)

type diagnosticQueryApplication struct {
	application.API
	query application.DiagnosticQuery
}

func (value *diagnosticQueryApplication) Diagnostics(_ context.Context, query application.DiagnosticQuery) (application.DiagnosticReport, error) {
	value.query = query
	return application.DiagnosticReport{Revision: "query-revision", Diagnostics: []domain.Diagnostic{{Code: "test", RelationshipID: "edge", Recovery: "Keep this guidance"}}}, nil
}

func TestOKFHTTPForwardsDiagnosticQueryAndPreservesReport(t *testing.T) {
	api := &diagnosticQueryApplication{}
	server, err := NewServer(fixtureModel(t), ServerOptions{OKFApplication: api})
	if err != nil {
		t.Fatal(err)
	}
	host := httptest.NewServer(server.Handler())
	defer host.Close()
	response := getOKFHTTP(t, host.URL+"/v1/okf/diagnostics?project_id=project&bundle_id=bundle&profile_id=profile&concept_id=concept&relationship_id=edge&operation_id=operation&severity=warning&category=source")
	want := application.DiagnosticQuery{ProjectID: "project", BundleID: "bundle", ProfileID: "profile", ConceptID: "concept", RelationshipID: "edge", OperationID: "operation", Severity: "warning", Category: "source"}
	if response.status != 200 || !reflect.DeepEqual(api.query, want) {
		t.Fatalf("query: %d %+v", response.status, api.query)
	}
	var envelope struct {
		Data application.DiagnosticReport `json:"data"`
		Meta map[string]any               `json:"meta"`
	}
	decodeOKFHTTP(t, response, &envelope)
	if envelope.Meta["revision"] != "query-revision" || len(envelope.Data.Diagnostics) != 1 || envelope.Data.Diagnostics[0].RelationshipID != "edge" || envelope.Data.Diagnostics[0].Recovery != "Keep this guidance" {
		t.Fatalf("report changed: %+v", envelope)
	}
}
