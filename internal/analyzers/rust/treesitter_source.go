package rustanalyzer

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/syntax"
)

type rustSyntaxAttribute struct {
	Name      string
	Value     string
	Cfg       string
	Path      string
	TestOnly  bool
	Uncertain bool
	Source    syntax.Node
}

type rustSyntaxUseLeaf struct {
	Path  string
	Alias string
}

func extractRustSyntaxItems(root syntax.Node, project Project, modules map[string]*rustModule, sourceReferences map[string]analysis.SourceReference, module *rustModule, fileSource analysis.SourceReference, includeTests bool, targetKind string) ([]rustModuleDeclaration, []rustUseObservation, []analysis.Diagnostic) {
	walker := rustSyntaxItemWalker{
		project:          project,
		modules:          modules,
		sourceReferences: sourceReferences,
		fileSource:       fileSource,
		includeTests:     includeTests,
		targetKind:       targetKind,
	}
	walker.walkContainer(root, module)
	return walker.declarations, walker.uses, walker.diagnostics
}

type rustSyntaxItemWalker struct {
	project          Project
	modules          map[string]*rustModule
	sourceReferences map[string]analysis.SourceReference
	fileSource       analysis.SourceReference
	includeTests     bool
	targetKind       string
	declarations     []rustModuleDeclaration
	uses             []rustUseObservation
	diagnostics      []analysis.Diagnostic
}

func (walker *rustSyntaxItemWalker) walkContainer(container syntax.Node, current *rustModule) {
	if container == nil || current == nil {
		return
	}
	pending := make([]rustSyntaxAttribute, 0)
	for index := 0; index < container.NamedChildCount(); index++ {
		child := container.NamedChild(index)
		if child == nil {
			continue
		}
		switch child.Type() {
		case "attribute_item":
			if attribute, ok := rustSyntaxAttributeFromNode(child); ok {
				pending = append(pending, attribute)
			}
			continue
		case "inner_attribute_item":
			// Inner attributes belong to the enclosing crate/module. They are
			// intentionally consumed here so they cannot become the next
			// item's attributes.
			continue
		}
		walker.walkItem(child, current, pending)
		pending = pending[:0]
	}
}

func (walker *rustSyntaxItemWalker) walkItem(item syntax.Node, current *rustModule, attributes []rustSyntaxAttribute) {
	if item == nil || current == nil {
		return
	}
	for _, attribute := range attributes {
		walker.recordAttribute(current, attribute)
	}
	cfgConditions := rustSyntaxCfgConditions(attributes)
	if rustSyntaxAttributesTestOnly(attributes) && !walker.includeTests {
		return
	}
	if len(cfgConditions) > 0 && !rustCfgKnown(cfgConditions) {
		walker.diagnostics = append(walker.diagnostics, rustSyntaxCfgDiagnostic(walker.fileSource.Path, item, cfgConditions))
	}

	switch item.Type() {
	case "mod_item":
		walker.walkModuleItem(item, current, attributes, cfgConditions)
	case "use_declaration":
		walker.walkUseItem(item, current, cfgConditions)
	case "extern_crate_declaration":
		walker.walkExternCrateItem(item, current, cfgConditions)
	default:
		walker.recordMacroUncertainty(item, current)
	}
}

func (walker *rustSyntaxItemWalker) recordAttribute(current *rustModule, attribute rustSyntaxAttribute) {
	if attribute.Name == "" {
		return
	}
	if attribute.Name != "cfg" && attribute.Name != "cfg_attr" && attribute.Name != "path" && attribute.Name != "derive" && attribute.Name != "allow" && attribute.Name != "warn" && attribute.Name != "deny" && attribute.Name != "doc" {
		current.MacroAttributes[attribute.Name] = struct{}{}
		walker.diagnostics = append(walker.diagnostics, rustSyntaxAttributeDiagnostic(walker.fileSource.Path, attribute.Source, attribute.Name))
	}
	if attribute.Name == "derive" {
		current.MacroAttributes[attribute.Name] = struct{}{}
		walker.diagnostics = append(walker.diagnostics, rustSyntaxAttributeDiagnostic(walker.fileSource.Path, attribute.Source, attribute.Name))
	}
}

