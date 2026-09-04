package profile

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
)

type ownedPresentationFixture struct {
	testPresentationProvider
	annotations map[string]any
}

func (provider ownedPresentationFixture) Properties(context.Context, domain.ConceptDocument, map[string]any) (ports.PresentationProperties, error) {
	return ports.PresentationProperties{Annotations: provider.annotations}, nil
}

func TestProviderOutputDetachesTypedContainersWithoutRoundingNumbers(t *testing.T) {
	for _, kind := range []string{"presentation", "diagnostic", "rule"} {
		t.Run(kind, func(t *testing.T) {
			nested := map[string]string{"label": "original"}
			rows := []map[string]string{{"label": "original"}}
			values := map[string]any{"nested": nested, "rows": rows, "integer": int64(9007199254740993)}
			registry := NewRegistry()
			var captured map[string]any
			if kind == "presentation" {
				provider := ownedPresentationFixture{annotations: values}
				if err := registry.RegisterPresentationPropertyProvider(provider); err != nil {
					t.Fatal(err)
				}
				result, diagnostics := registry.Evaluate(context.Background(), domain.ConceptDocument{}, domain.Profile{Rules: []domain.RuleInvocation{{RuleID: provider.Metadata().ID, Version: "1", Enabled: true}}})
				if len(diagnostics) != 0 {
					t.Fatalf("provider failed: %+v", diagnostics)
				}
				captured = result.Annotations
			} else if kind == "rule" {
				rule := compositionRule{id: "test.owned.rule", result: ports.RuleResult{Annotations: values}}
				if err := registry.Register(rule); err != nil {
					t.Fatal(err)
				}
				result, diagnostics := registry.Evaluate(context.Background(), domain.ConceptDocument{}, domain.Profile{Rules: []domain.RuleInvocation{{RuleID: rule.ID(), Version: "1", Enabled: true}}})
				if len(diagnostics) != 0 {
					t.Fatalf("rule failed: %+v", diagnostics)
				}
				captured = result.Annotations
			} else {
				provider := diagnosticCallback{"test.ownership", func() ([]domain.Diagnostic, error) {
					return []domain.Diagnostic{{Code: "test.owned", Severity: "info", Category: "source", Message: "Owned result", Details: values}}, nil
				}}
				if err := registry.RegisterDiagnosticProvider(provider); err != nil {
					t.Fatal(err)
				}
				result, err := registry.Diagnose(context.Background(), domain.BundleIndex{})
				if err != nil || len(result) != 1 {
					t.Fatalf("provider failed: %+v %v", result, err)
				}
				captured = result[0].Details
			}
			before, err := json.Marshal(captured)
			if err != nil {
				t.Fatal(err)
			}
			nested["label"], rows[0]["label"] = "mutated", "mutated"
			after, err := json.Marshal(captured)
			if err != nil || string(before) != string(after) {
				t.Fatalf("provider mutated captured result: %s -> %s", before, after)
			}
			var number struct {
				Integer json.Number `json:"integer"`
			}
			if err := json.Unmarshal(after, &number); err != nil || number.Integer.String() != "9007199254740993" {
				t.Fatalf("integer precision lost: %s", after)
			}
		})
	}
}
