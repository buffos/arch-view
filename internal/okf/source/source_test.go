package source

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestFilesystemScannerDiscoversIndependentBundlesAndPreservesFacts(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "first", ".okf", "index.md"), "# First\n")
	writeFile(t, filepath.Join(root, "first", ".okf", "one.md"), "---\ntype: Reference\ntitle: One\ncustom:\n  owner: team\ntags: [one]\n---\nSee [two](/two.md).\n")
	writeFile(t, filepath.Join(root, "first", ".okf", "two.md"), "---\ntype: Reference\ntitle: Two\n---\nSecond.\n")
	writeFile(t, filepath.Join(root, "second", ".okf", "bad.md"), "not frontmatter\n")
	writeFile(t, filepath.Join(root, "node_modules", "hidden", ".okf", "ignored.md"), "not frontmatter\n")

	scanner := NewFilesystemScanner()
	candidates, diagnostics := scanner.Scan(context.Background(), root)
	if len(diagnostics) != 0 {
		t.Fatalf("discovery diagnostics = %#v", diagnostics)
	}
	if len(candidates) != 2 {
		t.Fatalf("candidates = %d, want 2", len(candidates))
	}
	if candidates[0].BundleID != "first/.okf" || candidates[0].Status != domain.BundleValid || !candidates[0].Selectable {
		t.Fatalf("first candidate = %#v", candidates[0])
	}
	if candidates[1].Status != domain.BundleInvalid || candidates[1].Selectable {
		t.Fatalf("second candidate = %#v", candidates[1])
	}

	index, err := scanner.Index(context.Background(), candidates[0])
	if err != nil {
		t.Fatalf("index: %v", err)
	}
	document := index.Documents["one"]
	if document.UnknownFrontmatter["custom"] == nil || document.Title != "One" || len(document.Links) != 1 {
		t.Fatalf("lossless document = %#v", document)
	}
	if len(index.Relationships) != 1 || index.Relationships[0].Kind != domain.RelationshipSemantic {
		t.Fatalf("semantic links = %#v", index.Relationships)
	}
}

func TestFilesystemScannerDiagnosesBoundaryAndUnsafeLinksWithoutMutation(t *testing.T) {
	root := t.TempDir()
	pathValue := filepath.Join(root, ".okf", "one.md")
	content := "---\ntype: Reference\n---\n[escape](../../outside.md) [script](javascript:alert(1)) [missing](/missing.md)\n"
	writeFile(t, pathValue, content)
	scanner := NewFilesystemScanner()
	candidates, _ := scanner.Scan(context.Background(), root)
	if len(candidates) != 1 || candidates[0].Status != domain.BundleValid {
		t.Fatalf("candidate = %#v", candidates)
	}
	index, err := scanner.Index(context.Background(), candidates[0])
	if err != nil {
		t.Fatalf("index: %v", err)
	}
	var sawBoundary, sawUnsafe, sawMissing bool
	for _, diagnostic := range index.Diagnostics {
		sawBoundary = sawBoundary || diagnostic.Code == "okf_bundle_boundary_violation"
		sawUnsafe = sawUnsafe || diagnostic.Code == "okf_unsafe_link"
		sawMissing = sawMissing || diagnostic.Code == "okf_relationship_unresolved"
	}
	if !sawBoundary || !sawUnsafe || !sawMissing {
		t.Fatalf("link diagnostics = %#v", index.Diagnostics)
	}
	if link := index.Documents["one"].Links; len(link) != 3 || link[2].Safe {
		t.Fatalf("unresolved local link should not be safe: %#v", link)
	}
	if value, readErr := os.ReadFile(pathValue); readErr != nil || string(value) != content {
		t.Fatalf("source was changed: %v %q", readErr, value)
	}
}

func TestFilesystemScannerRejectsMissingTypeAndKeepsCandidateVisible(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".okf", "broken.md"), "---\ntitle: Missing type\n---\n")
	candidates, _ := NewFilesystemScanner().Scan(context.Background(), root)
	if len(candidates) != 1 || candidates[0].Status != domain.BundleInvalid || candidates[0].Selectable {
		t.Fatalf("candidate = %#v", candidates)
	}
	if len(candidates[0].Diagnostics) == 0 {
		t.Fatal("missing validation diagnostics")
	}
}

func writeFile(t *testing.T, pathValue, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(pathValue), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pathValue, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
