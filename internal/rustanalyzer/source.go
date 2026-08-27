package rustanalyzer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/buffo/arch-view/internal/analysis"
)

type discoveryResult struct {
	Modules          map[string]*rustModule
	Uses             []rustUseObservation
	SourceReferences map[string]analysis.SourceReference
	Diagnostics      []analysis.Diagnostic
	Visited          map[string]struct{}
}

type rustModule struct {
	ID                 string
	Path               []string
	Kind               string
	Name               string
	FilePath           string
	ModuleDir          string
	SourceReferenceIDs map[string]struct{}
	Tags               map[string]struct{}
	CfgConditions      map[string]struct{}
	DeclaredModules    map[string]struct{}
	Paths              map[string]struct{}
	TargetKinds        map[string]struct{}
	MacroAttributes    map[string]struct{}
	Generated          bool
}

type rustUseObservation struct {
	FromModuleID  string
	Kind          string
	Path          string
	Alias         string
	ImportedName  string
	CfgConditions []string
	Source        analysis.SourceReference
}

type rustModuleDeclaration struct {
	Parent        *rustModule
	Name          string
	PathOverride  string
	CfgConditions []string
	Source        analysis.SourceReference
}

type rustAttribute struct {
	Name      string
	Value     string
	Tokens    []rustToken
	Source    rustToken
	Cfg       string
	Path      string
	TestOnly  bool
	Uncertain bool
}

type rustToken struct {
	Kind      rustTokenKind
	Text      string
	Value     string
	Start     int
	End       int
	Line      int
	Column    int
	EndLine   int
	EndColumn int
}

type rustTokenKind uint8

const (
	rustTokenIdent rustTokenKind = iota
	rustTokenString
	rustTokenPunct
)

type rustLexIssue struct {
	Offset  int
	Line    int
	Column  int
	Message string
}

func Discover(ctx context.Context, project Project, options analysis.EffectiveOptions) (discoveryResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return discoveryResult{}, err
	}
	discovery := discoveryResult{
		Modules:          make(map[string]*rustModule),
		SourceReferences: make(map[string]analysis.SourceReference),
		Visited:          make(map[string]struct{}),
		Diagnostics:      append([]analysis.Diagnostic(nil), project.Diagnostics...),
	}
	if len(project.Targets) == 0 {
		discovery.Diagnostics = append(discovery.Diagnostics, analysis.Diagnostic{
			Code:        "rust_no_targets",
			Severity:    "warning",
			Message:     "The selected Cargo crate has no readable source targets.",
			Path:        project.RelativeManifestPath,
			Recoverable: true,
		})
	}
	for _, target := range project.Targets {
		if err := ctx.Err(); err != nil {
			return discoveryResult{}, err
		}
		root := ensureRustModule(discovery.Modules, project, nil, "crate", target.AbsolutePath, []string{}, target.Kind)
		root.TargetKinds[target.Kind] = struct{}{}
		if err := discoverRustFile(ctx, project, &discovery, root, target.AbsolutePath, target.Kind); err != nil {
			if err == context.Canceled || err == context.DeadlineExceeded || ctx.Err() != nil {
				return discoveryResult{}, err
			}
			return discoveryResult{}, err
		}
	}
	return discovery, nil
}

