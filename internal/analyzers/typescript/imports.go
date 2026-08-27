package tsanalyzer

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/buffo/arch-view/internal/analysis"
)

type tsImportObservation struct {
	FromModuleID string
	Source       analysis.SourceReference
	Specifier    string
	Kind         string
	ImportNames  []string
	Aliases      []string
	TypeOnly     bool
	Reexport     bool
	Dynamic      bool
	Computed     bool
	Expression   string
}

type tsTokenKind uint8

const (
	tsTokenIdentifier tsTokenKind = iota
	tsTokenString
	tsTokenNumber
	tsTokenPunctuation
	tsTokenNewline
)

type tsToken struct {
	Kind      tsTokenKind
	Text      string
	Raw       string
	Computed  bool
	Line      int
	Column    int
	EndLine   int
	EndColumn int
}

// extractTSImports performs a conservative lexical pass. It intentionally
// recognizes dependency-bearing syntax without trying to type-check or
// execute JavaScript/TypeScript.
func extractTSImports(repositoryPath, content, fromModuleID string) ([]tsImportObservation, []analysis.Diagnostic) {
	tokens, diagnostics := lexTypeScript(repositoryPath, content)
	observations := make([]tsImportObservation, 0)
	ordinal := 0
	for index := 0; index < len(tokens); index++ {
		token := tokens[index]
		if token.Kind != tsTokenIdentifier {
			continue
		}
		switch token.Text {
		case "import":
			previous := previousToken(tokens, index-1)
			if previous.Kind == tsTokenPunctuation && previous.Text == "." {
				continue
			}
			nextValue := nextToken(tokens, index+1)
			if nextValue.Kind == tsTokenPunctuation && nextValue.Text == "(" {
				observation, next, ok := parseTSCall(tokens, index, repositoryPath, fromModuleID, "dynamic_import", ordinal)
				if ok {
					observations = append(observations, observation)
					ordinal++
					index = next - 1
				} else {
					diagnostics = append(diagnostics, tsImportSyntaxDiagnostic(repositoryPath, "TypeScript dynamic import call could not be interpreted statically.", token))
				}
				continue
			}
			observation, next, ok := parseTSModuleDeclaration(tokens, index, repositoryPath, fromModuleID, false, ordinal)
			if ok {
				observations = append(observations, observation)
				ordinal++
				index = next - 1
			} else if looksLikeTSModuleDeclaration(tokens, index, false) {
				diagnostics = append(diagnostics, tsImportSyntaxDiagnostic(repositoryPath, "TypeScript import declaration could not be interpreted statically.", token))
			}
		case "export":
			observation, next, ok := parseTSModuleDeclaration(tokens, index, repositoryPath, fromModuleID, true, ordinal)
			if ok {
				observations = append(observations, observation)
				ordinal++
				index = next - 1
			} else if looksLikeTSModuleDeclaration(tokens, index, true) {
				diagnostics = append(diagnostics, tsImportSyntaxDiagnostic(repositoryPath, "TypeScript export declaration could not be interpreted statically.", token))
			}
		case "require":
			previous := previousToken(tokens, index-1)
			if previous.Kind == tsTokenPunctuation && previous.Text == "." {
				continue
			}
			observation, next, ok := parseTSCall(tokens, index, repositoryPath, fromModuleID, "require", ordinal)
			if ok {
				observations = append(observations, observation)
				ordinal++
				index = next - 1
			} else if looksLikeTSCall(tokens, index) {
				diagnostics = append(diagnostics, tsImportSyntaxDiagnostic(repositoryPath, "TypeScript require call could not be interpreted statically.", token))
			}
		}
	}
	return observations, diagnostics
}

func looksLikeTSModuleDeclaration(tokens []tsToken, start int, reexport bool) bool {
	if start+1 >= len(tokens) {
		return false
	}
	next := tokens[start+1]
	if !reexport {
		return next.Kind != tsTokenPunctuation || (next.Text != "." && next.Text != "=" && next.Text != ":")
	}
	if next.Kind == tsTokenPunctuation && (next.Text == "{" || next.Text == "*") {
		return true
	}
	if next.Text != "type" {
		return false
	}
	return start+2 < len(tokens) && tokens[start+2].Kind == tsTokenPunctuation && (tokens[start+2].Text == "{" || tokens[start+2].Text == "*")
}

