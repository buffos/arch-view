package profile

import (
	"encoding/json"
	"github.com/buffo/arch-view/internal/okf/domain"
	"reflect"
	"testing"
)

func TestDetailRendererSelectionRoundTripAndComposition(t *testing.T) {
	var base domain.Profile
	if err := json.Unmarshal([]byte(`{"profile_id":"project:base","details":{"renderer":{"id":"okf.detail.commonmark","version":"1"}}}`), &base); err != nil {
		t.Fatal(err)
	}
	registry := NewRegistry()
	registry.SetProjectProfiles([]domain.Profile{base, {ProfileID: "project:child", Bases: []string{"project:base"}}})
	for _, candidate := range []*Registry{registry, registry.WithProjectProfiles([]domain.Profile{base})} {
		value, diagnostics := candidate.ResolveProfile("project:base")
		if value.Status == domain.ProfileInvalid || len(diagnostics) > 0 || value.Details.Renderer == nil {
			t.Fatalf("selection lost: %+v %v", value, diagnostics)
		}
	}
	child, diagnostics := registry.ResolveProfile("project:child")
	if child.Status == domain.ProfileInvalid || len(diagnostics) > 0 || !reflect.DeepEqual(child.Details.Renderer, base.Details.Renderer) {
		t.Fatalf("inheritance: %+v %v", child, diagnostics)
	}
	data, err := json.Marshal(child)
	if err != nil {
		t.Fatal(err)
	}
	var decoded domain.Profile
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(child.Details.Renderer, decoded.Details.Renderer) {
		t.Fatal("codec lost selection")
	}
	child.Details.Renderer.Parameters = map[string]any{"nested": map[string]any{"value": "original"}}
	copy := domain.CloneProfile(child)
	copy.Details.Renderer.Parameters["nested"].(map[string]any)["value"] = "changed"
	if child.Details.Renderer.Parameters["nested"].(map[string]any)["value"] != "original" {
		t.Fatal("clone shares renderer parameters")
	}
}

func TestDetailRendererInvalidSelectionAndCatalogOwnership(t *testing.T) {
	registry := NewRegistry()
	for _, selection := range []domain.DetailRendererSelection{
		{ID: "missing.renderer", Version: "1"},
		{ID: DefaultDetailRendererID, Version: "2"},
		{ID: DefaultDetailRendererID, Version: "1", Parameters: map[string]any{"unexpected": true}},
	} {
		value, _ := registry.ResolveCandidate(domain.Profile{ProfileID: "project:test", Details: domain.DetailSettings{Renderer: &selection}})
		if value.Status != domain.ProfileInvalid {
			t.Fatalf("invalid renderer accepted: %+v", selection)
		}
	}
	catalog := registry.DetailRendererCatalog()
	catalog[0].ParameterSchema["type"] = "changed"
	catalog[0].Capabilities[0] = "changed"
	if registry.DetailRendererCatalog()[0].ParameterSchema["type"] != "object" || registry.DetailRendererCatalog()[0].Capabilities[0] != "sanitized-commonmark" {
		t.Fatal("catalog shares metadata")
	}
	if err := registry.RegisterDetailRenderer(commonMarkRenderer{}); err == nil {
		t.Fatal("duplicate accepted")
	}
}
