package viewer

import (
	"context"
	"fmt"
	"testing"

	"github.com/buffo/arch-view/internal/okf/application"
	"github.com/buffo/arch-view/internal/okf/domain"
)

type rejectingOKFLayout struct{ calls int }

func (validator *rejectingOKFLayout) Validate(domain.LayoutSettings) error {
	validator.calls++
	return fmt.Errorf("test host rejects this layout")
}

func TestViewerPreservesInjectedApplicationDependencies(t *testing.T) {
	root := t.TempDir()
	service := application.New(root)
	validator := &rejectingOKFLayout{}
	service.SetLayoutValidator(validator)
	server, err := NewServer(fixtureModel(t), ServerOptions{SourceRoot: root, OKFApplication: service})
	if err != nil {
		t.Fatal(err)
	}
	_, diagnostics := server.okf.ValidateProfile(context.Background(), domain.Profile{ProfileID: "project:test"})
	if validator.calls == 0 || len(diagnostics) == 0 {
		t.Fatal("viewer replaced the injected validator")
	}
}
