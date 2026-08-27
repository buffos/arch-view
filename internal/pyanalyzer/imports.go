package pyanalyzer

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/buffo/arch-view/internal/analysis"
)

type pythonImportObservation struct {
	FromModuleID    string
	Source          analysis.SourceReference
	Spelling        string
	Module          string
	ImportedName    string
	Alias           string
	RelativeLevel   int
	Kind            string
	Conditional     bool
	Condition       string
	DynamicFunction string
	DynamicTarget   string
}

type pythonTokenKind uint8

const (
	pythonTokenName pythonTokenKind = iota
	pythonTokenString
	pythonTokenPunctuation
	pythonTokenNewline
)

type pythonToken struct {
	Kind      pythonTokenKind
	Text      string
	Line      int
	Column    int
	EndLine   int
	EndColumn int
}

type pythonConditionFrame struct {
	Indent int
	Text   string
}

type pythonConditionState uint8

const (
	pythonConditionUnknown pythonConditionState = iota
	pythonConditionTrue
	pythonConditionFalse
)

type pythonModuleIndex struct {
	ByQualified map[string][]string
	ByID        map[string]analysis.ModuleObservation
	TopLevel    map[string]struct{}
	Ambiguous   map[string]bool
}

type pythonImportResolution struct {
	Name        string
	ModuleID    string
	Local       bool
	Ambiguous   bool
	Resolution  string
	Conditional bool
	Conditions  []string
}

type pythonReexportKey struct {
	Package string
	Name    string
}

type pythonReexport struct {
	ModuleID      string
	Ambiguous     bool
	Conditional   bool
	Conditions    []string
	Unconditional bool
}

type pythonStdlibRule struct {
	MinMinor int
	MaxMinor int
}

// The list is intentionally a static table. Using the host interpreter's
// import machinery would make analysis environment-dependent and would break
// the analyzer's read-only safety boundary.
var pythonStdlibModules = func() map[string]pythonStdlibRule {
	result := make(map[string]pythonStdlibRule)
	for _, name := range strings.Fields(`
		__future__ __main__ _thread abc aifc annotationlib antigravity argparse
		array asynchat ast asyncio atexit audioop base64
		bdb binascii bisect builtins bz2 calendar cgi cgitb cmath cmd code codecs
		codeop collections colorsys compileall compression concurrent configparser contextlib
		contextvars copy copyreg cProfile crypt csv ctypes curses csv dataclasses
		datetime dbm decimal difflib dis distutils doctest email encodings enum
		ensurepip errno faulthandler fcntl filecmp fileinput fnmatch fractions ftplib functools
		gc getopt getpass gettext glob graphlib grp gzip hashlib heapq hmac html
		http idlelib imaplib imghdr imp importlib inspect io ipaddress itertools json keyword
		lib2to3 linecache locale logging lzma mailbox mailcap marshal math mimetypes
		mmap modulefinder msilib msvcrt multiprocessing netrc nis nntplib numbers opcode operator optparse
		os ossaudiodev pathlib pdb pickle pickletools pipes pkgutil platform plistlib
		poplib posix pprint profile pstats pty pwd py_compile pyclbr pydoc pyexpat queue
		quopri random re readline reprlib resource rlcompleter runpy sched secrets
		select selectors shelve shlex shutil signal site smtpd smtplib sndhdr socket spwd
		socketserver sqlite3 ssl stat statistics string stringprep struct subprocess
		sunau symtable sys sysconfig syslog tabnanny tarfile telnetlib tempfile
		termios textwrap this threading time timeit tkinter token tokenize tomllib
		trace traceback tracemalloc tty turtle types typing unicodedata unittest
		urllib uu uuid venv warnings wave weakref webbrowser winreg winsound
		wsgiref xdrlib xml xmlrpc zipapp zipfile zipimport zlib zoneinfo
	`) {
		result[name] = pythonStdlibRule{}
	}
	result["contextvars"] = pythonStdlibRule{MinMinor: 7}
	result["annotationlib"] = pythonStdlibRule{MinMinor: 14}
	result["compression"] = pythonStdlibRule{MinMinor: 14}
	result["dataclasses"] = pythonStdlibRule{MinMinor: 7}
	result["graphlib"] = pythonStdlibRule{MinMinor: 9}
	result["ipaddress"] = pythonStdlibRule{MinMinor: 3}
	result["pathlib"] = pythonStdlibRule{MinMinor: 4}
	result["secrets"] = pythonStdlibRule{MinMinor: 6}
	result["statistics"] = pythonStdlibRule{MinMinor: 4}
	result["tomllib"] = pythonStdlibRule{MinMinor: 11}
	result["tracemalloc"] = pythonStdlibRule{MinMinor: 4}
	result["venv"] = pythonStdlibRule{MinMinor: 3}
	result["zoneinfo"] = pythonStdlibRule{MinMinor: 9}
	result["aifc"] = pythonStdlibRule{MaxMinor: 12}
	result["asynchat"] = pythonStdlibRule{MaxMinor: 11}
	result["asyncore"] = pythonStdlibRule{MaxMinor: 11}
	result["audioop"] = pythonStdlibRule{MaxMinor: 12}
	result["cgi"] = pythonStdlibRule{MaxMinor: 12}
	result["cgitb"] = pythonStdlibRule{MaxMinor: 12}
	result["chunk"] = pythonStdlibRule{MaxMinor: 12}
	result["crypt"] = pythonStdlibRule{MaxMinor: 12}
	result["distutils"] = pythonStdlibRule{MaxMinor: 11}
	result["imghdr"] = pythonStdlibRule{MaxMinor: 12}
	result["imp"] = pythonStdlibRule{MaxMinor: 11}
	result["lib2to3"] = pythonStdlibRule{MaxMinor: 12}
	result["mailcap"] = pythonStdlibRule{MaxMinor: 12}
	result["msilib"] = pythonStdlibRule{MaxMinor: 12}
	result["nis"] = pythonStdlibRule{MaxMinor: 12}
	result["nntplib"] = pythonStdlibRule{MaxMinor: 12}
	result["ossaudiodev"] = pythonStdlibRule{MaxMinor: 12}
	result["pipes"] = pythonStdlibRule{MaxMinor: 12}
	result["smtpd"] = pythonStdlibRule{MaxMinor: 11}
	result["sndhdr"] = pythonStdlibRule{MaxMinor: 12}
	result["spwd"] = pythonStdlibRule{MaxMinor: 12}
	result["sunau"] = pythonStdlibRule{MaxMinor: 12}
	result["telnetlib"] = pythonStdlibRule{MaxMinor: 12}
	result["uu"] = pythonStdlibRule{MaxMinor: 12}
	result["xdrlib"] = pythonStdlibRule{MaxMinor: 12}
	return result
}()

