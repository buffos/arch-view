package clojureanalyzer

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/syntax"
	clojuresyntax "github.com/buffo/arch-view/internal/analysis/syntax/clojure"
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
	Line    int
	Column  int
	Message string
}

// parseClojureForms uses the built-in Tree-sitter provider for configuration
// forms. Analyzer discovery injects the same provider seam so production never
// falls back to the removed reader implementation.
func parseClojureForms(content string) ([]*cljForm, []syntaxIssue) {
	forms, issues, _ := parseClojureFormsWithProvider(context.Background(), clojuresyntax.NewProvider(), "", content)
	return forms, issues
}

func parseClojureFormsWithProvider(ctx context.Context, provider syntax.Provider, path, content string) ([]*cljForm, []syntaxIssue, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if provider == nil {
		return nil, []syntaxIssue{{Line: 1, Column: 1, Message: "Clojure Tree-sitter syntax provider is not configured."}}, errors.New("clojure tree-sitter syntax provider is not configured")
	}
	parsed, err := provider.Parse(ctx, syntax.Source{Path: path, Content: []byte(content)})
	if err != nil {
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}
		return nil, []syntaxIssue{{Line: 1, Column: 1, Message: err.Error()}}, err
	}
	defer parsed.Close()
	issues := make([]syntaxIssue, 0, len(parsed.Issues))
	for _, issue := range parsed.Issues {
		issues = append(issues, clojureSyntaxIssue(issue))
	}
	if parsed.Tree == nil || parsed.Tree.Root() == nil {
		err := errors.New("tree-sitter returned no Clojure syntax tree")
		issues = append(issues, syntaxIssue{Line: 1, Column: 1, Message: err.Error()})
		return nil, issues, err
	}
	root := parsed.Tree.Root()
	forms := make([]*cljForm, 0, root.NamedChildCount())
	for index := 0; index < root.NamedChildCount(); index++ {
		if form := clojureFormFromSyntax(root.NamedChild(index)); form != nil {
			forms = append(forms, form)
		}
	}
	return forms, issues, nil
}

func clojureSyntaxIssue(issue syntax.Issue) syntaxIssue {
	return syntaxIssue{
		Line:    int(issue.Range.Start.Row) + 1,
		Column:  int(issue.Range.Start.Column) + 1,
		Message: issue.Message,
	}
}

func clojureSyntaxBackendDiagnostic(path string, err error) analysis.Diagnostic {
	message := "Clojure Tree-sitter syntax backend failed."
	if err != nil {
		message += " " + err.Error()
	}
	return analysis.Diagnostic{
		Code:        "clojure_syntax_backend",
		Severity:    "warning",
		Message:     message,
		Path:        path,
		Location:    &analysis.Position{Line: 1, Column: 1},
		Recoverable: true,
	}
}

func clojureFormFromSyntax(node syntax.Node) *cljForm {
	if node == nil {
		return nil
	}
	switch node.Type() {
	case "comment", "dis_expr":
		return nil
	case "num_lit":
		return clojureAtomForm(node, strings.TrimSpace(node.Text()), atomNumber)
	case "kwd_lit":
		return clojureAtomForm(node, strings.TrimSpace(node.Text()), atomKeyword)
	case "str_lit":
		return clojureAtomForm(node, decodeClojureString(node.Text()), atomString)
	case "char_lit":
		return clojureAtomForm(node, strings.TrimSpace(node.Text()), atomCharacter)
	case "nil_lit", "bool_lit":
		return clojureAtomForm(node, strings.TrimSpace(node.Text()), atomOther)
	case "sym_lit":
		return clojureAtomForm(node, clojureSyntaxSymbolText(node), atomSymbol)
	case "list_lit":
		return clojureCollectionForm(node, formList)
	case "vec_lit":
		return clojureCollectionForm(node, formVector)
	case "map_lit":
		return clojureCollectionForm(node, formMap)
	case "set_lit":
		return clojureCollectionForm(node, formSet)
	case "read_cond_lit", "splicing_read_cond_lit":
		result := &cljForm{Kind: formReaderConditional, Start: clojureSyntaxStart(node), End: clojureSyntaxEnd(node), ReaderSplice: node.Type() == "splicing_read_cond_lit"}
		for index := 0; index < node.NamedChildCount(); index++ {
			if child := clojureFormFromSyntax(node.NamedChild(index)); child != nil {
				result.Items = append(result.Items, child)
			}
		}
		return result
	case "quoting_lit", "syn_quoting_lit", "var_quoting_lit", "unquoting_lit", "unquote_splicing_lit", "derefing_lit", "evaling_lit", "anon_fn_lit", "regex_lit", "ns_map_lit":
		return clojureSyntaxPrefixedForm(node)
	case "tagged_or_ctor_lit":
		return clojureSyntaxTaggedForm(node)
	case "meta_lit", "old_meta_lit":
		return clojureSyntaxPrefixedForm(node)
	default:
		return clojureSyntaxFallbackForm(node)
	}
}

