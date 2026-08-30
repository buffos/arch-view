package quality

import (
	"fmt"
	"sort"
	"strings"
)

const solidProviderVersion = "1.0.0"

const (
	solidStructureCapability = "source:solid.structure"
	solidMemberMetric        = "source:solid.member_count"
	solidMethodMetric        = "source:solid.method_count"
	solidDependencyMetric    = "source:solid.dependency_count"
	solidConcreteMetric      = "source:solid.concrete_dependency_count"
	solidInterfaceMetric     = "source:solid.interface_method_count"
	solidTypeSwitchMetric    = "source:solid.type_switch_count"
	solidHierarchyMetric     = "source:solid.hierarchy_depth"
	solidDerivedMetric       = "source:solid.derived_type_count"
	solidAbstractionMetric   = "source:solid.abstraction_count"
)

type solidMetricProvider struct{}

func newSolidMetricProvider() MetricProvider { return solidMetricProvider{} }
func (solidMetricProvider) ID() string       { return "provider:solid-structure" }
func (solidMetricProvider) Version() string  { return solidProviderVersion }
func (solidMetricProvider) Capabilities() []CapabilityDescriptor {
	return []CapabilityDescriptor{{ID: solidStructureCapability, Version: solidProviderVersion, Description: "Extractor-reported structural counts for advisory SOLID signals."}}
}

type solidMetricDefinition struct {
	ID    string
	Alias string
}

var solidMetricDefinitions = []solidMetricDefinition{
	{ID: solidMemberMetric, Alias: "member_count"},
	{ID: solidMethodMetric, Alias: "method_count"},
	{ID: solidDependencyMetric, Alias: "dependency_count"},
	{ID: solidConcreteMetric, Alias: "concrete_dependency_count"},
	{ID: solidInterfaceMetric, Alias: "interface_method_count"},
	{ID: solidTypeSwitchMetric, Alias: "type_switch_count"},
	{ID: solidHierarchyMetric, Alias: "hierarchy_depth"},
	{ID: solidDerivedMetric, Alias: "derived_type_count"},
	{ID: solidAbstractionMetric, Alias: "abstraction_count"},
}

func (solidMetricProvider) Compute(context EvaluationContext) (MetricBatch, error) {
	batch := MetricBatch{Metrics: []MetricFact{}, Coverage: []QualityCoverage{}, Diagnostics: []QualityDiagnostic{}}
	for _, snapshot := range context.Input.SourceSnapshots {
		if status, ok := sourceCoverage(snapshot, solidStructureCapability); ok && status.Status != "" && status.Status != CoverageObserved {
			// An explicit unsupported/unknown source contract is authoritative;
			// do not turn an extractor's absence into a structural pass.
			continue
		}
		for _, symbol := range snapshot.Symbols {
			facts := structuralFacts(symbol, snapshot)
			for _, definition := range solidMetricDefinitions {
				value, ok := facts[definition.Alias]
				if !ok || value < 0 {
					continue
				}
				subject := EntityRef{Kind: "symbol", ID: symbol.ID, StableKey: sourceSymbolStableKey(symbol), SnapshotID: snapshot.SnapshotID, ScopeID: snapshot.ScopeID}
				batch.Metrics = append(batch.Metrics, MetricFact{
					ID:             sourceMetricID(definition.ID, snapshot, subject, definition.Alias),
					SubjectRef:     subject,
					MetricID:       definition.ID,
					Value:          MetricValue{Kind: ValueInteger, Value: value},
					Unit:           "unit:count",
					FormulaID:      "formula:source-solid." + strings.TrimPrefix(definition.Alias, ""),
					FormulaVersion: solidProviderVersion,
					Provenance:     solidMetricProvenance(snapshot, symbol),
					Extensions:     []ExtensionBlock{},
				})
			}
		}
	}
	return batch, nil
}

func structuralFacts(symbol SourceSymbol, snapshot SourceSnapshot) map[string]int {
	result := make(map[string]int)
	values := map[string]*int{
		"member_count":              symbol.MemberCount,
		"method_count":              symbol.MethodCount,
		"dependency_count":          symbol.DependencyCount,
		"concrete_dependency_count": symbol.ConcreteDependencyCount,
		"interface_method_count":    symbol.InterfaceMethodCount,
		"type_switch_count":         symbol.TypeSwitchCount,
		"hierarchy_depth":           symbol.HierarchyDepth,
		"derived_type_count":        symbol.DerivedTypeCount,
		"abstraction_count":         symbol.AbstractionCount,
	}
	for key, value := range values {
		if value != nil {
			result[key] = *value
		}
	}
	for key, value := range symbol.StructuralFacts {
		if value >= 0 {
			result[key] = value
		}
	}
	if _, exists := result["dependency_count"]; !exists {
		count := 0
		for _, relation := range snapshot.Relations {
			if relation.FromRef.Kind == "symbol" && relation.FromRef.ID == symbol.ID && relation.ToRef != nil {
				count++
			}
		}
		if count > 0 {
			result["dependency_count"] = count
		}
	}
	return result
}