func (walker *rustSyntaxItemWalker) walkModuleItem(item syntax.Node, current *rustModule, attributes []rustSyntaxAttribute, cfgConditions []string) {
	nameNode := item.ChildByFieldName("name")
	if nameNode == nil || strings.TrimSpace(nameNode.Text()) == "" {
		walker.diagnostics = append(walker.diagnostics, rustSyntaxParseDiagnostic(walker.fileSource.Path, item, "rust_module_syntax", "Rust module declaration has no readable module name."))
		return
	}
	name := normalizeRustIdent(strings.TrimSpace(nameNode.Text()))
	sourceStart := rustSyntaxDirectChild(item, "mod")
	if sourceStart == nil {
		sourceStart = item
	}
	declarationSource := rustSyntaxSource(walker.fileSource.Path, "module_declaration", sourceStart, item, current.DisplayName(walker.project.PackageName)+"::"+name)
	body := item.ChildByFieldName("body")
	if body != nil {
		child := ensureRustModule(walker.modules, walker.project, current, "inline_module", fileSourcePathAbsolute(walker.project, walker.fileSource.Path), append(append([]string(nil), current.Path...), name), walker.targetKind)
		child.SourceReferenceIDs[declarationSource.ID] = struct{}{}
		child.SourceReferenceIDs[walker.fileSource.ID] = struct{}{}
		child.Paths[walker.fileSource.Path] = struct{}{}
		child.Tags["inline"] = struct{}{}
		current.SourceReferenceIDs[declarationSource.ID] = struct{}{}
		walker.sourceReferences[declarationSource.ID] = declarationSource
		for _, condition := range cfgConditions {
			child.CfgConditions[condition] = struct{}{}
		}
		walker.walkContainer(body, child)
		return
	}

	pathOverride := ""
	for _, attribute := range attributes {
		if attribute.Name == "path" {
			pathOverride = attribute.Path
			break
		}
	}
	walker.declarations = append(walker.declarations, rustModuleDeclaration{
		Parent:        current,
		Name:          name,
		PathOverride:  pathOverride,
		CfgConditions: cfgConditions,
		Source:        declarationSource,
	})
}

func (walker *rustSyntaxItemWalker) walkUseItem(item syntax.Node, current *rustModule, cfgConditions []string) {
	argument := item.ChildByFieldName("argument")
	if argument == nil {
		return
	}
	leaves := extractRustSyntaxUseLeaves(argument)
	for index, leaf := range leaves {
		if leaf.Path == "" {
			continue
		}
		kind := mapUseKind(rustSyntaxIsPublic(item))
		source := rustSyntaxSource(walker.fileSource.Path, kind, item, item, leaf.Path+useAliasSuffix(leaf.Alias)+fmt.Sprintf("#%d", index))
		walker.uses = append(walker.uses, rustUseObservation{
			FromModuleID:  current.ID,
			Kind:          kind,
			Path:          leaf.Path,
			Alias:         leaf.Alias,
			ImportedName:  useImportedName(leaf.Path),
			CfgConditions: append([]string(nil), cfgConditions...),
			Source:        source,
		})
	}
}

func (walker *rustSyntaxItemWalker) walkExternCrateItem(item syntax.Node, current *rustModule, cfgConditions []string) {
	nameNode := item.ChildByFieldName("name")
	if nameNode == nil || strings.TrimSpace(nameNode.Text()) == "" {
		return
	}
	name := normalizeRustIdent(strings.TrimSpace(nameNode.Text()))
	source := rustSyntaxSource(walker.fileSource.Path, "extern_crate", item, item, name)
	walker.uses = append(walker.uses, rustUseObservation{
		FromModuleID:  current.ID,
		Kind:          "dependency",
		Path:          name,
		ImportedName:  name,
		CfgConditions: append([]string(nil), cfgConditions...),
		Source:        source,
	})
}

