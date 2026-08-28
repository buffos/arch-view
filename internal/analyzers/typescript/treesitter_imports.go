package tsanalyzer

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/syntax"
)

type treeSitterTSImportExtractor struct {
	provider syntax.Provider
}

var _ tsImportExtractor = treeSitterTSImportExtractor{}

func (extractor treeSitterTSImportExtractor) Extract(ctx context.Context, repositoryPath, content, fromModuleID string) tsImportExtraction {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return tsImportExtraction{}
	}
	if extractor.provider == nil {
		return tsImportExtraction{BackendError: errors.New("tree-sitter syntax provider is not configured")}
	}

	parsed, err := extractor.provider.Parse(ctx, syntax.Source{Path: repositoryPath, Content: []byte(content)})
	if err != nil {
		if ctx.Err() != nil {
			return tsImportExtraction{}
		}
		return tsImportExtraction{BackendError: fmt.Errorf("tree-sitter could not parse TypeScript source: %w", err)}
	}
	defer parsed.Close()

	diagnostics := make([]analysis.Diagnostic, 0, len(parsed.Issues))
	for _, issue := range parsed.Issues {
		diagnostics = append(diagnostics, tsSyntaxDiagnostic(repositoryPath, issue.Message, issue.Range))
	}
	if parsed.Tree == nil {
		return tsImportExtraction{
			Diagnostics:  diagnostics,
			BackendError: errors.New("tree-sitter returned no syntax tree"),
		}
	}

	observations := make([]tsImportObservation, 0)
	ordinal := 0
	syntax.Walk(parsed.Tree.Root(), func(node syntax.Node) bool {
		if err := ctx.Err(); err != nil {
			return false
		}
		switch node.Type() {
		case "import_statement":
			observation, ok := extractTSSyntaxModule(node, repositoryPath, fromModuleID, false, ordinal)
			if !ok {
				return true
			}
			observations = append(observations, observation)
			ordinal++
			return false
		case "export_statement":
			observation, ok := extractTSSyntaxModule(node, repositoryPath, fromModuleID, true, ordinal)
			if !ok {
				return true
			}
			observations = append(observations, observation)
			ordinal++
			return false
		case "call_expression":
			observation, recognized, ok := extractTSSyntaxCall(node, repositoryPath, fromModuleID, ordinal)
			if !recognized {
				return true
			}
			if !ok {
				diagnostics = append(diagnostics, tsSyntaxDiagnostic(repositoryPath, "TypeScript dependency call could not be interpreted statically.", node.Range()))
				return false
			}
			observations = append(observations, observation)
			ordinal++
			return false
		default:
			return true
		}
	})
	return tsImportExtraction{Observations: observations, Diagnostics: diagnostics}
}

func extractTSSyntaxModule(node syntax.Node, repositoryPath, fromModuleID string, reexport bool, ordinal int) (tsImportObservation, bool) {
	source := node.ChildByFieldName("source")
	if source == nil || source.Type() != "string" {
		return tsImportObservation{}, false
	}
	raw := source.Text()
	if len(raw) == 0 {
		return tsImportObservation{}, false
	}

	specifier := decodeTSString(raw, raw[0])
	kind := "import"
	sourceKind := "import"
	if reexport {
		kind = "reexport"
		sourceKind = "export"
	}
	typeOnly := tsSyntaxTypeModifier(node, reexport)
	importNames, aliases := tsSyntaxImportNames(node, reexport)
	if !typeOnly && tsSyntaxHasOnlyTypeNamedSpecifiers(node, reexport) {
		typeOnly = true
	}
	if typeOnly && !reexport {
		kind = "type_import"
	}
	sourceReference := tsSyntaxImportSource(repositoryPath, source, specifier, "import", ordinal)
	sourceReference.Kind = sourceKind

	return tsImportObservation{
		FromModuleID: fromModuleID,
		Source:       sourceReference,
		Specifier:    specifier,
		Kind:         kind,
		ImportNames:  importNames,
		Aliases:      aliases,
		TypeOnly:     typeOnly,
		Reexport:     reexport,
	}, true
}

