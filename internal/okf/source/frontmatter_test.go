package source

import (
	"strings"
	"testing"
)

func TestFrontmatterPreservesAliasesAndMergeValues(t *testing.T) {
	document, err := parseConcept("one.md", []byte("---\ntype: concept\noriginal: &values\n  label: Example\n  count: 2\ncopy: *values\nmerged:\n  <<: *values\n  count: 3\n---\nText"))
	if err != nil {
		t.Fatal(err)
	}
	copy := document.Frontmatter["copy"].(map[string]any)
	merged := document.Frontmatter["merged"].(map[string]any)
	if copy["label"] != "Example" || copy["count"] != 2 || merged["count"] != 3 || merged["label"] != "Example" {
		t.Fatalf("metadata lost: %#v", document.Frontmatter)
	}
	copy["label"] = "changed"
	if document.Frontmatter["original"].(map[string]any)["label"] != "Example" {
		t.Fatal("alias values share mutable state")
	}
}

func TestFrontmatterRejectsLossyOrRecursiveValues(t *testing.T) {
	for _, metadata := range []string{
		"nested:\n  key: first\n  key: second\n",
		"nested:\n  42: numeric-key\n",
		"nested: &self [*self]\n",
		"value: .inf\n",
	} {
		if _, err := parseConcept("one.md", []byte("---\ntype: concept\n"+metadata+"---\n")); err == nil {
			t.Errorf("accepted lossy metadata: %s", metadata)
		}
	}
}

func TestConceptPreservesMarkdownLineEndings(t *testing.T) {
	for _, ending := range []string{"\n", "\r\n"} {
		body := "# Ελληνικά" + ending + ending + "Text with € and 日本語." + ending
		source := "---" + ending + "type: concept" + ending + "title: Example" + ending + "---" + ending + body
		document, err := parseConcept("one.md", []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		if document.Markdown != body {
			t.Fatalf("source body changed: want %q got %q", body, document.Markdown)
		}
	}
}

func TestConceptRejectsInvalidUTF8Body(t *testing.T) {
	data := append([]byte("---\ntype: concept\n---\nBody "), 0xff)
	if _, err := parseConcept("one.md", data); err == nil || !strings.Contains(err.Error(), "UTF-8") {
		t.Fatalf("invalid body encoding accepted: %v", err)
	}
}