func discoverRustFile(ctx context.Context, project Project, discovery *discoveryResult, module *rustModule, filePath, targetKind string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !safeRustFile(project, filePath) {
		discovery.Diagnostics = append(discovery.Diagnostics, analysis.Diagnostic{
			Code:        "rust_path_outside_root",
			Severity:    "warning",
			Message:     "Rust source path resolves outside the selected project and was ignored.",
			Path:        projectRelative(project.Root, filePath),
			Recoverable: true,
		})
		return nil
	}
	if rustExcluded(project, filePath) {
		return nil
	}
	visitKey := module.ID + "\x00" + filepath.Clean(filePath)
	if _, visited := discovery.Visited[visitKey]; visited {
		return nil
	}
	discovery.Visited[visitKey] = struct{}{}
	content, err := os.ReadFile(filePath)
	if err != nil {
		discovery.Diagnostics = append(discovery.Diagnostics, analysis.Diagnostic{
			Code:        "rust_unreadable_source",
			Severity:    "error",
			Message:     fmt.Sprintf("Rust source file could not be read: %v", err),
			Path:        projectRelative(project.Root, filePath),
			Recoverable: true,
		})
		return nil
	}
	relative := projectRelative(project.Root, filePath)
	if isRustGeneratedSource(content) {
		if len(module.Path) > 0 {
			delete(discovery.Modules, strings.Join(module.Path, "::"))
		}
		discovery.Diagnostics = append(discovery.Diagnostics, analysis.Diagnostic{
			Code:        "rust_generated_source",
			Severity:    "warning",
			Message:     "Rust source appears to be generated and was excluded from static discovery.",
			Path:        relative,
			Recoverable: true,
		})
		return nil
	}
	fileSource := rustSourceReference(relative, "file", module.DisplayName(project.PackageName), nil, nil)
	discovery.SourceReferences[fileSource.ID] = fileSource
	module.SourceReferenceIDs[fileSource.ID] = struct{}{}
	module.Paths[relative] = struct{}{}
	if targetKind != "" {
		module.TargetKinds[targetKind] = struct{}{}
	}

	tokens, issues := lexRust(string(content))
	for _, issue := range issues {
		discovery.Diagnostics = append(discovery.Diagnostics, analysis.Diagnostic{
			Code:        "rust_lex_error",
			Severity:    "error",
			Message:     issue.Message,
			Path:        relative,
			Location:    &analysis.Position{Line: issue.Line, Column: issue.Column},
			Recoverable: true,
		})
	}
	if issue := unmatchedRustDelimiter(tokens); issue != nil {
		discovery.Diagnostics = append(discovery.Diagnostics, analysis.Diagnostic{
			Code:        "rust_syntax_error",
			Severity:    "error",
			Message:     "Rust source contains an unmatched delimiter; recoverable top-level observations were retained.",
			Path:        relative,
			Location:    &analysis.Position{Line: issue.Line, Column: issue.Column},
			Recoverable: true,
		})
	}
	declarations, uses, parseDiagnostics := parseRustItems(tokens, project, discovery.Modules, discovery.SourceReferences, module, fileSource, project.IncludeTests, targetKind)
	discovery.Uses = append(discovery.Uses, uses...)
	discovery.Diagnostics = append(discovery.Diagnostics, parseDiagnostics...)
	for _, declaration := range declarations {
		if err := ctx.Err(); err != nil {
			return err
		}
		if declaration.Source.ID != "" {
			discovery.SourceReferences[declaration.Source.ID] = declaration.Source
			declaration.Parent.SourceReferenceIDs[declaration.Source.ID] = struct{}{}
		}
		childPath, resolveErr := resolveRustModulePath(project, declaration)
		if resolveErr != nil {
			discovery.Diagnostics = append(discovery.Diagnostics, rustModuleDiagnostic(declaration.Source, resolveErr.Error(), "rust_module_path_invalid"))
			continue
		}
		if childPath == "" {
			discovery.Diagnostics = append(discovery.Diagnostics, rustModuleDiagnostic(declaration.Source, fmt.Sprintf("Rust module %q has no readable source file.", declaration.Name), "rust_module_source_missing"))
			continue
		}
		child := ensureRustModule(discovery.Modules, project, declaration.Parent, "file_module", childPath, append(append([]string(nil), declaration.Parent.Path...), declaration.Name), targetKind)
		child.SourceReferenceIDs[declaration.Source.ID] = struct{}{}
		if len(declaration.CfgConditions) > 0 {
			for _, condition := range declaration.CfgConditions {
				child.CfgConditions[condition] = struct{}{}
			}
		}
		if err := discoverRustFile(ctx, project, discovery, child, childPath, targetKind); err != nil {
			return err
		}
	}
	return nil
}

func ensureRustModule(modules map[string]*rustModule, project Project, parent *rustModule, kind, filePath string, modulePath []string, targetKind string) *rustModule {
	key := strings.Join(modulePath, "::")
	module := modules[key]
	if module == nil {
		name := project.PackageName
		if len(modulePath) > 0 {
			name = modulePath[len(modulePath)-1]
		}
		module = &rustModule{
			ID:                 rustModuleID(project.PackageName, modulePath),
			Path:               append([]string(nil), modulePath...),
			Kind:               kind,
			Name:               name,
			FilePath:           filePath,
			ModuleDir:          rustModuleDir(filePath, name, len(modulePath) == 0),
			SourceReferenceIDs: make(map[string]struct{}),
			Tags:               make(map[string]struct{}),
			CfgConditions:      make(map[string]struct{}),
			DeclaredModules:    make(map[string]struct{}),
			Paths:              make(map[string]struct{}),
			TargetKinds:        make(map[string]struct{}),
			MacroAttributes:    make(map[string]struct{}),
		}
		modules[key] = module
	} else if module.Kind == "file_module" && kind == "inline_module" {
		module.Kind = kind
	}
	if module.FilePath == "" {
		module.FilePath = filePath
		module.ModuleDir = rustModuleDir(filePath, module.Name, len(module.Path) == 0)
	}
	if targetKind != "" {
		module.TargetKinds[targetKind] = struct{}{}
	}
	if parent != nil {
		parent.DeclaredModules[module.Name] = struct{}{}
	}
	return module
}

func rustModuleDir(filePath, name string, root bool) string {
	directory := filepath.Dir(filePath)
	if root || strings.EqualFold(filepath.Base(filePath), "mod.rs") {
		return directory
	}
	return filepath.Join(directory, name)
}

func resolveRustModulePath(project Project, declaration rustModuleDeclaration) (string, error) {
	base := declaration.Parent.ModuleDir
	if len(declaration.Parent.Path) == 0 && declaration.Source.Path != "" {
		// A crate root can be shared by lib/bin targets, whose source files may
		// live in different directories. Resolve each root declaration from the
		// file that declared it rather than from the first target's module dir.
		base = filepath.Dir(fileSourcePathAbsolute(project, declaration.Source.Path))
	}
	if declaration.PathOverride != "" {
		candidate := filepath.Clean(filepath.Join(base, filepath.FromSlash(declaration.PathOverride)))
		if !pathWithin(project.CrateRoot, candidate) || !resolvedPathWithin(project.Root, candidate) {
			return "", fmt.Errorf("rust #[path] module %q escapes the selected crate root", declaration.PathOverride)
		}
		if fileExists(candidate) {
			return candidate, nil
		}
		return "", nil
	}
	for _, candidate := range []string{
		filepath.Join(base, declaration.Name+".rs"),
		filepath.Join(base, declaration.Name, "mod.rs"),
	} {
		if !pathWithin(project.CrateRoot, candidate) || !resolvedPathWithin(project.Root, candidate) {
			continue
		}
		if fileExists(candidate) {
			return candidate, nil
		}
	}
	return "", nil
}

