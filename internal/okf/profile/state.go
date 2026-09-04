package profile

import (
	"strings"

	"github.com/buffo/arch-view/internal/okf/domain"
)

// DeclaredState reads the profile-selected frontmatter field without modifying source data.
func DeclaredState(document domain.ConceptDocument, field string) string {
	key := strings.TrimSpace(field)
	if key == "" {
		return "unknown"
	}
	lower := strings.ToLower(key)
	if strings.HasPrefix(lower, "frontmatter.") || strings.HasPrefix(lower, "frontmatter:") {
		key = key[len("frontmatter."):]
	}
	raw, _ := metadataField(document, key)
	if value, ok := raw.(string); ok {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return "unknown"
}

// MappedState applies vocabulary mapping before any rule or structural override.
func MappedState(value string, mapping map[string]string) string {
	if mapped, ok := mapping[value]; ok && mapped != "" {
		return mapped
	}
	if value == "" {
		return "unknown"
	}
	return value
}
