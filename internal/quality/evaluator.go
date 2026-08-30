package quality

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// EvaluateQualityProfile validates the profile, computes registered metric
// strategies, and evaluates each enabled rule. Provider and rule failures are
// isolated in diagnostics and coverage so they cannot erase unrelated facts.
func EvaluateQualityProfile(profile QualityProfile, input EvaluationInput, catalog *Catalog) (QualityEvaluation, error) {
	validated, err := ValidateQualityProfile(profile, catalog)
	if err != nil {
		return QualityEvaluation{}, err
	}
	if catalog == nil {
		return QualityEvaluation{}, newQualityError(ErrorCatalogInvalid, "quality rule catalog is unavailable", nil)
	}
	input = normalizeEvaluationInput(input)
	context := EvaluationContext{Profile: validated, Input: input, Architecture: input.Architecture}
	batch := MetricBatch{Metrics: []MetricFact{}, Coverage: []QualityCoverage{}, Diagnostics: []QualityDiagnostic{}}
	providers := catalog.ListMetricProviders()
	providerIdentities := make([]ProviderIdentity, 0, len(providers))
	providerFailures := make(map[string]string)
	for _, provider := range providers {
		providerIdentities = append(providerIdentities, ProviderIdentity{ID: provider.ID(), Version: provider.Version()})
		provided, providerErr := provider.Compute(context)
		if providerErr != nil {
			message := providerErr.Error()
			batch.Diagnostics = append(batch.Diagnostics, QualityDiagnostic{
				Code:     string(ErrorMetricUnavailable),
				Message:  "metric provider failed; affected rules remain explicitly unevaluable",
				Severity: SeverityError,
				Details:  map[string]any{"provider_id": provider.ID(), "provider_version": provider.Version(), "error": message},
			})
			for _, capability := range provider.Capabilities() {
				providerFailures[capability.ID] = message
			}
			continue
		}
		batch.Metrics = append(batch.Metrics, provided.Metrics...)
		batch.Coverage = append(batch.Coverage, provided.Coverage...)
		batch.Diagnostics = append(batch.Diagnostics, provided.Diagnostics...)
	}

	findings := make([]QualityFinding, 0)
	coverage := make([]QualityCoverage, 0)
	diagnostics := append([]QualityDiagnostic{}, batch.Diagnostics...)
	for _, binding := range validated.EnabledRules {
		if !binding.Enabled {
			continue
		}
		rule, ok := catalog.ResolveQualityRule(binding.RuleID, binding.RuleVersion)
		if !ok {
			// Validation already guards this path. Keep it defensive for a
			// catalog that is changed concurrently by a caller.
			diagnostics = append(diagnostics, QualityDiagnostic{Code: string(ErrorRuleUnknown), Message: "quality rule disappeared from the catalog during evaluation", Severity: SeverityError, RuleID: binding.RuleID, RuleVersion: binding.RuleVersion})
			continue
		}
		ruleContext := context
		bindingCopy := binding
		ruleContext.RuleBinding = &bindingCopy
		if unavailable, status, reason := requiredCapabilityStatus(rule.RequiredCapabilities(), input, batch, providerFailures); unavailable {
			values := coverageForRule(rule, status, reason, 0, 0, input, "quality:coverage")
			setAvailableCapabilities(values, rule, input, batch)
			coverage = append(coverage, values...)
			continue
		}
		result, evaluateErr := rule.Evaluate(ruleContext, batch)
		if evaluateErr != nil {
			diagnostics = append(diagnostics, QualityDiagnostic{Code: string(ErrorEvaluationFailed), Message: "quality rule evaluation failed", Severity: SeverityError, RuleID: rule.ID(), RuleVersion: rule.Version(), Details: map[string]any{"error": evaluateErr.Error()}})
			values := coverageForRule(rule, CoverageNotEvaluable, evaluateErr.Error(), 0, 0, input, "quality:evaluator")
			setAvailableCapabilities(values, rule, input, batch)
			coverage = append(coverage, values...)
			continue
		}
		findings = append(findings, result.Findings...)
		setAvailableCapabilities(result.Coverage, rule, input, batch)
		coverage = append(coverage, result.Coverage...)
		diagnostics = append(diagnostics, result.Diagnostics...)
	}

	report := QualityEvaluation{
		SchemaVersion:      SchemaVersion,
		SourceSnapshotIDs:  snapshotIDs(input.SourceSnapshots),
		ProfileID:          validated.ProfileID,
		ProfileVersion:     validated.ProfileVersion,
		ProviderIdentities: providerIdentities,
		Coverage:           coverage,
		Metrics:            batch.Metrics,
		Findings:           findings,
		Diagnostics:        diagnostics,
		Extensions:         []ExtensionBlock{},
	}
	report = normalizeEvaluation(report, validated, input.Options)
	if err := ValidateQualityEvaluation(report); err != nil {
		return QualityEvaluation{}, err
	}
	return report, nil
}

// Evaluate is a concise alias for callers that already have a validated
// catalog boundary.
func Evaluate(profile QualityProfile, input EvaluationInput, catalog *Catalog) (QualityEvaluation, error) {
	return EvaluateQualityProfile(profile, input, catalog)
}

