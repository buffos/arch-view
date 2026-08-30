package quality_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/buffo/arch-view/internal/quality"
)

func TestFileSizeBoundaryAndDeterministicReport(t *testing.T) {
	profile := thresholdProfile("source:file.max-lines", "rule-config:source-file-size", "greater_than", 500, "unit:line")
	input := quality.EvaluationInput{SourceSnapshots: []quality.SourceSnapshot{{SnapshotID: "snapshot-a", ScopeID: "scope-a", Capabilities: []quality.CapabilityDescriptor{{ID: "source:size", Version: "v1"}}, Coverage: []quality.SourceCoverage{{Capability: "source:size", SubjectKind: "file", Status: quality.CoverageObserved}}, Files: []quality.SourceFile{{ID: "file-500", Path: "exact.go", LineCount: 500}, {ID: "file-501", Path: "large.go", LineCount: 501}}}}}
	catalog := quality.NewDefaultCatalog()
	first, err := quality.EvaluateQualityProfile(profile, input, catalog)
	if err != nil {
		t.Fatalf("evaluate file threshold: %v", err)
	}
	if len(first.Findings) != 1 || first.Findings[0].SubjectRef.ID != "file-501" {
		t.Fatalf("file findings = %#v, want only the 501-line file", first.Findings)
	}
	second, err := quality.EvaluateQualityProfile(profile, input, catalog)
	if err != nil {
		t.Fatalf("evaluate file threshold twice: %v", err)
	}
	firstJSON, _ := json.Marshal(first)
	secondJSON, _ := json.Marshal(second)
	if string(firstJSON) != string(secondJSON) {
		t.Fatalf("equal quality inputs produced different reports\nfirst=%s\nsecond=%s", firstJSON, secondJSON)
	}
	if first.EvaluationFingerprint.Value == "" || first.ReportDigest.Value == "" {
		t.Fatal("deterministic report did not include fingerprints")
	}
	normalized, err := quality.NormalizeQualityReport(first)
	if err != nil {
		t.Fatalf("normalize evaluated report: %v", err)
	}
	normalizedJSON, _ := json.Marshal(normalized)
	if string(firstJSON) != string(normalizedJSON) {
		t.Fatalf("normalizing an evaluated report changed its semantic representation\nfirst=%s\nnormalized=%s", firstJSON, normalizedJSON)
	}
	var decoded quality.QualityEvaluation
	if err := json.Unmarshal(firstJSON, &decoded); err != nil {
		t.Fatalf("decode quality report: %v", err)
	}
	if err := quality.ValidateQualityEvaluation(decoded); err != nil {
		t.Fatalf("validate decoded quality report: %v", err)
	}
	decoded.Metrics[0].SubjectRef.SnapshotID = "undeclared-snapshot"
	if _, err := quality.NormalizeQualityEvaluation(decoded); err == nil {
		t.Fatal("metric reference to an undeclared source snapshot was accepted")
	}
}

func TestCatalogRegistrationIsAdditiveIdempotentAndConflictSafe(t *testing.T) {
	catalog := quality.NewCatalog()
	provider := fixtureMetricProvider{}
	if err := catalog.RegisterMetricProvider(provider); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	if err := catalog.RegisterMetricProvider(provider); err != nil {
		t.Fatalf("same provider registration should be idempotent: %v", err)
	}
	if err := catalog.RegisterMetricProvider(conflictingMetricProvider{}); err == nil {
		t.Fatal("conflicting provider registration unexpectedly succeeded")
	}
	rule := fixtureRule{}
	if err := catalog.RegisterQualityRule(rule); err != nil {
		t.Fatalf("register rule: %v", err)
	}
	if err := catalog.RegisterQualityRule(rule); err != nil {
		t.Fatalf("same rule registration should be idempotent: %v", err)
	}
	if len(catalog.ListQualityRules()) != 1 || len(catalog.ListQualityCapabilities()) != 1 {
		t.Fatalf("catalog listing did not retain additive entries: rules=%#v capabilities=%#v", catalog.ListQualityRules(), catalog.ListQualityCapabilities())
	}
}

