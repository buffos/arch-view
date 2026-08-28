package scanner

import (
	"fmt"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/syntax"
)

func goSyntaxPackageName(root syntax.Node) string {
	if root == nil {
		return ""
	}
	var packageName string
	syntax.Walk(root, func(node syntax.Node) bool {
		if node.Type() != "package_clause" {
			return true
		}
		for index := 0; index < node.NamedChildCount(); index++ {
			child := node.NamedChild(index)
			if child != nil && child.Type() == "package_identifier" {
				packageName = strings.TrimSpace(child.Text())
				break
			}
		}
		return false
	})
	return packageName
}

func goSyntaxImportSpecs(root syntax.Node) []syntax.Node {
	imports := make([]syntax.Node, 0)
	syntax.Walk(root, func(node syntax.Node) bool {
		if node.Type() == "import_spec" {
			imports = append(imports, node)
			return false
		}
		return true
	})
	return imports
}

func goSyntaxIssueDiagnostic(path string, issue syntax.Issue) analysis.Diagnostic {
	return analysis.Diagnostic{
		Code:        "go_parse_error",
		Severity:    "error",
		Message:     fmt.Sprintf("Go source could not be parsed: %s", issue.Message),
		Path:        path,
		Location:    &analysis.Position{Line: int(issue.Range.Start.Row) + 1, Column: int(issue.Range.Start.Column) + 1},
		Recoverable: true,
	}
}

func goSyntaxBackendDiagnostic(path string, err error) analysis.Diagnostic {
	message := "Go Tree-sitter syntax backend failed."
	if err != nil {
		message += " " + err.Error()
	}
	return analysis.Diagnostic{
		Code:        "go_syntax_backend",
		Severity:    "warning",
		Message:     message,
		Path:        path,
		Location:    &analysis.Position{Line: 1, Column: 1},
		Recoverable: true,
	}
}
