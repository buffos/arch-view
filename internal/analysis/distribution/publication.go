package distribution

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// PublishDirectory atomically replaces outputRoot with a fully prepared
// directory. The old directory is retained until the replacement succeeds.
func PublishDirectory(stagingRoot, outputRoot string) error {
	staging, err := absoluteExistingDirectory(stagingRoot, "staging root")
	if err != nil {
		return err
	}
	output, err := filepath.Abs(filepath.Clean(outputRoot))
	if err != nil {
		return fmt.Errorf("resolve publication root: %w", err)
	}
	if staging == output {
		return fmt.Errorf("staging root and publication root must differ")
	}
	stagingResolved, err := resolvePathForComparison(staging)
	if err != nil {
		return fmt.Errorf("resolve staging root for publication: %w", err)
	}
	outputResolved, err := resolvePathForComparison(output)
	if err != nil {
		return fmt.Errorf("resolve publication root: %w", err)
	}
	if pathsOverlap(stagingResolved, outputResolved) {
		return fmt.Errorf("staging root and publication root must not contain one another")
	}
	parent := filepath.Dir(output)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("create publication parent: %w", err)
	}

	outputInfo, statErr := os.Lstat(output)
	if statErr != nil && !os.IsNotExist(statErr) {
		return fmt.Errorf("stat publication root %s: %w", output, statErr)
	}
	if os.IsNotExist(statErr) {
		if err := os.Rename(staging, output); err != nil {
			return fmt.Errorf("rename staged distribution into place: %w", err)
		}
		return nil
	}
	if outputInfo.Mode()&os.ModeSymlink != 0 || !outputInfo.IsDir() {
		return fmt.Errorf("publication root %s must be a directory", output)
	}

	backup, err := reserveSibling(parent, "."+filepath.Base(output)+".backup-")
	if err != nil {
		return fmt.Errorf("reserve publication backup: %w", err)
	}
	if err := os.Rename(output, backup); err != nil {
		return fmt.Errorf("move previous distribution aside: %w", err)
	}
	if err := os.Rename(staging, output); err != nil {
		restoreErr := os.Rename(backup, output)
		if restoreErr != nil {
			return fmt.Errorf("publish staged distribution: %w; restore previous distribution from %s: %v", err, backup, restoreErr)
		}
		return fmt.Errorf("publish staged distribution: %w", err)
	}
	// Publication has succeeded at this point. Backup cleanup is best-effort so
	// a cleanup failure cannot report the build as failed after replacing output.
	_ = os.RemoveAll(backup)
	return nil
}

func ensureOutputDoesNotContainRepository(repositoryRoot, outputRoot string) error {
	repositoryResolved, err := resolvePathForComparison(repositoryRoot)
	if err != nil {
		return fmt.Errorf("resolve repository root for publication safety: %w", err)
	}
	outputResolved, err := resolvePathForComparison(outputRoot)
	if err != nil {
		return fmt.Errorf("resolve output root for publication safety: %w", err)
	}
	if pathContains(outputResolved, repositoryResolved) {
		return fmt.Errorf("output root %s must not be the repository root or one of its ancestors", outputRoot)
	}
	return nil
}

func pathsOverlap(left, right string) bool {
	return pathContains(left, right) || pathContains(right, left)
}

func pathContains(parent, child string) bool {
	if samePath(parent, child) {
		return true
	}
	relative, err := filepath.Rel(parent, child)
	return err == nil && !filepath.IsAbs(relative) && relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator))
}

func samePath(left, right string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
	}
	return filepath.Clean(left) == filepath.Clean(right)
}

func resolvePathForComparison(value string) (string, error) {
	absolute, err := filepath.Abs(filepath.Clean(value))
	if err != nil {
		return "", err
	}
	existing := absolute
	var suffix []string
	for {
		if _, statErr := os.Lstat(existing); statErr == nil {
			break
		} else if !os.IsNotExist(statErr) {
			return "", statErr
		}
		parent := filepath.Dir(existing)
		if samePath(parent, existing) {
			return "", fmt.Errorf("no existing ancestor for %s", absolute)
		}
		suffix = append(suffix, filepath.Base(existing))
		existing = parent
	}
	resolved, err := filepath.EvalSymlinks(existing)
	if err != nil {
		return "", err
	}
	for i := len(suffix) - 1; i >= 0; i-- {
		resolved = filepath.Join(resolved, suffix[i])
	}
	return filepath.Clean(resolved), nil
}

func reserveSibling(parent, pattern string) (string, error) {
	path, err := os.MkdirTemp(parent, pattern)
	if err != nil {
		return "", err
	}
	if err := os.Remove(path); err != nil {
		return "", err
	}
	return path, nil
}
