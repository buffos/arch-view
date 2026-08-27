package pyanalyzer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

func ensurePackage(values map[string]*packageObservation, qualified string) *packageObservation {
	value := values[qualified]
	if value == nil {
		value = &packageObservation{Qualified: qualified, SourceIDs: map[string]struct{}{}, Paths: map[string]struct{}{}, Tags: map[string]struct{}{}}
		values[qualified] = value
	}
	return value
}

func ensureModule(values map[string]*moduleObservation, qualified string) *moduleObservation {
	value := values[qualified]
	if value == nil {
		value = &moduleObservation{Qualified: qualified, SourceIDs: map[string]struct{}{}, Paths: map[string]struct{}{}, Tags: map[string]struct{}{}}
		values[qualified] = value
	}
	return value
}

func addFileTags(tags map[string]struct{}, file fileObservation) {
	if file.IsStub {
		tags["stub"] = struct{}{}
	}
	if file.IsTest {
		tags["test"] = struct{}{}
	}
}

func parentPackages(qualified, _ string) []string {
	parts := strings.Split(qualified, ".")
	if len(parts) > 0 {
		parts = parts[:len(parts)-1]
	}
	result := make([]string, 0, len(parts))
	for index := range parts {
		if parts[index] == "" || parts[index] == "__root__" {
			continue
		}
		result = append(result, strings.Join(parts[:index+1], "."))
	}
	return result
}

func qualifiedName(sourceRoot, filePath, name string) (string, string) {
	relative, err := filepath.Rel(sourceRoot, filepath.Dir(filePath))
	if err != nil {
		return "", ""
	}
	parts := []string{}
	if relative != "." {
		for _, part := range strings.Split(filepath.ToSlash(relative), "/") {
			if part != "" && part != "." {
				parts = append(parts, part)
			}
		}
	}
	base := strings.TrimSuffix(name, filepath.Ext(name))
	if base == "__init__" {
		if len(parts) == 0 {
			return "__root__", "package"
		}
		return strings.Join(parts, "."), "package"
	}
	parts = append(parts, base)
	return strings.Join(parts, "."), "module"
}

func moduleID(kind, qualified string) string {
	return "py:" + kind + ":" + qualified
}

func stableID(kind, value string) string {
	sum := sha256.Sum256([]byte(kind + "\x00" + value))
	return "py:" + kind + ":" + hex.EncodeToString(sum[:8])
}

func relativeProjectPath(root, filePath string) string {
	relative, err := filepath.Rel(root, filePath)
	if err != nil {
		return filepath.ToSlash(filepath.Clean(filePath))
	}
	return filepath.ToSlash(filepath.Clean(relative))
}

func isPythonFile(name string, includeStubs bool) bool {
	extension := strings.ToLower(filepath.Ext(name))
	return extension == ".py" || (includeStubs && extension == ".pyi")
}

func isExcludedPythonFile(relativePath, name string, includeTests bool, patterns []string) bool {
	if excludedFilePath(relativePath, patterns) {
		return true
	}
	if includeTests {
		return false
	}
	return isTestPath(relativePath, name)
}

func isTestPath(relativePath, name string) bool {
	base := strings.ToLower(name)
	stem := strings.TrimSuffix(strings.TrimSuffix(base, ".pyi"), ".py")
	if base == "conftest.py" || base == "conftest.pyi" || base == "test.py" || strings.HasPrefix(stem, "test_") || strings.HasSuffix(stem, "_test") {
		return true
	}
	for _, segment := range strings.Split(strings.ToLower(filepath.ToSlash(relativePath)), "/") {
		if segment == "test" || segment == "tests" || segment == "testing" {
			return true
		}
	}
	return false
}

var defaultExcludedDirectories = map[string]struct{}{
	".git": {}, ".hg": {}, ".svn": {}, ".cache": {}, "__pycache__": {},
	".mypy_cache": {}, ".pytest_cache": {}, ".tox": {}, ".nox": {},
	"build": {}, "cache": {}, "dist": {}, "generated": {}, "out": {},
	"target": {}, "tmp": {}, "vendor": {}, "external": {}, "node_modules": {},
	".venv": {}, "venv": {}, "env": {},
}