func (walker *rustSyntaxItemWalker) recordMacroUncertainty(item syntax.Node, current *rustModule) {
	name := ""
	if item.Type() == "macro_definition" {
		name = "macro_rules"
	} else {
		if macro := rustSyntaxFirstDescendant(item, "macro_invocation"); macro != nil {
			nameNode := macro.NamedChild(0)
			if nameNode != nil {
				name = strings.TrimSpace(nameNode.Text())
			}
		}
	}
	if name == "" {
		return
	}
	current.MacroAttributes[name] = struct{}{}
	walker.diagnostics = append(walker.diagnostics, rustSyntaxMacroDiagnostic(walker.fileSource.Path, item, name))
}

func rustSyntaxAttributeFromNode(item syntax.Node) (rustSyntaxAttribute, bool) {
	if item == nil || item.Type() != "attribute_item" {
		return rustSyntaxAttribute{}, false
	}
	attribute := rustSyntaxFirstNamedChild(item, "attribute")
	if attribute == nil {
		return rustSyntaxAttribute{}, false
	}
	pathNode := attribute.NamedChild(0)
	if pathNode == nil {
		return rustSyntaxAttribute{}, false
	}
	name := strings.TrimSpace(pathNode.Text())
	if index := strings.Index(name, "::"); index >= 0 {
		name = name[:index]
	}
	value := ""
	if valueNode := attribute.ChildByFieldName("value"); valueNode != nil {
		value = strings.TrimSpace(valueNode.Text())
	}
	if arguments := attribute.ChildByFieldName("arguments"); arguments != nil {
		value = rustSyntaxDelimitedText(arguments.Text())
	}
	result := rustSyntaxAttribute{Name: name, Value: value, Source: item}
	if name == "cfg" || name == "cfg_attr" {
		result.Cfg = value
		result.TestOnly = rustCfgTestOnly(result.Cfg)
		result.Uncertain = result.Cfg != "" && !rustCfgKnown([]string{result.Cfg})
	}
	if name == "path" {
		if literal := rustSyntaxFirstStringLiteral(attribute); literal != nil {
			result.Path = decodeRustStringLiteral(literal.Text())
		}
	}
	return result, true
}

func rustSyntaxCfgConditions(attributes []rustSyntaxAttribute) []string {
	conditions := make([]string, 0)
	for _, attribute := range attributes {
		if (attribute.Name == "cfg" || attribute.Name == "cfg_attr") && attribute.Cfg != "" {
			conditions = appendUniqueString(conditions, attribute.Cfg)
		}
	}
	sort.Strings(conditions)
	return conditions
}

func rustSyntaxAttributesTestOnly(attributes []rustSyntaxAttribute) bool {
	for _, attribute := range attributes {
		if attribute.TestOnly {
			return true
		}
	}
	return false
}

func extractRustSyntaxUseLeaves(node syntax.Node) []rustSyntaxUseLeaf {
	return extractRustSyntaxUseLeavesWithPrefix(node, "")
}

func extractRustSyntaxUseLeavesWithPrefix(node syntax.Node, prefix string) []rustSyntaxUseLeaf {
	if node == nil {
		return nil
	}
	switch node.Type() {
	case "use_list":
		result := make([]rustSyntaxUseLeaf, 0)
		for index := 0; index < node.NamedChildCount(); index++ {
			result = append(result, extractRustSyntaxUseLeavesWithPrefix(node.NamedChild(index), prefix)...)
		}
		return result
	case "scoped_use_list":
		path := ""
		if pathNode := node.ChildByFieldName("path"); pathNode != nil {
			path = rustSyntaxUsePathText(pathNode)
		}
		combined := rustSyntaxJoinUsePath(prefix, path)
		return extractRustSyntaxUseLeavesWithPrefix(node.ChildByFieldName("list"), combined)
	case "use_as_clause":
		path := ""
		if pathNode := node.ChildByFieldName("path"); pathNode != nil {
			path = rustSyntaxUsePathText(pathNode)
		}
		path = rustSyntaxJoinUsePath(prefix, path)
		alias := ""
		if aliasNode := node.ChildByFieldName("alias"); aliasNode != nil {
			alias = normalizeRustIdent(strings.TrimSpace(aliasNode.Text()))
		}
		if path == "" {
			return nil
		}
		return []rustSyntaxUseLeaf{{Path: path, Alias: alias}}
	case "use_wildcard":
		path := ""
		if node.NamedChildCount() > 0 {
			path = rustSyntaxUsePathText(node.NamedChild(0))
		}
		path = rustSyntaxJoinUsePath(prefix, path)
		if path == "" {
			path = "*"
		} else {
			path += "::*"
		}
		return []rustSyntaxUseLeaf{{Path: path}}
	default:
		path := rustSyntaxJoinUsePath(prefix, rustSyntaxUsePathText(node))
		if path == "" {
			return nil
		}
		return []rustSyntaxUseLeaf{{Path: path}}
	}
}

