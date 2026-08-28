package pyanalyzer

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/syntax"
)

type pythonSyntaxExtraction struct {
	Observations []pythonImportObservation
	Diagnostics  []analysis.Diagnostic
	BackendError error
}

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

func extractPythonSyntaxImports(ctx context.Context, provider syntax.Provider, path, content, fromModuleID string, packageInit bool) pythonSyntaxExtraction {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return pythonSyntaxExtraction{}
	}
	if provider == nil {
		return pythonSyntaxExtraction{BackendError: errors.New("python syntax provider is not configured")}
	}
	parsed, err := provider.Parse(ctx, syntax.Source{Path: path, Content: []byte(content)})
	if err != nil {
		if ctx.Err() != nil {
			return pythonSyntaxExtraction{}
		}
		return pythonSyntaxExtraction{BackendError: fmt.Errorf("tree-sitter could not parse Python source: %w", err)}
	}
	defer parsed.Close()

	result := pythonSyntaxExtraction{
		Observations: make([]pythonImportObservation, 0),
		Diagnostics:  make([]analysis.Diagnostic, 0, len(parsed.Issues)),
	}
	for _, issue := range parsed.Issues {
		result.Diagnostics = append(result.Diagnostics, pythonSyntaxDiagnostic(path, issue.Message, issue.Range))
	}
	if parsed.Tree == nil || parsed.Tree.Root() == nil {
		result.BackendError = errors.New("tree-sitter returned no Python syntax tree")
		return result
	}

	walker := pythonSyntaxWalker{
		ctx:         ctx,
		path:        path,
		fromModule:  fromModuleID,
		packageInit: packageInit,
		aliases:     make(map[string]struct{}),
		result:      &result,
	}
	walker.walk(parsed.Tree.Root(), nil)
	return result
}

type pythonSyntaxWalker struct {
	ctx         context.Context
	path        string
	fromModule  string
	packageInit bool
	aliases     map[string]struct{}
	result      *pythonSyntaxExtraction
}

func (walker *pythonSyntaxWalker) walk(node syntax.Node, conditions []string) {
	if node == nil || walker.ctx.Err() != nil {
		return
	}

	switch node.Type() {
	case "if_statement":
		walker.walkIfStatement(node, conditions)
		return
	case "elif_clause":
		walker.walkConditionalClause(node, conditions, "elif")
		return
	case "else_clause":
		walker.walkElseClause(node, conditions)
		return
	case "import_statement":
		observations := extractPythonSyntaxImportStatement(node, walker.path, walker.fromModule, conditions)
		walker.appendObservations(observations)
		registerPythonDynamicAliases(walker.aliases, observations)
		return
	case "import_from_statement":
		observations := extractPythonSyntaxFromStatement(node, walker.path, walker.fromModule, walker.packageInit, conditions)
		walker.appendObservations(observations)
		registerPythonDynamicAliases(walker.aliases, observations)
		return
	case "future_import_statement":
		observations := extractPythonSyntaxFutureStatement(node, walker.path, walker.fromModule, conditions)
		walker.appendObservations(observations)
		registerPythonDynamicAliases(walker.aliases, observations)
		return
	case "call":
		if observation, ok := extractPythonSyntaxDynamicCall(node, walker.path, walker.fromModule, conditions, walker.aliases); ok {
			walker.appendObservations([]pythonImportObservation{observation})
			return
		}
	}

	for index := 0; index < node.ChildCount(); index++ {
		walker.walk(node.Child(index), conditions)
	}
}

func (walker *pythonSyntaxWalker) walkIfStatement(node syntax.Node, conditions []string) {
	condition := node.ChildByFieldName("condition")
	conditionText := "if:"
	if condition != nil {
		conditionText = "if " + strings.TrimSpace(condition.Text()) + ":"
	}
	if consequence := node.ChildByFieldName("consequence"); consequence != nil {
		walker.walk(consequence, appendPythonCondition(conditions, conditionText))
	}
	for index := 0; index < node.ChildCount(); index++ {
		child := node.Child(index)
		if child == nil || child.Type() == "block" || child.Type() == "if_clause" {
			continue
		}
		if child.Type() == "elif_clause" || child.Type() == "else_clause" {
			walker.walk(child, conditions)
		}
	}
}

func (walker *pythonSyntaxWalker) walkConditionalClause(node syntax.Node, conditions []string, keyword string) {
	condition := node.ChildByFieldName("condition")
	conditionText := keyword + ":"
	if condition != nil {
		conditionText = keyword + " " + strings.TrimSpace(condition.Text()) + ":"
	}
	if consequence := pythonSyntaxBlock(node); consequence != nil {
		walker.walk(consequence, appendPythonCondition(conditions, conditionText))
	}
}

func (walker *pythonSyntaxWalker) walkElseClause(node syntax.Node, conditions []string) {
	if consequence := pythonSyntaxBlock(node); consequence != nil {
		walker.walk(consequence, appendPythonCondition(conditions, "else:"))
	}
}

