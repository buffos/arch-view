package source

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"sort"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestIncrementalRevisionPreservesSortedRawSourceIdentity(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"a/child.md": "---\ntype: concept\n---\nChild\n",
		"a.md":       "---\r\ntype: concept\r\n---\r\nRoot\r\n",
		"a-other.md": "---\ntype: concept\n---\nSibling\n",
		"index.md":   "# Index\n", "log.md": "# Log\n",
	}
	paths := make([]string, 0, len(files))
	for path, text := range files {
		writeFile(t, filepath.Join(root, filepath.FromSlash(path)), text)
		paths = append(paths, path)
	}
	sort.Strings(paths)
	hash := sha256.New()
	for _, path := range paths {
		hash.Write([]byte(path + "\x00" + files[path] + "\x00"))
	}
	want := "sha256:" + hex.EncodeToString(hash.Sum(nil))
	scanner := NewFilesystemScanner()
	index, diagnostics := scanner.buildIndex(context.Background(), domain.BundleCandidate{BundleID: "test", AbsolutePath: root})
	if len(diagnostics) != 0 || index.SourceRevision != want || len(index.Documents) != 3 {
		t.Fatalf("revision=%s want=%s documents=%d diagnostics=%+v", index.SourceRevision, want, len(index.Documents), diagnostics)
	}
	if index.Documents["a"].Markdown != "Root\r\n" {
		t.Fatal("revision refactor changed source content")
	}
	writeFile(t, filepath.Join(root, "index.md"), "# Changed index\n")
	changed, _ := scanner.buildIndex(context.Background(), domain.BundleCandidate{BundleID: "test", AbsolutePath: root})
	if changed.SourceRevision == want {
		t.Fatal("index file omitted from revision")
	}
}
