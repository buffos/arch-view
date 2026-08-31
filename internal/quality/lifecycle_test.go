package quality_test

import (
	"errors"
	"testing"

	"github.com/buffo/arch-view/internal/quality"
)

func TestFindingKeySurvivesSourceLineShiftButReportIDChanges(t *testing.T) {
	profile := thresholdProfile("source:callable.max-lines", "rule-config:source-callable-size", "greater_than", 2, "unit:line")
	first := callableLifecycleInput(10, 12)
	second := callableLifecycleInput(20, 22)
	firstReport, err := quality.EvaluateQualityProfile(profile, first, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("first evaluation: %v", err)
	}
	secondReport, err := quality.EvaluateQualityProfile(profile, second, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("second evaluation: %v", err)
	}
	if len(firstReport.Findings) != 1 || len(secondReport.Findings) != 1 {
		t.Fatalf("finding counts = %d/%d, want one each", len(firstReport.Findings), len(secondReport.Findings))
	}
	if firstReport.Findings[0].FindingKey != secondReport.Findings[0].FindingKey {
		t.Fatalf("finding keys changed with line shift: %q != %q", firstReport.Findings[0].FindingKey, secondReport.Findings[0].FindingKey)
	}
	if firstReport.Findings[0].ID == secondReport.Findings[0].ID || firstReport.EvaluationID == secondReport.EvaluationID {
		t.Fatal("report-local IDs or evaluation identity did not change after evidence moved")
	}
}

func TestBaselineSuppressesExactFindingWithoutDeletingIt(t *testing.T) {
	profile := thresholdProfile("source:file.max-lines", "rule-config:source-file-size", "greater_than", 10, "unit:line")
	input := fileLifecycleInput()
	initial, err := quality.EvaluateQualityProfile(profile, input, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("initial evaluation: %v", err)
	}
	entry, err := quality.CreateBaselineEntry(initial, initial.Findings[0].ID, "legacy fixture is accepted for now", "maintainer")
	if err != nil {
		t.Fatalf("create baseline entry: %v", err)
	}
	baseline := quality.Baseline{SchemaVersion: quality.BaselineSchemaVersion, BaselineID: "baseline:fixture", Revision: "1.0.0", Entries: []quality.BaselineEntry{}, Extensions: []quality.ExtensionBlock{}}
	baseline, err = quality.AddBaselineEntry(baseline, entry)
	if err != nil {
		t.Fatalf("add baseline entry: %v", err)
	}
	profile.Baseline = &quality.BaselineRef{BaselineID: baseline.BaselineID, Revision: baseline.Revision}
	input.Baseline = &baseline
	suppressed, err := quality.EvaluateQualityProfile(profile, input, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("baseline evaluation: %v", err)
	}
	if len(suppressed.Findings) != 1 || suppressed.Findings[0].Status != quality.StatusSuppressed || suppressed.Findings[0].Suppression == nil || suppressed.Findings[0].Suppression.Reason != entry.Reason {
		t.Fatalf("suppressed report = %#v, want retained finding and reason", suppressed)
	}
	if suppressed.Findings[0].FindingKey != initial.Findings[0].FindingKey {
		t.Fatal("baseline reference changed the stable finding key")
	}

	changed := baseline
	changed.Entries = append([]quality.BaselineEntry(nil), baseline.Entries...)
	changed.Entries[0].RuleVersion = "2.0.0"
	changed.Entries[0].FormulaVersions = []quality.FormulaVersion{}
	input.Baseline = &changed
	// The profile still references the original exact rule version. The
	// changed baseline entry must remain visible rather than suppressing it.
	unsuppressed, err := quality.EvaluateQualityProfile(profile, input, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("changed baseline evaluation: %v", err)
	}
	if unsuppressed.Findings[0].Status != quality.StatusActive {
		t.Fatalf("changed-version baseline status = %q, want active", unsuppressed.Findings[0].Status)
	}
	if _, err := quality.AddBaselineEntry(baseline, entry); err == nil {
		t.Fatal("duplicate baseline entry unexpectedly accepted")
	}
}

