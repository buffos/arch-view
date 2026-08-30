package quality_test

import (
	"errors"
	"testing"

	"github.com/buffo/arch-view/internal/quality"
)

func TestQualityQueryServiceFiltersScopesFilesAndPaginatesDeterministically(t *testing.T) {
	profile := thresholdProfile("source:file.max-lines", "rule-config:source-file-size", "greater_than", 10, "unit:line")
	input := quality.EvaluationInput{SourceSnapshots: []quality.SourceSnapshot{
		{SnapshotID: "snapshot-a", ScopeID: "scope-a", Capabilities: []quality.CapabilityDescriptor{{ID: "source:size", Version: "1.0.0"}}, Coverage: []quality.SourceCoverage{{Capability: "source:size", SubjectKind: "file", Status: quality.CoverageObserved}}, Files: []quality.SourceFile{{ID: "file:a", Path: "a.go", LineCount: 11, ByteCount: 100}, {ID: "file:a-clean", Path: "clean.go", LineCount: 10, ByteCount: 80}}},
		{SnapshotID: "snapshot-b", ScopeID: "scope-b", Capabilities: []quality.CapabilityDescriptor{{ID: "source:size", Version: "1.0.0"}}, Coverage: []quality.SourceCoverage{{Capability: "source:size", SubjectKind: "file", Status: quality.CoverageObserved}}, Files: []quality.SourceFile{{ID: "file:b", Path: "b.go", LineCount: 12, ByteCount: 100}}},
	}}
	report, err := quality.EvaluateQualityProfile(profile, input, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("evaluate query fixture: %v", err)
	}
	service := quality.NewQualityQueryService(report)
	page, err := service.ListFindings(report.EvaluationID, quality.QualityFindingQueryOptions{ScopeID: "scope-a", FileID: "file:a", Limit: 1})
	if err != nil {
		t.Fatalf("list scoped findings: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].SubjectRef.ID != "file:a" || page.Items[0].SubjectRef.ScopeID != "scope-a" {
		t.Fatalf("scoped findings = %#v, want one exact file subject", page)
	}
	if len(page.Coverage) != 1 || page.Coverage[0].Provenance.EvidenceIDs[0] != "scope:scope-a:snapshot-a" {
		t.Fatalf("scoped coverage = %#v, want only scope-a", page.Coverage)
	}

	all, err := service.ListFindings(report.EvaluationID, quality.QualityFindingQueryOptions{RuleID: "source:file.max-lines", Limit: 1})
	if err != nil {
		t.Fatalf("list paged findings: %v", err)
	}
	if all.Total != 2 || len(all.Items) != 1 || all.NextCursor == "" {
		t.Fatalf("first findings page = %#v, want two total and a cursor", all)
	}
	next, err := service.ListFindings(report.EvaluationID, quality.QualityFindingQueryOptions{RuleID: "source:file.max-lines", Limit: 1, Cursor: all.NextCursor})
	if err != nil {
		t.Fatalf("list second findings page: %v", err)
	}
	if len(next.Items) != 1 || next.Items[0].FindingKey == all.Items[0].FindingKey {
		t.Fatalf("second findings page = %#v, want the next stable finding", next)
	}

	coverage, err := service.GetQualityCoverage(report.EvaluationID, quality.QualityCoverageQueryOptions{RuleID: "source:file.max-lines"})
	if err != nil {
		t.Fatalf("get coverage: %v", err)
	}
	if coverage.Total != 2 {
		t.Fatalf("coverage total = %d, want one result per source scope", coverage.Total)
	}
	if _, err := service.ListFindings(report.EvaluationID, quality.QualityFindingQueryOptions{Limit: quality.MaxQualityQueryLimit + 1}); err == nil {
		t.Fatal("over-limit quality query unexpectedly succeeded")
	}
}

func TestQualityQueryServiceOwnsAndReturnsIndependentReports(t *testing.T) {
	profile := thresholdProfile("source:file.max-lines", "rule-config:source-file-size", "greater_than", 10, "unit:line")
	report, err := quality.EvaluateQualityProfile(profile, fileLifecycleInput(), quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("evaluate immutable fixture: %v", err)
	}
	service := quality.NewQualityQueryService(report)
	originalMessage := report.Findings[0].Message

	// Mutating either the constructor input or a returned report must not alter
	// the service's immutable projection.
	report.Findings[0].Message = "mutated constructor input"
	first, err := service.GetReport(report.EvaluationID)
	if err != nil {
		t.Fatalf("get first report: %v", err)
	}
	first.Findings[0].Message = "mutated query result"
	first.Findings[0].Evidence.MetricRefs[0] = "metric:mutated"
	second, err := service.GetReport(report.EvaluationID)
	if err != nil {
		t.Fatalf("get second report: %v", err)
	}
	if second.Findings[0].Message != originalMessage || second.Findings[0].Evidence.MetricRefs[0] == "metric:mutated" {
		t.Fatalf("stored report was mutated through an alias: %#v", second.Findings[0])
	}
}

func TestQualityQueryServiceValidatesSelectorsAndUsesCanonicalOrdering(t *testing.T) {
	profile := thresholdProfile("source:file.max-lines", "rule-config:source-file-size", "greater_than", 10, "unit:line")
	report, err := quality.EvaluateQualityProfile(profile, quality.EvaluationInput{SourceSnapshots: []quality.SourceSnapshot{
		{SnapshotID: "snapshot-z", ScopeID: "scope-z", Capabilities: []quality.CapabilityDescriptor{{ID: "source:size", Version: "1.0.0"}}, Coverage: []quality.SourceCoverage{{Capability: "source:size", SubjectKind: "file", Status: quality.CoverageObserved}}, Files: []quality.SourceFile{{ID: "file:a", Path: "a.go", LineCount: 12}}},
		{SnapshotID: "snapshot-a", ScopeID: "scope-a", Capabilities: []quality.CapabilityDescriptor{{ID: "source:size", Version: "1.0.0"}}, Coverage: []quality.SourceCoverage{{Capability: "source:size", SubjectKind: "file", Status: quality.CoverageObserved}}, Files: []quality.SourceFile{{ID: "file:z", Path: "z.go", LineCount: 12}}},
	}}, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("evaluate ordering fixture: %v", err)
	}
	report.Findings[0], report.Findings[1] = report.Findings[1], report.Findings[0]
	service := quality.NewQualityQueryService(report)
	page, err := service.ListFindings(report.EvaluationID, quality.QualityFindingQueryOptions{})
	if err != nil {
		t.Fatalf("list ordered findings: %v", err)
	}
	if page.Items[0].SubjectRef.ScopeID != "scope-a" || page.Items[1].SubjectRef.ScopeID != "scope-z" {
		t.Fatalf("finding order = %q, %q; want scope-a then scope-z", page.Items[0].SubjectRef.ScopeID, page.Items[1].SubjectRef.ScopeID)
	}
	if _, err := service.ListFindings(report.EvaluationID, quality.QualityFindingQueryOptions{AssessmentKind: "proof"}); err == nil {
		t.Fatal("unsupported assessment selector unexpectedly succeeded")
	}
	if _, err := service.ListFindings(report.EvaluationID, quality.QualityFindingQueryOptions{Severity: "urgent"}); err == nil {
		t.Fatal("unsupported severity selector unexpectedly succeeded")
	}
	if _, err := service.ListFindings(report.EvaluationID, quality.QualityFindingQueryOptions{Status: "dismissed"}); err == nil {
		t.Fatal("unsupported status selector unexpectedly succeeded")
	}
}

func TestQualityQueryEvidenceIsCompactAndSourceContextIsExplicitlyBudgeted(t *testing.T) {
	profile := thresholdProfile("source:file.max-lines", "rule-config:source-file-size", "greater_than", 10, "unit:line")
	report, err := quality.EvaluateQualityProfile(profile, fileLifecycleInput(), quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("evaluate evidence fixture: %v", err)
	}
	service := quality.NewQualityQueryService(report)
	evidence, err := service.GetFindingEvidence(report.EvaluationID, report.Findings[0].ID, quality.QualityEvidenceQueryOptions{})
	if err != nil {
		t.Fatalf("get compact evidence: %v", err)
	}
	if evidence.SourceContextRequested || len(evidence.Finding.Evidence.MetricRefs) == 0 {
		t.Fatalf("compact evidence = %#v, want refs without source context", evidence)
	}
	budgeted, err := service.GetFindingEvidence(report.EvaluationID, report.Findings[0].ID, quality.QualityEvidenceQueryOptions{IncludeSourceContext: true, MaxLines: 40, MaxBytes: 4096})
	if err != nil {
		t.Fatalf("get budgeted evidence: %v", err)
	}
	if !budgeted.SourceContextRequested || budgeted.SourceContextMaxLines != 40 || budgeted.SourceContextMaxBytes != 4096 {
		t.Fatalf("budgeted evidence = %#v", budgeted)
	}
	if _, err := service.GetFindingEvidence(report.EvaluationID, report.Findings[0].ID, quality.QualityEvidenceQueryOptions{IncludeSourceContext: true, MaxLines: quality.MaxEvidenceLines + 1}); err == nil {
		t.Fatal("over-budget source context unexpectedly succeeded")
	} else {
		var qualityErr *quality.QualityError
		if !errors.As(err, &qualityErr) {
			t.Fatalf("over-budget error = %T %v", err, err)
		}
	}
	if _, err := service.GetFindingEvidence(report.EvaluationID, "finding:missing", quality.QualityEvidenceQueryOptions{}); err == nil {
		t.Fatal("missing finding evidence unexpectedly succeeded")
	}
}
