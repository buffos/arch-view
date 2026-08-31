package quality

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// FindingTransition is one report-history change. Previous or Current is nil
// for a finding that was resolved or newly added, respectively. The stable
// finding key is deliberately the transition identity; report-local finding
// IDs are not.
type FindingTransition struct {
	FindingKey string          `json:"finding_key"`
	Previous   *QualityFinding `json:"previous,omitempty"`
	Current    *QualityFinding `json:"current,omitempty"`
}

// QualityReportComparison is a bounded, deterministic projection of two
// compatible reports. It preserves the complete finding values so consumers
// can explain why a transition occurred without re-evaluating source facts.
type QualityReportComparison struct {
	SchemaVersion        string              `json:"schema_version"`
	PreviousEvaluationID string              `json:"previous_evaluation_id"`
	CurrentEvaluationID  string              `json:"current_evaluation_id"`
	PreviousReportDigest ContentDigest       `json:"previous_report_digest"`
	CurrentReportDigest  ContentDigest       `json:"current_report_digest"`
	Added                []FindingTransition `json:"added"`
	Unchanged            []FindingTransition `json:"unchanged"`
	Suppressed           []FindingTransition `json:"suppressed"`
	Resolved             []FindingTransition `json:"resolved"`
}

// ValidateBaseline checks the identity and canonical ordering of a baseline
// before it is allowed to influence an evaluation. A baseline never contains
// source text or a finding payload; it only records exact semantic identities.
func ValidateBaseline(baseline Baseline) error {
	if baseline.SchemaVersion != BaselineSchemaVersion {
		return newQualityError(ErrorBaselineInvalid, "baseline schema version is unsupported", map[string]any{"schema_version": baseline.SchemaVersion})
	}
	if !validNamespacedID(baseline.BaselineID) || !strings.HasPrefix(baseline.BaselineID, "baseline:") {
		return newQualityError(ErrorBaselineInvalid, "baseline identity must be a namespaced baseline ID", map[string]any{"baseline_id": baseline.BaselineID})
	}
	if baseline.Revision != "" && !validVersion(baseline.Revision) {
		return newQualityError(ErrorBaselineInvalid, "baseline revision is invalid", map[string]any{"revision": baseline.Revision})
	}
	if baseline.Entries == nil || baseline.Extensions == nil {
		return newQualityError(ErrorBaselineInvalid, "baseline collections must be serialized as arrays", nil)
	}
	if err := validateQualityExtensions(baseline.Extensions); err != nil {
		return WrapQualityError(ErrorBaselineInvalid, "baseline extensions are invalid", err, nil)
	}
	seen := make(map[string]struct{}, len(baseline.Entries))
	for _, entry := range baseline.Entries {
		if strings.TrimSpace(entry.FindingKey) == "" || !validNamespacedID(entry.RuleID) || !validVersion(entry.RuleVersion) || !validProfileID(entry.ProfileID) || !validVersion(entry.ProfileVersion) || strings.TrimSpace(entry.Reason) == "" {
			return newQualityError(ErrorBaselineInvalid, "baseline entry identity, profile, or reason is incomplete", map[string]any{"finding_key": entry.FindingKey})
		}
		key := baselineEntryKey(entry)
		if _, exists := seen[key]; exists {
			return newQualityError(ErrorBaselineInvalid, "baseline entries must be unique by finding and exact versions", map[string]any{"finding_key": entry.FindingKey})
		}
		seen[key] = struct{}{}
		if entry.FormulaVersions == nil {
			return newQualityError(ErrorBaselineInvalid, "baseline formula_versions must be serialized as an array", map[string]any{"finding_key": entry.FindingKey})
		}
		if !sort.SliceIsSorted(entry.FormulaVersions, func(i, j int) bool {
			return formulaVersionKey(entry.FormulaVersions[i]) < formulaVersionKey(entry.FormulaVersions[j])
		}) {
			return newQualityError(ErrorBaselineInvalid, "baseline formula versions must be canonically ordered", map[string]any{"finding_key": entry.FindingKey})
		}
		for index, formula := range entry.FormulaVersions {
			if !validNamespacedID(formula.MetricID) || !validVersion(formula.Version) || index > 0 && formulaVersionKey(entry.FormulaVersions[index-1]) == formulaVersionKey(formula) {
				return newQualityError(ErrorBaselineInvalid, "baseline formula version identity is invalid or duplicated", map[string]any{"finding_key": entry.FindingKey, "metric_id": formula.MetricID})
			}
		}
	}
	if !sort.SliceIsSorted(baseline.Entries, func(i, j int) bool {
		return baselineEntryKey(baseline.Entries[i]) < baselineEntryKey(baseline.Entries[j])
	}) {
		return newQualityError(ErrorBaselineInvalid, "baseline entries must be canonically ordered", nil)
	}
	return nil
}

