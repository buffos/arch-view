package interaction

import (
	"context"
	"fmt"
	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
)

func prepareDetail(ctx context.Context, document domain.ConceptDocument, effective domain.Profile, renderer ports.DetailRenderer) (domain.ConceptDocument, []domain.Diagnostic, error) {
	if err := ctx.Err(); err != nil {
		return document, nil, err
	}
	if renderer == nil {
		return document, nil, nil
	}
	var parameters map[string]any
	if effective.Details.Renderer != nil {
		parameters = effective.Details.Renderer.Parameters
	}
	output, err := invokeDetailRenderer(ctx, renderer, document, parameters)
	if ctx.Err() != nil {
		return document, nil, ctx.Err()
	}
	if err != nil {
		return document, []domain.Diagnostic{{Code: "okf_detail_renderer_failed", Severity: "warning", Category: "presentation", ProfileID: effective.ProfileID, ConceptID: document.ConceptID, Message: "Custom detail rendering failed; the original Markdown is displayed.", Details: map[string]any{"reason": err.Error()}, Recovery: "Select the default CommonMark renderer or repair the registered renderer."}}, nil
	}
	document.Markdown = output
	return document, nil, nil
}

func invokeDetailRenderer(ctx context.Context, renderer ports.DetailRenderer, document domain.ConceptDocument, parameters map[string]any) (output string, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("renderer failed: %v", failure)
		}
	}()
	return renderer.Render(ctx, domain.CloneDocument(document), domain.CloneMap(parameters))
}
