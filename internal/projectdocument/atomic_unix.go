//go:build !windows

package projectdocument

import "os"

func replaceFile(source, target string) error { return os.Rename(source, target) }
