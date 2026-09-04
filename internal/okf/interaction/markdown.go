package interaction

import (
	"context"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/hierarchy"
	"github.com/buffo/arch-view/internal/okf/markdown"
	"github.com/buffo/arch-view/internal/okf/ports"
	"github.com/buffo/arch-view/internal/okf/profile"
)

const maxDetailBytes = 512 << 10

func Detail(ctx context.Context, index domain.BundleIndex, effective domain.Profile, conceptID string, registry ports.RuleEvaluator) (domain.ConceptDetail, error) {
	return DetailWithRenderer(ctx, index, effective, conceptID, registry, nil)
}

func DetailWithRenderer(ctx context.Context, index domain.BundleIndex, effective domain.Profile, conceptID string, registry ports.RuleEvaluator, renderer ports.DetailRenderer) (domain.ConceptDetail, error) {
	document, exists := index.Documents[conceptID]
	if !exists {
		return domain.ConceptDetail{}, domain.NewError("okf_concept_not_found", 404, "the requested concept is not in the selected bundle", map[string]any{"concept_id": conceptID})
	}
	containment, diagnostics := hierarchy.Normalize(index, effective.Hierarchy)
	for indexValue := range diagnostics {
		diagnostics[indexValue].BundleID = index.BundleID
	}
	parent, children := relationshipContext(containment, conceptID)
	state, stateDiagnostics, err := profile.EvaluateConceptState(ctx, conceptID, index, effective, children, registry, make(map[string]ports.RuleResult))
	if err != nil {
		return domain.ConceptDetail{}, domain.ContextOperationError(err, "detail")
	}
	diagnostics = append(diagnostics, stateDiagnostics...)
	displayDocument, rendererDiagnostics, err := prepareDetail(ctx, document, effective, renderer)
	if err != nil {
		return domain.ConceptDetail{}, domain.ContextOperationError(err, "detail")
	}
	for indexValue := range rendererDiagnostics {
		rendererDiagnostics[indexValue].BundleID = index.BundleID
	}
	diagnostics = append(diagnostics, rendererDiagnostics...)
	rendered, links, markdownDiagnostics := SanitizeMarkdown(displayDocument, index)
	diagnostics = append(diagnostics, markdownDiagnostics...)
	rawMarkdown := boundedMarkdown(document.Markdown)
	if effective.Details.RawConfigured && !effective.Details.ShowRawMarkdown {
		rawMarkdown = ""
	}
	mappedMetadata := domain.CloneMap(document.UnknownFrontmatter)
	if effective.Details.UnknownConfigured && !effective.Details.ShowUnknown {
		mappedMetadata = nil
	}
	frontmatter := domain.CloneMap(document.Frontmatter)
	declaredState := state.Declared
	if effective.State.ShowDeclaredConfigured && !effective.State.ShowDeclared {
		declaredState = ""
	}
	return domain.ConceptDetail{
		ConceptID: conceptID, BundleID: index.BundleID, SourceRevision: index.SourceRevision,
		Overview:       map[string]string{"title": displayTitle(document), "description": document.Description, "type": document.Type, "source_path": document.SourcePath},
		MappedMetadata: mappedMetadata, DeclaredState: declaredState, EffectiveState: state.Effective,
		RenderedMarkdown: domain.RenderedMarkdown{Format: "sanitized_commonmark", Content: rendered, Links: links}, RawMarkdown: rawMarkdown,
		Frontmatter: frontmatter, Containment: map[string]any{"parent": parent, "children": children}, SemanticLinks: append([]domain.Link(nil), document.Links...),
		Provenance: detailProvenance(document, containment), Diagnostics: diagnostics,
	}, nil
}

func detailProvenance(document domain.ConceptDocument, containment []domain.Relationship) []domain.Provenance {
	result := append([]domain.Provenance(nil), document.Provenance...)
	seen := make(map[domain.Provenance]bool, len(result))
	for _, proof := range result {
		seen[proof] = true
	}
	for _, relationship := range containment {
		if relationship.From != document.ConceptID && relationship.To != document.ConceptID {
			continue
		}
		for _, proof := range relationship.Provenance {
			if !seen[proof] {
				result = append(result, proof)
				seen[proof] = true
			}
		}
	}
	return result
}

func SanitizeMarkdown(document domain.ConceptDocument, index domain.BundleIndex) (string, []domain.Link, []domain.Diagnostic) {
	truncated := len(document.Markdown) > maxDetailBytes
	document.Markdown = boundedMarkdown(document.Markdown)
	rendered, links, diagnostics := markdown.Render(document, index)
	if truncated {
		diagnostics = append(diagnostics, domain.Diagnostic{Code: "okf_detail_truncated", Severity: "warning", Category: "scale", BundleID: index.BundleID, ConceptID: document.ConceptID, Message: "Markdown detail was truncated at the display size limit.", Details: map[string]any{"max_bytes": maxDetailBytes}})
	}
	return rendered, links, diagnostics
}

func boundedMarkdown(value string) string {
	if len(value) <= maxDetailBytes {
		return value
	}
	end := maxDetailBytes
	for end > 0 && !utf8.RuneStart(value[end]) {
		end--
	}
	return value[:end]
}

func relationshipContext(relationships []domain.Relationship, conceptID string) (string, []string) {
	parent := ""
	children := make([]string, 0)
	for _, relationship := range relationships {
		if relationship.To == conceptID {
			parent = relationship.From
		}
		if relationship.From == conceptID {
			children = append(children, relationship.To)
		}
	}
	return parent, children
}

func displayTitle(document domain.ConceptDocument) string {
	if strings.TrimSpace(document.Title) != "" {
		return document.Title
	}
	return strings.TrimSuffix(filepath.Base(document.SourcePath), filepath.Ext(document.SourcePath))
}
