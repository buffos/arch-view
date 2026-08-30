package quality

import (
	"strings"
)

// ExitPolicy is a caller-owned projection over a complete quality report. It
// never changes the report and intentionally allows callers to choose whether
// suppressed or baseline findings participate in the decision.
type ExitPolicy struct {
	MinimumSeverity string   `json:"minimum_severity,omitempty"`
	Statuses        []string `json:"statuses,omitempty"`
}

// ValidateExitPolicy checks the small, transport-independent policy shape.
func ValidateExitPolicy(policy ExitPolicy) error {
	if policy.MinimumSeverity != "" && severityRank(policy.MinimumSeverity) < 0 {
		return newQualityError(ErrorQueryInvalid, "quality exit policy severity is unsupported", map[string]any{"severity": policy.MinimumSeverity})
	}
	seen := make(map[string]struct{}, len(policy.Statuses))
	for _, status := range policy.Statuses {
		if !validFindingStatus(status) {
			return newQualityError(ErrorQueryInvalid, "quality exit policy status is unsupported", map[string]any{"status": status})
		}
		if _, exists := seen[status]; exists {
			return newQualityError(ErrorQueryInvalid, "quality exit policy statuses must be unique", map[string]any{"status": status})
		}
		seen[status] = struct{}{}
	}
	return nil
}

// ExitPolicyTriggered reports whether the policy matches any finding. A nil
// report never triggers. An omitted severity or status list means that axis is
// unrestricted; the CLI supplies active as its explicit default.
func ExitPolicyTriggered(report *QualityEvaluation, policy ExitPolicy) (bool, error) {
	if err := ValidateExitPolicy(policy); err != nil {
		return false, err
	}
	if report == nil {
		return false, nil
	}
	if policy.MinimumSeverity == "" && len(policy.Statuses) == 0 {
		return false, nil
	}
	statuses := make(map[string]struct{}, len(policy.Statuses))
	for _, status := range policy.Statuses {
		statuses[status] = struct{}{}
	}
	minimum := severityRank(policy.MinimumSeverity)
	for _, finding := range report.Findings {
		if minimum >= 0 && severityRank(finding.Severity) < minimum {
			continue
		}
		if len(statuses) > 0 {
			if _, ok := statuses[finding.Status]; !ok {
				continue
			}
		}
		return true, nil
	}
	return false, nil
}

func severityRank(value string) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case SeverityInfo:
		return 0
	case SeverityWarning:
		return 1
	case SeverityError:
		return 2
	case SeverityBlocker:
		return 3
	default:
		return -1
	}
}
