package profile

import (
	"fmt"
	"strings"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
)

func validateRuleParameters(strategy ports.RuleStrategy, parameters map[string]any) (err error) {
	validator, ok := strategy.(ports.RuleParameterValidator)
	if !ok {
		return nil
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("rule parameter validation panicked: %v", recovered)
		}
	}()
	return validator.ValidateParameters(domain.CloneMap(parameters))
}

func (registry *Registry) validateResolvedParameters(value domain.Profile) []domain.Diagnostic {
	var diagnostics []domain.Diagnostic
	for _, rule := range value.Rules {
		if !rule.Enabled {
			continue
		}
		strategy, exists := registry.Resolve(rule.RuleID, rule.Version)
		if !exists {
			continue
		}
		if err := validateRuleParameters(strategy, rule.Parameters); err != nil {
			diagnostics = append(diagnostics, domain.Diagnostic{Code: "okf_rule_invalid", Severity: "error", Category: "rule", ProfileID: value.ProfileID, Message: err.Error(), Details: map[string]any{"rule_id": rule.RuleID}})
		}
	}
	return diagnostics
}

func requiredText(parameters map[string]any, key string) error {
	value, ok := parameters[key].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s must be a nonempty string", key)
	}
	return nil
}

func validateMatch(parameters map[string]any) error {
	if err := requiredText(parameters, "field"); err != nil {
		return err
	}
	if _, exists := parameters["value"]; !exists {
		return fmt.Errorf("value is required")
	}
	return nil
}

func validatePresentation(parameters map[string]any) error {
	for _, key := range []string{"role", "token", "shape", "label", "state"} {
		if value, exists := parameters[key]; exists {
			if _, ok := value.(string); !ok {
				return fmt.Errorf("%s must be a string", key)
			}
		}
	}
	if value, exists := parameters["annotations"]; exists {
		if _, ok := value.(map[string]any); !ok {
			return fmt.Errorf("annotations must be an object")
		}
	}
	return nil
}

func (metadataEqualsStrategy) ValidateParameters(parameters map[string]any) error {
	if err := validateMatch(parameters); err != nil {
		return err
	}
	return validatePresentation(parameters)
}

func (metadataContainsStrategy) ValidateParameters(parameters map[string]any) error {
	if err := validateMatch(parameters); err != nil {
		return err
	}
	if parameters["value"] == nil || strings.TrimSpace(fmt.Sprint(parameters["value"])) == "" {
		return fmt.Errorf("value must not be empty")
	}
	return validatePresentation(parameters)
}

func (stateMappingStrategy) ValidateParameters(parameters map[string]any) error {
	if _, exists := parameters["field"]; exists {
		if err := requiredText(parameters, "field"); err != nil {
			return err
		}
	}
	mapping, ok := parameters["mapping"].(map[string]any)
	if !ok {
		return fmt.Errorf("mapping must be an object of strings")
	}
	for _, value := range mapping {
		if _, ok := value.(string); !ok {
			return fmt.Errorf("mapping values must be strings")
		}
	}
	return validatePresentation(parameters)
}

func (labelTemplateStrategy) ValidateParameters(parameters map[string]any) error {
	return requiredText(parameters, "template")
}

func (visibilityStrategy) ValidateParameters(parameters map[string]any) error {
	if err := validateMatch(parameters); err != nil {
		return err
	}
	if value, exists := parameters["visible"]; exists {
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("visible must be a boolean")
		}
	}
	return nil
}
