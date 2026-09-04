package application

import (
	"context"
	"fmt"
	"sort"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
	"github.com/buffo/arch-view/internal/okf/presentation"
)

func (service *diagnosticService) Extensions(ctx context.Context) ([]any, error) {
	if err := service.ensureReady(ctx); err != nil {
		return nil, err
	}
	values, err := ruleExtensions(service.registry)
	if err != nil {
		return nil, err
	}
	for _, adapter := range service.relationshipAdapters {
		if adapter == nil {
			continue
		}
		value, err := relationshipExtension(adapter)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := ctx.Err(); err != nil {
		return nil, domain.ContextOperationError(err, "extension catalog")
	}
	for _, definition := range service.registry.ShapeCatalog() {
		values = append(values, ports.Extension{ID: definition.ID, Version: definition.Version,
			Kind: "shape", Description: definition.Description, Capabilities: []string{"renderer-neutral", "unit-box-geometry"},
			ShapeDefinition: &definition, DefinitionSchema: presentation.ShapeDefinitionSchema()})
	}
	values = append(values, service.registry.DetailRendererCatalog()...)
	values = append(values, service.registry.DiagnosticProviderCatalog()...)
	sort.SliceStable(values, func(i, j int) bool {
		if values[i].Kind != values[j].Kind {
			return values[i].Kind < values[j].Kind
		}
		if values[i].ID != values[j].ID {
			return values[i].ID < values[j].ID
		}
		return values[i].Version < values[j].Version
	})
	result := make([]any, len(values))
	for index, value := range values {
		result[index] = value
	}
	return result, nil
}

// Catalog callbacks are extension-owned code. A broken provider must return a
// structured failure rather than unwind through the HTTP handler.
func ruleExtensions(registry ports.RuleRegistry) (values []ports.Extension, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			values = nil
			err = domain.NewError("okf_extension_catalog_failed", 500, fmt.Sprintf("rule catalog metadata failed: %v", failure), nil)
		}
	}()
	return registry.Catalog(), nil
}

func relationshipExtension(adapter ports.RelationshipAdapter) (value ports.Extension, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			err = domain.NewError("okf_relationship_adapter_failed", 500, fmt.Sprintf("relationship adapter metadata failed: %v", failure), nil)
		}
	}()
	value = ports.Extension{ID: adapter.ID(), Version: adapter.Version(), Kind: "relationship_adapter", Capabilities: []string{"semantic-links", "renderer-neutral"}}
	if described, ok := adapter.(ports.ExtensionDescriber); ok {
		value.Description = described.Description()
	}
	if schema, ok := adapter.(ports.ParameterSchemaProvider); ok {
		value.ParameterSchema = domain.CloneMap(schema.ParameterSchema())
	}
	return value, nil
}
