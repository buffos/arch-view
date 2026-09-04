package application

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDiagnosticQueryUsesPublishedSourceAndPreservesSession(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	path := filepath.Join(root, ".okf", "root.md")
	writeApplicationFile(t, path, "---\ntype: topic\ntitle: Original\n---\n[escape](../outside.md)\n")
	writeApplicationFile(t, filepath.Join(root, "invalid", ".okf", "bad.md"), "missing frontmatter")
	service := New(root)
	before, err := service.Session(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	writeApplicationFile(t, path, "---\ntype: topic\ntitle: Changed on disk\n---\n")
	report, err := service.Diagnostics(ctx, DiagnosticQuery{BundleID: ".okf"})
	if err != nil || report.Revision == "" {
		t.Fatalf("diagnostics: %+v %v", report, err)
	}
	found := false
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.BundleID != "" && diagnostic.BundleID != ".okf" {
			t.Fatalf("bundle filter leaked: %+v", diagnostic)
		}
		found = found || diagnostic.Code == "okf_bundle_boundary_violation"
	}
	if !found {
		t.Fatal("published warning was replaced by fresh disk content")
	}
	after, err := service.Session(ctx, "")
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("diagnostics changed session: %v", err)
	}
	all, err := service.Diagnostics(ctx, DiagnosticQuery{})
	if err != nil || len(all.Diagnostics) <= len(report.Diagnostics) {
		t.Fatalf("unfiltered diagnostics missing: %+v %v", all, err)
	}
	all.Diagnostics[0].Message = "caller mutation"
	again, err := service.Diagnostics(ctx, DiagnosticQuery{})
	if err != nil || again.Diagnostics[0].Message == "caller mutation" {
		t.Fatal("report aliases published diagnostics")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := service.Diagnostics(cancelled, DiagnosticQuery{}); !hasApplicationCode(err, "okf_operation_cancelled") {
		t.Fatalf("cancelled query: %v", err)
	}
}
