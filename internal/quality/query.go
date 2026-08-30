package quality

import (
	"encoding/base64"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
)

const (
	QualityQuerySchemaVersion = "arch-view.quality-query/v1"
	DefaultQualityQueryLimit  = 50
	MaxQualityQueryLimit      = 200
	DefaultEvidenceMaxLines   = 120
	MaxEvidenceLines          = 120
	DefaultEvidenceMaxBytes   = 4 * 1024 * 1024
	MaxEvidenceBytes          = 4 * 1024 * 1024
)

// QualityQueryService is a read-only projection over immutable quality
// reports. It does not evaluate rules, resolve source paths, or return source
// text. Transport adapters can compose its evidence result with an existing
// bounded source boundary.
type QualityQueryService struct {
	reports  []QualityEvaluation
	maxLimit int
}

func NewQualityQueryService(reports ...QualityEvaluation) *QualityQueryService {
	values := make([]QualityEvaluation, len(reports))
	for index := range reports {
		values[index] = cloneQualityEvaluation(reports[index])
	}
	sort.Slice(values, func(i, j int) bool { return values[i].EvaluationID < values[j].EvaluationID })
	return &QualityQueryService{reports: values, maxLimit: MaxQualityQueryLimit}
}

// NewQualityQueryInvalid and NewQualityQueryNotFound let transport adapters
// preserve the quality error taxonomy without reaching into package-private
// constructors.
func NewQualityQueryInvalid(message string) error {
	return newQualityError(ErrorQueryInvalid, message, nil)
}

func NewQualityQueryNotFound(message string) error {
	return newQualityError(ErrorQueryNotFound, message, nil)
}

type QualityReportQueryOptions struct {
	ReportID   string
	SnapshotID string
	ScopeID    string
	Cursor     string
	Limit      int
}

type QualityReportSummary struct {
	EvaluationID          string        `json:"evaluation_id"`
	SourceSnapshotIDs     []string      `json:"source_snapshot_ids"`
	ProfileID             string        `json:"profile_id"`
	ProfileVersion        string        `json:"profile_version"`
	Baseline              *BaselineRef  `json:"baseline,omitempty"`
	FindingCount          int           `json:"finding_count"`
	CoverageCount         int           `json:"coverage_count"`
	DiagnosticCount       int           `json:"diagnostic_count"`
	EvaluationFingerprint ContentDigest `json:"evaluation_fingerprint"`
	ReportDigest          ContentDigest `json:"report_digest"`
}

type QualityReportPage struct {
	SchemaVersion string                 `json:"schema_version"`
	Items         []QualityReportSummary `json:"items"`
	Total         int                    `json:"total"`
	NextCursor    string                 `json:"next_cursor,omitempty"`
}

type QualityFindingQueryOptions struct {
	ReportID       string
	SnapshotID     string
	ScopeID        string
	SubjectID      string
	SubjectKind    string
	FileID         string
	RuleID         string
	AssessmentKind string
	Severity       string
	Status         string
	Cursor         string
	Limit          int
}

type QualityFindingPage struct {
	SchemaVersion string            `json:"schema_version"`
	ReportID      string            `json:"report_id"`
	EvaluationID  string            `json:"evaluation_id"`
	SnapshotIDs   []string          `json:"source_snapshot_ids"`
	ScopeID       string            `json:"scope_id,omitempty"`
	Items         []QualityFinding  `json:"items"`
	Total         int               `json:"total"`
	NextCursor    string            `json:"next_cursor,omitempty"`
	Coverage      []QualityCoverage `json:"coverage"`
}

type QualityCoverageQueryOptions struct {
	RuleID string
	Cursor string
	Limit  int
}

type QualityCoveragePage struct {
	SchemaVersion string            `json:"schema_version"`
	ReportID      string            `json:"report_id"`
	EvaluationID  string            `json:"evaluation_id"`
	Items         []QualityCoverage `json:"items"`
	Total         int               `json:"total"`
	NextCursor    string            `json:"next_cursor,omitempty"`
}

type QualityEvidenceQueryOptions struct {
	IncludeSourceContext bool
	MaxLines             int
	MaxBytes             int
}

type QualityFindingEvidence struct {
	SchemaVersion          string         `json:"schema_version"`
	ReportID               string         `json:"report_id"`
	EvaluationID           string         `json:"evaluation_id"`
	Finding                QualityFinding `json:"finding"`
	SourceContextRequested bool           `json:"source_context_requested,omitempty"`
	SourceContextMaxLines  int            `json:"source_context_max_lines,omitempty"`
	SourceContextMaxBytes  int            `json:"source_context_max_bytes,omitempty"`
}

