package quality

import (
	"fmt"
	"path"
	"strings"
)

const graphProviderID = "provider:canonical-graph"

type architectureMetricProvider struct{}

func newArchitectureMetricProvider() MetricProvider { return architectureMetricProvider{} }
func (architectureMetricProvider) ID() string       { return graphProviderID }
func (architectureMetricProvider) Version() string  { return qualityFormulaVersion }
func (architectureMetricProvider) Capabilities() []CapabilityDescriptor {
	return []CapabilityDescriptor{
		{ID: "architecture:relationships", Version: qualityFormulaVersion, Description: "Reported canonical module relationships."},
		{ID: "architecture:cycles", Version: qualityFormulaVersion, Description: "Canonical derived module cycles."},
		{ID: "architecture:module.efferent_coupling", Version: qualityFormulaVersion, Description: "Distinct reported target modules per module."},
		{ID: "architecture:module.afferent_coupling", Version: qualityFormulaVersion, Description: "Distinct reported source modules per module."},
	}
}

func (architectureMetricProvider) Compute(context EvaluationContext) (MetricBatch, error) {
	batch := MetricBatch{Metrics: []MetricFact{}, Coverage: []QualityCoverage{}, Diagnostics: []QualityDiagnostic{}}
	graph := context.Input.Architecture
	if graph == nil {
		return batch, nil
	}
	scopeID := graph.ScopeID
	if scopeID == "" {
		scopeID = "architecture"
	}
	targets := make(map[string]map[string]struct{}, len(graph.Modules))
	sources := make(map[string]map[string]struct{}, len(graph.Modules))
	targetEvidence := make(map[string][]string, len(graph.Modules))
	sourceEvidence := make(map[string][]string, len(graph.Modules))
	includeExternal := couplingIncludesExternal(context.Profile)
	for _, module := range graph.Modules {
		targets[module.ID] = map[string]struct{}{}
		sources[module.ID] = map[string]struct{}{}
	}
	for _, relationship := range graph.Relationships {
		if relationship.Type != "depends_on" {
			continue
		}
		if _, ok := targets[relationship.FromModuleID]; !ok {
			continue
		}
		if relationship.ToModuleID != "" {
			if _, ok := targets[relationship.ToModuleID]; !ok {
				continue
			}
			if relationship.FromModuleID == relationship.ToModuleID {
				continue
			}
			targets[relationship.FromModuleID][relationship.ToModuleID] = struct{}{}
			sources[relationship.ToModuleID][relationship.FromModuleID] = struct{}{}
			targetEvidence[relationship.FromModuleID] = append(targetEvidence[relationship.FromModuleID], relationship.ID)
			sourceEvidence[relationship.ToModuleID] = append(sourceEvidence[relationship.ToModuleID], relationship.ID)
			continue
		}
		if includeExternal && relationship.ToReferenceID != "" {
			targets[relationship.FromModuleID]["reference:"+relationship.ToReferenceID] = struct{}{}
			targetEvidence[relationship.FromModuleID] = append(targetEvidence[relationship.FromModuleID], relationship.ID)
		}
	}
	for _, module := range graph.Modules {
		stable := module.StableKey
		if stable == "" {
			stable = module.ID
		}
		subject := EntityRef{Kind: "module", ID: module.ID, StableKey: stable, ScopeID: scopeID}
		batch.Metrics = append(batch.Metrics,
			graphMetric("architecture:module.efferent_coupling", "formula:architecture.efferent-coupling", len(targets[module.ID]), subject, targetEvidence[module.ID], graphProviderID, "unit:module"),
			graphMetric("architecture:module.afferent_coupling", "formula:architecture.afferent-coupling", len(sources[module.ID]), subject, sourceEvidence[module.ID], graphProviderID, "unit:module"),
		)
	}
	cycleCounts := make(map[string]int, len(graph.Modules))
	cycleEvidence := make(map[string][]string, len(graph.Modules))
	for _, cycle := range graph.Cycles {
		for _, moduleID := range cycle.ModuleIDs {
			if _, exists := targets[moduleID]; !exists {
				continue
			}
			cycleCounts[moduleID]++
			cycleEvidence[moduleID] = append(cycleEvidence[moduleID], cycle.ID)
		}
	}
	for _, module := range graph.Modules {
		stable := moduleStableKey(module)
		subject := EntityRef{Kind: "module", ID: module.ID, StableKey: stable, ScopeID: scopeID}
		batch.Metrics = append(batch.Metrics, graphMetric("architecture:module.cycle_participation", "formula:architecture.cycle-participation", cycleCounts[module.ID], subject, cycleEvidence[module.ID], graphProviderID, "unit:cycle"))
	}
	return batch, nil
}

func graphMetric(metricID, formulaID string, value int, subject EntityRef, evidence []string, provider, unit string) MetricFact {
	evidence = normalizeStrings(evidence)
	return MetricFact{
		ID:             "metric:" + digestPart(metricID, subject.ScopeID, subject.ID),
		SubjectRef:     subject,
		MetricID:       metricID,
		Value:          MetricValue{Kind: ValueInteger, Value: value},
		Unit:           unit,
		FormulaID:      formulaID,
		FormulaVersion: qualityFormulaVersion,
		Provenance:     FactProvenance{Status: "observed", Basis: "graph", EvidenceIDs: evidence, Provider: provider, ProviderVersion: qualityFormulaVersion},
		Extensions:     []ExtensionBlock{},
	}
}