func TestProfileValidationReturnsStructuredDiagnostics(t *testing.T) {
	profile := thresholdProfile("source:file.max-lines", "rule-config:source-file-size", "greater_than", 500, "unit:banana")
	profile.EnabledRules[0].RuleVersion = "9.0.0"
	_, err := quality.ValidateQualityProfile(profile, quality.NewDefaultCatalog())
	if err == nil {
		t.Fatal("invalid profile unexpectedly validated")
	}
	var validationErr *quality.ProfileValidationError
	if !errors.As(err, &validationErr) || len(validationErr.Diagnostics) < 1 {
		t.Fatalf("validation error = %T %v, want structured diagnostics", err, err)
	}
	hasUnsupported := false
	for _, diagnostic := range validationErr.Diagnostics {
		if diagnostic.Code == "QualityRuleUnsupported" {
			hasUnsupported = true
		}
	}
	if !hasUnsupported {
		t.Fatalf("validation diagnostics = %#v, want unsupported rule version", validationErr.Diagnostics)
	}
}

func TestCallableSizeRequiresExtractorBodySpan(t *testing.T) {
	profile := thresholdProfile("source:callable.max-lines", "rule-config:source-callable-size", "greater_or_equal", 3, "unit:line")
	input := quality.EvaluationInput{SourceSnapshots: []quality.SourceSnapshot{{SnapshotID: "snapshot-callables", ScopeID: "scope-callables", Capabilities: []quality.CapabilityDescriptor{{ID: "source:declarations", Version: "v1"}}, Coverage: []quality.SourceCoverage{{Capability: "source:callable.metrics", SubjectKind: "symbol", Status: quality.CoverageObserved}}, Symbols: []quality.SourceSymbol{{ID: "with-body", Name: "WithBody", Category: "callable", StableKey: "WithBody", BodySpan: &quality.SourceSpan{FileID: "file.go", Start: quality.SpanPosition{ByteOffset: 0, Line: 2, Column: 1}, End: quality.SpanPosition{ByteOffset: 10, Line: 4, Column: 2}, CoordinateSystem: "utf8-byte", ContentHash: quality.ContentDigest{Algorithm: "hash:sha-256", Value: "0000000000000000000000000000000000000000000000000000000000000000"}}}, {ID: "without-body", Name: "WithoutBody", Category: "callable", StableKey: "WithoutBody"}}}}}
	input.SourceSnapshots[0].Files = []quality.SourceFile{{ID: "file.go", Path: "file.go", ByteCount: 100, ContentHash: quality.ContentDigest{Algorithm: "hash:sha-256", Value: "0000000000000000000000000000000000000000000000000000000000000000"}}}
	report, err := quality.EvaluateQualityProfile(profile, input, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("evaluate callable size: %v", err)
	}
	if len(report.Findings) != 1 || report.Findings[0].SubjectRef.ID != "with-body" {
		t.Fatalf("callable findings = %#v, want only the body-span callable", report.Findings)
	}
	if len(report.Diagnostics) == 0 || report.Coverage[0].Status != quality.CoveragePartial {
		t.Fatalf("callable coverage=%#v diagnostics=%#v, want partial coverage and diagnostic", report.Coverage, report.Diagnostics)
	}
}

func TestDocumentationCoverageExcludesUnknownAndUnsupportedSubjects(t *testing.T) {
	profile := emptyRuleProfile("source:public-symbol.documentation", "rule-config:documentation-coverage")
	symbols := make([]quality.SourceSymbol, 0, 10)
	documentation := make([]quality.SourceDocumentation, 0, 8)
	for index := 0; index < 10; index++ {
		visibility := "public"
		visibilityStatus := "observed"
		if index >= 8 {
			visibility = "public"
			visibilityStatus = "unknown"
		}
		symbols = append(symbols, quality.SourceSymbol{ID: "symbol-" + string(rune('a'+index)), Name: "Symbol", Category: "type", Visibility: visibility, VisibilityStatus: visibilityStatus})
		if index < 8 {
			status := "present"
			if index >= 6 {
				status = "unsupported"
			}
			documentation = append(documentation, quality.SourceDocumentation{ID: "doc-" + string(rune('a'+index)), SubjectRef: quality.EntityRef{Kind: "symbol", ID: "symbol-" + string(rune('a'+index))}, Status: status})
		}
	}
	report, err := quality.EvaluateQualityProfile(profile, quality.EvaluationInput{SourceSnapshots: []quality.SourceSnapshot{{SnapshotID: "snapshot-docs", ScopeID: "scope-docs", Capabilities: []quality.CapabilityDescriptor{{ID: "source:documentation", Version: "v1"}, {ID: "source:visibility", Version: "v1"}}, Symbols: symbols, Documentation: documentation}}}, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("evaluate documentation coverage: %v", err)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("documentation findings = %#v, unsupported/unknown subjects must not be reported as missing", report.Findings)
	}
	if len(report.Coverage) != 1 || report.Coverage[0].EvaluatedCount == nil || *report.Coverage[0].EvaluatedCount != 6 || report.Coverage[0].Status != quality.CoveragePartial {
		t.Fatalf("documentation coverage = %#v, want observed denominator six and partial status", report.Coverage)
	}
}

