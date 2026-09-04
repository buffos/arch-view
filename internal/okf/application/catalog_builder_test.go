package application

import (
	"context"
	"reflect"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

type catalogScanFunc func(context.Context, string) ([]domain.BundleCandidate, []domain.Diagnostic)

func (scan catalogScanFunc) Scan(ctx context.Context, root string) ([]domain.BundleCandidate, []domain.Diagnostic) {
	return scan(ctx, root)
}

type catalogIndexFunc func(context.Context, domain.BundleCandidate) (domain.BundleIndex, error)

func (index catalogIndexFunc) Index(ctx context.Context, candidate domain.BundleCandidate) (domain.BundleIndex, error) {
	return index(ctx, candidate)
}

func TestCatalogBuilderIsolatesIndexFailureWithoutConfigurationOrSessions(t *testing.T) {
	var indexed []string
	builder := catalogBuilder{
		scanner: catalogScanFunc(func(_ context.Context, root string) ([]domain.BundleCandidate, []domain.Diagnostic) {
			if root != "project" {
				t.Fatalf("unexpected root %q", root)
			}
			return []domain.BundleCandidate{
				{BundleID: "invalid", Selectable: false, Status: domain.BundleInvalid},
				{BundleID: "failed", Selectable: true},
				{BundleID: "valid", Selectable: true},
			}, []domain.Diagnostic{{Code: "scan-notice"}}
		}),
		indexer: catalogIndexFunc(func(_ context.Context, candidate domain.BundleCandidate) (domain.BundleIndex, error) {
			indexed = append(indexed, candidate.BundleID)
			if candidate.BundleID == "failed" {
				return domain.BundleIndex{}, domain.NewError("index-failed", 422, "cannot index", nil)
			}
			return domain.BundleIndex{BundleID: candidate.BundleID, SourceRevision: "source-1", ConceptOrder: []string{"root"}}, nil
		}),
	}
	result, err := builder.build(context.Background(), "project")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(indexed, []string{"failed", "valid"}) || len(result.indexes) != 1 {
		t.Fatalf("incorrect isolation: indexed=%v indexes=%v", indexed, result.indexes)
	}
	if result.candidates[1].Selectable || result.candidates[1].Status != domain.BundleInvalid || result.candidates[1].Diagnostics[0].Code != "index-failed" {
		t.Fatalf("failed candidate: %+v", result.candidates[1])
	}
	if result.candidates[2].ConceptCount != 1 || result.candidates[2].SourceRevision != "source-1" || result.indexes["valid"].BundleID != "valid" {
		t.Fatalf("valid candidate: %+v", result.candidates[2])
	}
	if len(result.diagnostics) != 1 || result.diagnostics[0].Code != "scan-notice" {
		t.Fatalf("scan diagnostics: %+v", result.diagnostics)
	}
}

func TestCatalogBuilderDoesNotPoisonScannerCandidatesAfterIndexFailure(t *testing.T) {
	candidates := []domain.BundleCandidate{{BundleID: "bundle", Selectable: true}}
	diagnostics := []domain.Diagnostic{{Code: "scanner-note", Details: map[string]any{"source": "scanner"}}}
	attempts := 0
	builder := catalogBuilder{
		scanner: catalogScanFunc(func(context.Context, string) ([]domain.BundleCandidate, []domain.Diagnostic) {
			return candidates, diagnostics
		}),
		indexer: catalogIndexFunc(func(context.Context, domain.BundleCandidate) (domain.BundleIndex, error) {
			attempts++
			if attempts == 1 {
				return domain.BundleIndex{}, domain.NewError("index-failed", 500, "temporary failure", nil)
			}
			return domain.BundleIndex{BundleID: "bundle", ConceptOrder: []string{"root"}}, nil
		}),
	}
	first, err := builder.build(context.Background(), "project")
	if err != nil || first.candidates[0].Selectable {
		t.Fatalf("initial failure not represented: %+v %v", first, err)
	}
	second, err := builder.build(context.Background(), "project")
	if err != nil || attempts != 2 || !second.candidates[0].Selectable || len(second.indexes) != 1 {
		t.Fatalf("scanner candidate was poisoned: attempts=%d result=%+v err=%v", attempts, second, err)
	}
	if !candidates[0].Selectable || candidates[0].ConceptCount != 0 || len(candidates[0].Diagnostics) != 0 {
		t.Fatalf("scanner-owned candidate changed: %+v", candidates[0])
	}
	first.diagnostics[0].Details["source"] = "consumer"
	if diagnostics[0].Details["source"] != "scanner" {
		t.Fatal("result diagnostics alias scanner-owned data")
	}
}
