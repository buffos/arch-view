package interaction

import (
	"context"
	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/profile"
	"strings"
	"testing"
)

type testDetailRenderer struct{ fail bool }

func (testDetailRenderer) ID() string                              { return "test.detail" }
func (testDetailRenderer) Version() string                         { return "1" }
func (testDetailRenderer) Description() string                     { return "Test display" }
func (testDetailRenderer) ParameterSchema() map[string]any         { return map[string]any{"type": "object"} }
func (testDetailRenderer) ValidateParameters(map[string]any) error { return nil }
func (renderer testDetailRenderer) Render(_ context.Context, document domain.ConceptDocument, parameters map[string]any) (string, error) {
	document.Frontmatter["custom"] = "mutated"
	parameters["label"] = "mutated"
	if renderer.fail {
		panic("broken extension")
	}
	return "# Custom detail\n\n<script>alert(1)</script>\n\n[unsafe](javascript:alert)\n", nil
}

func TestCustomDetailIsSanitizedAndSourceRemainsUntouched(t *testing.T) {
	for _, fail := range []bool{false, true} {
		document := domain.ConceptDocument{ConceptID: "one", SourcePath: "one.md", Markdown: "Original **content**", Frontmatter: map[string]any{"custom": "retained"}}
		index := domain.BundleIndex{BundleID: ".okf", ConceptOrder: []string{"one"}, Documents: map[string]domain.ConceptDocument{"one": document}}
		registry := profile.NewRegistry()
		effective, _ := registry.ResolveProfile(profile.DefaultProfileID)
		effective.Details.Renderer = &domain.DetailRendererSelection{ID: "test.detail", Version: "1", Parameters: map[string]any{"label": "original"}}
		detail, err := DetailWithRenderer(context.Background(), index, effective, "one", registry, testDetailRenderer{fail: fail})
		if err != nil {
			t.Fatal(err)
		}
		if detail.RawMarkdown != document.Markdown || detail.Frontmatter["custom"] != "retained" || document.Frontmatter["custom"] != "retained" || effective.Details.Renderer.Parameters["label"] != "original" {
			t.Fatal("renderer mutated source or parameters")
		}
		if strings.Contains(detail.RenderedMarkdown.Content, "<script") || strings.Contains(detail.RenderedMarkdown.Content, "href=\"javascript:") {
			t.Fatalf("unsafe output: %s", detail.RenderedMarkdown.Content)
		}
		if fail {
			if !strings.Contains(detail.RenderedMarkdown.Content, "Original") {
				t.Fatal("missing original fallback")
			}
			found := false
			for _, diagnostic := range detail.Diagnostics {
				found = found || diagnostic.Code == "okf_detail_renderer_failed"
			}
			if !found {
				t.Fatal("missing failure diagnostic")
			}
		} else if !strings.Contains(detail.RenderedMarkdown.Content, "<h1>Custom detail</h1>") {
			t.Fatal("custom renderer not used")
		}
	}
}