var pythonVersionConditionPattern = regexp.MustCompile(`^sys\.version_info\s*(>=|>|<=|<|==|!=)\s*\(\s*3\s*,\s*([0-9]+)\s*\)`)
var pythonVersionValuePattern = regexp.MustCompile(`^3\.([0-9]+)$`)

// extractPythonImports performs a conservative lexical pass. It recognizes
// import statements and import-like dynamic calls while deliberately avoiding
// a Python interpreter or an environment-assisted parser.
func extractPythonImports(path, content, fromModuleID string, packageInit bool) ([]pythonImportObservation, []analysis.Diagnostic) {
	tokens := lexPython(content)
	conditions := conditionalContexts(content)
	observations := make([]pythonImportObservation, 0)
	diagnostics := make([]analysis.Diagnostic, 0)
	dynamicAliases := make(map[string]struct{})
	statementStart := true
	bracketDepth := 0
	for index := 0; index < len(tokens); index++ {
		token := tokens[index]
		if token.Kind == pythonTokenNewline {
			if bracketDepth == 0 {
				statementStart = true
			}
			continue
		}

		if dynamic, next, ok := parseDynamicCall(tokens, index, path, fromModuleID, conditions, dynamicAliases); ok {
			observations = append(observations, dynamic)
			index = next
			statementStart = false
			continue
		}

		if statementStart && token.Kind == pythonTokenName && token.Text == "import" {
			parsed, next := parseImportStatement(tokens, index, path, fromModuleID, conditions)
			observations = append(observations, parsed...)
			registerPythonDynamicAliases(dynamicAliases, parsed)
			index = next - 1
			statementStart = false
			continue
		}
		if statementStart && token.Kind == pythonTokenName && token.Text == "from" {
			parsed, next, valid := parseFromStatement(tokens, index, path, fromModuleID, packageInit, conditions)
			observations = append(observations, parsed...)
			registerPythonDynamicAliases(dynamicAliases, parsed)
			if !valid {
				diagnostics = append(diagnostics, analysis.Diagnostic{
					Code:        "python_import_syntax",
					Severity:    "warning",
					Message:     "Python from-import could not be interpreted statically; unrelated observations were retained.",
					Path:        path,
					Location:    &analysis.Position{Line: token.Line, Column: token.Column},
					Recoverable: true,
				})
			}
			index = next - 1
			statementStart = false
			continue
		}

		switch token.Text {
		case "(", "[", "{":
			bracketDepth++
		case ")", "]", "}":
			if bracketDepth > 0 {
				bracketDepth--
			}
		case ";":
			if bracketDepth == 0 {
				statementStart = true
			}
		case ":":
			if bracketDepth == 0 {
				statementStart = true
			}
		default:
			statementStart = false
		}
	}
	return observations, diagnostics
}

func lexPython(content string) []pythonToken {
	tokens := make([]pythonToken, 0)
	line, column := 1, 1
	for index := 0; index < len(content); {
		char := content[index]
		if char == '\r' {
			if index+1 < len(content) && content[index+1] == '\n' {
				index++
			}
			char = '\n'
		}
		if char == '\n' {
			tokens = append(tokens, pythonToken{Kind: pythonTokenNewline, Text: "\n", Line: line, Column: column, EndLine: line + 1, EndColumn: 1})
			index++
			line++
			column = 1
			continue
		}
		if char == ' ' || char == '\t' || char == '\f' || char == '\v' {
			index++
			column++
			continue
		}
		if char == '#' {
			for index < len(content) && content[index] != '\n' && content[index] != '\r' {
				index++
				column++
			}
			continue
		}
		if char == '\\' && index+1 < len(content) && (content[index+1] == '\n' || content[index+1] == '\r') {
			index++
			if content[index] == '\r' && index+1 < len(content) && content[index+1] == '\n' {
				index++
			}
			index++
			line++
			column = 1
			continue
		}
		if isPythonStringStart(content, index) {
			token, next, nextLine, nextColumn := lexPythonString(content, index, line, column)
			tokens = append(tokens, token)
			index, line, column = next, nextLine, nextColumn
			continue
		}
		runeValue, runeSize := utf8.DecodeRuneInString(content[index:])
		if isPythonNameStart(runeValue) {
			startIndex, startLine, startColumn := index, line, column
			for index < len(content) {
				nameRune, nameSize := utf8.DecodeRuneInString(content[index:])
				if !isPythonNamePart(nameRune) {
					break
				}
				index += nameSize
				column++
			}
			tokens = append(tokens, pythonToken{Kind: pythonTokenName, Text: content[startIndex:index], Line: startLine, Column: startColumn, EndLine: line, EndColumn: column})
			continue
		}
		startLine, startColumn := line, column
		index += runeSize
		column++
		tokens = append(tokens, pythonToken{Kind: pythonTokenPunctuation, Text: string(runeValue), Line: startLine, Column: startColumn, EndLine: line, EndColumn: column})
	}
	return tokens
}

func isPythonStringStart(content string, index int) bool {
	if index >= len(content) {
		return false
	}
	if content[index] == '\'' || content[index] == '"' {
		return true
	}
	if !isPythonStringPrefix(content[index]) {
		return false
	}
	prefixEnd := index
	for prefixEnd < len(content) && prefixEnd-index < 2 && isPythonStringPrefix(content[prefixEnd]) {
		prefixEnd++
	}
	return prefixEnd < len(content) && (content[prefixEnd] == '\'' || content[prefixEnd] == '"')
}

func isPythonStringPrefix(value byte) bool {
	switch value {
	case 'r', 'R', 'u', 'U', 'b', 'B', 'f', 'F':
		return true
	default:
		return false
	}
}

