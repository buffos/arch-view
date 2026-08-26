//go:build !windows

package viewer

import "os"

func replaceLayoutFile(source, target string) error {
	return os.Rename(source, target)
}