type graphThresholdRule struct {
	id           string
	metricID     string
	formulaID    string
	capability   string
	description  string
	defaultLimit int64
}

func (rule graphThresholdRule) ID() string                     { return rule.id }
func (rule graphThresholdRule) Version() string                { return qualityFormulaVersion }
func (rule graphThresholdRule) AssessmentKind() string         { return AssessmentExact }
func (rule graphThresholdRule) RequiredCapabilities() []string { return []string{rule.capability} }
func (rule graphThresholdRule) ParameterSchema() ParameterSchema {
	return moduleCouplingParameterSchema()
}
func (rule graphThresholdRule) DefaultParameters() TypedConfigBlock {
	return TypedConfigBlock{
		Namespace:     "rule-config:module-coupling",
		SchemaVersion: qualityFormulaVersion,
		Payload: map[string]any{
			"operator":        "greater_than",
			"limit":           rule.defaultLimit,
			"unit":            "unit:module",
			"external_policy": "exclude",
		},
	}
}
func (rule graphThresholdRule) Descriptor() RuleDescriptor {
	return RuleDescriptor{ID: rule.ID(), Version: rule.Version(), AssessmentKind: rule.AssessmentKind(), RequiredCapabilities: rule.RequiredCapabilities(), ParameterSchema: rule.ParameterSchema(), DefaultSeverity: SeverityWarning, Description: rule.description}
}
func newEfferentCouplingRule() QualityRule {
	return graphThresholdRule{id: "architecture:module.max-efferent-coupling", metricID: "architecture:module.efferent_coupling", formulaID: "formula:architecture.efferent-coupling", capability: "architecture:relationships", description: "Checks how many different modules this module depends on. A high count can make changes ripple across many parts of the system.", defaultLimit: 10}
}
func newAfferentCouplingRule() QualityRule {
	return graphThresholdRule{id: "architecture:module.max-afferent-coupling", metricID: "architecture:module.afferent_coupling", formulaID: "formula:architecture.afferent-coupling", capability: "architecture:relationships", description: "Checks how many different modules depend on this module. A high count means changes here may affect many consumers.", defaultLimit: 10}
}
func (rule graphThresholdRule) Evaluate(context EvaluationContext, batch MetricBatch) (RuleResult, error) {
	threshold, err := readThreshold(context.RuleBinding, "rule-config:module-coupling", "unit:module")
	if err != nil {
		return RuleResult{}, err
	}
	result := RuleResult{Findings: []QualityFinding{}, Coverage: []QualityCoverage{}, Diagnostics: []QualityDiagnostic{}}
	graph := context.Input.Architecture
	if graph == nil {
		result.Coverage = append(result.Coverage, newCoverage(rule, CoverageUnsupported, "canonical architecture graph was not supplied", 0, 0, "", "", graphProviderID))
		return result, nil
	}
	if !architectureScopeIsExplicit(context, graph) {
		result.Coverage = append(result.Coverage, newCoverage(rule, CoverageNotEvaluable, "multiple source scopes require an explicit aggregate architecture graph", 0, 0, effectiveGraphScope(graph), "", graphProviderID))
		result.Diagnostics = append(result.Diagnostics, QualityDiagnostic{Code: "quality:architecture-scope-missing", Message: "architecture coupling was not evaluated across independent source scopes without an explicit aggregate graph", Severity: SeverityWarning, RuleID: rule.ID(), RuleVersion: rule.Version()})
		return result, nil
	}
	if graph.Relationships == nil {
		result.Coverage = append(result.Coverage, newCoverage(rule, CoverageNotEvaluable, "canonical relationship projection was not supplied", len(graph.Modules), 0, effectiveGraphScope(graph), "", graphProviderID))
		result.Diagnostics = append(result.Diagnostics, QualityDiagnostic{Code: string(ErrorMetricUnavailable), Message: "architecture coupling requires an explicit relationship collection", Severity: SeverityWarning, RuleID: rule.ID(), RuleVersion: rule.Version()})
		return result, nil
	}
	modules := make(map[string]ArchitectureModule, len(graph.Modules))
	for _, module := range graph.Modules {
		modules[module.ID] = module
	}
	evaluated := 0
	for _, metric := range batch.Metrics {
		if metric.MetricID != rule.metricID || metric.FormulaID != rule.formulaID || metric.SubjectRef.ScopeID != effectiveGraphScope(graph) || metric.FormulaVersion != qualityFormulaVersion || metric.Unit != threshold.unit || metric.Provenance.Status != "observed" {
			continue
		}
		module, ok := modules[metric.SubjectRef.ID]
		if !ok {
			continue
		}
		value, ok := integerValue(metric.Value.Value)
		if !ok {
			continue
		}
		evaluated++
		if !compareThreshold(value, threshold.limit, threshold.operator) {
			continue
		}
		entity := EntityRef{Kind: "module", ID: module.ID, StableKey: moduleStableKey(module), ScopeID: effectiveGraphScope(graph)}
		relationRefs := make([]EntityRef, 0, len(metric.Provenance.EvidenceIDs))
		for _, id := range metric.Provenance.EvidenceIDs {
			relationRefs = append(relationRefs, EntityRef{Kind: "relationship", ID: id, ScopeID: effectiveGraphScope(graph)})
		}
		severity := SeverityWarning
		if context.RuleBinding != nil && context.RuleBinding.Severity != "" {
			severity = context.RuleBinding.Severity
		}
		result.Findings = append(result.Findings, QualityFinding{RuleID: rule.ID(), RuleVersion: rule.Version(), AssessmentKind: AssessmentExact, Status: StatusActive, Severity: severity, SubjectRef: entity, MessageCode: "quality:coupling-exceeded", Message: fmt.Sprintf("module %s has %d distinct module coupling targets/sources; configured limit is %s %d", module.Name, value, threshold.operator, threshold.limit), ObservedMetricIDs: []string{metric.ID}, Comparison: &Comparison{Operator: threshold.operator, ObservedMetricID: metric.ID, Limit: MetricValue{Kind: ValueInteger, Value: threshold.limit}, Unit: threshold.unit}, Evidence: FindingEvidence{SourceSpans: []SourceSpan{}, EntityRefs: []EntityRef{entity}, RelationRefs: relationRefs, MetricRefs: []string{metric.ID}, DiagnosticRefs: []string{}}, Provenance: metric.Provenance, Extensions: []ExtensionBlock{}})
	}
	status, reason := CoverageObserved, ""
	if len(graph.Modules) > 0 && evaluated == 0 {
		status, reason = CoverageNotEvaluable, "no compatible architecture coupling metric was observed"
	} else if evaluated < len(graph.Modules) {
		status, reason = CoveragePartial, fmt.Sprintf("%d of %d modules had an observed coupling metric", evaluated, len(graph.Modules))
	}
	result.Coverage = append(result.Coverage, newCoverage(rule, status, reason, len(graph.Modules), evaluated, effectiveGraphScope(graph), "", graphProviderID))
	return result, nil
}