func clojureAtomForm(node syntax.Node, value string, kind atomKind) *cljForm {
	return &cljForm{Kind: formAtom, Atom: value, AtomKind: kind, Start: clojureSyntaxStart(node), End: clojureSyntaxEnd(node)}
}

func clojureCollectionForm(node syntax.Node, kind formKind) *cljForm {
	result := &cljForm{Kind: kind, Items: make([]*cljForm, 0), Start: clojureSyntaxStart(node), End: clojureSyntaxEnd(node)}
	for index := 0; index < node.NamedChildCount(); index++ {
		if child := clojureFormFromSyntax(node.NamedChild(index)); child != nil {
			result.Items = append(result.Items, child)
		}
	}
	return result
}

func clojureSyntaxPrefixedForm(node syntax.Node) *cljForm {
	child := clojureSyntaxFirstFormChild(node)
	if child == nil {
		return nil
	}
	if node.Type() == "quoting_lit" || node.Type() == "syn_quoting_lit" || node.Type() == "var_quoting_lit" {
		child.Quoted = true
	}
	if start := clojureSyntaxStart(node); start != nil && (child.Start == nil || positionBefore(start, child.Start)) {
		child.Start = start
	}
	return child
}

func clojureSyntaxTaggedForm(node syntax.Node) *cljForm {
	result := &cljForm{Kind: formList, Items: make([]*cljForm, 0), Start: clojureSyntaxStart(node), End: clojureSyntaxEnd(node)}
	for index := 0; index < node.NamedChildCount(); index++ {
		if child := clojureFormFromSyntax(node.NamedChild(index)); child != nil {
			result.Items = append(result.Items, child)
		}
	}
	if len(result.Items) == 1 {
		return result.Items[0]
	}
	return result
}

func clojureSyntaxFallbackForm(node syntax.Node) *cljForm {
	if node.NamedChildCount() == 1 {
		return clojureFormFromSyntax(node.NamedChild(0))
	}
	if node.NamedChildCount() > 1 {
		result := &cljForm{Kind: formList, Items: make([]*cljForm, 0), Start: clojureSyntaxStart(node), End: clojureSyntaxEnd(node)}
		for index := 0; index < node.NamedChildCount(); index++ {
			if child := clojureFormFromSyntax(node.NamedChild(index)); child != nil {
				result.Items = append(result.Items, child)
			}
		}
		return result
	}
	return clojureAtomForm(node, strings.TrimSpace(node.Text()), atomOther)
}

func clojureSyntaxFirstFormChild(node syntax.Node) *cljForm {
	if node == nil {
		return nil
	}
	for index := 0; index < node.NamedChildCount(); index++ {
		if child := clojureFormFromSyntax(node.NamedChild(index)); child != nil {
			return child
		}
	}
	return nil
}

func clojureSyntaxSymbolText(node syntax.Node) string {
	if node == nil {
		return ""
	}
	parts := make([]string, 0, 2)
	for index := 0; index < node.NamedChildCount(); index++ {
		child := node.NamedChild(index)
		if child == nil {
			continue
		}
		switch child.Type() {
		case "sym_ns", "sym_name":
			parts = append(parts, strings.TrimSpace(child.Text()))
		}
	}
	if len(parts) > 0 {
		if len(parts) == 2 {
			return parts[0] + "/" + parts[1]
		}
		return parts[len(parts)-1]
	}
	return strings.TrimSpace(node.Text())
}

func decodeClojureString(value string) string {
	value = strings.TrimSpace(value)
	if decoded, err := strconv.Unquote(value); err == nil {
		return decoded
	}
	return strings.TrimSuffix(strings.TrimPrefix(value, "\""), "\"")
}

func clojureSyntaxStart(node syntax.Node) *analysis.Position {
	if node == nil {
		return nil
	}
	rangeValue := node.Range()
	return &analysis.Position{Line: int(rangeValue.Start.Row) + 1, Column: int(rangeValue.Start.Column) + 1}
}

func clojureSyntaxEnd(node syntax.Node) *analysis.Position {
	if node == nil {
		return nil
	}
	rangeValue := node.Range()
	return &analysis.Position{Line: int(rangeValue.End.Row) + 1, Column: int(rangeValue.End.Column) + 1}
}

func positionBefore(left, right *analysis.Position) bool {
	if left == nil || right == nil {
		return false
	}
	return left.Line < right.Line || left.Line == right.Line && left.Column < right.Column
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