func rustSyntaxUsePathText(node syntax.Node) string {
	if node == nil {
		return ""
	}
	value := strings.TrimSpace(node.Text())
	parts := strings.Split(value, "::")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		result = append(result, normalizeRustIdent(part))
	}
	return strings.Join(result, "::")
}

func rustSyntaxJoinUsePath(prefix, path string) string {
	prefix = rustSyntaxUsePathTextValue(prefix)
	path = rustSyntaxUsePathTextValue(path)
	if path == "" {
		return prefix
	}
	parts := strings.Split(path, "::")
	if prefix != "" && len(parts) > 0 && parts[0] == "self" {
		parts = parts[1:]
	}
	if len(parts) == 0 {
		return prefix
	}
	if prefix == "" {
		return strings.Join(parts, "::")
	}
	return prefix + "::" + strings.Join(parts, "::")
}

func rustSyntaxUsePathTextValue(value string) string {
	parts := strings.Split(strings.TrimSpace(value), "::")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, normalizeRustIdent(part))
		}
	}
	return strings.Join(result, "::")
}

func rustSyntaxSource(relative, kind string, start, end syntax.Node, symbol string) analysis.SourceReference {
	if start == nil {
		start = end
	}
	if end == nil {
		end = start
	}
	startRange := start.Range()
	endRange := end.Range()
	startLine := int(startRange.Start.Row) + 1
	startColumn := int(startRange.Start.Column) + 1
	endLine := int(endRange.End.Row) + 1
	endColumn := int(endRange.End.Column) + 1
	return analysis.SourceReference{
		ID:     stableID("source", kind, relative, symbol, fmt.Sprintf("%d:%d", startLine, startColumn), fmt.Sprintf("%d:%d", endLine, endColumn)),
		Path:   relative,
		Start:  &analysis.Position{Line: startLine, Column: startColumn},
		End:    &analysis.Position{Line: endLine, Column: endColumn},
		Symbol: symbol,
		Kind:   kind,
	}
}

func rustSyntaxDiagnostic(path, message string, rangeValue syntax.Range) analysis.Diagnostic {
	return analysis.Diagnostic{
		Code:        "rust_syntax_error",
		Severity:    "error",
		Message:     "Rust source contains a Tree-sitter syntax issue: " + message,
		Path:        path,
		Location:    &analysis.Position{Line: int(rangeValue.Start.Row) + 1, Column: int(rangeValue.Start.Column) + 1},
		Recoverable: true,
	}
}

func rustSyntaxBackendDiagnostic(path string, err error) analysis.Diagnostic {
	message := "Rust Tree-sitter syntax backend failed."
	if err != nil {
		message += " " + err.Error()
	}
	return analysis.Diagnostic{
		Code:        "rust_syntax_backend",
		Severity:    "warning",
		Message:     message,
		Path:        path,
		Location:    &analysis.Position{Line: 1, Column: 1},
		Recoverable: true,
	}
}

func rustSyntaxParseDiagnostic(path string, node syntax.Node, code, message string) analysis.Diagnostic {
	return analysis.Diagnostic{
		Code:        code,
		Severity:    "warning",
		Message:     message,
		Path:        path,
		Location:    rustSyntaxNodePosition(node),
		Recoverable: true,
	}
}

