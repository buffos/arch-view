package profile

import (
	"context"
	"fmt"
	"sort"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
)

const DefaultDetailRendererID = "okf.detail.commonmark"

type registeredDetailRenderer struct {
	provider ports.DetailRenderer
	metadata ports.Extension
}

// RegisterDetailRenderer captures metadata once during host composition.
// Profile/source data can select registered code, never supply executable code.
func (registry *Registry) RegisterDetailRenderer(provider ports.DetailRenderer) (err error) {
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("detail renderer registration failed: %v", failure)
		}
	}()
	metadata := ports.Extension{ID: provider.ID(), Version: provider.Version(), Description: provider.Description(), Kind: "detail_renderer", Capabilities: []string{"sanitized-commonmark"}, ParameterSchema: domain.CloneMap(provider.ParameterSchema())}
	if !domain.ValidExtensionIdentity(metadata.ID, metadata.Version) {
		return fmt.Errorf("detail renderer requires a namespaced ID and version")
	}
	if metadata.Description == "" || metadata.ParameterSchema == nil {
		return fmt.Errorf("detail renderer requires description and parameter schema")
	}
	metadata, err = captureExtensionValue(metadata)
	if err != nil {
		return fmt.Errorf("detail renderer metadata: %w", err)
	}
	key := metadata.ID + "@" + metadata.Version
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if _, exists := registry.detailRenderers[key]; exists {
		return fmt.Errorf("detail renderer already registered: %s", key)
	}
	registry.detailRenderers[key] = registeredDetailRenderer{provider, metadata}
	return nil
}

func (registry *Registry) ResolveDetailRenderer(id, version string) (ports.DetailRenderer, bool) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	entry, exists := registry.detailRenderers[id+"@"+version]
	return entry.provider, exists
}

func (registry *Registry) DetailRendererCatalog() []ports.Extension {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	result := make([]ports.Extension, 0, len(registry.detailRenderers))
	for _, entry := range registry.detailRenderers {
		result = append(result, cloneExtensionMetadata(entry.metadata))
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].ID != result[j].ID {
			return result[i].ID < result[j].ID
		}
		return result[i].Version < result[j].Version
	})
	return result
}

func (registry *Registry) cloneDetailRenderersLocked() map[string]registeredDetailRenderer {
	result := make(map[string]registeredDetailRenderer, len(registry.detailRenderers))
	for key, value := range registry.detailRenderers {
		result[key] = value
	}
	return result
}

func (registry *Registry) validateDetailRenderer(value domain.Profile) []domain.Diagnostic {
	selection := value.Details.Renderer
	if selection == nil {
		return nil
	}
	provider, exists := registry.ResolveDetailRenderer(selection.ID, selection.Version)
	var err error
	if !exists {
		err = fmt.Errorf("renderer is not registered")
	} else {
		err = validateDetailParameters(provider, selection.Parameters)
	}
	if err == nil {
		return nil
	}
	return []domain.Diagnostic{{Code: "okf_detail_renderer_invalid", Severity: "error", Category: "presentation", ProfileID: value.ProfileID, Message: "Detail renderer selection is invalid: " + err.Error(), Recovery: "Select a registered renderer and valid parameters."}}
}

func validateDetailParameters(provider ports.DetailRenderer, parameters map[string]any) (err error) {
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("parameter validation failed: %v", failure)
		}
	}()
	return provider.ValidateParameters(domain.CloneMap(parameters))
}

type commonMarkRenderer struct{}

func (commonMarkRenderer) ID() string          { return DefaultDetailRendererID }
func (commonMarkRenderer) Version() string     { return "1" }
func (commonMarkRenderer) Description() string { return "Current formatted CommonMark detail" }
func (commonMarkRenderer) ParameterSchema() map[string]any {
	return map[string]any{"type": "object", "additionalProperties": false}
}
func (commonMarkRenderer) ValidateParameters(parameters map[string]any) error {
	if len(parameters) > 0 {
		return fmt.Errorf("CommonMark accepts no parameters")
	}
	return nil
}
func (commonMarkRenderer) Render(ctx context.Context, document domain.ConceptDocument, _ map[string]any) (string, error) {
	return document.Markdown, ctx.Err()
}
