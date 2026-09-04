package application

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/profile"
)

type lifecycleDetailRenderer struct{}

func (lifecycleDetailRenderer) ID() string          { return "test.detail.lifecycle" }
func (lifecycleDetailRenderer) Version() string     { return "1" }
func (lifecycleDetailRenderer) Description() string { return "Lifecycle test renderer" }
func (lifecycleDetailRenderer) ParameterSchema() map[string]any {
	return map[string]any{"type": "object"}
}
func (lifecycleDetailRenderer) ValidateParameters(parameters map[string]any) error {
	if _, ok := parameters["heading"].(string); !ok {
		return fmt.Errorf("heading must be a string")
	}
	return nil
}
func (lifecycleDetailRenderer) Render(ctx context.Context, document domain.ConceptDocument, parameters map[string]any) (string, error) {
	return "# " + parameters["heading"].(string) + "\n\n" + document.Markdown, ctx.Err()
}

func TestDetailRendererSurvivesProfileLifecycleAndReload(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	sourcePath := filepath.Join(root, ".okf", "root.md")
	source := "---\ntype: topic\n---\nOriginal **content**\n"
	writeApplicationFile(t, sourcePath, source)
	newService := func() *Service {
		t.Helper()
		registry := profile.NewRegistry()
		if err := registry.RegisterDetailRenderer(lifecycleDetailRenderer{}); err != nil {
			t.Fatal(err)
		}
		return NewWithDependencies(root, nil, nil, nil, registry)
	}
	service := newService()
	draft := domain.Profile{Name: "Custom", Details: domain.DetailSettings{Renderer: &domain.DetailRendererSelection{ID: "test.detail.lifecycle", Version: "1", Parameters: map[string]any{"heading": "Custom heading"}}}}
	created, err := service.SaveProfileAs(ctx, draft, "", "custom", "", "create", nil)
	if err != nil {
		t.Fatal(err)
	}
	renamed, err := service.RenameProfile(ctx, "project:custom", "renamed", "Renamed", created.Revision, "rename", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Bind(ctx, ".okf", "project:renamed", renamed.Revision, "bind", nil); err != nil {
		t.Fatal(err)
	}
	for _, current := range []*Service{service, newService()} {
		if _, err := current.Session(ctx, ""); err != nil {
			t.Fatal(err)
		}
		detail, err := current.Detail(ctx, "", "root")
		if err != nil || !strings.Contains(detail.RenderedMarkdown.Content, "<h1>Custom heading</h1>") || !strings.Contains(detail.RawMarkdown, "Original **content**") {
			t.Fatalf("custom detail lost: %+v %v", detail, err)
		}
		if _, err := current.SelectProfile(ctx, "", profile.DefaultProfileID); err != nil {
			t.Fatal(err)
		}
		neutral, err := current.Detail(ctx, "", "root")
		if err != nil || strings.Contains(neutral.RenderedMarkdown.Content, "Custom heading") || !strings.Contains(neutral.RenderedMarkdown.Content, "<strong>content</strong>") {
			t.Fatalf("default changed: %+v %v", neutral, err)
		}
	}
	bytes, err := os.ReadFile(sourcePath)
	if err != nil || string(bytes) != source {
		t.Fatalf("source changed: %v", err)
	}
}
