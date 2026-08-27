package clojureanalyzer

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/buffo/arch-view/internal/analysis"
)

type formKind uint8

const (
	formAtom formKind = iota
	formList
	formVector
	formMap
	formSet
	formReaderConditional
)

type atomKind uint8

const (
	atomSymbol atomKind = iota
	atomKeyword
	atomString
	atomNumber
	atomCharacter
	atomOther
)

type cljForm struct {
	Kind         formKind
	Atom         string
	AtomKind     atomKind
	Items        []*cljForm
	Start        *analysis.Position
	End          *analysis.Position
	ReaderSplice bool
	Quoted       bool
}

type syntaxIssue struct {
	Line   int
	Column int
}

type cljTokenKind uint8

const (
	cljTokenAtom cljTokenKind = iota
	cljTokenString
	cljTokenOpen
	cljTokenClose
	cljTokenReaderConditional
	cljTokenPrefix
)

type cljToken struct {
	Kind  cljTokenKind
	Text  string
	Start *analysis.Position
	End   *analysis.Position
}

// parseClojureForms parses the data-shaped portion of Clojure syntax needed by
// the adapter. It intentionally does not evaluate forms or expand macros.
func parseClojureForms(content string) ([]*cljForm, []syntaxIssue) {
	tokens, issues := lexClojure(content)
	state := cljParser{tokens: tokens, issues: issues}
	forms := make([]*cljForm, 0)
	for state.index < len(state.tokens) {
		before := state.index
		form := state.parseForm()
		if form != nil {
			forms = append(forms, form)
		}
		if state.index == before {
			state.index++
		}
	}
	return forms, state.issues
}

type cljParser struct {
	tokens []cljToken
	index  int
	issues []syntaxIssue
}

func (p *cljParser) parseForm() *cljForm {
	if p.index >= len(p.tokens) {
		return nil
	}
	token := p.tokens[p.index]
	p.index++
	switch token.Kind {
	case cljTokenAtom:
		return &cljForm{Kind: formAtom, Atom: token.Text, AtomKind: classifyAtom(token.Text), Start: token.Start, End: token.End}
	case cljTokenString:
		return &cljForm{Kind: formAtom, Atom: token.Text, AtomKind: atomString, Start: token.Start, End: token.End}
	case cljTokenOpen:
		return p.parseCollection(token)
	case cljTokenClose:
		p.issues = append(p.issues, syntaxIssue{Line: token.Start.Line, Column: token.Start.Column})
		return nil
	case cljTokenReaderConditional:
		child := p.parseForm()
		if child == nil || (child.Kind != formList && child.Kind != formVector) {
			p.issues = append(p.issues, syntaxIssue{Line: token.Start.Line, Column: token.Start.Column})
			return &cljForm{Kind: formReaderConditional, Start: token.Start, End: token.End, ReaderSplice: token.Text == "#?@"}
		}
		return &cljForm{Kind: formReaderConditional, Items: child.Items, Start: token.Start, End: child.End, ReaderSplice: token.Text == "#?@"}
	case cljTokenPrefix:
		if token.Text == "#_" {
			// Discarded forms are still parsed so delimiters and syntax errors are
			// accounted for, but they never become observations.
			if p.parseForm() == nil && p.index >= len(p.tokens) {
				p.issues = append(p.issues, syntaxIssue{Line: token.Start.Line, Column: token.Start.Column})
			}
			return nil
		}
		if token.Text == "^" {
			// Metadata is not part of the architecture observation, but the
			// following form must still be consumed before the target form.
			_ = p.parseForm()
		}
		child := p.parseForm()
		if child == nil {
			p.issues = append(p.issues, syntaxIssue{Line: token.Start.Line, Column: token.Start.Column})
			return nil
		}
		if child.Start == nil || child.Start.Line > token.Start.Line || (child.Start.Line == token.Start.Line && child.Start.Column > token.Start.Column) {
			child.Start = token.Start
		}
		if token.Text == "'" || token.Text == "`" || token.Text == "#'" {
			child.Quoted = true
		}
		return child
	default:
		return nil
	}
}