func (walker *pythonSyntaxWalker) appendObservations(observations []pythonImportObservation) {
	walker.result.Observations = append(walker.result.Observations, observations...)
}

func appendPythonCondition(values []string, condition string) []string {
	result := append([]string(nil), values...)
	return append(result, condition)
}

func pythonSyntaxBlock(node syntax.Node) syntax.Node {
	for _, field := range []string{"consequence", "body"} {
		if value := node.ChildByFieldName(field); value != nil {
			return value
		}
	}
	return nil
}

func extractPythonSyntaxImportStatement(node syntax.Node, path, fromModuleID string, conditions []string) []pythonImportObservation {
	result := make([]pythonImportObservation, 0)
	for index := 0; index < node.NamedChildCount(); index++ {
		child := node.NamedChild(index)
		if child == nil {
			continue
		}
		module := child.Text()
		alias := ""
		if child.Type() == "aliased_import" {
			if name := child.ChildByFieldName("name"); name != nil {
				module = name.Text()
			}
			if aliasNode := child.ChildByFieldName("alias"); aliasNode != nil {
				alias = aliasNode.Text()
			}
		}
		if child.Type() != "dotted_name" && child.Type() != "aliased_import" || module == "" {
			continue
		}
		result = append(result, pythonImportObservation{
			FromModuleID: fromModuleID,
			Source:       pythonSyntaxImportSource(path, child, child, module, "import"),
			Spelling:     module,
			Module:       module,
			Alias:        alias,
			Kind:         "import",
			Conditional:  len(conditions) > 0,
			Condition:    strings.Join(conditions, " -> "),
		})
	}
	return result
}

func extractPythonSyntaxFromStatement(node syntax.Node, path, fromModuleID string, packageInit bool, conditions []string) []pythonImportObservation {
	moduleNode := node.ChildByFieldName("module_name")
	module, relativeLevel := pythonSyntaxModuleName(moduleNode)
	kind := "from"
	if packageInit {
		kind = "reexport"
	}
	result := make([]pythonImportObservation, 0)
	for index := 0; index < node.NamedChildCount(); index++ {
		child := node.NamedChild(index)
		if child == nil || child.Type() == "wildcard_import" || pythonSyntaxSameNode(child, moduleNode) {
			continue
		}
		importedName, alias := pythonSyntaxImportedName(child)
		if importedName == "" {
			continue
		}
		spelling := strings.Repeat(".", relativeLevel) + module
		if module != "" {
			spelling += "."
		}
		spelling += importedName
		start := moduleNode
		if start == nil {
			start = child
		}
		result = append(result, pythonImportObservation{
			FromModuleID:  fromModuleID,
			Source:        pythonSyntaxImportSource(path, start, child, spelling, "import"),
			Spelling:      spelling,
			Module:        module,
			ImportedName:  importedName,
			Alias:         alias,
			RelativeLevel: relativeLevel,
			Kind:          kind,
			Conditional:   len(conditions) > 0,
			Condition:     strings.Join(conditions, " -> "),
		})
	}
	if len(result) == 0 {
		if wildcard := pythonSyntaxFirstNamedChild(node, "wildcard_import"); wildcard != nil {
			spelling := strings.Repeat(".", relativeLevel) + module
			if module != "" {
				spelling += "."
			}
			spelling += "*"
			start := moduleNode
			if start == nil {
				start = wildcard
			}
			result = append(result, pythonImportObservation{
				FromModuleID:  fromModuleID,
				Source:        pythonSyntaxImportSource(path, start, wildcard, spelling, "import"),
				Spelling:      spelling,
				Module:        module,
				ImportedName:  "*",
				RelativeLevel: relativeLevel,
				Kind:          kind,
				Conditional:   len(conditions) > 0,
				Condition:     strings.Join(conditions, " -> "),
			})
		}
	}
	return result
}

func extractPythonSyntaxFutureStatement(node syntax.Node, path, fromModuleID string, conditions []string) []pythonImportObservation {
	result := make([]pythonImportObservation, 0)
	for index := 0; index < node.NamedChildCount(); index++ {
		child := node.NamedChild(index)
		if child == nil {
			continue
		}
		importedName, alias := pythonSyntaxImportedName(child)
		if importedName == "" {
			continue
		}
		result = append(result, pythonImportObservation{
			FromModuleID: fromModuleID,
			Source:       pythonSyntaxImportSource(path, node, child, "__future__."+importedName, "import"),
			Spelling:     "__future__." + importedName,
			Module:       "__future__",
			ImportedName: importedName,
			Alias:        alias,
			Kind:         "from",
			Conditional:  len(conditions) > 0,
			Condition:    strings.Join(conditions, " -> "),
		})
	}
	return result
}