func excludedDirectory(relativePath string, patterns []string) bool {
	clean := strings.Trim(filepath.ToSlash(filepath.Clean(relativePath)), "/")
	if clean == "" || clean == "." {
		return false
	}
	for _, segment := range strings.Split(clean, "/") {
		if _, excluded := defaultExcludedDirectories[strings.ToLower(segment)]; excluded {
			return true
		}
	}
	return excludedFilePath(clean, patterns)
}

func excludedFilePath(relativePath string, patterns []string) bool {
	clean := filepath.ToSlash(filepath.Clean(relativePath))
	for _, pattern := range patterns {
		pattern = strings.TrimPrefix(filepath.ToSlash(filepath.Clean(pattern)), "./")
		if pattern == "" || pattern == "." {
			continue
		}
		if clean == strings.TrimSuffix(pattern, "/**") {
			return true
		}
		if matched, _ := path.Match(pattern, clean); matched {
			return true
		}
		if strings.HasSuffix(pattern, "/**") && strings.HasPrefix(clean, strings.TrimSuffix(pattern, "/**")+"/") {
			return true
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

func safeFileWithinRoot(projectRoot, filePath string) bool {
	if !pathWithin(projectRoot, filePath) {
		return false
	}
	return resolvedPathWithin(projectRoot, filePath)
}

func unreadableDiagnostic(root, filePath string, err error) analysis.Diagnostic {
	return analysis.Diagnostic{
		Code:        "python_unreadable_file",
		Severity:    "error",
		Message:     fmt.Sprintf("Python source path could not be read: %v", err),
		Path:        relativeProjectPath(root, filePath),
		Recoverable: true,
	}
}

func conflictingLayoutDiagnostic(path string, first, second fileObservation) analysis.Diagnostic {
	return analysis.Diagnostic{
		Code:        "python_conflicting_layout",
		Severity:    "warning",
		Message:     "The same repository file was reached through source roots with conflicting qualified names; the first deterministic observation was retained.",
		Path:        path,
		Recoverable: true,
		Metadata: map[string]any{
			"first_qualified_name":  first.Qualified,
			"second_qualified_name": second.Qualified,
			"first_source_root":     first.SourceRoot,
			"second_source_root":    second.SourceRoot,
		},
	}
}

func conflictingObservationDiagnostic(kind, qualified, firstPath, secondPath string) analysis.Diagnostic {
	return analysis.Diagnostic{
		Code:        "python_conflicting_layout",
		Severity:    "warning",
		Message:     fmt.Sprintf("Multiple source files provide the same Python %s qualified name; all evidence was retained.", kind),
		Subject:     qualified,
		Recoverable: true,
		Metadata: map[string]any{
			"kind":           kind,
			"qualified_name": qualified,
			"first_path":     firstPath,
			"second_path":    secondPath,
		},
	}
}

func firstDifferentPath(paths map[string]struct{}, current string) string {
	values := sortedKeys(paths)
	for _, value := range values {
		if value != current {
			return value
		}
	}
	return ""
}

func addProjectVersion(metadata map[string]any, version string) {
	if version != "" {
		metadata["python_version"] = version
	}
}

func hierarchy(qualified string) []string {
	if qualified == "" {
		return []string{}
	}
	return strings.Split(qualified, ".")
}

func displayName(qualified string) string {
	parts := strings.Split(qualified, ".")
	return parts[len(parts)-1]
}

func sortedPackageNames(values map[string]*packageObservation) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func sortedModuleNames(values map[string]*moduleObservation) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func sortedKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func sortedSources(values map[string]analysis.SourceReference) []analysis.SourceReference {
	result := make([]analysis.SourceReference, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func sortDiagnostics(values []analysis.Diagnostic) {
	sort.Slice(values, func(i, j int) bool {
		return diagnosticSortKey(values[i]) < diagnosticSortKey(values[j])
	})
}

func diagnosticSortKey(value analysis.Diagnostic) string {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%s\x00%s\x00%s\x00%s", value.Code, value.Path, value.Subject, value.Message)
	}
	return string(data)
}

func optionBool(options analysis.EffectiveOptions, name string) bool {
	value, _ := options.Values[name].(bool)
	return value
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
