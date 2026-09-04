package hierarchy

import (
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestCycleExclusionPreservesBranchesOutsideTheCycle(t *testing.T) {
	index := domain.BundleIndex{ConceptOrder: []string{"a-child", "b-cycle", "c-cycle"}, Documents: map[string]domain.ConceptDocument{
		"a-child": {ConceptID: "a-child", ExplicitParents: []string{"b-cycle"}},
		"b-cycle": {ConceptID: "b-cycle", ExplicitParents: []string{"c-cycle"}},
		"c-cycle": {ConceptID: "c-cycle", ExplicitParents: []string{"b-cycle"}},
	}}
	relationships, diagnostics := Normalize(index, domain.HierarchySettings{UseExplicit: true})
	if len(relationships) != 1 || relationships[0].From != "b-cycle" || relationships[0].To != "a-child" {
		t.Fatalf("noncyclic branch was discarded: %#v", relationships)
	}
	if len(diagnostics) != 1 {
		t.Fatalf("expected one cycle diagnostic: %#v", diagnostics)
	}
}

func TestNormalizePrefersExplicitHierarchyAndSeparatesSemanticLinks(t *testing.T) {
	index := domain.BundleIndex{BundleID: "bundle/.okf", ConceptOrder: []string{"a", "b", "c"}, Documents: map[string]domain.ConceptDocument{
		"a": {ConceptID: "a", SourcePath: "a.md", ExplicitChildren: []string{"b"}},
		"b": {ConceptID: "b", SourcePath: "nested/b.md", ExplicitParents: []string{"a"}},
		"c": {ConceptID: "c", SourcePath: "nested/c.md"},
	}}
	relationships, diagnostics := Normalize(index, domain.HierarchySettings{UseExplicit: true, UseFilesystem: true})
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
	if len(relationships) != 1 || relationships[0].From != "a" || relationships[0].To != "b" || relationships[0].Provenance[0].Source != "explicit_children" {
		t.Fatalf("relationships = %#v", relationships)
	}
	if len(relationships[0].Provenance) != 2 || relationships[0].Provenance[1].Source != "explicit_parent" {
		t.Fatalf("agreeing claim provenance was lost: %#v", relationships[0].Provenance)
	}
}

func TestNormalizeExcludesAmbiguousParentsAndCycles(t *testing.T) {
	index := domain.BundleIndex{BundleID: "bundle/.okf", ConceptOrder: []string{"a", "b", "c"}, Documents: map[string]domain.ConceptDocument{
		"a": {ConceptID: "a", SourcePath: "a.md", ExplicitParents: []string{"b"}},
		"b": {ConceptID: "b", SourcePath: "b.md", ExplicitParents: []string{"a"}},
		"c": {ConceptID: "c", SourcePath: "c.md", ExplicitParents: []string{"a", "b"}},
	}}
	relationships, diagnostics := Normalize(index, domain.HierarchySettings{UseExplicit: true})
	if len(relationships) != 0 {
		t.Fatalf("relationships = %#v", relationships)
	}
	if len(diagnostics) < 2 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
}

func TestNormalizeUsesFilesystemFallbackOnlyWhenExplicitParentIsAbsent(t *testing.T) {
	index := domain.BundleIndex{
		BundleID:     "knowledge/.okf",
		ConceptOrder: []string{"area", "area/topic", "area/topic/detail"},
		Documents: map[string]domain.ConceptDocument{
			"area":              {ConceptID: "area", SourcePath: "area.md"},
			"area/topic":        {ConceptID: "area/topic", SourcePath: "area/topic.md"},
			"area/topic/detail": {ConceptID: "area/topic/detail", SourcePath: "area/topic/detail.md", ExplicitParents: []string{"area"}},
		},
	}
	relationships, diagnostics := Normalize(index, domain.HierarchySettings{UseExplicit: true, UseFilesystem: true})
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
	if len(relationships) != 2 {
		t.Fatalf("relationships = %#v", relationships)
	}
	if relationships[0].From != "area" || relationships[0].To != "area/topic" || relationships[0].Provenance[0].Source != "filesystem_fallback" {
		t.Fatalf("fallback relationship = %#v", relationships[0])
	}
	if relationships[1].From != "area" || relationships[1].To != "area/topic/detail" || relationships[1].Provenance[0].Source != "explicit_parent" {
		t.Fatalf("explicit relationship = %#v", relationships[1])
	}
}

func TestNormalizeDoesNotLetAnExplicitChildClaimSuppressItsOwnParentFallback(t *testing.T) {
	index := domain.BundleIndex{
		BundleID:     "knowledge/.okf",
		ConceptOrder: []string{"area", "area/topic", "container"},
		Documents: map[string]domain.ConceptDocument{
			"container":  {ConceptID: "container", SourcePath: "container.md"},
			"area":       {ConceptID: "area", SourcePath: "container/area.md", ExplicitChildren: []string{"area/topic"}},
			"area/topic": {ConceptID: "area/topic", SourcePath: "container/area/topic.md"},
		},
	}
	relationships, diagnostics := Normalize(index, domain.HierarchySettings{UseExplicit: true, UseFilesystem: true})
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
	if len(relationships) != 2 {
		t.Fatalf("relationships = %#v", relationships)
	}
	if relationships[0].From != "container" || relationships[0].To != "area" || relationships[0].Provenance[0].Source != "filesystem_fallback" {
		t.Fatalf("area fallback relationship = %#v", relationships[0])
	}
	if relationships[1].From != "area" || relationships[1].To != "area/topic" || relationships[1].Provenance[0].Source != "explicit_children" {
		t.Fatalf("explicit child relationship = %#v", relationships[1])
	}
}
