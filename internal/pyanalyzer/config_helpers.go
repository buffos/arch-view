package pyanalyzer

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

var pythonVersionSelectorPattern = regexp.MustCompile(`^3\.(?:[0-9]+|x)$`)
var pythonVersionPattern = regexp.MustCompile(`(?:^|[^0-9])([23]\.[0-9]+)(?:[^0-9]|$)`)

func isRelevantTOMLKey(section, key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	return key == "source_roots" || key == "source-roots" || key == "where" || key == "package-dir" || key == "package_dir" || key == "\"\"" || (section == "tool.poetry" && key == "packages") || (section == "project" && key == "requires-python")
}

func splitTOMLAssignment(line string) (string, string, bool) {
	quote := byte(0)
	depth := 0
	for index := 0; index < len(line); index++ {
		switch line[index] {
		case '\'', '"':
			if quote == 0 {
				quote = line[index]
			} else if quote == line[index] && (index == 0 || line[index-1] != '\\') {
				quote = 0
			}
		case '[', '{':
			if quote == 0 {
				depth++
			}
		case ']', '}':
			if quote == 0 && depth > 0 {
				depth--
			}
		case '=':
			if quote == 0 && depth == 0 {
				return line[:index], line[index+1:], true
			}
		}
	}
	return "", "", false
}

func balancedTOMLValue(value string) bool {
	quote := byte(0)
	depth := 0
	for index := 0; index < len(value); index++ {
		char := value[index]
		if quote != 0 {
			if char == quote && (index == 0 || value[index-1] != '\\') {
				quote = 0
			}
			continue
		}
		switch char {
		case '\'', '"':
			quote = char
		case '[', '{':
			depth++
		case ']', '}':
			depth--
			if depth < 0 {
				return false
			}
		}
	}
	return quote == 0 && depth == 0
}

func parseTOMLStrings(value string) ([]string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("value is empty")
	}
	if value[0] != '[' {
		parsed, err := parseQuoted(value)
		if err != nil {
			return nil, err
		}
		return []string{parsed}, nil
	}
	if len(value) < 2 || value[len(value)-1] != ']' {
		return nil, fmt.Errorf("array is not closed")
	}
	body := strings.TrimSpace(value[1 : len(value)-1])
	if body == "" {
		return []string{}, nil
	}
	parts, err := splitDelimited(body, ',')
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		parsed, err := parseQuoted(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		result = append(result, parsed)
	}
	return result, nil
}

func parseTOMLPackageRoots(value string) []string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "{") {
		pattern := regexp.MustCompile(`["']{2}\s*=\s*["']([^"']+)["']`)
		if match := pattern.FindStringSubmatch(value); len(match) > 1 {
			return []string{strings.TrimSpace(match[1])}
		}
		return nil
	}
	if parsed, err := parseQuoted(value); err == nil {
		return []string{parsed}
	}
	return nil
}

func parsePoetryRoots(value string) []string {
	pattern := regexp.MustCompile(`(?s)\bfrom\s*=\s*["']([^"']+)["']`)
	matches := pattern.FindAllStringSubmatch(value, -1)
	result := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			result = append(result, strings.TrimSpace(match[1]))
		}
	}
	return result
}

func parseSetupPackageDir(value string) []string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "=") {
		value = strings.TrimSpace(strings.TrimPrefix(value, "="))
	}
	if value == "" {
		return nil
	}
	return []string{strings.Trim(value, "\"'")}
}

func splitConfigRoots(value string) []string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, ",", " ")
	fields := strings.Fields(value)
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.Trim(field, "\"'")
		if field != "" {
			result = append(result, field)
		}
	}
	return result
}

func parseQuoted(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) < 2 || (value[0] != '\'' && value[0] != '"') || value[len(value)-1] != value[0] {
		return "", fmt.Errorf("expected a quoted string")
	}
	if value[0] == '\'' {
		return value[1 : len(value)-1], nil
	}
	parsed, err := strconv.Unquote(value)
	if err != nil {
		return "", fmt.Errorf("invalid quoted string: %w", err)
	}
	return parsed, nil
}

func splitDelimited(value string, delimiter byte) ([]string, error) {
	parts := []string{}
	start := 0
	quote := byte(0)
	depth := 0
	for index := 0; index < len(value); index++ {
		switch value[index] {
		case '\'', '"':
			if quote == 0 {
				quote = value[index]
			} else if quote == value[index] && (index == 0 || value[index-1] != '\\') {
				quote = 0
			}
		case '[', '{':
			if quote == 0 {
				depth++
			}
		case ']', '}':
			if quote == 0 {
				depth--
			}
		case ',':
			if value[index] == delimiter && quote == 0 && depth == 0 {
				if strings.TrimSpace(value[start:index]) == "" {
					return nil, fmt.Errorf("empty array item")
				}
				parts = append(parts, value[start:index])
				start = index + 1
			}
		}
	}
	if quote != 0 || depth != 0 {
		return nil, fmt.Errorf("unbalanced value")
	}
	if strings.TrimSpace(value[start:]) != "" {
		return append(parts, value[start:]), nil
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty array item")
	}
	return parts, nil
}

func stripTOMLComment(value string) string {
	quote := byte(0)
	for index := 0; index < len(value); index++ {
		switch value[index] {
		case '\'', '"':
			if quote == 0 {
				quote = value[index]
			} else if quote == value[index] && (index == 0 || value[index-1] != '\\') {
				quote = 0
			}
		case '#':
			if quote == 0 {
				return value[:index]
			}
		}
	}
	return value
}

func splitINIAssignment(value string) (string, string, bool) {
	if index := strings.IndexAny(value, "=:"); index >= 0 {
		return value[:index], value[index+1:], true
	}
	return "", "", false
}

func stripINIComment(value string) string {
	for _, marker := range []string{" #", " ;"} {
		if index := strings.Index(value, marker); index >= 0 {
			return value[:index]
		}
	}
	return value
}

func normalizeConfigKey(value string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(value)), "-", "_")
}

func firstPythonVersion(value string) string {
	if match := pythonVersionPattern.FindStringSubmatch(value); len(match) > 1 {
		return match[1]
	}
	return ""
}

func configDiagnostic(code, path string, line int, message string) analysis.Diagnostic {
	diagnostic := analysis.Diagnostic{Code: code, Severity: "warning", Message: message, Path: path, Recoverable: true}
	if line > 0 {
		diagnostic.Message = fmt.Sprintf("%s (line %d)", message, line)
	}
	return diagnostic
}

func filepathJoin(root, relative string) string {
	return filepath.Join(root, filepath.FromSlash(relative))
}
