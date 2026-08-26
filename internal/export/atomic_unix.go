//go:build !windows

package export

import "os"

func replaceFile(source, target string, overwrite bool) error {
	if overwrite {
		return os.Rename(source, target)
	}

	// A hard link creates the target atomically only when it does not already
	// exist. This closes the check-then-rename race without truncating a file
	// that appeared after the initial Lstat.
	if err := os.Link(source, target); err != nil {
		return err
	}
	// The target is already the committed artifact. Removing the temporary
	// hard link is cleanup and must not turn a successful commit into a
	// reported write failure.
	_ = os.Remove(source)
	return nil
}
