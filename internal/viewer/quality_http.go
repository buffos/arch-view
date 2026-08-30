package viewer

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/orchestration"
	"github.com/buffo/arch-view/internal/analysis/sourceindex"
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/quality"
)

type qualityReportHTTPResponse struct {
	SchemaVersion string                     `json:"schema_version"`
	Status        string                     `json:"status"`
	Report        *quality.QualityEvaluation `json:"report,omitempty"`
	Coverage      []quality.QualityCoverage  `json:"coverage"`
	Message       string                     `json:"message,omitempty"`
}

type qualityFindingsHTTPResponse struct {
	quality.QualityFindingPage
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type qualityEvidenceHTTPResponse struct {
	quality.QualityFindingEvidence
	SourceContext *SourceExcerpt `json:"source_context,omitempty"`
}

func (s *Server) handleQuality(writer http.ResponseWriter, request *http.Request, parts []string, modelID string, report *quality.QualityEvaluation, value *model.Model, aggregate *orchestration.AnalysisRun) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}
	if len(parts) == 2 {
		if report == nil {
			writeJSON(writer, http.StatusOK, qualityReportHTTPResponse{SchemaVersion: quality.QualityQuerySchemaVersion, Status: "missing", Coverage: []quality.QualityCoverage{}, Message: "No quality report is attached to this model revision."})
			return
		}
		service := quality.NewQualityQueryService(*report)
		loaded, err := service.GetReport(report.EvaluationID)
		if err != nil {
			writeQualityHTTPError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, qualityReportHTTPResponse{SchemaVersion: quality.QualityQuerySchemaVersion, Status: "available", Report: &loaded, Coverage: append([]quality.QualityCoverage(nil), loaded.Coverage...)})
		return
	}
	if len(parts) < 3 || parts[2] != "findings" {
		if len(parts) == 3 && parts[2] == "coverage" {
			if report == nil {
				writeJSON(writer, http.StatusOK, quality.QualityCoveragePage{SchemaVersion: quality.QualityQuerySchemaVersion, ReportID: modelID, Items: []quality.QualityCoverage{}, Total: 0})
				return
			}
			options, err := qualityCoverageQueryOptions(request.URL.Query())
			if err != nil {
				writeQualityHTTPError(writer, err)
				return
			}
			page, err := quality.NewQualityQueryService(*report).GetQualityCoverage(report.EvaluationID, options)
			if err != nil {
				writeQualityHTTPError(writer, err)
				return
			}
			writeJSON(writer, http.StatusOK, page)
			return
		}
		http.NotFound(writer, request)
		return
	}
	if len(parts) == 3 {
		options, err := qualityFindingQueryOptions(request.URL.Query())
		if err != nil {
			writeQualityHTTPError(writer, err)
			return
		}
		if report == nil {
			writeJSON(writer, http.StatusOK, qualityFindingsHTTPResponse{QualityFindingPage: quality.QualityFindingPage{SchemaVersion: quality.QualityQuerySchemaVersion, ReportID: modelID, SnapshotIDs: []string{}, Items: []quality.QualityFinding{}, Coverage: []quality.QualityCoverage{}}, Status: "missing", Message: "No quality report is attached to this model revision."})
			return
		}
		service := quality.NewQualityQueryService(*report)
		page, err := service.ListFindings(report.EvaluationID, options)
		if err != nil {
			writeQualityHTTPError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, qualityFindingsHTTPResponse{QualityFindingPage: page, Status: "available"})
		return
	}
	if len(parts) != 5 || parts[4] != "evidence" {
		http.NotFound(writer, request)
		return
	}
	findingID, err := url.PathUnescape(parts[3])
	if err != nil || strings.TrimSpace(findingID) == "" {
		writeQualityHTTPError(writer, quality.NewQualityQueryInvalid("finding ID is malformed"))
		return
	}
	if report == nil {
		writeQualityHTTPError(writer, quality.NewQualityQueryNotFound("quality report is not attached to this model revision"))
		return
	}
	evidenceOptions, err := qualityEvidenceQueryOptions(request.URL.Query())
	if err != nil {
		writeQualityHTTPError(writer, err)
		return
	}
	service := quality.NewQualityQueryService(*report)
	evidence, err := service.GetFindingEvidence(report.EvaluationID, findingID, evidenceOptions)
	if err != nil {
		writeQualityHTTPError(writer, err)
		return
	}
	response := qualityEvidenceHTTPResponse{QualityFindingEvidence: evidence}
	if evidenceOptions.IncludeSourceContext {
		index, indexErr := sourceIndexForRequest(value, aggregate, request.URL.Query())
		if indexErr != nil {
			writeQualityHTTPError(writer, indexErr)
			return
		}
		if len(evidence.Finding.Evidence.SourceSpans) == 0 {
			writeQualityHTTPError(writer, analysis.NewHostError(analysis.ErrSourceReferenceInvalid, "quality finding has no bounded source span for context", map[string]any{"finding_id": findingID}))
			return
		}
		sourceEvidence := sourceFactEvidenceFromQuality(evidence.Finding)
		excerpt, excerptErr := s.sourceFactExcerptWithBudget(index, sourceEvidence, modelID, evidenceOptions.MaxLines, evidenceOptions.MaxBytes)
		if excerptErr != nil {
			writeQualityHTTPError(writer, excerptErr)
			return
		}
		response.SourceContext = excerpt
	}
	writeJSON(writer, http.StatusOK, response)
}