type cycleRule struct{}

func newCycleRule() QualityRule                  { return cycleRule{} }
func (cycleRule) ID() string                     { return "architecture:no-cycles" }
func (cycleRule) Version() string                { return qualityFormulaVersion }
func (cycleRule) AssessmentKind() string         { return AssessmentExact }
func (cycleRule) RequiredCapabilities() []string { return []string{"architecture:cycles"} }
func (cycleRule) ParameterSchema() ParameterSchema {
	return ParameterSchema{Namespace: "rule-config:no-cycles", SchemaVersion: qualityFormulaVersion, Fields: map[string]ParameterField{}, AllowAdditional: true}
}
func (rule cycleRule) Descriptor() RuleDescriptor {
	return RuleDescriptor{ID: rule.ID(), Version: rule.Version(), AssessmentKind: rule.AssessmentKind(), RequiredCapabilities: rule.RequiredCapabilities(), ParameterSchema: rule.ParameterSchema(), DefaultSeverity: SeverityError, Description: "Checks whether dependencies form a loop, such as A → B → A. Loops make ownership and change order harder to understand."}
}
func (cycleRule) DefaultParameters() TypedConfigBlock {
	return TypedConfigBlock{Namespace: "rule-config:no-cycles", SchemaVersion: qualityFormulaVersion, Payload: map[string]any{}}
}
func (rule cycleRule) Evaluate(context EvaluationContext, _ MetricBatch) (RuleResult, error) {
	result := RuleResult{Findings: []QualityFinding{}, Coverage: []QualityCoverage{}, Diagnostics: []QualityDiagnostic{}}
	graph := context.Input.Architecture
	if graph == nil {
		result.Coverage = append(result.Coverage, newCoverage(rule, CoverageUnsupported, "canonical cycle projection was not supplied", 0, 0, "", "", graphProviderID))
		return result, nil
	}
	if !architectureScopeIsExplicit(context, graph) {
		result.Coverage = append(result.Coverage, newCoverage(rule, CoverageNotEvaluable, "multiple source scopes require an explicit aggregate architecture graph", 0, 0, effectiveGraphScope(graph), "", graphProviderID))
		result.Diagnostics = append(result.Diagnostics, QualityDiagnostic{Code: "quality:architecture-scope-missing", Message: "canonical cycles were not evaluated across independent source scopes without an explicit aggregate graph", Severity: SeverityWarning, RuleID: rule.ID(), RuleVersion: rule.Version()})
		return result, nil
	}
	if graph.Relationships == nil {
		result.Coverage = append(result.Coverage, newCoverage(rule, CoverageNotEvaluable, "canonical relationship projection was not supplied", 0, 0, effectiveGraphScope(graph), "", graphProviderID))
		result.Diagnostics = append(result.Diagnostics, QualityDiagnostic{Code: string(ErrorMetricUnavailable), Message: "canonical cycle evaluation requires an explicit relationship collection", Severity: SeverityWarning, RuleID: rule.ID(), RuleVersion: rule.Version()})
		return result, nil
	}
	if graph.Cycles == nil {
		result.Coverage = append(result.Coverage, newCoverage(rule, CoverageNotEvaluable, "canonical cycle projection was not supplied", 0, 0, effectiveGraphScope(graph), "", graphProviderID))
		result.Diagnostics = append(result.Diagnostics, QualityDiagnostic{Code: string(ErrorMetricUnavailable), Message: "architecture cycle evaluation requires an explicit cycle collection", Severity: SeverityWarning, RuleID: rule.ID(), RuleVersion: rule.Version()})
		return result, nil
	}
	scopeID := effectiveGraphScope(graph)
	modules := make(map[string]ArchitectureModule, len(graph.Modules))
	relationships := make(map[string]struct{}, len(graph.Relationships))
	for _, module := range graph.Modules {
		modules[module.ID] = module
	}
	for _, relationship := range graph.Relationships {
		relationships[relationship.ID] = struct{}{}
	}
	validCycles := 0
	seenCycles := make(map[string]struct{}, len(graph.Cycles))
	for _, cycle := range graph.Cycles {
		if cycle.ID == "" {
			result.Diagnostics = append(result.Diagnostics, QualityDiagnostic{Code: string(ErrorMetricUnavailable), Message: "canonical cycle projection contains a cycle without an identity", Severity: SeverityWarning, RuleID: rule.ID(), RuleVersion: rule.Version()})
			continue
		}
		if _, seen := seenCycles[cycle.ID]; seen {
			continue
		}
		seenCycles[cycle.ID] = struct{}{}
		valid := len(cycle.ModuleIDs) > 0 && len(cycle.RelationshipIDs) > 0 && uniqueValues(cycle.ModuleIDs) && uniqueValues(cycle.RelationshipIDs)
		for _, id := range cycle.ModuleIDs {
			if _, exists := modules[id]; !exists {
				valid = false
				break
			}
		}
		if valid {
			for _, id := range cycle.RelationshipIDs {
				if _, exists := relationships[id]; !exists {
					valid = false
					break
				}
			}
		}
		if !valid {
			result.Diagnostics = append(result.Diagnostics, QualityDiagnostic{Code: string(ErrorMetricUnavailable), Message: "canonical cycle projection references an unknown module or relationship", Severity: SeverityWarning, RuleID: rule.ID(), RuleVersion: rule.Version(), Details: map[string]any{"cycle_id": cycle.ID}})
			continue
		}
		validCycles++
		moduleRefs := make([]EntityRef, 0, len(cycle.ModuleIDs))
		for _, id := range cycle.ModuleIDs {
			moduleRefs = append(moduleRefs, EntityRef{Kind: "module", ID: id, StableKey: moduleStableKey(modules[id]), ScopeID: scopeID})
		}
		relationRefs := make([]EntityRef, 0, len(cycle.RelationshipIDs))
		for _, id := range cycle.RelationshipIDs {
			relationRefs = append(relationRefs, EntityRef{Kind: "relationship", ID: id, ScopeID: scopeID})
		}
		subjectID := cycle.ID
		if subjectID == "" {
			subjectID = digestPart(strings.Join(cycle.ModuleIDs, "\x00"))
		}
		severity := SeverityError
		if context.RuleBinding != nil && context.RuleBinding.Severity != "" {
			severity = context.RuleBinding.Severity
		}
		result.Findings = append(result.Findings, QualityFinding{RuleID: rule.ID(), RuleVersion: rule.Version(), AssessmentKind: AssessmentExact, Status: StatusActive, Severity: severity, SubjectRef: EntityRef{Kind: "cycle", ID: subjectID, StableKey: subjectID, ScopeID: scopeID}, MessageCode: "quality:canonical-cycle", Message: fmt.Sprintf("canonical architecture graph contains a cycle involving %d module(s)", len(cycle.ModuleIDs)), Evidence: FindingEvidence{SourceSpans: []SourceSpan{}, EntityRefs: moduleRefs, RelationRefs: relationRefs, MetricRefs: []string{}, DiagnosticRefs: []string{}}, Provenance: FactProvenance{Status: "observed", Basis: "graph", EvidenceIDs: append([]string{}, cycle.RelationshipIDs...), Provider: graphProviderID, ProviderVersion: qualityFormulaVersion}, Extensions: []ExtensionBlock{}})
	}
	status := CoverageObserved
	reason := ""
	if len(result.Diagnostics) > 0 {
		status = CoveragePartial
		reason = "one or more canonical cycle projections could not be evaluated"
	}
	result.Coverage = append(result.Coverage, newCoverage(rule, status, reason, len(graph.Cycles), validCycles, scopeID, "", graphProviderID))
	return result, nil
}

