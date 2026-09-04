package domain

import (
	"strings"
	"unicode"
)

// ValidExtensionIdentity keeps namespaced ID@version references unambiguous.
func ValidExtensionIdentity(id, version string) bool {
	invalid := func(r rune) bool { return r == '@' || unicode.IsSpace(r) || unicode.IsControl(r) }
	if version == "" || strings.ContainsFunc(id, invalid) || strings.ContainsFunc(version, invalid) {
		return false
	}
	parts := strings.Split(id, ".")
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
	}
	return true
}