func parseRustItems(tokens []rustToken, project Project, modules map[string]*rustModule, sourceReferences map[string]analysis.SourceReference, module *rustModule, fileSource analysis.SourceReference, includeTests bool, targetKind string) ([]rustModuleDeclaration, []rustUseObservation, []analysis.Diagnostic) {
	declarations := make([]rustModuleDeclaration, 0)
	uses := make([]rustUseObservation, 0)
	diagnostics := make([]analysis.Diagnostic, 0)
	var parse func(int, int, *rustModule)
	parse = func(start, end int, current *rustModule) {
		for index := start; index < end; {
			attrs, next := parseRustAttributes(tokens, index, end)
			if len(attrs) > 0 {
				for _, attr := range attrs {
					if attr.Name != "cfg" && attr.Name != "cfg_attr" && attr.Name != "path" && attr.Name != "derive" && attr.Name != "allow" && attr.Name != "warn" && attr.Name != "deny" && attr.Name != "doc" {
						current.MacroAttributes[attr.Name] = struct{}{}
						diagnostics = append(diagnostics, rustAttributeDiagnostic(fileSource.Path, attr.Source, attr.Name))
					}
					if attr.Name == "derive" {
						current.MacroAttributes[attr.Name] = struct{}{}
						diagnostics = append(diagnostics, rustAttributeDiagnostic(fileSource.Path, attr.Source, attr.Name))
					}
				}
			}
			visibility, next := rustVisibility(tokens, next, end)
			if next >= end {
				break
			}
			cfgConditions := rustCfgConditions(attrs)
			testOnly := rustAttributesTestOnly(attrs)
			if testOnly && !includeTests {
				index = rustItemEnd(tokens, next, end)
				continue
			}
			if len(cfgConditions) > 0 {
				if !rustCfgKnown(cfgConditions) {
					diagnostics = append(diagnostics, rustCfgDiagnostic(fileSource.Path, tokens[next], cfgConditions))
				}
			}
			keyword := tokenText(tokens[next])
			switch keyword {
			case "mod":
				nameIndex := next + 1
				if nameIndex >= end || tokens[nameIndex].Kind != rustTokenIdent {
					diagnostics = append(diagnostics, rustParseDiagnostic(fileSource.Path, tokens[next], "rust_module_syntax", "Rust module declaration has no readable module name."))
					index = rustItemEnd(tokens, next, end)
					continue
				}
				name := normalizeRustIdent(tokens[nameIndex].Text)
				declarationEnd := rustItemEnd(tokens, next, end)
				open := next + 2
				if open < end && tokenText(tokens[open]) == "{" {
					close := matchingRustToken(tokens, open, "{", "}", end)
					if close < 0 {
						close = end - 1
					}
					declarationSource := rustTokenSource(fileSource.Path, "module_declaration", tokens[next], tokens[minRustIndex(close, end-1)], current.DisplayName(project.PackageName)+"::"+name)
					child := ensureRustModule(modules, project, current, "inline_module", fileSourcePathAbsolute(project, fileSource.Path), append(append([]string(nil), current.Path...), name), targetKind)
					child.SourceReferenceIDs[declarationSource.ID] = struct{}{}
					child.SourceReferenceIDs[fileSource.ID] = struct{}{}
					child.Paths[fileSource.Path] = struct{}{}
					child.Tags["inline"] = struct{}{}
					current.SourceReferenceIDs[declarationSource.ID] = struct{}{}
					sourceReferences[declarationSource.ID] = declarationSource
					for _, condition := range cfgConditions {
						child.CfgConditions[condition] = struct{}{}
					}
					parse(open+1, close, child)
					index = close + 1
					if index < end && tokenText(tokens[index]) == ";" {
						index++
					}
					continue
				}
				pathOverride := ""
				for _, attr := range attrs {
					if attr.Name == "path" {
						pathOverride = attr.Path
					}
				}
				endToken := tokens[minRustIndex(declarationEnd-1, end-1)]
				declarations = append(declarations, rustModuleDeclaration{
					Parent:        current,
					Name:          name,
					PathOverride:  pathOverride,
					CfgConditions: cfgConditions,
					Source:        rustTokenSource(fileSource.Path, "module_declaration", tokens[next], endToken, current.DisplayName(project.PackageName)+"::"+name),
				})
				index = declarationEnd
			case "use":
				statementEnd := rustItemEnd(tokens, next, end)
				endIndex := statementEnd - 1
				if endIndex >= end || tokenText(tokens[endIndex]) != ";" {
					endIndex = end - 1
				}
				leaves := parseRustUseTree(tokens[next+1 : minRustIndex(endIndex, end)])
				for leafIndex, leaf := range leaves {
					if leaf.Path == "" {
						continue
					}
					useSource := rustTokenSource(fileSource.Path, mapUseKind(visibility), tokens[next], tokens[minRustIndex(endIndex, end-1)], leaf.Path+useAliasSuffix(leaf.Alias)+fmt.Sprintf("#%d", leafIndex))
					uses = append(uses, rustUseObservation{FromModuleID: current.ID, Kind: mapUseKind(visibility), Path: leaf.Path, Alias: leaf.Alias, ImportedName: useImportedName(leaf.Path), CfgConditions: append([]string(nil), cfgConditions...), Source: useSource})
				}
				index = statementEnd
			case "extern":
				if next+2 < end && tokenText(tokens[next+1]) == "crate" && tokens[next+2].Kind == rustTokenIdent {
					statementEnd := rustItemEnd(tokens, next, end)
					endIndex := minRustIndex(statementEnd-1, end-1)
					name := normalizeRustIdent(tokens[next+2].Text)
					uses = append(uses, rustUseObservation{FromModuleID: current.ID, Kind: "dependency", Path: name, ImportedName: name, CfgConditions: append([]string(nil), cfgConditions...), Source: rustTokenSource(fileSource.Path, "extern_crate", tokens[next], tokens[endIndex], name)})
					index = statementEnd
				} else {
					index = rustItemEnd(tokens, next, end)
				}
			default:
				itemEnd := rustItemEnd(tokens, next, end)
				if keyword == "macro_rules" || containsMacroInvocation(tokens, next, itemEnd) {
					current.MacroAttributes[keyword] = struct{}{}
					diagnostics = append(diagnostics, rustMacroDiagnostic(fileSource.Path, tokens[next], keyword))
				}
				index = itemEnd
			}
		}
	}
	parse(0, len(tokens), module)
	return declarations, uses, diagnostics
}

