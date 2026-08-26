package viewer

import (
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

func normalizeSourceRoot(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	absolute, err := filepath.Abs(value)
	if err != nil {
		return "", analysis.WrapHostError(analysis.ErrInvalidRequest, "source root could not be normalized", err, nil)
	}
	absolute = filepath.Clean(absolute)
	info, err := os.Stat(absolute)
	if err != nil {
		return "", analysis.WrapHostError(analysis.ErrUnreadableProject, "source root could not be read", err, map[string]any{"project_root": value})
	}
	if !info.IsDir() {
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "source root must be a directory", map[string]any{"project_root": value})
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", analysis.WrapHostError(analysis.ErrUnreadableProject, "source root symlinks could not be resolved", err, map[string]any{"project_root": value})
	}
	return filepath.Clean(resolved), nil
}

func safeRepositoryPath(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "source path is required", nil)
	}
	normalized := strings.ReplaceAll(value, "\\", "/")
	cleaned := path.Clean(normalized)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.HasPrefix(cleaned, "/") || windowsAbsolutePath(cleaned) {
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "source path must remain repository-relative", map[string]any{"path": value})
	}
	return cleaned, nil
}

func containedFilePath(root, relative string) (string, error) {
	candidate, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		return "", analysis.WrapHostError(analysis.ErrInvalidRequest, "source path could not be normalized", err, nil)
	}
	if !samePathRoot(root, candidate) {
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "source path escapes the project root", map[string]any{"path": relative})
	}
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		if os.IsNotExist(err) {
			return "", analysis.NewHostError(analysis.ErrInvalidModel, "source file was not found", map[string]any{"path": relative})
		}
		return "", analysis.WrapHostError(analysis.ErrUnreadableProject, "source file could not be resolved", err, map[string]any{"path": relative})
	}
	if !samePathRoot(root, resolved) {
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "source symlink escapes the project root", map[string]any{"path": relative})
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", analysis.WrapHostError(analysis.ErrUnreadableProject, "source file could not be inspected", err, map[string]any{"path": relative})
	}
	if info.IsDir() {
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "source path is a directory", map[string]any{"path": relative})
	}
	return resolved, nil
}

func readSourceFile(filePath string) ([]byte, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, analysis.WrapHostError(analysis.ErrUnreadableProject, "source file could not be inspected", err, map[string]any{"path": filePath})
	}
	if info.Size() > maxSourceBytes {
		return nil, analysis.NewHostError(analysis.ErrInvalidRequest, "source file is too large to inspect", map[string]any{"max_bytes": maxSourceBytes})
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil, analysis.WrapHostError(analysis.ErrUnreadableProject, "source file could not be read", err, map[string]any{"path": filePath})
	}
	defer func() { _ = file.Close() }()
	content, err := io.ReadAll(io.LimitReader(file, maxSourceBytes+1))
	if err != nil {
		return nil, analysis.WrapHostError(analysis.ErrUnreadableProject, "source file could not be read", err, map[string]any{"path": filePath})
	}
	if len(content) > maxSourceBytes {
		return nil, analysis.NewHostError(analysis.ErrInvalidRequest, "source file is too large to inspect", map[string]any{"max_bytes": maxSourceBytes})
	}
	return content, nil
}

func samePathRoot(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

func samePath(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func windowsAbsolutePath(value string) bool {
	return len(value) >= 2 && ((value[0] >= 'a' && value[0] <= 'z') || (value[0] >= 'A' && value[0] <= 'Z')) && value[1] == ':'
}

func sourceErrorStatus(err error) int {
	switch analysis.ErrorCodeOf(err) {
	case analysis.ErrInvalidRequest:
		return http.StatusForbidden
	case analysis.ErrInvalidModel:
		return http.StatusNotFound
	default:
		return http.StatusUnprocessableEntity
	}
}