func lexTypeScript(repositoryPath, content string) ([]tsToken, []analysis.Diagnostic) {
	tokens := make([]tsToken, 0)
	diagnostics := make([]analysis.Diagnostic, 0)
	line, column := 1, 1
	for index := 0; index < len(content); {
		char := content[index]
		if char == '\r' || char == '\n' {
			startLine, startColumn := line, column
			if char == '\r' && index+1 < len(content) && content[index+1] == '\n' {
				index++
			}
			index++
			line++
			column = 1
			tokens = append(tokens, tsToken{Kind: tsTokenNewline, Text: "\n", Line: startLine, Column: startColumn, EndLine: line, EndColumn: column})
			continue
		}
		if char == ' ' || char == '\t' || char == '\f' || char == '\v' {
			index++
			column++
			continue
		}
		if char == '/' && index+1 < len(content) && content[index+1] == '/' {
			index += 2
			column += 2
			for index < len(content) && content[index] != '\r' && content[index] != '\n' {
				index++
				column++
			}
			continue
		}
		if char == '/' && index+1 < len(content) && content[index+1] == '*' {
			startLine, startColumn := line, column
			index += 2
			column += 2
			closed := false
			for index < len(content) {
				if index+1 < len(content) && content[index] == '*' && content[index+1] == '/' {
					index += 2
					column += 2
					closed = true
					break
				}
				if content[index] == '\r' || content[index] == '\n' {
					if content[index] == '\r' && index+1 < len(content) && content[index+1] == '\n' {
						index++
					}
					index++
					line++
					column = 1
					continue
				}
				index++
				column++
			}
			if !closed {
				diagnostics = append(diagnostics, analysis.Diagnostic{
					Code:        "typescript_unterminated_comment",
					Severity:    "error",
					Message:     "TypeScript source contains an unterminated block comment; dependency extraction continued with the available tokens.",
					Path:        repositoryPath,
					Location:    &analysis.Position{Line: startLine, Column: startColumn},
					Recoverable: true,
				})
			}
			continue
		}
		if isTSNumberStart(content, index) {
			start := index
			index = scanTSNumber(content, index)
			tokens = append(tokens, tsToken{Kind: tsTokenNumber, Text: content[start:index], Raw: content[start:index], Line: line, Column: column, EndLine: line, EndColumn: column + (index - start)})
			column += index - start
			continue
		}
		if char == '\'' || char == '"' || char == '`' {
			token, next, nextLine, nextColumn, closed := scanTSString(content, index, line, column, char)
			tokens = append(tokens, token)
			index, line, column = next, nextLine, nextColumn
			if !closed {
				diagnostics = append(diagnostics, analysis.Diagnostic{
					Code:        "typescript_unterminated_string",
					Severity:    "error",
					Message:     "TypeScript source contains an unterminated string or template literal; dependency extraction continued with the available tokens.",
					Path:        repositoryPath,
					Location:    &analysis.Position{Line: token.Line, Column: token.Column},
					Recoverable: true,
				})
			}
			continue
		}
		if char == '/' && shouldStartTSRegex(tokens) {
			startLine, startColumn := line, column
			next, nextLine, nextColumn, closed := scanTSRegex(content, index, line, column)
			if !closed {
				diagnostics = append(diagnostics, analysis.Diagnostic{
					Code:        "typescript_unterminated_regex",
					Severity:    "error",
					Message:     "TypeScript source contains an unterminated regular expression literal; dependency extraction continued with the available tokens.",
					Path:        repositoryPath,
					Location:    &analysis.Position{Line: line, Column: column},
					Recoverable: true,
				})
			}
			tokens = append(tokens, tsToken{
				Kind:      tsTokenString,
				Text:      "<regex>",
				Raw:       content[index:next],
				Line:      startLine,
				Column:    startColumn,
				EndLine:   nextLine,
				EndColumn: nextColumn,
			})
			index, line, column = next, nextLine, nextColumn
			continue
		}
		if isTSIdentifierStart(char) {
			start := index
			startLine, startColumn := line, column
			for index < len(content) && isTSIdentifierPart(content[index]) {
				index++
				column++
			}
			tokens = append(tokens, tsToken{Kind: tsTokenIdentifier, Text: content[start:index], Raw: content[start:index], Line: startLine, Column: startColumn, EndLine: line, EndColumn: column})
			continue
		}
		// Operators and punctuation only need their textual value for the
		// dependency grammar. Keeping each rune separate makes call parsing
		// deterministic and avoids a language parser dependency.
		text := string(char)
		startLine, startColumn := line, column
		index++
		column++
		tokens = append(tokens, tsToken{Kind: tsTokenPunctuation, Text: text, Raw: text, Line: startLine, Column: startColumn, EndLine: line, EndColumn: column})
	}
	return tokens, diagnostics
}