func rustSyntaxCfgDiagnostic(path string, node syntax.Node, conditions []string) analysis.Diagnostic {
	return analysis.Diagnostic{
		Code:        "rust_cfg_uncertain",
		Severity:    "warning",
		Message:     "Rust cfg conditions were retained as metadata but could not be evaluated statically.",
		Path:        path,
		Location:    rustSyntaxNodePosition(node),
		Recoverable: true,
		Metadata:    map[string]any{"conditions": append([]string(nil), conditions...)},
	}
}

func rustSyntaxAttributeDiagnostic(path string, node syntax.Node, name string) analysis.Diagnostic {
	return analysis.Diagnostic{
		Code:        "rust_macro_attribute",
		Severity:    "warning",
		Message:     fmt.Sprintf("Rust attribute %q was not expanded; generated declarations remain outside the static result.", name),
		Path:        path,
		Location:    rustSyntaxNodePosition(node),
		Recoverable: true,
		Metadata:    map[string]any{"attribute": name},
	}
}

func rustSyntaxMacroDiagnostic(path string, node syntax.Node, name string) analysis.Diagnostic {
	return analysis.Diagnostic{
		Code:        "rust_macro_uncertain",
		Severity:    "warning",
		Message:     fmt.Sprintf("Rust macro %q was not expanded; generated declarations were not inferred.", name),
		Path:        path,
		Location:    rustSyntaxNodePosition(node),
		Recoverable: true,
		Metadata:    map[string]any{"macro": name},
	}
}

func rustSyntaxNodePosition(node syntax.Node) *analysis.Position {
	if node == nil {
		return &analysis.Position{Line: 1, Column: 1}
	}
	rangeValue := node.Range()
	return &analysis.Position{Line: int(rangeValue.Start.Row) + 1, Column: int(rangeValue.Start.Column) + 1}
}

func rustSyntaxDirectChild(node syntax.Node, wanted string) syntax.Node {
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

func rustSyntaxFirstNamedChild(node syntax.Node, wanted string) syntax.Node {
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

func rustSyntaxFirstDescendant(node syntax.Node, wanted string) syntax.Node {
	if node == nil {
		return nil
	}
	if node.Type() == wanted {
		return node
	}
	for index := 0; index < node.NamedChildCount(); index++ {
		if found := rustSyntaxFirstDescendant(node.NamedChild(index), wanted); found != nil {
			return found
		}
	}
	return nil
}

func rustSyntaxIsPublic(node syntax.Node) bool {
	visibility := rustSyntaxDirectChild(node, "visibility_modifier")
	return visibility != nil && strings.HasPrefix(strings.TrimSpace(visibility.Text()), "pub")
}

func rustSyntaxDelimitedText(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		first, last := value[0], value[len(value)-1]
		if (first == '(' && last == ')') || (first == '[' && last == ']') || (first == '{' && last == '}') {
			return strings.TrimSpace(value[1 : len(value)-1])
		}
	}
	return value
}

func rustSyntaxFirstStringLiteral(node syntax.Node) syntax.Node {
	if node == nil {
		return nil
	}
	if node.Type() == "string_literal" || node.Type() == "raw_string_literal" {
		return node
	}
	for index := 0; index < node.NamedChildCount(); index++ {
		if found := rustSyntaxFirstStringLiteral(node.NamedChild(index)); found != nil {
			return found
		}
	}
	return nil
}

func decodeRustStringLiteral(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "b")
	if strings.HasPrefix(value, "r") {
		quote := strings.IndexByte(value, '"')
		if quote >= 0 {
			hashes := strings.Count(value[1:quote], "#")
			closing := `"` + strings.Repeat("#", hashes)
			if end := strings.LastIndex(value, closing); end > quote {
				return value[quote+1 : end]
			}
		}
	}
	if decoded, err := strconv.Unquote(value); err == nil {
		return decoded
	}
	return strings.Trim(value, "\"")
}