// NormalizeQualityEvaluation canonicalizes ordering and computes the
// reproducibility fingerprint and report digest. Operational timestamps are
// intentionally not part of the contract.
func NormalizeQualityEvaluation(report QualityEvaluation) (QualityEvaluation, error) {
	profile := QualityProfile{SchemaVersion: SchemaVersion, ProfileID: report.ProfileID, ProfileVersion: report.ProfileVersion}
	normalized := normalizeEvaluation(report, profile, nil)
	if err := ValidateQualityEvaluation(normalized); err != nil {
		return QualityEvaluation{}, err
	}
	return normalized, nil
}

func ValidateQualityEvaluation(report QualityEvaluation) error {
	if report.SchemaVersion != SchemaVersion {
		return newQualityError(ErrorReportInvalid, "quality report schema version is unsupported", map[string]any{"schema_version": report.SchemaVersion})
	}
	if strings.TrimSpace(report.EvaluationID) == "" || !strings.HasPrefix(report.EvaluationID, "evaluation:") {
		return newQualityError(ErrorReportInvalid, "quality report evaluation identity is missing or invalid", map[string]any{"evaluation_id": report.EvaluationID})
	}
	if !validNamespacedID(report.ProfileID) || !validVersion(report.ProfileVersion) {
		return newQualityError(ErrorReportInvalid, "quality report profile identity is invalid", map[string]any{"profile_id": report.ProfileID, "profile_version": report.ProfileVersion})
	}
	if !validContentDigest(report.EvaluationFingerprint) || !validContentDigest(report.ReportDigest) {
		return newQualityError(ErrorReportInvalid, "quality report digests must be lowercase SHA-256 values", nil)
	}
	if report.SourceSnapshotIDs == nil || report.ProviderIdentities == nil || report.Coverage == nil || report.Metrics == nil || report.Findings == nil || report.Diagnostics == nil || report.Extensions == nil {
		return newQualityError(ErrorReportInvalid, "quality report collections must be serialized as arrays", map[string]any{"source_snapshot_ids_nil": report.SourceSnapshotIDs == nil, "provider_identities_nil": report.ProviderIdentities == nil, "coverage_nil": report.Coverage == nil, "metrics_nil": report.Metrics == nil, "findings_nil": report.Findings == nil, "diagnostics_nil": report.Diagnostics == nil, "extensions_nil": report.Extensions == nil})
	}
	if !sort.StringsAreSorted(report.SourceSnapshotIDs) || !uniqueStrings(report.SourceSnapshotIDs) {
		return newQualityError(ErrorReportInvalid, "quality report source snapshot IDs must be sorted and unique", nil)
	}
	snapshotIDs := make(map[string]struct{}, len(report.SourceSnapshotIDs))
	for _, snapshotID := range report.SourceSnapshotIDs {
		if strings.TrimSpace(snapshotID) == "" || strings.TrimSpace(snapshotID) != snapshotID {
			return newQualityError(ErrorReportInvalid, "quality report source snapshot identity is invalid", map[string]any{"snapshot_id": snapshotID})
		}
		snapshotIDs[snapshotID] = struct{}{}
	}
	providerIDs := make(map[string]struct{}, len(report.ProviderIdentities))
	for _, provider := range report.ProviderIdentities {
		if !validNamespacedID(provider.ID) || !validVersion(provider.Version) {
			return newQualityError(ErrorReportInvalid, "quality report provider identity is invalid", map[string]any{"provider_id": provider.ID, "version": provider.Version})
		}
		key := providerIdentityKey(provider)
		if _, exists := providerIDs[key]; exists {
			return newQualityError(ErrorReportInvalid, "quality report provider identities must be unique", map[string]any{"provider_id": provider.ID, "version": provider.Version})
		}
		providerIDs[key] = struct{}{}
	}
	if !sort.SliceIsSorted(report.ProviderIdentities, func(i, j int) bool {
		return providerIdentityKey(report.ProviderIdentities[i]) < providerIdentityKey(report.ProviderIdentities[j])
	}) {
		return newQualityError(ErrorReportInvalid, "quality report providers must be canonically ordered", nil)
	}
	if err := validateQualityExtensions(report.Extensions); err != nil {
		return err
	}
	metricIDs := make(map[string]struct{}, len(report.Metrics))
	for _, metric := range report.Metrics {
		if metric.ID == "" || !validNamespacedID(metric.MetricID) || !validNamespacedID(metric.FormulaID) || !validVersion(metric.FormulaVersion) || !validEntityRef(metric.SubjectRef) || !validMetricValue(metric.Value) || (metric.Unit != "" && !validNamespacedID(metric.Unit)) || !validFactProvenance(metric.Provenance) {
			return newQualityError(ErrorReportInvalid, "quality report metric is incomplete", map[string]any{"metric_id": metric.ID})
		}
		if err := validateQualityExtensions(metric.Extensions); err != nil {
			return err
		}
		if !entitySnapshotIsDeclared(metric.SubjectRef, snapshotIDs) {
			return newQualityError(ErrorReportInvalid, "quality metric references an undeclared source snapshot", map[string]any{"metric_id": metric.ID, "snapshot_id": metric.SubjectRef.SnapshotID})
		}
		if _, exists := metricIDs[metric.ID]; exists {
			return newQualityError(ErrorReportInvalid, "quality report metric IDs must be unique", map[string]any{"metric_id": metric.ID})
		}
		metricIDs[metric.ID] = struct{}{}
	}
	if !sort.SliceIsSorted(report.Metrics, func(i, j int) bool { return metricKey(report.Metrics[i]) < metricKey(report.Metrics[j]) }) {
		return newQualityError(ErrorReportInvalid, "quality report metrics must be canonically ordered", nil)
	}
	for _, value := range report.Coverage {
		if !validNamespacedID(value.RuleID) || !validVersion(value.RuleVersion) || !validCoverageStatus(value.Status) || value.RequiredCapabilities == nil || value.AvailableCapabilities == nil || !sort.StringsAreSorted(value.RequiredCapabilities) || !uniqueStrings(value.RequiredCapabilities) || !sort.StringsAreSorted(value.AvailableCapabilities) || !uniqueStrings(value.AvailableCapabilities) || !nonNegativeCounts(value.SubjectCount, value.EvaluatedCount) || value.SubjectCount != nil && value.EvaluatedCount != nil && *value.EvaluatedCount > *value.SubjectCount || !validFactProvenance(value.Provenance) {
			return newQualityError(ErrorReportInvalid, "quality report coverage is incomplete or invalid", map[string]any{"rule_id": value.RuleID})
		}
	}
	if !sort.SliceIsSorted(report.Coverage, func(i, j int) bool { return coverageKey(report.Coverage[i]) < coverageKey(report.Coverage[j]) }) {
		return newQualityError(ErrorReportInvalid, "quality report coverage must be canonically ordered", nil)
	}
	findingIDs := make(map[string]struct{}, len(report.Findings))
	for _, finding := range report.Findings {
		if finding.ID == "" || finding.FindingKey == "" || !validNamespacedID(finding.RuleID) || !validVersion(finding.RuleVersion) || (finding.AssessmentKind != AssessmentExact && finding.AssessmentKind != AssessmentSignal) || !validFindingStatus(finding.Status) || !validSeverity(finding.Severity) || !validEntityRef(finding.SubjectRef) || !validNamespacedID(finding.MessageCode) || strings.TrimSpace(finding.Message) == "" || finding.ObservedMetricIDs == nil || finding.Evidence.SourceSpans == nil || finding.Evidence.EntityRefs == nil || finding.Evidence.RelationRefs == nil || finding.Evidence.MetricRefs == nil || finding.Evidence.DiagnosticRefs == nil || !sort.StringsAreSorted(finding.ObservedMetricIDs) || !uniqueStrings(finding.ObservedMetricIDs) || !sort.StringsAreSorted(finding.Evidence.MetricRefs) || !uniqueStrings(finding.Evidence.MetricRefs) || !sort.StringsAreSorted(finding.Evidence.DiagnosticRefs) || !uniqueStrings(finding.Evidence.DiagnosticRefs) || !validFactProvenance(finding.Provenance) {
			return newQualityError(ErrorReportInvalid, "quality report finding is incomplete or invalid", map[string]any{"finding_id": finding.ID})
		}
		if err := validateQualityExtensions(finding.Extensions); err != nil {
			return err
		}
		if !entitySnapshotIsDeclared(finding.SubjectRef, snapshotIDs) {
			return newQualityError(ErrorReportInvalid, "quality finding references an undeclared source snapshot", map[string]any{"finding_id": finding.ID, "snapshot_id": finding.SubjectRef.SnapshotID})
		}
		if _, exists := findingIDs[finding.ID]; exists {
			return newQualityError(ErrorReportInvalid, "quality report finding IDs must be unique", map[string]any{"finding_id": finding.ID})
		}
		findingIDs[finding.ID] = struct{}{}
		for _, metricID := range finding.ObservedMetricIDs {
			if _, exists := metricIDs[metricID]; !exists {
				return newQualityError(ErrorReportInvalid, "finding references an unknown observed metric", map[string]any{"finding_id": finding.ID, "metric_id": metricID})
			}
		}
		for _, metricID := range finding.Evidence.MetricRefs {
			if _, exists := metricIDs[metricID]; !exists {
				return newQualityError(ErrorReportInvalid, "finding evidence references an unknown metric", map[string]any{"finding_id": finding.ID, "metric_id": metricID})
			}
		}
		for _, span := range finding.Evidence.SourceSpans {
			if !validQualitySpan(span) {
				return newQualityError(ErrorReportInvalid, "finding source evidence contains an invalid span", map[string]any{"finding_id": finding.ID})
			}
		}
		for _, reference := range append(append([]EntityRef{}, finding.Evidence.EntityRefs...), finding.Evidence.RelationRefs...) {
			if !validEntityRef(reference) {
				return newQualityError(ErrorReportInvalid, "finding evidence contains an incomplete entity reference", map[string]any{"finding_id": finding.ID})
			}
			if !entitySnapshotIsDeclared(reference, snapshotIDs) {
				return newQualityError(ErrorReportInvalid, "finding evidence references an undeclared source snapshot", map[string]any{"finding_id": finding.ID, "snapshot_id": reference.SnapshotID})
			}
		}
		if finding.Comparison != nil {
			if !validOperator(finding.Comparison.Operator) || finding.Comparison.ObservedMetricID == "" {
				return newQualityError(ErrorReportInvalid, "finding comparison is invalid", map[string]any{"finding_id": finding.ID})
			}
			if _, exists := metricIDs[finding.Comparison.ObservedMetricID]; !exists {
				return newQualityError(ErrorReportInvalid, "finding comparison references an unknown metric", map[string]any{"finding_id": finding.ID, "metric_id": finding.Comparison.ObservedMetricID})
			}
			if !validMetricValue(finding.Comparison.Limit) || finding.Comparison.Unit != "" && !validNamespacedID(finding.Comparison.Unit) {
				return newQualityError(ErrorReportInvalid, "finding comparison limit is invalid", map[string]any{"finding_id": finding.ID})
			}
			observed := reportMetricByID(report.Metrics, finding.Comparison.ObservedMetricID)
			if observed == nil || observed.Value.Kind != finding.Comparison.Limit.Kind || observed.Unit != finding.Comparison.Unit {
				return newQualityError(ErrorReportInvalid, "finding comparison is incompatible with its observed metric", map[string]any{"finding_id": finding.ID})
			}
		}
		if finding.AssessmentKind == AssessmentExact && finding.Status == StatusActive && finding.Comparison != nil && finding.Comparison.ObservedMetricID != "" {
			if _, exists := metricIDs[finding.Comparison.ObservedMetricID]; !exists {
				return newQualityError(ErrorReportInvalid, "exact finding comparison references an unknown metric", map[string]any{"finding_id": finding.ID, "metric_id": finding.Comparison.ObservedMetricID})
			}
		}
	}
	if !sort.SliceIsSorted(report.Findings, func(i, j int) bool { return findingKey(report.Findings[i]) < findingKey(report.Findings[j]) }) {
		return newQualityError(ErrorReportInvalid, "quality report findings must be canonically ordered", nil)
	}
	if !sort.SliceIsSorted(report.Diagnostics, func(i, j int) bool {
		left, _ := json.Marshal(report.Diagnostics[i])
		right, _ := json.Marshal(report.Diagnostics[j])
		return string(left) < string(right)
	}) {
		return newQualityError(ErrorReportInvalid, "quality report diagnostics must be canonically ordered", nil)
	}
	for _, diagnostic := range report.Diagnostics {
		if strings.TrimSpace(diagnostic.Code) == "" || strings.TrimSpace(diagnostic.Message) == "" || !validSeverity(diagnostic.Severity) {
			return newQualityError(ErrorReportInvalid, "quality report diagnostic is incomplete or invalid", map[string]any{"code": diagnostic.Code})
		}
		if diagnostic.RuleID != "" && (!validNamespacedID(diagnostic.RuleID) || diagnostic.RuleVersion != "" && !validVersion(diagnostic.RuleVersion)) {
			return newQualityError(ErrorReportInvalid, "quality report diagnostic rule identity is invalid", map[string]any{"code": diagnostic.Code})
		}
		if diagnostic.SubjectRef != nil && !validEntityRef(*diagnostic.SubjectRef) {
			return newQualityError(ErrorReportInvalid, "quality report diagnostic subject reference is invalid", map[string]any{"code": diagnostic.Code})
		}
	}
	if expected := reportDigest(report); expected != report.ReportDigest {
		return newQualityError(ErrorReportInvalid, "quality report digest does not match its canonical payload", map[string]any{"expected": expected.Value, "actual": report.ReportDigest.Value})
	}
	if _, err := json.Marshal(report); err != nil {
		return newQualityError(ErrorReportInvalid, "quality report is not serializable", map[string]any{"error": err.Error()})
	}
	return nil
}