func TestDocumentationRuleHonorsExplicitUnsupportedCoverage(t *testing.T) {
	profile := emptyRuleProfile("source:public-symbol.documentation", "rule-config:documentation-coverage")
	snapshot := quality.SourceSnapshot{
		SnapshotID: "snapshot-unsupported-docs",
		ScopeID:    "scope-unsupported-docs",
		Capabilities: []quality.CapabilityDescriptor{
			{ID: "source:documentation", Version: "v1"},
			{ID: "source:visibility", Version: "v1"},
		},
		Coverage: []quality.SourceCoverage{
			{Capability: "source:documentation", SubjectKind: "symbol", Status: quality.CoverageUnsupported, Reason: "documentation extraction is unsupported"},
			{Capability: "source:visibility", SubjectKind: "symbol", Status: quality.CoverageObserved},
		},
		Symbols:       []quality.SourceSymbol{{ID: "public", Name: "Public", Category: "type", Visibility: "public", VisibilityStatus: "observed"}},
		Documentation: []quality.SourceDocumentation{{ID: "doc-public", SubjectRef: quality.EntityRef{Kind: "symbol", ID: "public"}, Status: "absent"}},
	}
	report, err := quality.EvaluateQualityProfile(profile, quality.EvaluationInput{SourceSnapshots: []quality.SourceSnapshot{snapshot}}, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("evaluate unsupported documentation coverage: %v", err)
	}
	coverage := coverageForRule(report, "source:public-symbol.documentation")
	if coverage == nil || coverage.Status != quality.CoverageUnsupported || len(report.Findings) != 0 {
		t.Fatalf("unsupported documentation report = %#v, want unsupported coverage and no finding", report)
	}
}