type forbiddenDependencyRule struct{}

func newForbiddenDependencyRule() QualityRule          { return forbiddenDependencyRule{} }
func (forbiddenDependencyRule) ID() string             { return "architecture:forbidden-dependency" }
func (forbiddenDependencyRule) Version() string        { return qualityFormulaVersion }
func (forbiddenDependencyRule) AssessmentKind() string { return AssessmentExact }
func (forbiddenDependencyRule) RequiredCapabilities() []string {
	return []string{"architecture:relationships"}
}
func (forbiddenDependencyRule) ParameterSchema() ParameterSchema {
	return constraintBindingSchema("rule-config:architecture-constraint")
}
func (rule forbiddenDependencyRule) Descriptor() RuleDescriptor {
	return RuleDescriptor{ID: rule.ID(), Version: rule.Version(), AssessmentKind: rule.AssessmentKind(), RequiredCapabilities: rule.RequiredCapabilities(), ParameterSchema: rule.ParameterSchema(), DefaultSeverity: SeverityError, Description: "Checks only dependency edges that you explicitly marked as forbidden. The profile chooses which modules or patterns are forbidden; this check does not invent that policy."}
}
func (forbiddenDependencyRule) DefaultParameters() TypedConfigBlock {
	return TypedConfigBlock{Namespace: "rule-config:architecture-constraint", SchemaVersion: qualityFormulaVersion, Payload: map[string]any{}}
}
func (rule forbiddenDependencyRule) Evaluate(context EvaluationContext, _ MetricBatch) (RuleResult, error) {
	result := RuleResult{Findings: []QualityFinding{}, Coverage: []QualityCoverage{}, Diagnostics: []QualityDiagnostic{}}
	return evaluateForbiddenConstraints(context, rule, result)
}