func validContentDigest(value ContentDigest) bool {
	if value.Algorithm != "hash:sha-256" || len(value.Value) != 64 || strings.ToLower(value.Value) != value.Value {
		return false
	}
	_, err := hex.DecodeString(value.Value)
	return err == nil
}

func validEntityRef(value EntityRef) bool {
	return strings.TrimSpace(value.Kind) == value.Kind && strings.TrimSpace(value.ID) == value.ID && strings.TrimSpace(value.Kind) != "" && strings.TrimSpace(value.ID) != ""
}

func entitySnapshotIsDeclared(value EntityRef, snapshotIDs map[string]struct{}) bool {
	if value.SnapshotID == "" {
		return true
	}
	_, ok := snapshotIDs[value.SnapshotID]
	return ok
}

func validMetricValue(value MetricValue) bool {
	if value.Value == nil {
		return false
	}
	switch value.Kind {
	case ValueInteger:
		_, ok := integerValue(value.Value)
		return ok
	case ValueDecimal:
		_, ok := decimalValue(value.Value)
		return ok
	case ValueBoolean:
		_, ok := value.Value.(bool)
		return ok
	case ValueText:
		_, ok := value.Value.(string)
		return ok
	default:
		return false
	}
}

func validFactProvenance(value FactProvenance) bool {
	if strings.TrimSpace(value.Status) == "" || strings.TrimSpace(value.Basis) == "" || !validNamespacedID(value.Provider) || !validVersion(value.ProviderVersion) || value.EvidenceIDs == nil || !sort.StringsAreSorted(value.EvidenceIDs) || !uniqueStrings(value.EvidenceIDs) {
		return false
	}
	switch value.Status {
	case "observed", "absent", "unknown", "unsupported", "partial", CoverageNotEvaluable:
		return true
	default:
		return false
	}
}

