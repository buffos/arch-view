package canonical

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
)

func mergeMetadata(left, right map[string]any) map[string]any {
	result := cloneMetadata(left)
	if result == nil {
		result = map[string]any{}
	}
	for key, value := range right {
		if _, exists := result[key]; !exists {
			result[key] = value
		}
	}
	return result
}

func cloneMetadata(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	result := make(map[string]any, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}

func sortedUnique(values []string) []string {
	result := append([]string{}, values...)
	sort.Strings(result)
	if len(result) < 2 {
		return result
	}
	write := 1
	for _, value := range result[1:] {
		if value != result[write-1] {
			result[write] = value
			write++
		}
	}
	return result[:write]
}

func normalizedPath(value string) string {
	if value == "" {
		return ""
	}
	return path.Clean(strings.ReplaceAll(value, "\\", "/"))
}

func repositoryRelative(value string) bool {
	normalized := normalizedPath(value)
	if normalized == "" || normalized == "." || strings.HasPrefix(normalized, "/") || windowsAbsolutePath(normalized) {
		return false
	}
	return normalized != ".." && !strings.HasPrefix(normalized, "../")
}

func windowsAbsolutePath(value string) bool {
	return len(value) >= 2 && ((value[0] >= 'a' && value[0] <= 'z') || (value[0] >= 'A' && value[0] <= 'Z')) && value[1] == ':'
}

func validPosition(position *analysis.Position) bool {
	return position == nil || (position.Line > 0 && position.Column > 0)
}

func clonePosition(position *analysis.Position) *analysis.Position {
	if position == nil {
		return nil
	}
	copy := *position
	return &copy
}

func positionKey(position *analysis.Position) string {
	if position == nil {
		return ""
	}
	return fmt.Sprintf("%d:%d", position.Line, position.Column)
}

func modelID(value model.Model) string {
	value.ModelID = ""
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return "model-" + hex.EncodeToString(sum[:12])
}

func stableID(kind string, parts ...string) string {
	payload := kind + "\x00" + strings.Join(parts, "\x00")
	sum := sha256.Sum256([]byte(payload))
	return "model:" + kind + ":" + hex.EncodeToString(sum[:8])
}