type layerDirectionRule struct{}

func newLayerDirectionRule() QualityRule          { return layerDirectionRule{} }
func (layerDirectionRule) ID() string             { return "architecture:layer-direction" }
func (layerDirectionRule) Version() string        { return qualityFormulaVersion }
func (layerDirectionRule) AssessmentKind() string { return AssessmentExact }
func (layerDirectionRule) RequiredCapabilities() []string {
	return []string{"architecture:relationships"}
}
func (layerDirectionRule) ParameterSchema() ParameterSchema {
	return constraintBindingSchema("rule-config:architecture-constraint")
}
func (rule layerDirectionRule) Descriptor() RuleDescriptor {
	return RuleDescriptor{ID: rule.ID(), Version: rule.Version(), AssessmentKind: rule.AssessmentKind(), RequiredCapabilities: rule.RequiredCapabilities(), ParameterSchema: rule.ParameterSchema(), DefaultSeverity: SeverityError, Description: "Checks whether dependencies follow the layer direction you explicitly configured. It reports only edges that cross a configured layer boundary the wrong way."}
}
func (layerDirectionRule) DefaultParameters() TypedConfigBlock {
	return TypedConfigBlock{Namespace: "rule-config:architecture-constraint", SchemaVersion: qualityFormulaVersion, Payload: map[string]any{}}
}
func (rule layerDirectionRule) Evaluate(context EvaluationContext, _ MetricBatch) (RuleResult, error) {
	result := RuleResult{Findings: []QualityFinding{}, Coverage: []QualityCoverage{}, Diagnostics: []QualityDiagnostic{}}
	graph := context.Input.Architecture
	if graph == nil {
		result.Coverage = append(result.Coverage, newCoverage(rule, CoverageUnsupported, "canonical architecture graph was not supplied", 0, 0, "", "", graphProviderID))
		return result, nil
	}
	if !architectureScopeIsExplicit(context, graph) {
		result.Coverage = append(result.Coverage, newCoverage(rule, CoverageNotEvaluable, "multiple source scopes require an explicit aggregate architecture graph", 0, 0, effectiveGraphScope(graph), "", graphProviderID))
		result.Diagnostics = append(result.Diagnostics, QualityDiagnostic{Code: "quality:architecture-scope-missing", Message: "layer direction was not evaluated across independent source scopes without an explicit aggregate graph", Severity: SeverityWarning, RuleID: rule.ID(), RuleVersion: rule.Version()})
		return result, nil
	}
	if graph.Relationships == nil {
		result.Coverage = append(result.Coverage, newCoverage(rule, CoverageNotEvaluable, "canonical relationship projection was not supplied", 0, 0, effectiveGraphScope(graph), "", graphProviderID))
		result.Diagnostics = append(result.Diagnostics, QualityDiagnostic{Code: string(ErrorMetricUnavailable), Message: "layer-direction evaluation requires an explicit relationship collection", Severity: SeverityWarning, RuleID: rule.ID(), RuleVersion: rule.Version()})
		return result, nil
	}
	constraints := constraintsOfKind(context.Profile, "layer_direction")
	if len(constraints) == 0 {
		result.Coverage = append(result.Coverage, newCoverage(rule, CoverageNotEvaluable, "no explicit layer-direction constraint is configured", 0, 0, effectiveGraphScope(graph), "", "quality:constraint"))
		result.Diagnostics = append(result.Diagnostics, QualityDiagnostic{Code: "quality:layer-policy-missing", Message: "layer direction cannot be inferred without an explicit policy", Severity: SeverityInfo, RuleID: rule.ID(), RuleVersion: rule.Version()})
		return result, nil
	}
	layers := moduleLayers(graph)
	if len(layers) == 0 {
		result.Coverage = append(result.Coverage, newCoverage(rule, CoverageNotEvaluable, "canonical graph has no explicit layer assignments", len(graph.Relationships), 0, effectiveGraphScope(graph), "", graphProviderID))
		result.Diagnostics = append(result.Diagnostics, QualityDiagnostic{Code: string(ErrorMetricUnavailable), Message: "layer-direction evaluation requires explicit layer assignments for the reported modules", Severity: SeverityWarning, RuleID: rule.ID(), RuleVersion: rule.Version()})
		return result, nil
	}
	evaluated := 0
	evaluatedRelationships := make(map[string]struct{}, len(graph.Relationships))
	missingAssignments := 0
	applicableConstraints := 0
	for _, constraint := range constraints {
		payload, ok := stringMap(constraint.Parameters.Payload)
		if !ok {
			result.Diagnostics = append(result.Diagnostics, constraintDiagnostic(rule, constraint, "constraint payload must be an object"))
			continue
		}
		direction := stringValue(payload, "direction", "allowed_direction")
		fromLayer, fromOK := integerField(payload, "from_layer")
		toLayer, toOK := integerField(payload, "to_layer")
		if direction == "" && (!fromOK || !toOK) {
			result.Diagnostics = append(result.Diagnostics, constraintDiagnostic(rule, constraint, "layer constraint requires direction or exact from_layer/to_layer selectors"))
			continue
		}
		if !constraintAppliesToGraph(constraint, graph) {
			continue
		}
		applicableConstraints++
		for _, relationship := range graph.Relationships {
			if relationship.Type != "depends_on" || relationship.ToModuleID == "" {
				continue
			}
			from, fromExists := layers[relationship.FromModuleID]
			to, toExists := layers[relationship.ToModuleID]
			if !fromExists || !toExists {
				missingAssignments++
				continue
			}
			fromSelector, hasFromSelector := selectorFromPayload(payload, "from")
			toSelector, hasToSelector := selectorFromPayload(payload, "to")
			if hasFromSelector && !fromSelector.matchesModuleID(relationship.FromModuleID, graph) || hasToSelector && !toSelector.matchesModuleID(relationship.ToModuleID, graph) {
				continue
			}
			if _, seen := evaluatedRelationships[relationship.ID]; !seen {
				evaluatedRelationships[relationship.ID] = struct{}{}
				evaluated++
			}
			if fromOK && int64(from) != fromLayer || toOK && int64(to) != toLayer {
				continue
			}
			violation := false
			if direction == "" && fromOK && toOK {
				violation = true
			}
			if direction == "lower_to_higher" || direction == "upward" {
				violation = violation || from > to
			}
			if direction == "higher_to_lower" || direction == "downward" {
				violation = violation || from < to
			}
			if direction == "same" {
				violation = violation || from != to
			}
			if !violation {
				continue
			}
			result.Findings = append(result.Findings, constraintFinding(rule, constraint, relationship, graph, fmt.Sprintf("dependency from layer %d to layer %d violates explicit layer policy", from, to), context.RuleBinding))
		}
	}
	status := CoverageObserved
	reason := ""
	if applicableConstraints == 0 {
		status, reason = CoverageNotEvaluable, "no configured layer-direction policy applies to this graph scope"
	} else if len(result.Diagnostics) > 0 {
		status, reason = CoveragePartial, "one or more layer constraints could not be evaluated"
	} else if missingAssignments > 0 {
		status, reason = CoveragePartial, fmt.Sprintf("%d reported relationships have no explicit layer assignment", missingAssignments)
		result.Diagnostics = append(result.Diagnostics, QualityDiagnostic{Code: string(ErrorMetricUnavailable), Message: "layer-direction coverage is partial because some reported relationships lack explicit layer assignments", Severity: SeverityWarning, RuleID: rule.ID(), RuleVersion: rule.Version(), Details: map[string]any{"missing_assignments": missingAssignments}})
	}
	result.Coverage = append(result.Coverage, newCoverage(rule, status, reason, len(graph.Relationships), evaluated, effectiveGraphScope(graph), "", graphProviderID))
	return result, nil
}

