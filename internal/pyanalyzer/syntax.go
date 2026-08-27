package pyanalyzer

import "unicode/utf8"

type syntaxIssue struct {
	Line   int
	Column int
}

// validatePythonSyntax is intentionally a conservative lexical check. The
// full Python parser belongs to import-resolution issue 018; this check still
// identifies malformed strings/brackets and invalid UTF-8 without executing
// or embedding a Python runtime.
func validatePythonSyntax(content string) *syntaxIssue {
	if !utf8.ValidString(content) {
		return &syntaxIssue{Line: 1, Column: 1}
	}
	line, column := 1, 1
	stack := []byte{}
	quote := byte(0)
	triple := false
	escaped := false
	for index := 0; index < len(content); {
		if content[index] == 0 {
			return &syntaxIssue{Line: line, Column: column}
		}
		if quote != 0 {
			if escaped {
				escaped = false
				index++
				column++
				continue
			}
			if content[index] == '\\' {
				escaped = true
				index++
				column++
				continue
			}
			if triple && index+2 < len(content) && content[index] == quote && content[index+1] == quote && content[index+2] == quote {
				quote = 0
				triple = false
				index += 3
				column += 3
				continue
			}
			if !triple && content[index] == quote {
				quote = 0
				index++
				column++
				continue
			}
			line, column, index = advancePythonPosition(content, index, line, column)
			continue
		}

		if content[index] == '#' {
			for index < len(content) && content[index] != '\n' {
				index++
				column++
			}
			continue
		}
		if (content[index] == 'r' || content[index] == 'R' || content[index] == 'u' || content[index] == 'U' || content[index] == 'f' || content[index] == 'F' || content[index] == 'b' || content[index] == 'B') && index+1 < len(content) && (content[index+1] == '\'' || content[index+1] == '"') {
			index++
			column++
		}
		if content[index] == '\'' || content[index] == '"' {
			quote = content[index]
			triple = index+2 < len(content) && content[index+1] == quote && content[index+2] == quote
			if triple {
				index += 3
				column += 3
			} else {
				index++
				column++
			}
			continue
		}
		switch content[index] {
		case '(', '[', '{':
			stack = append(stack, content[index])
		case ')', ']', '}':
			if len(stack) == 0 || !matchingDelimiter(stack[len(stack)-1], content[index]) {
				return &syntaxIssue{Line: line, Column: column}
			}
			stack = stack[:len(stack)-1]
		}
		line, column, index = advancePythonPosition(content, index, line, column)
	}
	if quote != 0 || len(stack) != 0 {
		return &syntaxIssue{Line: line, Column: column}
	}
	return nil
}

func advancePythonPosition(content string, index, line, column int) (int, int, int) {
	if content[index] == '\n' {
		return line + 1, 1, index + 1
	}
	return line, column + 1, index + 1
}

func matchingDelimiter(open, close byte) bool {
	return (open == '(' && close == ')') || (open == '[' && close == ']') || (open == '{' && close == '}')
}
