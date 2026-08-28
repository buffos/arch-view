package pyanalyzer

import (
	"context"

	"github.com/buffo/arch-view/internal/analysis/syntax"
	pysyntax "github.com/buffo/arch-view/internal/analysis/syntax/python"
)

type syntaxIssue struct {
	Line   int
	Column int
}

func validatePythonSyntax(content string) *syntaxIssue {
	result, err := pysyntax.NewProvider().Parse(context.Background(), syntax.Source{Path: "<configuration>", Content: []byte(content)})
	if err != nil {
		return &syntaxIssue{Line: 1, Column: 1}
	}
	defer result.Close()
	if len(result.Issues) == 0 {
		return nil
	}
	start := result.Issues[0].Range.Start
	return &syntaxIssue{Line: int(start.Row) + 1, Column: int(start.Column) + 1}
}