func evaluateForbiddenConstraints(context EvaluationContext, rule forbiddenDependencyRule, result RuleResult) (RuleResult, error) {
	graph := context.Input.Architecture
	if graph == nil {
		result.Coverage = append(result.Coverage, newCoverage(rule, CoverageUnsupported, "canonical architecture graph was not supplied", 0, 0, "", "", graphProviderID))
		return result, nil
	}
	if !architectureScopeIsExplicit(context, graph) {
		result.Coverage = append(result.Coverage, newCoverage(rule, CoverageNotEvaluable, "multiple source scopes require an explicit aggregate architecture graph", 0, 0, effectiveGraphScope(graph), "", graphProviderID))
		result.Diagnostics = append(result.Diagnostics, QualityDiagnostic{Code: "quality:architecture-scope-missing", Message: "forbidden dependencies were not evaluated across independent source scopes without an explicit aggregate graph", Severity: SeverityWarning, RuleID: rule.ID(), RuleVersion: rule.Version()})
		return result, nil
	}
	if graph.Relationships == nil {
		result.Coverage = append(result.Coverage, newCoverage(rule, CoverageNotEvaluable, "canonical relationship projection was not supplied", 0, 0, effectiveGraphScope(graph), "", graphProviderID))
		result.Diagnostics = append(result.Diagnostics, QualityDiagnostic{Code: string(ErrorMetricUnavailable), Message: "forbidden-dependency evaluation requires an explicit relationship collection", Severity: SeverityWarning, RuleID: rule.ID(), RuleVersion: rule.Version()})
		return result, nil
	}
	constraints := constraintsOfKind(context.Profile, "forbidden_dependency")
	if len(constraints) == 0 {
		result.Coverage = append(result.Coverage, newCoverage(rule, CoverageNotEvaluable, "no explicit forbidden-dependency constraint is configured", 0, 0, effectiveGraphScope(graph), "", "quality:constraint"))
		result.Diagnostics = append(result.Diagnostics, QualityDiagnostic{Code: "quality:forbidden-policy-missing", Message: "forbidden dependencies cannot be inferred without explicit policy", Severity: SeverityInfo, RuleID: rule.ID(), RuleVersion: rule.Version()})
		return result, nil
	}
	evaluated := 0
	evaluatedRelationships := make(map[string]struct{}, len(graph.Relationships))
	for _, constraint := range constraints {
		payload, ok := stringMap(constraint.Parameters.Payload)
		if !ok {
			result.Diagnostics = append(result.Diagnostics, constraintDiagnostic(rule, constraint, "constraint payload must be an object"))
			continue
		}
		fromSelector, fromOK := selectorFromPayload(payload, "from")
		toSelector, toOK := selectorFromPayload(payload, "to")
		if !fromOK || !toOK {
			result.Diagnostics = append(result.Diagnostics, constraintDiagnostic(rule, constraint, "forbidden dependency requires explicit from and to selectors"))
			continue
		}
		if !constraintAppliesToGraph(constraint, graph) {
			continue
		}
		for _, relationship := range graph.Relationships {
			if relationship.Type != "depends_on" || relationship.ToModuleID == "" {
				continue
			}
			from, fromExists := moduleByID(graph, relationship.FromModuleID)
			to, toExists := moduleByID(graph, relationship.ToModuleID)
			if !fromExists || !toExists {
				continue
			}
			if _, seen := evaluatedRelationships[relationship.ID]; !seen {
				evaluatedRelationships[relationship.ID] = struct{}{}
				evaluated++
			}
			if !fromSelector.matches(from, graph) || !toSelector.matches(to, graph) {
				continue
			}
			result.Findings = append(result.Findings, constraintFinding(rule, constraint, relationship, graph, fmt.Sprintf("dependency from %s to %s matches an explicit forbidden-dependency policy", from.Name, to.Name), context.RuleBinding))
		}
	}
	status := CoverageObserved
	reason := ""
	if len(result.Diagnostics) > 0 {
		status, reason = CoveragePartial, "one or more forbidden-dependency constraints could not be evaluated"
	}
	if evaluated == 0 && len(constraints) > 0 && len(result.Diagnostics) == 0 {
		status, reason = CoverageNotEvaluable, "explicit forbidden-dependency policy did not match this graph scope"
	}
	result.Coverage = append(result.Coverage, newCoverage(rule, status, reason, len(graph.Relationships), evaluated, effectiveGraphScope(graph), "", graphProviderID))
	return result, nil
}