func extractTSSyntaxCall(node syntax.Node, repositoryPath, fromModuleID string, ordinal int) (tsImportObservation, bool, bool) {
	function := node.ChildByFieldName("function")
	if function == nil {
		return tsImportObservation{}, false, false
	}
	kind := ""
	switch {
	case function.Type() == "import":
		kind = "dynamic_import"
	case function.Type() == "identifier" && function.Text() == "require":
		kind = "require"
	default:
		return tsImportObservation{}, false, false
	}

	arguments := node.ChildByFieldName("arguments")
	if arguments == nil || arguments.NamedChildCount() == 0 {
		return tsImportObservation{}, true, false
	}
	argument := arguments.NamedChild(0)
	if argument == nil {
		return tsImportObservation{}, true, false
	}
	namedArgumentCount := arguments.NamedChildCount()
	literal := argument.Type() == "string" && (namedArgumentCount == 1 || kind == "dynamic_import" && namedArgumentCount >= 2)
	specifier := argument.Text()
	computed := !literal
	expression := argument.Text()
	if literal && len(specifier) > 0 {
		specifier = decodeTSString(specifier, specifier[0])
	}
	sourceNode := tsSyntaxFirstToken(argument)
	if computed {
		specifier = "<computed>"
		expression = tsSyntaxCompactArguments(arguments)
	}
	if sourceNode == nil {
		sourceNode = argument
	}

	return tsImportObservation{
		FromModuleID: fromModuleID,
		Source:       tsSyntaxImportSource(repositoryPath, sourceNode, specifier, kind, ordinal),
		Specifier:    specifier,
		Kind:         kind,
		Dynamic:      kind == "dynamic_import" || computed,
		Computed:     computed,
		Expression:   expression,
	}, true, true
}

func tsSyntaxTypeModifier(statement syntax.Node, reexport bool) bool {
	clauseType := "import_clause"
	if reexport {
		clauseType = "export_clause"
	}
	clause := tsSyntaxFirstDirectChild(statement, clauseType)
	if clause == nil {
		return false
	}
	clauseStart := clause.Range().StartByte
	for index := 0; index < statement.ChildCount(); index++ {
		child := statement.Child(index)
		if child != nil && child.Type() == "type" && child.Range().EndByte <= clauseStart {
			return true
		}
	}
	return false
}

func tsSyntaxImportNames(statement syntax.Node, reexport bool) ([]string, []string) {
	clauseType := "import_clause"
	if reexport {
		clauseType = "export_clause"
	}
	clause := tsSyntaxFirstDirectChild(statement, clauseType)
	names := make([]string, 0)
	aliases := make([]string, 0)
	if clause != nil {
		collectTSSyntaxNames(clause, reexport, &names, &aliases)
	}
	if reexport {
		if namespace := tsSyntaxFirstDirectChild(statement, "namespace_export"); namespace != nil {
			if identifier := tsSyntaxFirstDirectChild(namespace, "identifier"); identifier != nil {
				aliases = appendUniqueString(aliases, identifier.Text())
			}
		}
	}
	return sortStringSet(names), sortStringSet(aliases)
}

func collectTSSyntaxNames(node syntax.Node, reexport bool, names, aliases *[]string) {
	if node == nil {
		return
	}
	switch node.Type() {
	case "import_specifier", "export_specifier":
		name := node.ChildByFieldName("name")
		alias := node.ChildByFieldName("alias")
		if name != nil && name.Type() == "identifier" && (!reexport || name.Text() != "default") {
			*names = appendUniqueString(*names, name.Text())
		}
		if alias != nil && alias.Type() == "identifier" {
			*aliases = appendUniqueString(*aliases, alias.Text())
		}
		return
	case "namespace_import", "namespace_export":
		if identifier := tsSyntaxFirstDirectChild(node, "identifier"); identifier != nil {
			*aliases = appendUniqueString(*aliases, identifier.Text())
		}
		return
	case "identifier":
		*names = appendUniqueString(*names, node.Text())
		return
	}
	for index := 0; index < node.NamedChildCount(); index++ {
		collectTSSyntaxNames(node.NamedChild(index), reexport, names, aliases)
	}
}

