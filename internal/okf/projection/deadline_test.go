package projection

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/profile"
)

func TestBuildDistinguishesDeadlineFromCancellation(t *testing.T) {
	for _, timeout := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		cause, code, status := context.Canceled, "okf_operation_cancelled", 409
		if timeout {
			cancel()
			ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			cause, code, status = context.DeadlineExceeded, "okf_operation_timeout", 504
		}
		cancel()
		registry := profile.NewRegistry()
		effective, _ := registry.ResolveProfile(profile.DefaultProfileID)
		snapshot, err := Build(ctx, domain.BundleIndex{}, effective, domain.NavigationState{Depth: 1}, registry)
		var failure *domain.Error
		if !errors.As(err, &failure) || failure.Code != code || failure.Status != status || !errors.Is(err, cause) {
			t.Fatalf("timeout=%v: error=%+v", timeout, err)
		}
		if snapshot.Status != "" {
			t.Fatal("interrupted projection returned a publishable snapshot")
		}
	}
}