type moduleSelector struct {
	IDs      map[string]struct{}
	Tags     map[string]struct{}
	Layers   map[string]struct{}
	Patterns []string
}

func (selector moduleSelector) matchesModuleID(id string, graph *ArchitectureModel) bool {
	module, ok := moduleByID(graph, id)
	return ok && selector.matches(module, graph)
}

func selectorFromPayload(payload map[string]any, prefix string) (moduleSelector, bool) {
	selector := moduleSelector{IDs: map[string]struct{}{}, Tags: map[string]struct{}{}, Layers: map[string]struct{}{}, Patterns: []string{}}
	for _, key := range []string{prefix + "_module_ids", prefix + "_ids"} {
		for _, value := range stringValues(payload[key]) {
			selector.IDs[value] = struct{}{}
		}
	}
	for _, key := range []string{prefix + "_tags", prefix + "_tag"} {
		for _, value := range stringValues(payload[key]) {
			selector.Tags[value] = struct{}{}
		}
	}
	for _, key := range []string{prefix + "_module_patterns", prefix + "_stable_key_globs", prefix + "_patterns"} {
		for _, value := range stringValues(payload[key]) {
			if strings.TrimSpace(value) != "" {
				selector.Patterns = append(selector.Patterns, value)
			}
		}
	}
	selector.Patterns = normalizeStrings(selector.Patterns)
	for _, key := range []string{prefix + "_layers", prefix + "_layer"} {
		for _, value := range integerValues(payload[key]) {
			selector.Layers[fmt.Sprint(value)] = struct{}{}
		}
	}
	return selector, len(selector.IDs) > 0 || len(selector.Tags) > 0 || len(selector.Layers) > 0 || len(selector.Patterns) > 0
}

func (selector moduleSelector) matches(module ArchitectureModule, graph *ArchitectureModel) bool {
	if len(selector.IDs) == 0 && len(selector.Tags) == 0 && len(selector.Layers) == 0 && len(selector.Patterns) == 0 {
		return false
	}
	if _, ok := selector.IDs[module.ID]; ok {
		return true
	}
	for _, tag := range module.Tags {
		if _, ok := selector.Tags[tag]; ok {
			return true
		}
	}
	stableKey := moduleStableKey(module)
	for _, pattern := range selector.Patterns {
		if matched, err := path.Match(pattern, stableKey); err == nil && matched {
			return true
		}
	}
	if layer, ok := moduleLayers(graph)[module.ID]; ok {
		_, ok = selector.Layers[fmt.Sprint(layer)]
		return ok
	}
	return false
}