func nonNegativeCounts(subjects, evaluated *int) bool {
	return (subjects == nil || *subjects >= 0) && (evaluated == nil || *evaluated >= 0)
}

func validateQualityExtensions(values []ExtensionBlock) error {
	if values == nil {
		return newQualityError(ErrorReportInvalid, "quality extensions must be serialized as an array", nil)
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if strings.TrimSpace(value.Namespace) == "" || strings.TrimSpace(value.SchemaVersion) == "" || strings.TrimSpace(value.Capability) == "" {
			return newQualityError(ErrorReportInvalid, "quality extension metadata is incomplete", nil)
		}
		key := value.Namespace + "\x00" + value.SchemaVersion + "\x00" + value.Capability
		if _, exists := seen[key]; exists {
			return newQualityError(ErrorReportInvalid, "quality extension keys must be unique", map[string]any{"key": key})
		}
		seen[key] = struct{}{}
	}
	if !sort.SliceIsSorted(values, func(i, j int) bool {
		left := values[i].Namespace + "\x00" + values[i].SchemaVersion + "\x00" + values[i].Capability
		right := values[j].Namespace + "\x00" + values[j].SchemaVersion + "\x00" + values[j].Capability
		return left < right
	}) {
		return newQualityError(ErrorReportInvalid, "quality extensions must be canonically ordered", nil)
	}
	return nil
}

