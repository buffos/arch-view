package profile

import (
	"context"
	"fmt"
	"strings"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
)

type metadataEqualsStrategy struct{}

func (metadataEqualsStrategy) ID() string      { return "okf.rule.metadata_equals" }
func (metadataEqualsStrategy) Version() string { return "1" }
func (metadataEqualsStrategy) Description() string {
	return "Map an exact frontmatter value to renderer-neutral presentation facts."
}
func (metadataEqualsStrategy) Evaluate(_ context.Context, document domain.ConceptDocument, invocation domain.RuleInvocation) (ports.RuleResult, error) {
	field, _ := invocation.Parameters["field"].(string)
	if field == "" {
		return ports.RuleResult{}, fmt.Errorf("metadata_equals requires a field")
	}
	actual, exists := metadataField(document, field)
	want, configured := invocation.Parameters["value"]
	if !exists || !configured || !metadataEqual(actual, want) {
		return ports.RuleResult{}, nil
	}
	return resultFromParameters(invocation.Parameters), nil
}

type metadataContainsStrategy struct{}

func (metadataContainsStrategy) ID() string      { return "okf.rule.metadata_contains" }
func (metadataContainsStrategy) Version() string { return "1" }
func (metadataContainsStrategy) Description() string {
	return "Map a collection or string containing a value to presentation facts."
}
func (metadataContainsStrategy) Evaluate(_ context.Context, document domain.ConceptDocument, invocation domain.RuleInvocation) (ports.RuleResult, error) {
	field, _ := invocation.Parameters["field"].(string)
	needle, configured := invocation.Parameters["value"]
	if field == "" || !configured || needle == nil {
		return ports.RuleResult{}, fmt.Errorf("metadata_contains requires field and value")
	}
	value, exists := metadataField(document, field)
	if !exists || !metadataContains(value, needle) {
		return ports.RuleResult{}, nil
	}
	return resultFromParameters(invocation.Parameters), nil
}

type stateMappingStrategy struct{}

func (stateMappingStrategy) ID() string      { return "okf.rule.state_mapping" }
func (stateMappingStrategy) Version() string { return "1" }
func (stateMappingStrategy) Description() string {
	return "Map a declared state to an effective state or presentation token."
}
func (stateMappingStrategy) Evaluate(_ context.Context, document domain.ConceptDocument, invocation domain.RuleInvocation) (ports.RuleResult, error) {
	field := "state"
	if configured, ok := invocation.Parameters["field"].(string); ok && configured != "" {
		field = configured
	}
	value, _ := metadataField(document, field)
	state, _ := value.(string)
	mapping, _ := invocation.Parameters["mapping"].(map[string]any)
	if state == "" || mapping == nil {
		return ports.RuleResult{}, nil
	}
	mapped, ok := mapping[state].(string)
	if !ok {
		return ports.RuleResult{}, nil
	}
	return ports.RuleResult{EffectiveState: mapped, Token: stringValue(invocation.Parameters["token"])}, nil
}

type labelTemplateStrategy struct{}

func (labelTemplateStrategy) ID() string      { return "okf.rule.label_template" }
func (labelTemplateStrategy) Version() string { return "1" }
func (labelTemplateStrategy) Description() string {
	return "Select a bounded label template from source metadata."
}
func (labelTemplateStrategy) Evaluate(_ context.Context, document domain.ConceptDocument, invocation domain.RuleInvocation) (ports.RuleResult, error) {
	template, _ := invocation.Parameters["template"].(string)
	if template == "" {
		return ports.RuleResult{}, fmt.Errorf("label_template requires a template")
	}
	label := strings.ReplaceAll(template, "{title}", document.Title)
	label = strings.ReplaceAll(label, "{type}", document.Type)
	return ports.RuleResult{Label: label}, nil
}

type visibilityStrategy struct{}

func (visibilityStrategy) ID() string      { return "okf.rule.visibility" }
func (visibilityStrategy) Version() string { return "1" }
func (visibilityStrategy) Description() string {
	return "Control visibility with a declarative metadata match."
}
func (visibilityStrategy) Evaluate(_ context.Context, document domain.ConceptDocument, invocation domain.RuleInvocation) (ports.RuleResult, error) {
	field, _ := invocation.Parameters["field"].(string)
	value, exists := metadataField(document, field)
	want, configured := invocation.Parameters["value"]
	if field == "" || !exists || !configured || !metadataEqual(value, want) {
		return ports.RuleResult{}, nil
	}
	visible := true
	if configured, ok := invocation.Parameters["visible"].(bool); ok {
		visible = configured
	}
	return ports.RuleResult{Visible: &visible}, nil
}

func resultFromParameters(parameters map[string]any) ports.RuleResult {
	result := ports.RuleResult{Annotations: make(map[string]any)}
	if value, ok := parameters["role"].(string); ok {
		result.Role = value
	}
	if value, ok := parameters["token"].(string); ok {
		result.Token = value
	}
	if value, ok := parameters["shape"].(string); ok {
		result.Shape = value
	}
	if value, ok := parameters["label"].(string); ok {
		result.Label = value
	}
	if value, ok := parameters["state"].(string); ok {
		result.EffectiveState = value
	}
	if annotations, ok := parameters["annotations"].(map[string]any); ok {
		result.Annotations = domain.CloneMap(annotations)
	}
	return result
}

func stringValue(value any) string { text, _ := value.(string); return text }