func constraintFinding(rule QualityRule, constraint ArchitectureConstraint, relationship ArchitectureRelationship, graph *ArchitectureModel, message string, binding *RuleBinding) QualityFinding {
	scope := effectiveGraphScope(graph)
	from, _ := moduleByID(graph, relationship.FromModuleID)
	to, _ := moduleByID(graph, relationship.ToModuleID)
	fromRef := EntityRef{Kind: "module", ID: from.ID, StableKey: moduleStableKey(from), ScopeID: scope}
	toRef := EntityRef{Kind: "module", ID: to.ID, StableKey: moduleStableKey(to), ScopeID: scope}
	relRef := EntityRef{Kind: "relationship", ID: relationship.ID, StableKey: relationship.ID, ScopeID: scope}
	provenance := constraint.Provenance
	if provenance.Status == "" {
		provenance.Status = "observed"
	}
	if provenance.Basis == "" {
		provenance.Basis = "configuration"
	}
	if provenance.Provider == "" {
		provenance.Provider = "quality:constraint"
	}
	if provenance.ProviderVersion == "" {
		provenance.ProviderVersion = qualityFormulaVersion
	}
	if provenance.EvidenceIDs == nil {
		provenance.EvidenceIDs = []string{}
	}
	severity := SeverityError
	if binding != nil && binding.Severity != "" {
		severity = binding.Severity
	}
	return QualityFinding{RuleID: rule.ID(), RuleVersion: rule.Version(), AssessmentKind: AssessmentExact, Status: StatusActive, Severity: severity, SubjectRef: relRef, MessageCode: "quality:architecture-constraint", Message: message, Evidence: FindingEvidence{SourceSpans: []SourceSpan{}, EntityRefs: []EntityRef{fromRef, toRef}, RelationRefs: []EntityRef{relRef}, MetricRefs: []string{}, DiagnosticRefs: []string{}}, Provenance: provenance, Extensions: []ExtensionBlock{{Namespace: "quality:constraint", SchemaVersion: qualityFormulaVersion, Capability: "constraint-id", Payload: constraint.ID}}}
}

func constraintDiagnostic(rule QualityRule, constraint ArchitectureConstraint, message string) QualityDiagnostic {
	return QualityDiagnostic{Code: string(ErrorConstraintInvalid), Message: message, Severity: SeverityWarning, RuleID: rule.ID(), RuleVersion: rule.Version(), Details: map[string]any{"constraint_id": constraint.ID}}
}

func constraintsOfKind(profile QualityProfile, kind string) []ArchitectureConstraint {
	result := make([]ArchitectureConstraint, 0)
	for _, constraint := range profile.Constraints {
		if constraint.Kind == kind {
			result = append(result, constraint)
		}
	}
	return result
}

func architectureScopeIsExplicit(context EvaluationContext, graph *ArchitectureModel) bool {
	return graph != nil && (len(context.Input.SourceSnapshots) <= 1 || graph.Aggregate)
}

func constraintAppliesToGraph(constraint ArchitectureConstraint, graph *ArchitectureModel) bool {
	payload, ok := stringMap(constraint.Parameters.Payload)
	if !ok {
		return false
	}
	scopeID, ok := payload["scope_id"].(string)
	if !ok || strings.TrimSpace(scopeID) == "" {
		return true
	}
	return scopeID == effectiveGraphScope(graph)
}

func moduleCouplingParameterSchema() ParameterSchema {
	schema := thresholdParameterSchema("rule-config:module-coupling", "unit:module")
	schema.Fields["external_policy"] = ParameterField{Kind: "string", Required: false, AllowedValues: []string{"exclude", "include"}}
	return schema
}

func couplingIncludesExternal(profile QualityProfile) bool {
	for _, binding := range profile.EnabledRules {
		if binding.RuleID != "architecture:module.max-efferent-coupling" && binding.RuleID != "architecture:module.max-afferent-coupling" {
			continue
		}
		payload, ok := stringMap(binding.Parameters.Payload)
		if !ok {
			continue
		}
		if value, ok := payload["external_policy"].(string); ok && value == "include" {
			return true
		}
	}
	return false
}

func moduleByID(graph *ArchitectureModel, id string) (ArchitectureModule, bool) {
	for _, module := range graph.Modules {
		if module.ID == id {
			return module, true
		}
	}
	return ArchitectureModule{}, false
}
func moduleStableKey(module ArchitectureModule) string {
	if module.StableKey != "" {
		return module.StableKey
	}
	return module.ID
}
func effectiveGraphScope(graph *ArchitectureModel) string {
	if graph == nil || graph.ScopeID == "" {
		return "architecture"
	}
	return graph.ScopeID
}

func moduleLayers(graph *ArchitectureModel) map[string]int {
	result := make(map[string]int)
	for _, layer := range graph.Layers {
		for _, id := range layer.ModuleIDs {
			result[id] = layer.Layer
		}
	}
	return result
}

func uniqueValues(values []string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" {
			return false
		}
		if _, exists := seen[value]; exists {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func constraintBindingSchema(namespace string) ParameterSchema {
	return ParameterSchema{Namespace: namespace, SchemaVersion: qualityFormulaVersion, Fields: map[string]ParameterField{}, AllowAdditional: true}
}

func stringValue(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := payload[key].(string); ok {
			return value
		}
	}
	return ""
}
func integerField(payload map[string]any, key string) (int64, bool) {
	value, exists := payload[key]
	if !exists {
		return 0, false
	}
	return integerValue(value)
}
func stringValues(value any) []string {
	if value == nil {
		return nil
	}
	if text, ok := value.(string); ok {
		return []string{text}
	}
	values, _ := stringSlice(value)
	return values
}
func integerValues(value any) []int64 {
	if value == nil {
		return nil
	}
	if integer, ok := integerValue(value); ok {
		return []int64{integer}
	}
	if integers, ok := integerSlice(value); ok {
		return integers
	}
	reflected, ok := stringSlice(value)
	if ok {
		result := make([]int64, 0, len(reflected))
		for _, text := range reflected {
			var parsed int64
			if _, err := fmt.Sscan(text, &parsed); err == nil {
				result = append(result, parsed)
			}
		}
		return result
	}
	return nil
}