func reportMetricByID(values []MetricFact, id string) *MetricFact {
	for index := range values {
		if values[index].ID == id {
			return &values[index]
		}
	}
	return nil
}

func reportDigest(report QualityEvaluation) ContentDigest {
	semantic := report
	semantic.ReportDigest = ContentDigest{}
	return digestJSON(semantic)
}

func requiredCapabilityStatus(required []string, input EvaluationInput, batch MetricBatch, failures map[string]string) (bool, string, string) {
	for _, capability := range required {
		if reason, failed := failures[capability]; failed {
			return true, CoveragePartial, reason
		}
		if capabilityAvailable(capability, input, batch) {
			continue
		}
		if sourceCapabilityMentioned(capability, input) {
			return true, CoverageNotEvaluable, "required capability did not produce a compatible metric"
		}
		return true, CoverageUnsupported, "no source or architecture provider advertises the required capability"
	}
	return false, "", ""
}

func capabilityAvailable(capability string, input EvaluationInput, batch MetricBatch) bool {
	for _, metric := range batch.Metrics {
		if metric.MetricID == capability {
			return true
		}
	}
	for _, snapshot := range input.SourceSnapshots {
		for _, descriptor := range snapshot.Capabilities {
			if descriptor.ID == capability {
				return true
			}
		}
	}
	switch capability {
	case "source:public-symbol.documentation":
		return sourceCapabilityMentioned("source:documentation", input) && sourceCapabilityMentioned("source:visibility", input)
	case "architecture:relationships":
		return input.Architecture != nil
	case "architecture:cycles":
		return input.Architecture != nil
	default:
		return false
	}
}

func sourceCapabilityMentioned(capability string, input EvaluationInput) bool {
	if capability == "source:public-symbol.documentation" {
		return sourceCapabilityMentioned("source:documentation", input) || sourceCapabilityMentioned("source:visibility", input)
	}
	for _, snapshot := range input.SourceSnapshots {
		for _, descriptor := range snapshot.Capabilities {
			if descriptor.ID == capability {
				return true
			}
		}
		for _, coverage := range snapshot.Coverage {
			if coverage.Capability == capability {
				return true
			}
		}
		for _, metric := range snapshot.Metrics {
			if metric.MetricID == capability {
				return true
			}
		}
		if capability == "source:file.line_count" && (sourceCapabilityMentionedInSnapshot(snapshot, "source:size") || sourceCapabilityMentionedInSnapshot(snapshot, "source:files")) {
			return true
		}
		if strings.HasPrefix(capability, "source:callable.") && (sourceCapabilityMentionedInSnapshot(snapshot, "source:callable.metrics") || sourceCapabilityMentionedInSnapshot(snapshot, "source:declarations")) {
			return true
		}
	}
	return false
}

func sourceCapabilityMentionedInSnapshot(snapshot SourceSnapshot, capability string) bool {
	for _, descriptor := range snapshot.Capabilities {
		if descriptor.ID == capability {
			return true
		}
	}
	for _, coverage := range snapshot.Coverage {
		if coverage.Capability == capability {
			return true
		}
	}
	return false
}

func coverageForRule(rule QualityRule, status, reason string, subjects, evaluated int, input EvaluationInput, provider string) []QualityCoverage {
	if len(input.SourceSnapshots) == 0 {
		return []QualityCoverage{newCoverage(rule, status, reason, subjects, evaluated, "", "", provider)}
	}
	result := make([]QualityCoverage, 0, len(input.SourceSnapshots))
	for _, snapshot := range input.SourceSnapshots {
		result = append(result, newCoverage(rule, status, reason, subjects, evaluated, snapshot.ScopeID, snapshot.SnapshotID, provider))
	}
	return result
}