// ListReports returns compact report summaries in deterministic order.
func (service *QualityQueryService) ListReports(options QualityReportQueryOptions) (QualityReportPage, error) {
	if service == nil {
		return QualityReportPage{}, newQualityError(ErrorQueryNotFound, "quality reports were not found", nil)
	}
	if err := validateQualityPageOptions(options.Cursor, options.Limit, serviceLimit(service)); err != nil {
		return QualityReportPage{}, err
	}
	items := make([]QualityReportSummary, 0, len(service.reports))
	for _, report := range service.reports {
		if options.ReportID != "" && report.EvaluationID != options.ReportID || options.SnapshotID != "" && !containsString(report.SourceSnapshotIDs, options.SnapshotID) || options.ScopeID != "" && !reportContainsScope(report, options.ScopeID) {
			continue
		}
		items = append(items, reportSummary(report))
	}
	start, end, next, err := qualityPageBounds(options.Cursor, options.Limit, len(items), serviceLimit(service))
	if err != nil {
		return QualityReportPage{}, err
	}
	return QualityReportPage{SchemaVersion: QualityQuerySchemaVersion, Items: items[start:end], Total: len(items), NextCursor: next}, nil
}

// GetReport returns one complete report without its source-index attachment.
// Quality reports contain bounded evidence references; source context remains
// an explicit follow-up request.
func (service *QualityQueryService) GetReport(reportID string) (QualityEvaluation, error) {
	report, err := service.findReport(reportID)
	if err != nil {
		return QualityEvaluation{}, err
	}
	return cloneQualityEvaluation(report), nil
}

// ListFindings filters a report without changing its canonical ordering. A
// file subject filter is exact: it only matches findings whose subject is the
// requested file, never a path/name coincidence.
func (service *QualityQueryService) ListFindings(reportID string, options QualityFindingQueryOptions) (QualityFindingPage, error) {
	if options.ReportID != "" && reportID != options.ReportID {
		return QualityFindingPage{}, newQualityError(ErrorQueryInvalid, "report selector does not match the requested report", map[string]any{"report_id": reportID})
	}
	if err := validateQualityPageOptions(options.Cursor, options.Limit, serviceLimit(service)); err != nil {
		return QualityFindingPage{}, err
	}
	if err := validateFindingSelectors(options); err != nil {
		return QualityFindingPage{}, err
	}
	report, err := service.findReport(reportID)
	if err != nil {
		return QualityFindingPage{}, err
	}
	items := make([]QualityFinding, 0, len(report.Findings))
	for _, finding := range report.Findings {
		if !matchesFinding(finding, options) {
			continue
		}
		items = append(items, cloneQualityFinding(finding))
	}
	sort.Slice(items, func(i, j int) bool { return qualityFindingSortKey(items[i]) < qualityFindingSortKey(items[j]) })
	start, end, next, err := qualityPageBounds(options.Cursor, options.Limit, len(items), serviceLimit(service))
	if err != nil {
		return QualityFindingPage{}, err
	}
	coverage := filterCoverage(report.Coverage, options.RuleID, options.ScopeID, options.SnapshotID)
	return QualityFindingPage{SchemaVersion: QualityQuerySchemaVersion, ReportID: report.EvaluationID, EvaluationID: report.EvaluationID, SnapshotIDs: append([]string(nil), report.SourceSnapshotIDs...), ScopeID: options.ScopeID, Items: items[start:end], Total: len(items), NextCursor: next, Coverage: coverage}, nil
}

// GetQualityCoverage exposes coverage as a first-class result rather than
// collapsing unsupported, partial, or not-evaluable input into an empty list.
func (service *QualityQueryService) GetQualityCoverage(reportID string, options QualityCoverageQueryOptions) (QualityCoveragePage, error) {
	if err := validateQualityPageOptions(options.Cursor, options.Limit, serviceLimit(service)); err != nil {
		return QualityCoveragePage{}, err
	}
	report, err := service.findReport(reportID)
	if err != nil {
		return QualityCoveragePage{}, err
	}
	items := filterCoverage(report.Coverage, options.RuleID, "", "")
	start, end, next, err := qualityPageBounds(options.Cursor, options.Limit, len(items), serviceLimit(service))
	if err != nil {
		return QualityCoveragePage{}, err
	}
	return QualityCoveragePage{SchemaVersion: QualityQuerySchemaVersion, ReportID: report.EvaluationID, EvaluationID: report.EvaluationID, Items: items[start:end], Total: len(items), NextCursor: next}, nil
}

