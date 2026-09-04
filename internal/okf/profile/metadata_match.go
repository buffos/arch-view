package profile

import (
	"bytes"
	"encoding/json"
	"strings"
)

// JSON comparison preserves metadata types while allowing equivalent numeric
// values decoded through YAML and JSON (for example int(1) and float64(1)).
func metadataEqual(left, right any) bool {
	a, err := json.Marshal(left)
	if err != nil {
		return false
	}
	b, err := json.Marshal(right)
	return err == nil && bytes.Equal(a, b)
}

func metadataContains(value, needle any) bool {
	switch collection := value.(type) {
	case string:
		text, ok := needle.(string)
		return ok && text != "" && strings.Contains(strings.ToLower(collection), strings.ToLower(text))
	case []any:
		for _, item := range collection {
			if metadataEqual(item, needle) {
				return true
			}
		}
	case []string:
		for _, item := range collection {
			if metadataEqual(item, needle) {
				return true
			}
		}
	}
	return false
}