func setAvailableCapabilities(values []QualityCoverage, rule QualityRule, input EvaluationInput, batch MetricBatch) {
	for index := range values {
		if len(values[index].AvailableCapabilities) > 0 {
			values[index].AvailableCapabilities = normalizeStrings(values[index].AvailableCapabilities)
			continue
		}
		available := make([]string, 0, len(rule.RequiredCapabilities()))
		for _, capability := range rule.RequiredCapabilities() {
			if capabilityAvailable(capability, input, batch) {
				available = append(available, capability)
			}
		}
		values[index].AvailableCapabilities = normalizeStrings(available)
	}
}

func newCoverage(rule QualityRule, status, reason string, subjects, evaluated int, scopeID, snapshotID, provider string) QualityCoverage {
	return QualityCoverage{
		RuleID:                rule.ID(),
		RuleVersion:           rule.Version(),
		Status:                status,
		RequiredCapabilities:  normalizeStrings(rule.RequiredCapabilities()),
		AvailableCapabilities: []string{},
		SubjectCount:          intPointer(subjects),
		EvaluatedCount:        intPointer(evaluated),
		Reason:                reason,
		Provenance: FactProvenance{
			Status:          status,
			Basis:           "quality",
			EvidenceIDs:     []string{scopeEvidenceID(scopeID, snapshotID)},
			Provider:        provider,
			ProviderVersion: "1.0.0",
		},
	}
}

func normalizeEvaluation(report QualityEvaluation, profile QualityProfile, options map[string]any) QualityEvaluation {
	report.SchemaVersion = SchemaVersion
	report.SourceSnapshotIDs = normalizeStrings(report.SourceSnapshotIDs)
	report.ProviderIdentities = append([]ProviderIdentity(nil), report.ProviderIdentities...)
	if report.ProviderIdentities == nil {
		report.ProviderIdentities = []ProviderIdentity{}
	}
	sort.Slice(report.ProviderIdentities, func(i, j int) bool {
		return providerIdentityKey(report.ProviderIdentities[i]) < providerIdentityKey(report.ProviderIdentities[j])
	})
	if report.Coverage == nil {
		report.Coverage = []QualityCoverage{}
	}
	if report.Metrics == nil {
		report.Metrics = []MetricFact{}
	}
	if report.Findings == nil {
		report.Findings = []QualityFinding{}
	}
	if report.Diagnostics == nil {
		report.Diagnostics = []QualityDiagnostic{}
	}
	if report.Extensions == nil {
		report.Extensions = []ExtensionBlock{}
	}
	report.Extensions = normalizeExtensions(report.Extensions)
	for index := range report.Metrics {
		report.Metrics[index] = normalizeMetric(report.Metrics[index])
	}
	sort.Slice(report.Metrics, func(i, j int) bool { return metricKey(report.Metrics[i]) < metricKey(report.Metrics[j]) })
	for index := range report.Coverage {
		report.Coverage[index] = normalizeCoverage(report.Coverage[index])
	}
	report.Coverage = mergeCoverage(report.Coverage)
	for index := range report.Findings {
		finding := &report.Findings[index]
		if finding.AssessmentKind == "" {
			finding.AssessmentKind = AssessmentExact
		}
		if finding.Status == "" {
			finding.Status = StatusActive
		}
		if finding.Severity == "" {
			finding.Severity = defaultSeverity(finding.AssessmentKind)
		}
		if finding.RuleID == "" {
			finding.RuleID = "quality:unknown"
		}
		if finding.RuleVersion == "" {
			finding.RuleVersion = "1.0.0"
		}
		if finding.ID == "" {
			finding.ID = "finding:" + digestPart(finding.RuleID, finding.SubjectRef.ID, finding.MessageCode, finding.Message)
		}
		if finding.FindingKey == "" {
			finding.FindingKey = stableFindingKey(profile, *finding)
		}
		finding.ObservedMetricIDs = normalizeStrings(finding.ObservedMetricIDs)
		finding.Evidence = normalizeEvidence(finding.Evidence)
		finding.Limitations = normalizeStrings(finding.Limitations)
		finding.Extensions = normalizeExtensions(finding.Extensions)
		if finding.Provenance.Status == "" {
			finding.Provenance.Status = "observed"
		}
		if finding.Provenance.Basis == "" {
			finding.Provenance.Basis = "quality"
		}
		if finding.Provenance.Provider == "" {
			finding.Provenance.Provider = "quality:evaluator"
		}
		if finding.Provenance.ProviderVersion == "" {
			finding.Provenance.ProviderVersion = "1.0.0"
		}
		if finding.Provenance.EvidenceIDs == nil {
			finding.Provenance.EvidenceIDs = []string{}
		}
	}
	sort.Slice(report.Findings, func(i, j int) bool { return findingKey(report.Findings[i]) < findingKey(report.Findings[j]) })
	sort.Slice(report.Diagnostics, func(i, j int) bool {
		left, _ := json.Marshal(report.Diagnostics[i])
		right, _ := json.Marshal(report.Diagnostics[j])
		return string(left) < string(right)
	})
	for index := range report.Findings {
		report.Findings[index].ID = "finding:" + digestPart(report.Findings[index].FindingKey, report.Findings[index].RuleVersion)
	}
	semantic := report
	semantic.EvaluationID = ""
	semantic.EvaluationFingerprint = ContentDigest{}
	semantic.ReportDigest = ContentDigest{}
	fingerprint := digestJSON(struct {
		SourceSnapshotIDs []string            `json:"source_snapshot_ids"`
		Profile           QualityProfile      `json:"profile"`
		Providers         []ProviderIdentity  `json:"providers"`
		Metrics           []MetricFact        `json:"metrics"`
		Coverage          []QualityCoverage   `json:"coverage"`
		Findings          []QualityFinding    `json:"findings"`
		Diagnostics       []QualityDiagnostic `json:"diagnostics"`
		Options           map[string]any      `json:"options,omitempty"`
	}{report.SourceSnapshotIDs, profile, report.ProviderIdentities, report.Metrics, report.Coverage, report.Findings, report.Diagnostics, options})
	report.EvaluationFingerprint = fingerprint
	report.EvaluationID = "evaluation:" + digestPart(fingerprint.Value)
	semantic.EvaluationID = report.EvaluationID
	semantic.EvaluationFingerprint = report.EvaluationFingerprint
	report.ReportDigest = digestJSON(semantic)
	return report
}