func tsSyntaxHasOnlyTypeNamedSpecifiers(statement syntax.Node, reexport bool) bool {
	clauseType := "import_clause"
	specifierType := "named_imports"
	if reexport {
		clauseType = "export_clause"
		specifierType = "export_clause"
	}
	clause := tsSyntaxFirstDirectChild(statement, clauseType)
	if clause == nil {
		return false
	}
	specifierContainer := clause
	if !reexport {
		specifierContainer = tsSyntaxFirstDescendant(clause, specifierType)
		if specifierContainer == nil {
			return false
		}
	}
	count := 0
	allTypeOnly := true
	var visit func(syntax.Node)
	visit = func(node syntax.Node) {
		if node == nil || !allTypeOnly {
			return
		}
		if node.Type() == "import_specifier" || node.Type() == "export_specifier" {
			count++
			hasType := false
			for index := 0; index < node.ChildCount(); index++ {
				if child := node.Child(index); child != nil && child.Type() == "type" {
					hasType = true
					break
				}
			}
			if !hasType {
				allTypeOnly = false
			}
			return
		}
		for index := 0; index < node.NamedChildCount(); index++ {
			visit(node.NamedChild(index))
		}
	}
	visit(specifierContainer)
	return count > 0 && allTypeOnly
}

func tsSyntaxFirstDirectChild(node syntax.Node, wanted string) syntax.Node {
	if node == nil {
		return nil
	}
	for index := 0; index < node.ChildCount(); index++ {
		child := node.Child(index)
		if child != nil && child.Type() == wanted {
			return child
		}
	}
	return nil
}

func tsSyntaxFirstDescendant(node syntax.Node, wanted string) syntax.Node {
	if node == nil {
		return nil
	}
	if node.Type() == wanted {
		return node
	}
	for index := 0; index < node.NamedChildCount(); index++ {
		if found := tsSyntaxFirstDescendant(node.NamedChild(index), wanted); found != nil {
			return found
		}
	}
	return nil
}

func tsSyntaxFirstToken(node syntax.Node) syntax.Node {
	if node == nil {
		return nil
	}
	switch node.Type() {
	case "string", "template_string", "regex", "number", "identifier", "property_identifier", "type_identifier":
		return node
	}
	if node.ChildCount() == 0 {
		return node
	}
	for index := 0; index < node.ChildCount(); index++ {
		if child := tsSyntaxFirstToken(node.Child(index)); child != nil {
			return child
		}
	}
	return node
}

func tsSyntaxCompactArguments(arguments syntax.Node) string {
	if arguments == nil {
		return ""
	}
	var builder strings.Builder
	for index := 0; index < arguments.ChildCount(); index++ {
		if index == 0 || index == arguments.ChildCount()-1 {
			continue
		}
		tsSyntaxAppendCompact(arguments.Child(index), &builder)
	}
	return builder.String()
}

func tsSyntaxAppendCompact(node syntax.Node, builder *strings.Builder) {
	if node == nil || node.Type() == "comment" {
		return
	}
	if node.ChildCount() == 0 {
		builder.WriteString(node.Text())
		return
	}
	for index := 0; index < node.ChildCount(); index++ {
		tsSyntaxAppendCompact(node.Child(index), builder)
	}
}

func tsSyntaxImportSource(repositoryPath string, node syntax.Node, symbol, kind string, ordinal int) analysis.SourceReference {
	rangeValue := node.Range()
	return analysis.SourceReference{
		ID:     stableTSID("source", repositoryPath, strconv.Itoa(int(rangeValue.Start.Row)+1), strconv.Itoa(int(rangeValue.Start.Column)+1), kind, symbol, strconv.Itoa(ordinal)),
		Path:   repositoryPath,
		Start:  &analysis.Position{Line: int(rangeValue.Start.Row) + 1, Column: int(rangeValue.Start.Column) + 1},
		End:    &analysis.Position{Line: int(rangeValue.End.Row) + 1, Column: int(rangeValue.End.Column) + 1},
		Symbol: symbol,
		Kind:   kind,
	}
}

func tsSyntaxDiagnostic(repositoryPath, message string, rangeValue syntax.Range) analysis.Diagnostic {
	return analysis.Diagnostic{
		Code:        "typescript_import_syntax",
		Severity:    "warning",
		Message:     message,
		Path:        repositoryPath,
		Location:    &analysis.Position{Line: int(rangeValue.Start.Row) + 1, Column: int(rangeValue.Start.Column) + 1},
		Recoverable: true,
	}
}