func TestMergeBaselineEntriesIsIdempotentAndRejectsMetadataConflicts(t *testing.T) {
	profile := thresholdProfile("source:file.max-lines", "rule-config:source-file-size", "greater_than", 10, "unit:line")
	report, err := quality.EvaluateQualityProfile(profile, fileLifecycleInput(), quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("evaluate report: %v", err)
	}
	entry, err := quality.CreateBaselineEntry(report, report.Findings[0].ID, "accepted for this legacy fixture", "maintainer")
	if err != nil {
		t.Fatalf("create baseline entry: %v", err)
	}
	baseline := quality.Baseline{SchemaVersion: quality.BaselineSchemaVersion, BaselineID: "baseline:merge", Revision: "1.0.0", Entries: []quality.BaselineEntry{}, Extensions: []quality.ExtensionBlock{}}

	first, err := quality.MergeBaselineEntries(baseline, []quality.BaselineEntry{entry})
	if err != nil {
		t.Fatalf("first merge: %v", err)
	}
	if len(first.Added) != 1 || len(first.Existing) != 0 || len(first.Baseline.Entries) != 1 {
		t.Fatalf("first merge = %#v", first)
	}
	second, err := quality.MergeBaselineEntries(first.Baseline, []quality.BaselineEntry{entry})
	if err != nil {
		t.Fatalf("idempotent merge: %v", err)
	}
	if len(second.Added) != 0 || len(second.Existing) != 1 || len(second.Baseline.Entries) != 1 {
		t.Fatalf("idempotent merge = %#v", second)
	}
	conflict := entry
	conflict.Reason = "a different decision"
	if _, err := quality.MergeBaselineEntries(first.Baseline, []quality.BaselineEntry{conflict}); err == nil {
		t.Fatal("conflicting duplicate reason was accepted")
	}
}

func TestNextBaselineRevisionUsesDeterministicNumericSuffixes(t *testing.T) {
	tests := map[string]string{"": "1.0.0", "1.0.0": "1.0.1", "2026-08": "2026-08.1", "1": "2"}
	for current, want := range tests {
		got, err := quality.NextBaselineRevision(current)
		if err != nil {
			t.Fatalf("next revision for %q: %v", current, err)
		}
		if got != want {
			t.Fatalf("next revision for %q = %q, want %q", current, got, want)
		}
	}
	if _, err := quality.NextBaselineRevision("not a revision"); err == nil {
		t.Fatal("invalid baseline revision was accepted")
	}
}

func TestCompareQualityReportsDoesNotResolveFromIncompleteCoverage(t *testing.T) {
	profile := thresholdProfile("source:file.max-lines", "rule-config:source-file-size", "greater_than", 10, "unit:line")
	previous, err := quality.EvaluateQualityProfile(profile, fileLifecycleInput(), quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("previous evaluation: %v", err)
	}

	resolved := previous
	resolved.Findings = []quality.QualityFinding{}
	resolved, err = quality.NormalizeQualityReport(resolved)
	if err != nil {
		t.Fatalf("normalize observed empty report: %v", err)
	}
	comparison, err := quality.CompareQualityReports(previous, resolved)
	if err != nil {
		t.Fatalf("compare observed empty report: %v", err)
	}
	if len(comparison.Resolved) != 1 {
		t.Fatalf("resolved transitions = %#v, want one", comparison.Resolved)
	}

	incomplete := previous
	incomplete.Findings = []quality.QualityFinding{}
	incomplete.Coverage = append([]quality.QualityCoverage(nil), previous.Coverage...)
	incomplete.Coverage[0].Status = quality.CoveragePartial
	incomplete.Coverage[0].Provenance.Status = quality.CoveragePartial
	incomplete, err = quality.NormalizeQualityReport(incomplete)
	if err != nil {
		t.Fatalf("normalize incomplete report: %v", err)
	}
	comparison, err = quality.CompareQualityReports(previous, incomplete)
	if err != nil {
		t.Fatalf("compare incomplete report: %v", err)
	}
	if len(comparison.Resolved) != 0 {
		t.Fatalf("incomplete coverage resolved transitions = %#v, want none", comparison.Resolved)
	}
}