func solidMetricProvenance(snapshot SourceSnapshot, symbol SourceSymbol) FactProvenance {
	value := symbol.Provenance
	if value.Status == "" || value.Basis == "" || value.Provider == "" || value.ProviderVersion == "" {
		value = FactProvenance{Status: "observed", Basis: "structural-facts", Provider: "provider:solid-structure", ProviderVersion: solidProviderVersion}
	}
	if value.EvidenceIDs == nil {
		value.EvidenceIDs = []string{}
	}
	value.EvidenceIDs = normalizeStrings(append(value.EvidenceIDs, scopeEvidenceID(snapshot.ScopeID, snapshot.SnapshotID)))
	return value
}

type solidSignalRule struct {
	id          string
	description string
	metrics     []string
	message     string
	limitation  string
}

func newSolidSignalRules() []QualityRule {
	return []QualityRule{
		solidSignalRule{id: "signal:solid.srp", description: "Advisory structural signal for types with many members and dependencies", metrics: []string{solidMemberMetric, solidDependencyMetric}, message: "observed member and dependency counts may indicate multiple structural responsibilities", limitation: "Static structure cannot prove responsibility boundaries or an SRP violation."},
		solidSignalRule{id: "signal:solid.ocp", description: "Advisory structural signal for repeated type-switch structure", metrics: []string{solidTypeSwitchMetric}, message: "observed type-switch count may indicate extension pressure in the current structure", limitation: "Static structure cannot prove extension intent or an OCP violation."},
		solidSignalRule{id: "signal:solid.lsp", description: "Advisory structural signal for hierarchy shape", metrics: []string{solidHierarchyMetric, solidDerivedMetric}, message: "observed hierarchy depth and derived-type count may warrant a substitutability review", limitation: "Hierarchy shape cannot prove behavioral substitutability or an LSP violation."},
		solidSignalRule{id: "signal:solid.isp", description: "Advisory structural signal for large interfaces", metrics: []string{solidInterfaceMetric}, message: "observed interface method count may indicate a broad abstraction", limitation: "Interface size cannot prove client-specific usage or an ISP violation."},
		solidSignalRule{id: "signal:solid.dip", description: "Advisory structural signal for concrete dependencies", metrics: []string{solidConcreteMetric}, message: "observed concrete dependency count may warrant an abstraction-boundary review", limitation: "Concrete dependency counts cannot prove abstraction intent or a DIP violation."},
	}
}

func (rule solidSignalRule) ID() string             { return rule.id }
func (rule solidSignalRule) Version() string        { return solidProviderVersion }
func (rule solidSignalRule) AssessmentKind() string { return AssessmentSignal }
func (rule solidSignalRule) RequiredCapabilities() []string {
	return []string{solidStructureCapability}
}
func (rule solidSignalRule) ParameterSchema() ParameterSchema {
	fields := map[string]ParameterField{
		"threshold":                     {Kind: "integer", Required: false, Minimum: floatPointer(0)},
		"minimum":                       {Kind: "integer", Required: false, Minimum: floatPointer(0)},
		"member_threshold":              {Kind: "integer", Required: false, Minimum: floatPointer(0)},
		"dependency_threshold":          {Kind: "integer", Required: false, Minimum: floatPointer(0)},
		"type_switch_threshold":         {Kind: "integer", Required: false, Minimum: floatPointer(0)},
		"hierarchy_depth_threshold":     {Kind: "integer", Required: false, Minimum: floatPointer(0)},
		"derived_type_threshold":        {Kind: "integer", Required: false, Minimum: floatPointer(0)},
		"interface_method_threshold":    {Kind: "integer", Required: false, Minimum: floatPointer(0)},
		"concrete_dependency_threshold": {Kind: "integer", Required: false, Minimum: floatPointer(0)},
	}
	return ParameterSchema{Namespace: "rule-config:solid-signal", SchemaVersion: solidProviderVersion, Fields: fields, AllowAdditional: false}
}
func (rule solidSignalRule) Descriptor() RuleDescriptor {
	return RuleDescriptor{ID: rule.ID(), Version: rule.Version(), AssessmentKind: rule.AssessmentKind(), RequiredCapabilities: rule.RequiredCapabilities(), ParameterSchema: rule.ParameterSchema(), DefaultSeverity: SeverityInfo, Description: rule.description, Limitations: []string{rule.limitation}}
}
func (solidSignalRule) DefaultParameters() TypedConfigBlock {
	return TypedConfigBlock{Namespace: "rule-config:solid-signal", SchemaVersion: solidProviderVersion, Payload: map[string]any{}}
}

