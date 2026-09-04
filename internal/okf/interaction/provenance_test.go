package interaction

import (
	"context"
	"reflect"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/profile"
)

func TestDetailPreservesDiagnosticOwnershipAndHierarchyProvenance(t *testing.T) {
	index := domain.BundleIndex{BundleID: "bundle", ConceptOrder: []string{"root", "root/child", "unrelated"}, Documents: map[string]domain.ConceptDocument{
		"root":       {ConceptID: "root", SourcePath: "root.md", ExplicitChildren: []string{"root/child"}},
		"root/child": {ConceptID: "root/child", SourcePath: "root/child.md", ExplicitParents: []string{"root"}, Provenance: []domain.Provenance{{Source: "document", Path: "root/child.md"}}},
		"unrelated":  {ConceptID: "unrelated", SourcePath: "unrelated.md", ExplicitParents: []string{"unrelated"}},
	}}
	registry := profile.NewRegistry()
	effective, _ := registry.ResolveProfile(profile.DefaultProfileID)
	detail, err := Detail(context.Background(), index, effective, "root/child", registry)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Diagnostics) != 1 || detail.Diagnostics[0].ConceptID != "unrelated" {
		t.Fatalf("diagnostic ownership changed: %+v", detail.Diagnostics)
	}
	var sources []string
	for _, proof := range detail.Provenance {
		sources = append(sources, proof.Source)
	}
	if !reflect.DeepEqual(sources, []string{"document", "explicit_children", "explicit_parent"}) {
		t.Fatalf("provenance=%+v", detail.Provenance)
	}
	if len(index.Documents["root/child"].Provenance) != 1 {
		t.Fatal("detail mutated source provenance")
	}
	effective.Hierarchy.UseExplicit = false
	detail, err = Detail(context.Background(), index, effective, "root/child", registry)
	if err != nil || len(detail.Provenance) != 2 || detail.Provenance[1].Source != "filesystem_fallback" {
		t.Fatalf("fallback provenance=%+v, error=%v", detail.Provenance, err)
	}
}