func extractPythonSyntaxDynamicCall(node syntax.Node, path, fromModuleID string, conditions []string, aliases map[string]struct{}) (pythonImportObservation, bool) {
	function := node.ChildByFieldName("function")
	if function == nil {
		return pythonImportObservation{}, false
	}
	functionName := strings.TrimSpace(function.Text())
	if !knownPythonDynamicCallable(functionName) {
		if _, ok := aliases[functionName]; !ok {
			return pythonImportObservation{}, false
		}
	}
	arguments := node.ChildByFieldName("arguments")
	target := ""
	if arguments != nil {
		target, _ = pythonSyntaxStaticCallTarget(arguments)
	}
	name := target
	if name == "" {
		name = "<dynamic>"
	}
	return pythonImportObservation{
		FromModuleID:    fromModuleID,
		Source:          pythonSyntaxImportSource(path, function, node, name, "dynamic"),
		Spelling:        name,
		Kind:            "dynamic",
		Conditional:     len(conditions) > 0,
		Condition:       strings.Join(conditions, " -> "),
		DynamicFunction: functionName,
		DynamicTarget:   target,
	}, true
}

func pythonSyntaxStaticCallTarget(arguments syntax.Node) (string, bool) {
	if arguments == nil || arguments.NamedChildCount() == 0 {
		return "", false
	}
	node := arguments.NamedChild(0)
	if node == nil {
		return "", false
	}
	if node.Type() == "parenthesized_expression" {
		if inner := node.NamedChild(0); inner != nil {
			node = inner
		}
	}
	if node.Type() == "concatenated_string" {
		var builder strings.Builder
		for index := 0; index < node.NamedChildCount(); index++ {
			child := node.NamedChild(index)
			value, ok := staticPythonString(child.Text())
			if !ok {
				return "", false
			}
			builder.WriteString(value)
		}
		return builder.String(), true
	}
	return staticPythonString(node.Text())
}

func pythonSyntaxModuleName(node syntax.Node) (string, int) {
	if node == nil {
		return "", 0
	}
	value := strings.TrimSpace(node.Text())
	level := len(value) - len(strings.TrimLeft(value, "."))
	return strings.TrimLeft(value, "."), level
}

func pythonSyntaxImportedName(node syntax.Node) (string, string) {
	if node == nil {
		return "", ""
	}
	if node.Type() == "wildcard_import" {
		return "*", ""
	}
	if node.Type() == "aliased_import" {
		name := node.ChildByFieldName("name")
		alias := node.ChildByFieldName("alias")
		if name == nil {
			return "", ""
		}
		aliasValue := ""
		if alias != nil {
			aliasValue = alias.Text()
		}
		return name.Text(), aliasValue
	}
	if node.Type() == "dotted_name" || node.Type() == "identifier" {
		return node.Text(), ""
	}
	return "", ""
}

func pythonSyntaxFirstNamedChild(node syntax.Node, wanted string) syntax.Node {
	if node == nil {
		return nil
	}
	for index := 0; index < node.NamedChildCount(); index++ {
		child := node.NamedChild(index)
		if child != nil && child.Type() == wanted {
			return child
		}
	}
	return nil
}

func pythonSyntaxSameNode(left, right syntax.Node) bool {
	if left == nil || right == nil {
		return left == right
	}
	leftRange := left.Range()
	rightRange := right.Range()
	return left.Type() == right.Type() && leftRange.StartByte == rightRange.StartByte && leftRange.EndByte == rightRange.EndByte
}

func pythonSyntaxImportSource(path string, start, end syntax.Node, symbol, kind string) analysis.SourceReference {
	startRange := start.Range()
	endRange := end.Range()
	startLine := int(startRange.Start.Row) + 1
	startColumn := int(startRange.Start.Column) + 1
	endLine := int(endRange.End.Row) + 1
	endColumn := int(endRange.End.Column) + 1
	return analysis.SourceReference{
		ID:     stableID("import", path, strconv.Itoa(startLine), strconv.Itoa(startColumn), strconv.Itoa(endLine), strconv.Itoa(endColumn), symbol, kind),
		Path:   path,
		Start:  &analysis.Position{Line: startLine, Column: startColumn},
		End:    &analysis.Position{Line: endLine, Column: endColumn},
		Symbol: symbol,
		Kind:   kind,
	}
}

func pythonSyntaxDiagnostic(path, message string, rangeValue syntax.Range) analysis.Diagnostic {
	return analysis.Diagnostic{
		Code:        "python_syntax_error",
		Severity:    "error",
		Message:     "Python source contains a Tree-sitter syntax issue: " + message,
		Path:        path,
		Location:    &analysis.Position{Line: int(rangeValue.Start.Row) + 1, Column: int(rangeValue.Start.Column) + 1},
		Recoverable: true,
	}
}

func pythonSyntaxBackendDiagnostic(path string, err error) analysis.Diagnostic {
	message := "Python Tree-sitter syntax backend failed."
	if err != nil {
		message += " " + err.Error()
	}
	return analysis.Diagnostic{
		Code:        "python_syntax_backend",
		Severity:    "warning",
		Message:     message,
		Path:        path,
		Location:    &analysis.Position{Line: 1, Column: 1},
		Recoverable: true,
	}
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
