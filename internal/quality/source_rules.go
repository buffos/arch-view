package quality

import (
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

const (
	fileLineMetricID      = "source:file.line_count"
	callableBodyMetricID  = "source:callable.body_line_count"
	complexityMetricID    = "source:callable.cyclomatic_complexity"
	nestingMetricID       = "source:callable.max_nesting_depth"
	documentationMetricID = "source:documentation.coverage"
	metricProviderID      = "provider:source-index"
	metricProviderVersion = "1.0.0"
	qualityFormulaVersion = "1.0.0"
)

type sourceMetricProvider struct{}

func newSourceMetricProvider() MetricProvider { return sourceMetricProvider{} }
func (sourceMetricProvider) ID() string       { return metricProviderID }
func (sourceMetricProvider) Version() string  { return metricProviderVersion }

func (sourceMetricProvider) Capabilities() []CapabilityDescriptor {
	return []CapabilityDescriptor{
		{ID: fileLineMetricID, Version: qualityFormulaVersion, Description: "Intrinsic physical source-file line count."},
		{ID: callableBodyMetricID, Version: qualityFormulaVersion, Description: "Physical lines in an extractor-supplied callable body span."},
		{ID: complexityMetricID, Version: qualityFormulaVersion, Description: "Provider-declared callable cyclomatic complexity."},
		{ID: nestingMetricID, Version: qualityFormulaVersion, Description: "Provider-declared callable maximum nesting depth."},
		{ID: "source:public-symbol.documentation", Version: qualityFormulaVersion, Description: "Observed visibility and documentation status for public symbols."},
	}
}

func (sourceMetricProvider) Compute(context EvaluationContext) (MetricBatch, error) {
	batch := MetricBatch{Metrics: []MetricFact{}, Coverage: []QualityCoverage{}, Diagnostics: []QualityDiagnostic{}}
	for _, snapshot := range context.Input.SourceSnapshots {
		for _, file := range snapshot.Files {
			subject := EntityRef{Kind: "file", ID: file.ID, StableKey: sourceFileStableKey(file), SnapshotID: snapshot.SnapshotID, ScopeID: snapshot.ScopeID}
			batch.Metrics = append(batch.Metrics, MetricFact{
				ID:             sourceMetricID(fileLineMetricID, snapshot, subject, file.ID),
				SubjectRef:     subject,
				MetricID:       fileLineMetricID,
				Value:          MetricValue{Kind: ValueInteger, Value: file.LineCount},
				Unit:           "unit:line",
				FormulaID:      "formula:source-file.line-count",
				FormulaVersion: qualityFormulaVersion,
				Provenance:     sourceProvenance(snapshot, file.Provenance, "intrinsic"),
				Extensions:     []ExtensionBlock{},
			})
		}

		documented, supported, unknown := 0, 0, 0
		documentationBySubject := documentationStatusBySubject(snapshot.Documentation)
		for _, symbol := range snapshot.Symbols {
			documentation := documentationBySubject[symbol.ID]
			if symbol.Visibility == "unknown" || symbol.Visibility == "" || symbol.VisibilityStatus == "unknown" || symbol.VisibilityStatus == "unsupported" {
				unknown++
			} else if symbol.Visibility == "public" {
				switch documentation {
				case "present":
					documented++
					supported++
				case "absent":
					supported++
				default:
					unknown++
				}
			}
			if !isCallable(symbol) {
				continue
			}
			subject := EntityRef{Kind: "symbol", ID: symbol.ID, StableKey: sourceSymbolStableKey(symbol), SnapshotID: snapshot.SnapshotID, ScopeID: snapshot.ScopeID}
			if bodySpan, ok := callableBodySpan(snapshot, symbol); ok {
				lineCount := bodySpan.End.Line - bodySpan.Start.Line + 1
				if lineCount < 1 {
					lineCount = 1
				}
				batch.Metrics = append(batch.Metrics, MetricFact{
					ID:             sourceMetricID(callableBodyMetricID, snapshot, subject, symbol.ID),
					SubjectRef:     subject,
					MetricID:       callableBodyMetricID,
					Value:          MetricValue{Kind: ValueInteger, Value: lineCount},
					Unit:           "unit:line",
					FormulaID:      "formula:source-callable.body-line-count",
					FormulaVersion: qualityFormulaVersion,
					Provenance:     sourceProvenance(snapshot, symbol.Provenance, "extractor-body-span"),
					Extensions: []ExtensionBlock{{
						Namespace: "quality:source-metric", SchemaVersion: qualityFormulaVersion,
						Capability: "body-span", Payload: bodySpan,
					}},
				})
			}
			if metric := firstMetric(snapshot.Metrics, complexityMetricID, symbol.ID, snapshot.ScopeID, snapshot.SnapshotID); metric != nil {
				batch.Metrics = append(batch.Metrics, normalizeSourceMetric(*metric, snapshot, subject))
			}
			if metric := firstMetric(snapshot.Metrics, nestingMetricID, symbol.ID, snapshot.ScopeID, snapshot.SnapshotID); metric != nil {
				batch.Metrics = append(batch.Metrics, normalizeSourceMetric(*metric, snapshot, subject))
			}
		}
		coverageMetric := MetricFact{
			ID:             sourceMetricID(documentationMetricID, snapshot, EntityRef{Kind: "scope", ID: snapshot.ScopeID, StableKey: snapshot.ScopeID, SnapshotID: snapshot.SnapshotID, ScopeID: snapshot.ScopeID}, snapshot.ScopeID),
			SubjectRef:     EntityRef{Kind: "scope", ID: snapshot.ScopeID, StableKey: snapshot.ScopeID, SnapshotID: snapshot.SnapshotID, ScopeID: snapshot.ScopeID},
			MetricID:       documentationMetricID,
			Value:          MetricValue{Kind: ValueDecimal, Value: documentationRatio(documented, supported)},
			FormulaID:      "formula:source-documentation.coverage",
			FormulaVersion: qualityFormulaVersion,
			Provenance:     FactProvenance{Status: coverageStatusForDocumentation(supported, unknown), Basis: "source-index", EvidenceIDs: []string{scopeEvidenceID(snapshot.ScopeID, snapshot.SnapshotID)}, Provider: metricProviderID, ProviderVersion: metricProviderVersion},
			Extensions: []ExtensionBlock{{
				Namespace: "quality:documentation", SchemaVersion: qualityFormulaVersion, Capability: "counts",
				Payload: map[string]any{"documented": documented, "supported": supported, "unknown": unknown},
			}},
		}
		batch.Metrics = append(batch.Metrics, coverageMetric)
	}
	return batch, nil
}

type thresholdRule struct {
	id             string
	capability     string
	metricID       string
	formulaID      string
	formulaVersion string
	namespace      string
	description    string
	unit           string
	defaultLimit   int64
}

func (rule thresholdRule) ID() string                     { return rule.id }
func (rule thresholdRule) Version() string                { return qualityFormulaVersion }
func (rule thresholdRule) AssessmentKind() string         { return AssessmentExact }
func (rule thresholdRule) RequiredCapabilities() []string { return []string{rule.capability} }
func (rule thresholdRule) ParameterSchema() ParameterSchema {
	return thresholdParameterSchema(rule.namespace, rule.unit)
}
func (rule thresholdRule) DefaultParameters() TypedConfigBlock {
	return TypedConfigBlock{
		Namespace:     rule.namespace,
		SchemaVersion: qualityFormulaVersion,
		Payload: map[string]any{
			"operator": "greater_than",
			"limit":    rule.defaultLimit,
			"unit":     rule.unit,
		},
	}
}
func (rule thresholdRule) Descriptor() RuleDescriptor {
	return RuleDescriptor{ID: rule.id, Version: rule.Version(), AssessmentKind: rule.AssessmentKind(), RequiredCapabilities: rule.RequiredCapabilities(), ParameterSchema: rule.ParameterSchema(), DefaultSeverity: SeverityWarning, Description: rule.description}
}

func newFileSizeRule() QualityRule {
	return thresholdRule{id: "source:file.max-lines", capability: fileLineMetricID, metricID: fileLineMetricID, formulaID: "formula:source-file.line-count", formulaVersion: qualityFormulaVersion, namespace: "rule-config:source-file-size", description: "Flag files whose intrinsic physical line count exceeds the configured limit.", unit: "unit:line", defaultLimit: 500}
}

func newCallableSizeRule() QualityRule {
	return thresholdRule{id: "source:callable.max-lines", capability: callableBodyMetricID, metricID: callableBodyMetricID, formulaID: "formula:source-callable.body-line-count", formulaVersion: qualityFormulaVersion, namespace: "rule-config:source-callable-size", description: "Flag callables whose extractor-declared body span exceeds the configured limit.", unit: "unit:line", defaultLimit: 50}
}

func newComplexityRule() QualityRule {
	return thresholdRule{id: "source:callable.max-cyclomatic-complexity", capability: complexityMetricID, metricID: complexityMetricID, formulaVersion: qualityFormulaVersion, namespace: "rule-config:source-callable-complexity", description: "Flag callables using a provider-declared cyclomatic-complexity formula.", unit: "unit:complexity", defaultLimit: 10}
}

func newNestingRule() QualityRule {
	return thresholdRule{id: "source:callable.max-nesting-depth", capability: nestingMetricID, metricID: nestingMetricID, formulaVersion: qualityFormulaVersion, namespace: "rule-config:source-callable-nesting", description: "Flag callables using a provider-declared maximum-nesting formula.", unit: "unit:depth", defaultLimit: 4}
}

func (rule thresholdRule) Evaluate(context EvaluationContext, batch MetricBatch) (RuleResult, error) {
	threshold, err := readThreshold(context.RuleBinding, rule.namespace, rule.unit)
	if err != nil {
		return RuleResult{}, err
	}
	result := RuleResult{Findings: []QualityFinding{}, Coverage: []QualityCoverage{}, Diagnostics: []QualityDiagnostic{}}
	severity := SeverityWarning
	if context.RuleBinding != nil && context.RuleBinding.Severity != "" {
		severity = context.RuleBinding.Severity
	}
	for _, snapshot := range context.Input.SourceSnapshots {
		metrics := metricsForSnapshot(batch.Metrics, snapshot, rule.metricID, rule.formulaID, rule.formulaVersion, threshold.unit)
		if rule.metricID == fileLineMetricID {
			result = mergeRuleResult(result, evaluateFileThreshold(rule, threshold, snapshot, metrics, severity))
			continue
		}
		result = mergeRuleResult(result, evaluateCallableThreshold(rule, threshold, snapshot, metrics, severity))
	}
	if len(context.Input.SourceSnapshots) == 0 {
		result.Coverage = append(result.Coverage, newCoverage(rule, CoverageUnsupported, "no source snapshot was supplied", 0, 0, "", "", metricProviderID))
	}
	return result, nil
}

type threshold struct {
	operator string
	limit    int64
	unit     string
}

func readThreshold(binding *RuleBinding, namespace, unit string) (threshold, error) {
	if binding == nil {
		return threshold{}, newQualityError(ErrorParameterInvalid, "quality rule binding is missing", nil)
	}
	if binding.Parameters.Namespace != namespace || binding.Parameters.SchemaVersion != qualityFormulaVersion {
		return threshold{}, newQualityError(ErrorParameterInvalid, "threshold configuration does not match the rule schema", map[string]any{"namespace": binding.Parameters.Namespace})
	}
	payload, ok := stringMap(binding.Parameters.Payload)
	if !ok {
		return threshold{}, newQualityError(ErrorParameterInvalid, "threshold configuration payload must be an object", nil)
	}
	operator, ok := payload["operator"].(string)
	if !ok || !validOperator(operator) {
		return threshold{}, newQualityError(ErrorParameterInvalid, "threshold operator is invalid", map[string]any{"operator": payload["operator"]})
	}
	limit, ok := integerValue(payload["limit"])
	if !ok || limit < 0 {
		return threshold{}, newQualityError(ErrorParameterInvalid, "threshold limit must be a non-negative integer", map[string]any{"limit": payload["limit"]})
	}
	configuredUnit, ok := payload["unit"].(string)
	if !ok || configuredUnit != unit {
		return threshold{}, newQualityError(ErrorParameterInvalid, "threshold unit is incompatible with the metric", map[string]any{"expected": unit, "actual": payload["unit"]})
	}
	return threshold{operator: operator, limit: limit, unit: configuredUnit}, nil
}

func thresholdParameterSchema(namespace, unit string) ParameterSchema {
	return ParameterSchema{Namespace: namespace, SchemaVersion: qualityFormulaVersion, Fields: map[string]ParameterField{
		"operator": {Kind: "string", Required: true, AllowedValues: []string{OperatorEqual, OperatorGreaterThan, OperatorGreaterThanOrEqual, OperatorLessThan, OperatorLessThanOrEqual, OperatorNotEqual}},
		"limit":    {Kind: "integer", Required: true, Minimum: floatPointer(0)},
		"unit":     {Kind: "string", Required: true, AllowedValues: []string{unit}},
	}, AllowAdditional: false}
}

func validOperator(value string) bool {
	switch value {
	case OperatorGreaterThan, OperatorGreaterThanOrEqual, OperatorLessThan, OperatorLessThanOrEqual, OperatorEqual, OperatorNotEqual:
		return true
	default:
		return false
	}
}

func compareThreshold(observed, limit int64, operator string) bool {
	switch operator {
	case OperatorGreaterThan:
		return observed > limit
	case OperatorGreaterThanOrEqual:
		return observed >= limit
	case OperatorLessThan:
		return observed < limit
	case OperatorLessThanOrEqual:
		return observed <= limit
	case OperatorEqual:
		return observed == limit
	case OperatorNotEqual:
		return observed != limit
	default:
		return false
	}
}

func evaluateFileThreshold(rule thresholdRule, threshold threshold, snapshot SourceSnapshot, metrics []MetricFact, severity string) RuleResult {
	result := RuleResult{Findings: []QualityFinding{}, Coverage: []QualityCoverage{}, Diagnostics: []QualityDiagnostic{}}
	fileByID := make(map[string]SourceFile, len(snapshot.Files))
	for _, file := range snapshot.Files {
		fileByID[file.ID] = file
	}
	status, reason := sourceRuleCoverage(snapshot, "source:size", len(snapshot.Files), len(metrics))
	if status == CoverageUnknown || status == CoverageUnsupported || status == CoverageNotEvaluable {
		result.Coverage = append(result.Coverage, newCoverage(rule, status, reason, len(snapshot.Files), 0, snapshot.ScopeID, snapshot.SnapshotID, metricProviderID))
		return result
	}
	evaluated := 0
	for _, metric := range metrics {
		file, ok := fileByID[metric.SubjectRef.ID]
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
		result.Findings = append(result.Findings, exactThresholdFinding(rule, metric, EntityRef{Kind: "file", ID: file.ID, StableKey: sourceFileStableKey(file), SnapshotID: snapshot.SnapshotID, ScopeID: snapshot.ScopeID}, value, threshold, []SourceSpan{}, severity))
	}
	status, reason = sourceRuleCoverage(snapshot, "source:size", len(snapshot.Files), evaluated)
	if len(snapshot.Files) == 0 && status == CoverageObserved {
		reason = "no eligible source files"
	}
	result.Coverage = append(result.Coverage, newCoverage(rule, status, reason, len(snapshot.Files), evaluated, snapshot.ScopeID, snapshot.SnapshotID, metricProviderID))
	return result
}

func evaluateCallableThreshold(rule thresholdRule, threshold threshold, snapshot SourceSnapshot, metrics []MetricFact, severity string) RuleResult {
	result := RuleResult{Findings: []QualityFinding{}, Coverage: []QualityCoverage{}, Diagnostics: []QualityDiagnostic{}}
	callables := make(map[string]SourceSymbol)
	for _, symbol := range snapshot.Symbols {
		if isCallable(symbol) {
			callables[symbol.ID] = symbol
		}
	}
	inputStatus, inputReason := sourceCoverageStatus(snapshot, "source:callable.metrics")
	if inputStatus == CoverageUnknown || inputStatus == CoverageUnsupported || inputStatus == CoverageNotEvaluable {
		result.Coverage = append(result.Coverage, newCoverage(rule, inputStatus, inputReason, len(callables), 0, snapshot.ScopeID, snapshot.SnapshotID, metricProviderID))
		return result
	}
	evaluated := 0
	for _, metric := range metrics {
		symbol, ok := callables[metric.SubjectRef.ID]
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
		spans := []SourceSpan{}
		if bodySpan, ok := callableBodySpan(snapshot, symbol); ok {
			spans = append(spans, bodySpan)
		}
		result.Findings = append(result.Findings, exactThresholdFinding(rule, metric, EntityRef{Kind: "symbol", ID: symbol.ID, StableKey: sourceSymbolStableKey(symbol), SnapshotID: snapshot.SnapshotID, ScopeID: snapshot.ScopeID}, value, threshold, spans, severity))
	}
	missing := 0
	if rule.metricID == callableBodyMetricID {
		for _, symbol := range callables {
			if _, ok := callableBodySpan(snapshot, symbol); !ok {
				missing++
			}
		}
		if missing > 0 {
			result.Diagnostics = append(result.Diagnostics, QualityDiagnostic{Code: string(ErrorMetricUnavailable), Message: "callable body size was not evaluated because the extractor supplied no valid body span", Severity: SeverityWarning, RuleID: rule.ID(), RuleVersion: rule.Version(), Details: map[string]any{"missing_body_spans": missing, "scope_id": snapshot.ScopeID}})
		}
	}
	status := inputStatus
	reason := inputReason
	if status == "" {
		status = CoverageObserved
	}
	if len(callables) == 0 {
		status = CoverageObserved
		reason = "no eligible callable symbols"
	} else if evaluated == 0 {
		status = CoverageNotEvaluable
		reason = "no compatible callable metric was observed"
	} else if evaluated < len(callables) || missing > 0 || status == CoveragePartial {
		status = CoveragePartial
		if reason == "" {
			reason = fmt.Sprintf("%d of %d callable subjects had a compatible metric", evaluated, len(callables))
		}
	}
	result.Coverage = append(result.Coverage, newCoverage(rule, status, reason, len(callables), evaluated, snapshot.ScopeID, snapshot.SnapshotID, metricProviderID))
	return result
}

func exactThresholdFinding(rule thresholdRule, metric MetricFact, subject EntityRef, observed int64, threshold threshold, spans []SourceSpan, severity string) QualityFinding {
	if severity == "" {
		severity = SeverityWarning
	}
	messageCode := "quality:threshold-exceeded"
	if rule.ID() == "source:file.max-lines" {
		messageCode = "quality:file-lines-exceeded"
	}
	return QualityFinding{
		RuleID:            rule.ID(),
		RuleVersion:       rule.Version(),
		AssessmentKind:    AssessmentExact,
		Status:            StatusActive,
		Severity:          severity,
		SubjectRef:        subject,
		MessageCode:       messageCode,
		Message:           thresholdFindingMessage(rule, observed, threshold),
		ObservedMetricIDs: []string{metric.ID},
		Comparison:        &Comparison{Operator: threshold.operator, ObservedMetricID: metric.ID, Limit: MetricValue{Kind: ValueInteger, Value: threshold.limit}, Unit: threshold.unit},
		Evidence:          FindingEvidence{SourceSpans: spans, EntityRefs: []EntityRef{subject}, RelationRefs: []EntityRef{}, MetricRefs: []string{metric.ID}, DiagnosticRefs: []string{}},
		Provenance:        metric.Provenance,
		Extensions:        []ExtensionBlock{},
	}
}

func thresholdFindingMessage(rule thresholdRule, observed int64, threshold threshold) string {
	subject := "Subject"
	metric := strings.TrimPrefix(threshold.unit, "unit:")
	switch rule.ID() {
	case "source:file.max-lines":
		subject, metric = "File", "lines"
	case "source:callable.max-lines":
		subject, metric = "Callable", "lines"
	case "source:callable.max-cyclomatic-complexity":
		subject, metric = "Callable", "complexity"
	case "source:callable.max-nesting-depth":
		subject, metric = "Callable", "nesting depth"
	}
	switch threshold.operator {
	case OperatorGreaterThan:
		return fmt.Sprintf("%s has %d %s; configured maximum is %d", subject, observed, metric, threshold.limit)
	case OperatorGreaterThanOrEqual:
		return fmt.Sprintf("%s has %d %s; configured minimum is %d", subject, observed, metric, threshold.limit)
	case OperatorLessThan:
		return fmt.Sprintf("%s has %d %s; configured condition is less than %d", subject, observed, metric, threshold.limit)
	case OperatorLessThanOrEqual:
		return fmt.Sprintf("%s has %d %s; configured maximum is %d or less", subject, observed, metric, threshold.limit)
	case OperatorEqual:
		return fmt.Sprintf("%s has %d %s; configured value is %d", subject, observed, metric, threshold.limit)
	case OperatorNotEqual:
		return fmt.Sprintf("%s has %d %s; configured value must not be %d", subject, observed, metric, threshold.limit)
	default:
		return fmt.Sprintf("%s has %d %s; configured limit is %d", subject, observed, metric, threshold.limit)
	}
}

type documentationRule struct{}

func newDocumentationRule() QualityRule          { return documentationRule{} }
func (documentationRule) ID() string             { return "source:public-symbol.documentation" }
func (documentationRule) Version() string        { return qualityFormulaVersion }
func (documentationRule) AssessmentKind() string { return AssessmentExact }
func (documentationRule) RequiredCapabilities() []string {
	return []string{"source:public-symbol.documentation"}
}
func (documentationRule) ParameterSchema() ParameterSchema {
	return ParameterSchema{Namespace: "rule-config:documentation-coverage", SchemaVersion: qualityFormulaVersion, Fields: map[string]ParameterField{}, AllowAdditional: true}
}
func (rule documentationRule) Descriptor() RuleDescriptor {
	return RuleDescriptor{ID: rule.ID(), Version: rule.Version(), AssessmentKind: rule.AssessmentKind(), RequiredCapabilities: rule.RequiredCapabilities(), ParameterSchema: rule.ParameterSchema(), DefaultSeverity: SeverityWarning, Description: "Report public symbols whose documentation presence is observed as absent."}
}
func (documentationRule) DefaultParameters() TypedConfigBlock {
	return TypedConfigBlock{Namespace: "rule-config:documentation-coverage", SchemaVersion: qualityFormulaVersion, Payload: map[string]any{}}
}

func (rule documentationRule) Evaluate(context EvaluationContext, batch MetricBatch) (RuleResult, error) {
	result := RuleResult{Findings: []QualityFinding{}, Coverage: []QualityCoverage{}, Diagnostics: []QualityDiagnostic{}}
	severity := SeverityWarning
	if context.RuleBinding != nil && context.RuleBinding.Severity != "" {
		severity = context.RuleBinding.Severity
	}
	for _, snapshot := range context.Input.SourceSnapshots {
		if status, reason, unavailable := documentationCoverageUnavailable(snapshot); unavailable {
			result.Coverage = append(result.Coverage, newCoverage(rule, status, reason, len(snapshot.Symbols), 0, snapshot.ScopeID, snapshot.SnapshotID, metricProviderID))
			continue
		}
		metric := firstMetric(batch.Metrics, documentationMetricID, snapshot.ScopeID, snapshot.ScopeID, snapshot.SnapshotID)
		docs := documentationStatusBySubject(snapshot.Documentation)
		eligible, evaluated, documented, observedPublic := 0, 0, 0, 0
		unknownVisibility, unsupportedDocumentation := 0, 0
		for _, symbol := range snapshot.Symbols {
			if symbol.Visibility == "unknown" || symbol.Visibility == "" || symbol.VisibilityStatus == "unknown" || symbol.VisibilityStatus == "unsupported" {
				unknownVisibility++
				continue
			}
			if symbol.Visibility != "public" {
				continue
			}
			if symbol.VisibilityStatus != "observed" {
				unknownVisibility++
				continue
			}
			observedPublic++
			status := docs[symbol.ID]
			switch status {
			case "present":
				eligible++
				documented++
				evaluated++
			case "absent":
				eligible++
				evaluated++
				result.Findings = append(result.Findings, QualityFinding{
					RuleID: rule.ID(), RuleVersion: rule.Version(), AssessmentKind: AssessmentExact, Status: StatusActive, Severity: severity,
					SubjectRef:  EntityRef{Kind: "symbol", ID: symbol.ID, StableKey: sourceSymbolStableKey(symbol), SnapshotID: snapshot.SnapshotID, ScopeID: snapshot.ScopeID},
					MessageCode: "quality:public-symbol-undocumented", Message: fmt.Sprintf("public symbol %s has no reported documentation", symbol.Name),
					ObservedMetricIDs: metricIDs(metric), Evidence: FindingEvidence{SourceSpans: symbolSpans(symbol), EntityRefs: []EntityRef{{Kind: "symbol", ID: symbol.ID, StableKey: sourceSymbolStableKey(symbol), SnapshotID: snapshot.SnapshotID, ScopeID: snapshot.ScopeID}}, RelationRefs: []EntityRef{}, MetricRefs: metricIDs(metric), DiagnosticRefs: []string{}},
					Provenance: symbol.Provenance, Extensions: []ExtensionBlock{},
				})
			case "unknown", "":
				unsupportedDocumentation++
			default:
				unsupportedDocumentation++
			}
		}
		status := CoverageObserved
		reason := ""
		if evaluated == 0 && observedPublic == 0 && unknownVisibility > 0 {
			status = CoverageUnknown
			reason = "visibility was unknown or unsupported for all public candidates"
		} else if observedPublic == 0 {
			status = CoverageObserved
			reason = "no eligible public symbols"
		} else if evaluated == 0 {
			status = CoverageNotEvaluable
			reason = "documentation status was not observed for eligible public symbols"
		} else if evaluated < observedPublic || unknownVisibility > 0 {
			status = CoveragePartial
			reason = fmt.Sprintf("%d public symbols have observed documentation status; %d remain unknown or unsupported", evaluated, observedPublic-evaluated+unknownVisibility)
		}
		if unknownVisibility > 0 || unsupportedDocumentation > 0 {
			result.Diagnostics = append(result.Diagnostics, QualityDiagnostic{Code: "quality:documentation-coverage-partial", Message: "documentation coverage excludes subjects whose visibility or documentation status is unknown/unsupported", Severity: SeverityInfo, RuleID: rule.ID(), RuleVersion: rule.Version(), Details: map[string]any{"unknown_visibility": unknownVisibility, "unknown_or_unsupported_documentation": unsupportedDocumentation, "documented": documented, "observed_candidates": observedPublic, "observed_denominator": evaluated}})
		}
		coverage := newCoverage(rule, status, reason, eligible, evaluated, snapshot.ScopeID, snapshot.SnapshotID, metricProviderID)
		coverage.AvailableCapabilities = []string{"source:documentation", "source:visibility"}
		result.Coverage = append(result.Coverage, coverage)
	}
	return result, nil
}

func documentationCoverageUnavailable(snapshot SourceSnapshot) (string, string, bool) {
	for _, capability := range []string{"source:visibility", "source:documentation"} {
		coverage, ok := sourceCoverage(snapshot, capability)
		if !ok {
			continue
		}
		switch coverage.Status {
		case CoverageUnsupported, CoverageUnknown, CoverageNotEvaluable, CoverageAbsent:
			status := coverage.Status
			if status == CoverageAbsent {
				status = CoverageNotEvaluable
			}
			reason := coverage.Reason
			if reason == "" {
				reason = capability + " is unavailable"
			}
			return status, reason, true
		}
	}
	return "", "", false
}

func metricIDs(metric *MetricFact) []string {
	if metric == nil {
		return []string{}
	}
	return []string{metric.ID}
}

func mergeRuleResult(left, right RuleResult) RuleResult {
	left.Findings = append(left.Findings, right.Findings...)
	left.Coverage = append(left.Coverage, right.Coverage...)
	left.Diagnostics = append(left.Diagnostics, right.Diagnostics...)
	return left
}

func metricsForSnapshot(metrics []MetricFact, snapshot SourceSnapshot, metricID, formulaID, formulaVersion, unit string) []MetricFact {
	result := make([]MetricFact, 0)
	for _, metric := range metrics {
		if metric.MetricID != metricID {
			continue
		}
		if formulaID != "" && metric.FormulaID != formulaID {
			continue
		}
		if metric.SubjectRef.ScopeID != snapshot.ScopeID || metric.SubjectRef.SnapshotID != snapshot.SnapshotID {
			continue
		}
		if metric.Provenance.Status != "observed" {
			continue
		}
		if formulaVersion != "" && metric.FormulaVersion != formulaVersion {
			continue
		}
		if unit != "" && metric.Unit != unit {
			continue
		}
		result = append(result, metric)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func firstMetric(metrics []MetricFact, metricID, subjectID, scopeID, snapshotID string) *MetricFact {
	for index := range metrics {
		if metrics[index].MetricID == metricID && metrics[index].SubjectRef.ID == subjectID && (scopeID == "" || metrics[index].SubjectRef.ScopeID == scopeID) && (snapshotID == "" || metrics[index].SubjectRef.SnapshotID == snapshotID) {
			return &metrics[index]
		}
	}
	return nil
}

func normalizeSourceMetric(metric MetricFact, snapshot SourceSnapshot, subject EntityRef) MetricFact {
	metric.SubjectRef = subject
	metric.ID = sourceMetricID(metric.MetricID, snapshot, subject, metric.ID)
	if metric.Extensions == nil {
		metric.Extensions = []ExtensionBlock{}
	}
	if metric.Provenance.Status == "" {
		metric.Provenance.Status = "observed"
	}
	if metric.Provenance.Basis == "" {
		metric.Provenance.Basis = "syntax"
	}
	if metric.Provenance.Provider == "" {
		metric.Provenance.Provider = "extractor:source-index"
	}
	if metric.Provenance.ProviderVersion == "" {
		metric.Provenance.ProviderVersion = qualityFormulaVersion
	}
	if metric.Provenance.EvidenceIDs == nil {
		metric.Provenance.EvidenceIDs = []string{}
	}
	return metric
}

func sourceMetricID(metricID string, snapshot SourceSnapshot, subject EntityRef, salt string) string {
	return "metric:" + digestPart(metricID, snapshot.ScopeID, snapshot.SnapshotID, subject.Kind, subject.ID, salt)
}

func sourceProvenance(snapshot SourceSnapshot, source FactProvenance, basis string) FactProvenance {
	if source.Status == "" {
		source.Status = "observed"
	}
	if source.Basis == "" {
		source.Basis = basis
	}
	if source.Provider == "" {
		source.Provider = metricProviderID
	}
	if source.ProviderVersion == "" {
		source.ProviderVersion = metricProviderVersion
	}
	if source.EvidenceIDs == nil {
		source.EvidenceIDs = []string{}
	}
	if len(source.EvidenceIDs) == 0 {
		source.EvidenceIDs = []string{scopeEvidenceID(snapshot.ScopeID, snapshot.SnapshotID)}
	}
	return source
}

func sourceRuleCoverage(snapshot SourceSnapshot, capability string, subjects, evaluated int) (string, string) {
	status, reason := sourceCoverageStatus(snapshot, capability)
	if status == "" {
		status = CoverageObserved
	}
	if evaluated < subjects && status == CoverageObserved {
		status = CoveragePartial
		reason = fmt.Sprintf("%d of %d subjects had an observed metric", evaluated, subjects)
	}
	if subjects > 0 && evaluated == 0 && status == CoverageObserved {
		status = CoverageNotEvaluable
		reason = "required source metric was not observed"
	}
	return status, reason
}

func sourceCoverageStatus(snapshot SourceSnapshot, capability string) (string, string) {
	source, ok := sourceCoverage(snapshot, capability)
	if !ok {
		return CoverageNotEvaluable, "source-index did not report coverage for the required capability"
	}
	switch source.Status {
	case CoverageUnsupported:
		return CoverageUnsupported, source.Reason
	case CoverageUnknown:
		return CoverageUnknown, source.Reason
	case CoveragePartial:
		return CoveragePartial, source.Reason
	case CoverageAbsent:
		return CoverageNotEvaluable, "source-index does not contain the required source capability"
	case CoverageNotEvaluable:
		return CoverageNotEvaluable, source.Reason
	default:
		return CoverageObserved, source.Reason
	}
}

func sourceCoverage(snapshot SourceSnapshot, capability string) (SourceCoverage, bool) {
	for _, value := range snapshot.Coverage {
		if value.Capability == capability {
			return value, true
		}
	}
	return SourceCoverage{}, false
}

func documentationStatusBySubject(values []SourceDocumentation) map[string]string {
	result := make(map[string]string, len(values))
	for _, value := range values {
		if _, exists := result[value.SubjectRef.ID]; exists {
			continue
		}
		result[value.SubjectRef.ID] = value.Status
	}
	return result
}

func documentationRatio(documented, supported int) float64 {
	if supported == 0 {
		return 0
	}
	return float64(documented) / float64(supported)
}

func coverageStatusForDocumentation(supported, unknown int) string {
	if supported == 0 && unknown > 0 {
		return CoverageNotEvaluable
	}
	if unknown > 0 {
		return CoveragePartial
	}
	return CoverageObserved
}

func isCallable(symbol SourceSymbol) bool {
	return symbol.Category == "callable" || strings.HasPrefix(symbol.LanguageKind, "go:function") || strings.HasPrefix(symbol.LanguageKind, "go:method")
}

func sourceFileStableKey(file SourceFile) string {
	if file.StableKey != "" {
		return file.StableKey
	}
	return file.Path
}

func sourceSymbolStableKey(symbol SourceSymbol) string {
	if symbol.StableKey != "" {
		return symbol.StableKey
	}
	if symbol.QualifiedName != "" {
		return symbol.QualifiedName
	}
	return symbol.Name
}

func symbolSpans(symbol SourceSymbol) []SourceSpan {
	result := make([]SourceSpan, 0, len(symbol.Locations))
	for _, location := range symbol.Locations {
		if validQualitySpan(location.Span) {
			result = append(result, location.Span)
		}
	}
	return result
}

func callableBodySpan(snapshot SourceSnapshot, symbol SourceSymbol) (SourceSpan, bool) {
	if symbol.BodySpan == nil || !validQualitySpan(*symbol.BodySpan) {
		return SourceSpan{}, false
	}
	span := *symbol.BodySpan
	if span.End.ByteOffset <= span.Start.ByteOffset {
		return SourceSpan{}, false
	}
	if len(symbol.Locations) > 0 {
		locatedInSameFile := false
		for _, location := range symbol.Locations {
			if location.Span.FileID == span.FileID {
				locatedInSameFile = true
				break
			}
		}
		if !locatedInSameFile {
			return SourceSpan{}, false
		}
	}
	for _, file := range snapshot.Files {
		if file.ID != span.FileID {
			continue
		}
		if span.ContentHash != file.ContentHash || span.End.ByteOffset > file.ByteCount {
			return SourceSpan{}, false
		}
		return span, true
	}
	return SourceSpan{}, false
}

func validQualitySpan(span SourceSpan) bool {
	if span.FileID == "" || span.CoordinateSystem != "utf8-byte" || span.Start.Line < 1 || span.End.Line < span.Start.Line || span.Start.Column < 1 || span.End.Column < 1 || span.Start.ByteOffset < 0 || span.End.ByteOffset < span.Start.ByteOffset || span.End.Line == span.Start.Line && span.End.Column < span.Start.Column || span.ContentHash.Algorithm != "hash:sha-256" || len(span.ContentHash.Value) != 64 || strings.ToLower(span.ContentHash.Value) != span.ContentHash.Value {
		return false
	}
	_, err := hex.DecodeString(span.ContentHash.Value)
	return err == nil
}

func floatPointer(value float64) *float64 { return &value }
