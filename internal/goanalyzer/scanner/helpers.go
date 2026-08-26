package scanner

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/build"
	"go/scanner"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

var defaultExcludedRootDirectories = map[string]struct{}{
	".cache":   {},
	"bin":      {},
	"build":    {},
	"cache":    {},
	"dist":     {},
	"external": {},
	"out":      {},
	"target":   {},
	"tmp":      {},
}

var defaultExcludedNestedDirectories = map[string]struct{}{
	".git":   {},
	"vendor": {},
}

func matchesBuildContext(filePath string, tags []string) (bool, []string, error) {
	buildContext := build.Default
	buildContext.BuildTags = append([]string(nil), tags...)
	matched, err := buildContext.MatchFile(filepath.Dir(filePath), filepath.Base(filePath))
	return matched, buildConstraintLines(filePath), err
}

func buildConstraintLines(filePath string) []string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	var constraints []string
	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineCount := 0
	for scanner.Scan() {
		lineCount++
		if lineCount > 100 {
			break
		}
		trimmed := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(trimmed, "//go:build "):
			constraints = append(constraints, strings.TrimSpace(strings.TrimPrefix(trimmed, "//go:build ")))
		case strings.HasPrefix(trimmed, "// +build "):
			constraints = append(constraints, strings.TrimSpace(strings.TrimPrefix(trimmed, "// +build ")))
		case trimmed == "", strings.HasPrefix(trimmed, "//"):
			continue
		default:
			sort.Strings(constraints)
			return constraints
		}
	}
	sort.Strings(constraints)
	return constraints
}

func parseDiagnostic(err error, relativePath string) analysis.Diagnostic {
	location := &analysis.Position{Line: 1, Column: 1}
	switch errors := err.(type) {
	case scanner.ErrorList:
		if len(errors) > 0 {
			location = parseErrorPosition(errors[0].Pos, location)
		}
	case *scanner.ErrorList:
		if errors != nil && len(*errors) > 0 {
			location = parseErrorPosition((*errors)[0].Pos, location)
		}
	}
	return analysis.Diagnostic{
		Code:        "go_parse_error",
		Severity:    "error",
		Message:     fmt.Sprintf("Go source could not be parsed: %v", err),
		Path:        relativePath,
		Location:    location,
		Recoverable: true,
	}
}

func parseErrorPosition(position token.Position, fallback *analysis.Position) *analysis.Position {
	if position.Line <= 0 || position.Column <= 0 {
		return fallback
	}
	return &analysis.Position{Line: position.Line, Column: position.Column}
}

func isGeneratedSource(content []byte) bool {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for index := 0; scanner.Scan() && index < 20; index++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			if strings.Contains(line, "Code generated") && strings.Contains(line, "DO NOT EDIT") {
				return true
			}
			continue
		}
		break
	}
	return false
}

func excludedDirectory(relativePath string, patterns []string) bool {
	clean := filepath.ToSlash(filepath.Clean(relativePath))
	segments := strings.Split(clean, "/")
	if len(segments) > 0 {
		if _, excluded := defaultExcludedRootDirectories[segments[0]]; excluded {
			return true
		}
	}
	for _, segment := range segments {
		if _, excluded := defaultExcludedNestedDirectories[segment]; excluded {
			return true
		}
	}
	return matchesAnyExclude(clean, patterns)
}

func matchesAnyExclude(relativePath string, patterns []string) bool {
	clean := filepath.ToSlash(filepath.Clean(relativePath))
	for _, pattern := range patterns {
		pattern = strings.TrimPrefix(filepath.ToSlash(filepath.Clean(pattern)), "./")
		if pattern == "" || pattern == "." {
			continue
		}
		if matched, _ := path.Match(pattern, clean); matched {
			return true
		}
		if strings.HasSuffix(pattern, "/**") {
			prefix := strings.TrimSuffix(pattern, "/**")
			if clean == prefix || strings.HasPrefix(clean, prefix+"/") {
				return true
			}
		}
		if strings.HasPrefix(pattern, "**/") {
			short := strings.TrimPrefix(pattern, "**/")
			if matched, _ := path.Match(short, path.Base(clean)); matched {
				return true
			}
		}
	}
	return false
}

func HasGoFile(directory string) bool {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".go" {
			return true
		}
	}
	return false
}

func goImportPath(modulePath, relativeDirectory string) string {
	if relativeDirectory == "." || relativeDirectory == "" {
		return modulePath
	}
	return strings.TrimSuffix(modulePath, "/") + "/" + filepath.ToSlash(relativeDirectory)
}

func PackageID(modulePath, relativeDirectory string) string {
	return "go:" + goImportPath(modulePath, relativeDirectory)
}

func Hierarchy(relativeDirectory string) []string {
	if relativeDirectory == "." || relativeDirectory == "" {
		return []string{}
	}
	segments := strings.Split(filepath.ToSlash(relativeDirectory), "/")
	result := make([]string, 0, len(segments))
	for _, segment := range segments {
		if segment != "" && segment != "." {
			result = append(result, segment)
		}
	}
	return result
}

func relativeProjectPath(root, filePath string) string {
	relative, err := filepath.Rel(root, filePath)
	if err != nil {
		return filepath.ToSlash(filepath.Clean(filePath))
	}
	return filepath.ToSlash(filepath.Clean(relative))
}

func StableID(kind string, parts ...string) string {
	payload := kind + "\x00" + strings.Join(parts, "\x00")
	sum := sha256.Sum256([]byte(payload))
	return "go:" + kind + ":" + hex.EncodeToString(sum[:8])
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func optionBool(values map[string]any, name string) bool {
	value, _ := values[name].(bool)
	return value
}

func optionStrings(values map[string]any, name string) []string {
	value, _ := values[name].([]string)
	return append([]string(nil), value...)
}
