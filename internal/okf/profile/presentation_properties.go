package profile

import (
	"context"
	"fmt"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
)

// Providers use the existing profile rule invocation format, priority ordering,
// parameter validation, failure isolation and conflict handling.
func (registry *Registry) RegisterPresentationPropertyProvider(provider ports.PresentationPropertyProvider) (err error) {
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("presentation provider registration failed: %v", failure)
		}
	}()
	metadata := provider.Metadata()
	if !domain.ValidExtensionIdentity(metadata.ID, metadata.Version) || metadata.Description == "" || metadata.ParameterSchema == nil || len(metadata.Capabilities) == 0 {
		return fmt.Errorf("presentation provider requires valid identity, description, capabilities and parameter schema")
	}
	metadata.Kind = "presentation_property"
	metadata.ShapeDefinition = nil
	metadata, err = captureExtensionValue(metadata)
	if err != nil {
		return err
	}
	return registry.registerStrategy(presentationPropertyStrategy{provider, metadata}, &metadata)
}

type presentationPropertyStrategy struct {
	provider ports.PresentationPropertyProvider
	metadata ports.Extension
}

func (strategy presentationPropertyStrategy) ID() string      { return strategy.metadata.ID }
func (strategy presentationPropertyStrategy) Version() string { return strategy.metadata.Version }
func (strategy presentationPropertyStrategy) Description() string {
	return strategy.metadata.Description
}
func (strategy presentationPropertyStrategy) ParameterSchema() map[string]any {
	return domain.CloneMap(strategy.metadata.ParameterSchema)
}
func (strategy presentationPropertyStrategy) ValidateParameters(parameters map[string]any) error {
	return strategy.provider.ValidateParameters(parameters)
}
func (strategy presentationPropertyStrategy) Evaluate(ctx context.Context, document domain.ConceptDocument, invocation domain.RuleInvocation) (ports.RuleResult, error) {
	value, err := strategy.provider.Properties(ctx, document, invocation.Parameters)
	if err != nil {
		return ports.RuleResult{}, err
	}
	return ports.RuleResult{Label: value.Label, Token: value.Token, Shape: value.Shape, Annotations: value.Annotations}, nil
}
