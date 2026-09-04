package source

import "github.com/buffo/arch-view/internal/okf/domain"

// IndexLimits bound accepted source input, independently of projection limits.
// Non-positive values use defaults. Configure before sharing a scanner.
type IndexLimits struct {
	MaxFiles int
	MaxBytes int64
}

func (limits IndexLimits) resolved() IndexLimits {
	if limits.MaxFiles <= 0 {
		limits.MaxFiles = 10000
	}
	if limits.MaxBytes <= 0 {
		limits.MaxBytes = 64 << 20
	}
	return limits
}

func sourceLimitDiagnostic(bundleID, limit string, maximum int64) domain.Diagnostic {
	return domain.Diagnostic{
		Code: "okf_bundle_source_limit", Severity: "error", Category: "scale",
		BundleID: bundleID, Message: "Bundle exceeds the source indexing safety limit.",
		Details:  map[string]any{"limit": limit, "maximum": maximum},
		Recovery: "Split the bundle into smaller independent bundles or increase the host scanner limit, then refresh.",
	}
}