func (p *cljParser) parseCollection(open cljToken) *cljForm {
	collectionKind := formList
	closing := ")"
	switch open.Text {
	case "[":
		collectionKind = formVector
		closing = "]"
	case "{":
		collectionKind = formMap
		closing = "}"
	case "#{":
		collectionKind = formSet
		closing = "}"
	}
	form := &cljForm{Kind: collectionKind, Items: make([]*cljForm, 0), Start: open.Start, End: open.End}
	for p.index < len(p.tokens) {
		token := p.tokens[p.index]
		if token.Kind == cljTokenClose {
			p.index++
			if token.Text != closing {
				p.issues = append(p.issues, syntaxIssue{Line: token.Start.Line, Column: token.Start.Column})
				return form
			}
			form.End = token.End
			return form
		}
		before := p.index
		child := p.parseForm()
		if child != nil {
			form.Items = append(form.Items, child)
		}
		if p.index == before {
			p.index++
		}
	}
	p.issues = append(p.issues, syntaxIssue{Line: open.Start.Line, Column: open.Start.Column})
	return form
}

func lexClojure(content string) ([]cljToken, []syntaxIssue) {
	if !utf8.ValidString(content) {
		return nil, []syntaxIssue{{Line: 1, Column: 1}}
	}
	tokens := make([]cljToken, 0)
	issues := make([]syntaxIssue, 0)
	index, line, column := 0, 1, 1
	for index < len(content) {
		value, size := utf8.DecodeRuneInString(content[index:])
		if value == utf8.RuneError && size == 1 {
			issues = append(issues, syntaxIssue{Line: line, Column: column})
			index++
			column++
			continue
		}
		if unicode.IsSpace(value) || value == ',' {
			index, line, column = advanceClojurePosition(content, index, line, column)
			continue
		}
		if value == ';' {
			for index < len(content) && content[index] != '\n' {
				index, line, column = advanceClojurePosition(content, index, line, column)
			}
			continue
		}
		start := &analysis.Position{Line: line, Column: column}
		if value == '"' {
			text, next, nextLine, nextColumn, ok := scanClojureString(content, index, line, column)
			if !ok {
				issues = append(issues, syntaxIssue{Line: line, Column: column})
				return tokens, issues
			}
			tokens = append(tokens, cljToken{Kind: cljTokenString, Text: text, Start: start, End: &analysis.Position{Line: nextLine, Column: nextColumn}})
			index, line, column = next, nextLine, nextColumn
			continue
		}
		if value == '\\' {
			index, line, column = advanceClojurePosition(content, index, line, column)
			begin := index
			for index < len(content) {
				runeValue, _ := utf8.DecodeRuneInString(content[index:])
				if unicode.IsSpace(runeValue) || strings.ContainsRune("()[]{}\";,'`^@~", runeValue) {
					break
				}
				index, line, column = advanceClojurePosition(content, index, line, column)
			}
			tokens = append(tokens, cljToken{Kind: cljTokenAtom, Text: "\\" + content[begin:index], Start: start, End: &analysis.Position{Line: line, Column: column}})
			continue
		}
		if open := clojureOpeningDelimiter(content, index); open != "" {
			tokens = append(tokens, cljToken{Kind: cljTokenOpen, Text: open, Start: start, End: advancePositionCopy(start, len([]rune(open)))})
			for count := 0; count < len([]rune(open)); count++ {
				index, line, column = advanceClojurePosition(content, index, line, column)
			}
			continue
		}
		if close := clojureClosingDelimiter(value); close != "" {
			tokens = append(tokens, cljToken{Kind: cljTokenClose, Text: close, Start: start, End: advancePositionCopy(start, 1)})
			index, line, column = advanceClojurePosition(content, index, line, column)
			continue
		}
		if prefix := clojureReaderConditional(content, index); prefix != "" {
			tokens = append(tokens, cljToken{Kind: cljTokenReaderConditional, Text: prefix, Start: start, End: advancePositionCopy(start, len([]rune(prefix)))})
			for count := 0; count < len([]rune(prefix)); count++ {
				index, line, column = advanceClojurePosition(content, index, line, column)
			}
			continue
		}
		if prefix := clojurePrefix(content, index); prefix != "" {
			tokens = append(tokens, cljToken{Kind: cljTokenPrefix, Text: prefix, Start: start, End: advancePositionCopy(start, len([]rune(prefix)))})
			for count := 0; count < len([]rune(prefix)); count++ {
				index, line, column = advanceClojurePosition(content, index, line, column)
			}
			continue
		}
		begin := index
		for index < len(content) {
			runeValue, _ := utf8.DecodeRuneInString(content[index:])
			if unicode.IsSpace(runeValue) || runeValue == ',' || strings.ContainsRune("()[]{}\";,'`^@~", runeValue) {
				break
			}
			index, line, column = advanceClojurePosition(content, index, line, column)
		}
		if begin == index {
			index, line, column = advanceClojurePosition(content, index, line, column)
			continue
		}
		raw := content[begin:index]
		tokens = append(tokens, cljToken{Kind: cljTokenAtom, Text: raw, Start: start, End: &analysis.Position{Line: line, Column: column}})
	}
	return tokens, issues
}