func TestComplexityNestingGraphAndExplicitConstraintsUseDeclaredFacts(t *testing.T) {
	profile := thresholdProfile("source:callable.max-cyclomatic-complexity", "rule-config:source-callable-complexity", "greater_than", 2, "unit:complexity")
	profile.EnabledRules = append(profile.EnabledRules, quality.RuleBinding{RuleID: "source:callable.max-nesting-depth", RuleVersion: "1.0.0", Enabled: true, Parameters: typed("rule-config:source-callable-nesting", map[string]any{"operator": "greater_than", "limit": 1, "unit": "unit:depth"})})
	profile.EnabledRules = append(profile.EnabledRules, quality.RuleBinding{RuleID: "architecture:no-cycles", RuleVersion: "1.0.0", Enabled: true, Parameters: typed("rule-config:no-cycles", map[string]any{})})
	profile.EnabledRules = append(profile.EnabledRules, quality.RuleBinding{RuleID: "architecture:forbidden-dependency", RuleVersion: "1.0.0", Enabled: true, Parameters: typed("rule-config:architecture-constraint", map[string]any{})})
	profile.Constraints = []quality.ArchitectureConstraint{{ID: "constraint:forbidden", Kind: "forbidden_dependency", Parameters: typed("constraint:forbidden", map[string]any{"from_module_ids": []string{"module-a"}, "to_module_ids": []string{"module-b"}})}}
	complexityFact := quality.MetricFact{
		ID:             "complexity",
		SubjectRef:     quality.EntityRef{Kind: "symbol", ID: "callable", ScopeID: "scope-c", SnapshotID: "snapshot-c"},
		MetricID:       "source:callable.cyclomatic_complexity",
		Value:          quality.MetricValue{Kind: quality.ValueInteger, Value: 3},
		Unit:           "unit:complexity",
		FormulaID:      "formula:go.cyclomatic-complexity",
		FormulaVersion: "1.0.0",
		Provenance:     quality.FactProvenance{Status: "observed", Basis: "syntax", Provider: "provider:go", ProviderVersion: "1.0.0"},
	}
	nestingFact := quality.MetricFact{
		ID:             "nesting",
		SubjectRef:     quality.EntityRef{Kind: "symbol", ID: "callable", ScopeID: "scope-c", SnapshotID: "snapshot-c"},
		MetricID:       "source:callable.max_nesting_depth",
		Value:          quality.MetricValue{Kind: quality.ValueInteger, Value: 2},
		Unit:           "unit:depth",
		FormulaID:      "formula:go.max-nesting-depth",
		FormulaVersion: "1.0.0",
		Provenance:     quality.FactProvenance{Status: "observed", Basis: "syntax", Provider: "provider:go", ProviderVersion: "1.0.0"},
	}
	snapshot := quality.SourceSnapshot{
		SnapshotID: "snapshot-c",
		ScopeID:    "scope-c",
		Capabilities: []quality.CapabilityDescriptor{
			{ID: "source:declarations", Version: "v1"},
			{ID: "source:callable.metrics", Version: "v1"},
		},
		Coverage: []quality.SourceCoverage{{Capability: "source:callable.metrics", SubjectKind: "symbol", Status: quality.CoverageObserved}},
		Symbols:  []quality.SourceSymbol{{ID: "callable", StableKey: "callable", Name: "Callable", Category: "callable"}},
		Metrics:  []quality.MetricFact{complexityFact, nestingFact},
	}
	architecture := &quality.ArchitectureModel{
		ScopeID: "scope-c",
		Modules: []quality.ArchitectureModule{{ID: "module-a", Name: "a"}, {ID: "module-b", Name: "b"}},
		Relationships: []quality.ArchitectureRelationship{
			{ID: "edge-ab", Type: "depends_on", FromModuleID: "module-a", ToModuleID: "module-b"},
			{ID: "edge-ba", Type: "depends_on", FromModuleID: "module-b", ToModuleID: "module-a"},
		},
		Cycles: []quality.ArchitectureCycle{{ID: "cycle-ab", ModuleIDs: []string{"module-a", "module-b"}, RelationshipIDs: []string{"edge-ab", "edge-ba"}}},
	}
	input := quality.EvaluationInput{SourceSnapshots: []quality.SourceSnapshot{snapshot}, Architecture: architecture}
	report, err := quality.EvaluateQualityProfile(profile, input, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("evaluate complexity/graph rules: %v", err)
	}
	if len(report.Findings) < 4 {
		t.Fatalf("findings = %#v, want complexity, nesting, cycle, and constraint findings", report.Findings)
	}
	for _, finding := range report.Findings {
		if finding.RuleID == "architecture:forbidden-dependency" && !reflect.DeepEqual(finding.Evidence.RelationRefs[0].ID, "edge-ab") {
			t.Fatalf("forbidden dependency evidence = %#v, want canonical edge", finding.Evidence)
		}
	}
}

func TestMissingAndUnsupportedSourceCoverageNeverLooksLikeApass(t *testing.T) {
	profile := thresholdProfile("source:file.max-lines", "rule-config:source-file-size", "greater_than", 10, "unit:line")
	baseSnapshot := quality.SourceSnapshot{
		SnapshotID: "snapshot-coverage",
		ScopeID:    "scope-coverage",
		Capabilities: []quality.CapabilityDescriptor{{
			ID: "source:size", Version: "v1",
		}},
		Files: []quality.SourceFile{{ID: "large.go", Path: "large.go", LineCount: 11}},
	}

	missing, err := quality.EvaluateQualityProfile(profile, quality.EvaluationInput{SourceSnapshots: []quality.SourceSnapshot{baseSnapshot}}, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("evaluate missing source coverage: %v", err)
	}
	missingCoverage := coverageForRule(missing, "source:file.max-lines")
	if missingCoverage == nil || missingCoverage.Status != quality.CoverageNotEvaluable || len(missing.Findings) != 0 {
		t.Fatalf("missing source coverage report = %#v, want not-evaluable coverage and no finding", missing)
	}

	unsupportedSnapshot := baseSnapshot
	unsupportedSnapshot.Coverage = []quality.SourceCoverage{{Capability: "source:size", SubjectKind: "file", Status: quality.CoverageUnsupported, Reason: "language is not supported"}}
	unsupported, err := quality.EvaluateQualityProfile(profile, quality.EvaluationInput{SourceSnapshots: []quality.SourceSnapshot{unsupportedSnapshot}}, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("evaluate unsupported source coverage: %v", err)
	}
	unsupportedCoverage := coverageForRule(unsupported, "source:file.max-lines")
	if unsupportedCoverage == nil || unsupportedCoverage.Status != quality.CoverageUnsupported || len(unsupported.Findings) != 0 {
		t.Fatalf("unsupported source coverage report = %#v, want unsupported coverage and no finding", unsupported)
	}
}

