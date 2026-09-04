package source

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadConceptFileBoundsAllocation(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "large.md")
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(int64(maxConceptBytes * 4)); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := readConceptFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != maxConceptBytes+1 {
		t.Fatalf("read %d bytes, expected bounded oversized sentinel", len(data))
	}
	if _, err := readConceptFile(filepath.Dir(filePath)); err == nil {
		t.Fatal("accepted a directory as a concept file")
	}
}
