package profile

import (
	"context"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
)

type testDiagnosticProvider struct{ mode string }

func (provider testDiagnosticProvider) Metadata() ports.Extension {
	return ports.Extension{ID: "test.diagnostic." + provider.mode, Version: "1", Description: "Test diagnostics", Capabilities: []string{"source-check"}, DefinitionSchema: map[string]any{"type": "array"}}
}
func (provider testDiagnosticProvider) Diagnose(_ context.Context, index domain.BundleIndex) ([]domain.Diagnostic, error) {
	index.Documents["root"].Frontmatter["original"] = "mutated"
	if provider.mode == "panic" {
		panic("provider panic")
	}
	if provider.mode == "invalid" {
		return []domain.Diagnostic{{Code: "invalid"}}, nil
	}
	return []domain.Diagnostic{{Code: "test.warning", Severity: "warning", Category: "source", Message: "Custom explanation", BundleID: "wrong", ConceptID: "root", Recovery: "Keep source intact"}}, nil
}

func TestDiagnosticProvidersAreIsolatedValidatedAndRetained(t *testing.T) {
	registry := NewRegistry()
	before, err := registry.Revision()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"valid", "panic", "invalid"} {
		if err := registry.RegisterDiagnosticProvider(testDiagnosticProvider{mode}); err != nil {
			t.Fatal(err)
		}
	}
	after, err := registry.Revision()
	if err != nil || before == after {
		t.Fatal("registry revision ignores diagnostic providers")
	}
	if err := registry.RegisterDiagnosticProvider(testDiagnosticProvider{"valid"}); err == nil {
		t.Fatal("duplicate accepted")
	}
	if err := registry.RegisterDiagnosticProvider(nil); err == nil {
		t.Fatal("nil provider accepted")
	}
	index := domain.BundleIndex{BundleID: "bundle", Documents: map[string]domain.ConceptDocument{"root": {Frontmatter: map[string]any{"original": "retained"}}}}
	for _, candidate := range []*Registry{registry, registry.WithProjectProfiles(nil)} {
		values, err := candidate.Diagnose(context.Background(), index)
		if err != nil || len(values) != 3 {
			t.Fatalf("diagnostics: %+v %v", values, err)
		}
		for i, value := range values {
			if value.BundleID != "bundle" || value.Recovery == "" {
				t.Fatalf("unscoped explanation: %+v", value)
			}
			if i < 2 && value.Code != "okf_diagnostic_provider_failed" {
				t.Fatalf("invalid provider output accepted: %+v", value)
			}
		}
		if values[2].Code != "test.warning" || index.Documents["root"].Frontmatter["original"] != "retained" {
			t.Fatal("provider changed original source or ordering")
		}
	}
	catalog := registry.DiagnosticProviderCatalog()
	catalog[0].DefinitionSchema["type"] = "mutated"
	if registry.DiagnosticProviderCatalog()[0].DefinitionSchema["type"] != "array" {
		t.Fatal("catalog aliases provider metadata")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if values, err := registry.Diagnose(ctx, index); err != context.Canceled || values != nil {
		t.Fatalf("cancelled provider query: %v %v", values, err)
	}
}