func TestCallableRuleRequiresCompatibleVersionedMetric(t *testing.T) {
	profile := thresholdProfile("source:callable.max-cyclomatic-complexity", "rule-config:source-callable-complexity", "greater_than", 2, "unit:complexity")
	snapshot := quality.SourceSnapshot{
		SnapshotID: "snapshot-formula",
		ScopeID:    "scope-formula",
		Capabilities: []quality.CapabilityDescriptor{{
			ID: "source:callable.metrics", Version: "v1",
		}},
		Coverage: []quality.SourceCoverage{{Capability: "source:callable.metrics", SubjectKind: "symbol", Status: quality.CoverageObserved}},
		Symbols:  []quality.SourceSymbol{{ID: "callable", Name: "Callable", Category: "callable"}},
		Metrics: []quality.MetricFact{{
			ID: "metric:wrong-formula", SubjectRef: quality.EntityRef{Kind: "symbol", ID: "callable", ScopeID: "scope-formula", SnapshotID: "snapshot-formula"},
			MetricID: "source:callable.cyclomatic_complexity", Value: quality.MetricValue{Kind: quality.ValueInteger, Value: 9}, Unit: "unit:complexity", FormulaID: "formula:go.cyclomatic-complexity", FormulaVersion: "2.0.0",
			Provenance: quality.FactProvenance{Status: "observed", Basis: "syntax", Provider: "provider:go", ProviderVersion: "1.0.0"},
		}},
	}
	compatibleSnapshot := snapshot
	compatibleSnapshot.Metrics = append([]quality.MetricFact(nil), snapshot.Metrics...)
	compatibleSnapshot.Metrics[0].ID = "metric:alternate-provider"
	compatibleSnapshot.Metrics[0].FormulaID = "formula:alternate.cyclomatic-complexity"
	compatibleSnapshot.Metrics[0].FormulaVersion = "1.0.0"
	compatibleSnapshot.Metrics[0].Provenance.Provider = "provider:alternate"
	compatible, err := quality.EvaluateQualityProfile(profile, quality.EvaluationInput{SourceSnapshots: []quality.SourceSnapshot{compatibleSnapshot}}, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("evaluate compatible alternate callable metric: %v", err)
	}
	if len(compatible.Findings) != 1 || compatible.Findings[0].ObservedMetricIDs[0] == "" {
		t.Fatalf("compatible alternate formula report = %#v, want one evidence-backed finding", compatible)
	}

	report, err := quality.EvaluateQualityProfile(profile, quality.EvaluationInput{SourceSnapshots: []quality.SourceSnapshot{snapshot}}, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("evaluate incompatible callable metric: %v", err)
	}
	coverage := coverageForRule(report, "source:callable.max-cyclomatic-complexity")
	if coverage == nil || coverage.Status != quality.CoverageNotEvaluable || len(report.Findings) != 0 {
		t.Fatalf("incompatible metric report = %#v, want not-evaluable coverage and no finding", report)
	}
}

