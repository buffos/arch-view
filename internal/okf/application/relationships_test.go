package application

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
	"github.com/buffo/arch-view/internal/okf/profile"
	"github.com/buffo/arch-view/internal/okf/source"
)

type failingRelationshipAdapter struct {
	relationshipTestAdapter
	cancel context.CancelFunc
}

type metadataFailureAdapter struct {
	relationshipTestAdapter
	failID bool
}

func (adapter metadataFailureAdapter) ID() string {
	if adapter.failID {
		panic("ID failure")
	}
	return "metadata-test"
}

func (metadataFailureAdapter) Version() string { panic("version failure") }

func TestRelationshipAdapterMetadataPanicIsIsolated(t *testing.T) {
	for _, failID := range []bool{true, false} {
		index := domain.BundleIndex{BundleID: "bundle", Documents: map[string]domain.ConceptDocument{
			"root": {ConceptID: "root"}, "child": {ConceptID: "child"},
		}}
		result, diagnostics, err := applyRelationshipAdapters(context.Background(), index, []ports.RelationshipAdapter{
			metadataFailureAdapter{failID: failID}, relationshipTestAdapter{},
		})
		if err != nil || len(result.Relationships) != 1 || len(diagnostics) != 2 {
			t.Fatalf("metadata panic poisoned later adapter: %+v %+v %v", result, diagnostics, err)
		}
		wantID := "metadata-test"
		if failID {
			wantID = "anonymous"
		}
		if diagnostics[0].Code != "okf_relationship_adapter_failed" || diagnostics[0].Details["adapter_id"] != wantID {
			t.Fatalf("incorrect metadata failure diagnostic: %+v", diagnostics[0])
		}
	}
}

func (adapter failingRelationshipAdapter) Relationships(context.Context, domain.BundleIndex) ([]domain.Relationship, []domain.Diagnostic) {
	if adapter.cancel != nil {
		adapter.cancel()
		return nil, nil
	}
	panic("adapter failure")
}

func TestRelationshipAdapterPanicAndEmptyCancellation(t *testing.T) {
	index := domain.BundleIndex{BundleID: "bundle", Documents: map[string]domain.ConceptDocument{
		"root": {ConceptID: "root", Title: "Root"}, "child": {ConceptID: "child"},
	}}
	result, diagnostics, err := applyRelationshipAdapters(context.Background(), index, []ports.RelationshipAdapter{failingRelationshipAdapter{}, relationshipTestAdapter{}})
	if err != nil || len(result.Relationships) != 1 || len(diagnostics) != 2 {
		t.Fatalf("failed adapter poisoned later adapter: %+v %+v %v", result, diagnostics, err)
	}
	if diagnostics[0].Code != "okf_relationship_adapter_failed" || diagnostics[0].Details["adapter_id"] != "test" {
		t.Fatalf("missing adapter-scoped failure: %+v", diagnostics)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, _, err = applyRelationshipAdapters(ctx, index, []ports.RelationshipAdapter{failingRelationshipAdapter{cancel: cancel}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("empty result swallowed cancellation: %v", err)
	}
}

func TestRelationshipAdaptersAddOnlyValidatedSemanticLinks(t *testing.T) {
	root := t.TempDir()
	writeApplicationFile(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: area\ntitle: Root\n---\n")
	writeApplicationFile(t, filepath.Join(root, ".okf", "child.md"), "---\ntype: concept\ntitle: Child\n---\n")
	scanner := source.NewFilesystemScanner()
	adapter := relationshipTestAdapter{}
	service := NewWithDependencies(root, scanner, scanner, nil, profile.NewRegistry(), adapter)
	catalog, err := service.Refresh(context.Background())
	if err != nil || !catalog.Bundles[0].Selectable {
		t.Fatalf("catalog = %#v err=%v", catalog, err)
	}
	summary, err := service.Summary(context.Background(), ".okf")
	if err != nil || summary.LinkCount != 1 {
		t.Fatalf("summary = %#v err=%v", summary, err)
	}
	index := service.indexes[".okf"]
	if index.Documents["root"].Title != "Root" {
		t.Fatalf("adapter mutated the indexed source document: %#v", index.Documents["root"])
	}
	if index.Relationships[0].Provenance[len(index.Relationships[0].Provenance)-1].Source != "relationship_adapter" {
		t.Fatalf("adapter provenance = %#v", index.Relationships[0].Provenance)
	}
	if len(catalog.Bundles[0].Diagnostics) == 0 {
		t.Fatal("invalid adapter relationship should remain diagnosable")
	}
}

type relationshipTestAdapter struct{}

var _ ports.RelationshipAdapter = relationshipTestAdapter{}

func (relationshipTestAdapter) ID() string      { return "test" }
func (relationshipTestAdapter) Version() string { return "1" }

func (relationshipTestAdapter) Relationships(_ context.Context, index domain.BundleIndex) ([]domain.Relationship, []domain.Diagnostic) {
	index.Documents["root"] = domain.ConceptDocument{Title: "mutated"}
	return []domain.Relationship{
		{Kind: domain.RelationshipSemantic, From: "root", To: "child"},
		{Kind: domain.RelationshipContainment, From: "root", To: "child"},
	}, []domain.Diagnostic{}
}
