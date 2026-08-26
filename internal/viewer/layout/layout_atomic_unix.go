//go:build !windows

package layout

import "os"

func replaceLayoutFile(source, target string) error {
	return os.Rename(source, target)
}