func TestCompareQualityReportsClassifiesSuppression(t *testing.T) {
	profile := thresholdProfile("source:file.max-lines", "rule-config:source-file-size", "greater_than", 10, "unit:line")
	baseInput := fileLifecycleInput()
	active, err := quality.EvaluateQualityProfile(profile, baseInput, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("active evaluation: %v", err)
	}
	entry, err := quality.CreateBaselineEntry(active, active.Findings[0].ID, "accepted", "owner")
	if err != nil {
		t.Fatalf("baseline entry: %v", err)
	}
	baseline := quality.Baseline{SchemaVersion: quality.BaselineSchemaVersion, BaselineID: "baseline:compare", Revision: "1.0.0", Entries: []quality.BaselineEntry{}, Extensions: []quality.ExtensionBlock{}}
	baseline, err = quality.AddBaselineEntry(baseline, entry)
	if err != nil {
		t.Fatalf("baseline: %v", err)
	}
	profile.Baseline = &quality.BaselineRef{BaselineID: baseline.BaselineID, Revision: baseline.Revision}
	baseInput.Baseline = &baseline
	suppressed, err := quality.EvaluateQualityProfile(profile, baseInput, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("suppressed evaluation: %v", err)
	}
	activeWithBaseline := suppressed
	activeWithBaseline.Findings = append([]quality.QualityFinding(nil), suppressed.Findings...)
	activeWithBaseline.Findings[0].Status = quality.StatusActive
	activeWithBaseline.Findings[0].Suppression = nil
	activeWithBaseline, err = quality.NormalizeQualityReport(activeWithBaseline)
	if err != nil {
		t.Fatalf("normalize active baseline report: %v", err)
	}
	comparison, err := quality.CompareQualityReports(activeWithBaseline, suppressed)
	if err != nil {
		t.Fatalf("compare suppression: %v", err)
	}
	if len(comparison.Suppressed) != 1 || len(comparison.Added) != 0 {
		t.Fatalf("suppression comparison = %#v, want one suppressed transition", comparison)
	}
}

func TestCompareQualityReportsClassifiesAddedAndUnchangedFindings(t *testing.T) {
	profile := thresholdProfile("source:file.max-lines", "rule-config:source-file-size", "greater_than", 10, "unit:line")
	previous, err := quality.EvaluateQualityProfile(profile, fileLifecycleInput(), quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("previous evaluation: %v", err)
	}
	currentInput := fileLifecycleInput()
	currentInput.SourceSnapshots[0].Files = append(currentInput.SourceSnapshots[0].Files, quality.SourceFile{ID: "file:second", StableKey: "internal/second.go", Path: "internal/second.go", LineCount: 12, ByteCount: 130})
	current, err := quality.EvaluateQualityProfile(profile, currentInput, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("current evaluation: %v", err)
	}
	comparison, err := quality.CompareQualityReports(previous, current)
	if err != nil {
		t.Fatalf("compare reports: %v", err)
	}
	if len(comparison.Added) != 1 || comparison.Added[0].Current == nil || comparison.Added[0].Current.SubjectRef.ID != "file:second" {
		t.Fatalf("added transitions = %#v, want the second file", comparison.Added)
	}
	if len(comparison.Unchanged) != 1 || comparison.Unchanged[0].Current == nil || comparison.Unchanged[0].Current.SubjectRef.ID != "file:large" {
		t.Fatalf("unchanged transitions = %#v, want the original file", comparison.Unchanged)
	}
}

