package projectdocument

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
)

func writeAtomically(outputPath string, data []byte) error {
	directory := filepath.Dir(filepath.Clean(outputPath))
	if info, err := os.Stat(directory); err != nil || !info.IsDir() {
		if err != nil {
			return err
		}
		return os.ErrInvalid
	}
	temporary, err := os.CreateTemp(directory, ".archview.json.tmp-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	removeTemporary := true
	defer func() {
		if removeTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()
	mode := os.FileMode(0o644)
	if info, err := os.Stat(outputPath); err == nil {
		mode = info.Mode().Perm()
	}
	if err := temporary.Chmod(mode); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := io.Copy(temporary, bytes.NewReader(data)); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := replaceFile(temporaryPath, outputPath); err != nil {
		return err
	}
	removeTemporary = false
	return nil
}