// CreateBaselineEntry creates an exact-version entry for a finding in a
// validated report. The caller decides whether to append it to a baseline.
func CreateBaselineEntry(report QualityEvaluation, findingID, reason, owner string) (BaselineEntry, error) {
	if err := ValidateQualityEvaluation(report); err != nil {
		return BaselineEntry{}, err
	}
	findingID = strings.TrimSpace(findingID)
	reason = strings.TrimSpace(reason)
	if findingID == "" || reason == "" {
		return BaselineEntry{}, newQualityError(ErrorBaselineInvalid, "finding ID/key and baseline reason are required", nil)
	}
	var selected *QualityFinding
	for index := range report.Findings {
		if report.Findings[index].ID == findingID || report.Findings[index].FindingKey == findingID {
			selected = &report.Findings[index]
			break
		}
	}
	if selected == nil {
		return BaselineEntry{}, newQualityError(ErrorBaselineInvalid, "the report does not contain the requested finding", map[string]any{"finding_id": findingID})
	}
	if selected.Status != StatusActive {
		return BaselineEntry{}, newQualityError(ErrorBaselineInvalid, "only active findings may be added to a baseline", map[string]any{"finding_id": findingID, "status": selected.Status})
	}
	coverageFound := false
	for _, coverage := range report.Coverage {
		if coverage.RuleID != selected.RuleID || coverage.RuleVersion != selected.RuleVersion {
			continue
		}
		coverageFound = true
		if coverage.Status != CoverageObserved {
			return BaselineEntry{}, newQualityError(ErrorBaselineInvalid, "baseline entries require observed rule coverage", map[string]any{"finding_id": findingID, "rule_id": selected.RuleID, "rule_version": selected.RuleVersion, "coverage": coverage.Status})
		}
	}
	if !coverageFound {
		return BaselineEntry{}, newQualityError(ErrorBaselineInvalid, "baseline entries require observed rule coverage", map[string]any{"finding_id": findingID, "rule_id": selected.RuleID, "rule_version": selected.RuleVersion})
	}
	return BaselineEntry{
		FindingKey:      selected.FindingKey,
		RuleID:          selected.RuleID,
		RuleVersion:     selected.RuleVersion,
		ProfileID:       report.ProfileID,
		ProfileVersion:  report.ProfileVersion,
		FormulaVersions: formulaVersionsForFinding(*selected, report.Metrics),
		Reason:          reason,
		Owner:           strings.TrimSpace(owner),
	}, nil
}

// AddBaselineEntry returns a canonical copy of baseline with entry appended.
func AddBaselineEntry(baseline Baseline, entry BaselineEntry) (Baseline, error) {
	if err := ValidateBaseline(baseline); err != nil {
		return Baseline{}, err
	}
	baseline.Entries = append(append([]BaselineEntry(nil), baseline.Entries...), entry)
	sort.Slice(baseline.Entries, func(i, j int) bool {
		return baselineEntryKey(baseline.Entries[i]) < baselineEntryKey(baseline.Entries[j])
	})
	if err := ValidateBaseline(baseline); err != nil {
		return Baseline{}, err
	}
	return baseline, nil
}

// BaselineMergeResult describes an idempotent append operation. Entries that
// already exist with the same exact identity are reported as existing rather
// than duplicated. A matching identity with different review metadata is an
// error because silently changing the reason would hide a policy decision.
type BaselineMergeResult struct {
	Baseline Baseline
	Added    []BaselineEntry
	Existing []BaselineEntry
}