func scanTSString(content string, start, line, column int, quote byte) (tsToken, int, int, int, bool) {
	startLine, startColumn := line, column
	index := start + 1
	column++
	escaped := false
	computed := false
	for index < len(content) {
		char := content[index]
		if quote == '`' && !escaped && char == '$' && index+1 < len(content) && content[index+1] == '{' {
			computed = true
		}
		if !escaped && char == quote {
			index++
			column++
			raw := content[start:index]
			return tsToken{Kind: tsTokenString, Text: decodeTSString(raw, quote), Raw: raw, Computed: computed, Line: startLine, Column: startColumn, EndLine: line, EndColumn: column}, index, line, column, true
		}
		if char == '\\' && !escaped {
			escaped = true
			index++
			column++
			continue
		}
		escaped = false
		if char == '\r' || char == '\n' {
			if char == '\r' && index+1 < len(content) && content[index+1] == '\n' {
				index++
			}
			index++
			line++
			column = 1
			continue
		}
		index++
		column++
	}
	raw := content[start:]
	return tsToken{Kind: tsTokenString, Text: decodeTSString(raw, quote), Raw: raw, Computed: computed, Line: startLine, Column: startColumn, EndLine: line, EndColumn: column}, len(content), line, column, false
}

func shouldStartTSRegex(tokens []tsToken) bool {
	for index := len(tokens) - 1; index >= 0; index-- {
		previous := tokens[index]
		if previous.Kind == tsTokenNewline {
			continue
		}
		if previous.Kind == tsTokenIdentifier {
			switch previous.Text {
			case "await", "case", "delete", "else", "in", "instanceof", "of", "return", "throw", "typeof", "void", "yield":
				return true
			default:
				return false
			}
		}
		if previous.Kind == tsTokenString {
			return false
		}
		if previous.Kind == tsTokenNumber {
			return false
		}
		switch previous.Text {
		case "+", "-":
			previousIndex := index - 1
			for previousIndex >= 0 && tokens[previousIndex].Kind == tsTokenNewline {
				previousIndex--
			}
			if previousIndex >= 0 && tokens[previousIndex].Kind == tsTokenPunctuation && tokens[previousIndex].Text == previous.Text && tokens[previousIndex].EndLine == previous.Line && tokens[previousIndex].EndColumn == previous.Column {
				return false
			}
			return true
		case ")":
			return tsRegexAfterControlParen(tokens, index)
		case "]", "}":
			return false
		default:
			return true
		}
	}
	return true
}

func tsRegexAfterControlParen(tokens []tsToken, closeIndex int) bool {
	depth := 1
	for index := closeIndex - 1; index >= 0; index-- {
		token := tokens[index]
		if token.Kind != tsTokenPunctuation {
			continue
		}
		switch token.Text {
		case ")":
			depth++
		case "(":
			depth--
			if depth != 0 {
				continue
			}
			control := previousToken(tokens, index-1)
			if control.Kind != tsTokenIdentifier {
				return false
			}
			switch control.Text {
			case "catch", "for", "if", "switch", "while", "with":
				return true
			default:
				return false
			}
		}
	}
	return false
}

func isTSNumberStart(content string, index int) bool {
	if index >= len(content) {
		return false
	}
	if content[index] >= '0' && content[index] <= '9' {
		return true
	}
	return content[index] == '.' && index+1 < len(content) && content[index+1] >= '0' && content[index+1] <= '9'
}