func fileSourcePathAbsolute(project Project, relative string) string {
	return filepath.Join(project.Root, filepath.FromSlash(relative))
}

type rustUseLeaf struct {
	Path  string
	Alias string
}

type rustUseParser struct {
	tokens []rustToken
	index  int
}

func parseRustUseTree(tokens []rustToken) []rustUseLeaf {
	parser := &rustUseParser{tokens: tokens}
	return parser.parseGroup(nil)
}

func (parser *rustUseParser) parseGroup(prefix []string) []rustUseLeaf {
	result := make([]rustUseLeaf, 0)
	for parser.index < len(parser.tokens) {
		if tokenText(parser.tokens[parser.index]) == "}" {
			parser.index++
			break
		}
		segments := append([]string(nil), prefix...)
		for parser.index < len(parser.tokens) {
			token := parser.tokens[parser.index]
			text := tokenText(token)
			switch {
			case text == "::":
				parser.index++
			case text == "{", text == ",", text == "}":
				goto pathDone
			case text == "as":
				parser.index++
				alias := ""
				if parser.index < len(parser.tokens) && parser.tokens[parser.index].Kind == rustTokenIdent {
					alias = normalizeRustIdent(parser.tokens[parser.index].Text)
					parser.index++
				}
				for parser.index < len(parser.tokens) && tokenText(parser.tokens[parser.index]) != "," && tokenText(parser.tokens[parser.index]) != "}" {
					parser.index++
				}
				if len(segments) > 0 {
					result = append(result, rustUseLeaf{Path: strings.Join(segments, "::"), Alias: alias})
				}
				goto entryDone
			case token.Kind == rustTokenIdent || text == "*":
				if text == "self" && len(segments) > 0 {
					parser.index++
					if parser.index < len(parser.tokens) && tokenText(parser.tokens[parser.index]) == "::" {
						parser.index++
					}
					continue
				}
				segments = append(segments, normalizeRustIdent(text))
				parser.index++
			default:
				parser.index++
			}
		}
	pathDone:
		if parser.index < len(parser.tokens) && tokenText(parser.tokens[parser.index]) == "{" {
			parser.index++
			result = append(result, parser.parseGroup(segments)...)
			goto entryDone
		}
		if len(segments) > 0 {
			if segments[len(segments)-1] == "self" {
				segments = segments[:len(segments)-1]
			}
			if len(segments) > 0 {
				result = append(result, rustUseLeaf{Path: strings.Join(segments, "::")})
			}
		}
	entryDone:
		if parser.index < len(parser.tokens) && tokenText(parser.tokens[parser.index]) == "," {
			parser.index++
			continue
		}
		if parser.index < len(parser.tokens) && tokenText(parser.tokens[parser.index]) == "}" {
			parser.index++
			break
		}
		if parser.index < len(parser.tokens) {
			parser.index++
		}
	}
	return result
}

func parseRustAttributes(tokens []rustToken, start, end int) ([]rustAttribute, int) {
	attributes := make([]rustAttribute, 0)
	index := start
	for index+1 < end && tokenText(tokens[index]) == "#" {
		bracket := index + 1
		if tokenText(tokens[bracket]) == "!" {
			bracket++
		}
		if bracket >= end || tokenText(tokens[bracket]) != "[" {
			break
		}
		close := matchingRustToken(tokens, bracket, "[", "]", end)
		if close < 0 {
			break
		}
		// Inner attributes (#![...]) apply to the enclosing crate/module,
		// rather than the next item. Consume them here so they cannot be
		// mistaken for a code item and hide the remainder of the file. Their
		// module-wide semantics remain conservative until cfg evaluation is
		// available.
		if tokenText(tokens[index+1]) == "!" {
			index = close + 1
			continue
		}
		body := tokens[bracket+1 : close]
		name := ""
		for _, token := range body {
			if token.Kind == rustTokenIdent {
				name = normalizeRustIdent(token.Text)
				break
			}
		}
		attribute := rustAttribute{Name: name, Tokens: append([]rustToken(nil), body...), Source: tokens[index]}
		attribute.Value = rustTokensText(body)
		if name == "cfg" || name == "cfg_attr" {
			attribute.Cfg = rustAttributeArgument(body)
			attribute.TestOnly = rustCfgTestOnly(attribute.Cfg)
			attribute.Uncertain = attribute.Cfg != "" && !rustCfgKnown([]string{attribute.Cfg})
		}
		if name == "path" {
			for _, token := range body {
				if token.Kind == rustTokenString {
					attribute.Path = token.Value
					break
				}
			}
		}
		attributes = append(attributes, attribute)
		index = close + 1
	}
	return attributes, index
}

