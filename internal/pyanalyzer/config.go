package pyanalyzer

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

type configuration struct {
	SourceRoots   []string
	PythonVersion string
	Diagnostics   []analysis.Diagnostic
}

func readConfiguration(root, boundary string) configuration {
	path := filepathJoin(root, boundary)
	if !safeProjectFile(root, boundary) {
		return configuration{Diagnostics: []analysis.Diagnostic{configDiagnostic("python_configuration_outside_root", boundary, 0, "Python project configuration resolves outside the selected project root and was ignored.")}}
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return configuration{Diagnostics: []analysis.Diagnostic{{
			Code:        "python_configuration_unreadable",
			Severity:    "warning",
			Message:     fmt.Sprintf("Python project configuration %q could not be read: %v", boundary, err),
			Path:        boundary,
			Recoverable: true,
		}}}
	}
	switch boundary {
	case "pyproject.toml":
		return parsePyProject(string(content), boundary)
	case "setup.cfg":
		return parseSetupCFG(string(content), boundary)
	case "setup.py":
		return parseSetupPy(string(content), boundary)
	default:
		return configuration{}
	}
}

func parsePyProject(content, boundary string) configuration {
	result := configuration{}
	section := ""
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	var pending strings.Builder
	var pendingKey, pendingSection string
	for lineNumber := 0; lineNumber < len(lines); lineNumber++ {
		line := stripTOMLComment(lines[lineNumber])
		trimmed := strings.TrimSpace(line)
		if pendingKey != "" {
			if pending.Len() > 0 {
				pending.WriteByte('\n')
			}
			pending.WriteString(trimmed)
			if balancedTOMLValue(pending.String()) {
				parseTOMLValue(&result, pendingSection, pendingKey, pending.String(), boundary)
				pending.Reset()
				pendingKey = ""
				pendingSection = ""
			}
			continue
		}
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "[") {
			if !strings.HasSuffix(trimmed, "]") {
				result.Diagnostics = append(result.Diagnostics, configDiagnostic("python_configuration_invalid", boundary, lineNumber+1, "Python project configuration contains an unterminated table header."))
				continue
			}
			section = strings.TrimSpace(trimmed[1 : len(trimmed)-1])
			continue
		}
		key, value, ok := splitTOMLAssignment(trimmed)
		if !ok {
			// Unknown TOML constructs are intentionally ignored. A line that
			// looks like a relevant assignment is diagnosed below instead of
			// making unrelated valid build metadata unusable.
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if !isRelevantTOMLKey(section, key) {
			continue
		}
		if !balancedTOMLValue(value) {
			pendingKey, pendingSection = key, section
			pending.WriteString(value)
			continue
		}
		parseTOMLValue(&result, section, key, value, boundary)
	}
	if pendingKey != "" {
		result.Diagnostics = append(result.Diagnostics, configDiagnostic("python_configuration_invalid", boundary, len(lines), "Python project configuration contains an unterminated value."))
	}
	return result
}

func parseTOMLValue(result *configuration, section, key, value, boundary string) {
	canonicalKey := strings.ToLower(strings.TrimSpace(key))
	switch {
	case canonicalKey == "source_roots" || canonicalKey == "source-roots":
		values, err := parseTOMLStrings(value)
		if err != nil {
			result.Diagnostics = append(result.Diagnostics, configDiagnostic("python_configuration_invalid", boundary, 0, fmt.Sprintf("Python source_roots is invalid: %v", err)))
			return
		}
		result.SourceRoots = append(result.SourceRoots, values...)
	case section == "tool.setuptools.packages.find" && canonicalKey == "where":
		values, err := parseTOMLStrings(value)
		if err != nil {
			result.Diagnostics = append(result.Diagnostics, configDiagnostic("python_configuration_invalid", boundary, 0, fmt.Sprintf("Python setuptools package roots are invalid: %v", err)))
			return
		}
		result.SourceRoots = append(result.SourceRoots, values...)
	case (section == "tool.setuptools" || section == "tool.setuptools.package-dir") && (canonicalKey == "package-dir" || canonicalKey == "package_dir" || canonicalKey == "\"\""):
		values := parseTOMLPackageRoots(value)
		if len(values) == 0 {
			result.Diagnostics = append(result.Diagnostics, configDiagnostic("python_configuration_invalid", boundary, 0, "Python setuptools package-dir does not contain a usable root."))
			return
		}
		result.SourceRoots = append(result.SourceRoots, values...)
	case section == "tool.poetry" && canonicalKey == "packages":
		result.SourceRoots = append(result.SourceRoots, parsePoetryRoots(value)...)
	case section == "project" && canonicalKey == "requires-python":
		if version := firstPythonVersion(value); version != "" {
			result.PythonVersion = version
		}
	}
}