func scanTSNumber(content string, start int) int {
	index := start
	if content[index] == '.' {
		index++
		for index < len(content) && ((content[index] >= '0' && content[index] <= '9') || content[index] == '_') {
			index++
		}
		return index
	}
	if index+1 < len(content) && content[index] == '0' {
		switch content[index+1] {
		case 'x', 'X', 'b', 'B', 'o', 'O':
			index += 2
			for index < len(content) && (isTSIdentifierPart(content[index]) || content[index] == '_') {
				index++
			}
			return index
		}
	}
	for index < len(content) && ((content[index] >= '0' && content[index] <= '9') || content[index] == '_') {
		index++
	}
	if index < len(content) && content[index] == '.' {
		index++
		for index < len(content) && ((content[index] >= '0' && content[index] <= '9') || content[index] == '_') {
			index++
		}
	}
	if index < len(content) && (content[index] == 'e' || content[index] == 'E') {
		index++
		if index < len(content) && (content[index] == '+' || content[index] == '-') {
			index++
		}
		for index < len(content) && ((content[index] >= '0' && content[index] <= '9') || content[index] == '_') {
			index++
		}
	}
	if index < len(content) && content[index] == 'n' {
		index++
	}
	return index
}

func scanTSRegex(content string, start, line, column int) (int, int, int, bool) {
	index := start + 1
	column++
	escaped := false
	inClass := false
	for index < len(content) {
		char := content[index]
		if char == '\r' || char == '\n' {
			return index, line, column, false
		}
		if escaped {
			escaped = false
			index++
			column++
			continue
		}
		if char == '\\' {
			escaped = true
			index++
			column++
			continue
		}
		if char == '[' {
			inClass = true
		} else if char == ']' {
			inClass = false
		} else if char == '/' && !inClass {
			index++
			column++
			for index < len(content) && isTSIdentifierPart(content[index]) {
				index++
				column++
			}
			return index, line, column, true
		}
		index++
		column++
	}
	return index, line, column, false
}

func decodeTSString(raw string, quote byte) string {
	if len(raw) >= 2 && raw[0] == quote && raw[len(raw)-1] == quote {
		raw = raw[1 : len(raw)-1]
	} else if len(raw) > 0 && raw[0] == quote {
		raw = raw[1:]
	}
	var builder strings.Builder
	for index := 0; index < len(raw); index++ {
		if raw[index] != '\\' || index+1 >= len(raw) {
			builder.WriteByte(raw[index])
			continue
		}
		index++
		switch raw[index] {
		case 'n':
			builder.WriteByte('\n')
		case 'r':
			builder.WriteByte('\r')
		case 't':
			builder.WriteByte('\t')
		case 'b':
			builder.WriteByte('\b')
		case 'f':
			builder.WriteByte('\f')
		case 'v':
			builder.WriteByte('\v')
		case '\\', '\'', '"', '`', '$':
			builder.WriteByte(raw[index])
		case '\r', '\n':
			// JavaScript line continuations do not contribute to the path.
			if raw[index] == '\r' && index+1 < len(raw) && raw[index+1] == '\n' {
				index++
			}
		default:
			builder.WriteByte(raw[index])
		}
	}
	return builder.String()
}

func isTSIdentifierStart(char byte) bool {
	return char == '_' || char == '$' || char >= 0x80 || unicode.IsLetter(rune(char))
}

func isTSIdentifierPart(char byte) bool {
	return isTSIdentifierStart(char) || char >= '0' && char <= '9'
}

func parseTSModuleDeclaration(tokens []tsToken, start int, repositoryPath, fromModuleID string, reexport bool, ordinal int) (tsImportObservation, int, bool) {
	depth := 0
	fromIndex := -1
	stringIndex := -1
	typeOnly := isTSImportTypeModifier(tokens, start)
	for index := start + 1; index < len(tokens); index++ {
		token := tokens[index]
		if token.Kind == tsTokenNewline && depth == 0 {
			break
		}
		if token.Kind == tsTokenPunctuation && token.Text == ";" && depth == 0 {
			break
		}
		if token.Kind == tsTokenPunctuation {
			switch token.Text {
			case "{", "(", "[":
				depth++
			case "}", ")", "]":
				if depth > 0 {
					depth--
				}
			}
		}
		if token.Kind == tsTokenIdentifier && token.Text == "from" && depth == 0 {
			fromIndex = index
			continue
		}
		if token.Kind == tsTokenString && (fromIndex >= 0 || (!reexport && index == start+1)) {
			stringIndex = index
			break
		}
	}
	if stringIndex < 0 {
		return tsImportObservation{}, start + 1, false
	}
	token := tokens[stringIndex]
	observation := tsImportObservation{
		FromModuleID: fromModuleID,
		Specifier:    token.Text,
		Kind:         "import",
		TypeOnly:     typeOnly,
		Reexport:     reexport,
		Source:       tsImportSource(repositoryPath, token, token.Text, "import", ordinal),
	}
	if reexport {
		observation.Kind = "reexport"
		observation.Source.Kind = "export"
	}
	if typeOnly {
		observation.Kind = "type_import"
		if reexport {
			observation.Kind = "reexport"
		}
	}
	observation.ImportNames, observation.Aliases = parseTSImportNames(tokens[start:stringIndex], reexport, typeOnly)
	if !typeOnly && hasOnlyTypeNamedSpecifiers(tokens[start:stringIndex]) {
		observation.TypeOnly = true
	}
	if observation.TypeOnly && !reexport {
		observation.Kind = "type_import"
	}
	return observation, stringIndex + 1, true
}