func rustVisibility(tokens []rustToken, start, end int) (bool, int) {
	if start >= end || tokenText(tokens[start]) != "pub" {
		return false, start
	}
	start++
	if start < end && tokenText(tokens[start]) == "(" {
		close := matchingRustToken(tokens, start, "(", ")", end)
		if close >= 0 {
			start = close + 1
		}
	}
	return true, start
}

func rustCfgConditions(attributes []rustAttribute) []string {
	conditions := make([]string, 0)
	for _, attr := range attributes {
		if attr.Name == "cfg" || attr.Name == "cfg_attr" {
			if attr.Cfg != "" {
				conditions = appendUniqueString(conditions, attr.Cfg)
			}
		}
	}
	sort.Strings(conditions)
	return conditions
}

func rustAttributesTestOnly(attributes []rustAttribute) bool {
	for _, attr := range attributes {
		if attr.TestOnly {
			return true
		}
	}
	return false
}

func rustCfgTestOnly(expression string) bool {
	expression = strings.TrimSpace(strings.ToLower(expression))
	if expression == "test" {
		return true
	}
	open := strings.IndexByte(expression, '(')
	if open <= 0 || !strings.HasSuffix(expression, ")") {
		return false
	}
	operator := strings.TrimSpace(expression[:open])
	args := splitRustCfgArguments(expression[open+1 : len(expression)-1])
	if len(args) == 0 {
		return false
	}
	switch operator {
	case "all":
		for _, argument := range args {
			if rustCfgTestOnly(argument) {
				return true
			}
		}
		return false
	case "any":
		for _, argument := range args {
			if !rustCfgTestOnly(argument) {
				return false
			}
		}
		return true
	case "not":
		return false
	default:
		return false
	}
}

func splitRustCfgArguments(value string) []string {
	arguments := make([]string, 0)
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
		if char == '\'' || char == '"' {
			quote = char
			continue
		}
		switch char {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				arguments = append(arguments, strings.TrimSpace(value[start:index]))
				start = index + 1
			}
		}
	}
	if tail := strings.TrimSpace(value[start:]); tail != "" {
		arguments = append(arguments, tail)
	}
	return arguments
}

func rustCfgKnown(conditions []string) bool {
	for _, condition := range conditions {
		lower := strings.ToLower(strings.TrimSpace(condition))
		if lower == "test" || lower == "true" || lower == "false" {
			continue
		}
		return false
	}
	return true
}

func rustItemEnd(tokens []rustToken, start, end int) int {
	depth := 0
	for index := start; index < end; index++ {
		switch tokenText(tokens[index]) {
		case "(", "[":
			depth++
		case ")", "]":
			if depth > 0 {
				depth--
			}
		case "{":
			if depth == 0 {
				close := matchingRustToken(tokens, index, "{", "}", end)
				if close >= 0 {
					itemEnd := close + 1
					if itemEnd < end && tokenText(tokens[itemEnd]) == ";" {
						itemEnd++
					}
					return itemEnd
				}
				return end
			}
		case ";":
			if depth == 0 {
				return index + 1
			}
		}
	}
	return end
}