func (rule solidSignalRule) Evaluate(context EvaluationContext, batch MetricBatch) (RuleResult, error) {
	result := RuleResult{Findings: []QualityFinding{}, Coverage: []QualityCoverage{}, Diagnostics: []QualityDiagnostic{}}
	for _, snapshot := range context.Input.SourceSnapshots {
		result = mergeSolidRuleResult(result, rule.evaluateSnapshot(context, batch, snapshot))
	}
	if len(context.Input.SourceSnapshots) == 0 {
		result.Coverage = append(result.Coverage, newCoverage(rule, CoverageUnsupported, "no source snapshot was supplied", 0, 0, "", "", rule.ID()))
	}
	return result, nil
}

func (rule solidSignalRule) evaluateSnapshot(context EvaluationContext, batch MetricBatch, snapshot SourceSnapshot) RuleResult {
	result := RuleResult{Findings: []QualityFinding{}, Coverage: []QualityCoverage{}, Diagnostics: []QualityDiagnostic{}}
	if status, ok := sourceCoverage(snapshot, solidStructureCapability); ok && status.Status != "" && status.Status != CoverageObserved {
		result.Coverage = append(result.Coverage, newCoverage(rule, status.Status, reasonOr(status.Reason, "structural signal facts are not available"), len(snapshot.Symbols), 0, snapshot.ScopeID, snapshot.SnapshotID, rule.ID()))
		return result
	}
	bySubject := make(map[string][]MetricFact)
	for _, metric := range batch.Metrics {
		if metric.SubjectRef.SnapshotID != snapshot.SnapshotID || !containsString(rule.metrics, metric.MetricID) {
			continue
		}
		key := entityKey(metric.SubjectRef)
		bySubject[key] = append(bySubject[key], metric)
	}
	if len(bySubject) == 0 {
		status := CoverageNotEvaluable
		reason := "no required structural facts were reported for this signal"
		if len(snapshot.Symbols) == 0 {
			reason = "no symbol subjects were reported for this signal"
		}
		result.Coverage = append(result.Coverage, newCoverage(rule, status, reason, len(snapshot.Symbols), 0, snapshot.ScopeID, snapshot.SnapshotID, rule.ID()))
		result.Diagnostics = append(result.Diagnostics, QualityDiagnostic{Code: "quality:solid-signal-not-evaluable", Message: "the SOLID signal was not evaluated because its required structural facts were not reported", Severity: SeverityInfo, RuleID: rule.ID(), RuleVersion: rule.Version(), Details: map[string]any{"scope_id": snapshot.ScopeID}})
		return result
	}
	thresholds, err := solidThresholds(context.RuleBinding, rule.id)
	if err != nil {
		return RuleResult{Diagnostics: []QualityDiagnostic{{Code: string(ErrorParameterInvalid), Message: err.Error(), Severity: SeverityWarning, RuleID: rule.ID(), RuleVersion: rule.Version()}}}
	}
	evaluated := 0
	for _, metrics := range bySubject {
		facts := make(map[string]MetricFact, len(metrics))
		for _, metric := range metrics {
			facts[metric.MetricID] = metric
		}
		if !hasAllSolidMetrics(facts, rule.metrics) {
			continue
		}
		evaluated++
		indicators, triggered := rule.triggered(facts, thresholds)
		if !triggered {
			continue
		}
		metricIDs := make([]string, 0, len(facts))
		for _, metric := range facts {
			metricIDs = append(metricIDs, metric.ID)
		}
		sort.Strings(metricIDs)
		subject := facts[rule.metrics[0]].SubjectRef
		result.Findings = append(result.Findings, QualityFinding{
			RuleID: rule.ID(), RuleVersion: rule.Version(), AssessmentKind: AssessmentSignal, Status: StatusActive, Severity: solidSeverity(context.RuleBinding), SubjectRef: subject,
			MessageCode: "quality:solid-structural-signal", Message: fmt.Sprintf("Structural signal for %s: %s.", subjectLabel(subject), rule.message), ObservedMetricIDs: metricIDs,
			Evidence:    FindingEvidence{SourceSpans: solidSubjectSpans(context.Input, subject), EntityRefs: []EntityRef{subject}, RelationRefs: []EntityRef{}, MetricRefs: metricIDs, DiagnosticRefs: []string{}},
			Limitations: []string{rule.limitation}, Provenance: FactProvenance{Status: "observed", Basis: "heuristic", EvidenceIDs: append([]string{}, metricIDs...), Provider: "rule:solid-signal", ProviderVersion: solidProviderVersion},
			Extensions: []ExtensionBlock{{Namespace: "quality:solid", SchemaVersion: solidProviderVersion, Capability: "indicators", Payload: map[string]any{"rule_id": rule.ID(), "thresholds": thresholds, "values": indicators}}},
		})
	}
	status := CoverageObserved
	reason := ""
	if evaluated == 0 {
		status = CoverageNotEvaluable
		reason = "structural facts were present but did not form a complete indicator set"
	} else if evaluated < len(bySubject) {
		status = CoveragePartial
		reason = fmt.Sprintf("%d of %d structural subjects had all required indicator facts", evaluated, len(bySubject))
	}
	result.Coverage = append(result.Coverage, newCoverage(rule, status, reason, len(bySubject), evaluated, snapshot.ScopeID, snapshot.SnapshotID, rule.ID()))
	return result
}

