package markdown

import (
	"bytes"
	"fmt"
	"html"
	"net/url"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type safeRenderer struct {
	document    domain.ConceptDocument
	index       domain.BundleIndex
	links       []domain.Link
	diagnostics []domain.Diagnostic
	anchors     map[ast.Node]bool
}

func Render(document domain.ConceptDocument, index domain.BundleIndex) (string, []domain.Link, []domain.Diagnostic) {
	safe := &safeRenderer{document: document, index: index, anchors: make(map[ast.Node]bool), links: []domain.Link{}}
	engine := goldmark.New(goldmark.WithRendererOptions(renderer.WithNodeRenderers(util.Prioritized(safe, 100))))
	var output bytes.Buffer
	if err := engine.Convert([]byte(document.Markdown), &output); err != nil {
		safe.diagnostics = append(safe.diagnostics, domain.Diagnostic{Code: "okf_markdown_failed", Severity: "error", Message: err.Error()})
	}
	return output.String(), safe.links, safe.diagnostics
}

func (safe *safeRenderer) RegisterFuncs(registry renderer.NodeRendererFuncRegisterer) {
	registry.Register(ast.KindLink, safe.link)
	registry.Register(ast.KindAutoLink, safe.link)
	registry.Register(ast.KindImage, safe.image)
}

func (safe *safeRenderer) link(writer util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		if safe.anchors[node] {
			_, _ = writer.WriteString("</a>")
		}
		return ast.WalkContinue, nil
	}
	target, label, _ := destination(node, source)
	value, diagnostic := Resolve(safe.document, safe.index, domain.Link{ID: fmt.Sprintf("%s#detail-link-%d", safe.document.ConceptID, len(safe.links)+1), RawTarget: target, Text: label})
	safe.links = append(safe.links, value)
	if diagnostic != nil && len(safe.diagnostics) < 200 {
		safe.diagnostics = append(safe.diagnostics, *diagnostic)
	}
	if value.Safe {
		href, attributes := "#okf-concept="+url.PathEscape(value.TargetID), ""
		if value.External {
			href, attributes = target, ` class="okf-external-link" target="_blank" rel="noopener noreferrer"`
		}
		_, _ = writer.WriteString(`<a href="` + html.EscapeString(href) + `"` + attributes + `>`)
		safe.anchors[node] = true
	}
	if node.Kind() == ast.KindAutoLink {
		_, _ = writer.WriteString(html.EscapeString(label))
	}
	return ast.WalkContinue, nil
}

// Images stay inert; inspecting Markdown must not fetch remote media or files.
func (safe *safeRenderer) image(writer util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = writer.WriteString(html.EscapeString(string(node.Text(source))))
	}
	return ast.WalkSkipChildren, nil
}
