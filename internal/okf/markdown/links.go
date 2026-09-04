// Package markdown provides CommonMark parsing and one bundle-scoped link
// policy shared by indexing and concept inspection.
package markdown

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

func Extract(source string) []domain.Link {
	data := []byte(source)
	tree := goldmark.New().Parser().Parse(text.NewReader(data))
	links := []domain.Link{}
	_ = ast.Walk(tree, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if node.Kind() == ast.KindImage {
			return ast.WalkSkipChildren, nil
		}
		if target, label, ok := destination(node, data); ok {
			links = append(links, domain.Link{ID: fmt.Sprintf("link:%d", len(links)+1), RawTarget: target, Text: label, Kind: "markdown"})
		}
		return ast.WalkContinue, nil
	})
	return links
}

func destination(node ast.Node, source []byte) (string, string, bool) {
	switch value := node.(type) {
	case *ast.Link:
		return string(util.URLEscape(value.Destination, true)), string(value.Text(source)), true
	case *ast.AutoLink:
		target := string(value.URL(source))
		if value.AutoLinkType == ast.AutoLinkEmail && !strings.HasPrefix(strings.ToLower(target), "mailto:") {
			target = "mailto:" + target
		}
		return string(util.URLEscape([]byte(target), false)), string(value.Label(source)), true
	}
	return "", "", false
}

func Resolve(document domain.ConceptDocument, index domain.BundleIndex, link domain.Link) (domain.Link, *domain.Diagnostic) {
	link.Safe, link.Resolved, link.External = false, false, false
	fail := func(code, message string) (domain.Link, *domain.Diagnostic) {
		return link, &domain.Diagnostic{Code: code, Severity: "warning", Category: "relationship", BundleID: index.BundleID, ConceptID: document.ConceptID, Message: message, Details: map[string]any{"target": link.RawTarget}}
	}
	parsed, err := url.Parse(link.RawTarget)
	if err != nil {
		return fail("okf_unsafe_link", "The Markdown link could not be parsed safely.")
	}
	link.Fragment = parsed.Fragment
	scheme := strings.ToLower(parsed.Scheme)
	if scheme == "http" || scheme == "https" || scheme == "mailto" {
		link.External, link.Kind = true, "external"
		link.Safe = (scheme == "mailto" && parsed.Opaque != "") || (scheme != "mailto" && parsed.Host != "")
		if !link.Safe {
			return fail("okf_unsafe_link", "The external link target is invalid.")
		}
		return link, nil
	}
	if scheme != "" || parsed.Host != "" {
		return fail("okf_unsafe_link", "The link scheme is not permitted.")
	}
	if strings.ContainsAny(parsed.Path, "\\\x00") {
		return fail("okf_bundle_boundary_violation", "The local link uses an unsafe path.")
	}
	if parsed.Path == "" && parsed.Fragment != "" {
		link.TargetID, link.TargetPath, link.Kind = document.ConceptID, document.SourcePath, "local_fragment"
		link.Safe, link.Resolved = true, true
		return link, nil
	}
	target := parsed.Path
	if target == "" {
		return fail("okf_relationship_unresolved", "The local link has no concept target.")
	}
	if strings.HasPrefix(target, "/") {
		target = strings.TrimPrefix(target, "/")
	} else {
		target = path.Join(path.Dir(document.SourcePath), target)
	}
	target = path.Clean(target)
	if target == "." || target == ".." || strings.HasPrefix(target, "../") || strings.HasPrefix(target, "/") {
		return fail("okf_bundle_boundary_violation", "A local link escapes the selected bundle boundary.")
	}
	if !strings.HasSuffix(strings.ToLower(target), ".md") {
		target += ".md"
	}
	link.TargetPath, link.TargetID, link.Kind = target, target[:len(target)-3], "local"
	_, link.Resolved = index.Documents[link.TargetID]
	link.Safe = link.Resolved
	if !link.Resolved {
		return fail("okf_relationship_unresolved", "The local Markdown link target is not present in this bundle.")
	}
	return link, nil
}