// GetFindingEvidence returns the complete compact evidence envelope. It never
// reads a path or includes source text. The source-context fields only record
// a validated explicit budget for a transport adapter to execute separately.
func (service *QualityQueryService) GetFindingEvidence(reportID, findingID string, options QualityEvidenceQueryOptions) (QualityFindingEvidence, error) {
	report, err := service.findReport(reportID)
	if err != nil {
		return QualityFindingEvidence{}, err
	}
	findingID = strings.TrimSpace(findingID)
	if findingID == "" {
		return QualityFindingEvidence{}, newQualityError(ErrorQueryInvalid, "finding ID is required", nil)
	}
	if !options.IncludeSourceContext && (options.MaxLines != 0 || options.MaxBytes != 0) {
		return QualityFindingEvidence{}, newQualityError(ErrorQueryInvalid, "source-context budgets require IncludeSourceContext=true", nil)
	}
	for _, finding := range report.Findings {
		if finding.ID != findingID && finding.FindingKey != findingID {
			continue
		}
		result := QualityFindingEvidence{SchemaVersion: QualityQuerySchemaVersion, ReportID: report.EvaluationID, EvaluationID: report.EvaluationID, Finding: cloneQualityFinding(finding)}
		if options.IncludeSourceContext {
			lines, bytes, err := validateEvidenceBudget(options.MaxLines, options.MaxBytes)
			if err != nil {
				return QualityFindingEvidence{}, err
			}
			result.SourceContextRequested = true
			result.SourceContextMaxLines = lines
			result.SourceContextMaxBytes = bytes
		}
		return result, nil
	}
	return QualityFindingEvidence{}, newQualityError(ErrorQueryNotFound, "finding was not found in the requested report", map[string]any{"report_id": reportID, "finding_id": findingID})
}

func reportSummary(report QualityEvaluation) QualityReportSummary {
	return QualityReportSummary{EvaluationID: report.EvaluationID, SourceSnapshotIDs: append([]string(nil), report.SourceSnapshotIDs...), ProfileID: report.ProfileID, ProfileVersion: report.ProfileVersion, Baseline: cloneBaselineRef(report.Baseline), FindingCount: len(report.Findings), CoverageCount: len(report.Coverage), DiagnosticCount: len(report.Diagnostics), EvaluationFingerprint: report.EvaluationFingerprint, ReportDigest: report.ReportDigest}
}

func (service *QualityQueryService) findReport(reportID string) (QualityEvaluation, error) {
	if service == nil {
		return QualityEvaluation{}, newQualityError(ErrorQueryNotFound, "quality report was not found", map[string]any{"report_id": reportID})
	}
	for _, report := range service.reports {
		if report.EvaluationID == reportID {
			return report, nil
		}
	}
	return QualityEvaluation{}, newQualityError(ErrorQueryNotFound, "quality report was not found", map[string]any{"report_id": reportID})
}

func matchesFinding(finding QualityFinding, options QualityFindingQueryOptions) bool {
	if options.SnapshotID != "" && finding.SubjectRef.SnapshotID != options.SnapshotID || options.ScopeID != "" && finding.SubjectRef.ScopeID != options.ScopeID || options.SubjectID != "" && finding.SubjectRef.ID != options.SubjectID || options.SubjectKind != "" && finding.SubjectRef.Kind != options.SubjectKind || options.FileID != "" && (finding.SubjectRef.Kind != "file" || finding.SubjectRef.ID != options.FileID) || options.RuleID != "" && finding.RuleID != options.RuleID || options.AssessmentKind != "" && finding.AssessmentKind != options.AssessmentKind || options.Severity != "" && finding.Severity != options.Severity || options.Status != "" && finding.Status != options.Status {
		return false
	}
	return true
}

func reportContainsScope(report QualityEvaluation, scopeID string) bool {
	for _, finding := range report.Findings {
		if finding.SubjectRef.ScopeID == scopeID {
			return true
		}
	}
	for _, coverage := range report.Coverage {
		for _, evidenceID := range coverage.Provenance.EvidenceIDs {
			if strings.HasPrefix(evidenceID, "scope:"+scopeID+":") {
				return true
			}
		}
	}
	return false
}

func filterCoverage(values []QualityCoverage, ruleID, scopeID, snapshotID string) []QualityCoverage {
	result := make([]QualityCoverage, 0, len(values))
	for _, value := range values {
		if ruleID != "" && value.RuleID != ruleID || !coverageMatchesSource(value, scopeID, snapshotID) {
			continue
		}
		result = append(result, cloneQualityCoverage(value))
	}
	return result
}

func qualityFindingSortKey(value QualityFinding) string {
	return value.SubjectRef.ScopeID + "\x00" + value.SubjectRef.Kind + "\x00" + value.SubjectRef.ID + "\x00" + value.RuleID + "\x00" + value.Severity + "\x00" + value.FindingKey + "\x00" + value.ID
}