func (rule solidSignalRule) triggered(facts map[string]MetricFact, thresholds map[string]int64) (map[string]int64, bool) {
	values := make(map[string]int64, len(facts))
	for metricID, metric := range facts {
		value, ok := integerValue(metric.Value.Value)
		if ok {
			values[metricID] = value
		}
	}
	minimum := func(key string, fallback int64) int64 {
		if value, ok := thresholds[key]; ok {
			return value
		}
		return fallback
	}
	switch rule.id {
	case "signal:solid.srp":
		return values, values[solidMemberMetric] >= minimum("member_threshold", 10) && values[solidDependencyMetric] >= minimum("dependency_threshold", 3)
	case "signal:solid.ocp":
		return values, values[solidTypeSwitchMetric] >= minimum("type_switch_threshold", 1)
	case "signal:solid.lsp":
		return values, values[solidHierarchyMetric] >= minimum("hierarchy_depth_threshold", 2) && values[solidDerivedMetric] >= minimum("derived_type_threshold", 1)
	case "signal:solid.isp":
		return values, values[solidInterfaceMetric] >= minimum("interface_method_threshold", 8)
	case "signal:solid.dip":
		return values, values[solidConcreteMetric] >= minimum("concrete_dependency_threshold", 3)
	default:
		return values, false
	}
}

func solidThresholds(binding *RuleBinding, ruleID string) (map[string]int64, error) {
	result := map[string]int64{}
	if binding == nil {
		return result, nil
	}
	payload, ok := stringMap(binding.Parameters.Payload)
	if !ok {
		return nil, fmt.Errorf("SOLID signal configuration payload must be an object")
	}
	for _, key := range []string{"threshold", "minimum", "member_threshold", "dependency_threshold", "type_switch_threshold", "hierarchy_depth_threshold", "derived_type_threshold", "interface_method_threshold", "concrete_dependency_threshold"} {
		if raw, exists := payload[key]; exists {
			value, valid := integerValue(raw)
			if !valid || value < 0 {
				return nil, fmt.Errorf("SOLID signal threshold %q must be a non-negative integer", key)
			}
			result[key] = value
		}
	}
	if value, ok := result["threshold"]; ok {
		switch ruleID {
		case "signal:solid.srp":
			result["member_threshold"] = value
		case "signal:solid.ocp":
			result["type_switch_threshold"] = value
		case "signal:solid.lsp":
			result["hierarchy_depth_threshold"] = value
		case "signal:solid.isp":
			result["interface_method_threshold"] = value
		case "signal:solid.dip":
			result["concrete_dependency_threshold"] = value
		}
	}
	return result, nil
}

func hasAllSolidMetrics(values map[string]MetricFact, required []string) bool {
	for _, metricID := range required {
		metric, ok := values[metricID]
		if !ok || metric.Value.Kind != ValueInteger {
			return false
		}
	}
	return true
}

func solidSubjectSpans(input EvaluationInput, subject EntityRef) []SourceSpan {
	for _, snapshot := range input.SourceSnapshots {
		if snapshot.SnapshotID != subject.SnapshotID {
			continue
		}
		for _, symbol := range snapshot.Symbols {
			if symbol.ID == subject.ID {
				return symbolSpans(symbol)
			}
		}
	}
	return []SourceSpan{}
}

func subjectLabel(subject EntityRef) string {
	if subject.StableKey != "" {
		return subject.StableKey
	}
	return subject.ID
}

func solidSeverity(binding *RuleBinding) string {
	if binding != nil && binding.Severity != "" {
		return binding.Severity
	}
	return SeverityInfo
}

func mergeSolidRuleResult(left, right RuleResult) RuleResult {
	left.Findings = append(left.Findings, right.Findings...)
	left.Coverage = append(left.Coverage, right.Coverage...)
	left.Diagnostics = append(left.Diagnostics, right.Diagnostics...)
	return left
}

func reasonOr(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
