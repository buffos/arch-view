package markdown

import (
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestCommonMarkLinksAndRenderingSharePolicy(t *testing.T) {
	source := "[**Reference**][ref] and ` [ignored](missing.md) `\n\n[ref]: <child name.md>\n\n[balanced](child(test).md)\n\n![image](https://example.com/image.png)\n\n```\n[example](missing.md)\n```\n"
	document := domain.ConceptDocument{ConceptID: "root", SourcePath: "root.md", Markdown: source}
	index := domain.BundleIndex{Documents: map[string]domain.ConceptDocument{"child name": {}, "child(test)": {}}}
	extracted := Extract(source)
	if len(extracted) != 2 {
		t.Fatalf("expected only two real links, got %#v", extracted)
	}
	rendered, links, diagnostics := Render(document, index)
	if len(links) != 2 || len(diagnostics) != 0 {
		t.Fatalf("links=%#v diagnostics=%#v", links, diagnostics)
	}
	for i, link := range extracted {
		resolved, diagnostic := Resolve(document, index, link)
		if diagnostic != nil || !resolved.Safe || resolved.TargetID != links[i].TargetID {
			t.Fatalf("inconsistent policy: %#v %#v", resolved, diagnostic)
		}
	}
	for _, expected := range []string{"<strong>Reference</strong>", "#okf-concept=child%20name", "<code>", "[example](missing.md)"} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("missing %q in %s", expected, rendered)
		}
	}
	if strings.Contains(rendered, "<img") {
		t.Fatalf("images must remain inert: %s", rendered)
	}
}

func TestUnsafeLinksAndRawHTMLRemainInert(t *testing.T) {
	document := domain.ConceptDocument{ConceptID: "root", SourcePath: "root.md", Markdown: "[bad](javascript&#58;alert(1)) [network](//example.com/file) [escape](../outside.md)\n\n<script>alert(1)</script>\n\n`<b>literal</b>`\n\n<https://example.com>"}
	rendered, links, diagnostics := Render(document, domain.BundleIndex{})
	if len(links) != 4 || len(diagnostics) != 3 {
		t.Fatalf("links=%#v diagnostics=%#v", links, diagnostics)
	}
	if strings.Contains(rendered, "<script") || strings.Contains(rendered, "href=\"javascript") || strings.Contains(rendered, "href=\"//") {
		t.Fatalf("unsafe HTML: %s", rendered)
	}
	if !strings.Contains(rendered, "&lt;b&gt;literal&lt;/b&gt;") || !strings.Contains(rendered, "rel=\"noopener noreferrer\"") || !strings.Contains(rendered, "https://example.com</a>") {
		t.Fatalf("incorrect safe rendering: %s", rendered)
	}
}