func lexPythonString(content string, index, line, column int) (pythonToken, int, int, int) {
	startIndex, startLine, startColumn := index, line, column
	prefixEnd := index
	for prefixEnd < len(content) && prefixEnd-index < 2 && isPythonStringPrefix(content[prefixEnd]) {
		prefixEnd++
	}
	quote := content[prefixEnd]
	triple := prefixEnd+2 < len(content) && content[prefixEnd+1] == quote && content[prefixEnd+2] == quote
	index = prefixEnd
	column += prefixEnd - startIndex
	if triple {
		index += 3
		column += 3
	} else {
		index++
		column++
	}
	for index < len(content) {
		if content[index] == '\\' && index+1 < len(content) {
			index++
			column++
			if content[index] == '\r' && index+1 < len(content) && content[index+1] == '\n' {
				index++
			}
			if content[index] == '\n' || content[index] == '\r' {
				index++
				line++
				column = 1
			} else {
				_, escapedSize := utf8.DecodeRuneInString(content[index:])
				index += escapedSize
				column++
			}
			continue
		}
		if triple && index+2 < len(content) && content[index] == quote && content[index+1] == quote && content[index+2] == quote {
			index += 3
			column += 3
			break
		}
		if !triple && content[index] == quote {
			index++
			column++
			break
		}
		if content[index] == '\n' || content[index] == '\r' {
			if content[index] == '\r' && index+1 < len(content) && content[index+1] == '\n' {
				index++
			}
			index++
			line++
			column = 1
			continue
		}
		_, runeSize := utf8.DecodeRuneInString(content[index:])
		index += runeSize
		column++
	}
	return pythonToken{Kind: pythonTokenString, Text: content[startIndex:index], Line: startLine, Column: startColumn, EndLine: line, EndColumn: column}, index, line, column
}

func isPythonNameStart(value rune) bool {
	return value == '_' || unicode.IsLetter(value)
}

func isPythonNamePart(value rune) bool {
	return isPythonNameStart(value) || unicode.IsDigit(value) || unicode.Is(unicode.Mn, value) || unicode.Is(unicode.Mc, value) || unicode.Is(unicode.Pc, value)
}

func parseImportStatement(tokens []pythonToken, start int, path, fromModuleID string, conditions map[int][]string) ([]pythonImportObservation, int) {
	result := make([]pythonImportObservation, 0)
	index := start + 1
	for index < len(tokens) {
		if tokens[index].Kind == pythonTokenNewline || tokens[index].Text == ";" {
			break
		}
		if tokens[index].Text == "(" {
			index++
			continue
		}
		if tokens[index].Kind != pythonTokenName {
			break
		}
		first := tokens[index]
		parts := []string{first.Text}
		last := first
		index++
		for index+1 < len(tokens) && tokens[index].Text == "." && tokens[index+1].Kind == pythonTokenName {
			parts = append(parts, tokens[index+1].Text)
			last = tokens[index+1]
			index += 2
		}
		spelling := strings.Join(parts, ".")
		alias := ""
		if index < len(tokens) && tokens[index].Kind == pythonTokenName && tokens[index].Text == "as" {
			index++
			if index < len(tokens) && tokens[index].Kind == pythonTokenName {
				alias = tokens[index].Text
				index++
			}
		}
		result = append(result, makeStaticImportObservation(path, fromModuleID, first, last, spelling, spelling, "", "import", 0, conditions, alias))
		if index >= len(tokens) || tokens[index].Text != "," {
			break
		}
		index++
	}
	return result, index
}

func parseFromStatement(tokens []pythonToken, start int, path, fromModuleID string, packageInit bool, conditions map[int][]string) ([]pythonImportObservation, int, bool) {
	index := start + 1
	level := 0
	var moduleStart pythonToken
	var moduleEnd pythonToken
	moduleParts := make([]string, 0)
	for index < len(tokens) && tokens[index].Text == "." {
		if moduleStart.Text == "" {
			moduleStart = tokens[index]
		}
		moduleEnd = tokens[index]
		level++
		index++
	}
	for index < len(tokens) && tokens[index].Kind == pythonTokenName && tokens[index].Text != "import" {
		if moduleStart.Text == "" {
			moduleStart = tokens[index]
		}
		moduleParts = append(moduleParts, tokens[index].Text)
		moduleEnd = tokens[index]
		index++
		if index >= len(tokens) || tokens[index].Text != "." || (index+1 < len(tokens) && tokens[index+1].Kind != pythonTokenName) {
			break
		}
		index++
	}
	if index >= len(tokens) || tokens[index].Kind != pythonTokenName || tokens[index].Text != "import" {
		return nil, index, false
	}
	index++
	if moduleStart.Text == "" {
		moduleStart = tokens[start]
		moduleEnd = tokens[start]
	}
	result := make([]pythonImportObservation, 0)
	depth := 0
	for index < len(tokens) {
		if tokens[index].Kind == pythonTokenNewline {
			if depth > 0 {
				index++
				continue
			}
			break
		}
		if tokens[index].Text == "(" {
			depth++
			index++
			continue
		}
		if tokens[index].Text == ")" {
			if depth == 0 {
				break
			}
			depth--
			index++
			continue
		}
		if tokens[index].Text == "," {
			index++
			continue
		}
		if tokens[index].Kind != pythonTokenName && tokens[index].Text != "*" {
			break
		}
		imported := tokens[index]
		name := imported.Text
		index++
		alias := ""
		if index < len(tokens) && tokens[index].Kind == pythonTokenName && tokens[index].Text == "as" {
			index++
			if index < len(tokens) && tokens[index].Kind == pythonTokenName {
				alias = tokens[index].Text
				index++
			}
		}
		if moduleStart.Text == "" {
			moduleStart = imported
		}
		if moduleEnd.Text == "" {
			moduleEnd = imported
		}
		module := strings.Join(moduleParts, ".")
		spelling := strings.Repeat(".", level)
		if module != "" {
			spelling += module + "."
		}
		spelling += name
		kind := "from"
		if packageInit {
			kind = "reexport"
		}
		result = append(result, makeStaticImportObservation(path, fromModuleID, moduleStart, imported, spelling, module, name, kind, level, conditions, alias))
		moduleEnd = imported
		if index >= len(tokens) || tokens[index].Text != "," {
			break
		}
	}
	return result, index, true
}

func makeStaticImportObservation(path, fromModuleID string, start, end pythonToken, spelling, module, importedName, kind string, relativeLevel int, conditions map[int][]string, aliases ...string) pythonImportObservation {
	conditionValues := append([]string(nil), conditions[start.Line]...)
	condition := strings.Join(conditionValues, " -> ")
	alias := ""
	if len(aliases) > 0 {
		alias = aliases[0]
	}
	return pythonImportObservation{
		FromModuleID:  fromModuleID,
		Source:        makePythonImportSource(path, start, end, spelling, "import"),
		Spelling:      spelling,
		Module:        module,
		ImportedName:  importedName,
		Alias:         alias,
		RelativeLevel: relativeLevel,
		Kind:          kind,
		Conditional:   condition != "",
		Condition:     condition,
	}
}

