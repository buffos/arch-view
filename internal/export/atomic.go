package export

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/buffo/arch-view/internal/analysis"
)

func writeAtomically(outputPath string, data []byte, overwrite bool, ctx context.Context) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	outputPath = filepath.Clean(outputPath)
	if info, err := os.Lstat(outputPath); err == nil {
		if info.IsDir() {
			return analysis.NewHostError(analysis.ErrInvalidRequest, "export output path is a directory", map[string]any{"output": outputPath})
		}
		if !overwrite {
			return analysis.NewHostError(analysis.ErrInvalidRequest, "export output already exists; pass --overwrite to replace it", map[string]any{"output": outputPath})
		}
	} else if !os.IsNotExist(err) {
		return analysis.WrapHostError(analysis.ErrHostFailure, "export output could not be inspected", err, map[string]any{"output": outputPath})
	}

	directory := filepath.Dir(outputPath)
	base := filepath.Base(outputPath)
	temporary, err := os.CreateTemp(directory, "."+base+".tmp-*")
	if err != nil {
		return analysis.WrapHostError(analysis.ErrHostFailure, "export temporary file could not be created", err, map[string]any{"output": outputPath})
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
		return analysis.WrapHostError(analysis.ErrHostFailure, "export temporary file permissions could not be set", err, map[string]any{"output": outputPath})
	}
	if _, err := io.Copy(temporary, bytes.NewReader(data)); err != nil {
		_ = temporary.Close()
		return analysis.WrapHostError(analysis.ErrHostFailure, "export artifact could not be written", err, map[string]any{"output": outputPath})
	}
	if err := checkContext(ctx); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return analysis.WrapHostError(analysis.ErrHostFailure, "export artifact could not be flushed", err, map[string]any{"output": outputPath})
	}
	if err := temporary.Close(); err != nil {
		return analysis.WrapHostError(analysis.ErrHostFailure, "export temporary file could not be closed", err, map[string]any{"output": outputPath})
	}
	if err := checkContext(ctx); err != nil {
		return err
	}
	if err := replaceFile(temporaryPath, outputPath, overwrite); err != nil {
		return analysis.WrapHostError(analysis.ErrHostFailure, "export artifact could not replace its output atomically", err, map[string]any{"output": outputPath})
	}
	removeTemporary = false
	return nil
}