// MergeBaselineEntries appends reviewed entries to a baseline while keeping
// canonical ordering and exact-version identity. It never mutates the input
// baseline or entry slices.
func MergeBaselineEntries(baseline Baseline, additions []BaselineEntry) (BaselineMergeResult, error) {
	if err := ValidateBaseline(baseline); err != nil {
		return BaselineMergeResult{}, err
	}
	result := BaselineMergeResult{Baseline: baseline, Added: []BaselineEntry{}, Existing: []BaselineEntry{}}
	result.Baseline.Entries = append([]BaselineEntry(nil), baseline.Entries...)
	byIdentity := make(map[string]BaselineEntry, len(baseline.Entries))
	for _, entry := range baseline.Entries {
		byIdentity[baselineEntryKey(entry)] = entry
	}
	seenAdditions := make(map[string]struct{}, len(additions))
	for _, entry := range additions {
		if err := validateBaselineEntry(entry); err != nil {
			return BaselineMergeResult{}, err
		}
		identity := baselineEntryKey(entry)
		if _, duplicate := seenAdditions[identity]; duplicate {
			return BaselineMergeResult{}, newQualityError(ErrorBaselineInvalid, "baseline additions must be unique by finding and exact versions", map[string]any{"finding_key": entry.FindingKey})
		}
		seenAdditions[identity] = struct{}{}
		if existing, ok := byIdentity[identity]; ok {
			if existing.Reason != entry.Reason || existing.Owner != entry.Owner {
				return BaselineMergeResult{}, newQualityError(ErrorBaselineInvalid, "baseline entry already exists with different review metadata", map[string]any{"finding_key": entry.FindingKey})
			}
			result.Existing = append(result.Existing, existing)
			continue
		}
		byIdentity[identity] = entry
		result.Baseline.Entries = append(result.Baseline.Entries, entry)
		result.Added = append(result.Added, entry)
	}
	sort.Slice(result.Baseline.Entries, func(i, j int) bool {
		return baselineEntryKey(result.Baseline.Entries[i]) < baselineEntryKey(result.Baseline.Entries[j])
	})
	if err := ValidateBaseline(result.Baseline); err != nil {
		return BaselineMergeResult{}, err
	}
	return result, nil
}

func validateBaselineEntry(entry BaselineEntry) error {
	probe := Baseline{
		SchemaVersion: BaselineSchemaVersion,
		BaselineID:    "baseline:entry-validation",
		Entries:       []BaselineEntry{entry},
		Extensions:    []ExtensionBlock{},
	}
	return ValidateBaseline(probe)
}

// NextBaselineRevision returns a deterministic next revision for a managed
// baseline. Numeric trailing components are incremented, so 1.0.0 becomes
// 1.0.1 and 2026-08 becomes 2026-08.1. A new baseline starts at 1.0.0.
func NextBaselineRevision(current string) (string, error) {
	current = strings.TrimSpace(current)
	if current == "" {
		return "1.0.0", nil
	}
	if !validVersion(current) {
		return "", newQualityError(ErrorBaselineInvalid, "current baseline revision is invalid", map[string]any{"revision": current})
	}
	separator := strings.LastIndexByte(current, '.')
	if separator < 0 {
		if value, err := strconv.ParseUint(current, 10, 64); err == nil {
			if value == ^uint64(0) {
				return "", newQualityError(ErrorBaselineInvalid, "baseline revision cannot be incremented", map[string]any{"revision": current})
			}
			return strconv.FormatUint(value+1, 10), nil
		}
	}
	prefix, suffix := current, ""
	if separator >= 0 {
		prefix, suffix = current[:separator], current[separator+1:]
	}
	if suffix != "" {
		if value, err := strconv.ParseUint(suffix, 10, 64); err == nil {
			if value == ^uint64(0) {
				return "", newQualityError(ErrorBaselineInvalid, "baseline revision cannot be incremented", map[string]any{"revision": current})
			}
			return prefix + "." + strconv.FormatUint(value+1, 10), nil
		}
	}
	return current + ".1", nil
}

// ResolveSuppression returns a finding with its detected payload intact and a
// suppression projection applied only when every baseline version matches.
func ResolveSuppression(finding QualityFinding, report QualityEvaluation, baseline *Baseline) (QualityFinding, bool) {
	if baseline == nil || finding.FindingKey == "" || report.ProfileID == "" || report.ProfileVersion == "" {
		return finding, false
	}
	formulaVersions := formulaVersionsForFinding(finding, report.Metrics)
	for _, entry := range baseline.Entries {
		if entry.FindingKey != finding.FindingKey || entry.RuleID != finding.RuleID || entry.RuleVersion != finding.RuleVersion || entry.ProfileID != report.ProfileID || entry.ProfileVersion != report.ProfileVersion || !sameFormulaVersions(entry.FormulaVersions, formulaVersions) {
			continue
		}
		finding.Status = StatusSuppressed
		finding.Suppression = &SuppressionInfo{BaselineID: baseline.BaselineID, Reason: entry.Reason, ExactVersionMatch: true}
		return finding, true
	}
	return finding, false
}