func TestArchitectureConstraintValidationRejectsWrongSchemasAndUnknownFields(t *testing.T) {
	profile := emptyRuleProfile("architecture:forbidden-dependency", "rule-config:architecture-constraint")
	profile.Constraints = []quality.ArchitectureConstraint{{
		ID:   "constraint:invalid",
		Kind: "forbidden_dependency",
		Parameters: typed("constraint:layers", map[string]any{
			"from_module_ids": []string{"module-a"},
			"to_module_ids":   []string{"module-b"},
		}),
	}}
	if _, err := quality.ValidateQualityProfile(profile, quality.NewDefaultCatalog()); err == nil {
		t.Fatal("wrong constraint namespace was accepted")
	}

	profile.Constraints[0].Parameters = typed("constraint:forbidden", map[string]any{
		"from_module_ids": []string{"module-a"},
		"to_module_ids":   []string{"module-b"},
		"scpoe_id":        "typo",
	})
	if _, err := quality.ValidateQualityProfile(profile, quality.NewDefaultCatalog()); err == nil {
		t.Fatal("unknown constraint parameter was accepted")
	}
}

func TestGraphCouplingCountsDistinctReportedTargetsAndReferences(t *testing.T) {
	profile := thresholdProfile("architecture:module.max-efferent-coupling", "rule-config:module-coupling", "greater_than", 1, "unit:module")
	profile.EnabledRules = append(profile.EnabledRules, quality.RuleBinding{RuleID: "architecture:module.max-afferent-coupling", RuleVersion: "1.0.0", Enabled: true, Parameters: typed("rule-config:module-coupling", map[string]any{"operator": "greater_than", "limit": 1, "unit": "unit:module", "external_policy": "exclude"})})
	profile.EnabledRules[0].Parameters.Payload.(map[string]any)["external_policy"] = "exclude"
	graph := &quality.ArchitectureModel{
		ScopeID: "model-scope",
		Modules: []quality.ArchitectureModule{{ID: "module-a", StableKey: "internal/a", Name: "a"}, {ID: "module-b", StableKey: "internal/b", Name: "b"}, {ID: "module-c", StableKey: "internal/c", Name: "c"}},
		Relationships: []quality.ArchitectureRelationship{
			{ID: "edge-ab-1", Type: "depends_on", FromModuleID: "module-a", ToModuleID: "module-b"},
			{ID: "edge-ab-2", Type: "depends_on", FromModuleID: "module-a", ToModuleID: "module-b"},
			{ID: "edge-ac", Type: "depends_on", FromModuleID: "module-a", ToModuleID: "module-c"},
			{ID: "edge-ba", Type: "depends_on", FromModuleID: "module-b", ToModuleID: "module-a"},
			{ID: "edge-external-1", Type: "depends_on", FromModuleID: "module-a", ToReferenceID: "reference-one"},
			{ID: "edge-external-2", Type: "depends_on", FromModuleID: "module-a", ToReferenceID: "reference-one"},
		},
	}
	report, err := quality.EvaluateQualityProfile(profile, quality.EvaluationInput{Architecture: graph}, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("evaluate coupling: %v", err)
	}
	efferent := metricForSubject(report, "architecture:module.efferent_coupling", "module-a")
	if efferent == nil || efferent.Value.Value != 2 {
		t.Fatalf("efferent metric = %#v, want two distinct internal targets", efferent)
	}
	if len(report.Findings) != 1 || report.Findings[0].RuleID != "architecture:module.max-efferent-coupling" {
		t.Fatalf("coupling findings = %#v, want only module-a efferent finding", report.Findings)
	}

	for index := range profile.EnabledRules {
		profile.EnabledRules[index].Parameters.Payload.(map[string]any)["external_policy"] = "include"
		profile.EnabledRules[index].Parameters.Payload.(map[string]any)["limit"] = 2
	}
	report, err = quality.EvaluateQualityProfile(profile, quality.EvaluationInput{Architecture: graph}, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("evaluate coupling with external references: %v", err)
	}
	efferent = metricForSubject(report, "architecture:module.efferent_coupling", "module-a")
	if efferent == nil || efferent.Value.Value != 3 || len(report.Findings) != 1 {
		t.Fatalf("external coupling report = %#v, want three distinct targets and one finding", report)
	}
}