func makePythonImportSource(path string, start, end pythonToken, symbol, kind string) analysis.SourceReference {
	return analysis.SourceReference{
		ID:     stableID("import", path, strconv.Itoa(start.Line), strconv.Itoa(start.Column), strconv.Itoa(end.Line), strconv.Itoa(end.EndColumn), symbol, kind),
		Path:   path,
		Start:  &analysis.Position{Line: start.Line, Column: start.Column},
		End:    &analysis.Position{Line: end.EndLine, Column: end.EndColumn},
		Symbol: symbol,
		Kind:   kind,
	}
}

func parseDynamicCall(tokens []pythonToken, index int, path, fromModuleID string, conditions map[int][]string, dynamicAliases map[string]struct{}) (pythonImportObservation, int, bool) {
	if tokens[index].Kind != pythonTokenName {
		return pythonImportObservation{}, index, false
	}
	if index > 0 && tokens[index-1].Kind == pythonTokenName && (tokens[index-1].Text == "def" || tokens[index-1].Text == "class") {
		return pythonImportObservation{}, index, false
	}
	next := nextPythonToken(tokens, index+1)
	if next >= len(tokens) || tokens[next].Text != "(" {
		return pythonImportObservation{}, index, false
	}
	start := index
	for start >= 2 && tokens[start-1].Text == "." && tokens[start-2].Kind == pythonTokenName {
		start -= 2
	}
	function := pythonTokenText(tokens[start : index+1])
	if !knownPythonDynamicCallable(function) {
		if _, ok := dynamicAliases[function]; !ok {
			return pythonImportObservation{}, index, false
		}
	}
	end := matchingCallEnd(tokens, next)
	argument := nextPythonToken(tokens, next+1)
	target := staticPythonCallTarget(tokens, argument, end)
	name := target
	if name == "" {
		name = "<dynamic>"
	}
	endToken := tokens[next]
	if end < len(tokens) {
		endToken = tokens[end]
	}
	condition := strings.Join(conditions[tokens[start].Line], " -> ")
	return pythonImportObservation{
		FromModuleID:    fromModuleID,
		Source:          makePythonImportSource(path, tokens[start], endToken, name, "dynamic"),
		Spelling:        name,
		Kind:            "dynamic",
		Conditional:     condition != "",
		Condition:       condition,
		DynamicFunction: function,
		DynamicTarget:   target,
	}, end, true
}

func staticPythonCallTarget(tokens []pythonToken, argument, callEnd int) string {
	if argument >= len(tokens) || argument >= callEnd || tokens[argument].Kind != pythonTokenString {
		return ""
	}
	var target strings.Builder
	cursor := argument
	for cursor < callEnd && tokens[cursor].Kind == pythonTokenString {
		value, ok := staticPythonString(tokens[cursor].Text)
		if !ok {
			return ""
		}
		target.WriteString(value)
		cursor = nextPythonToken(tokens, cursor+1)
	}
	if cursor < callEnd && tokens[cursor].Text != "," {
		return ""
	}
	return target.String()
}

func registerPythonDynamicAliases(aliases map[string]struct{}, observations []pythonImportObservation) {
	for _, observation := range observations {
		if observation.Kind == "import" {
			binding := observation.Alias
			if binding == "" && !strings.Contains(observation.Module, ".") {
				binding = observation.Module
			}
			registerPythonDynamicModuleBinding(aliases, binding, observation.Module)
			continue
		}
		canonical := joinPythonQualified(observation.Module, observation.ImportedName)
		name := observation.Alias
		if name == "" {
			name = observation.ImportedName
		}
		if knownPythonDynamicCallable(canonical) {
			aliases[name] = struct{}{}
		}
		registerPythonDynamicModuleBinding(aliases, name, canonical)
	}
}

func registerPythonDynamicModuleBinding(aliases map[string]struct{}, binding, module string) {
	if binding == "" {
		return
	}
	var names []string
	switch module {
	case "importlib":
		names = []string{"import_module"}
	case "importlib.util":
		names = []string{"find_spec", "module_from_spec"}
	case "importlib.metadata":
		names = []string{"entry_points"}
	case "pkg_resources":
		names = []string{"iter_entry_points", "load_entry_point", "load_setuptools_entrypoints"}
	case "stevedore.extension":
		names = []string{"ExtensionManager"}
	}
	for _, name := range names {
		aliases[binding+"."+name] = struct{}{}
	}
}

func knownPythonDynamicCallable(name string) bool {
	switch name {
	case "__import__", "builtins.__import__", "importlib.import_module", "importlib.util.find_spec", "importlib.util.module_from_spec", "importlib.metadata.entry_points", "pkg_resources.iter_entry_points", "pkg_resources.load_entry_point", "pkg_resources.load_setuptools_entrypoints", "stevedore.extension.ExtensionManager":
		return true
	default:
		return false
	}
}

func nextPythonToken(tokens []pythonToken, index int) int {
	for index < len(tokens) && tokens[index].Kind == pythonTokenNewline {
		index++
	}
	return index
}