func applyBaseline(report *QualityEvaluation, profile QualityProfile, baseline Baseline) error {
	if report == nil {
		return newQualityError(ErrorBaselineInvalid, "quality report is required for baseline application", nil)
	}
	if err := ValidateBaseline(baseline); err != nil {
		return err
	}
	if profile.Baseline == nil || profile.Baseline.BaselineID != baseline.BaselineID || profile.Baseline.Revision != "" && profile.Baseline.Revision != baseline.Revision {
		return newQualityError(ErrorBaselineInvalid, "baseline does not match the profile reference", map[string]any{"baseline_id": baseline.BaselineID})
	}
	for index := range report.Findings {
		if report.Findings[index].FindingKey == "" {
			report.Findings[index].FindingKey = stableFindingKey(profile, report.Findings[index], report.Metrics)
		}
		resolved, matched := ResolveSuppression(report.Findings[index], *report, &baseline)
		if matched {
			report.Findings[index] = resolved
		}
	}
	return nil
}

// CompareQualityReports compares two immutable reports that share the same
// profile, rule, provider, and formula semantics. Source snapshot identities
// may differ because a report can represent a later revision.
func CompareQualityReports(previous, current QualityEvaluation) (QualityReportComparison, error) {
	if err := ValidateQualityEvaluation(previous); err != nil {
		return QualityReportComparison{}, err
	}
	if err := ValidateQualityEvaluation(current); err != nil {
		return QualityReportComparison{}, err
	}
	if !compatibleReportSemantics(previous, current) {
		return QualityReportComparison{}, newQualityError(ErrorReportIncompatible, "quality reports use incompatible profile, provider, rule, or formula versions", map[string]any{"previous_profile": previous.ProfileID + "@" + previous.ProfileVersion, "current_profile": current.ProfileID + "@" + current.ProfileVersion})
	}
	comparison := QualityReportComparison{
		SchemaVersion:        SchemaVersion,
		PreviousEvaluationID: previous.EvaluationID,
		CurrentEvaluationID:  current.EvaluationID,
		PreviousReportDigest: previous.ReportDigest,
		CurrentReportDigest:  current.ReportDigest,
		Added:                []FindingTransition{},
		Unchanged:            []FindingTransition{},
		Suppressed:           []FindingTransition{},
		Resolved:             []FindingTransition{},
	}
	previousByKey := findingByKey(previous.Findings)
	currentByKey := findingByKey(current.Findings)
	for key, finding := range currentByKey {
		previousFinding, existed := previousByKey[key]
		transition := FindingTransition{FindingKey: key, Current: cloneFindingPointer(finding)}
		if existed {
			transition.Previous = cloneFindingPointer(previousFinding)
		}
		if isSuppressedStatus(finding.Status) {
			comparison.Suppressed = append(comparison.Suppressed, transition)
		} else if !existed || isSuppressedStatus(previousFinding.Status) {
			comparison.Added = append(comparison.Added, transition)
		} else if finding.Status == StatusResolved {
			comparison.Resolved = append(comparison.Resolved, transition)
		} else {
			comparison.Unchanged = append(comparison.Unchanged, transition)
		}
	}
	for key, finding := range previousByKey {
		if _, exists := currentByKey[key]; exists {
			continue
		}
		// A missing finding is a resolution only when the current report has
		// observed coverage for that exact rule. Partial, unknown,
		// unsupported, absent, or not-evaluable coverage cannot prove that a
		// previous finding disappeared.
		if !ruleCanResolve(current, finding.RuleID, finding.RuleVersion) {
			continue
		}
		comparison.Resolved = append(comparison.Resolved, FindingTransition{FindingKey: key, Previous: cloneFindingPointer(finding)})
	}
	sortTransitions(comparison.Added)
	sortTransitions(comparison.Unchanged)
	sortTransitions(comparison.Suppressed)
	sortTransitions(comparison.Resolved)
	return comparison, nil
}

func compatibleReportSemantics(previous, current QualityEvaluation) bool {
	if previous.SchemaVersion != current.SchemaVersion || previous.ProfileID != current.ProfileID || previous.ProfileVersion != current.ProfileVersion || previous.Baseline == nil != (current.Baseline == nil) {
		return false
	}
	if previous.Baseline != nil && (*previous.Baseline != *current.Baseline) {
		return false
	}
	if !sameProviderIdentities(previous.ProviderIdentities, current.ProviderIdentities) || !sameStringSet(ruleIdentitySet(previous), ruleIdentitySet(current)) || !sameStringSet(formulaIdentitySet(previous), formulaIdentitySet(current)) {
		return false
	}
	return true
}

