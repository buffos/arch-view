package rustanalyzer

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

type cargoManifest struct {
	packageName           string
	edition               string
	libName               string
	libPath               string
	binTargets            []rustTargetSpec
	workspaceMembers      []string
	workspaceExclude      []string
	workspaceDependencies map[string]dependencySpec
	features              map[string]struct{}
	dependencies          []CargoDependency
	diagnostics           []analysis.Diagnostic
}

type rustTargetSpec struct {
	Name string
	Path string
}

type dependencySpec struct {
	Version         string
	Path            string
	Registry        string
	Optional        bool
	Workspace       bool
	DefaultFeatures *bool
	Features        []string
}

type cargoSection struct {
	Name           string
	TargetCond     string
	BinIndex       int
	DependencyName string
}

func readCargoManifest(path string) (cargoManifest, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cargoManifest{}, analysis.NewHostError(analysis.ErrUnsupportedProject, "Cargo.toml could not be found", map[string]any{"path": path})
		}
		return cargoManifest{}, analysis.WrapHostError(analysis.ErrUnreadableProject, "Cargo.toml could not be read", err, map[string]any{"path": path})
	}
	return parseCargoManifest(string(content), path), nil
}

func parseCargoManifest(content, path string) cargoManifest {
	manifest := cargoManifest{workspaceDependencies: make(map[string]dependencySpec), features: make(map[string]struct{})}
	section := cargoSection{Name: "root", BinIndex: -1}
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		rawLine := scanner.Text()
		line := strings.TrimSpace(stripTomlComment(rawLine))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			header := strings.TrimSpace(line[1 : len(line)-1])
			if strings.HasPrefix(header, "[") && strings.HasSuffix(header, "]") {
				header = strings.TrimSpace(header[1 : len(header)-1])
			}
			section = parseCargoSection(header, len(manifest.binTargets))
			if section.Name == "bin" {
				manifest.binTargets = append(manifest.binTargets, rustTargetSpec{})
				section.BinIndex = len(manifest.binTargets) - 1
			}
			continue
		}
		key, value, ok := splitTomlAssignment(line)
		if !ok {
			manifest.diagnostics = append(manifest.diagnostics, cargoDiagnostic(path, lineNumber, 1, "rust_manifest_syntax", "Cargo.toml contains a line that is not a supported data assignment; it was ignored.", true))
			continue
		}
		complete := true
		for !tomlValueComplete(value) {
			next, scanErr := nextCargoLine(scanner, &lineNumber)
			if scanErr != nil {
				manifest.diagnostics = append(manifest.diagnostics, cargoDiagnostic(path, lineNumber, 1, "rust_manifest_syntax", "Cargo.toml contains an unterminated data value; the partial value was ignored.", true))
				complete = false
				break
			}
			value += "\n" + stripTomlComment(next)
		}
		if !complete {
			continue
		}
		parsed, parseErr := parseTomlValue(strings.TrimSpace(value))
		if parseErr != nil {
			manifest.diagnostics = append(manifest.diagnostics, cargoDiagnostic(path, lineNumber, tomlColumn(rawLine, key), "rust_manifest_value", fmt.Sprintf("Cargo.toml value %q could not be interpreted as data: %v", key, parseErr), true))
			continue
		}
		source := analysis.SourceReference{ID: fmt.Sprintf("rust:manifest:%d:%s", lineNumber, key), Path: path, Start: &analysis.Position{Line: lineNumber, Column: tomlColumn(rawLine, key)}, Symbol: key, Kind: "manifest"}
		applyCargoValue(&manifest, section, key, parsed, source)
	}
	if err := scanner.Err(); err != nil {
		manifest.diagnostics = append(manifest.diagnostics, cargoDiagnostic(path, lineNumber, 1, "rust_manifest_read", fmt.Sprintf("Cargo.toml could not be scanned: %v", err), true))
	}
	return manifest
}