func scanClojureString(content string, index, line, column int) (string, int, int, int, bool) {
	begin := index
	index, line, column = advanceClojurePosition(content, index, line, column)
	escaped := false
	for index < len(content) {
		value, _ := utf8.DecodeRuneInString(content[index:])
		if escaped {
			escaped = false
			index, line, column = advanceClojurePosition(content, index, line, column)
			continue
		}
		if value == '\\' {
			escaped = true
			index, line, column = advanceClojurePosition(content, index, line, column)
			continue
		}
		if value == '"' {
			index, line, column = advanceClojurePosition(content, index, line, column)
			raw := content[begin:index]
			decoded, err := strconv.Unquote(raw)
			if err != nil {
				decoded = strings.TrimSuffix(strings.TrimPrefix(raw, "\""), "\"")
			}
			return decoded, index, line, column, true
		}
		index, line, column = advanceClojurePosition(content, index, line, column)
	}
	return "", index, line, column, false
}

func advanceClojurePosition(content string, index, line, column int) (int, int, int) {
	value, size := utf8.DecodeRuneInString(content[index:])
	if value == '\n' {
		return index + size, line + 1, 1
	}
	return index + size, line, column + 1
}

func advancePositionCopy(position *analysis.Position, columns int) *analysis.Position {
	if position == nil {
		return nil
	}
	return &analysis.Position{Line: position.Line, Column: position.Column + columns}
}

func clojureOpeningDelimiter(content string, index int) string {
	if strings.HasPrefix(content[index:], "#{") {
		return "#{"
	}
	switch content[index] {
	case '(':
		return "("
	case '[':
		return "["
	case '{':
		return "{"
	default:
		return ""
	}
}

func clojureClosingDelimiter(value rune) string {
	switch value {
	case ')':
		return ")"
	case ']':
		return "]"
	case '}':
		return "}"
	default:
		return ""
	}
}

func clojureReaderConditional(content string, index int) string {
	for _, candidate := range []string{"#?@", "#?"} {
		if strings.HasPrefix(content[index:], candidate) {
			return candidate
		}
	}
	return ""
}

func clojurePrefix(content string, index int) string {
	for _, candidate := range []string{"#_", "#'", "~@", "'", "`", "~", "^", "@", "#"} {
		if strings.HasPrefix(content[index:], candidate) {
			return candidate
		}
	}
	return ""
}

func classifyAtom(value string) atomKind {
	if strings.HasPrefix(value, ":") {
		return atomKeyword
	}
	if value == "nil" || value == "true" || value == "false" {
		return atomOther
	}
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		return atomNumber
	}
	if strings.HasPrefix(value, "\\") {
		return atomCharacter
	}
	return atomSymbol
}

func formSymbol(form *cljForm) string {
	if form == nil || form.Kind != formAtom || form.AtomKind != atomSymbol {
		return ""
	}
	return form.Atom
}

func formKeyword(form *cljForm) string {
	if form == nil || form.Kind != formAtom || form.AtomKind != atomKeyword {
		return ""
	}
	return form.Atom
}

func formString(form *cljForm) (string, bool) {
	if form == nil || form.Kind != formAtom || form.AtomKind != atomString {
		return "", false
	}
	return form.Atom, true
}

func validClojureNamespace(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "/()[]{}\"';, \t\r\n") || strings.HasPrefix(value, ":") || strings.HasPrefix(value, ".") || strings.HasSuffix(value, ".") {
		return false
	}
	for _, part := range strings.Split(value, ".") {
		if part == "" {
			return false
		}
	}
	return true
}