func qualityFindingQueryOptions(query url.Values) (quality.QualityFindingQueryOptions, error) {
	options := quality.QualityFindingQueryOptions{ReportID: query.Get("report_id"), SnapshotID: query.Get("snapshot_id"), ScopeID: normalizedQualityScope(query.Get("scope")), SubjectID: query.Get("subject_id"), SubjectKind: query.Get("subject_kind"), FileID: query.Get("file_id"), RuleID: query.Get("rule_id"), AssessmentKind: query.Get("assessment_kind"), Severity: query.Get("severity"), Status: query.Get("status"), Cursor: query.Get("cursor")}
	if options.AssessmentKind != "" && options.AssessmentKind != quality.AssessmentExact && options.AssessmentKind != quality.AssessmentSignal {
		return quality.QualityFindingQueryOptions{}, quality.NewQualityQueryInvalid("assessment_kind is unsupported")
	}
	if options.Severity != "" && options.Severity != quality.SeverityInfo && options.Severity != quality.SeverityWarning && options.Severity != quality.SeverityError && options.Severity != quality.SeverityBlocker {
		return quality.QualityFindingQueryOptions{}, quality.NewQualityQueryInvalid("severity is unsupported")
	}
	if options.Status != "" && options.Status != quality.StatusActive && options.Status != quality.StatusSuppressed && options.Status != quality.StatusBaseline && options.Status != quality.StatusResolved && options.Status != quality.StatusNotEvaluable {
		return quality.QualityFindingQueryOptions{}, quality.NewQualityQueryInvalid("status is unsupported")
	}
	if raw := query.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil {
			return quality.QualityFindingQueryOptions{}, quality.NewQualityQueryInvalid("limit must be an integer")
		}
		options.Limit = limit
	}
	return options, nil
}

func qualityEvidenceQueryOptions(query url.Values) (quality.QualityEvidenceQueryOptions, error) {
	options := quality.QualityEvidenceQueryOptions{}
	include := query.Get("include_source_context")
	if include == "" {
		include = query.Get("include_source")
	}
	if include != "" {
		parsed, err := strconv.ParseBool(include)
		if err != nil {
			return options, quality.NewQualityQueryInvalid("include_source_context must be a boolean")
		}
		options.IncludeSourceContext = parsed
	}
	var err error
	if raw := query.Get("max_lines"); raw != "" {
		options.MaxLines, err = strconv.Atoi(raw)
		if err != nil {
			return options, quality.NewQualityQueryInvalid("max_lines must be an integer")
		}
	}
	if raw := query.Get("max_bytes"); raw != "" {
		options.MaxBytes, err = strconv.Atoi(raw)
		if err != nil {
			return options, quality.NewQualityQueryInvalid("max_bytes must be an integer")
		}
	}
	if !options.IncludeSourceContext && (options.MaxLines != 0 || options.MaxBytes != 0) {
		return options, quality.NewQualityQueryInvalid("source-context budgets require include_source_context=true")
	}
	return options, nil
}

func qualityCoverageQueryOptions(query url.Values) (quality.QualityCoverageQueryOptions, error) {
	options := quality.QualityCoverageQueryOptions{RuleID: query.Get("rule_id"), Cursor: query.Get("cursor")}
	if raw := query.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil {
			return options, quality.NewQualityQueryInvalid("limit must be an integer")
		}
		options.Limit = limit
	}
	return options, nil
}

func normalizedQualityScope(value string) string {
	value = strings.TrimSpace(value)
	if strings.EqualFold(value, "all") {
		return ""
	}
	return value
}

func sourceFactEvidenceFromQuality(finding quality.QualityFinding) sourceindex.SourceFactEvidence {
	spans := make([]analysis.SourceSpan, 0, len(finding.Evidence.SourceSpans))
	for _, span := range finding.Evidence.SourceSpans {
		spans = append(spans, analysis.SourceSpan{FileID: span.FileID, Start: analysis.SpanPosition{ByteOffset: span.Start.ByteOffset, Line: span.Start.Line, Column: span.Start.Column}, End: analysis.SpanPosition{ByteOffset: span.End.ByteOffset, Line: span.End.Line, Column: span.End.Column}, CoordinateSystem: span.CoordinateSystem, ContentHash: analysis.ContentDigest{Algorithm: span.ContentHash.Algorithm, Value: span.ContentHash.Value}})
	}
	return sourceindex.SourceFactEvidence{SnapshotID: finding.SubjectRef.SnapshotID, ScopeID: finding.SubjectRef.ScopeID, EntityID: finding.ID, Spans: spans, SourceReferenceIDs: []string{}, Documentation: []analysis.DocumentationRecord{}, Relations: []analysis.CodeRelation{}}
}

func writeQualityHTTPError(writer http.ResponseWriter, err error) {
	if qualityErr, ok := err.(*quality.QualityError); ok {
		writeHTTPError(writer, qualityHTTPStatus(qualityErr.Code), analysis.NewHostError(analysis.ErrorCode(qualityErr.Code), qualityErr.Message, qualityErr.Details))
		return
	}
	writeHTTPError(writer, sourceIndexHTTPStatus(err), err)
}

func qualityHTTPStatus(code quality.ErrorCode) int {
	switch code {
	case quality.ErrorQueryNotFound:
		return http.StatusNotFound
	case quality.ErrorEvidenceBudget:
		return http.StatusRequestEntityTooLarge
	default:
		return http.StatusBadRequest
	}
}
