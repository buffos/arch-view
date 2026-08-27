package tsanalyzer

import "bytes"

func isJSONNull(value []byte) bool {
	return bytes.Equal(bytes.TrimSpace(value), []byte("null"))
}

// stripJSONC removes JSONC comments and trailing commas while preserving
// strings. tsconfig files are commonly JSON-with-comments, and this parser is
// deliberately limited to syntax needed for static configuration data.
func stripJSONC(data []byte) []byte {
	if bytes.HasPrefix(data, []byte{0xEF, 0xBB, 0xBF}) {
		data = data[3:]
	}
	result := make([]byte, 0, len(data))
	inString := false
	escaped := false
	for index := 0; index < len(data); index++ {
		char := data[index]
		if inString {
			result = append(result, char)
			if escaped {
				escaped = false
			} else if char == '\\' {
				escaped = true
			} else if char == '"' {
				inString = false
			}
			continue
		}
		switch char {
		case '"':
			inString = true
			result = append(result, char)
		case '/':
			if index+1 < len(data) && data[index+1] == '/' {
				index += 2
				for index < len(data) && data[index] != '\n' && data[index] != '\r' {
					index++
				}
				index--
			} else if index+1 < len(data) && data[index+1] == '*' {
				index += 2
				for index+1 < len(data) && (data[index] != '*' || data[index+1] != '/') {
					if data[index] == '\n' || data[index] == '\r' {
						result = append(result, data[index])
					}
					index++
				}
				if index+1 < len(data) {
					index++
				}
			} else {
				result = append(result, char)
			}
		default:
			result = append(result, char)
		}
	}
	return stripTrailingJSONCommas(result)
}

func stripTrailingJSONCommas(data []byte) []byte {
	result := make([]byte, 0, len(data))
	inString := false
	escaped := false
	for index := 0; index < len(data); index++ {
		char := data[index]
		if inString {
			result = append(result, char)
			if escaped {
				escaped = false
			} else if char == '\\' {
				escaped = true
			} else if char == '"' {
				inString = false
			}
			continue
		}
		if char == '"' {
			inString = true
			result = append(result, char)
			continue
		}
		if char != ',' {
			result = append(result, char)
			continue
		}
		next := index + 1
		for next < len(data) && (data[next] == ' ' || data[next] == '\t' || data[next] == '\r' || data[next] == '\n') {
			next++
		}
		if next < len(data) && (data[next] == '}' || data[next] == ']') {
			continue
		}
		result = append(result, char)
	}
	return result
}