func isTSImportTypeModifier(tokens []tsToken, start int) bool {
	modifierIndex := nextNonNewlineIndex(tokens, start+1)
	if modifierIndex >= len(tokens) || tokens[modifierIndex].Kind != tsTokenIdentifier || tokens[modifierIndex].Text != "type" {
		return false
	}
	nextIndex := nextNonNewlineIndex(tokens, modifierIndex+1)
	if nextIndex >= len(tokens) {
		return false
	}
	if tokens[nextIndex].Kind == tsTokenPunctuation {
		return tokens[nextIndex].Text == "{" || tokens[nextIndex].Text == "*"
	}
	return tokens[nextIndex].Kind == tsTokenIdentifier && tokens[nextIndex].Text != "from"
}

func hasOnlyTypeNamedSpecifiers(tokens []tsToken) bool {
	open := -1
	close := -1
	for index, token := range tokens {
		if token.Kind == tsTokenPunctuation && token.Text == "{" {
			open = index
			break
		}
	}
	if open < 0 {
		return false
	}
	for index := open + 1; index < len(tokens); index++ {
		if tokens[index].Kind == tsTokenPunctuation && tokens[index].Text == "}" {
			close = index
			break
		}
	}
	if close <= open+1 {
		return false
	}
	hasItem := false
	index := open + 1
	for index < close {
		for index < close && tokens[index].Kind == tsTokenPunctuation && tokens[index].Text == "," {
			index++
		}
		if index >= close || tokens[index].Kind != tsTokenIdentifier || tokens[index].Text != "type" {
			return false
		}
		index++
		if index >= close || tokens[index].Kind != tsTokenIdentifier {
			return false
		}
		hasItem = true
		for index < close && (tokens[index].Kind != tsTokenPunctuation || tokens[index].Text != ",") {
			index++
		}
	}
	return hasItem
}

func parseTSCall(tokens []tsToken, start int, repositoryPath, fromModuleID, kind string, ordinal int) (tsImportObservation, int, bool) {
	openIndex := nextNonNewlineIndex(tokens, start+1)
	if openIndex >= len(tokens) || tokens[openIndex].Kind != tsTokenPunctuation || tokens[openIndex].Text != "(" {
		return tsImportObservation{}, start + 1, false
	}
	argumentIndex := nextNonNewlineIndex(tokens, openIndex+1)
	if argumentIndex >= len(tokens) || (tokens[argumentIndex].Kind == tsTokenPunctuation && tokens[argumentIndex].Text == ")") {
		return tsImportObservation{}, len(tokens), false
	}
	argument := tokens[argumentIndex]
	callEnd, closed := findTSCallEnd(tokens, openIndex)
	if !closed {
		return tsImportObservation{}, callEnd, false
	}
	if callEnd <= argumentIndex {
		callEnd = len(tokens)
	}
	specifier := argument.Text
	nextArgumentToken := nextNonNewlineIndex(tokens, argumentIndex+1)
	soleArgument := nextArgumentToken >= callEnd-1
	dynamicImportOptions := kind == "dynamic_import" && nextArgumentToken < callEnd-1 && tokens[nextArgumentToken].Kind == tsTokenPunctuation && tokens[nextArgumentToken].Text == ","
	literal := argument.Kind == tsTokenString && !argument.Computed && (soleArgument || dynamicImportOptions)
	computed := !literal
	expression := argument.Raw
	if computed {
		specifier = "<computed>"
		expression = joinTSTokens(tokens[argumentIndex : callEnd-1])
	}
	observation := tsImportObservation{
		FromModuleID: fromModuleID,
		Specifier:    specifier,
		Kind:         kind,
		Source:       tsImportSource(repositoryPath, argument, specifier, kind, ordinal),
		Dynamic:      kind == "dynamic_import" || computed,
		Computed:     computed,
		Expression:   expression,
	}
	return observation, callEnd, true
}

