package profile

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
)

type diagnosticCallback struct {
	id   string
	call func() ([]domain.Diagnostic, error)
}

func (provider diagnosticCallback) Metadata() ports.Extension {
	return ports.Extension{ID: provider.id, Version: "1", Description: "Failure fixture", Capabilities: []string{"diagnostic"}, DefinitionSchema: map[string]any{"type": "array"}}
}
func (provider diagnosticCallback) Diagnose(context.Context, domain.BundleIndex) ([]domain.Diagnostic, error) {
	return provider.call()
}

func TestDiagnosticFailuresDiscardPartialOutputAndContinue(t *testing.T) {
	valid := domain.Diagnostic{Code: "test.valid", Severity: "info", Category: "source", Message: "Valid explanation"}
	for name, broken := range map[string]domain.Diagnostic{
		"missing-code":     {Severity: "info", Category: "source", Message: "Incomplete"},
		"invalid-severity": {Code: "test.bad", Severity: "fatal", Category: "source", Message: "Invalid"},
		"unencodable":      {Code: "test.bad", Severity: "error", Category: "source", Message: "Invalid", Details: map[string]any{"number": math.NaN()}},
	} {
		t.Run(name, func(t *testing.T) {
			registry := NewRegistry()
			for _, provider := range []diagnosticCallback{
				{"test.a", func() ([]domain.Diagnostic, error) { return []domain.Diagnostic{valid, broken}, nil }},
				{"test.b", func() ([]domain.Diagnostic, error) {
					return []domain.Diagnostic{valid}, errors.New("failed after producing output")
				}},
				{"test.c", func() ([]domain.Diagnostic, error) { return []domain.Diagnostic{valid}, nil }},
			} {
				if err := registry.RegisterDiagnosticProvider(provider); err != nil {
					t.Fatal(err)
				}
			}
			values, err := registry.Diagnose(context.Background(), domain.BundleIndex{BundleID: "bundle"})
			if err != nil || len(values) != 3 {
				t.Fatalf("report: %+v %v", values, err)
			}
			for i, id := range []string{"test.a", "test.b"} {
				if values[i].Code != "okf_diagnostic_provider_failed" || values[i].Details["provider_id"] != id || values[i].Recovery == "" || values[i].BundleID != "bundle" {
					t.Fatalf("failure not isolated: %+v", values[i])
				}
			}
			if values[2].Code != valid.Code {
				t.Fatal("healthy provider was skipped")
			}
		})
	}
}

func TestDiagnosticCancellationAfterCallbackDiscardsResults(t *testing.T) {
	registry := NewRegistry()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	laterCalls := 0
	for _, provider := range []diagnosticCallback{
		{"test.a", func() ([]domain.Diagnostic, error) { cancel(); return nil, nil }},
		{"test.b", func() ([]domain.Diagnostic, error) { laterCalls++; return nil, nil }},
	} {
		if err := registry.RegisterDiagnosticProvider(provider); err != nil {
			t.Fatal(err)
		}
	}
	if values, err := registry.Diagnose(ctx, domain.BundleIndex{}); !errors.Is(err, context.Canceled) || values != nil || laterCalls != 0 {
		t.Fatalf("cancelled report leaked results: %+v %v calls=%d", values, err, laterCalls)
	}
	if _, err := registry.Diagnose(context.Background(), domain.BundleIndex{}); err != nil || laterCalls != 1 {
		t.Fatal("cancelled query poisoned later request")
	}
}
