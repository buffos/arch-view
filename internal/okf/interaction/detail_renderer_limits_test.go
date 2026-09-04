package interaction

import (
	"context"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/profile"
)

type callbackDetailRenderer struct {
	testDetailRenderer
	call func() (string, error)
}

func (renderer callbackDetailRenderer) Render(context.Context, domain.ConceptDocument, map[string]any) (string, error) {
	return renderer.call()
}

func TestCustomDetailCancellationDiscardsOutput(t *testing.T) {
	for _, cancelBefore := range []bool{true, false} {
		ctx, cancel := context.WithCancel(context.Background())
		calls := 0
		renderer := callbackDetailRenderer{call: func() (string, error) {
			calls++
			cancel()
			return "Must not be displayed", nil
		}}
		if cancelBefore {
			cancel()
		}
		registry := profile.NewRegistry()
		effective, _ := registry.ResolveProfile(profile.DefaultProfileID)
		index := domain.BundleIndex{BundleID: "bundle", Documents: map[string]domain.ConceptDocument{"root": {ConceptID: "root", Markdown: "Original"}}}
		detail, err := DetailWithRenderer(ctx, index, effective, "root", registry, renderer)
		cancel()
		var problem *domain.Error
		if !errors.As(err, &problem) || problem.Code != "okf_operation_cancelled" {
			t.Fatalf("cancellation lost: %v", err)
		}
		if detail.ConceptID != "" || detail.RenderedMarkdown.Content != "" || len(detail.Diagnostics) != 0 {
			t.Fatalf("cancelled request returned partial detail: %+v", detail)
		}
		if (cancelBefore && calls != 0) || (!cancelBefore && calls != 1) {
			t.Fatalf("unexpected renderer calls: %d", calls)
		}
		if _, err := Detail(context.Background(), index, effective, "root", registry); err != nil {
			t.Fatalf("later request poisoned: %v", err)
		}
	}
}

func TestCustomDetailUsesExistingUTF8DisplayBudget(t *testing.T) {
	registry := profile.NewRegistry()
	effective, _ := registry.ResolveProfile(profile.DefaultProfileID)
	document := domain.ConceptDocument{ConceptID: "root", SourcePath: "root.md", Markdown: "Original **content**", Frontmatter: map[string]any{"custom": "retained"}}
	index := domain.BundleIndex{BundleID: "bundle", SourceRevision: "source", Documents: map[string]domain.ConceptDocument{"root": document}}
	renderer := callbackDetailRenderer{call: func() (string, error) { return strings.Repeat("界", maxDetailBytes/3+1) + "OMITTED_TAIL", nil }}
	detail, err := DetailWithRenderer(context.Background(), index, effective, "root", registry, renderer)
	if err != nil {
		t.Fatal(err)
	}
	content := detail.RenderedMarkdown.Content
	if !utf8.ValidString(content) || strings.Contains(content, "OMITTED_TAIL") || strings.Count(content, "界") != maxDetailBytes/3 {
		t.Fatal("custom renderer bypassed UTF-8-safe input budget")
	}
	if detail.RawMarkdown != document.Markdown || detail.Frontmatter["custom"] != "retained" || index.Documents["root"].Markdown != document.Markdown {
		t.Fatal("display truncation changed source")
	}
	if len(detail.Diagnostics) != 1 || detail.Diagnostics[0].Code != "okf_detail_truncated" || detail.Diagnostics[0].BundleID != "bundle" || detail.Diagnostics[0].ConceptID != "root" {
		t.Fatalf("missing scoped truncation warning: %+v", detail.Diagnostics)
	}
}
