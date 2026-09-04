package application

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
)

type pausedDiagnosticProvider struct {
	entered, release chan struct{}
	started          atomic.Bool
}

func (*pausedDiagnosticProvider) Metadata() ports.Extension {
	return ports.Extension{ID: "test.snapshot", Version: "1", Description: "Snapshot check", Capabilities: []string{"source"}, DefinitionSchema: map[string]any{"type": "array"}}
}

func (provider *pausedDiagnosticProvider) Diagnose(ctx context.Context, index domain.BundleIndex) ([]domain.Diagnostic, error) {
	if provider.started.CompareAndSwap(false, true) {
		close(provider.entered)
		select {
		case <-provider.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return []domain.Diagnostic{{Code: "test.snapshot", Severity: "info", Category: "source", Message: "Captured revision", Details: map[string]any{"revision": index.SourceRevision}}}, nil
}

func TestDiagnosticsRetainsSinglePublicationDuringRefresh(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	root := t.TempDir()
	for _, bundle := range []string{".okf", "second/.okf"} {
		writeApplicationFile(t, filepath.Join(root, bundle, "root.md"), "---\ntype: topic\ntitle: Original\n---\n")
	}
	service := New(root)
	before, err := service.Refresh(ctx)
	if err != nil {
		t.Fatal(err)
	}
	provider := &pausedDiagnosticProvider{entered: make(chan struct{}), release: make(chan struct{})}
	if err := service.registry.RegisterDiagnosticProvider(provider); err != nil {
		t.Fatal(err)
	}
	type result struct {
		report DiagnosticReport
		err    error
	}
	done := make(chan result, 1)
	go func() { report, err := service.Diagnostics(ctx, DiagnosticQuery{}); done <- result{report, err} }()
	select {
	case <-provider.entered:
	case <-ctx.Done():
		t.Fatal("report did not start")
	}
	writeApplicationFile(t, filepath.Join(root, "second/.okf/root.md"), "---\ntype: topic\ntitle: Refreshed\n---\n")
	after, err := service.Refresh(ctx)
	if err != nil || after.Revision == before.Revision {
		t.Fatalf("refresh: %v", err)
	}
	close(provider.release)
	select {
	case value := <-done:
		if value.err != nil || value.report.Revision != before.Revision {
			t.Fatalf("report: %+v", value)
		}
		expected := make(map[string]string)
		for _, bundle := range before.Bundles {
			expected[bundle.BundleID] = bundle.SourceRevision
		}
		seen := 0
		for _, diagnostic := range value.report.Diagnostics {
			if diagnostic.Code == "test.snapshot" {
				seen++
				if diagnostic.Details["revision"] != expected[diagnostic.BundleID] {
					t.Fatalf("mixed publications: %+v", diagnostic)
				}
			}
		}
		if seen != 2 {
			t.Fatalf("expected both captured bundles, got %d", seen)
		}
	case <-ctx.Done():
		t.Fatal("report did not finish")
	}
}