func matchingCallEnd(tokens []pythonToken, open int) int {
	depth := 0
	for index := open; index < len(tokens); index++ {
		switch tokens[index].Text {
		case "(":
			depth++
		case ")":
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return open
}

func pythonTokenText(tokens []pythonToken) string {
	var builder strings.Builder
	for _, token := range tokens {
		if token.Kind != pythonTokenNewline {
			builder.WriteString(token.Text)
		}
	}
	return builder.String()
}

func staticPythonString(value string) (string, bool) {
	value = strings.TrimSpace(value)
	quoteIndex := strings.IndexAny(value, "'\"")
	if quoteIndex < 0 || len(value) < quoteIndex+2 {
		return "", false
	}
	prefix := value[:quoteIndex]
	quote := value[quoteIndex]
	if strings.ContainsAny(prefix, "fF") {
		return "", false
	}
	triple := quoteIndex+2 < len(value) && value[quoteIndex+1] == quote && value[quoteIndex+2] == quote
	start := quoteIndex + 1
	end := len(value) - 1
	if triple {
		if len(value) < quoteIndex+6 || value[len(value)-3] != quote || value[len(value)-2] != quote || value[len(value)-1] != quote {
			return "", false
		}
		start = quoteIndex + 3
		end = len(value) - 3
	}
	body := value[start:end]
	if strings.Contains(prefix, "r") || strings.Contains(prefix, "R") {
		return body, true
	}
	if quote == '"' {
		parsed, err := strconv.Unquote("\"" + body + "\"")
		return parsed, err == nil
	}
	parsed := strings.ReplaceAll(body, `\\`, `\`)
	parsed = strings.ReplaceAll(parsed, `\'`, `'`)
	parsed = strings.ReplaceAll(parsed, `\"`, `"`)
	return parsed, !strings.Contains(parsed, "\\")
}

func conditionalContexts(content string) map[int][]string {
	result := make(map[int][]string)
	stack := make([]pythonConditionFrame, 0)
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	pendingHeader := ""
	pendingIndent := 0
	var pendingContexts []string
	for lineNumber, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		line := lineNumber + 1
		if pendingHeader != "" {
			pendingHeader += " " + normalizePythonConditionalLine(trimmed)
			result[line] = append([]string(nil), pendingContexts...)
			if header, block := pythonConditionalHeader(pendingHeader); header != "" {
				if block {
					stack = append(stack, pythonConditionFrame{Indent: pendingIndent, Text: header})
				} else {
					result[line] = append(result[line], header)
				}
				pendingHeader = ""
				pendingContexts = nil
			}
			continue
		}
		indent := pythonIndent(raw)
		for len(stack) > 0 && indent <= stack[len(stack)-1].Indent {
			stack = stack[:len(stack)-1]
		}
		values := make([]string, 0, len(stack))
		for _, frame := range stack {
			values = append(values, frame.Text)
		}
		result[line] = values
		if header, block := pythonConditionalHeader(trimmed); header != "" {
			if block {
				stack = append(stack, pythonConditionFrame{Indent: indent, Text: header})
			} else {
				result[line] = append(values, header)
			}
		} else if pythonConditionalHeaderCanContinue(trimmed) {
			pendingHeader = normalizePythonConditionalLine(trimmed)
			pendingIndent = indent
			pendingContexts = append([]string(nil), values...)
		}
	}
	return result
}

func pythonConditionalHeaderCanContinue(line string) bool {
	lower := strings.ToLower(strings.TrimSpace(line))
	return strings.HasPrefix(lower, "if ") || strings.HasPrefix(lower, "elif ") || strings.HasPrefix(lower, "while ") || strings.HasPrefix(lower, "for ") || strings.HasPrefix(lower, "except ") || strings.HasPrefix(lower, "case ")
}

func normalizePythonConditionalLine(line string) string {
	line = strings.TrimSpace(pythonCodeBeforeComment(line))
	line = strings.TrimSpace(strings.TrimSuffix(line, "\\"))
	return line
}

func pythonCodeBeforeComment(line string) string {
	quote := byte(0)
	for index := 0; index < len(line); index++ {
		char := line[index]
		if quote != 0 {
			if char == '\\' {
				index++
				continue
			}
			if char == quote {
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

func pythonIndent(value string) int {
	indent := 0
	for _, char := range value {
		switch char {
		case ' ':
			indent++
		case '\t':
			indent += 8 - indent%8
		default:
			return indent
		}
	}
	return indent
}

func pythonConditionalHeader(line string) (string, bool) {
	colon := pythonSuiteColon(line)
	if colon < 0 {
		return "", false
	}
	prefix := strings.TrimSpace(line[:colon])
	lower := strings.ToLower(prefix)
	conditional := lower == "try" || lower == "else" || lower == "finally" || strings.HasPrefix(lower, "if ") || strings.HasPrefix(lower, "elif ") || strings.HasPrefix(lower, "while ") || strings.HasPrefix(lower, "for ") || strings.HasPrefix(lower, "except") || strings.HasPrefix(lower, "case ")
	if !conditional {
		return "", false
	}
	header := prefix + ":"
	rest := strings.TrimSpace(line[colon+1:])
	if comment := strings.Index(rest, "#"); comment >= 0 {
		rest = strings.TrimSpace(rest[:comment])
	}
	return header, rest == ""
}

func pythonSuiteColon(line string) int {
	quote := byte(0)
	depth := 0
	for index := 0; index < len(line); index++ {
		char := line[index]
		if quote != 0 {
			if char == '\\' {
				index++
				continue
			}
			if char == quote {
				quote = 0
			}
			continue
		}
		if char == '\'' || char == '"' {
			quote = char
			continue
		}
		if char == '#' {
			break
		}
		switch char {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		case ':':
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}

func buildImportObservations(project Project, discovery discoveryResult) ([]analysis.RelationshipObservation, []analysis.Reference, []analysis.Diagnostic) {
	index := buildPythonModuleIndex(discovery.Modules, discovery.AmbiguousNames)
	reexports := collectPythonReexports(project, discovery, index)
	relationships := make(map[string]analysis.RelationshipObservation)
	references := make(map[string]analysis.Reference)
	diagnostics := make([]analysis.Diagnostic, 0)
	for _, observation := range discovery.Imports {
		conditionState := evaluatePythonCondition(observation.Condition, project.PythonVersion)
		if conditionState == pythonConditionFalse {
			diagnostics = append(diagnostics, pythonImportDiagnostic(observation, "python_conditional_import_excluded", "Python import was excluded because its condition is false for the selected Python version.", "info", false, map[string]any{"condition": observation.Condition, "condition_evaluation": "false"}))
			continue
		}
		if observation.Kind == "dynamic" {
			addDynamicPythonObservation(project, observation, relationships, references, &diagnostics, conditionState)
			continue
		}
		resolution := resolvePythonImport(observation, index, reexports)
		if resolution.Local && resolution.ModuleID == observation.FromModuleID {
			continue
		}
		scope := "local"
		if !resolution.Local {
			scope = classifyPythonReference(resolution.Name, index, project.PythonVersion, resolution.Ambiguous || observation.RelativeLevel > 0)
		}
		metadata := pythonImportMetadata(observation, scope, resolution)
		conditional := resolution.Conditional || (observation.Conditional && conditionState != pythonConditionTrue)
		if conditional {
			metadata["condition_evaluation"] = "unknown"
		} else if observation.Conditional {
			metadata["condition_evaluation"] = conditionStateName(conditionState)
		}
		var targetID string
		var referenceID string
		if resolution.Local {
			targetID = resolution.ModuleID
		} else {
			referenceID = stableID("reference", "python", scope, resolution.Name)
			reference := references[referenceID]
			if reference.ID == "" {
				reference = analysis.Reference{ID: referenceID, Name: resolution.Name, Scope: scope, Language: "python", Metadata: map[string]any{}}
			}
			mergePythonMetadata(reference.Metadata, pythonReferenceMetadata(observation, scope, resolution, project.PythonVersion))
			references[referenceID] = reference
		}
		relationshipID := stableID("relationship", observation.FromModuleID, scope, targetID, referenceID)
		relationship := relationships[relationshipID]
		if relationship.ID == "" {
			relationship = analysis.RelationshipObservation{ID: relationshipID, Type: "depends_on", FromModuleID: observation.FromModuleID, ToModuleID: targetID, ToReferenceID: referenceID, SourceReferenceIDs: []string{}, Confidence: pythonConfidence(scope, conditional), Metadata: map[string]any{}}
		}
		relationship.SourceReferenceIDs = appendUniquePythonString(relationship.SourceReferenceIDs, observation.Source.ID)
		mergePythonMetadata(relationship.Metadata, metadata)
		relationship.Confidence = mergePythonConfidence(relationship.Confidence, pythonConfidence(scope, conditional))
		relationships[relationshipID] = relationship

		if scope == "unresolved" {
			reason := "could not be resolved to a proven project-local module"
			if resolution.Ambiguous {
				reason = "has multiple possible project-local targets"
			}
			diagnostics = append(diagnostics, pythonImportDiagnostic(observation, "python_unresolved_import", fmt.Sprintf("Python import %q %s; it was retained as an unresolved reference.", observation.Spelling, reason), "warning", true, map[string]any{"target_scope": scope, "reference_id": referenceID, "resolution": resolution.Resolution}))
		}
		if conditional && conditionState == pythonConditionUnknown {
			diagnostics = append(diagnostics, pythonImportDiagnostic(observation, "python_conditional_import", "Python import condition could not be proven for the selected view; the observation is partial.", "warning", true, map[string]any{"condition": observation.Condition, "target_scope": scope}))
		}
	}
	return sortedPythonRelationships(relationships), sortedPythonReferences(references), diagnostics
}

func buildPythonModuleIndex(modules []analysis.ModuleObservation, ambiguous map[string]bool) pythonModuleIndex {
	result := pythonModuleIndex{ByQualified: map[string][]string{}, ByID: map[string]analysis.ModuleObservation{}, TopLevel: map[string]struct{}{}, Ambiguous: map[string]bool{}}
	for qualified, value := range ambiguous {
		result.Ambiguous[qualified] = value
	}
	for _, module := range modules {
		result.ByID[module.ID] = module
		qualified := module.DisplayName
		if value, ok := module.Metadata["qualified_name"].(string); ok && value != "" {
			qualified = value
		}
		result.ByQualified[qualified] = append(result.ByQualified[qualified], module.ID)
		parts := strings.Split(qualified, ".")
		if len(parts) > 0 && parts[0] != "" && parts[0] != "__root__" {
			result.TopLevel[parts[0]] = struct{}{}
		}
	}
	for qualified, IDs := range result.ByQualified {
		sort.Strings(IDs)
		result.ByQualified[qualified] = IDs
		if len(IDs) > 1 {
			result.Ambiguous[qualified] = true
		}
	}
	return result
}

func collectPythonReexports(project Project, discovery discoveryResult, index pythonModuleIndex) map[pythonReexportKey]pythonReexport {
	result := make(map[pythonReexportKey]pythonReexport)
	for _, observation := range discovery.Imports {
		if observation.Kind != "reexport" || observation.ImportedName == "" || observation.ImportedName == "*" {
			continue
		}
		conditionState := evaluatePythonCondition(observation.Condition, project.PythonVersion)
		if conditionState == pythonConditionFalse {
			continue
		}
		fromModule, ok := index.ByID[observation.FromModuleID]
		if !ok || fromModule.Kind != "package" || fromModule.DisplayName == "__root__" {
			continue
		}
		resolution := resolvePythonImport(observation, index, nil)
		if !resolution.Local {
			continue
		}
		name := observation.ImportedName
		if observation.Alias != "" {
			name = observation.Alias
		}
		key := pythonReexportKey{Package: fromModule.DisplayName, Name: name}
		value := result[key]
		if value.ModuleID != "" && value.ModuleID != resolution.ModuleID {
			value.Ambiguous = true
		}
		if value.ModuleID == "" {
			value.ModuleID = resolution.ModuleID
		}
		if observation.Conditional && conditionState != pythonConditionTrue {
			if !value.Unconditional {
				value.Conditional = true
				value.Conditions = appendUniquePythonString(value.Conditions, observation.Condition)
			}
		} else {
			value.Unconditional = true
			value.Conditional = false
			value.Conditions = nil
		}
		result[key] = value
	}
	return result
}

func resolvePythonImport(observation pythonImportObservation, index pythonModuleIndex, reexports map[pythonReexportKey]pythonReexport) pythonImportResolution {
	fromModule := index.ByID[observation.FromModuleID]
	if observation.Kind == "import" {
		name := observation.Module
		moduleID, ambiguous := lookupPythonModule(index, name)
		return pythonImportResolution{Name: name, ModuleID: moduleID, Local: moduleID != "" && !ambiguous, Ambiguous: ambiguous, Resolution: "absolute"}
	}
	base := observation.Module
	if observation.RelativeLevel > 0 {
		context := pythonPackageContext(fromModule)
		if len(context) == 0 || observation.RelativeLevel-1 >= len(context) {
			return pythonImportResolution{Name: strings.Repeat(".", observation.RelativeLevel) + joinPythonQualified(base, observation.ImportedName), Ambiguous: true, Resolution: "relative-parent-outside-root"}
		}
		context = context[:len(context)-(observation.RelativeLevel-1)]
		base = joinPythonQualified(strings.Join(context, "."), base)
	}
	if base != "" {
		_, ambiguousBase := lookupPythonModule(index, base)
		if ambiguousBase {
			name := base
			if observation.ImportedName != "" && observation.ImportedName != "*" {
				name = joinPythonQualified(base, observation.ImportedName)
			}
			return pythonImportResolution{Name: name, Ambiguous: true, Resolution: "ambiguous-base"}
		}
	}
	if observation.ImportedName == "" || observation.ImportedName == "*" {
		moduleID, ambiguous := lookupPythonModule(index, base)
		return pythonImportResolution{Name: base, ModuleID: moduleID, Local: moduleID != "" && !ambiguous, Ambiguous: ambiguous, Resolution: "base"}
	}
	child := joinPythonQualified(base, observation.ImportedName)
	moduleID, ambiguous := lookupPythonModule(index, child)
	if ambiguous {
		return pythonImportResolution{Name: child, Ambiguous: true, Resolution: "ambiguous-child"}
	}
	if moduleID != "" {
		return pythonImportResolution{Name: child, ModuleID: moduleID, Local: true, Resolution: "child"}
	}
	if reexports != nil {
		key := pythonReexportKey{Package: base, Name: observation.ImportedName}
		if value, ok := reexports[key]; ok {
			if value.Ambiguous {
				return pythonImportResolution{Name: child, Ambiguous: true, Resolution: "ambiguous-reexport"}
			}
			return pythonImportResolution{
				Name:        child,
				ModuleID:    value.ModuleID,
				Local:       true,
				Resolution:  "reexport",
				Conditional: value.Conditional,
				Conditions:  append([]string(nil), value.Conditions...),
			}
		}
	}
	if observation.Kind == "reexport" {
		moduleID, ambiguous = lookupPythonModule(index, base)
		if ambiguous {
			return pythonImportResolution{Name: base, Ambiguous: true, Resolution: "ambiguous-base"}
		}
		if moduleID != "" && moduleID != observation.FromModuleID {
			return pythonImportResolution{Name: base, ModuleID: moduleID, Local: true, Resolution: "base"}
		}
	} else if moduleID, ambiguous = lookupPythonModule(index, base); ambiguous {
		return pythonImportResolution{Name: child, Ambiguous: true, Resolution: "ambiguous-base"}
	} else if moduleID != "" {
		if baseModule, ok := index.ByID[moduleID]; ok && baseModule.Kind == "module" {
			return pythonImportResolution{Name: base, ModuleID: moduleID, Local: true, Resolution: "base"}
		}
	}
	return pythonImportResolution{Name: child, Resolution: "unresolved"}
}

func lookupPythonModule(index pythonModuleIndex, qualified string) (string, bool) {
	if qualified == "" {
		return "", false
	}
	IDs := index.ByQualified[qualified]
	if pythonQualifiedPathAmbiguous(index, qualified) || len(IDs) > 1 {
		return "", true
	}
	if len(IDs) == 1 {
		return IDs[0], false
	}
	return "", false
}

func pythonQualifiedPathAmbiguous(index pythonModuleIndex, qualified string) bool {
	for candidate := qualified; candidate != ""; {
		if index.Ambiguous[candidate] {
			return true
		}
		dot := strings.LastIndexByte(candidate, '.')
		if dot < 0 {
			break
		}
		candidate = candidate[:dot]
	}
	return false
}

func pythonPackageContext(module analysis.ModuleObservation) []string {
	qualified := module.DisplayName
	if value, ok := module.Metadata["qualified_name"].(string); ok && value != "" {
		qualified = value
	}
	parts := strings.Split(qualified, ".")
	if module.Kind == "package" {
		if qualified == "" || qualified == "__root__" {
			return nil
		}
		return parts
	}
	if len(parts) == 0 || (len(parts) == 1 && parts[0] == "") {
		return nil
	}
	return parts[:len(parts)-1]
}

func joinPythonQualified(left, right string) string {
	left = strings.Trim(left, ".")
	right = strings.Trim(right, ".")
	if left == "" {
		return right
	}
	if right == "" {
		return left
	}
	return left + "." + right
}

func classifyPythonReference(name string, index pythonModuleIndex, version string, ambiguous bool) string {
	if ambiguous || strings.HasPrefix(name, ".") {
		return "unresolved"
	}
	root := name
	if dot := strings.IndexByte(root, '.'); dot >= 0 {
		root = root[:dot]
	}
	if rule, ok := pythonStdlibModules[root]; ok && pythonStdlibRuleApplies(rule, version) {
		return "standard_library"
	}
	if _, ok := index.TopLevel[root]; ok {
		return "unresolved"
	}
	return "external"
}

func pythonStdlibRuleApplies(rule pythonStdlibRule, version string) bool {
	minor, known := pythonVersionMinor(version)
	if !known {
		return true
	}
	return (rule.MinMinor == 0 || minor >= rule.MinMinor) && (rule.MaxMinor == 0 || minor <= rule.MaxMinor)
}

func pythonVersionMinor(value string) (int, bool) {
	match := pythonVersionValuePattern.FindStringSubmatch(strings.TrimSpace(value))
	if len(match) < 2 {
		return 0, false
	}
	minor, err := strconv.Atoi(match[1])
	return minor, err == nil
}

func evaluatePythonCondition(condition, version string) pythonConditionState {
	if condition == "" {
		return pythonConditionTrue
	}
	state := pythonConditionTrue
	for _, part := range strings.Split(condition, " -> ") {
		trimmed := strings.TrimSpace(strings.TrimSuffix(part, ":"))
		lower := strings.ToLower(trimmed)
		partState := pythonConditionUnknown
		switch lower {
		case "if true", "elif true":
			partState = pythonConditionTrue
		case "if false", "elif false":
			partState = pythonConditionFalse
		default:
			expression := trimmed
			if strings.HasPrefix(expression, "if ") || strings.HasPrefix(expression, "elif ") {
				expression = strings.TrimSpace(expression[strings.IndexByte(expression, ' ')+1:])
			}
			if match := pythonVersionConditionPattern.FindStringSubmatch(expression); len(match) > 2 {
				if minor, known := pythonVersionMinor(version); known {
					threshold, _ := strconv.Atoi(match[2])
					partState = comparePythonVersion(minor, threshold, match[1])
				}
			}
		}
		if partState == pythonConditionFalse {
			return pythonConditionFalse
		}
		if partState == pythonConditionUnknown {
			state = pythonConditionUnknown
		}
	}
	return state
}

func comparePythonVersion(actual, threshold int, operator string) pythonConditionState {
	value := false
	switch operator {
	case ">=":
		value = actual >= threshold
	case ">":
		value = actual > threshold
	case "<=":
		value = actual <= threshold
	case "<":
		value = actual < threshold
	case "==":
		value = actual == threshold
	case "!=":
		value = actual != threshold
	default:
		return pythonConditionUnknown
	}
	if value {
		return pythonConditionTrue
	}
	return pythonConditionFalse
}

func conditionStateName(value pythonConditionState) string {
	switch value {
	case pythonConditionTrue:
		return "true"
	case pythonConditionFalse:
		return "false"
	default:
		return "unknown"
	}
}

func pythonImportMetadata(observation pythonImportObservation, scope string, resolution pythonImportResolution) map[string]any {
	metadata := map[string]any{
		"target_scope":     scope,
		"import_paths":     []string{observation.Spelling},
		"import_kinds":     []string{observation.Kind},
		"resolution_kinds": []string{resolution.Resolution},
		"relative_levels":  []int{observation.RelativeLevel},
		"import_names":     []string{},
	}
	if observation.ImportedName != "" {
		metadata["import_names"] = []string{observation.ImportedName}
	}
	if observation.Alias != "" {
		metadata["aliases"] = []string{observation.Alias}
	}
	if observation.RelativeLevel > 0 {
		metadata["relative"] = true
	}
	if observation.Kind == "reexport" {
		metadata["reexport"] = true
	}
	if resolution.Resolution == "reexport" {
		metadata["reexport"] = true
	}
	conditions := make([]string, 0, 1+len(resolution.Conditions))
	if observation.Conditional {
		conditions = appendUniquePythonString(conditions, observation.Condition)
	}
	if resolution.Conditional {
		for _, condition := range resolution.Conditions {
			conditions = appendUniquePythonString(conditions, condition)
		}
	}
	if len(conditions) > 0 {
		metadata["conditional"] = true
		sort.Strings(conditions)
		metadata["conditions"] = conditions
	}
	return metadata
}

func pythonReferenceMetadata(observation pythonImportObservation, scope string, resolution pythonImportResolution, version string) map[string]any {
	metadata := pythonImportMetadata(observation, scope, resolution)
	metadata["reference_scope"] = scope
	if observation.Conditional {
		metadata["condition_evaluation"] = conditionStateName(evaluatePythonCondition(observation.Condition, version))
	}
	if resolution.Ambiguous {
		metadata["ambiguous"] = true
	}
	if version != "" {
		metadata["python_version"] = version
	}
	return metadata
}

func addDynamicPythonObservation(project Project, observation pythonImportObservation, relationships map[string]analysis.RelationshipObservation, references map[string]analysis.Reference, diagnostics *[]analysis.Diagnostic, conditionState pythonConditionState) {
	name := observation.DynamicTarget
	if name == "" {
		name = "<dynamic>"
	}
	referenceID := stableID("reference", "python", "dynamic", name)
	reference := references[referenceID]
	if reference.ID == "" {
		reference = analysis.Reference{ID: referenceID, Name: name, Scope: "dynamic", Language: "python", Metadata: map[string]any{}}
	}
	metadata := map[string]any{
		"target_scope":      "dynamic",
		"dynamic_functions": []string{observation.DynamicFunction},
		"dynamic_targets":   []string{},
		"import_kinds":      []string{"dynamic"},
	}
	if observation.DynamicTarget != "" {
		metadata["dynamic_targets"] = []string{observation.DynamicTarget}
	}
	if observation.Conditional {
		metadata["conditional"] = true
		metadata["conditions"] = []string{observation.Condition}
		metadata["condition_evaluation"] = conditionStateName(conditionState)
	}
	if project.PythonVersion != "" {
		metadata["python_version"] = project.PythonVersion
	}
	mergePythonMetadata(reference.Metadata, metadata)
	references[referenceID] = reference
	relationshipID := stableID("relationship", observation.FromModuleID, "dynamic", referenceID)
	relationship := relationships[relationshipID]
	if relationship.ID == "" {
		relationship = analysis.RelationshipObservation{ID: relationshipID, Type: "depends_on", FromModuleID: observation.FromModuleID, ToReferenceID: referenceID, SourceReferenceIDs: []string{}, Confidence: &analysis.Confidence{Basis: "dynamic", Score: 0.2}, Metadata: map[string]any{}}
	}
	relationship.SourceReferenceIDs = appendUniquePythonString(relationship.SourceReferenceIDs, observation.Source.ID)
	mergePythonMetadata(relationship.Metadata, metadata)
	relationship.Confidence = mergePythonConfidence(relationship.Confidence, &analysis.Confidence{Basis: "dynamic", Score: 0.2})
	relationships[relationshipID] = relationship
	*diagnostics = append(*diagnostics, pythonImportDiagnostic(observation, "python_dynamic_import", fmt.Sprintf("Python dynamic import %q through %s could not be resolved statically; it was retained as a dynamic reference.", name, observation.DynamicFunction), "warning", true, metadata))
	if observation.Conditional && conditionState == pythonConditionUnknown {
		*diagnostics = append(*diagnostics, pythonImportDiagnostic(observation, "python_conditional_import", "Python dynamic import condition could not be proven for the selected view; the observation is partial.", "warning", true, map[string]any{"condition": observation.Condition, "target_scope": "dynamic"}))
	}
}

func pythonImportDiagnostic(observation pythonImportObservation, code, message, severity string, recoverable bool, metadata map[string]any) analysis.Diagnostic {
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["source_reference_id"] = observation.Source.ID
	return analysis.Diagnostic{Code: code, Severity: severity, Message: message, Path: observation.Source.Path, Location: observation.Source.Start, Recoverable: recoverable, Metadata: metadata}
}

func pythonConfidence(scope string, conditional bool) *analysis.Confidence {
	if scope == "dynamic" {
		return &analysis.Confidence{Basis: "dynamic", Score: 0.2}
	}
	if scope == "unresolved" {
		return &analysis.Confidence{Basis: "unresolved", Score: 0.2}
	}
	if conditional {
		return &analysis.Confidence{Basis: "inferred", Score: 0.5}
	}
	return &analysis.Confidence{Basis: "resolved", Score: 1}
}

func mergePythonConfidence(left, right *analysis.Confidence) *analysis.Confidence {
	if left == nil {
		return right
	}
	if right == nil {
		return left
	}
	if right.Score < left.Score {
		return right
	}
	return left
}

func mergePythonMetadata(target, values map[string]any) {
	if target == nil {
		return
	}
	for key, value := range values {
		switch typed := value.(type) {
		case []string:
			existing, exists := target[key].([]string)
			if !exists {
				existing = []string{}
			}
			for _, item := range typed {
				existing = appendUniquePythonString(existing, item)
			}
			sort.Strings(existing)
			target[key] = existing
		case []int:
			existing, _ := target[key].([]int)
			for _, item := range typed {
				found := false
				for _, current := range existing {
					if current == item {
						found = true
						break
					}
				}
				if !found {
					existing = append(existing, item)
				}
			}
			sort.Ints(existing)
			target[key] = existing
		case bool:
			if _, ok := target[key].(bool); !ok || typed {
				target[key] = typed
			}
		case string:
			existing, exists := target[key].(string)
			if !exists {
				target[key] = typed
			} else if key == "condition_evaluation" && existing != typed {
				target[key] = "unknown"
			}
		default:
			if _, exists := target[key]; !exists {
				target[key] = value
			}
		}
	}
}

func appendUniquePythonString(values []string, value string) []string {
	for _, current := range values {
		if current == value {
			return values
		}
	}
	return append(values, value)
}

func sortedPythonRelationships(values map[string]analysis.RelationshipObservation) []analysis.RelationshipObservation {
	result := make([]analysis.RelationshipObservation, 0, len(values))
	for _, value := range values {
		sort.Strings(value.SourceReferenceIDs)
		mergePythonMetadata(value.Metadata, nil)
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func sortedPythonReferences(values map[string]analysis.Reference) []analysis.Reference {
	result := make([]analysis.Reference, 0, len(values))
	for _, value := range values {
		mergePythonMetadata(value.Metadata, nil)
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}
