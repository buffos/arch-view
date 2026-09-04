package source

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestDiscoveryDistinguishesTimeoutAndCancellation(t *testing.T) {
	for _, timeout := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		code := "okf_operation_cancelled"
		if timeout {
			cancel()
			ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			code = "okf_operation_timeout"
		}
		cancel()
		_, diagnostics := NewFilesystemScanner().Scan(ctx, t.TempDir())
		if len(diagnostics) != 1 || diagnostics[0].Code != code {
			t.Fatalf("timeout=%v: diagnostics=%+v", timeout, diagnostics)
		}
	}
}

func TestCancelledIndexDoesNotReportMissingSourceAsInvalidBundle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewFilesystemScanner().Index(ctx, domain.BundleCandidate{BundleID: "test", Selectable: true, AbsolutePath: filepath.Join(t.TempDir(), "missing")})
	var failure *domain.Error
	if !errors.As(err, &failure) || failure.Code != "okf_operation_cancelled" {
		t.Fatalf("error=%v", err)
	}
}

type cancelAfterChecks struct {
	context.Context
	checks int
	done   chan struct{}
}

func (ctx *cancelAfterChecks) Done() <-chan struct{} {
	ctx.checks++
	if ctx.checks == 3 {
		close(ctx.done)
	}
	return ctx.done
}

func (ctx *cancelAfterChecks) Err() error {
	return context.Canceled
}

func TestLinkResolutionStopsWithinConceptOnCancellation(t *testing.T) {
	ctx := &cancelAfterChecks{Context: context.Background(), done: make(chan struct{})}
	index := domain.BundleIndex{
		ConceptOrder: []string{"one"},
		Documents: map[string]domain.ConceptDocument{"one": {
			ConceptID: "one", SourcePath: "one.md", Links: make([]domain.Link, 100),
		}},
	}
	var diagnostics []domain.Diagnostic
	resolveLinks(ctx, &index, &diagnostics)
	if ctx.checks != 3 {
		t.Fatalf("cancellation checks=%d, want cancellation between links", ctx.checks)
	}
	if len(diagnostics) > 1 {
		t.Fatalf("processed links after cancellation: %d diagnostics", len(diagnostics))
	}
}