func looksLikeTSCall(tokens []tsToken, start int) bool {
	openIndex := nextNonNewlineIndex(tokens, start+1)
	return openIndex < len(tokens) && tokens[openIndex].Kind == tsTokenPunctuation && tokens[openIndex].Text == "("
}

func parseTSImportNames(tokens []tsToken, reexport, typeOnly bool) ([]string, []string) {
	names := make([]string, 0)
	aliases := make([]string, 0)
	for index := 1; index < len(tokens); index++ {
		if tokens[index].Kind != tsTokenIdentifier {
			continue
		}
		if tokens[index].Kind == tsTokenIdentifier && tokens[index].Text == "type" {
			if (typeOnly && index == 1) || (index > 0 && tokens[index-1].Kind == tsTokenPunctuation && (tokens[index-1].Text == "{" || tokens[index-1].Text == ",") && index+1 < len(tokens) && tokens[index+1].Kind == tsTokenIdentifier && tokens[index+1].Text != "as") {
				continue
			}
		}
		if tokens[index].Kind == tsTokenIdentifier && (tokens[index].Text == "from" || tokens[index].Text == "as") {
			continue
		}
		if tokens[index-1].Kind == tsTokenIdentifier && tokens[index-1].Text == "as" {
			aliases = appendUniqueString(aliases, tokens[index].Text)
			continue
		}
		if tokens[index].Text == "import" || tokens[index].Text == "export" || tokens[index].Text == "from" {
			continue
		}
		if reexport && tokens[index].Text == "default" {
			continue
		}
		names = appendUniqueString(names, tokens[index].Text)
	}
	return sortStringSet(names), sortStringSet(aliases)
}

func tsImportSource(repositoryPath string, token tsToken, symbol, kind string, ordinal int) analysis.SourceReference {
	return analysis.SourceReference{
		ID:     stableTSID("source", repositoryPath, strconv.Itoa(token.Line), strconv.Itoa(token.Column), kind, symbol, strconv.Itoa(ordinal)),
		Path:   repositoryPath,
		Start:  &analysis.Position{Line: token.Line, Column: token.Column},
		End:    &analysis.Position{Line: token.EndLine, Column: token.EndColumn},
		Symbol: symbol,
		Kind:   kind,
	}
}

func nextToken(tokens []tsToken, index int) tsToken {
	index = nextNonNewlineIndex(tokens, index)
	if index >= len(tokens) {
		return tsToken{}
	}
	return tokens[index]
}

func nextNonNewlineIndex(tokens []tsToken, index int) int {
	for index < len(tokens) && tokens[index].Kind == tsTokenNewline {
		index++
	}
	return index
}

func previousToken(tokens []tsToken, index int) tsToken {
	for index >= 0 && tokens[index].Kind == tsTokenNewline {
		index--
	}
	if index < 0 {
		return tsToken{}
	}
	return tokens[index]
}

func findTSCallEnd(tokens []tsToken, openIndex int) (int, bool) {
	depth := 0
	for index := openIndex; index < len(tokens); index++ {
		if tokens[index].Kind != tsTokenPunctuation {
			continue
		}
		switch tokens[index].Text {
		case "(":
			depth++
		case ")":
			depth--
			if depth == 0 {
				return index + 1, true
			}
		}
	}
	return len(tokens), false
}

func joinTSTokens(tokens []tsToken) string {
	parts := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if token.Kind == tsTokenNewline {
			continue
		}
		if token.Raw != "" {
			parts = append(parts, token.Raw)
		} else {
			parts = append(parts, token.Text)
		}
	}
	return strings.Join(parts, "")
}

func tsImportSyntaxDiagnostic(repositoryPath, message string, token tsToken) analysis.Diagnostic {
	return analysis.Diagnostic{
		Code:        "typescript_import_syntax",
		Severity:    "warning",
		Message:     message,
		Path:        repositoryPath,
		Location:    &analysis.Position{Line: token.Line, Column: token.Column},
		Recoverable: true,
	}
}