func validateFindingSelectors(options QualityFindingQueryOptions) error {
	if options.AssessmentKind != "" && options.AssessmentKind != AssessmentExact && options.AssessmentKind != AssessmentSignal {
		return newQualityError(ErrorQueryInvalid, "quality finding assessment selector is unsupported", map[string]any{"assessment_kind": options.AssessmentKind})
	}
	if options.Severity != "" && severityRank(options.Severity) < 0 {
		return newQualityError(ErrorQueryInvalid, "quality finding severity selector is unsupported", map[string]any{"severity": options.Severity})
	}
	if options.Status != "" && !validFindingStatus(options.Status) {
		return newQualityError(ErrorQueryInvalid, "quality finding status selector is unsupported", map[string]any{"status": options.Status})
	}
	return nil
}

func coverageMatchesSource(value QualityCoverage, scopeID, snapshotID string) bool {
	if scopeID == "" && snapshotID == "" {
		return true
	}
	for _, evidenceID := range value.Provenance.EvidenceIDs {
		matchesScope := scopeID == "" || strings.HasPrefix(evidenceID, "scope:"+scopeID+":")
		matchesSnapshot := snapshotID == "" || strings.HasSuffix(evidenceID, ":"+snapshotID)
		if matchesScope && matchesSnapshot {
			return true
		}
	}
	return false
}

func cloneQualityEvaluation(value QualityEvaluation) QualityEvaluation {
	data, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var result QualityEvaluation
	if err := json.Unmarshal(data, &result); err != nil {
		return value
	}
	return result
}

func cloneQualityFinding(value QualityFinding) QualityFinding {
	report := cloneQualityEvaluation(QualityEvaluation{Findings: []QualityFinding{value}})
	if len(report.Findings) == 1 {
		return report.Findings[0]
	}
	return value
}

func cloneQualityCoverage(value QualityCoverage) QualityCoverage {
	report := cloneQualityEvaluation(QualityEvaluation{Coverage: []QualityCoverage{value}})
	if len(report.Coverage) == 1 {
		return report.Coverage[0]
	}
	return value
}

func serviceLimit(service *QualityQueryService) int {
	if service == nil || service.maxLimit <= 0 || service.maxLimit > MaxQualityQueryLimit {
		return MaxQualityQueryLimit
	}
	return service.maxLimit
}

func validateQualityPageOptions(cursor string, limit, max int) error {
	if limit != 0 && (limit < 1 || limit > max) {
		return newQualityError(ErrorQueryInvalid, "quality query limit is outside the allowed bound", map[string]any{"limit": limit, "max": max})
	}
	if cursor != "" {
		if _, err := decodeQualityCursor(cursor); err != nil {
			return err
		}
	}
	return nil
}

func qualityPageBounds(cursor string, limit, total, max int) (int, int, string, error) {
	if limit == 0 {
		limit = DefaultQualityQueryLimit
	}
	if limit < 1 || limit > max {
		return 0, 0, "", newQualityError(ErrorQueryInvalid, "quality query limit is outside the allowed bound", map[string]any{"limit": limit, "max": max})
	}
	start := 0
	if cursor != "" {
		value, err := decodeQualityCursor(cursor)
		if err != nil {
			return 0, 0, "", err
		}
		start = value
	}
	if start < 0 || start > total {
		return 0, 0, "", newQualityError(ErrorQueryInvalid, "quality query cursor is outside the result set", map[string]any{"cursor": cursor})
	}
	end := start + limit
	if end > total {
		end = total
	}
	next := ""
	if end < total {
		next = encodeQualityCursor(end)
	}
	return start, end, next, nil
}

func encodeQualityCursor(value int) string {
	return base64.RawURLEncoding.EncodeToString([]byte("qv1:" + strconv.Itoa(value)))
}

func decodeQualityCursor(value string) (int, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || !strings.HasPrefix(string(decoded), "qv1:") {
		return 0, newQualityError(ErrorQueryInvalid, "quality query cursor is malformed", map[string]any{"cursor": value})
	}
	position, err := strconv.Atoi(strings.TrimPrefix(string(decoded), "qv1:"))
	if err != nil || position < 0 {
		return 0, newQualityError(ErrorQueryInvalid, "quality query cursor is malformed", map[string]any{"cursor": value})
	}
	return position, nil
}

func validateEvidenceBudget(lines, bytes int) (int, int, error) {
	if lines == 0 {
		lines = DefaultEvidenceMaxLines
	}
	if bytes == 0 {
		bytes = DefaultEvidenceMaxBytes
	}
	if lines < 1 || lines > MaxEvidenceLines || bytes < 1 || bytes > MaxEvidenceBytes {
		return 0, 0, newQualityError(ErrorEvidenceBudget, "source-context budget is outside the allowed bound", map[string]any{"max_lines": MaxEvidenceLines, "max_bytes": MaxEvidenceBytes, "lines": lines, "bytes": bytes})
	}
	return lines, bytes, nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