func normalizeEvaluationInput(input EvaluationInput) EvaluationInput {
	input.SourceSnapshots = append([]SourceSnapshot(nil), input.SourceSnapshots...)
	sort.Slice(input.SourceSnapshots, func(i, j int) bool {
		if input.SourceSnapshots[i].ScopeID == input.SourceSnapshots[j].ScopeID {
			return input.SourceSnapshots[i].SnapshotID < input.SourceSnapshots[j].SnapshotID
		}
		return input.SourceSnapshots[i].ScopeID < input.SourceSnapshots[j].ScopeID
	})
	for index := range input.SourceSnapshots {
		snapshot := &input.SourceSnapshots[index]
		if snapshot.ScopeID == "" {
			snapshot.ScopeID = snapshot.SnapshotID
		}
		if snapshot.SnapshotID == "" {
			snapshot.SnapshotID = snapshot.ScopeID
		}
		snapshot.Capabilities = append([]CapabilityDescriptor(nil), snapshot.Capabilities...)
		snapshot.Coverage = append([]SourceCoverage(nil), snapshot.Coverage...)
		snapshot.Files = append([]SourceFile(nil), snapshot.Files...)
		snapshot.Symbols = append([]SourceSymbol(nil), snapshot.Symbols...)
		snapshot.Documentation = append([]SourceDocumentation(nil), snapshot.Documentation...)
		snapshot.Relations = append([]SourceRelation(nil), snapshot.Relations...)
		snapshot.Metrics = append([]MetricFact(nil), snapshot.Metrics...)
	}
	return input
}

func normalizeMetric(metric MetricFact) MetricFact {
	if metric.Extensions == nil {
		metric.Extensions = []ExtensionBlock{}
	}
	metric.Extensions = normalizeExtensions(metric.Extensions)
	if metric.Provenance.EvidenceIDs == nil {
		metric.Provenance.EvidenceIDs = []string{}
	}
	metric.Provenance.EvidenceIDs = normalizeStrings(metric.Provenance.EvidenceIDs)
	return metric
}

func normalizeCoverage(value QualityCoverage) QualityCoverage {
	value.RequiredCapabilities = normalizeStrings(value.RequiredCapabilities)
	value.AvailableCapabilities = normalizeStrings(value.AvailableCapabilities)
	if value.Provenance.EvidenceIDs == nil {
		value.Provenance.EvidenceIDs = []string{}
	}
	value.Provenance.EvidenceIDs = normalizeStrings(value.Provenance.EvidenceIDs)
	return value
}

func mergeCoverage(values []QualityCoverage) []QualityCoverage {
	sort.Slice(values, func(i, j int) bool { return coverageKey(values[i]) < coverageKey(values[j]) })
	result := make([]QualityCoverage, 0, len(values))
	for _, value := range values {
		if len(result) == 0 || coverageKey(result[len(result)-1]) != coverageKey(value) {
			result = append(result, value)
			continue
		}
		previous := &result[len(result)-1]
		previous.SubjectCount = sumPointers(previous.SubjectCount, value.SubjectCount)
		previous.EvaluatedCount = sumPointers(previous.EvaluatedCount, value.EvaluatedCount)
		previous.Status = combineCoverageStatus(previous.Status, value.Status)
		previous.Provenance.Status = previous.Status
		if previous.Reason == "" {
			previous.Reason = value.Reason
		}
		previous.Provenance.EvidenceIDs = normalizeStrings(append(previous.Provenance.EvidenceIDs, value.Provenance.EvidenceIDs...))
	}
	return result
}

func combineCoverageStatus(left, right string) string {
	if left == right {
		return left
	}
	if left == CoverageUnknown || right == CoverageUnknown {
		return CoverageUnknown
	}
	if left == CoverageUnsupported && right == CoverageUnsupported {
		return CoverageUnsupported
	}
	if left == CoverageNotEvaluable || right == CoverageNotEvaluable {
		return CoverageNotEvaluable
	}
	return CoveragePartial
}