func findingByKey(values []QualityFinding) map[string]QualityFinding {
	result := make(map[string]QualityFinding, len(values))
	for _, finding := range values {
		result[finding.FindingKey] = finding
	}
	return result
}

func ruleCanResolve(report QualityEvaluation, ruleID, version string) bool {
	found := false
	for _, coverage := range report.Coverage {
		if coverage.RuleID != ruleID || coverage.RuleVersion != version {
			continue
		}
		found = true
		if coverage.Status != CoverageObserved {
			return false
		}
	}
	return found
}

func isSuppressedStatus(status string) bool {
	return status == StatusSuppressed || status == StatusBaseline
}

func sortTransitions(values []FindingTransition) {
	sort.Slice(values, func(i, j int) bool { return values[i].FindingKey < values[j].FindingKey })
}

func sameProviderIdentities(left, right []ProviderIdentity) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func ruleIdentitySet(report QualityEvaluation) []string {
	values := make([]string, 0, len(report.Coverage)+len(report.Findings))
	for _, coverage := range report.Coverage {
		values = append(values, coverage.RuleID+"\x00"+coverage.RuleVersion)
	}
	for _, finding := range report.Findings {
		values = append(values, finding.RuleID+"\x00"+finding.RuleVersion)
	}
	return uniqueSorted(values)
}

func formulaIdentitySet(report QualityEvaluation) []string {
	values := make([]string, 0, len(report.Metrics))
	for _, metric := range report.Metrics {
		values = append(values, metric.MetricID+"\x00"+metric.FormulaVersion)
	}
	return uniqueSorted(values)
}

func sameStringSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func uniqueSorted(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	sort.Strings(values)
	result := values[:0]
	for _, value := range values {
		if len(result) == 0 || result[len(result)-1] != value {
			result = append(result, value)
		}
	}
	return result
}

func cloneFindingPointer(value QualityFinding) *QualityFinding {
	copy := value
	return &copy
}

func cloneBaselineRef(value *BaselineRef) *BaselineRef {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func baselineEntryKey(value BaselineEntry) string {
	return value.FindingKey + "\x00" + value.RuleID + "\x00" + value.RuleVersion + "\x00" + value.ProfileID + "\x00" + value.ProfileVersion + "\x00" + formulaVersionsKey(value.FormulaVersions)
}

func formulaVersionsForFinding(finding QualityFinding, metrics []MetricFact) []FormulaVersion {
	metricIDs := make(map[string]struct{}, len(finding.ObservedMetricIDs)+len(finding.Evidence.MetricRefs))
	for _, id := range finding.ObservedMetricIDs {
		metricIDs[id] = struct{}{}
	}
	for _, id := range finding.Evidence.MetricRefs {
		metricIDs[id] = struct{}{}
	}
	result := make([]FormulaVersion, 0, len(metricIDs))
	seen := make(map[string]struct{}, len(metricIDs))
	for _, metric := range metrics {
		if _, ok := metricIDs[metric.ID]; !ok || metric.FormulaVersion == "" {
			continue
		}
		value := FormulaVersion{MetricID: metric.MetricID, Version: metric.FormulaVersion}
		key := formulaVersionKey(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return formulaVersionKey(result[i]) < formulaVersionKey(result[j]) })
	if result == nil {
		return []FormulaVersion{}
	}
	return result
}

func sameFormulaVersions(left, right []FormulaVersion) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func formulaVersionKey(value FormulaVersion) string { return value.MetricID + "\x00" + value.Version }
func formulaVersionsKey(values []FormulaVersion) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, formulaVersionKey(value))
	}
	return strings.Join(parts, "\x00")
}

// WrapQualityError keeps lifecycle failures in the same structured error
// shape as profile and report validation without exposing implementation
// details to transport adapters.
func WrapQualityError(code ErrorCode, message string, cause error, details map[string]any) error {
	if details == nil {
		details = map[string]any{}
	}
	if cause != nil {
		details["cause"] = cause.Error()
		message = fmt.Sprintf("%s: %v", message, cause)
	}
	return newQualityError(code, message, details)
}