func TestCanonicalCycleProducesOneFindingPerReportedCycle(t *testing.T) {
	profile := emptyRuleProfile("architecture:no-cycles", "rule-config:no-cycles")
	graph := &quality.ArchitectureModel{
		ScopeID:       "cycle-scope",
		Modules:       []quality.ArchitectureModule{{ID: "module-a", Name: "a"}, {ID: "module-b", Name: "b"}},
		Relationships: []quality.ArchitectureRelationship{{ID: "edge-ab", Type: "depends_on", FromModuleID: "module-a", ToModuleID: "module-b"}, {ID: "edge-ba", Type: "depends_on", FromModuleID: "module-b", ToModuleID: "module-a"}},
		Cycles:        []quality.ArchitectureCycle{{ID: "cycle-ab", ModuleIDs: []string{"module-a", "module-b"}, RelationshipIDs: []string{"edge-ab", "edge-ba"}}, {ID: "cycle-ab", ModuleIDs: []string{"module-a", "module-b"}, RelationshipIDs: []string{"edge-ab", "edge-ba"}}},
	}
	report, err := quality.EvaluateQualityProfile(profile, quality.EvaluationInput{Architecture: graph}, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("evaluate canonical cycle: %v", err)
	}
	if len(report.Findings) != 1 || report.Findings[0].SubjectRef.ID != "cycle-ab" || len(report.Findings[0].Evidence.RelationRefs) != 2 {
		t.Fatalf("cycle findings = %#v, want one canonical finding with two relationship refs", report.Findings)
	}
	coverage := coverageForRule(report, "architecture:no-cycles")
	if coverage == nil || coverage.EvaluatedCount == nil || *coverage.EvaluatedCount != 1 {
		t.Fatalf("cycle coverage = %#v, want one evaluated cycle", coverage)
	}
}

func TestExplicitConstraintSelectorsAndLayerAssignmentsRemainEvidenceBacked(t *testing.T) {
	profile := emptyRuleProfile("architecture:forbidden-dependency", "rule-config:architecture-constraint")
	profile.EnabledRules = append(profile.EnabledRules, quality.RuleBinding{RuleID: "architecture:layer-direction", RuleVersion: "1.0.0", Enabled: true, Parameters: typed("rule-config:architecture-constraint", map[string]any{})})
	profile.Constraints = []quality.ArchitectureConstraint{
		{ID: "constraint:forbidden-internal", Kind: "forbidden_dependency", Parameters: typed("constraint:forbidden", map[string]any{"from_module_patterns": []string{"internal/*"}, "to_module_ids": []string{"module-a"}})},
		{ID: "constraint:layer-downward", Kind: "layer_direction", Parameters: typed("constraint:layers", map[string]any{"from_layer": 1, "to_layer": 0})},
	}
	graph := &quality.ArchitectureModel{
		ScopeID:       "constraint-scope",
		Modules:       []quality.ArchitectureModule{{ID: "module-a", StableKey: "internal/a", Name: "a"}, {ID: "module-b", StableKey: "internal/b", Name: "b"}},
		Relationships: []quality.ArchitectureRelationship{{ID: "edge-ba", Type: "depends_on", FromModuleID: "module-b", ToModuleID: "module-a"}},
		Layers:        []quality.ArchitectureLayer{{Layer: 0, ModuleIDs: []string{"module-a"}}, {Layer: 1, ModuleIDs: []string{"module-b"}}},
	}
	report, err := quality.EvaluateQualityProfile(profile, quality.EvaluationInput{Architecture: graph}, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("evaluate explicit constraints: %v", err)
	}
	if len(report.Findings) != 2 {
		t.Fatalf("constraint findings = %#v, want forbidden and layer findings", report.Findings)
	}
	for _, finding := range report.Findings {
		if len(finding.Evidence.EntityRefs) != 2 || len(finding.Evidence.RelationRefs) != 1 {
			t.Fatalf("constraint evidence = %#v, want source/target and relationship refs", finding.Evidence)
		}
	}
}

