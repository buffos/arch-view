package profile

import (
	"strings"

	"github.com/buffo/arch-view/internal/okf/domain"
)

// Literal metadata keys take precedence over dotted nested paths.
func metadataField(document domain.ConceptDocument, field string) (any, bool) {
	if value, exists := document.Frontmatter[field]; exists {
		return value, true
	}
	if field == "type" && document.Type != "" {
		return document.Type, true
	}
	var value any = document.Frontmatter
	for _, key := range strings.Split(field, ".") {
		object, ok := value.(map[string]any)
		if !ok {
			return nil, false
		}
		value, ok = object[key]
		if !ok {
			return nil, false
		}
	}
	return value, true
}
