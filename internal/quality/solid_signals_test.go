package quality_test

import (
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/quality"
)

func TestSolidSignalsRegisterAndEmitConservativeEvidenceBackedResults(t *testing.T) {
	catalog := quality.NewDefaultCatalog()
	ruleIDs := []string{"signal:solid.srp", "signal:solid.ocp", "signal:solid.lsp", "signal:solid.isp", "signal:solid.dip"}
	registered := catalog.ListQualityRules()
	for _, ruleID := range ruleIDs {
		found := false
		for _, descriptor := range registered {
			if descriptor.ID == ruleID && descriptor.AssessmentKind == quality.AssessmentSignal {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("catalog does not register advisory signal %q", ruleID)
		}
	}

	profile := quality.QualityProfile{SchemaVersion: quality.SchemaVersion, ProfileID: "profile:solid", ProfileVersion: "1.0.0", EnabledRules: []quality.RuleBinding{}, SeverityPolicy: typed("severity:default", map[string]any{}), Extensions: []quality.ExtensionBlock{}}
	for _, ruleID := range ruleIDs {
		profile.EnabledRules = append(profile.EnabledRules, quality.RuleBinding{RuleID: ruleID, RuleVersion: "1.0.0", Enabled: true, Parameters: typed("rule-config:solid-signal", map[string]any{})})
	}
	value := func(number int) *int { return &number }
	snapshot := quality.SourceSnapshot{
		SnapshotID:   "snapshot-solid",
		ScopeID:      "scope-solid",
		Capabilities: []quality.CapabilityDescriptor{{ID: "source:solid.structure", Version: "1.0.0"}},
		Coverage:     []quality.SourceCoverage{{Capability: "source:solid.structure", SubjectKind: "symbol", Status: quality.CoverageObserved}},
		Symbols: []quality.SourceSymbol{{
			ID: "symbol:service", StableKey: "internal/service.go#Service", Name: "Service", Category: "type", LanguageKind: "go:struct",
			MemberCount: value(12), DependencyCount: value(4), TypeSwitchCount: value(1), HierarchyDepth: value(2), DerivedTypeCount: value(1), InterfaceMethodCount: value(9), ConcreteDependencyCount: value(3),
		}},
	}
	report, err := quality.EvaluateQualityProfile(profile, quality.EvaluationInput{SourceSnapshots: []quality.SourceSnapshot{snapshot}}, catalog)
	if err != nil {
		t.Fatalf("evaluate SOLID signals: %v", err)
	}
	if len(report.Findings) != len(ruleIDs) {
		t.Fatalf("SOLID findings = %d, want %d: %#v", len(report.Findings), len(ruleIDs), report.Findings)
	}
	for _, finding := range report.Findings {
		if finding.AssessmentKind != quality.AssessmentSignal || finding.Severity != quality.SeverityInfo || finding.Status != quality.StatusActive {
			t.Fatalf("signal finding shape = %#v", finding)
		}
		if len(finding.ObservedMetricIDs) == 0 || len(finding.Evidence.MetricRefs) == 0 || len(finding.Limitations) == 0 {
			t.Fatalf("signal finding lacks indicators/evidence/limitations = %#v", finding)
		}
		if strings.Contains(strings.ToLower(finding.Message), "violation") || strings.Contains(strings.ToLower(finding.Message), "proven") {
			t.Fatalf("signal message overclaims a verdict: %q", finding.Message)
		}
		if finding.Provenance.Basis != "heuristic" || finding.Provenance.Provider != "rule:solid-signal" {
			t.Fatalf("signal provenance = %#v", finding.Provenance)
		}
	}
}

func TestSolidSignalsRespectThresholdsAndUnsupportedCoverage(t *testing.T) {
	profile := quality.QualityProfile{SchemaVersion: quality.SchemaVersion, ProfileID: "profile:solid-threshold", ProfileVersion: "1.0.0", EnabledRules: []quality.RuleBinding{{RuleID: "signal:solid.srp", RuleVersion: "1.0.0", Enabled: true, Parameters: typed("rule-config:solid-signal", map[string]any{"member_threshold": 20, "dependency_threshold": 5})}}, SeverityPolicy: typed("severity:default", map[string]any{}), Extensions: []quality.ExtensionBlock{}}
	snapshot := quality.SourceSnapshot{SnapshotID: "snapshot-solid-threshold", ScopeID: "scope-solid-threshold", Capabilities: []quality.CapabilityDescriptor{{ID: "source:solid.structure", Version: "1.0.0"}}, Coverage: []quality.SourceCoverage{{Capability: "source:solid.structure", SubjectKind: "symbol", Status: quality.CoverageObserved}}, Symbols: []quality.SourceSymbol{{ID: "symbol:small", StableKey: "Small", Name: "Small", Category: "type", MemberCount: intPointerForTest(19), DependencyCount: intPointerForTest(4)}}}
	report, err := quality.EvaluateQualityProfile(profile, quality.EvaluationInput{SourceSnapshots: []quality.SourceSnapshot{snapshot}}, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("evaluate below threshold: %v", err)
	}
	if len(report.Findings) != 0 || len(report.Coverage) != 1 || report.Coverage[0].Status != quality.CoverageObserved {
		t.Fatalf("below-threshold report = %#v, want observed clean structural signal", report)
	}

	snapshot.Coverage[0].Status = quality.CoverageUnsupported
	snapshot.Coverage[0].Reason = "extractor does not report structural counts"
	unsupported, err := quality.EvaluateQualityProfile(profile, quality.EvaluationInput{SourceSnapshots: []quality.SourceSnapshot{snapshot}}, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("evaluate unsupported structural facts: %v", err)
	}
	if len(unsupported.Findings) != 0 || len(unsupported.Coverage) != 1 || unsupported.Coverage[0].Status != quality.CoverageUnsupported {
		t.Fatalf("unsupported SOLID report = %#v, want unsupported coverage and no finding", unsupported)
	}
}

func intPointerForTest(value int) *int { return &value }