func TestArchitectureRulesRequireExplicitAggregateForMultipleScopes(t *testing.T) {
	profile := thresholdProfile("architecture:module.max-efferent-coupling", "rule-config:module-coupling", "greater_than", 0, "unit:module")
	graph := &quality.ArchitectureModel{ScopeID: "aggregate-candidate", Modules: []quality.ArchitectureModule{{ID: "module-a", Name: "a"}}, Relationships: []quality.ArchitectureRelationship{}}
	input := quality.EvaluationInput{SourceSnapshots: []quality.SourceSnapshot{{SnapshotID: "snapshot-a", ScopeID: "scope-a"}, {SnapshotID: "snapshot-b", ScopeID: "scope-b"}}, Architecture: graph}
	report, err := quality.EvaluateQualityProfile(profile, input, quality.NewDefaultCatalog())
	if err != nil {
		t.Fatalf("evaluate mixed-scope graph: %v", err)
	}
	coverage := coverageForRule(report, "architecture:module.max-efferent-coupling")
	if coverage == nil || coverage.Status != quality.CoverageNotEvaluable || len(report.Findings) != 0 {
		t.Fatalf("mixed-scope graph report = %#v, want not-evaluable coverage and no finding", report)
	}
}

func coverageForRule(report quality.QualityEvaluation, ruleID string) *quality.QualityCoverage {
	for index := range report.Coverage {
		if report.Coverage[index].RuleID == ruleID {
			return &report.Coverage[index]
		}
	}
	return nil
}

func metricForSubject(report quality.QualityEvaluation, metricID, subjectID string) *quality.MetricFact {
	for index := range report.Metrics {
		if report.Metrics[index].MetricID == metricID && report.Metrics[index].SubjectRef.ID == subjectID {
			return &report.Metrics[index]
		}
	}
	return nil
}

func thresholdProfile(ruleID, namespace, operator string, limit int, unit string) quality.QualityProfile {
	return quality.QualityProfile{SchemaVersion: quality.SchemaVersion, ProfileID: "profile:test", ProfileVersion: "1.0.0", EnabledRules: []quality.RuleBinding{{RuleID: ruleID, RuleVersion: "1.0.0", Enabled: true, Parameters: typed(namespace, map[string]any{"operator": operator, "limit": limit, "unit": unit})}}, SeverityPolicy: typed("severity:default", map[string]any{}), Extensions: []quality.ExtensionBlock{}}
}

func emptyRuleProfile(ruleID, namespace string) quality.QualityProfile {
	return quality.QualityProfile{SchemaVersion: quality.SchemaVersion, ProfileID: "profile:test", ProfileVersion: "1.0.0", EnabledRules: []quality.RuleBinding{{RuleID: ruleID, RuleVersion: "1.0.0", Enabled: true, Parameters: typed(namespace, map[string]any{})}}, SeverityPolicy: typed("severity:default", map[string]any{}), Extensions: []quality.ExtensionBlock{}}
}

func typed(namespace string, payload map[string]any) quality.TypedConfigBlock {
	return quality.TypedConfigBlock{Namespace: namespace, SchemaVersion: "1.0.0", Payload: payload}
}

type fixtureMetricProvider struct{}

func (fixtureMetricProvider) ID() string      { return "provider:fixture" }
func (fixtureMetricProvider) Version() string { return "1.0.0" }
func (fixtureMetricProvider) Capabilities() []quality.CapabilityDescriptor {
	return []quality.CapabilityDescriptor{{ID: "metric:fixture", Version: "1.0.0"}}
}
func (fixtureMetricProvider) Compute(quality.EvaluationContext) (quality.MetricBatch, error) {
	return quality.MetricBatch{}, nil
}

type conflictingMetricProvider struct{}

func (conflictingMetricProvider) ID() string      { return "provider:fixture" }
func (conflictingMetricProvider) Version() string { return "1.0.0" }
func (conflictingMetricProvider) Capabilities() []quality.CapabilityDescriptor {
	return []quality.CapabilityDescriptor{{ID: "metric:fixture", Version: "1.0.0"}}
}
func (conflictingMetricProvider) Compute(quality.EvaluationContext) (quality.MetricBatch, error) {
	return quality.MetricBatch{Diagnostics: []quality.QualityDiagnostic{{Code: "different"}}}, nil
}

type fixtureRule struct{}

func (fixtureRule) ID() string                     { return "rule:fixture" }
func (fixtureRule) Version() string                { return "1.0.0" }
func (fixtureRule) AssessmentKind() string         { return quality.AssessmentExact }
func (fixtureRule) RequiredCapabilities() []string { return []string{"metric:fixture"} }
func (fixtureRule) Evaluate(quality.EvaluationContext, quality.MetricBatch) (quality.RuleResult, error) {
	return quality.RuleResult{}, nil
}
