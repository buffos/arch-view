package tsanalyzer

import (
	"strings"

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
