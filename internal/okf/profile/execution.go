package profile

import (
	"context"
	"fmt"
	"reflect"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
)

func applyAnnotation(result *ports.RuleResult, priorities map[string]int, priority int, key string, value any, diagnostics *[]domain.Diagnostic, conceptID string) {
	field := "annotation:" + key
	previous, exists := priorities[field]
	if !exists || priority > previous {
		result.Annotations[key] = domain.CloneMap(map[string]any{key: value})[key]
		priorities[field] = priority
	} else if priority == previous && !reflect.DeepEqual(result.Annotations[key], value) {
		delete(result.Annotations, key)
		*diagnostics = append(*diagnostics, domain.Diagnostic{Code: "okf_rule_conflict", Severity: "warning", Category: "rule", ConceptID: conceptID, Message: "Equal-priority rules produced conflicting annotation values.", Details: map[string]any{"field": key}})
	}
}

// Extensions receive private input copies and cannot unwind the host request.
func evaluateRule(ctx context.Context, strategy ports.RuleStrategy, document domain.ConceptDocument, invocation domain.RuleInvocation) (result ports.RuleResult, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			result = ports.RuleResult{}
			err = fmt.Errorf("rule %s panicked: %v", invocation.RuleID, recovered)
		}
	}()
	if err := ctx.Err(); err != nil {
		return result, err
	}
	invocation.Parameters = domain.CloneMap(invocation.Parameters)
	result, err = strategy.Evaluate(ctx, domain.CloneDocument(document), invocation)
	if err != nil {
		return ports.RuleResult{}, err
	}
	return captureExtensionValue(result)
}