func parseCargoSection(header string, binCount int) cargoSection {
	if header == "package" || header == "lib" || header == "workspace" || header == "dependencies" || header == "dev-dependencies" || header == "build-dependencies" || header == "workspace.dependencies" || header == "features" {
		return cargoSection{Name: header, BinIndex: -1}
	}
	if strings.HasPrefix(header, "bin") && strings.TrimSpace(header) == "bin" {
		return cargoSection{Name: "bin", BinIndex: binCount}
	}
	parts := splitCargoHeaderParts(header)
	if len(parts) == 2 && isCargoDependencyKind(parts[0]) {
		return cargoSection{Name: parts[0], DependencyName: unquoteTomlString(parts[1]), BinIndex: -1}
	}
	if strings.HasPrefix(header, "target.") {
		rest := strings.TrimPrefix(header, "target.")
		targetParts := splitCargoHeaderParts(rest)
		if len(targetParts) >= 2 {
			kindIndex := len(targetParts) - 1
			dependencyName := ""
			if len(targetParts) >= 3 && isCargoDependencyKind(targetParts[len(targetParts)-2]) {
				kindIndex = len(targetParts) - 2
				dependencyName = unquoteTomlString(targetParts[len(targetParts)-1])
			}
			kind := targetParts[kindIndex]
			if isCargoDependencyKind(kind) {
				condition := strings.Join(targetParts[:kindIndex], ".")
				condition = unquoteTomlString(strings.TrimSpace(condition))
				return cargoSection{Name: kind, TargetCond: condition, DependencyName: dependencyName, BinIndex: -1}
			}
		}
	}
	return cargoSection{Name: "unknown", BinIndex: -1}
}

func isCargoDependencyKind(value string) bool {
	return value == "dependencies" || value == "dev-dependencies" || value == "build-dependencies"
}

func splitCargoHeaderParts(value string) []string {
	parts := make([]string, 0, 4)
	start := 0
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
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		case '.':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(value[start:index]))
				start = index + 1
			}
		}
	}
	parts = append(parts, strings.TrimSpace(value[start:]))
	return parts
}

func applyCargoValue(manifest *cargoManifest, section cargoSection, key string, value any, source analysis.SourceReference) {
	stringValue, _ := value.(string)
	switch section.Name {
	case "package":
		switch key {
		case "name":
			manifest.packageName = stringValue
		case "edition":
			manifest.edition = stringValue
		}
	case "lib":
		switch key {
		case "name":
			manifest.libName = stringValue
		case "path":
			manifest.libPath = stringValue
		}
	case "bin":
		if section.BinIndex >= 0 && section.BinIndex < len(manifest.binTargets) {
			switch key {
			case "name":
				manifest.binTargets[section.BinIndex].Name = stringValue
			case "path":
				manifest.binTargets[section.BinIndex].Path = stringValue
			}
		}
	case "workspace":
		switch key {
		case "members":
			manifest.workspaceMembers = stringValues(value)
		case "exclude":
			manifest.workspaceExclude = stringValues(value)
		}
	case "workspace.dependencies":
		manifest.workspaceDependencies[key] = dependencyFromValue(value)
	case "features":
		manifest.features[key] = struct{}{}
	case "dependencies", "dev-dependencies", "build-dependencies":
		if section.DependencyName != "" {
			applyCargoDependencyTableValue(manifest, section, key, value, source)
			return
		}
		dependency := dependencyFromValue(value)
		manifest.dependencies = append(manifest.dependencies, CargoDependency{
			Name:            key,
			Kind:            section.Name,
			Version:         dependency.Version,
			Path:            dependency.Path,
			Registry:        dependency.Registry,
			Target:          section.TargetCond,
			Optional:        dependency.Optional,
			Workspace:       dependency.Workspace,
			DefaultFeatures: dependency.DefaultFeatures,
			Features:        append([]string(nil), dependency.Features...),
			Source:          source,
		})
	}
}

func applyCargoDependencyTableValue(manifest *cargoManifest, section cargoSection, key string, value any, source analysis.SourceReference) {
	index := -1
	for candidate := range manifest.dependencies {
		dependency := manifest.dependencies[candidate]
		if dependency.Name == section.DependencyName && dependency.Kind == section.Name && dependency.Target == section.TargetCond {
			index = candidate
			break
		}
	}
	if index < 0 {
		manifest.dependencies = append(manifest.dependencies, CargoDependency{Name: section.DependencyName, Kind: section.Name, Target: section.TargetCond, Source: source})
		index = len(manifest.dependencies) - 1
	}
	dependency := &manifest.dependencies[index]
	if dependency.Source.ID == "" {
		dependency.Source = source
	}
	switch key {
	case "version":
		dependency.Version, _ = value.(string)
	case "path":
		dependency.Path, _ = value.(string)
	case "registry":
		dependency.Registry, _ = value.(string)
	case "optional":
		dependency.Optional, _ = value.(bool)
	case "workspace":
		dependency.Workspace, _ = value.(bool)
	case "default-features":
		if defaultFeatures, ok := value.(bool); ok {
			dependency.DefaultFeatures = &defaultFeatures
		}
	case "features":
		dependency.Features = stringValues(value)
	}
}