func parseSetupCFG(content, boundary string) configuration {
	result := configuration{}
	section := ""
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	lastKey := ""
	lastSection := ""
	for lineNumber := 0; lineNumber < len(lines); lineNumber++ {
		raw := strings.TrimRight(lines[lineNumber], " \t")
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			section = strings.ToLower(strings.TrimSpace(trimmed[1 : len(trimmed)-1]))
			lastKey, lastSection = "", ""
			continue
		}
		if len(raw) > 0 && (raw[0] == ' ' || raw[0] == '\t') && lastKey != "" {
			if lastSection == "options" && normalizeConfigKey(lastKey) == "package_dir" {
				result.SourceRoots = append(result.SourceRoots, parseSetupPackageDir(trimmed)...)
			}
			continue
		}
		key, value, ok := splitINIAssignment(trimmed)
		if !ok {
			result.Diagnostics = append(result.Diagnostics, configDiagnostic("python_configuration_invalid", boundary, lineNumber+1, "Python setup.cfg contains a setting without a key/value separator."))
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(stripINIComment(value))
		lastKey, lastSection = key, section
		switch {
		case section == "options.packages.find" && normalizeConfigKey(key) == "where":
			result.SourceRoots = append(result.SourceRoots, splitConfigRoots(value)...)
		case section == "options" && normalizeConfigKey(key) == "package_dir":
			result.SourceRoots = append(result.SourceRoots, parseSetupPackageDir(value)...)
		case section == "metadata" && normalizeConfigKey(key) == "python_requires":
			if version := firstPythonVersion(value); version != "" {
				result.PythonVersion = version
			}
		}
	}
	return result
}

func parseSetupPy(content, boundary string) configuration {
	result := configuration{}
	if issue := validatePythonSyntax(content); issue != nil {
		result.Diagnostics = append(result.Diagnostics, configDiagnostic("python_configuration_invalid", boundary, issue.Line, "Python setup.py could not be parsed as static configuration data."))
	}
	packageDirPattern := regexp.MustCompile(`(?s)package_dir\s*=\s*\{.*?["']{2}\s*:\s*["']([^"']+)["']`)
	for _, match := range packageDirPattern.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			result.SourceRoots = append(result.SourceRoots, strings.TrimSpace(match[1]))
		}
	}
	findPattern := regexp.MustCompile(`(?s)find_(?:namespace_)?packages\s*\((.*?)\)`)
	wherePattern := regexp.MustCompile(`["']?where["']?\s*=\s*["']([^"']+)["']`)
	for _, match := range findPattern.FindAllStringSubmatch(content, -1) {
		if len(match) <= 1 {
			continue
		}
		if where := wherePattern.FindStringSubmatch(match[1]); len(where) > 1 {
			result.SourceRoots = append(result.SourceRoots, strings.TrimSpace(where[1]))
		}
	}
	if version := firstPythonVersion(content); version != "" {
		result.PythonVersion = version
	}
	_ = boundary // setup.py is treated as opaque data; it is never executed.
	return result
}