func TestProviderFailurePreservesUnrelatedFindings(t *testing.T) {
	catalog := quality.NewDefaultCatalog()
	if err := catalog.RegisterMetricProvider(failingMetricProvider{}); err != nil {
		t.Fatalf("register failing provider: %v", err)
	}
	if err := catalog.RegisterQualityRule(failingRule{}); err != nil {
		t.Fatalf("register failing rule: %v", err)
	}
	profile := thresholdProfile("source:file.max-lines", "rule-config:source-file-size", "greater_than", 10, "unit:line")
	profile.EnabledRules = append(profile.EnabledRules, quality.RuleBinding{RuleID: "rule:failing", RuleVersion: "1.0.0", Enabled: true})
	report, err := quality.EvaluateQualityProfile(profile, fileLifecycleInput(), catalog)
	if err != nil {
		t.Fatalf("evaluate with failing provider: %v", err)
	}
	if len(report.Findings) != 1 || report.Findings[0].RuleID != "source:file.max-lines" {
		t.Fatalf("unrelated findings = %#v, want file finding retained", report.Findings)
	}
	foundPartial := false
	for _, coverage := range report.Coverage {
		if coverage.RuleID == "rule:failing" && coverage.Status == quality.CoveragePartial {
			foundPartial = true
		}
	}
	if !foundPartial {
		t.Fatalf("failure coverage = %#v, want partial", report.Coverage)
	}
	if len(report.Diagnostics) == 0 {
		t.Fatal("provider failure did not produce a diagnostic")
	}
}

func callableLifecycleInput(startLine, endLine int) quality.EvaluationInput {
	hash := quality.ContentDigest{Algorithm: "hash:sha-256", Value: "0000000000000000000000000000000000000000000000000000000000000000"}
	return quality.EvaluationInput{SourceSnapshots: []quality.SourceSnapshot{{SnapshotID: "snapshot-lifecycle", ScopeID: "scope-lifecycle", Capabilities: []quality.CapabilityDescriptor{{ID: "source:declarations", Version: "1.0.0"}}, Coverage: []quality.SourceCoverage{{Capability: "source:callable.metrics", SubjectKind: "symbol", Status: quality.CoverageObserved}}, Files: []quality.SourceFile{{ID: "file:lifecycle", Path: "internal/lifecycle.go", ByteCount: 1000, ContentHash: hash}}, Symbols: []quality.SourceSymbol{{ID: "symbol:decode", StableKey: "internal/lifecycle.go#Decode", Name: "Decode", Category: "callable", BodySpan: &quality.SourceSpan{FileID: "file:lifecycle", Start: quality.SpanPosition{ByteOffset: startLine * 10, Line: startLine, Column: 1}, End: quality.SpanPosition{ByteOffset: endLine * 10, Line: endLine, Column: 2}, CoordinateSystem: "utf8-byte", ContentHash: hash}}}}}}
}

func fileLifecycleInput() quality.EvaluationInput {
	return quality.EvaluationInput{SourceSnapshots: []quality.SourceSnapshot{{SnapshotID: "snapshot-file-lifecycle", ScopeID: "scope-file-lifecycle", Capabilities: []quality.CapabilityDescriptor{{ID: "source:size", Version: "1.0.0"}}, Coverage: []quality.SourceCoverage{{Capability: "source:size", SubjectKind: "file", Status: quality.CoverageObserved}}, Files: []quality.SourceFile{{ID: "file:large", StableKey: "internal/large.go", Path: "internal/large.go", LineCount: 11, ByteCount: 120}}}}}
}

type failingMetricProvider struct{}

func (failingMetricProvider) ID() string      { return "provider:failing" }
func (failingMetricProvider) Version() string { return "1.0.0" }
func (failingMetricProvider) Capabilities() []quality.CapabilityDescriptor {
	return []quality.CapabilityDescriptor{{ID: "metric:failing", Version: "1.0.0"}}
}
func (failingMetricProvider) Compute(quality.EvaluationContext) (quality.MetricBatch, error) {
	return quality.MetricBatch{}, errors.New("fixture provider failed")
}

type failingRule struct{}

func (failingRule) ID() string                     { return "rule:failing" }
func (failingRule) Version() string                { return "1.0.0" }
func (failingRule) AssessmentKind() string         { return quality.AssessmentExact }
func (failingRule) RequiredCapabilities() []string { return []string{"metric:failing"} }
func (failingRule) Evaluate(quality.EvaluationContext, quality.MetricBatch) (quality.RuleResult, error) {
	return quality.RuleResult{}, nil
}
