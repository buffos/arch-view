package source

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestSourceLimitsRejectBundleWithoutPartialIndex(t *testing.T) {
	const content = "---\ntype: concept\n---\nBody\n"
	for _, limit := range []struct {
		name   string
		limits IndexLimits
	}{
		{"files", IndexLimits{MaxFiles: 1}},
		{"bytes", IndexLimits{MaxBytes: int64(len(content))}},
	} {
		t.Run(limit.name, func(t *testing.T) {
			root := t.TempDir()
			bad := filepath.Join(root, "large", ".okf")
			writeFile(t, filepath.Join(bad, "a.md"), content)
			writeFile(t, filepath.Join(bad, "b.md"), content)
			writeFile(t, filepath.Join(root, "small", ".okf", "a.md"), content)
			scanner := NewFilesystemScanner()
			scanner.Limits = limit.limits
			candidates, _ := scanner.Scan(context.Background(), root)
			if len(candidates) != 2 || candidates[0].Selectable || !candidates[1].Selectable {
				t.Fatalf("bundle isolation=%+v", candidates)
			}
			found := false
			for _, diagnostic := range candidates[0].Diagnostics {
				found = found || diagnostic.Code == "okf_bundle_source_limit" && diagnostic.Details["limit"] == limit.name && diagnostic.Recovery != ""
			}
			if !found {
				t.Fatalf("missing limit diagnostic: %+v", candidates[0])
			}
			index, err := scanner.Index(context.Background(), domain.BundleCandidate{BundleID: "large/.okf", AbsolutePath: bad, Selectable: true})
			if err == nil || len(index.Documents) != 0 || index.SourceRevision != "" {
				t.Fatalf("partial index leaked: %+v %v", index, err)
			}
			accepted, err := scanner.Index(context.Background(), candidates[1])
			if err != nil || len(accepted.Documents) != 1 || accepted.Documents["a"].Markdown != "Body\n" {
				t.Fatalf("exact-limit source not preserved: %+v %v", accepted, err)
			}
		})
	}
}

func TestSourceLimitsCountIndexAndLogFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "index.md"), "# index\n")
	writeFile(t, filepath.Join(root, "log.md"), "# log\n")
	scanner := NewFilesystemScanner()
	scanner.Limits = IndexLimits{MaxBytes: 8}
	_, diagnostics := scanner.buildIndex(context.Background(), domain.BundleCandidate{BundleID: "test", AbsolutePath: root})
	if len(diagnostics) != 1 || diagnostics[0].Code != "okf_bundle_source_limit" {
		t.Fatalf("index/log bypassed source budget: %+v", diagnostics)
	}
}
