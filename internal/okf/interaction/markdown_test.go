package interaction

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/profile"
	"github.com/buffo/arch-view/internal/okf/projection"
)

func TestDetailAndProjectionUseSameSourceState(t *testing.T) {
	for _, field := range []string{"status", "frontmatter.status", "frontmatter:status", " FRONTMATTER.status "} {
		for _, source := range []any{" complete ", " ", 42, nil} {
			document := domain.ConceptDocument{ConceptID: "one", SourcePath: "one.md", Frontmatter: map[string]any{"status": source}}
			index := domain.BundleIndex{ConceptOrder: []string{"one"}, Documents: map[string]domain.ConceptDocument{"one": document}}
			registry := profile.NewRegistry()
			value, _ := registry.ResolveProfile(profile.DefaultProfileID)
			value.State.Field = field
			value.State.Mapping = map[string]string{"complete": "done"}
			detail, err := Detail(context.Background(), index, value, "one", registry)
			if err != nil {
				t.Fatal(err)
			}
			snapshot, err := projection.Build(context.Background(), index, value, domain.NavigationState{Depth: 1}, registry)
			if err != nil {
				t.Fatal(err)
			}
			wantDeclared, wantEffective := "unknown", "unknown"
			if source == " complete " {
				wantDeclared, wantEffective = "complete", "done"
			}
			if detail.DeclaredState != wantDeclared || detail.EffectiveState != wantEffective {
				t.Fatalf("field %q source %v: detail states %q/%q", field, source, detail.DeclaredState, detail.EffectiveState)
			}
			if len(snapshot.Nodes) != 1 || snapshot.Nodes[0].DeclaredState != detail.DeclaredState || snapshot.Nodes[0].EffectiveState != detail.EffectiveState {
				t.Fatalf("graph and detail disagree: %#v", snapshot.Nodes)
			}
		}
	}
}

func TestDetailVisibilityAndUnicodeTruncation(t *testing.T) {
	markdown := strings.Repeat("a", maxDetailBytes-1) + "€ trailing"
	document := domain.ConceptDocument{ConceptID: "one", SourcePath: "one.md", Markdown: markdown, UnknownFrontmatter: map[string]any{"custom": "value"}}
	index := domain.BundleIndex{Documents: map[string]domain.ConceptDocument{"one": document}}
	value, _ := profile.NewRegistry().ResolveProfile(profile.DefaultProfileID)
	detail, err := Detail(context.Background(), index, value, "one", profile.NewRegistry())
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.RawMarkdown) != maxDetailBytes-1 || !utf8.ValidString(detail.RawMarkdown) || !utf8.ValidString(detail.RenderedMarkdown.Content) {
		t.Fatal("truncation split a character")
	}
	if len(detail.Diagnostics) != 1 || detail.Diagnostics[0].Code != "okf_detail_truncated" {
		t.Fatalf("missing truncation notice: %#v", detail.Diagnostics)
	}
	value.Details = domain.DetailSettings{RawConfigured: true, UnknownConfigured: true}
	hidden, err := Detail(context.Background(), index, value, "one", profile.NewRegistry())
	if err != nil {
		t.Fatal(err)
	}
	if hidden.RawMarkdown != "" || hidden.MappedMetadata != nil || hidden.RenderedMarkdown.Content == "" {
		t.Fatalf("visibility settings ignored: raw=%d metadata=%v", len(hidden.RawMarkdown), hidden.MappedMetadata)
	}
	if index.Documents["one"].Markdown != markdown {
		t.Fatal("detail mutated source")
	}
}

func TestSanitizeMarkdownRendersSupportedCommonMarkAndRejectsUnsafeLinks(t *testing.T) {
	index := domain.BundleIndex{BundleID: "bundle/.okf", SourceRevision: "source:1", Documents: map[string]domain.ConceptDocument{
		"one": {ConceptID: "one", SourcePath: "one.md", Type: "reference", Markdown: "# Heading\n\n**bold** and *emphasis* with `literal` and [safe](/two.md) [bad](javascript:alert(1))\n\n- First item\n- Second item\n\n<script>alert(1)</script>"},
		"two": {ConceptID: "two", SourcePath: "two.md", Type: "reference"},
	}}
	profileValue, _ := profile.NewRegistry().ResolveProfile(profile.DefaultProfileID)
	detail, err := Detail(context.Background(), index, profileValue, "one", profile.NewRegistry())
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if detail.RenderedMarkdown.Format != "sanitized_commonmark" || len(detail.RenderedMarkdown.Content) == 0 {
		t.Fatalf("rendered = %#v", detail.RenderedMarkdown)
	}
	if detail.RenderedMarkdown.Content == "" || contains(detail.RenderedMarkdown.Content, "<script") || contains(detail.RenderedMarkdown.Content, "javascript:") {
		t.Fatalf("unsafe markup survived: %s", detail.RenderedMarkdown.Content)
	}
	if len(detail.RenderedMarkdown.Links) < 2 {
		t.Fatalf("links = %#v", detail.RenderedMarkdown.Links)
	}
	for _, fragment := range []string{"<h1>Heading</h1>", "<strong>bold</strong>", "<em>emphasis</em>", "<code>literal</code>", "<ul>", "<li>First item</li>", "<li>Second item</li>"} {
		if !strings.Contains(detail.RenderedMarkdown.Content, fragment) {
			t.Fatalf("supported CommonMark missing %q: %s", fragment, detail.RenderedMarkdown.Content)
		}
	}
}

func contains(value, needle string) bool {
	for index := 0; index+len(needle) <= len(value); index++ {
		if value[index:index+len(needle)] == needle {
			return true
		}
	}
	return false
}
