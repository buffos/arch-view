package application

import (
	"github.com/buffo/arch-view/internal/okf/domain"
	"testing"
)

func TestDiagnosticQueryCombinesEveryFilter(t *testing.T) {
	value := domain.Diagnostic{BundleID: "bundle", ProfileID: "profile", ConceptID: "concept", RelationshipID: "edge", OperationID: "operation", Severity: "warning", Category: "source"}
	query := DiagnosticQuery{ProjectID: "project", BundleID: "bundle", ProfileID: "profile", ConceptID: "concept", RelationshipID: "edge", OperationID: "operation", Severity: "warning", Category: "source"}
	if !query.matches("project", value) {
		t.Fatal("matching intersection rejected")
	}
	for _, mismatch := range []DiagnosticQuery{
		{ProjectID: "other"}, {BundleID: "other"}, {ProfileID: "other"}, {ConceptID: "other"},
		{RelationshipID: "other"}, {OperationID: "other"}, {Severity: "error"}, {Category: "other"},
	} {
		if mismatch.matches("project", value) {
			t.Fatalf("mismatched filter accepted: %+v", mismatch)
		}
	}
	query.Category = "other"
	if query.matches("project", value) {
		t.Fatal("filters combined as union, not intersection")
	}
	if !(DiagnosticQuery{BundleID: "bundle"}).matches("project", domain.Diagnostic{}) {
		t.Fatal("bundle filter lost unscoped warning")
	}
	if (DiagnosticQuery{ProfileID: "profile"}).matches("project", domain.Diagnostic{}) {
		t.Fatal("profile filter matched unrelated unscoped warning")
	}
}