func matchingRustToken(tokens []rustToken, start int, open, close string, end int) int {
	depth := 0
	for index := start; index < end; index++ {
		text := tokenText(tokens[index])
		switch text {
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}

func containsMacroInvocation(tokens []rustToken, start, end int) bool {
	for index := start; index+1 < end; index++ {
		if tokenText(tokens[index+1]) == "!" && tokens[index].Kind == rustTokenIdent {
			return true
		}
	}
	return false
}

func mapUseKind(public bool) string {
	if public {
		return "pub_use"
	}
	return "use"
}

func useAliasSuffix(alias string) string {
	if alias == "" {
		return ""
	}
	return " as " + alias
}

func useImportedName(value string) string {
	value = strings.TrimSuffix(value, "::*")
	if index := strings.LastIndex(value, "::"); index >= 0 {
		return value[index+2:]
	}
	return value
}

func rustSourceReference(relative, kind, symbol string, start, end *analysis.Position) analysis.SourceReference {
	id := stableID("source", kind, relative, symbol, positionKey(start), positionKey(end))
	return analysis.SourceReference{ID: id, Path: relative, Start: start, End: end, Symbol: symbol, Kind: kind}
}

func rustTokenSource(relative, kind string, start, end rustToken, symbol string) analysis.SourceReference {
	return rustSourceReference(relative, kind, symbol, &analysis.Position{Line: start.Line, Column: start.Column}, &analysis.Position{Line: end.EndLine, Column: end.EndColumn})
}

func positionKey(position *analysis.Position) string {
	if position == nil {
		return ""
	}
	return fmt.Sprintf("%d:%d", position.Line, position.Column)
}

func stableID(kind string, parts ...string) string {
	payload := kind + "\x00" + strings.Join(parts, "\x00")
	sum := sha256.Sum256([]byte(payload))
	return "rust:" + kind + ":" + hex.EncodeToString(sum[:8])
}

func rustModuleID(packageName string, modulePath []string) string {
	if len(modulePath) == 0 {
		return "rust:crate:" + normalizeRustIdent(packageName)
	}
	return "rust:module:" + normalizeRustIdent(packageName) + "::" + strings.Join(modulePath, "::")
}

func (module *rustModule) DisplayName(packageName string) string {
	if len(module.Path) == 0 {
		return packageName
	}
	return packageName + "::" + strings.Join(module.Path, "::")
}

func normalizeRustIdent(value string) string {
	return strings.TrimPrefix(value, "r#")
}

func tokenText(token rustToken) string {
	// Parser control flow must use the lexical spelling. Returning a string's
	// decoded value would make delimiters or keywords inside a literal look
	// like syntax to item and delimiter matching.
	return token.Text
}

func rustTokensText(tokens []rustToken) string {
	values := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if token.Kind == rustTokenString {
			values = append(values, strconvQuoteRust(token.Value))
		} else {
			values = append(values, token.Text)
		}
	}
	return strings.Join(values, " ")
}

func strconvQuoteRust(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}

func rustAttributeArgument(tokens []rustToken) string {
	start := -1
	end := -1
	for index, token := range tokens {
		if tokenText(token) == "(" {
			start = index + 1
			end = matchingRustToken(tokens, index, "(", ")", len(tokens))
			break
		}
	}
	if start < 0 || end < start {
		return ""
	}
	return strings.TrimSpace(rustTokensText(tokens[start:end]))
}

func rustModuleDiagnostic(source analysis.SourceReference, message, code string) analysis.Diagnostic {
	return analysis.Diagnostic{Code: code, Severity: "warning", Message: message, Path: source.Path, Location: source.Start, Recoverable: true, Metadata: map[string]any{"source_reference_id": source.ID}}
}

func rustParseDiagnostic(path string, token rustToken, code, message string) analysis.Diagnostic {
	return analysis.Diagnostic{Code: code, Severity: "warning", Message: message, Path: path, Location: &analysis.Position{Line: token.Line, Column: token.Column}, Recoverable: true}
}

func rustCfgDiagnostic(path string, token rustToken, conditions []string) analysis.Diagnostic {
	return analysis.Diagnostic{Code: "rust_cfg_uncertain", Severity: "warning", Message: "Rust cfg conditions were retained as metadata but could not be evaluated statically.", Path: path, Location: &analysis.Position{Line: token.Line, Column: token.Column}, Recoverable: true, Metadata: map[string]any{"conditions": append([]string(nil), conditions...)}}
}

func rustAttributeDiagnostic(path string, token rustToken, name string) analysis.Diagnostic {
	return analysis.Diagnostic{Code: "rust_macro_attribute", Severity: "warning", Message: fmt.Sprintf("Rust attribute %q was not expanded; generated declarations remain outside the static result.", name), Path: path, Location: &analysis.Position{Line: token.Line, Column: token.Column}, Recoverable: true, Metadata: map[string]any{"attribute": name}}
}

func rustMacroDiagnostic(path string, token rustToken, name string) analysis.Diagnostic {
	return analysis.Diagnostic{Code: "rust_macro_uncertain", Severity: "warning", Message: fmt.Sprintf("Rust macro %q was not expanded; generated declarations were not inferred.", name), Path: path, Location: &analysis.Position{Line: token.Line, Column: token.Column}, Recoverable: true, Metadata: map[string]any{"macro": name}}
}

func safeRustFile(project Project, filePath string) bool {
	return pathWithin(project.CrateRoot, filePath) && resolvedPathWithin(project.Root, filePath) && fileExists(filePath)
}

func rustExcluded(project Project, filePath string) bool {
	relativeProject := projectRelative(project.Root, filePath)
	relativeCrate := projectRelative(project.CrateRoot, filePath)
	for _, value := range []string{relativeProject, relativeCrate} {
		if matchesRustDefaultExclusion(value, project.IncludeTests, project.IncludeExamples) || matchesAnyRustExclude(value, project.ExcludePatterns) {
			return true
		}
	}
	return false
}

func matchesRustDefaultExclusion(relative string, includeTests, includeExamples bool) bool {
	clean := filepath.ToSlash(filepath.Clean(relative))
	segments := strings.Split(clean, "/")
	for _, segment := range segments {
		switch strings.ToLower(segment) {
		case ".git", "target", "build", "cache", ".cache", "dist", "out", "tmp", "vendor", "external", "generated":
			return true
		case "tests":
			if !includeTests {
				return true
			}
		case "examples", "benches":
			if !includeExamples {
				return true
			}
		}
	}
	if !includeTests && strings.HasSuffix(strings.ToLower(filepath.Base(clean)), "_test.rs") {
		return true
	}
	return false
}

func matchesAnyRustExclude(relative string, patterns []string) bool {
	clean := filepath.ToSlash(filepath.Clean(relative))
	for _, pattern := range patterns {
		pattern = filepath.ToSlash(filepath.Clean(strings.TrimSpace(pattern)))
		pattern = strings.TrimPrefix(pattern, "./")
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
			if matched, _ := path.Match(strings.TrimPrefix(pattern, "**/"), path.Base(clean)); matched {
				return true
			}
		}
	}
	return false
}