func dependencyFromValue(value any) dependencySpec {
	switch typed := value.(type) {
	case string:
		return dependencySpec{Version: typed}
	case map[string]any:
		result := dependencySpec{}
		result.Version, _ = typed["version"].(string)
		result.Path, _ = typed["path"].(string)
		result.Registry, _ = typed["registry"].(string)
		result.Optional, _ = typed["optional"].(bool)
		result.Workspace, _ = typed["workspace"].(bool)
		if defaultFeatures, ok := typed["default-features"].(bool); ok {
			result.DefaultFeatures = &defaultFeatures
		}
		result.Features = stringValues(typed["features"])
		return result
	default:
		return dependencySpec{}
	}
}

func splitTomlAssignment(line string) (string, string, bool) {
	quote := byte(0)
	depth := 0
	for index := 0; index < len(line); index++ {
		char := line[index]
		if quote != 0 {
			if char == quote && (index == 0 || line[index-1] != '\\') {
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
			if depth > 0 {
				depth--
			}
		case '=':
			if depth == 0 {
				key := strings.TrimSpace(line[:index])
				if key == "" {
					return "", "", false
				}
				return strings.Trim(key, "\"' "), strings.TrimSpace(line[index+1:]), true
			}
		}
	}
	return "", "", false
}

func parseTomlValue(value string) (any, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("empty value")
	}
	if value[0] == '"' || value[0] == '\'' {
		return unquoteTomlString(value), nil
	}
	if value == "true" || value == "false" {
		return value == "true", nil
	}
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		inner := strings.TrimSpace(value[1 : len(value)-1])
		if inner == "" {
			return []any{}, nil
		}
		parts := splitTomlList(inner)
		result := make([]any, 0, len(parts))
		for index, part := range parts {
			if strings.TrimSpace(part) == "" && index == len(parts)-1 {
				continue
			}
			parsed, err := parseTomlValue(part)
			if err != nil {
				return nil, err
			}
			result = append(result, parsed)
		}
		return result, nil
	}
	if strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}") {
		inner := strings.TrimSpace(value[1 : len(value)-1])
		result := make(map[string]any)
		if inner == "" {
			return result, nil
		}
		for _, part := range splitTomlList(inner) {
			key, item, ok := splitTomlAssignment(part)
			if !ok {
				return nil, fmt.Errorf("inline table item %q is not an assignment", part)
			}
			parsed, err := parseTomlValue(item)
			if err != nil {
				return nil, err
			}
			result[key] = parsed
		}
		return result, nil
	}
	// Cargo dependency versions and other metadata are retained as strings;
	// numeric TOML values are not needed for the static boundary.
	return strings.TrimSpace(value), nil
}

func splitTomlList(value string) []string {
	parts := make([]string, 0)
	start := 0
	depth := 0
	quote := byte(0)
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
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(value[start:index]))
				start = index + 1
			}
		}
	}
	parts = append(parts, strings.TrimSpace(value[start:]))
	return parts
}

func stringValues(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				result = append(result, text)
			}
		}
		return result
	case string:
		return []string{typed}
	default:
		return nil
	}
}

func unquoteTomlString(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		if decoded, err := strconv.Unquote(value); err == nil {
			return decoded
		}
	}
	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		return strings.ReplaceAll(value[1:len(value)-1], "''", "'")
	}
	return strings.Trim(value, "\"'")
}

func stripTomlComment(line string) string {
	quote := byte(0)
	for index := 0; index < len(line); index++ {
		char := line[index]
		if quote != 0 {
			if char == quote && (index == 0 || line[index-1] != '\\') {
				quote = 0
			}
			continue
		}
		if char == '\'' || char == '"' {
			quote = char
			continue
		}
		if char == '#' {
			return line[:index]
		}
	}
	return line
}

func tomlValueComplete(value string) bool {
	depth := 0
	quote := byte(0)
	for index := 0; index < len(value); index++ {
		char := value[index]
		if quote != 0 {
			if char == quote && (index == 0 || value[index-1] != '\\') {
				quote = 0
			}
			continue
		}
		if char == '\'' || char == '"' {
			quote = char
			continue
		}
		switch char {
		case '[', '{':
			depth++
		case ']', '}':
			if depth > 0 {
				depth--
			}
		}
	}
	return quote == 0 && depth == 0
}

func nextCargoLine(scanner *bufio.Scanner, lineNumber *int) (string, error) {
	if !scanner.Scan() {
		return "", scanner.Err()
	}
	*lineNumber = *lineNumber + 1
	return scanner.Text(), nil
}

func tomlColumn(line, key string) int {
	index := strings.Index(line, key)
	if index < 0 {
		return 1
	}
	return index + 1
}

func cargoDiagnostic(path string, line, column int, code, message string, recoverable bool) analysis.Diagnostic {
	return analysis.Diagnostic{Code: code, Severity: "warning", Message: message, Path: path, Location: &analysis.Position{Line: line, Column: column}, Recoverable: recoverable}
}