func normalizeEvidence(value FindingEvidence) FindingEvidence {
	if value.SourceSpans == nil {
		value.SourceSpans = []SourceSpan{}
	}
	if value.EntityRefs == nil {
		value.EntityRefs = []EntityRef{}
	}
	if value.RelationRefs == nil {
		value.RelationRefs = []EntityRef{}
	}
	if value.MetricRefs == nil {
		value.MetricRefs = []string{}
	}
	if value.DiagnosticRefs == nil {
		value.DiagnosticRefs = []string{}
	}
	sort.Slice(value.SourceSpans, func(i, j int) bool { return spanKey(value.SourceSpans[i]) < spanKey(value.SourceSpans[j]) })
	sort.Slice(value.EntityRefs, func(i, j int) bool { return entityKey(value.EntityRefs[i]) < entityKey(value.EntityRefs[j]) })
	sort.Slice(value.RelationRefs, func(i, j int) bool { return entityKey(value.RelationRefs[i]) < entityKey(value.RelationRefs[j]) })
	value.MetricRefs = normalizeStrings(value.MetricRefs)
	value.DiagnosticRefs = normalizeStrings(value.DiagnosticRefs)
	return value
}

func normalizeExtensions(values []ExtensionBlock) []ExtensionBlock {
	if values == nil {
		return []ExtensionBlock{}
	}
	result := make([]ExtensionBlock, len(values))
	copy(result, values)
	sort.Slice(result, func(i, j int) bool {
		left := result[i].Namespace + "\x00" + result[i].SchemaVersion + "\x00" + result[i].Capability
		right := result[j].Namespace + "\x00" + result[j].SchemaVersion + "\x00" + result[j].Capability
		return left < right
	})
	return result
}

func snapshotIDs(values []SourceSnapshot) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value.SnapshotID != "" {
			result = append(result, value.SnapshotID)
		}
	}
	return normalizeStrings(result)
}

func providerIdentityKey(value ProviderIdentity) string { return value.ID + "\x00" + value.Version }
func metricKey(value MetricFact) string {
	return entityKey(value.SubjectRef) + "\x00" + value.MetricID + "\x00" + value.FormulaID + "\x00" + value.FormulaVersion + "\x00" + value.ID
}
func coverageKey(value QualityCoverage) string {
	return value.RuleID + "\x00" + value.RuleVersion + "\x00" + strings.Join(value.Provenance.EvidenceIDs, "\x00")
}
func findingKey(value QualityFinding) string { return value.FindingKey + "\x00" + value.ID }
func entityKey(value EntityRef) string {
	return value.ScopeID + "\x00" + value.SnapshotID + "\x00" + value.Kind + "\x00" + value.ID + "\x00" + value.StableKey
}
func spanKey(value SourceSpan) string {
	return fmt.Sprintf("%s\x00%012d\x00%012d", value.FileID, value.Start.ByteOffset, value.End.ByteOffset)
}
func scopeEvidenceID(scopeID, snapshotID string) string { return "scope:" + scopeID + ":" + snapshotID }

func stableFindingKey(profile QualityProfile, finding QualityFinding) string {
	subject := finding.SubjectRef.StableKey
	if subject == "" {
		subject = finding.SubjectRef.ID
	}
	profileFingerprint := digestJSON(profile).Value
	return strings.Join([]string{finding.RuleID, finding.RuleVersion, finding.AssessmentKind, profile.ProfileID, profile.ProfileVersion, profileFingerprint, finding.SubjectRef.ScopeID, finding.SubjectRef.Kind, subject, finding.MessageCode, findingOccurrenceAnchor(finding)}, "|")
}

func findingOccurrenceAnchor(finding QualityFinding) string {
	for _, extension := range finding.Extensions {
		if extension.Capability != "constraint-id" {
			continue
		}
		if value, ok := extension.Payload.(string); ok && value != "" {
			return "constraint:" + value
		}
	}
	return ""
}

func digestJSON(value any) ContentDigest {
	data, err := json.Marshal(value)
	if err != nil {
		return ContentDigest{Algorithm: "hash:sha-256"}
	}
	hash := sha256.Sum256(data)
	return ContentDigest{Algorithm: "hash:sha-256", Value: hex.EncodeToString(hash[:])}
}

func digestPart(values ...string) string {
	hash := sha256.Sum256([]byte(strings.Join(values, "\x00")))
	return hex.EncodeToString(hash[:12])
}

func intPointer(value int) *int { return &value }

func sumPointers(left, right *int) *int {
	if left == nil && right == nil {
		return nil
	}
	value := 0
	if left != nil {
		value += *left
	}
	if right != nil {
		value += *right
	}
	return &value
}

func uniqueStrings(values []string) bool {
	for index := 1; index < len(values); index++ {
		if values[index-1] == values[index] {
			return false
		}
	}
	return true
}

func validCoverageStatus(value string) bool {
	switch value {
	case CoverageObserved, CoverageAbsent, CoverageUnknown, CoverageUnsupported, CoveragePartial, CoverageNotEvaluable:
		return true
	default:
		return false
	}
}

func validFindingStatus(value string) bool {
	switch value {
	case StatusActive, StatusSuppressed, StatusBaseline, StatusResolved, StatusNotEvaluable:
		return true
	default:
		return false
	}
}