func appendUniqueString(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func minRustIndex(value, maximum int) int {
	if value < 0 {
		return 0
	}
	if value > maximum {
		return maximum
	}
	return value
}

func unmatchedRustDelimiter(tokens []rustToken) *rustToken {
	stack := make([]rustToken, 0)
	pairs := map[string]string{"(": ")", "[": "]", "{": "}"}
	for _, token := range tokens {
		text := tokenText(token)
		if _, ok := pairs[text]; ok {
			stack = append(stack, token)
			continue
		}
		if text == ")" || text == "]" || text == "}" {
			if len(stack) == 0 || pairs[tokenText(stack[len(stack)-1])] != text {
				copyToken := token
				return &copyToken
			}
			stack = stack[:len(stack)-1]
		}
	}
	if len(stack) > 0 {
		copyToken := stack[len(stack)-1]
		return &copyToken
	}
	return nil
}

func lexRust(content string) ([]rustToken, []rustLexIssue) {
	tokens := make([]rustToken, 0)
	issues := make([]rustLexIssue, 0)
	line, column := 1, 1
	for index := 0; index < len(content); {
		start := index
		startLine, startColumn := line, column
		char := content[index]
		if char == '\r' || char == '\n' || char == ' ' || char == '\t' || char == '\f' {
			index, line, column = advanceRustCursor(content, index, line, column)
			continue
		}
		if char == '/' && index+1 < len(content) && content[index+1] == '/' {
			for index < len(content) && content[index] != '\n' && content[index] != '\r' {
				index, line, column = advanceRustCursor(content, index, line, column)
			}
			continue
		}
		if char == '/' && index+1 < len(content) && content[index+1] == '*' {
			depth := 1
			index, line, column = advanceRustCursor(content, index, line, column)
			index, line, column = advanceRustCursor(content, index, line, column)
			for index < len(content) && depth > 0 {
				if index+1 < len(content) && content[index] == '/' && content[index+1] == '*' {
					depth++
					index, line, column = advanceRustCursor(content, index, line, column)
					index, line, column = advanceRustCursor(content, index, line, column)
					continue
				}
				if index+1 < len(content) && content[index] == '*' && content[index+1] == '/' {
					depth--
					index, line, column = advanceRustCursor(content, index, line, column)
					index, line, column = advanceRustCursor(content, index, line, column)
					continue
				}
				index, line, column = advanceRustCursor(content, index, line, column)
			}
			if depth > 0 {
				issues = append(issues, rustLexIssue{Offset: start, Line: startLine, Column: startColumn, Message: "Rust source contains an unterminated block comment."})
			}
			continue
		}
		if rawPrefix, ok := rustRawStringPrefix(content[index:]); ok {
			value, next, endLine, endColumn, found := consumeRustRawString(content, index, rawPrefix, line, column)
			if !found {
				issues = append(issues, rustLexIssue{Offset: start, Line: startLine, Column: startColumn, Message: "Rust source contains an unterminated raw string."})
				index, line, column = next, endLine, endColumn
				continue
			}
			tokens = append(tokens, rustToken{Kind: rustTokenString, Text: content[start:next], Value: value, Start: start, End: next, Line: startLine, Column: startColumn, EndLine: endLine, EndColumn: endColumn})
			index, line, column = next, endLine, endColumn
			continue
		}
		if char == 'r' && index+2 < len(content) && content[index+1] == '#' && rustIdentifierStart(content, index+2) {
			index, line, column = advanceRustCursor(content, index, line, column)
			index, line, column = advanceRustCursor(content, index, line, column)
			index, line, column = advanceRustIdentifier(content, index, line, column)
			tokens = append(tokens, rustToken{Kind: rustTokenIdent, Text: content[start:index], Value: content[start:index], Start: start, End: index, Line: startLine, Column: startColumn, EndLine: line, EndColumn: column})
			continue
		}
		if char == '"' || (char == 'b' && index+1 < len(content) && content[index+1] == '"') {
			prefix := 0
			if char == 'b' {
				prefix = 1
			}
			value, next, endLine, endColumn, found := consumeRustQuotedString(content, index+prefix, line, column)
			if !found {
				issues = append(issues, rustLexIssue{Offset: start, Line: startLine, Column: startColumn, Message: "Rust source contains an unterminated string literal."})
				index, line, column = next, endLine, endColumn
				continue
			}
			tokens = append(tokens, rustToken{Kind: rustTokenString, Text: content[start:next], Value: value, Start: start, End: next, Line: startLine, Column: startColumn, EndLine: endLine, EndColumn: endColumn})
			index, line, column = next, endLine, endColumn
			continue
		}
		if char == '\'' || (char == 'b' && index+1 < len(content) && content[index+1] == '\'') {
			prefix := 0
			if char == 'b' {
				prefix = 1
			}
			value, next, endLine, endColumn, found := consumeRustCharLiteral(content, index+prefix, line, column)
			if found {
				tokens = append(tokens, rustToken{Kind: rustTokenString, Text: content[start:next], Value: value, Start: start, End: next, Line: startLine, Column: startColumn, EndLine: endLine, EndColumn: endColumn})
				index, line, column = next, endLine, endColumn
				continue
			}
		}
		if rustIdentifierStart(content, index) {
			index, line, column = advanceRustIdentifier(content, index, line, column)
			tokens = append(tokens, rustToken{Kind: rustTokenIdent, Text: content[start:index], Value: content[start:index], Start: start, End: index, Line: startLine, Column: startColumn, EndLine: line, EndColumn: column})
			continue
		}
		text := content[index : index+1]
		if index+1 < len(content) {
			for _, candidate := range []string{"::", "=>", "->", "..", "&&", "||", "==", "!=", "<=", ">="} {
				if strings.HasPrefix(content[index:], candidate) {
					text = candidate
					break
				}
			}
		}
		for range text {
			index, line, column = advanceRustCursor(content, index, line, column)
		}
		tokens = append(tokens, rustToken{Kind: rustTokenPunct, Text: text, Value: text, Start: start, End: index, Line: startLine, Column: startColumn, EndLine: line, EndColumn: column})
	}
	return tokens, issues
}

func rustIdentifierStart(content string, index int) bool {
	if index >= len(content) {
		return false
	}
	runeValue, _ := utf8.DecodeRuneInString(content[index:])
	return runeValue == '_' || unicode.IsLetter(runeValue) || runeValue >= utf8.RuneSelf
}

func isRustGeneratedSource(content []byte) bool {
	lines := strings.Split(string(content), "\n")
	if len(lines) > 20 {
		lines = lines[:20]
	}
	for _, line := range lines {
		lower := strings.ToLower(strings.TrimSpace(line))
		if strings.Contains(lower, "generated") && (strings.HasPrefix(lower, "//") || strings.HasPrefix(lower, "/*") || strings.HasPrefix(lower, "*") || strings.HasPrefix(lower, "#")) {
			return true
		}
	}
	return false
}

func advanceRustIdentifier(content string, index, line, column int) (int, int, int) {
	for index < len(content) {
		runeValue, size := utf8.DecodeRuneInString(content[index:])
		if runeValue != '_' && !unicode.IsLetter(runeValue) && !unicode.IsDigit(runeValue) && runeValue < utf8.RuneSelf {
			break
		}
		index, line, column = advanceRustCursor(content, index, line, column)
		if size == 0 {
			break
		}
	}
	return index, line, column
}

func advanceRustCursor(content string, index, line, column int) (int, int, int) {
	next, lineDelta, columnDelta := advanceRustPosition(content, index)
	if lineDelta > 0 {
		line += lineDelta
		column = columnDelta
	} else {
		column += columnDelta
	}
	return next, line, column
}

func advanceRustPosition(content string, index int) (int, int, int) {
	if index >= len(content) {
		return index, 0, 0
	}
	if content[index] == '\r' {
		index++
		if index < len(content) && content[index] == '\n' {
			index++
		}
		return index, 1, 1
	}
	if content[index] == '\n' {
		return index + 1, 1, 1
	}
	_, size := utf8.DecodeRuneInString(content[index:])
	if size == 0 {
		size = 1
	}
	return index + size, 0, 1
}

func rustRawStringPrefix(value string) (string, bool) {
	index := 0
	if strings.HasPrefix(value, "br") {
		index = 2
	} else if strings.HasPrefix(value, "r") {
		index = 1
	} else {
		return "", false
	}
	for index < len(value) && value[index] == '#' {
		index++
	}
	if index < len(value) && value[index] == '"' {
		return value[:index+1], true
	}
	return "", false
}

func consumeRustRawString(content string, start int, prefix string, line, column int) (string, int, int, int, bool) {
	hashCount := strings.Count(prefix, "#")
	openingLength := len(prefix)
	index := start + openingLength
	for index < len(content) {
		if content[index] == '"' && strings.HasPrefix(content[index+1:], strings.Repeat("#", hashCount)) {
			end := index + 1 + hashCount
			value := content[start+openingLength : index]
			endIndex, endLine, endColumn := advanceRustRange(content, start, end, line, column)
			return value, endIndex, endLine, endColumn, true
		}
		index++
	}
	endIndex, endLine, endColumn := advanceRustRange(content, start, len(content), line, column)
	return "", endIndex, endLine, endColumn, false
}

func consumeRustQuotedString(content string, quoteIndex, line, column int) (string, int, int, int, bool) {
	index := quoteIndex + 1
	for index < len(content) {
		if content[index] == '\\' {
			index += 2
			continue
		}
		if content[index] == '"' {
			end := index + 1
			value := content[quoteIndex+1 : index]
			if decoded, err := strconvUnquoteRust(content[quoteIndex:end]); err == nil {
				value = decoded
			}
			endIndex, endLine, endColumn := advanceRustRange(content, quoteIndex, end, line, column)
			return value, endIndex, endLine, endColumn, true
		}
		index++
	}
	endIndex, endLine, endColumn := advanceRustRange(content, quoteIndex, len(content), line, column)
	return "", endIndex, endLine, endColumn, false
}

func consumeRustCharLiteral(content string, quoteIndex, line, column int) (string, int, int, int, bool) {
	index := quoteIndex + 1
	if index >= len(content) {
		return "", index, line, column, false
	}
	if content[index] == '\\' {
		index += 2
	} else {
		_, size := utf8.DecodeRuneInString(content[index:])
		if size == 0 {
			size = 1
		}
		index += size
	}
	if index >= len(content) || content[index] != '\'' {
		// An apostrophe without a closing quote is commonly a lifetime, so
		// leave it for normal punctuation/identifier lexing.
		return "", index, line, column, false
	}
	end := index + 1
	value := content[quoteIndex+1 : index]
	if decoded, err := strconvUnquoteRust(content[quoteIndex:end]); err == nil {
		value = decoded
	}
	endIndex, endLine, endColumn := advanceRustRange(content, quoteIndex, end, line, column)
	return value, endIndex, endLine, endColumn, true
}

func strconvUnquoteRust(value string) (string, error) {
	if len(value) > 0 && value[0] == 'b' {
		value = value[1:]
	}
	return strconv.Unquote(value)
}

func advanceRustRange(content string, start, end, line, column int) (int, int, int) {
	currentLine, currentColumn := line, column
	index := start
	for index < end {
		index, currentLine, currentColumn = advanceRustCursor(content, index, currentLine, currentColumn)
	}
	return end, currentLine, currentColumn
}
