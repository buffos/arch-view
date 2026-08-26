package viewer

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	"github.com/buffo/arch-view/internal/analysis"
)

func writeLayoutConfigAtomically(outputPath string, data []byte) error {
	outputPath = filepath.Clean(outputPath)
	info, err := os.Stat(filepath.Dir(outputPath))
	if err != nil {
		return analysis.WrapHostError(analysis.ErrUnreadableProject, "layout configuration directory could not be inspected", err, map[string]any{"directory": filepath.Dir(outputPath)})
	}
	if !info.IsDir() {
		return analysis.NewHostError(analysis.ErrInvalidRequest, "layout configuration destination is not a directory", map[string]any{"directory": filepath.Dir(outputPath)})
	}
	temporary, err := os.CreateTemp(filepath.Dir(outputPath), ".archview.json.tmp-*")
	if err != nil {
		return analysis.WrapHostError(analysis.ErrHostFailure, "layout configuration temporary file could not be created", err, map[string]any{"path": outputPath})
	}
	temporaryPath := temporary.Name()
	removeTemporary := true
	defer func() {
		if removeTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0o644); err != nil {
		_ = temporary.Close()
		return analysis.WrapHostError(analysis.ErrHostFailure, "layout configuration permissions could not be set", err, map[string]any{"path": outputPath})
	}
	if _, err := io.Copy(temporary, bytes.NewReader(data)); err != nil {
		_ = temporary.Close()
		return analysis.WrapHostError(analysis.ErrHostFailure, "layout configuration could not be written", err, map[string]any{"path": outputPath})
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return analysis.WrapHostError(analysis.ErrHostFailure, "layout configuration could not be flushed", err, map[string]any{"path": outputPath})
	}
	if err := temporary.Close(); err != nil {
		return analysis.WrapHostError(analysis.ErrHostFailure, "layout configuration temporary file could not be closed", err, map[string]any{"path": outputPath})
	}
	if err := replaceLayoutFile(temporaryPath, outputPath); err != nil {
		return analysis.WrapHostError(analysis.ErrHostFailure, "layout configuration could not replace its destination atomically", err, map[string]any{"path": outputPath})
	}
	removeTemporary = false
	return nil
}
